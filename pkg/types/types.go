// Copyright 2018 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

var (
	// SockDir is the default Kubelet device plugin socket directory
	SockDir = "/var/lib/kubelet/plugins_registry"
	// DeprecatedSockDir is the deprecated Kubelet device plugin socket directory
	DeprecatedSockDir = "/var/lib/kubelet/device-plugins"
)

const (
	// KubeEndPoint is kubelet socket name
	KubeEndPoint = "kubelet.sock"
)

// ResourceConfig contains cofiguration paremeters for a resource pool
type ResourceConfig struct {
	ResourceName string `json:"resourceName"` // the resource name will be added with resource prefix in K8s api
	IsRdma       bool   // the resource support rdma
	Selectors    struct {
		Vendors   []string `json:"vendors,omitempty"`
		Devices   []string `json:"devices,omitempty"`
		Drivers   []string `json:"drivers,omitempty"`
		PfNames   []string `json:"pfNames,omitempty"`
		LinkTypes []string `json:"linkTypes,omitempty"`
	} `json:"selectors,omitempty"` // Whether devices have SRIOV virtual function capabilities or not
}

// ResourceConfList is list of ResourceConfig
type ResourceConfList struct {
	ResourceList []ResourceConfig `json:"resourceList"` // config file: "resourceList" :[{<ResourceConfig configs>},{},{},...]
}

// ResourceServer is gRPC server implements K8s device plugin api
type ResourceServer interface {
	// Device manager API
	pluginapi.DevicePluginServer
	// grpc server related
	Start() error
	Stop() error
	// Init initializes resourcePool
	Init() error
	// Watch watches for socket file deleteion and restart server if needed
	Watch()
}

// ResourceFactory is an interface to get instances of ResourcePool and ResouceServer
type ResourceFactory interface {
	GetResourceServer(ResourcePool) (ResourceServer, error)
	GetInfoProvider(string) DeviceInfoProvider
	GetSelector(string, []string) (DeviceSelector, error)
	GetResourcePool(rc *ResourceConfig, deviceList []PciNetDevice) (ResourcePool, error)
	GetRdmaSpec(string) RdmaSpec
}

// ResourcePool represents a generic resource entity
type ResourcePool interface {
	// extended API for internal use
	GetResourceName() string
	GetDevices() map[string]*pluginapi.Device // for ListAndWatch
	Probe() bool
	GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec
	GetEnvs(deviceIDs []string) []string
	GetMounts(deviceIDs []string) []*pluginapi.Mount
}

// PciNetDevice provides an interface to get device specific information
type PciNetDevice interface {
	GetPFName() string
	GetPfPciAddr() string
	GetVendor() string
	GetDriver() string
	GetDeviceCode() string
	GetPciAddr() string
	GetNetName() string
	IsSriovPF() bool
	GetLinkSpeed() string
	GetLinkType() string
	GetSubClass() string
	GetDeviceSpecs() []*pluginapi.DeviceSpec
	GetEnvVal() string
	GetMounts() []*pluginapi.Mount
	GetAPIDevice() *pluginapi.Device
	GetRdmaSpec() RdmaSpec
	GetVFID() int
}

// DeviceInfoProvider is an interface to get Device Plugin API specific device information
type DeviceInfoProvider interface {
	GetDeviceSpecs(pciAddr string) []*pluginapi.DeviceSpec
	GetEnvVal(pciAddr string) string
	GetMounts(pciAddr string) []*pluginapi.Mount
}

// DeviceSelector provides an interface for filtering a list of devices
type DeviceSelector interface {
	Filter([]PciNetDevice) []PciNetDevice
}

// LinkWatcher in interface to watch Network link status
type LinkWatcher interface { // This is not fully defined yet!!
	Subscribe()
}

// RdmaSpec rdma device data
type RdmaSpec interface {
	IsRdma() bool
	GetRdmaDeviceSpec() []*pluginapi.DeviceSpec
}
