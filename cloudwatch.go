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
#   Ben Bangert (bbangert@mozilla.com)
#
# ***** END LICENSE BLOCK *****/

package heka_mozsvc_plugins

import (
	"github.com/crowdmob/goamz/cloudwatch"
)

// Cloudwatch Input Config
type CloudwatchInputConfig struct {
	// AWS Region, ie. us-west-1, eu-west-1
	Region string
	// Cloudwatch Namespace, ie. AWS/Billing, AWS/DynamoDB, custom...
	Namespace string
	// List of dimensions to query
	Dimensions map[string]string
	// Metric name
	MetricName string
	// Unit
	Unit string
	// Period for data points, must be factor of 60
	Period int
	// Polling period, how often as a duration to poll for new
	// metrics. The first poll on startup will not be done until this
	// period is reached.
	PollInterval string
	// What statistic values to retrieve for the metrics, valid values
	// are Average, Sum, SampleCount, Maximum, Minumum
	Statistics []string
}

type CloudwatchInput struct {
	region    string
	namespace string
	cw        *cloudwatch.CloudWatch
}
