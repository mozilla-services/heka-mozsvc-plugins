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
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"errors"
	"strings"

	"github.com/mozilla-services/heka/pipeline"
)

// A filter that expects `counter` or `timer` type messages that have come in
// via a heka client and injects them into the StatMonitor so the system
// behaves exactly as though they came in through the StatsdInput.
type HekaStatsFilter struct {
	statAccumName string
}

// HekaStatsFilter config struct.
type HekaStatsFilterConfig struct {
	// Configured name of StatAccumulator input plugin to which this filter
	// should be delivering its Stats. Defaults to "StatAccumInput".
	StatAccumName string `toml:"stat_accum_name"`
}

func (hsf *HekaStatsFilter) ConfigStruct() interface{} {
	return &HekaStatsFilterConfig{
		StatAccumName: "StatAccumInput",
	}
}

func (hsf *HekaStatsFilter) Init(config interface{}) (err error) {
	conf := config.(*HekaStatsFilterConfig)
	hsf.statAccumName = conf.StatAccumName
	return
}

func (hsf *HekaStatsFilter) Run(fr pipeline.FilterRunner, h pipeline.PluginHelper) (
	err error) {

	var (
		tmp       interface{}
		pack      *pipeline.PipelinePack
		ns, name  string
		rate      float64
		statAccum pipeline.StatAccumulator
		stat      pipeline.Stat
		ok        bool
	)

	if statAccum, err = h.StatAccumulator(hsf.statAccumName); err != nil {
		return
	}

	for pack = range fr.InChan() {
		ns = pack.Message.GetLogger()

		if tmp, ok = pack.Message.GetFieldValue("name"); !ok {
			fr.UpdateCursor(pack.QueueCursor)
			pack.Recycle(errors.New("stats message missing stat name"))
			continue
		}
		if name, ok = tmp.(string); !ok {
			fr.UpdateCursor(pack.QueueCursor)
			pack.Recycle(errors.New("stats message name is not a string"))
			continue
		}

		if strings.TrimSpace(ns) != "" {
			name = strings.Join([]string{ns, name}, ".")
		}

		if tmp, ok = pack.Message.GetFieldValue("rate"); !ok {
			fr.UpdateCursor(pack.QueueCursor)
			pack.Recycle(errors.New("stats message missing rate value"))
			continue
		}
		if rate, ok = tmp.(float64); !ok {
			fr.LogError(errors.New("stats message rate is not a float"))
		}

		stat.Bucket = name
		stat.Value = pack.Message.GetPayload()
		stat.Sampling = float32(rate)
		if pack.Message.GetType() == "timer" {
			stat.Modifier = "ms"
		} else {
			stat.Modifier = ""
		}
		statAccum.DropStat(stat)
		fr.UpdateCursor(pack.QueueCursor)
		pack.Recycle(nil)
	}

	return
}

func init() {
	pipeline.RegisterPlugin("HekaStatsFilter", func() interface{} {
		return new(HekaStatsFilter)
	})
}
