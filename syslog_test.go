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
	"bufio"
	"errors"
	"fmt"
	gs "github.com/rafrombrc/gospec/src/gospec"
	"io/ioutil"
	"log"
	"log/syslog"
	"net"
	"os"
	"time"
)

var crashy = false

func runPktSyslog(c net.PacketConn, done chan<- string) {
	var buf [4096]byte
	var rcvd string
	ct := 0
	for {
		var n int
		var err error

		c.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, _, err = c.ReadFrom(buf[:])
		rcvd += string(buf[:n])
		if err != nil {
			if oe, ok := err.(*net.OpError); ok {
				if ct < 3 && oe.Temporary() {
					ct++
					continue
				}
			}
			break
		}
	}
	c.Close()
	done <- rcvd
}

func runStreamSyslog(l net.Listener, done chan<- string) {
	for {
		var c net.Conn
		var err error
		if c, err = l.Accept(); err != nil {
			fmt.Print(err)
			return
		}
		go func(c net.Conn) {
			b := bufio.NewReader(c)
			for ct := 1; !crashy || ct&7 != 0; ct++ {
				s, err := b.ReadString('\n')
				if err != nil {
					break
				}
				done <- s
			}
			c.Close()
		}(c)
	}
}

func startServer(n, la string, done chan<- string) (addr string) {
	if n == "udp" || n == "tcp" {
		la = "127.0.0.1:0"
	} else {
		// unix and unixgram: choose an address if none given
		if la == "" {
			// use ioutil.TempFile to get a name that is unique
			f, err := ioutil.TempFile("", "syslogtest")
			if err != nil {
				log.Fatal("TempFile: ", err)
			}
			f.Close()
			la = f.Name()
		}
		os.Remove(la)
	}

	if n == "udp" || n == "unixgram" {
		l, e := net.ListenPacket(n, la)
		if e != nil {
			log.Fatalf("startServer failed: %v", e)
		}
		addr = l.LocalAddr().String()
		go runPktSyslog(l, done)
	} else {
		l, e := net.Listen(n, la)
		if e != nil {
			log.Fatalf("startServer failed: %v", e)
		}
		addr = l.Addr().String()
		go runStreamSyslog(l, done)
	}
	return
}

func SyslogWriterSpec(c gs.Context) {

	prefix := "syslog_test"
	c.Specify("test simulated syslogd", func() {

		msg := "Test 123"
		transport := []string{"unix", "unixgram", "udp", "tcp"}

		for _, tr := range transport {
			done := make(chan string)
			addr := startServer(tr, "", done)
			logwriter, err := SyslogDial(tr, addr)
			c.Expect(err, gs.Equals, nil)
			_, err = logwriter.WriteString(syslog.LOG_INFO|syslog.LOG_USER, prefix, msg)
			c.Expect(err, gs.Equals, nil)
			err = check(c, msg, <-done)
			c.Expect(err, gs.Equals, nil)
			logwriter.Close()
		}
	})

	c.Specify("TestFlap", func() {
		net := "unix"
		done := make(chan string)
		addr := startServer(net, "", done)

		logwriter, err := SyslogDial(net, addr)

		if err != nil {
			log.Fatalf("SyslogDial() failed: %v", err)
		}
		msg := "Moo 2"

		_, err = logwriter.WriteString(syslog.LOG_INFO|syslog.LOG_USER, prefix, msg)
		if err != nil {
			log.Fatalf("log failed: %v", err)
		}
		check(c, msg, <-done)

		// restart the server
		startServer(net, addr, done)

		// and try retransmitting
		msg = "Moo 3"
		_, err = logwriter.WriteString(syslog.LOG_INFO|syslog.LOG_USER, prefix, msg)
		if err != nil {
			log.Fatalf("log failed: %v", err)
		}
		check(c, msg, <-done)

		logwriter.Close()
	})

	c.Specify("TestNew", func() {
		// Not ported as it's not relevant for us
	})

	c.Specify("TestDial", func() {
		f, err := SyslogDial("", "")

		_, err = f.WriteString(syslog.LOG_LOCAL7|syslog.LOG_DEBUG+1, prefix, "")
		if err == nil {
			log.Fatalf("Should have trapped bad priority that is too high")
		}

		_, err = f.WriteString(-1, prefix, "")
		if err == nil {
			log.Fatalf("Should have trapped bad priority that is too low")
		}

		f, err = SyslogDial("", "")
		_, err = f.WriteString(syslog.LOG_USER|syslog.LOG_ERR, prefix, "")
		if err != nil {
			log.Fatalf("Syslog WriteString() failed: %s", err)
		}
		f.Close()
	})

}

func check(c gs.Context, in, out string) (err error) {
	tmpl := fmt.Sprintf("<%d>%%s %%s syslog_test[%%d]: %s\n", syslog.LOG_USER+syslog.LOG_INFO, in)
	if _, err := os.Hostname(); err != nil {
		return errors.New("Error retrieving hostname")
	} else {
		var parsedHostname, timestamp string
		var pid int
		n, err := fmt.Sscanf(out, tmpl, &timestamp, &parsedHostname, &pid)

		// The stdlib tests that hostname matches parsedHostname, we
		// don't bother
		if err != nil || n != 3 {
			return errors.New("Error extracting timestamp, parsedHostname, pid")
		}
		computed_in := fmt.Sprintf(tmpl, timestamp, parsedHostname, pid)
		c.Expect(computed_in, gs.Equals, out)
	}

	return nil
}
