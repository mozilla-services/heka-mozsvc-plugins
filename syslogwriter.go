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
	"fmt"
	"log/syslog"
	"net"
)

type SyslogWriter struct {
	conn syslogServerConn
}

type syslogServerConn interface {
	writeString(p syslog.Priority, prefix string, s string) (int, error)
	close() error
}

type syslogNetConn struct {
	conn net.Conn
}

func SyslogDial(network, raddr string) (w *SyslogWriter, err error) {
	var conn syslogServerConn
	if network == "" {
		conn, err = unixSyslog()
	} else {
		var c net.Conn
		c, err = net.Dial(network, raddr)
		conn = syslogNetConn{c}
	}
	return &SyslogWriter{conn}, err
}

func (w *SyslogWriter) WriteString(p syslog.Priority, prefix string, s string) (int, error) {
	return w.conn.writeString(p, prefix, s)
}

func (w *SyslogWriter) Close() error { return w.conn.close() }

func (n syslogNetConn) writeString(p syslog.Priority, prefix string, s string) (int, error) {
	nl := ""
	if len(s) == 0 || s[len(s)-1] != '\n' {
		nl = "\n"
	}
	_, err := fmt.Fprintf(n.conn, "<%d>%s: %s%s", p, prefix, s, nl)
	if err != nil {
		return 0, err
	}
	return len(s), nil
}

func (n syslogNetConn) close() error {
	return n.conn.Close()
}
