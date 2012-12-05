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
	"fmt"
	"heka/pipeline"
	"log"
	"strconv"
	"strings"
)

// This maps statsd URLs to a writerunner
var StatsdWriteRunners = make(map[string]*pipeline.WriteRunner)

type StatsdOutputWriter struct {
	MyStatsdWriter *StatsdWriter
	statsdMsg      *StatsdMsg
	err            error
}

func NewStatsdOutputWriter(url string) (*StatsdOutputWriter,
	error) {
	statsdWriter, err := StatsdDial(url)
	if err != nil {
		return nil, err
	}
	self := &StatsdOutputWriter{MyStatsdWriter: statsdWriter}
	return self, nil
}

func (self *StatsdOutputWriter) MakeOutputData() interface{} {
	return new(StatsdMsg)
}

func (self *StatsdOutputWriter) Write(outputData interface{}) error {
	self.statsdMsg = outputData.(*StatsdMsg)
	self.MyStatsdWriter.Write(self.statsdMsg)
	return nil
}

func (self *StatsdOutputWriter) Stop() {
	// Don't need to do anything here as statsd is just UDP
}

type StatsdMsg struct {
	msgType string
	key     string
	value   int
	rate    float32
}

type StatsdOutput struct {
	dataChan    chan interface{}
	recycleChan chan interface{}
	statsdMsg   *StatsdMsg

	/* The variables below are used when decoding the ns, key, value
	 * and rate from the pipelinepack
	 */
	ns string

	key    string
	key_ok bool

	tmp_value int64
	value_ok  error
	value     int

	rate     float32
	tmp_rate float64
	rate_ok  bool
}

type StatsdOutputConfig struct {
	Url string
}

func (self *StatsdOutput) ConfigStruct() interface{} {
	// Default the statsd output to localhost port 5555
	return &StatsdOutputConfig{Url: "localhost:5555"}
}

func (self *StatsdOutput) Init(config interface{}) (err error) {
	conf := config.(*StatsdOutputConfig)

	statsdUrl := conf.Url

	// Using a map to guarantee there's only one WriteRunner is only safe b/c
	// the PipelinePacks (and therefore the StatsdOutputs) are initialized in
	// series.
	writeRunner, ok := StatsdWriteRunners[statsdUrl]
	if !ok {
		statsdOutputWriter, err := NewStatsdOutputWriter(statsdUrl)
		if err != nil {
			return fmt.Errorf("Error creating StatsdOutputWriter: %s", err)
		}
		writeRunner = pipeline.NewWriteRunner(statsdOutputWriter)
		StatsdWriteRunners[statsdUrl] = writeRunner
	}
	self.dataChan = writeRunner.DataChan
	self.recycleChan = writeRunner.RecycleChan
	return nil

}

func (self *StatsdOutput) Deliver(pack *pipeline.PipelinePack) {
	self.statsdMsg = (<-self.recycleChan).(*StatsdMsg)

	// we need the ns for the full key
	self.ns = pack.Message.Logger

	self.key, self.key_ok = pack.Message.Fields["name"].(string)
	if self.key_ok == false {
		log.Printf("Error parsing key for statsd from msg.Fields[\"name\"]")
		return
	}

	if strings.TrimSpace(self.ns) != "" {
		s := []string{self.ns, self.key}
		self.key = strings.Join(s, ".")
	}

	self.tmp_value, self.value_ok = strconv.ParseInt(pack.Message.Payload, 10, 32)
	if self.value_ok != nil {
		log.Printf("Error parsing value for statsd")
		return
	}
	// Downcast this
	self.value = int(self.tmp_value)

	self.tmp_rate, self.rate_ok = pack.Message.Fields["rate"].(float64)
	if self.rate_ok == false {
		log.Printf("Error parsing key for statsd from msg.Fields[\"rate\"]")
		return
	}

	self.rate = float32(self.tmp_rate)

	// Set all the statsdMsg attributes and fire them down the data
	// channel for the writer
	self.statsdMsg.msgType = pack.Message.Fields["type"].(string)
	self.statsdMsg.key = self.key
	self.statsdMsg.value = self.value
	self.statsdMsg.rate = self.rate

	self.dataChan <- self.statsdMsg
}
