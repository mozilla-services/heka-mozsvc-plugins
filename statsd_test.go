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
	"code.google.com/p/gomock/gomock"
	"github.com/rafrombrc/go-notify"
	gs "github.com/rafrombrc/gospec/src/gospec"
	pipeline "heka/pipeline"
	ts "heka/testsupport"
	"runtime"
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

		c.Specify("writes text", func() {
			orig_outputWriter := StatsdWriteRunners[config.Url].TheOutputWriter

			// TODO: Clobber the statsd client in the StatsdWriter

			mockOutputWriter := ts.NewMockOutputWriter(ctrl)
			StatsdWriteRunners[config.Url].TheOutputWriter = mockOutputWriter

			defer func() {
				// Revert the mock
				StatsdWriteRunners[config.Url].TheOutputWriter = orig_outputWriter
			}()

			mockOutputWriter.EXPECT().Write(gomock.Any())
			statsdOutput.Deliver(pipelinePack)
			runtime.Gosched()
			// TODO: check that the mock writer
		})

	})

}
