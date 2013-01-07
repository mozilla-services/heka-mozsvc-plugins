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

import (
	"github.com/rafrombrc/gospec/src/gospec"
	"heka/pipeline"
	"testing"
)

func mockDecoderCreator() map[string]pipeline.Decoder {
	return make(map[string]pipeline.Decoder)
}

func mockFilterCreator() map[string]pipeline.Filter {
	return make(map[string]pipeline.Filter)
}

func mockOutputCreator() map[string]pipeline.Output {
	return make(map[string]pipeline.Output)
}

var config = pipeline.PipelineConfig{DefaultDecoder: "TEST", DefaultFilterChain: "TEST"}

func TestAllSpecs(t *testing.T) {
	r := gospec.NewRunner()
	r.Parallel = false

	r.AddSpec(StatsdOutWriterSpec)
	r.AddSpec(SentryOutWriterSpec)

	gospec.MainGoTest(r, t)
}

func getTestPipelinePack() *pipeline.PipelinePack {
	return pipeline.NewPipelinePack(&config)
}
