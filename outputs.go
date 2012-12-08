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
#   Rob Miller (rmiller@mozilla.com)
#
# ***** END LICENSE BLOCK *****/
package heka_mozsvc_plugins

import (
	"fmt"
	"heka/pipeline"
	"log"
	"log/syslog"
)

var (
	SYSLOG_PRIORITY = map[string]syslog.Priority{
		"EMERG":   syslog.LOG_EMERG,
		"ALERT":   syslog.LOG_ALERT,
		"CRIT":    syslog.LOG_CRIT,
		"ERR":     syslog.LOG_ERR,
		"WARNING": syslog.LOG_WARNING,
		"NOTICE":  syslog.LOG_NOTICE,
		"INFO":    syslog.LOG_INFO,
		"DEBUG":   syslog.LOG_DEBUG,
	}
)

type SyslogMsg struct {
	priority syslog.Priority
	prefix   string
	payload  string
}

var SyslogWriteRunners = make(map[string]pipeline.WriteRunner)

type SyslogOutputWriter struct {
	syslogWriter *SyslogWriter
	syslogMsg    *SyslogMsg
	err          error
}

func NewSyslogOutputWriter(network, raddr string) (*SyslogOutputWriter,
	error) {
	syslogWriter, err := SyslogDial(network, raddr)
	if err != nil {
		return nil, err
	}
	self := &SyslogOutputWriter{syslogWriter: syslogWriter}
	return self, nil
}

func (self *SyslogOutputWriter) MakeOutputData() interface{} {
	return new(SyslogMsg)
}

func (self *SyslogOutputWriter) Write(outputData interface{}) error {
	self.syslogMsg = outputData.(*SyslogMsg)
	_, self.err = self.syslogWriter.WriteString(self.syslogMsg.priority,
		self.syslogMsg.prefix, self.syslogMsg.payload)
	self.syslogMsg.priority = syslog.LOG_INFO
	if self.err != nil {
		return fmt.Errorf("SyslogOutputWriter error: %s", self.err)
	}
	return nil
}

func (self *SyslogOutputWriter) Stop() {
	self.syslogWriter.Close()
}

// CefOutput uses syslog to send CEF messages to an external ArcSight server
type CefOutput struct {
	writeRunner      pipeline.WriteRunner
	cefMetaInterface interface{}
	cefMetaMap       map[string]interface{}
	syslogMsg        *SyslogMsg
	tempStr          string
	ok               bool
}

type CefOutputConfig struct {
	Network string
	Raddr   string
}

func (self *CefOutput) ConfigStruct() interface{} {
	return new(CefOutputConfig)
}

func (self *CefOutput) Init(config interface{}) (err error) {
	conf := config.(*CefOutputConfig)
	// Using a map to guarantee there's only one WriteRunner is only safe b/c
	// the PipelinePacks (and therefore the CefOutputs) are initialized in
	// series. If this ever changes such that outputs might be created in
	// different threads then this will require a lock to make sure we don't
	// end up w/ multiple syslog connections to the same endpoint.
	syslogUrl := fmt.Sprintf("%s:%s", conf.Network, conf.Raddr)
	writeRunner, ok := SyslogWriteRunners[syslogUrl]
	if !ok {
		syslogOutputWriter, err := NewSyslogOutputWriter(conf.Network, conf.Raddr)
		if err != nil {
			return fmt.Errorf("Error creating SyslogOutputWriter: %s", err)
		}
		writeRunner = pipeline.NewWriteRunner(syslogOutputWriter)
		SyslogWriteRunners[syslogUrl] = writeRunner
	}
	self.writeRunner = writeRunner
	return nil
}

func (self *CefOutput) Deliver(pack *pipeline.PipelinePack) {
	// For b/w compatibility reasons the priority info is stored as a string
	// and we have to look it up in the SYSLOG_PRIORITY map. In the future
	// we should be storing the priority integer value directly to avoid the
	// need for the lookup.
	self.syslogMsg = self.writeRunner.RetrieveDataObject().(*SyslogMsg)
	self.cefMetaInterface, self.ok = pack.Message.Fields["cef_meta"]
	if !self.ok {
		log.Println("Can't output CEF message, missing CEF metadata.")
		return
	}
	self.cefMetaMap, self.ok = self.cefMetaInterface.(map[string]interface{})
	if !self.ok {
		log.Println("Can't output CEF message, CEF metadata of wrong type.")
	}
	self.tempStr, _ = self.cefMetaMap["syslog_priority"].(string)
	self.syslogMsg.priority, self.ok = SYSLOG_PRIORITY[self.tempStr]
	if !self.ok {
		self.syslogMsg.priority = syslog.LOG_INFO
	}
	self.syslogMsg.prefix, _ = self.cefMetaMap["syslog_ident"].(string)
	self.syslogMsg.payload = pack.Message.Payload
	self.writeRunner.SendOutputData(self.syslogMsg)
}
