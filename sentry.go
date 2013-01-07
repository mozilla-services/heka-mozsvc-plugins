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

func (self *SentryOutWriter) get_auth_header(protocol float32, signature string, timestamp string, client_id string, api_key string) string {
	header_tmpl := "Sentry sentry_timestamp=%s, sentry_client=%s, sentry_version=%0.1f, sentry_key=%s"
	return fmt.Sprintf(header_tmpl, timestamp, client_id, protocol, api_key)
}

func (self *SentryOutWriter) compute_headers(message string, uri url.URL, timestamp string) (map[string]string, bool) {

	// TODO: uncomment
	//password, ok := uri.User.Password()
	//if !ok {
	//return nil, ok
	//}

	//client_version := 1.0

	headers := make(map[string]string)
	headers["X-Sentry-Auth"] = "blah"
	//get_auth_header( protocol=2.0, signature=get_signature(message, timestamp, password), timestamp=timestamp, client_id="raven-logstash/#{client_version}", api_key=uri.user)
	headers["Content-Type"] = "application/octet-stream"
	return headers, true
}

func (self *SentryOutWriter) Init(config interface{}) (err error) {
	conf := config.(*SentryOutWriterConfig)
	self.DSN = conf.DSN
	// TODO: instantiate the actual raven client
	return nil
}
