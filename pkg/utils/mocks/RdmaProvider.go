// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// RdmaProvider is an autogenerated mock type for the RdmaProvider type
type RdmaProvider struct {
	mock.Mock
}

// GetRdmaCharDevices provides a mock function with given fields: rdmaDeviceName
func (_m *RdmaProvider) GetRdmaCharDevices(rdmaDeviceName string) []string {
	ret := _m.Called(rdmaDeviceName)

	if len(ret) == 0 {
		panic("no return value specified for GetRdmaCharDevices")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(rdmaDeviceName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// GetRdmaDevicesForAuxdev provides a mock function with given fields: deviceID
func (_m *RdmaProvider) GetRdmaDevicesForAuxdev(deviceID string) []string {
	ret := _m.Called(deviceID)

	if len(ret) == 0 {
		panic("no return value specified for GetRdmaDevicesForAuxdev")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(deviceID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// GetRdmaDevicesForPcidev provides a mock function with given fields: pciAddr
func (_m *RdmaProvider) GetRdmaDevicesForPcidev(pciAddr string) []string {
	ret := _m.Called(pciAddr)

	if len(ret) == 0 {
		panic("no return value specified for GetRdmaDevicesForPcidev")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(pciAddr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// NewRdmaProvider creates a new instance of RdmaProvider. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRdmaProvider(t interface {
	mock.TestingT
	Cleanup(func())
}) *RdmaProvider {
	mock := &RdmaProvider{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
