// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	types "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	mock "github.com/stretchr/testify/mock"
	v1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// AuxNetDevice is an autogenerated mock type for the AuxNetDevice type
type AuxNetDevice struct {
	mock.Mock
}

// GetAPIDevice provides a mock function with no fields
func (_m *AuxNetDevice) GetAPIDevice() *v1beta1.Device {
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

// GetAuxType provides a mock function with no fields
func (_m *AuxNetDevice) GetAuxType() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAuxType")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDeviceCode provides a mock function with no fields
func (_m *AuxNetDevice) GetDeviceCode() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDeviceCode")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDeviceID provides a mock function with no fields
func (_m *AuxNetDevice) GetDeviceID() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDeviceID")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDeviceSpecs provides a mock function with no fields
func (_m *AuxNetDevice) GetDeviceSpecs() []*v1beta1.DeviceSpec {
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

// GetDriver provides a mock function with no fields
func (_m *AuxNetDevice) GetDriver() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDriver")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetEnvVal provides a mock function with no fields
func (_m *AuxNetDevice) GetEnvVal() map[string]types.AdditionalInfo {
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

// GetFuncID provides a mock function with no fields
func (_m *AuxNetDevice) GetFuncID() int {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetFuncID")
	}

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetHealth provides a mock function with no fields
func (_m *AuxNetDevice) GetHealth() bool {
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

// GetLinkSpeed provides a mock function with no fields
func (_m *AuxNetDevice) GetLinkSpeed() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLinkSpeed")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetLinkType provides a mock function with no fields
func (_m *AuxNetDevice) GetLinkType() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetLinkType")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetMounts provides a mock function with no fields
func (_m *AuxNetDevice) GetMounts() []*v1beta1.Mount {
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

// GetNetName provides a mock function with no fields
func (_m *AuxNetDevice) GetNetName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetNetName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPfNetName provides a mock function with no fields
func (_m *AuxNetDevice) GetPfNetName() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetPfNetName")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPfPciAddr provides a mock function with no fields
func (_m *AuxNetDevice) GetPfPciAddr() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetPfPciAddr")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetVendor provides a mock function with no fields
func (_m *AuxNetDevice) GetVendor() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetVendor")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// IsPfLinkUp provides a mock function with no fields
func (_m *AuxNetDevice) IsPfLinkUp() (bool, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsPfLinkUp")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func() (bool, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsRdma provides a mock function with no fields
func (_m *AuxNetDevice) IsRdma() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for IsRdma")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// SetHealth provides a mock function with given fields: _a0
func (_m *AuxNetDevice) SetHealth(_a0 bool) {
	_m.Called(_a0)
}

// NewAuxNetDevice creates a new instance of AuxNetDevice. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAuxNetDevice(t interface {
	mock.TestingT
	Cleanup(func())
}) *AuxNetDevice {
	mock := &AuxNetDevice{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
