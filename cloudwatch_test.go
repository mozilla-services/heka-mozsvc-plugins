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
#   Ben Bangert (bbangert@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"code.google.com/p/go-uuid/uuid"
	"code.google.com/p/gomock/gomock"
	ts "github.com/mozilla-services/heka-mozsvc-plugins/testsupport"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/testsupport"
	gs "github.com/rafrombrc/gospec/src/gospec"
	"time"
)

type InputTestHelper struct {
	Msg             *message.Message
	Pack            *pipeline.PipelinePack
	AddrStr         string
	ResolvedAddrStr string
	MockHelper      *pipeline.MockPluginHelper
	MockInputRunner *pipeline.MockInputRunner
	MockDecoderSet  *pipeline.MockDecoderSet
	Decoders        []pipeline.DecoderRunner
	PackSupply      chan *pipeline.PipelinePack
	DecodeChan      chan *pipeline.PipelinePack
}

func getTestMessage() *Message {
	hostname, _ := os.Hostname()
	field, _ := message.NewField("foo", "bar", "")
	msg := &message.Message{}
	msg.SetType("TEST")
	msg.SetTimestamp(time.Now().UnixNano())
	msg.SetUuid(uuid.NewRandom())
	msg.SetLogger("GoSpec")
	msg.SetSeverity(int32(6))
	msg.SetPayload("Test Payload")
	msg.SetEnvVersion("0.8")
	msg.SetPid(int32(os.Getpid()))
	msg.SetHostname(hostname)
	msg.AddField(field)
	return msg
}

func CloudwatchInputSpec(c gs.Context) {
	t := new(pipeline_ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c.Specify("A CloudwatchInput", func() {
		input := new(CloudwatchInput)
		config := input.ConfigStruct().(*CloudwatchInputConfig)
		config.MetricName = "Test"
		config.Statistics = []string{"Average"}
		config.PollInterval = "10s"
		config.Region = "us-east-1"
		err := input.Init(config)
		c.Assume(err, gs.IsNil)
		serv := ts.NewMockAWSService(ctrl)
		input.cw.Service = serv

		ith := new(InputTestHelper)

        ith.Msg = getTestMessage()
        ith.Pack = NewPipelinePack(config.inputRecycleChan)

        // set up mock helper, decoder set, and packSupply channel
        ith.MockHelper = NewMockPluginHelper(ctrl)
        ith.MockInputRunner = NewMockInputRunner(ctrl)
        ith.Decoders = make([]pipeline.DecoderRunner, int(message.Header_JSON+1))
        ith.Decoders[message.Header_PROTOCOL_BUFFER] = NewMockDecoderRunner(ctrl)
        ith.Decoders[message.Header_JSON] = NewMockDecoderRunner(ctrl)
        ith.PackSupply = make(chan *PipelinePack, 1)
        ith.DecodeChan = make(chan *PipelinePack)
        ith.MockDecoderSet = NewMockDecoderSet(ctrl)

        ith.MockInputRunner.EXPECT().InChan().Return(ith.PackSupply)
        ith.MockHelper.EXPECT().DecoderSet().Times(2).Return(ith.MockDecoderSet)
        encCall := ith.MockDecoderSet.EXPECT().ByEncodings()
        encCall.Return(ith.Decoders, nil)
        ith.Msg = getTestMessage()
        ith.Pack = NewPipelinePack(config.inputRecycleChan)

        // set up mock helper, decoder set, and packSupply channel
        ith.MockHelper = NewMockPluginHelper(ctrl)
        ith.MockInputRunner = NewMockInputRunner(ctrl)
        ith.Decoders = make([]DecoderRunner, int(message.Header_JSON+1))
        ith.Decoders[message.Header_PROTOCOL_BUFFER] = NewMockDecoderRunner(ctrl)
        ith.Decoders[message.Header_JSON] = NewMockDecoderRunner(ctrl)
        ith.PackSupply = make(chan *PipelinePack, 1)
        ith.DecodeChan = make(chan *PipelinePack)
        ith.MockDecoderSet = NewMockDecoderSet(ctrl)

        ith.MockInputRunner.EXPECT().InChan().Return(ith.PackSupply)
        ith.MockHelper.EXPECT().DecoderSet().Times(2).Return(ith.MockDecoderSet)
        encCall := ith.MockDecoderSet.EXPECT().ByEncodings()
        encCall.Return(ith.Decoders, nil)

		})
	})
}
