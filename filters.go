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
	"errors"
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"strings"
)

// A filter that expects `counter` or `timer` type messages that have come in
// via a heka client and injects them into the StatMonitor so the system
// behaves exactly as though they came in through the StatsdInput.
type HekaStatsFilter struct {
	inputName string
}

// HekaStatsFilter config struct.
type HekaStatsFilterConfig struct {
	// Configured name of StatsdInput plugin to which this filter should be
	// delivering its output. Defaults to "StatsdInput".
	StatsdInputName string
}

func (hsf *HekaStatsFilter) ConfigStruct() interface{} {
	return &HekaStatsFilterConfig{
		StatsdInputName: "StatsdInput",
	}
}

func (hsf *HekaStatsFilter) Init(config interface{}) (err error) {
	conf := config.(*HekaStatsFilterConfig)
	hsf.inputName = conf.StatsdInputName
	return
}

func (hsf *HekaStatsFilter) Run(fr pipeline.OutputRunner, h pipeline.PluginHelper) (
	err error) {

	var (
		tmp       interface{}
		pack      *pipeline.PipelinePack
		sp        pipeline.StatPacket
		ns, name  string
		rate      float64
		ir        pipeline.InputRunner
		statInput *pipeline.StatsdInput
		ok        bool
	)

	// Get the StatMonitor input channel.
	if ir, ok = h.PipelineConfig().InputRunners[hsf.inputName]; !ok {
		return fmt.Errorf("Unable to locate StatsdInput '%s', was it configured?",
			hsf.inputName)
	}
	if statInput, ok = ir.Plugin().(*pipeline.StatsdInput); !ok {
		return fmt.Errorf("Unable to coerce '%s' input plugin to StatsdInput",
			hsf.inputName)
	}
	spChan := statInput.Packet

	for plc := range fr.InChan() {
		pack = plc.Pack

		ns = pack.Message.GetLogger()

		if tmp, ok = pack.Message.GetFieldValue("name"); !ok {
			fr.LogError(errors.New("stats message missing stat name"))
			pack.Recycle()
			continue
		}
		if name, ok = tmp.(string); !ok {
			fr.LogError(errors.New("stats message name is not a string"))
			pack.Recycle()
			continue
		}

		if strings.TrimSpace(ns) != "" {
			name = strings.Join([]string{ns, name}, ".")
		}

		if tmp, ok = pack.Message.GetFieldValue("rate"); !ok {
			fr.LogError(errors.New("stats message missing rate value"))
			pack.Recycle()
			continue
		}
		if rate, ok = tmp.(float64); !ok {
			fr.LogError(errors.New("stats message rate is not a float"))
		}

		sp.Bucket = name
		sp.Value = pack.Message.GetPayload()
		sp.Sampling = float32(rate)
		if pack.Message.GetType() == "timer" {
			sp.Modifier = "ms"
		} else {
			sp.Modifier = ""
		}
		spChan <- sp
		pack.Recycle()
	}

	return
}

func init() {
	pipeline.RegisterPlugin("HekaStatsFilter", func() interface{} {
		return new(HekaStatsFilter)
	})
}
