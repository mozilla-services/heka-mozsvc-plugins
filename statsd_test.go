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
	plugin_ts "./testsupport"
	"code.google.com/p/gomock/gomock"
	"github.com/rafrombrc/go-notify"
	gs "github.com/rafrombrc/gospec/src/gospec"
	pipeline "heka/pipeline"
	ts "heka/testsupport"
)

func getStatsdOutput() pipeline.Output {
	pipeline.AvailablePlugins["StatsdOutput"] = func() pipeline.Plugin {
		return new(StatsdOutput)
	}

	plugin := pipeline.AvailablePlugins["StatsdOutput"]()
	statsdOutput, ok := plugin.(*StatsdOutput)
	if !ok {
		return nil
	}
	return statsdOutput
}

func getIncrPipelinePack() *pipeline.PipelinePack {
	pipelinePack := getTestPipelinePack()

	fields := make(map[string]interface{})
	pipelinePack.Message.Fields = fields

	// Force the message to be a statsd increment message
	pipelinePack.Message.Logger = "thenamespace"
	pipelinePack.Message.Fields["name"] = "myname"
	pipelinePack.Message.Fields["rate"] = float64(30.0)
	pipelinePack.Message.Fields["type"] = "counter"
	pipelinePack.Message.Payload = "-1"
	return pipelinePack
}

func StatsdOutputsSpec(c gs.Context) {
	origPoolSize := pipeline.PoolSize
	pipeline.NewPipelineConfig(1)
	defer func() {
		pipeline.PoolSize = origPoolSize
	}()

	t := new(ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c.Specify("A StatsdOutput", func() {

		statsdOutput := new(StatsdOutput)
		defer notify.StopAll(pipeline.STOP)
		config := statsdOutput.ConfigStruct().(*StatsdOutputConfig)

		statsdOutput.Init(config)

		pipelinePack := getIncrPipelinePack()
		pipelinePack.Decoded = true

		// The tests are littered w/ scheduler yields (i.e. runtime.Gosched()
		// calls) so we give the output a chance to respond to the messages
		// we're sending.

		expected_msg := &StatsdMsg{msgType: "counter",
			key:   "thenamespace.myname",
			value: -1,
			rate:  float32(30)}

		c.Specify("pipelinepack is converted to statsdmsg for outputwriter", func() {
			origWriteRunner := statsdOutput.writeRunner

			statsdUrl := config.Url

			defer func() {
				statsdOutput.writeRunner = origWriteRunner
				StatsdWriteRunners[statsdUrl] = origWriteRunner
			}()
			mock_writeRunner := NewMockStatsdWriteRunner(ctrl)
			statsdOutput.writeRunner = mock_writeRunner
			StatsdWriteRunners[statsdUrl] = mock_writeRunner

			mock_writeRunner.EXPECT().RetrieveDataObject()
			mock_writeRunner.EXPECT().SendOutputData(expected_msg)
			statsdOutput.Deliver(pipelinePack)
		})

		c.Specify("StatsdMsg through the StatsdWriter will hit statsd", func() {
			// Note that underscores are magically ignored by the
			// compiler if you don't reference them later
			statsdWriter, _ := StatsdDial("udp://localhost:5000")

			orig_statsdclient := statsdWriter.MyStatsdClient
			defer func() {
				statsdWriter.MyStatsdClient = orig_statsdclient
			}()

			// ok, clobber the statsdclient with a mock
			mock_statsd := plugin_ts.NewMockStatsdClient(ctrl)
			statsdWriter.MyStatsdClient = mock_statsd

			// assert the increment method is called
			mock_statsd.EXPECT().IncrementSampledCounter("thenamespace.myname", -1, float32(30))

			// deliver some data to the writer
			statsdWriter.Write(expected_msg)

		})
	})

}
