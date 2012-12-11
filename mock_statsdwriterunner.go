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

/*
This file contains a mock for a WriteRunner that returns StatsdMsg
instances from RetrieveDataObject.  We only implement the methods for
WriteRunner and delegate down to the underlying MockWriteRunner.
*/

import (
	gomock "code.google.com/p/gomock/gomock"
)

type MockStatsdWriteRunner struct {
	mock *MockWriteRunner
}

func NewMockStatsdWriteRunner(ctrl *gomock.Controller) *MockStatsdWriteRunner {
	mock := NewMockWriteRunner(ctrl)
	result := &MockStatsdWriteRunner{mock: mock}
	return result
}

func (self *MockStatsdWriteRunner) EXPECT() *_MockWriteRunnerRecorder {
	return self.mock.recorder
}

func (self *MockStatsdWriteRunner) RetrieveDataObject() interface{} {
	self.mock.RetrieveDataObject()
	return new(StatsdMsg)
}

func (self *MockStatsdWriteRunner) SendOutputData(_param0 interface{}) {
	self.mock.ctrl.Call(self.mock, "SendOutputData", _param0)
}
