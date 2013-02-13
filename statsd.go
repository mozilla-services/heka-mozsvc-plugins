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
	"errors"
	"fmt"
	"github.com/crankycoder/g2s"
	"github.com/mozilla-services/heka/pipeline"
	"strconv"
	"strings"
	"time"
)

// Interface that all statsd clients must implement.
type StatsdClient interface {
	IncrementSampledCounter(bucket string, n int, srate float32)
	SendSampledTiming(bucket string, ms int, srate float32)
}

type StatsdMsg struct {
	msgType string
	key     string
	value   int
	rate    float32
}

type StatsdOutWriter struct {
	statsdClient StatsdClient
	statsdMsg    *StatsdMsg
	err          error
}

type StatsdOutWriterConfig struct {
	Url string
}

func (self *StatsdOutWriter) ConfigStruct() interface{} {
	// Default the statsd output to localhost port 5555
	return &StatsdOutWriterConfig{Url: "localhost:5555"}
}

func (self *StatsdOutWriter) Init(config interface{}) (err error) {
	conf := config.(*StatsdOutWriterConfig)
	self.statsdClient, err = g2s.NewStatsd(conf.Url, 0)
	return
}

func (self *StatsdOutWriter) MakeOutData() interface{} {
	return new(StatsdMsg)
}

func (self *StatsdOutWriter) ZeroOutData(outData interface{}) {
	// nothing to do
}

func (self *StatsdOutWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{},
	timeout *time.Duration) (err error) {
	statsdMsg := outData.(*StatsdMsg)

	// we need the ns for the full key
	ns := pack.Message.GetLogger()

	var tmp interface{}
	var ok bool
	var key string
	var rate64 float64

	tmp, ok = pack.Message.GetFieldValue("name")
	if !ok {
		return fmt.Errorf("Error parsing key for statsd from msg.GetFieldValue(\"name\")")
	}
	key, ok = tmp.(string)
	if !ok {
		return fmt.Errorf("statsd name is not a string")
	}

	if strings.TrimSpace(ns) != "" {
		s := []string{ns, key}
		key = strings.Join(s, ".")
	}

	val64, err := strconv.ParseInt(pack.Message.GetPayload(), 10, 32)
	if err != nil {
		err = fmt.Errorf("Error parsing value for statsd: ", err.Error())
		return
	}
	// Downcast this
	value := int(val64)

	tmp, ok = pack.Message.GetFieldValue("rate")
	if !ok {
		err = errors.New("Error parsing key for statsd from msg.GetFieldValue(\"rate\")")
		return
	}

	rate64, ok = tmp.(float64)
	if !ok {
		err = errors.New("Rate isn't a float")
		return
	}
	rate := float32(rate64)

	// Set all the statsdMsg attributes
	statsdMsg.msgType = pack.Message.GetType()
	statsdMsg.key = key
	statsdMsg.value = value
	statsdMsg.rate = rate

	return nil
}

func (self *StatsdOutWriter) Write(outData interface{}) (err error) {
	self.statsdMsg = outData.(*StatsdMsg)
	switch self.statsdMsg.msgType {
	case "counter":
		self.statsdClient.IncrementSampledCounter(self.statsdMsg.key, self.statsdMsg.value,
			self.statsdMsg.rate)
	case "timer":
		self.statsdClient.SendSampledTiming(self.statsdMsg.key, self.statsdMsg.value,
			self.statsdMsg.rate)
	default:
		err = fmt.Errorf("Unexpected event passed into StatsdOutWriter.\nEvent => %+v\n",
			self.statsdMsg)
	}
	return
}

func (self *StatsdOutWriter) Event(eventType string) {
	// Don't need to do anything here as statsd is just UDP
}

func init() {
	pipeline.RegisterPlugin("StatsdOutput", func() interface{} {
		return pipeline.RunnerMaker(new(StatsdOutWriter))
	})
}
