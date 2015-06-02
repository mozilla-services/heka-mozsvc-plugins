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
#   Rob Miller (rmiller@mozilla.com)
#   Victor Ng (vng@mozilla.com)
#
# ***** END LICENSE BLOCK *****/
package heka_mozsvc_plugins

import (
	"log/syslog"

	"github.com/mozilla-services/heka/pipeline"
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

type CefOutput struct {
	syslogWriter *SyslogWriter
	syslogMsg    *SyslogMsg
}

type CefOutputConfig struct {
	Network string `toml:"network"`
	Raddr   string `toml:"raddr"`
}

func (cef *CefOutput) ConfigStruct() interface{} {
	return new(CefOutputConfig)
}

func (cef *CefOutput) Init(config interface{}) (err error) {
	conf := config.(*CefOutputConfig)
	cef.syslogWriter, err = SyslogDial(conf.Network, conf.Raddr)
	return
}

func (cef *CefOutput) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {

	var (
		facility, priority syslog.Priority
		ident              string
		ok                 bool
		p                  syslog.Priority
		e                  error
		pack               *pipeline.PipelinePack
	)
	syslogMsg := new(SyslogMsg)
	for pack = range or.InChan() {

		// default values
		facility, priority = syslog.LOG_LOCAL4, syslog.LOG_INFO
		ident = "heka_no_ident"

		priField := pack.Message.FindFirstField("cef_meta.syslog_priority")
		if priField != nil {
			priStr := priField.ValueString[0]
			if p, ok = SYSLOG_PRIORITY[priStr]; ok {
				priority = p
			}
		}

		facField := pack.Message.FindFirstField("cef_meta.syslog_facility")
		if facField != nil {
			facStr := facField.ValueString[0]
			if p, ok = SYSLOG_FACILITY[facStr]; ok {
				facility = p
			}
		}

		idField := pack.Message.FindFirstField("cef_meta.syslog_ident")
		if idField != nil {
			ident = idField.ValueString[0]
		}

		syslogMsg.priority = priority | facility
		syslogMsg.prefix = ident
		syslogMsg.payload = pack.Message.GetPayload()

		_, e = cef.syslogWriter.WriteString(syslogMsg.priority, syslogMsg.prefix,
			syslogMsg.payload)

		if e != nil {
			e = pipeline.NewRetryMessageError("can't write to syslog: %s", e.Error())
			pack.Recycle(e)
		} else {
			or.UpdateCursor(pack.QueueCursor)
			pack.Recycle(nil)
		}
	}

	cef.syslogWriter.Close()
	return
}

func init() {
	pipeline.RegisterPlugin("CefOutput", func() interface{} {
		return new(CefOutput)
	})
}
