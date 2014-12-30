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
	"log/syslog"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type SyslogWriter struct {
	network  string
	raddr    string
	hostname string
	mu       sync.Mutex // guards conn

	conn syslogServerConn
}

type syslogServerConn interface {
	writeString(p syslog.Priority, timestamp int64, hostname string, prefix string, s string) (int, error)
	close() error
}

type syslogNetConn struct {
	conn net.Conn
}

func SyslogDial(network, raddr string) (w *SyslogWriter, err error) {
	var writer *SyslogWriter
	writer = &SyslogWriter{network: network, raddr: raddr}
	writer.mu.Lock()
	defer writer.mu.Unlock()

	err = writer.connect()
	if err != nil {
		return nil, err
	}
	return writer, err
}

func (w *SyslogWriter) connect() (err error) {
	if w.conn != nil {
		// ignore err from close, it makes sense to continue anyway
		w.conn.close()
		w.conn = nil
	}

	if w.network == "" {
		w.conn, err = unixSyslog()
		if w.hostname == "" {
			w.hostname = "localhost"
		}
	} else {
		var c net.Conn
		c, err = net.Dial(w.network, w.raddr)
		if err == nil {
			w.conn = &syslogNetConn{c}
			if w.hostname == "" {
				if w.hostname, err = os.Hostname(); err != nil {
					return errors.New("Error retrieving hostname")
				}
			}
		}
	}
	return
}

func (w *SyslogWriter) writeAndRetry(p syslog.Priority,
	timestamp int64,
	hostname string,
	prefix string,
	s string) (n int, err error) {

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		if n, err = w.conn.writeString(p, timestamp, hostname, prefix, s); err == nil {
			return n, err
		}
	}
	if err := w.connect(); err != nil {
		return 0, err
	}
	n, err = w.conn.writeString(p, timestamp, hostname, prefix, s)
	return n, err
}

func (w *SyslogWriter) WriteString(p syslog.Priority, timestamp int64, hostname string, prefix string, s string) (n int, err error) {
	return w.writeAndRetry(p, timestamp, hostname, prefix, s)
}

func (w *SyslogWriter) Close() (err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		err = w.conn.close()
		w.conn = nil
		return err
	}
	return nil
}

func (n syslogNetConn) writeString(p syslog.Priority, timestamp int64, hostname string, prefix string, msg string) (int, error) {
	if p < 0 || p > syslog.LOG_LOCAL7|syslog.LOG_DEBUG {
		return 0, errors.New("log/syslog: invalid priority")
	}

	// ensure it ends in a \n
	nl := ""
	if !strings.HasSuffix(msg, "\n") {
		nl = "\n"
	}

	formattedts := time.UnixNano(0,timestamp).Format(time.RFC3339)

	return fmt.Fprintf(n.conn, "<%d>%s %s %s[%d]: %s%s",
		p, formattedts, hostname,
		prefix, os.Getpid(), msg, nl)
}

func (n syslogNetConn) close() error {
	return n.conn.Close()
}

// unixSyslog opens a connection to the syslog daemon running on the
// local machine using a Unix domain socket.
func unixSyslog() (conn syslogServerConn, err error) {
	logTypes := []string{"unixgram", "unix"}
	logPaths := []string{"/dev/log", "/var/run/syslog"}
	var raddr string
	for _, network := range logTypes {
		for _, path := range logPaths {
			raddr = path
			conn, err := net.Dial(network, raddr)
			if err != nil {
				continue
			} else {
				return syslogNetConn{conn}, nil
			}
		}
	}
	return nil, errors.New("Unix syslog delivery error")
}
