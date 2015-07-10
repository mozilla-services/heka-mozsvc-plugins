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

	"github.com/getsentry/raven-go"
	"github.com/mozilla-services/heka/pipeline"
)

type SentryMsg struct {
	dsn string
}

type SentryOutput struct {
	config    *SentryOutputConfig
	clientMap map[string]*raven.Client
}

type SentryOutputConfig struct {
	MaxSentryBytes int    `toml:"max_sentry_bytes"`
	Matcher        string `toml:"message_matcher"`
	Dsn            string `toml:"dsn"`
}

func (so *SentryOutput) ConfigStruct() interface{} {
	return &SentryOutputConfig{
		MaxSentryBytes: 64000,
		Matcher:        "Type == 'sentry'",
		Dsn:            "",
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

	// Take dsn value from config if it is set.
	if so.config.Dsn != "" {
		sentryMsg.dsn = so.config.Dsn
		return
	}

	if tmp, ok = pack.Message.GetFieldValue("dsn"); !ok {
		return fmt.Errorf("no `dsn` field")
	}
	if sentryMsg.dsn, ok = tmp.(string); !ok {
		return fmt.Errorf("`dsn` isn't a string")
	}
	return
}

func (so *SentryOutput) getClient(dsn string) (client *raven.Client, err error) {
	var ok bool
	if client, ok = so.clientMap[dsn]; !ok {
		client, err = raven.NewClient(dsn, nil)
	}
	return
}

func (so *SentryOutput) Run(or pipeline.OutputRunner, h pipeline.PluginHelper) (err error) {
	var (
		e        error
		pack     *pipeline.PipelinePack
		client   *raven.Client
		contents []byte
	)

	if or.Encoder() == nil {
		return errors.New("Encoder required for SentryOutput")
	}
	sentryMsg := &SentryMsg{}

	for pack = range or.InChan() {
		contents, e = or.Encode(pack)
		if e != nil {
			or.UpdateCursor(pack.QueueCursor)
			pack.Recycle(fmt.Errorf("Error encoding message: %s", e))
			continue
		}

		e = so.prepSentryMsg(pack, sentryMsg)
		if e != nil {
			or.UpdateCursor(pack.QueueCursor)
			pack.Recycle(e)
			continue
		}

		if client, e = so.getClient(sentryMsg.dsn); e != nil {
			e = pipeline.NewRetryMessageError(e.Error())
			pack.Recycle(e)
			continue
		}
		client.CaptureMessage(string(contents), nil)
		pack.Recycle(nil)
	}
	return
}

func init() {
	pipeline.RegisterPlugin("SentryOutput", func() interface{} {
		return new(SentryOutput)
	})
}
