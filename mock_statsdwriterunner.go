package heka_mozsvc_plugins

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
