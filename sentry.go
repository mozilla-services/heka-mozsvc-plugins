/***** BEGIN LICENSE BLOCK *****
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at http://mozilla.org/MPL/2.0/.
#
# The Initial Developer of the Original Code is the Mozilla Foundation.
# Portions created by the Initial Developer are Copyright (C) 2012
# the Initial Developer. All Rights Reserved.
#
# Contributor(s):
#   Victor Ng (vng@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"net"
	"net/url"
	"time"
)

// CheckMAC returns true if messageMAC is a valid HMAC tag for
// message.
func hmac_sha1(message, key []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hex.EncodeToString(expectedMAC)
}

type SentryMsg struct {
	encoded_payload string

	// needed for auth and signature methods
	str_ts string

	parsed_dsn   *url.URL
	dsn_password string
	data_packet  []byte
}

type SentryOutputWriter struct {
	config *SentryOutputWriterConfig
	udpMap map[string]net.Conn
}

func (self *SentryMsg) get_auth_header(protocol float32, client_id string) string {
	header_tmpl := "Sentry sentry_timestamp=%s, sentry_client=%s, sentry_version=%0.1f, sentry_key=%s"
	return fmt.Sprintf(header_tmpl, self.str_ts, client_id, protocol, self.parsed_dsn.User.Username())
}

func (self *SentryMsg) get_signature() string {
	return hmac_sha1([]byte(fmt.Sprintf("%s %s", self.str_ts, self.encoded_payload)), []byte(self.dsn_password))
}

func (self *SentryMsg) compute_auth_header() (string, error) {

	var prep_bool bool

	self.dsn_password, prep_bool = self.parsed_dsn.User.Password()
	if !prep_bool {
		return "", fmt.Errorf("No password was found in the DSN URI")
	}

	return self.get_auth_header(2.0, "raven-go/1.0"), nil
}

type SentryOutputWriterConfig struct {
	MaxSentryBytes int
	MaxUdpSockets  int
}

func (self *SentryOutputWriter) ConfigStruct() interface{} {
	return &SentryOutputWriterConfig{MaxSentryBytes: 64000,
		MaxUdpSockets: 20}
}

func (self *SentryOutputWriter) Init(config interface{}) error {
	self.config = config.(*SentryOutputWriterConfig)
	self.udpMap = make(map[string]net.Conn)
	return nil
}

func (self *SentryOutputWriter) MakeOutData() interface{} {
	raw_bytes := make([]byte, 0, self.config.MaxSentryBytes)
	return &SentryMsg{data_packet: raw_bytes}
}

func (self *SentryOutputWriter) ZeroOutData(outData interface{}) {
	// Just zero out the byte array
	msg := outData.(*SentryMsg)
	msg.data_packet = msg.data_packet[:0]
}

func (self *SentryOutputWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{}, timeout *time.Duration) error {

	var prep_error error
	var prep_bool bool
	var epoch_ts64 float64
	var epoch_time time.Time
	var auth_header string
	var dsn string

	sentryMsg := outData.(*SentryMsg)
	sentryMsg.encoded_payload = pack.Message.Payload
	epoch_ts64, prep_bool = pack.Message.Fields["epoch_timestamp"].(float64)

	if !prep_bool {
		return fmt.Errorf("Error parsing epoch_timestamp")
	}

	epoch_time = (time.Unix(int64(epoch_ts64), int64((epoch_ts64-float64(int64(epoch_ts64)))*1e9)))
	sentryMsg.str_ts = epoch_time.Format(time.RFC3339Nano)

	dsn = pack.Message.Fields["dsn"].(string)

	sentryMsg.parsed_dsn, prep_error = url.Parse(dsn)
	if prep_error != nil {
		return fmt.Errorf("Error parsing DSN from sentry message")
	}

	auth_header, prep_error = sentryMsg.compute_auth_header()

	if prep_error != nil {
		return fmt.Errorf("Error computing authentication header from sentry message")
	}

	sentryMsg.data_packet = []byte(fmt.Sprintf("%s\n\n%s", auth_header, sentryMsg.encoded_payload))
	return nil
}

func (self *SentryOutputWriter) Write(outData interface{}) (err error) {
	var udp_addr *net.UDPAddr
	var socket_err error

	var socket net.Conn
	var host_ok bool

	var sentryMsg *SentryMsg

	sentryMsg = outData.(*SentryMsg)
	udp_addr_str := sentryMsg.parsed_dsn.Host

	socket, host_ok = self.udpMap[udp_addr_str]
	if !host_ok {
		if len(self.udpMap) > self.config.MaxUdpSockets {
			return fmt.Errorf("Maximum number of UDP sockets reached.")
		}

		udp_addr, socket_err = net.ResolveUDPAddr("udp", udp_addr_str)
		if err != nil {
			return fmt.Errorf("UdpOutput error resolving UDP address %s: %s", udp_addr_str, err.Error())
		}

		socket, socket_err = net.DialUDP("udp", nil, udp_addr)
		if socket_err != nil {
			return fmt.Errorf("Error while dialing the UDP socket")
		}
		self.udpMap[sentryMsg.parsed_dsn.Host] = socket
	}
	socket.Write(sentryMsg.data_packet)
	return nil
}

func (self *SentryOutputWriter) Event(eventType string) {
	// Don't need to do anything here as sentry is just UDP
	// so we don't need to respond to RELOAD or STOP requests
}

func init() {
	pipeline.RegisterPlugin("SentryOutput", func() interface{} {
		return pipeline.RunnerMaker(new(SentryOutputWriter))
	})
}
