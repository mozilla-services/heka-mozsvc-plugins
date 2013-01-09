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
	"heka/pipeline"
	"log"
	"net"
	"net/url"
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

type SentryMsg struct {
	encoded_payload string
	epoch_timestamp string
	dsn             string
	parsed_dsn      *url.URL
	auth_header     string
	prep_error      error
	headers         map[string]string

	// TODO: i think this might be the only thing we really need
	data_packet []byte
}

type SentryOutWriter struct {
	DSN string
}

type SentryOutWriterConfig struct {
	DSN string
}

func (self *SentryOutWriter) ConfigStruct() interface{} {
	// Default the statsd output to localhost port 5555
	return &SentryOutWriterConfig{DSN: "udp://mockuser:mockpassword@localhost:5565"}
}

func get_auth_header(protocol float32, signature string, timestamp string, client_id string, api_key string) string {
	header_tmpl := "Sentry sentry_timestamp=%s, sentry_client=%s, sentry_version=%0.1f, sentry_key=%s"
	return fmt.Sprintf(header_tmpl, timestamp, client_id, protocol, api_key)
}

func get_signature(message, timestamp, key string) string {
	return hmac_sha1([]byte(fmt.Sprintf("%s %s", timestamp, message)), []byte(key))
}

func send(event map[string]interface{}) {
	message := event["payload"].(string)
	field_map := event["fields"].(map[string]interface{})
	timestamp := field_map["epoch_timestamp"].(string)
	dsn := field_map["dsn"].(string)

	dsn_uri, err := url.Parse(dsn)
	if err != nil {
		// TODO: log an error for an invalid DSN
		return
	}

	headers, err := compute_headers(message, dsn_uri, timestamp)
	if err != nil {
		// TODO: log an error for an invalid DSN
		return
	}

	auth_header := headers["X-Sentry-Auth"]

	// TODO: pull the socket up and out into something we can mock
	conn, err := net.Dial("udp", dsn_uri.Host)
	conn.Write([]byte(fmt.Sprintf("%s\n\n%s", auth_header, message)))
}

type MissingPassword struct {
}

func (e MissingPassword) Error() string {
	return "No password was found in the DSN URI"
}

func compute_headers(message string, uri *url.URL, timestamp string) (map[string]string, error) {

	password, ok := uri.User.Password()
	if !ok {
		return nil, MissingPassword{}
	}

	headers := make(map[string]string)
	headers["X-Sentry-Auth"] = get_auth_header(2.0,
		get_signature(message, timestamp, password),
		timestamp,
		"raven-go/1.0",
		uri.User.Username())

	// TODO: I don't think this content-type is actually used
	// anywhere, we can probably ditch the entire map return value
	headers["Content-Type"] = "application/octet-stream"
	return headers, nil
}

func (self *SentryOutWriter) Init(config interface{}) (err error) {
	conf := config.(*SentryOutWriterConfig)
	self.DSN = conf.DSN
	return nil
}

func (self *SentryOutWriter) MakeOutData() interface{} {
	raw_bytes := make([]byte, 0, MAX_SENTRY_BYTES)
	headers := make(map[string]string)
	return SentryMsg{data_packet: raw_bytes}
}

func (self *SentryOutWriter) ZeroOutData(outData interface{}) {
	// Just zero out the byte array
	msg := outData.(*SentryMsg)
	msg.data_packet = msg.data_packet[:0]
}

func (self *SentryOutWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{}) {
	sentryMsg := outData.(*SentryMsg)

	sentryMsg.encoded_payload = pack.Message.Payload
	sentryMsg.epoch_timestamp = pack.Message.Fields["epoch_timestamp"].(string)

	sentryMsg.dsn = pack.Message.Fields["dsn"].(string)

	sentryMsg.parsed_dsn, sentryMsg.prep_error = url.Parse(sentryMsg.dsn)
	if sentryMsg.prep_error != nil {
		log.Printf("Error parsing DSN")
		return
	}

	sentryMsg.headers, sentryMsg.prep_error = compute_headers(sentryMsg.encoded_payload,
		sentryMsg.dsn,
		sentyrMsg.epoch_timestamp)

	if prep_error != nil {
		log.Printf("Invalid DSN: [%s]", sentryMsg.dsn)
		return
	}

	sentryMsg.auth_header = headers["X-Sentry-Auth"]

	// TODO: i think the data_packet is the only thing we really need
	// to keep track of is the data_packet and the UDP host/port
	sentryMsg.data_packet = []byte(fmt.Sprintf("%s\n\n%s", auth_header, message))
}

func (self *SentryOutWriter) Write(outData interface{}) (err error) {
	self.sentryMsg = outData.(*SentryMsg)

	// TODO: pull the socket up and out into something we can mock
	conn, err := net.Dial("udp", self.sentryMsg.parsed_dsn.Host)
	conn.Write(sentryMsg.data_packet)
	return
}

func (self *SentryOutWriter) Event(eventType string) {
	// Don't need to do anything here as sentry is just UDP
}
