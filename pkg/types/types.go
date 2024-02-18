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
	"encoding/json"

	"github.com/jaypipes/ghw"
	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
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

// DeviceType is custom type to define supported device types
type DeviceType string

// VdpaType is a type to define the supported vdpa device types
type VdpaType string

const (
	// NetDeviceType is DeviceType for network class devices
	NetDeviceType DeviceType = "netDevice"
	// AcceleratorType is DeviceType for accelerator class devices
	AcceleratorType DeviceType = "accelerator"
	// AuxNetDeviceType is DeviceType for auxiliary network devices
	AuxNetDeviceType DeviceType = "auxNetDevice"

	// VdpaVirtioType is VdpaType for virtio-net devices
	VdpaVirtioType VdpaType = "virtio"
	// VdpaVhostType is VdpaType for vhost-vdpa devices
	VdpaVhostType VdpaType = "vhost"
	// VdpaInvalidType is VdpaType to represent an invalid or unsupported type
	VdpaInvalidType VdpaType = "invalid"
)

// SupportedDevices is map of 'device identifier as string' to 'device class hexcode as int'
/*
Supported PCI Device Classes. ref: https://pci-ids.ucw.cz/read/PD
02	Network controller
12	Processing accelerators

Network controller subclasses. ref: https://pci-ids.ucw.cz/read/PD/02
00	Ethernet controller
01	Token ring network controller
02	FDDI network controller
03	ATM network controller
04	ISDN controller
05	WorldFip controller
06	PICMG controller
07	Infiniband controller
08	Fabric controller
80	Network controller

Processing accelerators subclasses. ref: https://pci-ids.ucw.cz/read/PD/12
00	Processing accelerators
01	AI Inference Accelerator
*/
var SupportedDevices = map[DeviceType]int{
	NetDeviceType:    0x02,
	AcceleratorType:  0x12,
	AuxNetDeviceType: 0x02,
}

// SupportedVdpaTypes is a map of 'vdpa device type as string' to 'vdpa device driver as string'
var SupportedVdpaTypes = map[VdpaType]string{
	VdpaVirtioType: kvdpa.VirtioVdpaDriver,
	VdpaVhostType:  kvdpa.VhostVdpaDriver,
}

// ResourceConfig contains configuration parameters for a resource pool
type ResourceConfig struct {
	// optional resource prefix that will overwrite	global prefix specified in cli params
	ResourcePrefix string `json:"resourcePrefix,omitempty"`
	//nolint:lll
	ResourceName    string                    `json:"resourceName"` // the resource name will be added with resource prefix in K8s api
	DeviceType      DeviceType                `json:"deviceType,omitempty"`
	ExcludeTopology bool                      `json:"excludeTopology,omitempty"`
	Selectors       *json.RawMessage          `json:"selectors,omitempty"`
	AdditionalInfo  map[string]AdditionalInfo `json:"additionalInfo,omitempty"`
	SelectorObjs    []interface{}
}

// DeviceSelectors contains common device selectors fields
type DeviceSelectors struct {
	Vendors []string `json:"vendors,omitempty"`
	Devices []string `json:"devices,omitempty"`
	Drivers []string `json:"drivers,omitempty"`
}

// AdditionalInfo contains all the per device or global extra information as key value pairs
type AdditionalInfo map[string]string

// GenericPciDeviceSelectors contains common PCI device selectors fields
type GenericPciDeviceSelectors struct {
	PciAddresses []string `json:"pciAddresses,omitempty"`
}

// GenericNetDeviceSelectors contains common net device selectors fields
type GenericNetDeviceSelectors struct {
	PfNames     []string `json:"pfNames,omitempty"`
	RootDevices []string `json:"rootDevices,omitempty"`
	LinkTypes   []string `json:"linkTypes,omitempty"`
	IsRdma      bool     // the resource support rdma
	AcpiIndexes []string `json:"acpiIndexes,omitempty"`
}

// NetDeviceSelectors contains network device related selectors fields
type NetDeviceSelectors struct {
	DeviceSelectors
	GenericPciDeviceSelectors
	GenericNetDeviceSelectors
	DDPProfiles  []string `json:"ddpProfiles,omitempty"`
	NeedVhostNet bool     // share vhost-net along the selected resource
	VdpaType     VdpaType `json:"vdpaType,omitempty"`
	PKeys        []string `json:"pKeys,omitempty"`
}

// AccelDeviceSelectors contains accelerator(FPGA etc.) related selectors fields
type AccelDeviceSelectors struct {
	DeviceSelectors
	GenericPciDeviceSelectors
}

// AuxNetDeviceSelectors contains auxiliary device related selector fields
type AuxNetDeviceSelectors struct {
	DeviceSelectors
	GenericNetDeviceSelectors
	AuxTypes []string `json:"auxTypes,omitempty"`
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
	// Watch watches for socket file deletion and restart server if needed
	Watch()
}

// ResourceFactory is an interface to get instances of ResourcePool and ResourceServer
type ResourceFactory interface {
	GetResourceServer(ResourcePool) (ResourceServer, error)
	GetDefaultInfoProvider(string, string) []DeviceInfoProvider
	GetSelector(string, []string) (DeviceSelector, error)
	GetResourcePool(rc *ResourceConfig, deviceList []HostDevice) (ResourcePool, error)
	GetRdmaSpec(DeviceType, string) RdmaSpec
	GetVdpaDevice(string) VdpaDevice
	GetDeviceProvider(DeviceType) DeviceProvider
	GetDeviceFilter(*ResourceConfig) ([]interface{}, error)
	GetNadUtils() NadUtils
	FilterBySelector(string, []string, []HostDevice) []HostDevice
}

// ResourcePool represents a generic resource entity
type ResourcePool interface {
	// extended API for internal use
	GetResourceName() string
	GetResourcePrefix() string
	GetDevices() map[string]*pluginapi.Device // for ListAndWatch
	Probe() bool
	GetDeviceSpecs(deviceIDs []string) []*pluginapi.DeviceSpec
	GetEnvs(prefix string, deviceIDs []string) (map[string]string, error)
	GetMounts(deviceIDs []string) []*pluginapi.Mount
	StoreDeviceInfoFile(resourceNamePrefix string) error
	CleanDeviceInfoFile(resourceNamePrefix string) error
	GetCDIName() string
}

// DeviceProvider provides interface for device discovery
type DeviceProvider interface {
	// AddTargetDevices adds a list of devices in a DeviceProvider that matches the 'device class hexcode as int'
	AddTargetDevices([]*ghw.PCIDevice, int) error
	GetDiscoveredDevices() []*ghw.PCIDevice

	// GetDevices runs through the Discovered Devices and returns a list of fully populated HostDevices according to the given ResourceConfig
	GetDevices(*ResourceConfig, int) []HostDevice

	// GetFilteredDevices runs through the provided []HostDevice and filters eligible devices based on the selectors in ResourceConfig. Since
	// the ResourceConfig contains a slice of selectors, the third argument is the index into that array to get the correct selectors to apply.
	GetFilteredDevices([]HostDevice, *ResourceConfig, int) ([]HostDevice, error)

	// ValidConfig performs validation of DeviceType-specific configuration
	ValidConfig(*ResourceConfig) bool
}

// APIDevice provides an interface to expose device information to Kubernetes API
type APIDevice interface {
	// GetDeviceSpecs returns a list of specs which describes host devices
	GetDeviceSpecs() []*pluginapi.DeviceSpec
	// GetEnvVal returns device information to be exposed via environment variable
	GetEnvVal() map[string]AdditionalInfo
	// GetMounts returns a list of host volumes associated with device
	GetMounts() []*pluginapi.Mount
	// GetAPIDevice returns k8s API device
	GetAPIDevice() *pluginapi.Device
}

// HostDevice provides an interface to get generic device information
// represents generic device used by the plugin
type HostDevice interface {
	APIDevice
	// GetVendor returns vendor identifier number of the device
	GetVendor() string
	// GetDriver returns driver name of the device
	GetDriver() string
	// GetDeviceID returns device unique identifier, for ex. PCI address
	GetDeviceID() string
	// GetDeviceCode returns identifier number of the device
	GetDeviceCode() string
}

// PciDevice provides an interface to get generic PCI device information
// represents generic functionality of all PCI devices
// extends HostDevice interface
type PciDevice interface {
	HostDevice
	// GetPciAddr returns PCI address of the device
	GetPciAddr() string
	// GetAcpiIndex returns ACPI index of the device
	GetAcpiIndex() string
}

// NetDevice provides an interface to get generic network device information
// represents generic network device
type NetDevice interface {
	HostDevice
	// GetPfNetName returns netdevice name of the parent PCI device
	GetPfNetName() string
	// GetPfPciAddr returns PCI address of the parent PCI device
	GetPfPciAddr() string
	// GetNetName returns netdevice name of the device
	GetNetName() string
	// GetLinkSpeed returns link speed of the devuce
	GetLinkType() string
	// GetLinkType returns link type of the devuce
	GetLinkSpeed() string
	// GetFuncID returns ID > -1 if device is a PCI Virtual Function or Scalable Function
	GetFuncID() int
	// IsRdma returns true if device is RDMA capable
	IsRdma() bool
}

// PciNetDevice extends PciDevice and NetDevice interfaces
// represents generic PCI network device
type PciNetDevice interface {
	PciDevice
	NetDevice
	// GetDDPProfiles returns DDP profile if device is Intel Ethernet 700 Series NIC
	GetDDPProfiles() string
	// GetVdpaDevice returns VDPA device
	GetVdpaDevice() VdpaDevice
	// GetPKey return IB Partition key
	GetPKey() string
}

// AccelDevice extends PciDevice interface
// represents generic PCI accelerator device
type AccelDevice interface {
	PciDevice
}

// AuxNetDevice extends NetDevice interface
type AuxNetDevice interface {
	NetDevice
	// GetAuxType returns type of auxiliary device
	GetAuxType() string
}

// DeviceInfoProvider is an interface to get Device Plugin API specific device information
type DeviceInfoProvider interface {
	GetName() string
	GetDeviceSpecs() []*pluginapi.DeviceSpec
	GetEnvVal() AdditionalInfo
	GetMounts() []*pluginapi.Mount
}

// DeviceSelector provides an interface for filtering a list of devices
type DeviceSelector interface {
	Filter([]HostDevice) []HostDevice
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

// NadUtils is an interface for Network-Attachment-Definition utilities
type NadUtils interface {
	SaveDeviceInfoFile(resourceName string, deviceID string, devInfo *nettypes.DeviceInfo) error
	CleanDeviceInfoFile(resourceName string, deviceID string) error
}

// VdpaDevice is an interface to access vDPA device information
type VdpaDevice interface {
	GetPath() (string, error)
	GetParent() string
	GetType() VdpaType
}
