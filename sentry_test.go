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
#   Rob Miller (rmiller@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"code.google.com/p/gomock/gomock"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/pipeline/testsupport"
	plugins_ts "github.com/mozilla-services/heka/plugins/testsupport"
	gs "github.com/rafrombrc/gospec/src/gospec"
)

const (
	DSN      = "http://username:password@localhost/2"
	PAYLOAD  = "not_real_encoded_data"
	EPOCH_TS = 1358969429.508
)

func getSentryPack() (pack *pipeline.PipelinePack) {
	recycleChan := make(chan *pipeline.PipelinePack, 1)
	pack = pipeline.NewPipelinePack(recycleChan)
	pack.Message.SetType("sentry")
	fDsn, _ := message.NewField("dsn", DSN, "uri")
	pack.Message.AddField(fDsn)
	pack.Message.SetPayload(PAYLOAD)
	pack.Decoded = true
	pack.Message.SetTimestamp(int64(EPOCH_TS * 1e9))
	return
}

func SentryOutputSpec(c gs.Context) {

	t := new(pipeline_ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c.Specify("A SentryOutput", func() {
		output := new(SentryOutput)
		output.Init(output.ConfigStruct())

		c.Specify("doesn't die with missing or invalid dsn", func() {
			var err error

			sentryMsg := &SentryMsg{}
			pack := getSentryPack()

			f := pack.Message.FindFirstField("dsn")
			*f.Name = "other"
			err = output.prepSentryMsg(pack, sentryMsg)
			c.Expect(err.Error(), gs.Equals, "no `dsn` field")

			f, _ = message.NewField("dsn", 42, "")
			pack.Message.AddField(f)
			err = output.prepSentryMsg(pack, sentryMsg)
			c.Expect(err.Error(), gs.Equals, "`dsn` isn't a string")

			_, err = output.getClient("http://localhost")
			c.Expect(err.Error(), gs.Equals, "raven: dsn missing public key and/or private key")
		})

		c.Specify("calls CaptureMessage with the payload when it has a dsn", func() {
			oth := plugins_ts.NewOutputTestHelper(ctrl)
			inChan := make(chan *pipeline.PipelinePack, 1)
			oth.MockOutputRunner.EXPECT().InChan().Return(inChan)

			pack := getSentryPack()
			inChan <- pack
			close(inChan)

			output.Run(oth.MockOutputRunner, oth.MockHelper)
		})
	})

}
