/***** BEGIN LICENSE BLOCK *****
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this file,
# You can obtain one at http://mozilla.org/MPL/2.0/.
#
# The Initial Developer of the Original Code is the Mozilla Foundation.
# Portions created by the Initial Developer are Copyright (C) 2012-2015
# the Initial Developer. All Rights Reserved.
#
# Contributor(s):
#   Victor Ng (vng@mozilla.com)
#   Rob Miller (rmiller@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"sync"

	ts "github.com/mozilla-services/heka-mozsvc-plugins/testsupport"
	"github.com/mozilla-services/heka/message"
	pipeline "github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/pipeline/testsupport"
	plugins_ts "github.com/mozilla-services/heka/plugins/testsupport"
	"github.com/rafrombrc/gomock/gomock"
	gs "github.com/rafrombrc/gospec/src/gospec"
)

func getStatsdPack(typeStr string, payload string) (pack *pipeline.PipelinePack) {
	recycleChan := make(chan *pipeline.PipelinePack, 1)
	pack = pipeline.NewPipelinePack(recycleChan)
	pack.Message.SetType(typeStr)
	pack.Message.SetLogger("thenamespace")
	fName, _ := message.NewField("name", "myname", "")
	fRate, _ := message.NewField("rate", .30, "")
	pack.Message.AddField(fName)
	pack.Message.AddField(fRate)
	pack.Message.SetPayload(payload)
	return pack
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

			oth := plugins_ts.NewOutputTestHelper(ctrl)
			inChan := make(chan *pipeline.PipelinePack, 1)
			oth.MockOutputRunner.EXPECT().InChan().Return(inChan)
			oth.MockOutputRunner.EXPECT().UpdateCursor("").AnyTimes()

			var wg sync.WaitGroup

			c.Specify("a decr msg", func() {
				pack := getStatsdPack("counter", "-1")
				msg := new(StatsdMsg)
				err := output.prepStatsdMsg(pack, msg)
				c.Expect(err, gs.IsNil)
				c.Expect(*msg, gs.Equals, *decrMsg)

				mockStatsdClient.EXPECT().IncrementSampledCounter("thenamespace.myname",
					-1, float32(.30))
				inChan <- pack
				close(inChan)
				wg.Add(1)

				go func() {
					output.Run(oth.MockOutputRunner, oth.MockHelper)
					wg.Done()
				}()

				wg.Wait()
			})

			c.Specify("a timer msg", func() {
				pack := getStatsdPack("timer", "123")
				msg := new(StatsdMsg)
				err := output.prepStatsdMsg(pack, msg)
				c.Expect(err, gs.IsNil)
				c.Expect(*msg, gs.Equals, *timerMsg)

				mockStatsdClient.EXPECT().SendSampledTiming("thenamespace.myname",
					123, float32(.30))
				inChan <- pack
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
