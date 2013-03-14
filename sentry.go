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
	"github.com/mozilla-services/heka/pipeline"
	"log"
	"net"
	"net/url"
	"sync"
	"time"
)

const (
	auth_header_tmpl   = "Sentry sentry_timestamp=%s, sentry_client=%s, sentry_version=%0.1f, sentry_key=%s"
	raven_client_id    = "raven-go/1.0"
	raven_protocol_rev = 2.0
)

type SentryMsg struct {
	encodedPayload string
	parsedDsn      *url.URL
	dataPacket     []byte
}

type SentryOutput struct {
	config *SentryOutputConfig
	udpMap map[string]net.Conn
}

type SentryOutputConfig struct {
	MaxSentryBytes int
	MaxUdpSockets  int
}

func (so *SentryOutput) ConfigStruct() interface{} {
	return &SentryOutputConfig{MaxSentryBytes: 64000,
		MaxUdpSockets: 20}
}

func (so *SentryOutput) Init(config interface{}) error {
	so.config = config.(*SentryOutputConfig)
	so.udpMap = make(map[string]net.Conn)
	return nil
}

func (so *SentryOutput) prepSentryMsg(pack *pipeline.PipelinePack,
	sentryMsg *SentryMsg) (err error) {

	var ok bool
	var tmp interface{}
	var epoch_ts64 float64
	var epoch_time time.Time
	var auth_header string
	var dsn string
	var str_ts string

	sentryMsg.encodedPayload = pack.Message.GetPayload()
	if tmp, ok = pack.Message.GetFieldValue("epoch_timestamp"); !ok {
		return fmt.Errorf("no `epoch_timestamp` field")
	}
	if epoch_ts64, ok = tmp.(float64); !ok {
		return fmt.Errorf("`epoch_timestamp` isn't a float64")
	}
	unix_nano := int64(epoch_ts64 * 1e9)
	epoch_time = time.Unix(unix_nano/1e9, unix_nano%1e9)
	str_ts = epoch_time.Format(time.RFC3339Nano)

	if tmp, ok = pack.Message.GetFieldValue("dsn"); !ok {
		return fmt.Errorf("no `dsn` field")
	}
	if dsn, ok = tmp.(string); !ok {
		return fmt.Errorf("`dsn` isn't a string")
	}

	if sentryMsg.parsedDsn, err = url.Parse(dsn); err != nil {
		return fmt.Errorf("can't parse DSN from sentry message: %s", err)
	}

	auth_header = fmt.Sprintf(auth_header_tmpl, str_ts, raven_client_id,
		raven_protocol_rev, sentryMsg.parsedDsn.User.Username())
	sentryMsg.dataPacket = []byte(fmt.Sprintf("%s\n\n%s", auth_header,
		sentryMsg.encodedPayload))
	return
}

func (so *SentryOutput) Start(or pipeline.OutputRunner, h pipeline.PluginHelper,
	wg *sync.WaitGroup) (err error) {

	go func() {
		var udpAddrStr string
		var udpAddr *net.UDPAddr
		var socket net.Conn
		var err error
		var ok bool
		sentryMsg := &SentryMsg{
			dataPacket: make([]byte, 0, so.config.MaxSentryBytes),
		}

		for pack := range or.InChan() {
			err = so.prepSentryMsg(pack, sentryMsg)
			pack.Recycle()
			if err != nil {
				or.LogError(err)
				continue
			}

			udpAddrStr = sentryMsg.parsedDsn.Host
			if socket, ok = so.udpMap[udpAddrStr]; !ok {
				if len(so.udpMap) > so.config.MaxUdpSockets {
					or.LogError(fmt.Errorf("Max # of UDP sockets [%d] reached.",
						so.config.MaxUdpSockets))
					continue
				}

				if udpAddr, err = net.ResolveUDPAddr("udp", udpAddrStr); err != nil {
					or.LogError(fmt.Errorf("can't resolve UDP address %s: %s",
						udpAddrStr, err))
					continue
				}

				if socket, err = net.DialUDP("udp", nil, udpAddr); err != nil {
					or.LogError(fmt.Errorf("can't dial UDP socket: %s", err))
					continue
				}
				so.udpMap[sentryMsg.parsedDsn.Host] = socket
			}
			socket.Write(sentryMsg.dataPacket)
		}

		log.Printf("SentryOutput '%s' stopped.", or.Name())
		wg.Done()
	}()

	return
}

func init() {
	pipeline.RegisterPlugin("SentryOutput", func() interface{} {
		return new(SentryOutput)
	})
}
