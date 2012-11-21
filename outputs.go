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

type CefMeta struct {
	priority syslog.Priority
	prefix   string
}

// CefOutput uses syslog to send CEF messages to an external ArcSight server
type CefOutput struct {
	writer           *SyslogWriter
	cefMetaInterface interface{}
	cefMetaMap       map[string]string
	cefMeta          *CefMeta
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
	self.writer, err = SyslogDial(conf.Network, conf.Raddr)
	if err != nil {
		return
	}
	self.cefMetaMap = make(map[string]string)
	self.cefMeta = new(CefMeta)
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
	self.cefMeta.priority, ok = SYSLOG_PRIORITY[self.cefMetaMap["syslog_priority"]]
	if !ok {
		self.cefMeta.priority = syslog.LOG_INFO
	}
	self.cefMeta.prefix = self.cefMetaMap["syslog_ident"]
	self.writer.WriteString(self.cefMeta.priority, self.cefMeta.prefix,
		pack.Message.Payload)
}
