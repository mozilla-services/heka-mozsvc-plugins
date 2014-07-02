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
#   Rob Miller (rmiller@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/mozilla-services/heka/pipeline"
)

type SentryMsg struct {
	encodedPayload string
	dsn            string
}

type SentryOutput struct {
	config    *SentryOutputConfig
	clientMap map[string]*raven.Client
}

type SentryOutputConfig struct {
	MaxSentryBytes int    `toml:"max_sentry_bytes"`
	Matcher        string `toml:"message_matcher"`
}

func (so *SentryOutput) ConfigStruct() interface{} {
	return &SentryOutputConfig{
		MaxSentryBytes: 64000,
		Matcher:        "Type == 'sentry'",
	}
}

func (so *SentryOutput) Init(config interface{}) error {
	so.config = config.(*SentryOutputConfig)
	so.clientMap = make(map[string]*raven.Client)
	return nil
}

func (so *SentryOutput) prepSentryMsg(pack *pipeline.PipelinePack,
	sentryMsg *SentryMsg) (err error) {

	var (
		ok  bool
		tmp interface{}
	)

	sentryMsg.encodedPayload = pack.Message.GetPayload()

	if tmp, ok = pack.Message.GetFieldValue("dsn"); !ok {
		return fmt.Errorf("no `dsn` field")
	}
	if sentryMsg.dsn, ok = tmp.(string); !ok {
		return fmt.Errorf("`dsn` isn't a string")
	}

	return
}

func (so *SentryOutput) getClient(dsn string) (client *raven.Client, err error) {
	var (
		ok bool
	)
	if client, ok = so.clientMap[dsn]; !ok {
		client, err = raven.NewClient(dsn, nil)
	}
	return
}

func (so *SentryOutput) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {
	var (
		e      error
		pack   *pipeline.PipelinePack
		client *raven.Client
	)

	sentryMsg := &SentryMsg{}

	for pack = range or.InChan() {
		e = so.prepSentryMsg(pack, sentryMsg)
		pack.Recycle()
		if e != nil {
			or.LogError(e)
			continue
		}

		if client, err = so.getClient(sentryMsg.dsn); err != nil {
			or.LogError(e)
			continue
		}
		client.CaptureMessage(sentryMsg.encodedPayload, nil)
	}
	return
}

func init() {
	pipeline.RegisterPlugin("SentryOutput", func() interface{} {
		return new(SentryOutput)
	})
}
