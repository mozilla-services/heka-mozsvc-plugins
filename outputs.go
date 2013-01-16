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
	"github.com/mozilla-services/heka/pipeline"
	"log"
	"log/syslog"
	"time"
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

type CefWriter struct {
	syslogWriter *SyslogWriter
	syslogMsg    *SyslogMsg
}

type CefWriterConfig struct {
	Network string
	Raddr   string
}

func (self *CefWriter) ConfigStruct() interface{} {
	return new(CefWriterConfig)
}

func (self *CefWriter) Init(config interface{}) (err error) {
	conf := config.(*CefWriterConfig)
	self.syslogWriter, err = SyslogDial(conf.Network, conf.Raddr)
	return
}

func (self *CefWriter) MakeOutData() interface{} {
	return new(SyslogMsg)
}

func (self *CefWriter) ZeroOutData(outData interface{}) {
	syslogMsg := outData.(*SyslogMsg)
	syslogMsg.priority = syslog.LOG_INFO
}

func (self *CefWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{}, timeout *time.Duration) error {
	// For b/w compatibility reasons the priority info is stored as a string
	// and we have to look it up in the SYSLOG_PRIORITY map. In the future
	// we should be storing the priority integer value directly to avoid the
	// need for the lookup.
	syslogMsg := outData.(*SyslogMsg)
	cefMetaInterface, ok := pack.Message.Fields["cef_meta"]
	if !ok {
		log.Println("Can't output CEF message, missing CEF metadata.")
		return CefError{time.Now(), "Error parsing epoch_timestamp"}
	}
	cefMetaMap, ok := cefMetaInterface.(map[string]interface{})
	if !ok {
		log.Println("Can't output CEF message, CEF metadata of wrong type.")
	}
	priorityStr, _ := cefMetaMap["syslog_priority"].(string)
	syslogMsg.priority, ok = SYSLOG_PRIORITY[priorityStr]
	if !ok {
		syslogMsg.priority = syslog.LOG_INFO
	}
	syslogMsg.prefix, _ = cefMetaMap["syslog_ident"].(string)
	syslogMsg.payload = pack.Message.Payload
	return nil
}

func (self *CefWriter) Write(outData interface{}) (err error) {
	self.syslogMsg = outData.(*SyslogMsg)
	_, err = self.syslogWriter.WriteString(self.syslogMsg.priority,
		self.syslogMsg.prefix, self.syslogMsg.payload)
	if err != nil {
		err = fmt.Errorf("CefWriter Write error: %s", err)
	}
	return
}

func (self *CefWriter) Event(eventType string) {
	if eventType == pipeline.STOP {
		self.syslogWriter.Close()
	}
}

type CefError struct {
	When time.Time
	What string
}

func (e CefError) Error() string {
	return fmt.Sprintf("%v: %v", e.When, e.What)
}
