// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"
import types "github.com/intel/sriov-network-device-plugin/pkg/types"
import v1beta1 "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

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
func (_m *PciNetDevice) GetEnvVal() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
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

// GetPFName provides a mock function with given fields:
func (_m *PciNetDevice) GetPFName() string {
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

// GetRdmaSpec provides a mock function with given fields:
func (_m *PciNetDevice) GetRdmaSpec() types.RdmaSpec {
	ret := _m.Called()

	var r0 types.RdmaSpec
	if rf, ok := ret.Get(0).(func() types.RdmaSpec); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(types.RdmaSpec)
		}
	}

	return r0
}

// GetSubClass provides a mock function with given fields:
func (_m *PciNetDevice) GetSubClass() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
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

// IsSriovPF provides a mock function with given fields:
func (_m *PciNetDevice) IsSriovPF() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
