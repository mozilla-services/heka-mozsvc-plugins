// Automatically generated by MockGen. DO NOT EDIT!
// Source: heka/pipeline (interfaces: WriteRunner)

package heka_mozsvc_plugins

import (
	gomock "code.google.com/p/gomock/gomock"
)

// Mock of WriteRunner interface
type MockWriteRunner struct {
	ctrl     *gomock.Controller
	recorder *_MockWriteRunnerRecorder
}

// Recorder for MockWriteRunner (not exported)
type _MockWriteRunnerRecorder struct {
	mock *MockWriteRunner
}

func NewMockWriteRunner(ctrl *gomock.Controller) *MockWriteRunner {
	mock := &MockWriteRunner{ctrl: ctrl}
	mock.recorder = &_MockWriteRunnerRecorder{mock}
	return mock
}

func (_m *MockWriteRunner) EXPECT() *_MockWriteRunnerRecorder {
	return _m.recorder
}

func (_m *MockWriteRunner) RetrieveDataObject() interface{} {
	ret := _m.ctrl.Call(_m, "RetrieveDataObject")
	ret0, _ := ret[0].(interface{})
	return ret0
}

func (_mr *_MockWriteRunnerRecorder) RetrieveDataObject() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "RetrieveDataObject")
}

func (_m *MockWriteRunner) SendOutputData(_param0 interface{}) {
	_m.ctrl.Call(_m, "SendOutputData", _param0)
}

func (_mr *_MockWriteRunnerRecorder) SendOutputData(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SendOutputData", arg0)
}
