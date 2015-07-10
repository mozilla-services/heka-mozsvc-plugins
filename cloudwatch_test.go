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
#   Ben Bangert (bbangert@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	ts "github.com/mozilla-services/heka-mozsvc-plugins/testsupport"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	pipeline_ts "github.com/mozilla-services/heka/pipeline/testsupport"
	"github.com/mozilla-services/heka/pipelinemock"
	"github.com/pborman/uuid"
	"github.com/rafrombrc/gomock/gomock"
	gs "github.com/rafrombrc/gospec/src/gospec"
)

type InputTestHelper struct {
	Msg             *message.Message
	Pack            *pipeline.PipelinePack
	AddrStr         string
	ResolvedAddrStr string
	MockHelper      *pipelinemock.MockPluginHelper
	MockInputRunner *pipelinemock.MockInputRunner
	Decoder         pipeline.DecoderRunner
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

var simpleJsonPayload = `
{"Datapoints":[{"MetricName":"Testval","Timestamp":"Fri Jul 12 12:59:52 2013","Value":7.82636926e-06,"Unit":"Kilobytes"}]}
`

var awsSuccessResponse = `
<PutMetricDataResponse xmlns="http://monitoring.amazonaws.com/doc/2010-08-01/">
  <ResponseMetadata>
    <RequestId>6d0916bd-ddfe-11e2-bb4d-cb095c9ec687</RequestId>
  </ResponseMetadata>
</PutMetricDataResponse>
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

	errChan := make(chan error)

	c.Specify("A CloudwatchInput", func() {
		input := new(CloudwatchInput)
		inputConfig := input.ConfigStruct().(*CloudwatchInputConfig)
		inputConfig.MetricName = "Test"
		inputConfig.Statistics = []string{"Average"}
		inputConfig.PollInterval = "1ms"
		inputConfig.Region = "us-east-1"
		inputConfig.Namespace = "Testing"
		err := input.Init(inputConfig)
		c.Assume(err, gs.IsNil)
		serv := ts.NewMockAWSService(ctrl)
		input.cw.Service = serv

		ith := new(InputTestHelper)
		recycleChan := make(chan *pipeline.PipelinePack, 500)

		// set up mock helper, decoder set, and packSupply channel
		ith.MockHelper = pipelinemock.NewMockPluginHelper(ctrl)
		ith.MockInputRunner = pipelinemock.NewMockInputRunner(ctrl)
		ith.Decoder = pipelinemock.NewMockDecoderRunner(ctrl)
		ith.PackSupply = make(chan *pipeline.PipelinePack, 1)
		ith.DecodeChan = make(chan *pipeline.PipelinePack)

		ith.Msg = getTestMessage()
		ith.Pack = pipeline.NewPipelinePack(recycleChan)

		c.Specify("can receive a set of metrics", func() {
			ith.PackSupply <- ith.Pack

			resp := new(http.Response)
			resp.Body = &RespCloser{strings.NewReader(awsResponse)}
			resp.StatusCode = 200

			// Setup the mock response
			ith.MockInputRunner.EXPECT().InChan().Return(ith.PackSupply)
			ith.MockInputRunner.EXPECT().Inject(ith.Pack)
			serv.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil)

			go func() {
				err := input.Run(ith.MockInputRunner, ith.MockHelper)
				errChan <- err
			}()
			ith.PackSupply <- ith.Pack
			close(input.stopChan)
			err = <-errChan
			c.Expect(err, gs.IsNil)
			c.Expect(ith.Pack.Message.GetLogger(), gs.Equals, "Testing")
			c.Expect(ith.Pack.Message.GetPayload(), gs.Equals, "Test")
			val, _ := ith.Pack.Message.GetFieldValue("Unit")
			c.Expect(val.(string), gs.Equals, "Seconds")
			val, _ = ith.Pack.Message.GetFieldValue("SampleCount")
			c.Expect(val.(float64), gs.Equals, float64(837721.0))
		})
	})

	c.Specify("A CloudwatchOutput", func() {
		mockOutputRunner := pipelinemock.NewMockOutputRunner(ctrl)
		mockHelper := pipelinemock.NewMockPluginHelper(ctrl)

		inChan := make(chan *pipeline.PipelinePack, 1)
		recycleChan := make(chan *pipeline.PipelinePack, 1)
		mockOutputRunner.EXPECT().InChan().Return(inChan)
		mockOutputRunner.EXPECT().UpdateCursor("").AnyTimes()

		msg := getTestMessage()
		pack := pipeline.NewPipelinePack(recycleChan)
		pack.Message = msg

		output := new(CloudwatchOutput)
		outputConfig := output.ConfigStruct().(*CloudwatchOutputConfig)
		outputConfig.Retries = 3
		outputConfig.Backlog = 10
		outputConfig.Namespace = "Test"
		outputConfig.Region = "us-east-1"
		err := output.Init(outputConfig)
		c.Assume(err, gs.IsNil)

		serv := ts.NewMockAWSService(ctrl)
		output.cw.Service = serv

		c.Specify("can send a batch of metrics", func() {
			resp := new(http.Response)
			resp.Body = &RespCloser{strings.NewReader(awsSuccessResponse)}
			resp.StatusCode = 200

			pack.Message.SetPayload(simpleJsonPayload)

			serv.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Return(resp, nil)
			mockOutputRunner.EXPECT().LogMessage(gomock.Any())

			inChan <- pack
			go func() {
				err := output.Run(mockOutputRunner, mockHelper)
				errChan <- err
			}()
			<-recycleChan
			close(inChan)
			err = <-errChan
			c.Expect(err, gs.IsNil)
		})

		c.Specify("can retry failed operations", func() {
			resp := new(http.Response)
			resp.Body = &RespCloser{strings.NewReader(awsSuccessResponse)}
			resp.StatusCode = 200
			err := errors.New("Oops, not working")

			pack.Message.SetPayload(simpleJsonPayload)

			serv.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any()).Times(3).Return(resp, err)
			mockOutputRunner.EXPECT().LogMessage(gomock.Any())
			mockOutputRunner.EXPECT().LogError(gomock.Any())

			inChan <- pack
			go func() {
				err := output.Run(mockOutputRunner, mockHelper)
				errChan <- err
			}()
			<-recycleChan
			close(inChan)
			err = <-errChan
			c.Expect(err, gs.IsNil)
		})
	})
}
