// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	types "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	mock "github.com/stretchr/testify/mock"
	v1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// APIDevice is an autogenerated mock type for the APIDevice type
type APIDevice struct {
	mock.Mock
}

// GetAPIDevice provides a mock function with no fields
func (_m *APIDevice) GetAPIDevice() *v1beta1.Device {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAPIDevice")
	}

	var r0 *v1beta1.Device
	if rf, ok := ret.Get(0).(func() *v1beta1.Device); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1beta1.Device)
		}
	}

	return r0
}

// GetDeviceSpecs provides a mock function with no fields
func (_m *APIDevice) GetDeviceSpecs() []*v1beta1.DeviceSpec {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDeviceSpecs")
	}

	var r0 []*v1beta1.DeviceSpec
	if rf, ok := ret.Get(0).(func() []*v1beta1.DeviceSpec); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v1beta1.DeviceSpec)
		}
	}

	return r0
}

// GetEnvVal provides a mock function with no fields
func (_m *APIDevice) GetEnvVal() map[string]types.AdditionalInfo {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetEnvVal")
	}

	var r0 map[string]types.AdditionalInfo
	if rf, ok := ret.Get(0).(func() map[string]types.AdditionalInfo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]types.AdditionalInfo)
		}
	}

	return r0
}

// GetHealth provides a mock function with no fields
func (_m *APIDevice) GetHealth() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetHealth")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// GetMounts provides a mock function with no fields
func (_m *APIDevice) GetMounts() []*v1beta1.Mount {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetMounts")
	}

	var r0 []*v1beta1.Mount
	if rf, ok := ret.Get(0).(func() []*v1beta1.Mount); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*v1beta1.Mount)
		}
	}

	return r0
}

// SetHealth provides a mock function with given fields: _a0
func (_m *APIDevice) SetHealth(_a0 bool) {
	_m.Called(_a0)
}

// NewAPIDevice creates a new instance of APIDevice. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAPIDevice(t interface {
	mock.TestingT
	Cleanup(func())
}) *APIDevice {
	mock := &APIDevice{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
