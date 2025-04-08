// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	types "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	mock "github.com/stretchr/testify/mock"
	v1beta1 "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// PciNetDevice is an autogenerated mock type for the PciNetDevice type
type PciNetDevice struct {
	mock.Mock
}

// DeviceExists provides a mock function with no fields
func (_m *PciNetDevice) DeviceExists() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for DeviceExists")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// GetAPIDevice provides a mock function with no fields
func (_m *PciNetDevice) GetAPIDevice() *v1beta1.Device {
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

// GetAcpiIndex provides a mock function with no fields
func (_m *PciNetDevice) GetAcpiIndex() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetAcpiIndex")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDDPProfiles provides a mock function with no fields
func (_m *PciNetDevice) GetDDPProfiles() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetDDPProfiles")
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
func (_m *PciNetDevice) GetDeviceCode() string {
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
func (_m *PciNetDevice) GetDeviceID() string {
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
func (_m *PciNetDevice) GetDeviceSpecs() []*v1beta1.DeviceSpec {
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
func (_m *PciNetDevice) GetDriver() string {
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
func (_m *PciNetDevice) GetEnvVal() map[string]types.AdditionalInfo {
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
func (_m *PciNetDevice) GetFuncID() int {
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
func (_m *PciNetDevice) GetHealth() bool {
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
func (_m *PciNetDevice) GetLinkSpeed() string {
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
func (_m *PciNetDevice) GetLinkType() string {
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
func (_m *PciNetDevice) GetMounts() []*v1beta1.Mount {
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
func (_m *PciNetDevice) GetNetName() string {
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

// GetPKey provides a mock function with no fields
func (_m *PciNetDevice) GetPKey() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetPKey")
	}

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPciAddr provides a mock function with no fields
func (_m *PciNetDevice) GetPciAddr() string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetPciAddr")
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
func (_m *PciNetDevice) GetPfNetName() string {
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
func (_m *PciNetDevice) GetPfPciAddr() string {
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

// GetVdpaDevice provides a mock function with no fields
func (_m *PciNetDevice) GetVdpaDevice() types.VdpaDevice {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetVdpaDevice")
	}

	var r0 types.VdpaDevice
	if rf, ok := ret.Get(0).(func() types.VdpaDevice); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.VdpaDevice)
		}
	}

	return r0
}

// GetVendor provides a mock function with no fields
func (_m *PciNetDevice) GetVendor() string {
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
func (_m *PciNetDevice) IsPfLinkUp() (bool, error) {
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
func (_m *PciNetDevice) IsRdma() bool {
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
func (_m *PciNetDevice) SetHealth(_a0 bool) {
	_m.Called(_a0)
}

// NewPciNetDevice creates a new instance of PciNetDevice. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPciNetDevice(t interface {
	mock.TestingT
	Cleanup(func())
}) *PciNetDevice {
	mock := &PciNetDevice{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
