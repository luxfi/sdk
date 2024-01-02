// Copyright (C) 2023-2024, Lux Partners Limited. All rights reserved.
// See the file LICENSE for licensing terms.
//

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/luxdefi/vmsdk/chain (interfaces: Action)

// Package chain is a generated GoMock package.
package chain

import (
	context "context"
	reflect "reflect"

	ids "github.com/luxdefi/node/ids"
	codec "github.com/luxdefi/vmsdk/codec"
	gomock "github.com/golang/mock/gomock"
)

// MockAction is a mock of Action interface.
type MockAction struct {
	ctrl     *gomock.Controller
	recorder *MockActionMockRecorder
}

// MockActionMockRecorder is the mock recorder for MockAction.
type MockActionMockRecorder struct {
	mock *MockAction
}

// NewMockAction creates a new mock instance.
func NewMockAction(ctrl *gomock.Controller) *MockAction {
	mock := &MockAction{ctrl: ctrl}
	mock.recorder = &MockActionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAction) EXPECT() *MockActionMockRecorder {
	return m.recorder
}

// Execute mocks base method.
func (m *MockAction) Execute(arg0 context.Context, arg1 Rules, arg2 Database, arg3 int64, arg4 Auth, arg5 ids.ID, arg6 bool) (*Result, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Execute", arg0, arg1, arg2, arg3, arg4, arg5, arg6)
	ret0, _ := ret[0].(*Result)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Execute indicates an expected call of Execute.
func (mr *MockActionMockRecorder) Execute(arg0, arg1, arg2, arg3, arg4, arg5, arg6 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockAction)(nil).Execute), arg0, arg1, arg2, arg3, arg4, arg5, arg6)
}

// Marshal mocks base method.
func (m *MockAction) Marshal(arg0 *codec.Packer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Marshal", arg0)
}

// Marshal indicates an expected call of Marshal.
func (mr *MockActionMockRecorder) Marshal(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Marshal", reflect.TypeOf((*MockAction)(nil).Marshal), arg0)
}

// MaxUnits mocks base method.
func (m *MockAction) MaxUnits(arg0 Rules) uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MaxUnits", arg0)
	ret0, _ := ret[0].(uint64)
	return ret0
}

// MaxUnits indicates an expected call of MaxUnits.
func (mr *MockActionMockRecorder) MaxUnits(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MaxUnits", reflect.TypeOf((*MockAction)(nil).MaxUnits), arg0)
}

// StateKeys mocks base method.
func (m *MockAction) StateKeys(arg0 Auth, arg1 ids.ID) [][]byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StateKeys", arg0, arg1)
	ret0, _ := ret[0].([][]byte)
	return ret0
}

// StateKeys indicates an expected call of StateKeys.
func (mr *MockActionMockRecorder) StateKeys(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StateKeys", reflect.TypeOf((*MockAction)(nil).StateKeys), arg0, arg1)
}

// ValidRange mocks base method.
func (m *MockAction) ValidRange(arg0 Rules) (int64, int64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidRange", arg0)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(int64)
	return ret0, ret1
}

// ValidRange indicates an expected call of ValidRange.
func (mr *MockActionMockRecorder) ValidRange(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidRange", reflect.TypeOf((*MockAction)(nil).ValidRange), arg0)
}
