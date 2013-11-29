Configuring the Plugins
=======================

Cloudwatch Input
----------------

The Cloudwatch input requests data from AWS Cloudwatch on a regular
interval and parses each returned datapoint into a heka message.

Options (required unless noted otherwise):

secret_key:
    AWS Secret Key to use.

access_key:
    AWS Access Key to use.

region:
    AWS region to poll. ie. us-west-1, eu-west-1, etc.

namespace:
    AWS Cloudwatch Namespace. ie. AWS/Billing, AWS/DynamoDB...

dimensions:
    Map of the dimension key/values to query. These are arbitrary
    key/value pairs that map to the desired dimensions. Optional.

metric_name:
    Name of the metric to query.

unit:
    Unit to query. Must be a valid AWS Cloudwatch Unit. Optional.

period:
    Period for data points, must be a factor of 60. Defaults to 60.

poll_interval:
    How often to poll AWS Cloudwatch. The first poll will not be done
    until this period. Value should be a string duration, ie. "30s" to
    indicate 30 seconds or "400ms" to indicate 400 milliseconds.

Statistics:
    What statistic values to retrieve for the metrics, valid values are
    Average, Sum, SampleCount, Maximum, Minimum.

Example snippet to retrieve estimated charges for AWS Billing:

.. code-block:: ini

    [cloudwatch_billing]
    type = "CloudwatchInput"
    secret_key = "super secret secret key here"
    access_key = "super secret access key here"
    region = "us-west-1"
    namespace = "AWS/Billing"
    metric_name = "EstimatedCharges"
    poll_interval = "30s"
    statistics = ["Sum", "Average"]

    [cloudwatch_billing.dimensions]
    ServiceName = "Amazon DynamoDB"

.. seealso:: `AWS Cloudwatch Metrics, Namespaces, and Dimensions <http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/CW_Support_For_AWS.html>`_


Cloudwatch Output
-----------------

The Cloudwatch Output takes specially crafted messages that contain a JSON
string payload and submits it to AWS Cloudwatch.

Options (required unless noted otherwise):

secret_key:
    AWS Secret Key to use.

access_key:
    AWS Access Key to use.

region:
    AWS region to poll. ie. us-west-1, eu-west-1, etc.

namespace:
    AWS Cloudwatch Namespace. ie. AWS/Billing, AWS/DynamoDB...

retries:
    How many times to retry sending the payload to AWS Cloudwatch before
    giving up. Defaults to 3, each retry will delay with an exponential
    backoff period.

backlog:
    How many messages to buffer sending at once, this is used to help
    prevent delays sending a message causing heka to block. Defaults
    to 10.

timestamp_location:
    The time zone in which timestamps in the JSON payload are presumed to
    be in. Should be a location name ("America/Los_Angeles"), as parsed
    by Go.

    .. seealso:: `Go LoadLocation strings <http://golang.org/pkg/time/#LoadLocation>`_

The payload that is parsed must be a JSON structure that looks like:

.. code-block:: json

    {
        "Datapoints":
            [
                {
                    "MetricName":"Testval",
                    "Timestamp":"Fri Jul 12 12:59:52 2013",
                    "Value":7.82636926e-06,
                    "Unit":"Kilobytes"
                }
            ]
    }

Each datapoint must meet the AWS Cloudwatch requirements for a
`MetricDatum <http://docs.aws.amazon.com/AmazonCloudWatch/latest/APIReference/API_MetricDatum.html>`_
 object. The time stamp can be a string and must be formatted as one of
 the time layouts known by Go (`Go time layouts <http://golang.org/pkg/time/#pkg-constants>`_).

The recommended way to generate the message is by using a Lua sandbox filter
that can emit Lua tables (which are serialized to JSON). This example uses
the Lua sandbox ticker feature to emit a message every ticker interval that
has a random value to send to AWS Cloudwatch:

.. code-block:: lua

    function timer_event(ns)
        local statmap = {_name="Datapoints"}
        local Datum = {}
        Datum["MetricName"] = "Testval"
        Datum["Timestamp"] = os.date()
        Datum["Unit"] = "Kilobytes"
        Datum["Value"] = math.random()
        table.insert(statmap, Datum)
        output(statmap)
        inject_message()
    end

An example configuration that sets up a Lua script like this and the
CloudwatchOutput to send it:

.. code-block:: ini

    [data_maker]
    type = "SandboxFilter"
    ticker_interval = 5
    script_type = "lua"
    filename = "inject_cloudwatch.lua"
    preserve_data = true
    memory_limit = 32767
    instruction_limit = 100
    output_limit = 256
    message_matcher = "FALSE"

    [CloudwatchOutput]
    secret_key = "super secret secret key here"
    access_key = "super secret access key here"
    region = "us-east-1"
    namespace = "Testing"
    message_matcher = "Logger == 'data_maker'"


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

Example Snippet to use a domain socket to syslog:

.. code-block:: ini

    [CefOutput]
    Network = ""
    Raddar = ""

Example Snippet to write to syslog over UDP :

.. code-block:: ini

    [CefOutput]
    Network = "UDP"
    Raddr = "syslogd1.host.com:9000"


Statsd Output
-------------

The Statsd output has a single configurable option.

Url:
    The host:port for the statsd server.. Note that you will not need
    to set the network scheme.  Just a host and port number separated
    by a colon is expected.

    Default value is "localhost:5555"

Example Snippet :

.. code-block:: ini

    [StatsdOutput]
    Url = "statsd1.host.com:8090"


Sentry Output
-------------

The Sentry output has 2 optional configuration parameters:

max_udp_sockets:
    Specifies the maximum number of open UDP sockets that heka will
    open.  This effectively limits the maximum number of Sentry
    servers that heka can communicate with as each UDP socket is in a one-to-one
    relationship with a single Sentry server.

    Default value is 20.

max_sentry_bytes:
    This specifies the size (in bytes) of the byte buffer that will
    hold base64 encoded sentry messages. The buffer is set on the
    the recycled outData.

    Default value is 64000.

Example snippet:

.. code-block:: ini

    [SentryOutput]
    max_udp_sockets = 100
    max_sentry_bytes = 100000
