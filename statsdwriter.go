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
	"github.com/crankycoder/g2s"
	"log"
)

// Interface that all statsd clients must implement.
type StatsdClient interface {
	IncrementSampledCounter(bucket string, n int, srate float32)
	SendSampledTiming(bucket string, ms int, srate float32)
}

type StatsdWriter struct {
	statsdClient StatsdClient
}

func NewStatsdClient(url string) (StatsdClient, error) {
	sd, err := g2s.NewStatsd(url, 0)
	if err != nil {
		return nil, err
	}
	return sd, nil
}

func (self *StatsdWriter) Write(msg *StatsdMsg) {
	switch msg.msgType {
	case "counter":
		self.statsdClient.IncrementSampledCounter(msg.key, msg.value,
			msg.rate)
	case "timer":
		self.statsdClient.SendSampledTiming(msg.key, msg.value, msg.rate)
	default:
		log.Printf("Warning: Unexpected event passed into StatsdWriter.\nEvent => %+v\n", msg)
	}
}

func (self *StatsdWriter) SendSampledTiming(bucket string, ms int, srate float32) {
	self.statsdClient.SendSampledTiming(bucket, ms, srate)
}

func StatsdDial(url string) (w *StatsdWriter, err error) {
	var client StatsdClient
	client, err = NewStatsdClient(url)
	return &StatsdWriter{client}, err
}
