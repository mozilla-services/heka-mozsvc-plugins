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
	"errors"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/cloudwatch"
	"github.com/feyeleanor/sets"
	"github.com/mozilla-services/heka/message"
	"github.com/mozilla-services/heka/pipeline"
	"log"
	"time"
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

type CloudwatchInput struct {
	cw           *cloudwatch.CloudWatch
	req          *cloudwatch.GetMetricStatisticsRequest
	pollInterval time.Duration
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
	cwi.cw, err = cloudwatch.NewCloudWatch(auth, region.CloudWatchServicepoint,
		conf.Namespace)
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
				pack = <-ir.InChan()
				pack.Message.SetType("cloudwatch")
				for _, dim = range cwi.req.Dimensions {
					newField(pack, "Dimension."+dim.Name, dim.Value)
				}
				newField(pack, "Period", cwi.req.Period)
				newField(pack, "Average", point.Average)
				newField(pack, "Maximum", point.Maximum)
				newField(pack, "Minimum", point.Minimum)
				newField(pack, "Samplecount", point.SampleCount)
				newField(pack, "Unit", point.Unit)
				newField(pack, "Sum", point.Sum)
				pack.Message.SetUuid(uuid.NewRandom())
				pack.Message.SetTimestamp(point.Timestamp.UTC().UnixNano())
				pack.Message.SetLogger(cwi.req.MetricName)
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

func init() {
	pipeline.RegisterPlugin("CloudwatchInput", func() interface{} {
		return new(CloudwatchInput)
	})
}
