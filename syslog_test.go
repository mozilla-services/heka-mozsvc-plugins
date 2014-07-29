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
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
	"net"
	"os"
	"sync"
	"time"
)

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

func runStreamSyslog(l net.Listener, done chan<- string, wg *sync.WaitGroup, crashy bool) {
	for {
		var c net.Conn
		var err error
		if c, err = l.Accept(); err != nil {
			return
		}
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			c.SetReadDeadline(time.Now().Add(5 * time.Second))
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

func startServer(n, la string, done chan<- string, crashy bool) (addr string,
	sock io.Closer, wg *sync.WaitGroup) {

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

	wg = new(sync.WaitGroup)
	if n == "udp" || n == "unixgram" {
		l, e := net.ListenPacket(n, la)
		if e != nil {
			log.Fatalf("startServer failed: %v", e)
		}
		addr = l.LocalAddr().String()
		sock = l
		wg.Add(1)
		go func() {
			defer wg.Done()
			runPktSyslog(l, done)
		}()
	} else {
		l, e := net.Listen(n, la)
		if e != nil {
			log.Fatalf("startServer failed: %v", e)
		}
		addr = l.Addr().String()
		sock = l
		wg.Add(1)
		go func() {
			defer wg.Done()
			runStreamSyslog(l, done, wg, crashy)
		}()
	}
	return
}

func SyslogWriterSpec(c gs.Context) {

	prefix := "syslog_test"
	crashy := false

	c.Specify("test simulated syslogd", func() {

		msg := "Test 123"
		transport := []string{"unix", "unixgram", "udp", "tcp"}

		for _, tr := range transport {
			done := make(chan string)
			addr, _, _ := startServer(tr, "", done, crashy)
			if tr == "unix" || tr == "unixgram" {
				defer os.Remove(addr)
			}
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
		addr, sock, _ := startServer(net, "", done, crashy)
		defer os.Remove(addr)
		defer sock.Close()

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
		_, sock2, _ := startServer(net, addr, done, crashy)
		defer sock2.Close()

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

	c.Specify("TestWrite", func() {
		tests := []struct {
			pri syslog.Priority
			pre string
			msg string
			exp string
		}{
			{syslog.LOG_USER | syslog.LOG_ERR, "syslog_test", "", "%s %s syslog_test[%d]: \n"},
			{syslog.LOG_USER | syslog.LOG_ERR, "syslog_test", "write test", "%s %s syslog_test[%d]: write test\n"},
			// Write should not add \n if there already is one
			{syslog.LOG_USER | syslog.LOG_ERR, "syslog_test", "write test 2\n", "%s %s syslog_test[%d]: write test 2\n"},
		}

		if hostname, err := os.Hostname(); err != nil {
			log.Fatalf("Error retrieving hostname")
		} else {
			for _, test := range tests {
				done := make(chan string)
				addr, sock, _ := startServer("udp", "", done, crashy)
				defer sock.Close()

				//l, err := Dial("udp", addr, test.pri, test.pre)
				l, err := SyslogDial("udp", addr)

				if err != nil {
					log.Fatalf("SyslogDial() failed: %v", err)
				}
				_, err = l.WriteString(test.pri, test.pre, test.msg)

				if err != nil {
					log.Fatalf("WriteString() failed: %v", err)
				}
				rcvd := <-done
				test.exp = fmt.Sprintf("<%d>", test.pri) + test.exp
				var parsedHostname, timestamp string
				var pid int
				if n, err := fmt.Sscanf(rcvd, test.exp, &timestamp, &parsedHostname, &pid); n != 3 || err != nil || hostname != parsedHostname {
					log.Fatalf("s.Info() = '%q', didn't match '%q' (%d %s)", rcvd, test.exp, n, err)
				}
			}
		}
	})

	c.Specify("TestConcurrentWrite", func() {
		addr, sock, _ := startServer("udp", "", make(chan string), crashy)
		defer sock.Close()

		//w, err := Dial("udp", addr, LOG_USER|LOG_ERR, "how's it going?")
		w, err := SyslogDial("udp", addr)

		if err != nil {
			log.Fatalf("syslog.Dial() failed: %v", err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				_, err := w.WriteString(syslog.LOG_USER|syslog.LOG_ERR, "how's it going?", "test")
				if err != nil {
					log.Fatalf("Info() failed: %v", err)
					return
				}
				wg.Done()
			}()
		}
		wg.Wait()
	})

	c.Specify("TestConcurrentReconnect", func() {
		crashy = true

		net := "unix"
		done := make(chan string)
		addr, sock, srvWG := startServer(net, "", done, crashy)
		defer os.Remove(addr)

		// count all the messages arriving
		count := make(chan int)
		go func() {
			ct := 0
			for _ = range done {
				ct++
				// we are looking for 500 out of 1000 events
				// here because lots of log messages are lost
				// in buffers (kernel and/or bufio)
				if ct > 500 {
					break
				}
			}
			count <- ct
		}()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				//w, err := Dial(net, addr, syslog.LOG_USER|syslog.LOG_ERR, "tag")
				w, err := SyslogDial(net, addr) //, syslog.LOG_USER|syslog.LOG_ERR, "tag")
				if err != nil {
					log.Fatalf("syslog.Dial() failed: %v", err)
				}
				for i := 0; i < 100; i++ {
					_, err := w.WriteString(syslog.LOG_USER|syslog.LOG_INFO, "tag", "test")
					if err != nil {
						log.Fatalf("Info() failed: %v", err)
						return
					}
				}
				wg.Done()
			}()
		}
		wg.Wait()
		sock.Close()
		srvWG.Wait()
		close(done)

		select {
		case <-count:
		case <-time.After(100 * time.Millisecond):
			log.Fatalf("timeout in concurrent reconnect")
		}
	})
}

func check(c gs.Context, in, out string) (err error) {
	tmpl := fmt.Sprintf("<%d>%%s %%s syslog_test[%%d]: %s\n", syslog.LOG_USER+syslog.LOG_INFO, in)
	if hostname, err := os.Hostname(); err != nil {
		return errors.New("Error retrieving hostname")
	} else {
		var parsedHostname, timestamp string
		var pid int

		// The stdlib tests that hostname matches parsedHostname, we
		// don't bother
		if n, err := fmt.Sscanf(out, tmpl, &timestamp, &parsedHostname, &pid); n != 3 || err != nil || hostname != parsedHostname {
			return errors.New("Error extracting timestamp, parsedHostname, pid")
		}
		computed_in := fmt.Sprintf(tmpl, timestamp, parsedHostname, pid)
		c.Expect(computed_in, gs.Equals, out)
	}

	return nil
}
