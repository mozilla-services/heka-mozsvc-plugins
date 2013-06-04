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
	ts "github.com/mozilla-services/heka-mozsvc-plugins/testsupport"
	"github.com/mozilla-services/heka/message"
	pipeline "github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/testsupport"
	gs "github.com/rafrombrc/gospec/src/gospec"
	"sync"
)

func getStatsdPlc(typeStr string, payload string) (plc *pipeline.PipelineCapture) {
	recycleChan := make(chan *pipeline.PipelinePack, 1)
	pack := pipeline.NewPipelinePack(recycleChan)
	pack.Message.SetType(typeStr)
	pack.Message.SetLogger("thenamespace")
	fName, _ := message.NewField("name", "myname", "")
	fRate, _ := message.NewField("rate", .30, "")
	pack.Message.AddField(fName)
	pack.Message.AddField(fRate)
	pack.Message.SetPayload(payload)
	pack.Decoded = true
	return &pipeline.PipelineCapture{Pack: pack}
}

func StatsdOutputSpec(c gs.Context) {
	t := new(pipeline_ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c.Specify("A StatsdOutput", func() {
		output := new(StatsdOutput)
		config := output.ConfigStruct().(*StatsdOutputConfig)
		output.Init(config)

		timerMsg := &StatsdMsg{msgType: "timer",
			key:   "thenamespace.myname",
			value: 123,
			rate:  float32(.30)}

		decrMsg := &StatsdMsg{msgType: "counter",
			key:   "thenamespace.myname",
			value: -1,
			rate:  float32(.30)}

		c.Specify("writes", func() {
			mockStatsdClient := ts.NewMockStatsdClient(ctrl)
			output.statsdClient = mockStatsdClient

			oth := ts.NewOutputTestHelper(ctrl)
			inChan := make(chan *pipeline.PipelineCapture, 1)
			oth.MockOutputRunner.EXPECT().InChan().Return(inChan)

			var wg sync.WaitGroup

			c.Specify("a decr msg", func() {
				plc := getStatsdPlc("counter", "-1")
				pack := plc.Pack
				msg := new(StatsdMsg)
				err := output.prepStatsdMsg(pack, msg)
				c.Expect(err, gs.IsNil)
				c.Expect(*msg, gs.Equals, *decrMsg)

				mockStatsdClient.EXPECT().IncrementSampledCounter("thenamespace.myname",
					-1, float32(.30))
				inChan <- plc
				close(inChan)
				wg.Add(1)

				go func() {
					output.Run(oth.MockOutputRunner, oth.MockHelper)
					wg.Done()
				}()

				wg.Wait()
			})

			c.Specify("a timer msg", func() {
				plc := getStatsdPlc("timer", "123")
				pack := plc.Pack
				msg := new(StatsdMsg)
				err := output.prepStatsdMsg(pack, msg)
				c.Expect(err, gs.IsNil)
				c.Expect(*msg, gs.Equals, *timerMsg)

				mockStatsdClient.EXPECT().SendSampledTiming("thenamespace.myname",
					123, float32(.30))
				inChan <- plc
				close(inChan)
				wg.Add(1)

				go func() {
					output.Run(oth.MockOutputRunner, oth.MockHelper)
					wg.Done()
				}()

				wg.Wait()
			})
		})
	})
}
