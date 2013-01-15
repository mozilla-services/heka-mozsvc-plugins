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
	"log"
	"net"
	"net/url"
	"time"
)

const (
	MAX_SENTRY_BYTES = 64000
)

// CheckMAC returns true if messageMAC is a valid HMAC tag for
// message.
func hmac_sha1(message, key []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hex.EncodeToString(expectedMAC)
}

type SentryOutputGlobal struct {
}

type SentryMsg struct {
	encoded_payload string
	epoch_ts64      float64
	epoch_time      time.Time
	dsn             string
	parsed_dsn      *url.URL
	auth_header     string

	prep_error error
	prep_bool  bool

	// TODO: i think this might be the only thing we really need
	data_packet []byte
}

type SentryOutputWriter struct {
	sentryMsg *SentryMsg

	// TODO: pull the connection map and
	// errors up here from Write(outData interface{})
	udpMap map[string]net.Conn

	socket_err error
	socket     net.Conn
	host_ok    bool
}

type SentryOutputConfig struct {
}

func (self *SentryOutputWriter) ConfigStruct() interface{} {
	// Default the statsd output to localhost port 5555
	return &SentryOutputConfig{}
}

func get_auth_header(protocol float32, signature string, timestamp string, client_id string, api_key string) string {
	header_tmpl := "Sentry sentry_timestamp=%s, sentry_client=%s, sentry_version=%0.1f, sentry_key=%s"
	return fmt.Sprintf(header_tmpl, timestamp, client_id, protocol, api_key)
}

func get_signature(message string, str_ts string, key string) string {
	return hmac_sha1([]byte(fmt.Sprintf("%s %s", str_ts, message)), []byte(key))
}

type PrepOutDataError struct {
	When time.Time
	What string
}

func (e PrepOutDataError) Error() string {
	return fmt.Sprintf("%v: %v", e.When, e.What)
}

type MissingPassword struct {
}

func (e MissingPassword) Error() string {
	return "No password was found in the DSN URI"
}

func compute_headers(message string, uri *url.URL, timestamp time.Time) (string, error) {
	password, ok := uri.User.Password()
	if !ok {
		return "", MissingPassword{}
	}

	// TODO: str_ts needs to be pulled into the sentryMsg
	// as it's a temp variable
	var str_ts string
	str_ts = timestamp.Format(time.RFC3339Nano)

	return get_auth_header(2.0,
		get_signature(message, str_ts, password),
		str_ts,
		"raven-go/1.0",
		uri.User.Username()), nil
}

func (self *SentryOutputWriter) Init(config interface{}) error {
	self.udpMap = make(map[string]net.Conn)
	return nil
}

func (self *SentryOutputWriter) MakeOutData() interface{} {
	raw_bytes := make([]byte, 0, MAX_SENTRY_BYTES)
	return &SentryMsg{data_packet: raw_bytes}
}

func (self *SentryOutputWriter) ZeroOutData(outData interface{}) {
	// Just zero out the byte array
	msg := outData.(*SentryMsg)
	msg.data_packet = msg.data_packet[:0]
}

func (self *SentryOutputWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{}, timeout *time.Duration) error {

	sentryMsg := outData.(*SentryMsg)

	sentryMsg.encoded_payload = pack.Message.Payload

	sentryMsg.epoch_ts64, sentryMsg.prep_bool = pack.Message.Fields["epoch_timestamp"].(float64)

	if !sentryMsg.prep_bool {
		log.Printf("Error parsing epoch_timestamp")
		return PrepOutDataError{time.Now(), "Error parsing epoch_timestamp"}
	}

	sentryMsg.epoch_time = time.Unix(int64(sentryMsg.epoch_ts64),
		int64((sentryMsg.epoch_ts64-float64(int64(sentryMsg.epoch_ts64)))*1e9))

	sentryMsg.dsn = pack.Message.Fields["dsn"].(string)

	sentryMsg.parsed_dsn, sentryMsg.prep_error = url.Parse(sentryMsg.dsn)
	if sentryMsg.prep_error != nil {
		log.Printf("Error parsing DSN")
		return sentryMsg.prep_error
	}

	sentryMsg.auth_header, sentryMsg.prep_error = compute_headers(sentryMsg.encoded_payload,
		sentryMsg.parsed_dsn,
		sentryMsg.epoch_time)

	if sentryMsg.prep_error != nil {
		log.Printf("Invalid DSN: [%s]", sentryMsg.dsn)
		return sentryMsg.prep_error
	}

	// TODO: i think the data_packet is the only thing we really need
	// to keep track of is the data_packet and the UDP host/port
	sentryMsg.data_packet = []byte(fmt.Sprintf("%s\n\n%s", sentryMsg.auth_header, sentryMsg.encoded_payload))
	//fmt.Printf("Preped data packet!: %s\n", string(sentryMsg.data_packet))

	return nil
}

func (self *SentryOutputWriter) Write(outData interface{}) (err error) {
	self.sentryMsg = outData.(*SentryMsg)
	// TODO: add a resolveaddr call here
	// TODO: pull up the socket into something we can stub out for
	// testing
	self.socket, self.host_ok = self.udpMap[self.sentryMsg.parsed_dsn.Host]
	if !self.host_ok {
		self.socket, self.socket_err = net.Dial("udp", self.sentryMsg.parsed_dsn.Host)
		if self.socket_err != nil {
			return self.socket_err
		}
		self.udpMap[self.sentryMsg.parsed_dsn.Host] = self.socket
	}
	self.socket.Write(self.sentryMsg.data_packet)
	return nil
}

func (self *SentryOutputWriter) Event(eventType string) {
	// Don't need to do anything here as sentry is just UDP
	// so we don't need to respond to RELOAD or STOP requests
}
