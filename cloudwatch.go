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
#   Rob Miller (rmiller@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/cloudwatch"
	"github.com/feyeleanor/sets"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	"github.com/pborman/uuid"
)

var validMetricStatistics = sets.SSet(
	"Average",
	"Sum",
	"SampleCount",
	"Maximum",
	"Minimum",
)

var validUnits = sets.SSet(
	"Seconds",
	"Microseconds",
	"Milliseconds",
	"Bytes",
	"Kilobytes",
	"Megabytes",
	"Gigabytes",
	"Terabytes",
	"Bits",
	"Kilobits",
	"Megabits",
	"Gigabits",
	"Terabits",
	"Percent",
	"Count",
	"Bytes/Second",
	"Kilobytes/Second",
	"Megabytes/Second",
	"Gigabytes/Second",
	"Terabytes/Second",
	"Bits/Second",
	"Kilobits/Second",
	"Megabits/Second",
	"Gigabits/Second",
	"Terabits/Second",
	"Count/Second",
)

// Cloudwatch Input Config
type CloudwatchInputConfig struct {
	// AWS Secret Key
	SecretKey string `toml:"secret_key"`
	// AWS Access Key
	AccessKey string `toml:"access_key"`
	// AWS Region, ie. us-west-1, eu-west-1
	Region string
	// Cloudwatch Namespace, ie. AWS/Billing, AWS/DynamoDB, custom...
	Namespace string
	// List of dimensions to query
	Dimensions map[string]string
	// Metric name
	MetricName string `toml:"metric_name"`
	// Unit
	Unit string
	// Period for data points, must be factor of 60
	Period int
	// Polling period, how often as a duration to poll for new
	// metrics. The first poll on startup will not be done until this
	// period is reached.
	PollInterval string `toml:"poll_interval"`
	// What statistic values to retrieve for the metrics, valid values
	// are Average, Sum, SampleCount, Maximum, Minumum
	Statistics []string
}

// Cloudwatch Output Config
type CloudwatchOutputConfig struct {
	// AWS Secret Key
	SecretKey string `toml:"secret_key"`
	// AWS Access Key
	AccessKey string `toml:"access_key"`
	// AWS Region, ie. us-west-1, eu-west-1
	Region string
	// Cloudwatch Namespace, ie. AWS/Billing, AWS/DynamoDB, custom...
	Namespace string
	// How many retries to attempt if AWS is not responding, increases
	// exponentially until retries are met for a message
	Retries int
	// Metric backlog, how many messages to buffer sending
	Backlog int
	// Time zone in which the timestamps in the text are presumed to be in.
	// Should be a location name corresponding to a file in the IANA Time Zone
	// database (e.g. "America/Los_Angeles"), as parsed by Go's
	// `time.LoadLocation()` function (see
	// http://golang.org/pkg/time/#LoadLocation). Defaults to "UTC". Not
	// required if valid time zone info is in the timestamp itself.
	TimestampLocation string `toml:"timestamp_location"`
}

type CloudwatchInput struct {
	cw           *cloudwatch.CloudWatch
	req          *cloudwatch.GetMetricStatisticsRequest
	pollInterval time.Duration
	namespace    string
	stopChan     chan bool
}

func (cwi *CloudwatchInput) ConfigStruct() interface{} {
	return &CloudwatchInputConfig{Period: 60}
}

func (cwi *CloudwatchInput) Init(config interface{}) (err error) {
	conf := config.(*CloudwatchInputConfig)

	statisticsSet := sets.SSet(conf.Statistics...)
	switch {
	case conf.MetricName == "":
		err = errors.New("No metric name supplied")
	case conf.Period < 60 || conf.Period%60 != 0:
		err = errors.New("Period must be divisible by 60")
	case len(conf.Statistics) < 1:
		err = errors.New("text·2")
	case conf.Unit != "" && !validUnits.Member(conf.Unit):
		err = errors.New("Unit is not a valid value")
	case len(conf.Statistics) < 1:
		err = errors.New("No statistics supplied")
	case validMetricStatistics.Union(statisticsSet).Len() != validMetricStatistics.Len():
		err = errors.New("Invalid statistic values supplied")
	}
	if err != nil {
		return
	}

	dims := make([]cloudwatch.Dimension, 0)
	for k, v := range conf.Dimensions {
		dims = append(dims, cloudwatch.Dimension{k, v})
	}

	auth := aws.Auth{AccessKey: conf.AccessKey, SecretKey: conf.SecretKey}

	cwi.req = &cloudwatch.GetMetricStatisticsRequest{
		MetricName: conf.MetricName,
		Period:     conf.Period,
		Unit:       conf.Unit,
		Statistics: conf.Statistics,
		Dimensions: dims,
		Namespace:  conf.Namespace,
	}
	cwi.pollInterval, err = time.ParseDuration(conf.PollInterval)
	if err != nil {
		return
	}
	region, ok := aws.Regions[conf.Region]
	if !ok {
		err = errors.New("Region of that name not found.")
		return
	}
	cwi.namespace = conf.Namespace
	cwi.cw, err = cloudwatch.NewCloudWatch(auth, region.CloudWatchServicepoint)
	return
}

func newField(pack *pipeline.PipelinePack, name string, value interface{}) {
	var field *message.Field
	var err error
	if field, err = message.NewField(name, value, ""); err == nil {
		pack.Message.AddField(field)
	} else {
		log.Println("CloudwatchInput can't add field: ", name)
	}
}

func (cwi *CloudwatchInput) Run(ir pipeline.InputRunner, h pipeline.PluginHelper) (err error) {
	cwi.stopChan = make(chan bool)
	cwi.req.StartTime = time.Now()
	ticker := time.NewTicker(cwi.pollInterval)

	ok := true
	var (
		resp  *cloudwatch.GetMetricStatisticsResponse
		point cloudwatch.Datapoint
		pack  *pipeline.PipelinePack
		dim   cloudwatch.Dimension
	)

metricLoop:
	for ok {
		select {
		case _, ok = <-cwi.stopChan:
			continue
		case <-ticker.C:
			cwi.req.EndTime = time.Now()
			resp, err = cwi.cw.GetMetricStatistics(cwi.req)
			if err != nil {
				ir.LogError(err)
				err = nil
				continue
			}
			for _, point = range resp.GetMetricStatisticsResult.Datapoints {
				pack, ok = <-ir.InChan()
				if !ok {
					break metricLoop
				}
				pack.Message.SetType("cloudwatch")
				for _, dim = range cwi.req.Dimensions {
					newField(pack, "Dimension."+dim.Name, dim.Value)
				}
				newField(pack, "Period", cwi.req.Period)
				newField(pack, "Average", point.Average)
				newField(pack, "Maximum", point.Maximum)
				newField(pack, "Minimum", point.Minimum)
				newField(pack, "SampleCount", point.SampleCount)
				newField(pack, "Unit", point.Unit)
				newField(pack, "Sum", point.Sum)
				pack.Message.SetUuid(uuid.NewRandom())
				pack.Message.SetTimestamp(point.Timestamp.UTC().UnixNano())
				pack.Message.SetLogger(cwi.namespace)
				pack.Message.SetPayload(cwi.req.MetricName)
				ir.Inject(pack)
			}
			cwi.req.StartTime = cwi.req.EndTime.Add(time.Duration(1) * time.Nanosecond)
		}
	}
	return nil
}

func (cwi *CloudwatchInput) Stop() {
	close(cwi.stopChan)
}

type JsonDatum struct {
	Dimensions      []cloudwatch.Dimension
	MetricName      string
	StatisticValues *cloudwatch.StatisticSet
	Timestamp       string
	Unit            string
	Value           float64
}

type CloudwatchDatapointPayload struct {
	Datapoints []JsonDatum
}

type CloudwatchDatapoints struct {
	Datapoints  []cloudwatch.MetricDatum
	QueueCursor string
}

type CloudwatchOutput struct {
	cw         *cloudwatch.CloudWatch
	retries    int
	backlog    int
	stopChan   chan bool
	tzLocation *time.Location
	namespace  string
}

func (cwo *CloudwatchOutput) ConfigStruct() interface{} {
	return &CloudwatchOutputConfig{Retries: 3, Backlog: 10}
}

func (cwo *CloudwatchOutput) Init(config interface{}) (err error) {
	conf := config.(*CloudwatchOutputConfig)
	auth := aws.Auth{AccessKey: conf.AccessKey, SecretKey: conf.SecretKey}
	cwo.stopChan = make(chan bool)
	region, ok := aws.Regions[conf.Region]
	if !ok {
		err = errors.New("Region of that name not found.")
		return
	}
	cwo.backlog = conf.Backlog
	cwo.retries = conf.Retries
	if cwo.cw, err = cloudwatch.NewCloudWatch(auth, region.CloudWatchServicepoint); err != nil {
		return
	}
	cwo.namespace = conf.Namespace
	if cwo.tzLocation, err = time.LoadLocation(conf.TimestampLocation); err != nil {
		err = fmt.Errorf("CloudwatchOutput unknown timestamp_location '%s': %s",
			conf.TimestampLocation, err)
	}
	return
}

func (cwo *CloudwatchOutput) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {
	inChan := or.InChan()

	payloads := make(chan CloudwatchDatapoints, cwo.backlog)
	go cwo.Submitter(payloads, or)

	var (
		pack          *pipeline.PipelinePack
		msg           *message.Message
		rawDataPoints *CloudwatchDatapointPayload
		dataPoints    *CloudwatchDatapoints
	)
	dataPoints = new(CloudwatchDatapoints)
	dataPoints.Datapoints = make([]cloudwatch.MetricDatum, 0, 0)

	for pack = range inChan {
		rawDataPoints = new(CloudwatchDatapointPayload)
		msg = pack.Message
		err = json.Unmarshal([]byte(msg.GetPayload()), rawDataPoints)
		if err != nil {
			err = fmt.Errorf("warning, unable to parse payload: %s", err)
			pack.Recycle(err)
			err = nil
			continue
		}
		// Run through the list and convert them to CloudwatchDatapoints
		for _, rawDatum := range rawDataPoints.Datapoints {
			datum := cloudwatch.MetricDatum{
				Dimensions:      rawDatum.Dimensions,
				MetricName:      rawDatum.MetricName,
				Unit:            rawDatum.Unit,
				Value:           rawDatum.Value,
				StatisticValues: rawDatum.StatisticValues,
			}
			if rawDatum.Timestamp != "" {
				parsedTime, err := message.ForgivingTimeParse("", rawDatum.Timestamp, cwo.tzLocation)
				if err != nil {
					or.LogError(fmt.Errorf("unable to parse timestamp for datum: %s", rawDatum))
					continue
				}
				datum.Timestamp = parsedTime
			}
			dataPoints.Datapoints = append(dataPoints.Datapoints, datum)
		}
		dataPoints.QueueCursor = pack.QueueCursor
		payloads <- *dataPoints
		dataPoints.Datapoints = dataPoints.Datapoints[:0]
		rawDataPoints.Datapoints = rawDataPoints.Datapoints[:0]
		pack.Recycle(nil)
	}
	or.LogMessage("shutting down AWS Cloudwatch submitter")
	cwo.stopChan <- true
	<-cwo.stopChan
	return
}

func (cwo *CloudwatchOutput) Submitter(payloads chan CloudwatchDatapoints,
	or pipeline.OutputRunner) {
	var (
		payload  CloudwatchDatapoints
		curTry   int
		backOff  time.Duration = time.Duration(10) * time.Millisecond
		err      error
		stopping bool
	)
	curDuration := backOff

	for !stopping {
		select {
		case stopping = <-cwo.stopChan:
			continue
		case payload = <-payloads:
			for curTry < cwo.retries {
				_, err = cwo.cw.PutMetricDataNamespace(payload.Datapoints, cwo.namespace)
				if err != nil {
					curTry += 1
					time.Sleep(curDuration)
					curDuration *= 2
				} else {
					or.UpdateCursor(payload.QueueCursor)
					break
				}
			}
			curDuration = backOff
			curTry = 0
			if err != nil {
				or.LogError(err)
				err = nil
			}
		}
	}

	close(cwo.stopChan)
}

func init() {
	pipeline.RegisterPlugin("CloudwatchInput", func() interface{} {
		return new(CloudwatchInput)
	})
	pipeline.RegisterPlugin("CloudwatchOutput", func() interface{} {
		return new(CloudwatchOutput)
	})
}
