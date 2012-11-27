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
	"github.com/rafrombrc/go-notify"
	"heka/pipeline"
	"log"
	"log/syslog"
	"runtime"
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

var SyslogSenders = make(map[string]*SyslogSender)

type SyslogSender struct {
	DataChan     chan *SyslogMsg
	syslogWriter *SyslogWriter
}

func NewSyslogSender(network, raddr string) (*SyslogSender, error) {
	syslogWriter, err := SyslogDial(network, raddr)
	if err != nil {
		return nil, err
	}
	dataChan := make(chan *SyslogMsg, 1000)
	self := &SyslogSender{dataChan, syslogWriter}
	go self.sendLoop()
	return self, nil
}

func (self *SyslogSender) sendLoop() {
	stopChan := make(chan interface{})
	notify.Stop(pipeline.STOP, stopChan)
	var syslogMsg *SyslogMsg
	var err error
sendLoop:
	for {
		// Yielding before a channel select improves scheduler performance
		runtime.Gosched()
		select {
		case syslogMsg = <-self.DataChan:
			_, err = self.syslogWriter.WriteString(syslogMsg.priority,
				syslogMsg.prefix, syslogMsg.payload)
			if err != nil {
				log.Printf("Error sending to syslog: %s", err.Error())
			}
		case <-stopChan:
			break sendLoop
		}
	}
}

type SyslogMsg struct {
	priority syslog.Priority
	prefix   string
	payload  string
}

// CefOutput uses syslog to send CEF messages to an external ArcSight server
type CefOutput struct {
	sender           *SyslogSender
	senderUrl        string
	cefMetaInterface interface{}
	cefMetaMap       map[string]string
	syslogMsg        *SyslogMsg
	dataChan         chan *SyslogMsg
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
	// Using a map to guarantee there's only one SyslogSender is only safe b/c
	// the PipelinePacks (and therefore the FileOutputs) are initialized in
	// series. If this ever changes such that outputs might be created in
	// different threads then this will require a lock to make sure we don't
	// end up w/ multiple syslog connections to the same endpoint.
	self.senderUrl = fmt.Sprintf("%s:%s", conf.Network, conf.Raddr)
	var ok bool
	self.sender, ok = SyslogSenders[self.senderUrl]
	if !ok {
		self.sender, err = NewSyslogSender(conf.Network, conf.Raddr)
		if err != nil {
			return
		}
		SyslogSenders[self.senderUrl] = self.sender
	}

	self.cefMetaMap = make(map[string]string)
	self.syslogMsg = new(SyslogMsg)
	return
}

func (self *CefOutput) Deliver(pack *pipeline.PipelinePack) {
	// For b/w compatibility reasons the priority info is stored as a string
	// and we have to look it up in the SYSLOG_PRIORITY map. In the future
	// we should be storing the priority integer value directly to avoid the
	// need for the lookup.
	var ok bool
	self.cefMetaInterface, ok = pack.Message.Fields["cef_meta"]
	if !ok {
		log.Println("Can't output CEF message, missing CEF metadata.")
		return
	}
	self.cefMetaMap, ok = self.cefMetaInterface.(map[string]string)
	if !ok {
		log.Println("Can't output CEF message, CEF metadata of wrong type.")
	}
	self.syslogMsg.priority, ok = SYSLOG_PRIORITY[self.cefMetaMap["syslog_priority"]]
	if !ok {
		self.syslogMsg.priority = syslog.LOG_INFO
	}
	self.syslogMsg.prefix = self.cefMetaMap["syslog_ident"]
	self.syslogMsg.payload = pack.Message.Payload
	self.dataChan <- self.syslogMsg
}
