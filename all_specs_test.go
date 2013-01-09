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
	. "github.com/mozilla-services/heka/pipeline"
	"github.com/rafrombrc/gospec/src/gospec"
	"testing"
)

func mockDecoderCreator() map[string]Decoder {
	return make(map[string]Decoder)
}

func mockFilterCreator() map[string]Filter {
	return make(map[string]Filter)
}

func mockOutputCreator() map[string]Output {
	return make(map[string]Output)
}

var config = PipelineConfig{DefaultDecoder: "TEST", DefaultFilterChain: "TEST"}

func TestAllSpecs(t *testing.T) {
	r := gospec.NewRunner()
	r.Parallel = false

	r.AddSpec(StatsdOutWriterSpec)

	gospec.MainGoTest(r, t)
}

func getTestPipelinePack() *PipelinePack {
	return NewPipelinePack(&config)
}
