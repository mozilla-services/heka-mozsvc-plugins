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
		return pipeline.RunnerMaker(new(mozsvc.UDPOutputWriter))
	}
*/

import (
	"fmt"
	"github.com/mozilla-services/heka/pipeline"
	"net"
	"time"
)

// Provides pipeline.PluginGlobal interface
func (self *UdpOutputWriter) Event(eventType string) {
	if eventType == pipeline.STOP {
		self.conn.Close()
	}
}

// This will be our Writer type
type UdpOutputWriter struct {
	conn net.Conn
}

// This is our plugin's custom config struct
type UdpOutputWriterConfig struct {
	Address string
}

// Provides pipeline.HasConfigStruct interface, populates default value
func (self *UdpOutputWriter) ConfigStruct() interface{} {
	return &UdpOutputWriterConfig{"my.example.com:44444"}
}

// Initialize UDP connection, store it on the PluginGlobal
func (self *UdpOutputWriter) Init(config interface{}) error {
	conf := config.(*UdpOutputWriterConfig)
	udpAddr, err := net.ResolveUDPAddr("udp", conf.Address)
	if err != nil {
		return fmt.Errorf("UdpOutput error resolving UDP address %s: %s",
			conf.Address, err.Error())
	}
	udpConn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("UdpOutput error dialing UDP address %s: %s",
			conf.Address, err.Error())
	}
	self.conn = udpConn
	return nil
}

func (self *UdpOutputWriter) PrepOutData(pack *pipeline.PipelinePack, outData interface{}, timeout *time.Duration) error {
	outBytesPtr := outData.(*[]byte)
	*outBytesPtr = append(*outBytesPtr, []byte(pack.Message.Payload)...)
	return nil
}

func (self *UdpOutputWriter) Write(outData interface{}) (err error) {
	bytePtr := outData.(*[]byte)
	self.conn.Write(*bytePtr)
	return nil
}

// Creates a byte slice for holding output data
func (self *UdpOutputWriter) MakeOutData() interface{} {
	b := make([]byte, 0, 1000)
	return &b
}

// Resets a byte slice to zero length for reuse
func (self *UdpOutputWriter) ZeroOutData(outData interface{}) {
	outBytesPtr := outData.(*[]byte)
	*outBytesPtr = (*outBytesPtr)[:0]
}
