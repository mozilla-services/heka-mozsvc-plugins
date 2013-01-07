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
	gs "github.com/rafrombrc/gospec/src/gospec"
)

func SentryOutWriterSpec(c gs.Context) {
	c.Specify("check that hmac hashes are correct", func() {

		// The following hexdigest was verified using a Python
		// hmac-sha1 snippet:
		//      hmac.new('this is the key', 'foobar', sha1).hexdigest()
		//      'c7cbdca495adb024647b64123ef8203ae333f0d6'
		expected_hexdigest := "c7cbdca495adb024647b64123ef8203ae333f0d6"

		actual_hexdigest := hmac_sha1([]byte("foobar"), []byte("this is the key"))
		c.Expect(actual_hexdigest, gs.Equals, expected_hexdigest)
	})

	c.Specify("check auth header", func() {
		writer := new(SentryOutWriter)
		actual_header := writer.get_auth_header(2.0, "some_sig", "some_time", "some_client", "some_api_key")
		expected_header := "Sentry sentry_timestamp=some_time, sentry_client=some_client, sentry_version=2.0, sentry_key=some_api_key"
		c.Expect(actual_header, gs.Equals, expected_header)
	})
}
