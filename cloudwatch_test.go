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
	"net/http"
	"os"
	"strings"
	"time"
)

type InputTestHelper struct {
	Msg             *message.Message
	Pack            *pipeline.PipelinePack
	AddrStr         string
	ResolvedAddrStr string
	MockHelper      *ts.MockPluginHelper
	MockInputRunner *ts.MockInputRunner
	MockDecoderSet  *ts.MockDecoderSet
	Decoders        []pipeline.DecoderRunner
	PackSupply      chan *pipeline.PipelinePack
	DecodeChan      chan *pipeline.PipelinePack
}

var awsResponse = `
<GetMetricStatisticsResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <GetMetricStatisticsResult>
    <Datapoints>
      <member>
        <Timestamp>2013-06-25T17:18:00Z</Timestamp>
        <SampleCount>837721.0</SampleCount>
        <Unit>Seconds</Unit>
        <Minimum>8.0E-6</Minimum>
        <Maximum>59.929617</Maximum>
        <Average>0.006249934529515202</Average>
      </member>
    </Datapoints>
    <Label>Latency</Label>
  </GetMetricStatisticsResult>
  <ResponseMetadata>
    <RequestId>6d0916bd-ddfe-11e2-bb4d-cb095c9ec687</RequestId>
  </ResponseMetadata>
</GetMetricStatisticsResponse>
`

func getTestMessage() *message.Message {
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

type RespCloser struct {
	*strings.Reader
}

func (r *RespCloser) Close() error {
	return nil
}

func CloudwatchInputSpec(c gs.Context) {
	t := new(pipeline_ts.SimpleT)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	c.Specify("A CloudwatchInput", func() {
		input := new(CloudwatchInput)
		inputConfig := input.ConfigStruct().(*CloudwatchInputConfig)
		inputConfig.MetricName = "Test"
		inputConfig.Statistics = []string{"Average"}
		inputConfig.PollInterval = "1ms"
		inputConfig.Region = "us-east-1"
		err := input.Init(inputConfig)
		c.Assume(err, gs.IsNil)
		serv := ts.NewMockAWSService(ctrl)
		input.cw.Service = serv

		ith := new(InputTestHelper)
		recycleChan := make(chan *pipeline.PipelinePack, 500)

		// set up mock helper, decoder set, and packSupply channel
		ith.MockHelper = ts.NewMockPluginHelper(ctrl)
		ith.MockInputRunner = ts.NewMockInputRunner(ctrl)
		ith.Decoders = make([]pipeline.DecoderRunner, int(message.Header_JSON+1))
		ith.Decoders[message.Header_PROTOCOL_BUFFER] = ts.NewMockDecoderRunner(ctrl)
		ith.Decoders[message.Header_JSON] = ts.NewMockDecoderRunner(ctrl)
		ith.PackSupply = make(chan *pipeline.PipelinePack, 1)
		ith.DecodeChan = make(chan *pipeline.PipelinePack)
		ith.MockDecoderSet = ts.NewMockDecoderSet(ctrl)

		ith.Msg = getTestMessage()
		ith.Pack = pipeline.NewPipelinePack(recycleChan)

		c.Specify("can recieve a set of metrics", func() {
			ith.PackSupply <- ith.Pack

			resp := new(http.Response)
			resp.Body = &RespCloser{strings.NewReader(awsResponse)}
			resp.StatusCode = 200

			// Setup the mock response
			ith.MockInputRunner.EXPECT().InChan().Return(ith.PackSupply)
			ith.MockInputRunner.EXPECT().Inject(ith.Pack)
			serv.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil)

			go func() {
				input.Run(ith.MockInputRunner, ith.MockHelper)
			}()
			ith.PackSupply <- ith.Pack
			close(input.stopChan)
			c.Expect(ith.Pack.Message.GetLogger(), gs.Equals, "Test")
		})
	})
}
