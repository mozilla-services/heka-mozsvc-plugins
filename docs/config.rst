Configuring the plugins
=======================

CEF Output
----------

The CEF output takes two options: 

Network:
    This can be blank, "TCP" or "UDP".
    If left blank, syslog will write to syslog using a unix domain
    socket. TCP and UDP will write out to the syslog daemon using a
    socket.

Raddr:
    This option is only used if TCP or UDP is specified by the Network
    option.  It specifies a host and port for a syslog daemon that the
    CEF output will write out to.

Example Snippet to use a domain socket to syslog::

        {
            "type": "CefOutput"
            "Network": "",
            "Raddr ": ""
        }

Example Snippet to write to syslog over UDP ::

        {
            "type": "CefOutput"
            "Network": "UDP",
            "Raddr ": "syslogd1.host.com:9000"
        }

Statsd Output
-------------

The Statsd output has a single configurable option.  

Url:
    The host:port for the statsd server.. Note that you will not need
    to set the network scheme.  Just a host and port number separated
    by a colon is expected.

    Default value is "localhost:5555"

Example Snippet ::

        {
            "type": "StatsdOutput",
            "Url": "statsd1.host.com:8090"
        }


Sentry Output
-------------

The Sentry output has 2 optional configuration parameters:

MaxUdpSocket:
    Specifies the maximum number of open UDP sockets that heka will 
    open.  This effectively limits the maximum number of Sentry
    servers that heka can communicate with as each UDP socket is in a one-to-one
    relationship with a single Sentry server.

    Default value is 20.

MaxSentryBytes:
    This specifies the size (in bytes) of the byte buffer that will
    hold base64 encoded sentry messages. The buffer is set on the
    the recycled outData.

    Default value is 64000.

Example snippet ::

        {
            "type": "SentryOutput",
            "MaxUdpSockets": 100,
            "MaxSentryBytes": 100000
        }

