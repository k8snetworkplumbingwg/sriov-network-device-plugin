// Code generated by mockery v2.26.1. DO NOT EDIT.

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

// GetAPIDevice provides a mock function with given fields:
func (_m *PciNetDevice) GetAPIDevice() *v1beta1.Device {
	ret := _m.Called()

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

// GetAcpiIndex provides a mock function with given fields:
func (_m *PciNetDevice) GetAcpiIndex() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDDPProfiles provides a mock function with given fields:
func (_m *PciNetDevice) GetDDPProfiles() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDeviceCode provides a mock function with given fields:
func (_m *PciNetDevice) GetDeviceCode() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDeviceID provides a mock function with given fields:
func (_m *PciNetDevice) GetDeviceID() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetDeviceSpecs provides a mock function with given fields:
func (_m *PciNetDevice) GetDeviceSpecs() []*v1beta1.DeviceSpec {
	ret := _m.Called()

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

// GetDriver provides a mock function with given fields:
func (_m *PciNetDevice) GetDriver() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetEnvVal provides a mock function with given fields:
func (_m *PciNetDevice) GetEnvVal() map[string]types.AdditionalInfo {
	ret := _m.Called()

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

// GetFuncID provides a mock function with given fields:
func (_m *PciNetDevice) GetFuncID() int {
	ret := _m.Called()

	var r0 int
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	return r0
}

// GetLinkSpeed provides a mock function with given fields:
func (_m *PciNetDevice) GetLinkSpeed() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetLinkType provides a mock function with given fields:
func (_m *PciNetDevice) GetLinkType() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetMounts provides a mock function with given fields:
func (_m *PciNetDevice) GetMounts() []*v1beta1.Mount {
	ret := _m.Called()

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

// GetNetName provides a mock function with given fields:
func (_m *PciNetDevice) GetNetName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPciAddr provides a mock function with given fields:
func (_m *PciNetDevice) GetPciAddr() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPfNetName provides a mock function with given fields:
func (_m *PciNetDevice) GetPfNetName() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetPfPciAddr provides a mock function with given fields:
func (_m *PciNetDevice) GetPfPciAddr() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetVdpaDevice provides a mock function with given fields:
func (_m *PciNetDevice) GetVdpaDevice() types.VdpaDevice {
	ret := _m.Called()

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

// GetVendor provides a mock function with given fields:
func (_m *PciNetDevice) GetVendor() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// IsRdma provides a mock function with given fields:
func (_m *PciNetDevice) IsRdma() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

type mockConstructorTestingTNewPciNetDevice interface {
	mock.TestingT
	Cleanup(func())
}

// NewPciNetDevice creates a new instance of PciNetDevice. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewPciNetDevice(t mockConstructorTestingTNewPciNetDevice) *PciNetDevice {
	mock := &PciNetDevice{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
