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
#   Victor Ng (vng@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

/*

Usage:
    Include a snippet like this into your hekad/plugin_loader.go file

	pipeline.AvailablePlugins["UdpOutput"] = func() interface{} {
		return new(mozsvc.UdpOutput)
	}

*/

import (
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"net"
)

// This will be our pipeline.PluginGlobal type
type UdpOutputGlobal struct {
	conn net.Conn
}

// Provides pipeline.PluginGlobal interface
func (self *UdpOutputGlobal) Event(eventType string) {
	if eventType == pipeline.STOP {
		self.conn.Close()
	}
}

// This will be our PluginWithGlobal type
type UdpOutput struct {
	global *UdpOutputGlobal
}

// This is our plugin's custom config struct
type UdpOutputConfig struct {
	Address string
}

// Provides pipeline.HasConfigStruct interface, populates default value
func (self *UdpOutput) ConfigStruct() interface{} {
	return &UdpOutputConfig{"my.example.com:44444"}
}

// Initialize UDP connection, store it on the PluginGlobal
func (self *UdpOutput) InitOnce(config interface{}) (pipeline.PluginGlobal, error) {
	conf := config.(*UdpOutputConfig)
	udpAddr, err := net.ResolveUDPAddr("udp", conf.Address)
	if err != nil {
		return nil, fmt.Errorf("UdpOutput error resolving UDP address %s: %s",
			conf.Address, err.Error())
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("UdpOutput error dialing UDP address %s: %s",
			conf.Address, err.Error())
	}
	return &UdpOutputGlobal{udpConn}, nil
}

// Store a reference to the global for use during pipeline processing
func (self *UdpOutput) Init(global pipeline.PluginGlobal, config interface{}) error {
	self.global = global.(*UdpOutputGlobal) // UDP connection available as self.global.conn
	return nil
}

func (self *UdpOutput) Deliver(pack *pipeline.PipelinePack) {
	// TODO: You will need to implement your own channel/goroutine
	// code to write bytes out into self.global.conn here
	// Directly accessing the self.global.conn UDP connection will
	// *not* be threadsafe.
	//
	// An easier way to do this is to use a Runner plugin, in place of
	// the PluginWithGlobal.
}
