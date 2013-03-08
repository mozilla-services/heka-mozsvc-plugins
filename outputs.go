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
	SYSLOG_FACILITY = map[string]syslog.Priority{
		"KERN":     syslog.LOG_KERN,
		"USER":     syslog.LOG_USER,
		"MAIL":     syslog.LOG_MAIL,
		"DAEMON":   syslog.LOG_DAEMON,
		"AUTH":     syslog.LOG_AUTH,
		"LPR":      syslog.LOG_LPR,
		"NEWS":     syslog.LOG_NEWS,
		"UUCP":     syslog.LOG_UUCP,
		"CRON":     syslog.LOG_CRON,
		"AUTHPRIV": syslog.LOG_AUTHPRIV,
		"FTP":      syslog.LOG_FTP,
		"LOCAL0":   syslog.LOG_LOCAL0,
		"LOCAL1":   syslog.LOG_LOCAL1,
		"LOCAL2":   syslog.LOG_LOCAL2,
		"LOCAL3":   syslog.LOG_LOCAL3,
		"LOCAL4":   syslog.LOG_LOCAL4,
		"LOCAL5":   syslog.LOG_LOCAL5,
		"LOCAL6":   syslog.LOG_LOCAL6,
		"LOCAL7":   syslog.LOG_LOCAL7,
	}
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
	syslogMsg.priority = syslog.LOG_INFO | syslog.LOG_LOCAL4
}

func (self *CefWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{}, timeout *time.Duration) error {
	// For b/w compatibility reasons the priority info is stored as a string
	// and we have to look it up in the SYSLOG_PRIORITY map. In the future
	// we should be storing the priority integer value directly to avoid the
	// need for the lookup.
	var facility, priority syslog.Priority

	syslogMsg := outData.(*SyslogMsg)
	cefMetaInterface, ok := pack.Message.Fields["cef_meta"]
	if !ok {
		log.Println("Can't output CEF message, missing CEF metadata.")
		return fmt.Errorf("Error parsing epoch_timestamp")
	}
	cefMetaMap, ok := cefMetaInterface.(map[string]interface{})

	if !ok {
		log.Println("Can't output CEF message, CEF metadata of wrong type.")
	}

	syslogMsg.prefix, _ = cefMetaMap["syslog_ident"].(string)
	syslogMsg.payload = pack.Message.Payload

	priorityStr, _ := cefMetaMap["syslog_priority"].(string)
	priority, ok = SYSLOG_PRIORITY[priorityStr]
	if !ok {
		syslogMsg.priority = syslog.LOG_INFO | syslog.LOG_LOCAL4
		return nil
	}

	facilityStr, _ := cefMetaMap["syslog_facility"].(string)
	facility, ok = SYSLOG_FACILITY[facilityStr]
	if !ok {
		syslogMsg.priority = syslog.LOG_INFO | syslog.LOG_LOCAL4
		return nil
	}
	syslogMsg.priority = priority | facility

	return nil
}

func (self *CefWriter) Write(outData interface{}) (err error) {
	self.syslogMsg = outData.(*SyslogMsg)
	_, err = self.syslogWriter.WriteString(
                        self.syslogMsg.priority,
                        self.syslogMsg.prefix,
                        self.syslogMsg.payload)
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

func init() {
	pipeline.RegisterPlugin("CefOutput", func() interface{} {
		return pipeline.RunnerMaker(new(CefWriter))
	})
}
