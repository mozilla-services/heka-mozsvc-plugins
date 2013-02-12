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

func add_field(msg *message.Message, field_name string, value interface{}) {
	f, _ := message.NewField(field_name, value, message.Field_RAW)
	msg.AddField(f)
}

func getSentryPack() *pipeline.PipelinePack {
	pipelinePack := getTestPipelinePack()
	pipelinePack.Message.SetType("sentry")
	fTimeStamp, _ := message.NewField("epoch_timestamp", EPOCH_TS, message.Field_UTC_SECONDS)
	fDsn, _ := message.NewField("dsn", DSN, message.Field_RAW)
	pipelinePack.Message.AddField(fTimeStamp)
	pipelinePack.Message.AddField(fDsn)
	pipelinePack.Message.SetPayload(PAYLOAD)
	pipelinePack.Decoded = true
	return pipelinePack
}

func SentryOutputSpec(c gs.Context) {
	c.Specify("verify data_packet bytes", func() {
		var outData *SentryMsg
		pack := getSentryPack()
		msg := pack.Message
		add_field(msg, "epoch_timestamp", EPOCH_TS)
		add_field(msg, "dsn", DSN)

		writer_ptr := &SentryOutputWriter{}
		writer_ptr.Init(writer_ptr.ConfigStruct())
		outData = writer_ptr.MakeOutData().(*SentryMsg)
		err := writer_ptr.PrepOutData(pack, outData, nil)

		c.Expect(err, gs.Equals, nil)

		actual := string(outData.data_packet)
		ts := int64(EPOCH_TS * 1e9)
		t := time.Unix(ts/1e9, ts%1e9)
		expected := fmt.Sprintf("Sentry sentry_timestamp=%s, sentry_client=raven-go/1.0, sentry_version=2.0, sentry_key=username\n\n%s", t.Format(time.RFC3339Nano), PAYLOAD)

		c.Expect(actual, gs.Equals, expected)
	})

	c.Specify("missing or invalid epoch_timestamp doesn't kill process", func() {
		var outData *SentryMsg
		var err error

		pack := getSentryPack()
		writer_ptr := &SentryOutputWriter{}
		writer_ptr.Init(writer_ptr.ConfigStruct())
		outData = writer_ptr.MakeOutData().(*SentryMsg)

<<<<<<< HEAD
		f := pack.Message.FindFirstField("epoch_timestamp")
		*f.Name = "other"
		err = writer_ptr.PrepOutData(pack, outData, nil)
		c.Expect(err.Error(), gs.Equals, "Error: no epoch_timestamp was found in Fields")

		f, _ = message.NewField("epoch_timestamp", "foo", message.Field_RAW)
		pack.Message.AddField(f)
=======
		msg := pack.Message
		add_field(msg, "dsn", DSN)
		add_field(msg, "payload", PAYLOAD)

		err = writer_ptr.PrepOutData(pack, outData, nil)
		c.Expect(err.Error(), gs.Equals, "Error: no epoch_timestamp was found in Fields")

		add_field(msg, "epoch_timestamp", "foo")
>>>>>>> master
		err = writer_ptr.PrepOutData(pack, outData, nil)
		c.Expect(err.Error(), gs.Equals, "Error: epoch_timestamp isn't a float64")
	})

	c.Specify("missing or invalid dsn doesn't kill process", func() {
		var outData *SentryMsg
		var err error

		pack := getSentryPack()
		msg := pack.Message
		add_field(msg, "epoch_timestamp", EPOCH_TS)
		add_field(msg, "payload", PAYLOAD)

		writer_ptr := &SentryOutputWriter{}
		writer_ptr.Init(writer_ptr.ConfigStruct())
		outData = writer_ptr.MakeOutData().(*SentryMsg)
<<<<<<< HEAD
		f := pack.Message.FindFirstField("dsn")
		*f.Name = "other"
=======
>>>>>>> master

		err = writer_ptr.PrepOutData(pack, outData, nil)
		c.Expect(err.Error(), gs.Equals, "Error: no dsn was found in Fields")

<<<<<<< HEAD
		f, _ = message.NewField("dsn", 42, message.Field_RAW)
		pack.Message.AddField(f)
=======
		add_field(msg, "dsn", 42)
>>>>>>> master
		err = writer_ptr.PrepOutData(pack, outData, nil)
		c.Expect(err.Error(), gs.Equals, "Error: dsn isn't a string")
	})

}
