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
	ts "./testsupport"
	"code.google.com/p/gomock/gomock"
	"github.com/mozilla-services/heka/message"
	pipeline "github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/testsupport"
	gs "github.com/rafrombrc/gospec/src/gospec"
)

func getStatsdPipelinePack(typeStr string, payload string) *pipeline.PipelinePack {
	pipelinePack := getTestPipelinePack()
	*pipelinePack.Message.Type = typeStr
	*pipelinePack.Message.Logger = "thenamespace"
	fName, _ := message.NewField("name", "myname", message.Field_RAW)
	fRate, _ := message.NewField("rate", .30, message.Field_RAW)
	pipelinePack.Message.AddField(fName)
	pipelinePack.Message.AddField(fRate)
	*pipelinePack.Message.Payload = payload
	pipelinePack.Decoded = true
	return pipelinePack
}

func StatsdOutWriterSpec(c gs.Context) {
	t := new(pipeline_ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c.Specify("A StatsdOutWriter", func() {
		statsdWriter := new(StatsdOutWriter)
		config := statsdWriter.ConfigStruct().(*StatsdOutWriterConfig)
		statsdWriter.Init(config)

		c.Specify("creates a *StatsdMsg for output", func() {
			outData := statsdWriter.MakeOutData()
			_, ok := outData.(*StatsdMsg)
			c.Expect(ok, gs.IsTrue)
		})

		timerMsg := &StatsdMsg{msgType: "timer",
			key:   "thenamespace.myname",
			value: 123,
			rate:  float32(.30)}

		decrMsg := &StatsdMsg{msgType: "counter",
			key:   "thenamespace.myname",
			value: -1,
			rate:  float32(.30)}

		c.Specify("correctly preps decr message", func() {
			pipelinePack := getStatsdPipelinePack("counter", "-1")
			msg := new(StatsdMsg)
			err := statsdWriter.PrepOutData(pipelinePack, msg, nil)
			c.Expect(err, gs.IsNil)
			c.Expect(*msg, gs.Equals, *decrMsg)
		})

		c.Specify("correctly preps timer message", func() {
			pipelinePack := getStatsdPipelinePack("timer", "123")
			msg := new(StatsdMsg)
			err := statsdWriter.PrepOutData(pipelinePack, msg, nil)
			c.Expect(err, gs.IsNil)
			c.Expect(*msg, gs.Equals, *timerMsg)
		})

		c.Specify("writes", func() {
			mockStatsdClient := ts.NewMockStatsdClient(ctrl)
			statsdWriter.statsdClient = mockStatsdClient

			c.Specify("a counter msg", func() {
				mockStatsdClient.EXPECT().IncrementSampledCounter("thenamespace.myname",
					-1, float32(.30))
				err := statsdWriter.Write(decrMsg)
				c.Expect(err, gs.IsNil)
			})

			c.Specify("a timer msg", func() {
				mockStatsdClient.EXPECT().SendSampledTiming("thenamespace.myname",
					123, float32(.30))
				err := statsdWriter.Write(timerMsg)
				c.Expect(err, gs.IsNil)
			})
		})
	})
}
