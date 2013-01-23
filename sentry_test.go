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
	gs "github.com/rafrombrc/gospec/src/gospec"
)

const (
	DSN      = "udp://username:password@localhost:9001/2"
	PAYLOAD  = "not_real_encoded_data"
	EPOCH_TS = 1358969429.508007
)

func getSentryPack() *pipeline.PipelinePack {
	pipelinePack := getTestPipelinePack()
	pipelinePack.Message.Type = "sentry"
	pipelinePack.Message.Fields = make(map[string]interface{})
	pipelinePack.Message.Fields["epoch_timestamp"] = EPOCH_TS
	pipelinePack.Message.Fields["dsn"] = DSN
	pipelinePack.Message.Payload = PAYLOAD
	pipelinePack.Decoded = true
	return pipelinePack
}

func SentryOutputSpec(c gs.Context) {
	c.Specify("verify data_packet bytes", func() {
		var outData *SentryMsg
		pack := getSentryPack()
		expected := fmt.Sprintf("Sentry sentry_timestamp=2013-01-23T14:30:29.508007049-05:00, sentry_client=raven-go/1.0, sentry_version=2.0, sentry_key=username\n\n%s", PAYLOAD)

		writer_ptr := &SentryOutputWriter{}
		writer_ptr.Init(writer_ptr.ConfigStruct())
		outData = writer_ptr.MakeOutData().(*SentryMsg)

		writer_ptr.PrepOutData(pack, outData, nil)
		actual := string(outData.data_packet)
		c.Expect(actual, gs.Equals, expected)
	})

}
