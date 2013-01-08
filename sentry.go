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

import "crypto/hmac"
import "crypto/sha1"
import "encoding/hex"
import "net/url"
import "net"
import "fmt"

// CheckMAC returns true if messageMAC is a valid HMAC tag for
// message.
func hmac_sha1(message, key []byte) string {
	mac := hmac.New(sha1.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hex.EncodeToString(expectedMAC)
}

type SentryOutWriterConfig struct {
	DSN string
}

type SentryOutWriter struct {
	DSN string
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
	headers["Content-Type"] = "application/octet-stream"
	return headers, nil
}

func (self *SentryOutWriter) Init(config interface{}) (err error) {
	conf := config.(*SentryOutWriterConfig)
	self.DSN = conf.DSN
	// TODO: instantiate the actual raven client
	return nil
}
