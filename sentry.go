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
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"net"
	"net/url"
	"time"
)

const (
	auth_header_tmpl   = "Sentry sentry_timestamp=%s, sentry_client=%s, sentry_version=%0.1f, sentry_key=%s"
	raven_client_id    = "raven-go/1.0"
	raven_protocol_rev = 2.0
)

type SentryMsg struct {
	encoded_payload string
	parsed_dsn      *url.URL
	data_packet     []byte
}

type SentryOutputWriter struct {
	config *SentryOutputWriterConfig
	udpMap map[string]net.Conn
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
	var ok bool
	var tmp interface{}
	var epoch_ts64 float64
	var epoch_time time.Time
	var auth_header string
	var dsn string
	var str_ts string

	sentryMsg := outData.(*SentryMsg)
	sentryMsg.encoded_payload = pack.Message.GetPayload()

	tmp, ok = pack.Message.GetFieldValue("epoch_timestamp")
	if !ok {
		return fmt.Errorf("Error: no epoch_timestamp was found in Fields")
	}

	epoch_ts64, ok = tmp.(float64)
	if !ok {
		return fmt.Errorf("Error: epoch_timestamp isn't a float64")
	}

	epoch_time = (time.Unix(int64(epoch_ts64), int64((epoch_ts64-float64(int64(epoch_ts64)))*1e9)))
	str_ts = epoch_time.Format(time.RFC3339Nano)

	tmp, ok = pack.Message.GetFieldValue("dsn")
	if !ok {
		return fmt.Errorf("Error: no dsn was found in Fields")
	}

	dsn, ok = tmp.(string)
	if !ok {
		return fmt.Errorf("Error: dsn isn't a string")
	}

	sentryMsg.parsed_dsn, prep_error = url.Parse(dsn)
	if prep_error != nil {
		return fmt.Errorf("Error parsing DSN from sentry message")
	}

	auth_header = fmt.Sprintf(auth_header_tmpl, str_ts, raven_client_id, raven_protocol_rev, sentryMsg.parsed_dsn.User.Username())
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
			return fmt.Errorf("Maximum number of UDP sockets reached.  Max=[%d]", self.config.MaxUdpSockets)
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
