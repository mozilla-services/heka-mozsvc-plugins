// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/mozilla-services/heka/pipeline (interfaces: InputRunner)

package testsupport

import (
	sync "sync"
	gomock "code.google.com/p/gomock/gomock"
	pipeline "github.com/mozilla-services/heka/pipeline"
)

// Mock of InputRunner interface
type MockInputRunner struct {
	ctrl     *gomock.Controller
	recorder *_MockInputRunnerRecorder
}

// Recorder for MockInputRunner (not exported)
type _MockInputRunnerRecorder struct {
	mock *MockInputRunner
}

func NewMockInputRunner(ctrl *gomock.Controller) *MockInputRunner {
	mock := &MockInputRunner{ctrl: ctrl}
	mock.recorder = &_MockInputRunnerRecorder{mock}
	return mock
}

func (_m *MockInputRunner) EXPECT() *_MockInputRunnerRecorder {
	return _m.recorder
}

func (_m *MockInputRunner) InChan() chan *pipeline.PipelinePack {
	ret := _m.ctrl.Call(_m, "InChan")
	ret0, _ := ret[0].(chan *pipeline.PipelinePack)
	return ret0
}

func (_mr *_MockInputRunnerRecorder) InChan() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "InChan")
}

func (_m *MockInputRunner) Inject(_param0 *pipeline.PipelinePack) {
	_m.ctrl.Call(_m, "Inject", _param0)
}

func (_mr *_MockInputRunnerRecorder) Inject(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Inject", arg0)
}

func (_m *MockInputRunner) Input() pipeline.Input {
	ret := _m.ctrl.Call(_m, "Input")
	ret0, _ := ret[0].(pipeline.Input)
	return ret0
}

func (_mr *_MockInputRunnerRecorder) Input() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Input")
}

func (_m *MockInputRunner) LogError(_param0 error) {
	_m.ctrl.Call(_m, "LogError", _param0)
}

func (_mr *_MockInputRunnerRecorder) LogError(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "LogError", arg0)
}

func (_m *MockInputRunner) LogMessage(_param0 string) {
	_m.ctrl.Call(_m, "LogMessage", _param0)
}

func (_mr *_MockInputRunnerRecorder) LogMessage(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "LogMessage", arg0)
}

func (_m *MockInputRunner) Name() string {
	ret := _m.ctrl.Call(_m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

func (_mr *_MockInputRunnerRecorder) Name() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Name")
}

func (_m *MockInputRunner) Plugin() pipeline.Plugin {
	ret := _m.ctrl.Call(_m, "Plugin")
	ret0, _ := ret[0].(pipeline.Plugin)
	return ret0
}

func (_mr *_MockInputRunnerRecorder) Plugin() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Plugin")
}

func (_m *MockInputRunner) PluginGlobals() *pipeline.PluginGlobals {
	ret := _m.ctrl.Call(_m, "PluginGlobals")
	ret0, _ := ret[0].(*pipeline.PluginGlobals)
	return ret0
}

func (_mr *_MockInputRunnerRecorder) PluginGlobals() *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "PluginGlobals")
}

func (_m *MockInputRunner) SetName(_param0 string) {
	_m.ctrl.Call(_m, "SetName", _param0)
}

func (_mr *_MockInputRunnerRecorder) SetName(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SetName", arg0)
}

func (_m *MockInputRunner) Start(_param0 pipeline.PluginHelper, _param1 *sync.WaitGroup) error {
	ret := _m.ctrl.Call(_m, "Start", _param0, _param1)
	ret0, _ := ret[0].(error)
	return ret0
}

func (_mr *_MockInputRunnerRecorder) Start(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Start", arg0, arg1)
}
