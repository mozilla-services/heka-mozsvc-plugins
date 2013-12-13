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
	"fmt"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	gs "github.com/rafrombrc/gospec/src/gospec"
	"time"
)

const (
	DSN      = "udp://username:password@localhost:9001/2"
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
	pack.Message.SetTimestamp(int64(EPOCH_TS * 1e9))
	pack.Decoded = true
	return
}

func SentryOutputSpec(c gs.Context) {

	pack := getSentryPack()
	output := new(SentryOutput)
	output.Init(output.ConfigStruct())
	sentryMsg := &SentryMsg{
		dataPacket: make([]byte, 0, output.config.MaxSentryBytes),
	}
	var err error

	c.Specify("verify data_packet bytes", func() {
		err = output.prepSentryMsg(pack, sentryMsg)
		c.Expect(err, gs.Equals, nil)

		actual := string(sentryMsg.dataPacket)
		ts := int64(EPOCH_TS * 1e9)
		t := time.Unix(ts/1e9, ts%1e9)
		expected := fmt.Sprintf("Sentry sentry_timestamp=%s, "+
			"sentry_client=raven-go/1.0, sentry_version=2.0, "+
			"sentry_key=username\n\n%s", t.Format(time.RFC3339Nano), PAYLOAD)

		c.Expect(actual, gs.Equals, expected)
	})

	c.Specify("missing or invalid dsn doesn't kill process", func() {
		f := pack.Message.FindFirstField("dsn")
		*f.Name = "other"
		err = output.prepSentryMsg(pack, sentryMsg)
		c.Expect(err.Error(), gs.Equals, "no `dsn` field")

		f, _ = message.NewField("dsn", 42, "")
		pack.Message.AddField(f)
		err = output.prepSentryMsg(pack, sentryMsg)
		c.Expect(err.Error(), gs.Equals, "`dsn` isn't a string")
	})

}
