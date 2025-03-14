// Code generated by mockery v2.51.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	ws "github.com/lorenzodonini/ocpp-go/ws"
)

// MockClientHandler is an autogenerated mock type for the ClientHandler type
type MockClientHandler struct {
	mock.Mock
}

type MockClientHandler_Expecter struct {
	mock *mock.Mock
}

func (_m *MockClientHandler) EXPECT() *MockClientHandler_Expecter {
	return &MockClientHandler_Expecter{mock: &_m.Mock}
}

// Execute provides a mock function with given fields: client
func (_m *MockClientHandler) Execute(client ws.Channel) {
	_m.Called(client)
}

// MockClientHandler_Execute_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Execute'
type MockClientHandler_Execute_Call struct {
	*mock.Call
}

// Execute is a helper method to define mock.On call
//   - client ws.Channel
func (_e *MockClientHandler_Expecter) Execute(client interface{}) *MockClientHandler_Execute_Call {
	return &MockClientHandler_Execute_Call{Call: _e.mock.On("Execute", client)}
}

func (_c *MockClientHandler_Execute_Call) Run(run func(client ws.Channel)) *MockClientHandler_Execute_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(ws.Channel))
	})
	return _c
}

func (_c *MockClientHandler_Execute_Call) Return() *MockClientHandler_Execute_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockClientHandler_Execute_Call) RunAndReturn(run func(ws.Channel)) *MockClientHandler_Execute_Call {
	_c.Run(run)
	return _c
}

// NewMockClientHandler creates a new instance of MockClientHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockClientHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockClientHandler {
	mock := &MockClientHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
