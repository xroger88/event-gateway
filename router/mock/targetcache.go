// Automatically generated by MockGen. DO NOT EDIT!
// Source: github.com/serverless/event-gateway/router (interfaces: Targeter)

package mock

import (
	gomock "github.com/golang/mock/gomock"
	event "github.com/serverless/event-gateway/event"
	functions "github.com/serverless/event-gateway/functions"
	pathtree "github.com/serverless/event-gateway/internal/pathtree"
)

// Mock of Targeter interface
type MockTargeter struct {
	ctrl     *gomock.Controller
	recorder *_MockTargeterRecorder
}

// Recorder for MockTargeter (not exported)
type _MockTargeterRecorder struct {
	mock *MockTargeter
}

func NewMockTargeter(ctrl *gomock.Controller) *MockTargeter {
	mock := &MockTargeter{ctrl: ctrl}
	mock.recorder = &_MockTargeterRecorder{mock}
	return mock
}

func (_m *MockTargeter) EXPECT() *_MockTargeterRecorder {
	return _m.recorder
}

func (_m *MockTargeter) Function(_param0 functions.FunctionID) *functions.Function {
	ret := _m.ctrl.Call(_m, "Function", _param0)
	ret0, _ := ret[0].(*functions.Function)
	return ret0
}

func (_mr *_MockTargeterRecorder) Function(arg0 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "Function", arg0)
}

func (_m *MockTargeter) HTTPBackingFunction(_param0 string, _param1 string) (*functions.FunctionID, pathtree.Params) {
	ret := _m.ctrl.Call(_m, "HTTPBackingFunction", _param0, _param1)
	ret0, _ := ret[0].(*functions.FunctionID)
	ret1, _ := ret[1].(pathtree.Params)
	return ret0, ret1
}

func (_mr *_MockTargeterRecorder) HTTPBackingFunction(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "HTTPBackingFunction", arg0, arg1)
}

func (_m *MockTargeter) SubscribersOfEvent(_param0 string, _param1 event.Type) []functions.FunctionID {
	ret := _m.ctrl.Call(_m, "SubscribersOfEvent", _param0, _param1)
	ret0, _ := ret[0].([]functions.FunctionID)
	return ret0
}

func (_mr *_MockTargeterRecorder) SubscribersOfEvent(arg0, arg1 interface{}) *gomock.Call {
	return _mr.mock.ctrl.RecordCall(_mr.mock, "SubscribersOfEvent", arg0, arg1)
}
