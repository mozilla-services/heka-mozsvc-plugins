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
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/crankycoder/g2s"
	"github.com/mozilla-services/heka/pipeline"
)

// Interface that all statsd clients must implement.
type StatsdClient interface {
	IncrementCounter(bucket string, n int)
	IncrementSampledCounter(bucket string, n int, srate float32)
	SendTiming(bucket string, ms int)
	SendSampledTiming(bucket string, ms int, srate float32)
}

type StatsdMsg struct {
	msgType string
	key     string
	value   int
	rate    float32
}

type StatsdOutput struct {
	statsdClient StatsdClient
	statsdMsg    *StatsdMsg
	err          error
}

type StatsdOutputConfig struct {
	Url string
}

func (so *StatsdOutput) ConfigStruct() interface{} {
	// Default the statsd output to localhost port 5555
	return &StatsdOutputConfig{Url: "localhost:5555"}
}

func (so *StatsdOutput) Init(config interface{}) (err error) {
	conf := config.(*StatsdOutputConfig)
	so.statsdClient, err = g2s.NewStatsd(conf.Url, 0)
	return
}

func (so *StatsdOutput) prepStatsdMsg(pack *pipeline.PipelinePack,
	statsdMsg *StatsdMsg) (err error) {

	// we need the ns for the full key
	ns := pack.Message.GetLogger()

	var tmp interface{}
	var ok bool
	var key string
	var rate64 float64

	if tmp, ok = pack.Message.GetFieldValue("name"); !ok {
		return errors.New("statsd message missing stat name")
	}
	if key, ok = tmp.(string); !ok {
		return errors.New("statsd message stat name is not a string")
	}

	if strings.TrimSpace(ns) != "" {
		s := []string{ns, key}
		key = strings.Join(s, ".")
	}

	var val64 int64
	if val64, err = strconv.ParseInt(pack.Message.GetPayload(), 10, 32); err != nil {
		return fmt.Errorf("can't parse statsd message payload '%s': %s",
			pack.Message.GetPayload(), err)
	}
	value := int(val64)

	if tmp, ok = pack.Message.GetFieldValue("rate"); !ok {
		return errors.New("statsd message missing rate value")
	}
	if rate64, ok = tmp.(float64); !ok {
		return errors.New("statsd message rate is not a float")
	}
	rate := float32(rate64)

	// Set all the statsdMsg attributes
	statsdMsg.msgType = pack.Message.GetType()
	statsdMsg.key = key
	statsdMsg.value = value
	statsdMsg.rate = rate
	return
}

func (so *StatsdOutput) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {

	var (
		e    error
		pack *pipeline.PipelinePack
	)
	statsdMsg := new(StatsdMsg)

	for pack = range or.InChan() {
		e = so.prepStatsdMsg(pack, statsdMsg)
		or.UpdateCursor(pack.QueueCursor)
		pack.Recycle(e)
		if e != nil {
			continue
		}

		switch statsdMsg.msgType {
		case "counter":
			if statsdMsg.rate == 1 {
				so.statsdClient.IncrementCounter(statsdMsg.key, statsdMsg.value)
			} else {
				so.statsdClient.IncrementSampledCounter(statsdMsg.key,
					statsdMsg.value, statsdMsg.rate)
			}
		case "timer":
			if statsdMsg.rate == 1 {
				so.statsdClient.SendTiming(statsdMsg.key, statsdMsg.value)
			} else {
				so.statsdClient.SendSampledTiming(statsdMsg.key,
					statsdMsg.value, statsdMsg.rate)
			}
		default:
			or.LogError(fmt.Errorf("unrecognized statsd message type: %s",
				statsdMsg))
		}
	}

	return
}

func init() {
	pipeline.RegisterPlugin("StatsdOutput", func() interface{} {
		return new(StatsdOutput)
	})
}
