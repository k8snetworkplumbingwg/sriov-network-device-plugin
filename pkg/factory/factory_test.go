// Copyright 2020 Intel Corp. All Rights Reserved.
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

package factory_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	utilmocks "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func TestFactory(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Factory Suite")
}

var _ = Describe("Factory", func() {
	Describe("getting factory instance", func() {
		Context("always", func() {
			It("should return the same instance", func() {
				f0 := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
				Expect(f0).NotTo(BeNil())
				f1 := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
				Expect(f1).To(Equal(f0))
			})
		})
	})
	DescribeTable("getting info provider",
		func(name string, expected reflect.Type) {
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			p := f.GetDefaultInfoProvider("fakePCIAddr", name)
			Expect(p).To(HaveLen(2)) // for all the providers except netdevice we expect 2 info providers
			Expect(reflect.TypeOf(p[1])).To(Equal(expected))

		},
		Entry("vfio-pci", "vfio-pci", reflect.TypeOf(infoprovider.NewVfioInfoProvider("fakePCIAddr"))),
		Entry("uio", "uio", reflect.TypeOf(infoprovider.NewUioInfoProvider("fakePCIAddr"))),
		Entry("igb_uio", "igb_uio", reflect.TypeOf(infoprovider.NewUioInfoProvider("fakePCIAddr"))),
	)

	Describe("getting info provider for generic netdevice", func() {
		f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
		p := f.GetDefaultInfoProvider("fakePCIAddr", "netdevice")
		Expect(p).To(HaveLen(1)) // for all the providers except netdevice we expect 2 info providers
		Expect(reflect.TypeOf(p[0])).To(Equal(reflect.TypeOf(infoprovider.NewGenericInfoProvider("fakePCIAddr"))))
	})

	DescribeTable("getting selector",
		func(selector string, shouldSucceed bool, expected reflect.Type) {
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			v := []string{"val1", "val2", "val3"}
			s, e := f.GetSelector(selector, v)

			if shouldSucceed {
				Expect(reflect.TypeOf(s)).To(Equal(expected))
				Expect(e).NotTo(HaveOccurred())
			} else {
				Expect(s).To(BeNil())
				Expect(e).To(HaveOccurred())
			}

			// if statement below because gomega refuses to do "nil == nil" assertions
			if expected != nil {
				Expect(reflect.TypeOf(s)).To(Equal(expected))
			} else {
				Expect(reflect.TypeOf(s)).To(BeNil())
			}
		},
		Entry("vendors", "vendors", true, reflect.TypeOf(resources.NewVendorSelector([]string{}))),
		Entry("devices", "devices", true, reflect.TypeOf(resources.NewDeviceSelector([]string{}))),
		Entry("drivers", "drivers", true, reflect.TypeOf(resources.NewDriverSelector([]string{}))),
		Entry("pciAddresses", "pciAddresses", true, reflect.TypeOf(resources.NewPciAddressSelector([]string{}))),
		Entry("pfNames", "pfNames", true, reflect.TypeOf(resources.NewPfNameSelector([]string{}))),
		Entry("rootDevices", "rootDevices", true, reflect.TypeOf(resources.NewRootDeviceSelector([]string{}))),
		Entry("linkTypes", "linkTypes", true, reflect.TypeOf(resources.NewLinkTypeSelector([]string{}))),
		Entry("ddpProfiles", "ddpProfiles", true, reflect.TypeOf(resources.NewDdpSelector([]string{}))),
		Entry("pKeys", "pKeys", true, reflect.TypeOf(resources.NewPKeySelector([]string{}))),
		Entry("invalid", "fakeAndInvalid", false, reflect.TypeOf(nil)),
	)
	Describe("getting resource pool for netdevice", func() {
		Context("with all types of selectors used and matching devices found", func() {
			utils.SetDefaultMockNetlinkProvider()
			var (
				rp   types.ResourcePool
				err  error
				devs []types.HostDevice
			)
			BeforeEach(func() {
				f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)

				devs = make([]types.HostDevice, 4)
				vendors := []string{"8086", "8086", "8086", "1234"}
				codes := []string{"1111", "1111", "1234", "4321"}
				drivers := []string{"vfio-pci", "i40evf", "igb_uio", "igb_uio"}
				pciAddr := []string{"0000:03:02.0", "0000:03:02.1", "0000:03:02.2", "0000:03:02.3"}
				pfNames := []string{"enp2s0f2", "ens0", "eth0", "net2"}
				rootDevices := []string{"0000:86:00.0", "0000:86:00.1", "0000:86:00.2", "0000:86:00.3"}
				linkTypes := []string{"ether", "infiniband", "other", "other2"}
				ddpProfiles := []string{"GTP", "PPPoE", "GTP", "PPPoE"}
				pKeys := []string{"0x1", "0x2", "0xABCD", "0x50"}
				for i := range devs {
					d := &mocks.PciNetDevice{}
					d.On("GetVendor").Return(vendors[i]).
						On("GetDeviceCode").Return(codes[i]).
						On("GetDriver").Return(drivers[i]).
						On("GetPciAddr").Return(pciAddr[i]).
						On("GetDeviceID").Return(pciAddr[i]).
						On("GetPfNetName").Return(pfNames[i]).
						On("GetPfPciAddr").Return(rootDevices[i]).
						On("GetAPIDevice").Return(&pluginapi.Device{}).
						On("GetLinkType").Return(linkTypes[i]).
						On("GetDDPProfiles").Return(ddpProfiles[i]).
						On("GetPKey").Return(pKeys[i])
					devs[i] = d
				}

				var selectors json.RawMessage
				err = selectors.UnmarshalJSON([]byte(`
					[
						{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["vfio-pci"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"],
							"pKeys": ["0x1"]
						}
					]`),
				)

				Expect(err).NotTo(HaveOccurred())

				c := types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors,
					DeviceType:   types.NetDeviceType,
				}

				rp, err = f.GetResourcePool(&c, devs)

			})
			It("should return valid resource pool", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(4))
				Expect(rp.GetDevices()).To(HaveKey("0000:03:02.0"))
			})
			It("should not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	DescribeTable("getting resource pool",
		func(selectorBytes []byte, hasDevices []string) {
			// create factory
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)

			// parse selector configuration & create resource config
			var selectors json.RawMessage
			err := selectors.UnmarshalJSON(selectorBytes)
			Expect(err).NotTo(HaveOccurred())

			c := &types.ResourceConfig{
				ResourceName: "fake",
				Selectors:    &selectors,
				DeviceType:   types.NetDeviceType,
			}

			// create mock devices
			devs := make([]types.HostDevice, 4)
			vendors := []string{"8086", "8086", "8086", "8086"}
			codes := []string{"1111", "1111", "1111", "1111"}
			drivers := []string{"iavf", "iavf", "vfio-pci", "vfio-pci"}
			pciAddr := []string{"0000:03:02.0", "0000:03:02.1", "0000:03:02.2", "0000:03:02.3"}
			pfNames := []string{"enp2s0f2", "ens0", "eth0", "net2"}
			rootDevices := []string{"0000:86:00.0", "0000:86:00.1", "0000:86:00.2", "0000:86:00.3"}
			linkTypes := []string{"ether", "infiniband", "other", "other2"}
			ddpProfiles := []string{"GTP", "PPPoE", "GTP", "PPPoE"}
			pKeys := []string{"0x1", "0x2", "0xABCD", "0x50"}
			for i := range devs {
				d := &mocks.PciNetDevice{}
				d.On("GetVendor").Return(vendors[i]).
					On("GetDeviceCode").Return(codes[i]).
					On("GetDriver").Return(drivers[i]).
					On("GetPciAddr").Return(pciAddr[i]).
					On("GetDeviceID").Return(pciAddr[i]).
					On("GetPfNetName").Return(pfNames[i]).
					On("GetPfPciAddr").Return(rootDevices[i]).
					On("GetAPIDevice").Return(&pluginapi.Device{}).
					On("GetLinkType").Return(linkTypes[i]).
					On("GetDDPProfiles").Return(ddpProfiles[i]).
					On("GetFuncID").Return(-1).
					On("GetPKey").Return(pKeys[i])
				devs[i] = d
			}

			deviceAllocated := make(map[string]bool)
			dp := f.GetDeviceProvider(c.DeviceType)
			c.SelectorObjs, err = f.GetDeviceFilter(c)
			Expect(err).NotTo(HaveOccurred())
			filteredDevices := make([]types.HostDevice, 0)
			for index := range c.SelectorObjs {
				currentSelectorFilteredDevices, err := dp.GetFilteredDevices(devs, c, index)
				Expect(err).NotTo(HaveOccurred())

				unallocatedDevices := []types.HostDevice{}
				for _, dev := range currentSelectorFilteredDevices {
					if !deviceAllocated[dev.GetDeviceID()] {
						deviceAllocated[dev.GetDeviceID()] = true
						unallocatedDevices = append(unallocatedDevices, dev)
					}
				}
				filteredDevices = append(filteredDevices, unallocatedDevices...)
			}

			rp, err := f.GetResourcePool(c, filteredDevices)
			Expect(err).NotTo(HaveOccurred())

			if hasDevices == nil {
				Expect(rp).Should(BeNil())
			} else {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(len(hasDevices)))
				for _, devAddr := range hasDevices {
					Expect(rp.GetDevices()).To(HaveKey(devAddr))
				}
			}
		},
		Entry("with a single selector object should match devices", []byte(`
						{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["iavf","vfio-pci"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"],
							"pKeys": ["0x1"]
						}`), []string{"0000:03:02.0"}),
		Entry("with a slice of one selector object it should match devices", []byte(`
						[{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["iavf","vfio-pci"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"],
							"pKeys": ["0x1"]
						}]`), []string{"0000:03:02.0"}),
		Entry("with more than one selector object, it should match devices from all selector objects", []byte(`
						[{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["iavf"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"],
							"pKeys": ["0x1"]
						}, {
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["vfio-pci"],
							"pciAddresses": ["0000:03:02.3"],
							"pfNames": ["net2"],
							"rootDevices": ["0000:86:00.3"],
							"linkTypes": ["other2"],
							"ddpProfiles": ["PPPoE"],
							"pKeys": ["0x50"]
						}]`), []string{"0000:03:02.0", "0000:03:02.3"}),
	)
	Describe("getting exclusive resource pool for netdevice", func() {
		Context("with all types of selectors used and matching devices found", func() {
			utils.SetDefaultMockNetlinkProvider()
			var (
				rp   types.ResourcePool
				rp2  types.ResourcePool
				err  error
				devs []types.HostDevice
			)
			BeforeEach(func() {
				f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
				devs = make([]types.HostDevice, 4)
				vendors := []string{"8086", "8086", "8086", "8086"}
				codes := []string{"1111", "1111", "1111", "1111"}
				drivers := []string{"iavf", "iavf", "vfio-pci", "vfio-pci"}
				pciAddr := []string{"0000:03:02.0", "0000:03:02.0", "0000:03:02.0", "0000:03:02.0"}
				pfNames := []string{"enp2s0f2", "ens0", "eth0", "net2"}
				rootDevices := []string{"0000:86:00.0", "0000:86:00.1", "0000:86:00.2", "0000:86:00.3"}
				linkTypes := []string{"ether", "infiniband", "other", "other2"}
				ddpProfiles := []string{"GTP", "PPPoE", "GTP", "PPPoE"}
				pKeys := []string{"0x1", "0x2", "0xABCD", "0x50"}
				for i := range devs {
					d := &mocks.PciNetDevice{}
					d.On("GetVendor").Return(vendors[i]).
						On("GetDeviceCode").Return(codes[i]).
						On("GetDriver").Return(drivers[i]).
						On("GetPciAddr").Return(pciAddr[i]).
						On("GetDeviceID").Return(pciAddr[i]).
						On("GetPfNetName").Return(pfNames[i]).
						On("GetPfPciAddr").Return(rootDevices[i]).
						On("GetAPIDevice").Return(&pluginapi.Device{}).
						On("GetLinkType").Return(linkTypes[i]).
						On("GetDDPProfiles").Return(ddpProfiles[i]).
						On("GetFuncID").Return(-1).
						On("GetPKey").Return(pKeys[i])
					devs[i] = d
				}

				var selectors json.RawMessage
				err = selectors.UnmarshalJSON([]byte(`
						{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["iavf","vfio-pci"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"],
							"pKeys": ["0x1"]
						}
					`),
				)
				Expect(err).NotTo(HaveOccurred())

				var selectors2 json.RawMessage
				err = selectors2.UnmarshalJSON([]byte(`
						[{
							"vendors": ["8086"],
							"devices": ["1111"],
							"drivers": ["iavf","vfio-pci"],
							"pciAddresses": ["0000:03:02.0"],
							"pfNames": ["enp2s0f2"],
							"rootDevices": ["0000:86:00.0"],
							"linkTypes": ["ether"],
							"ddpProfiles": ["GTP"],
							"pKeys": ["0x1"]
						}]
					`),
				)
				Expect(err).NotTo(HaveOccurred())

				c := &types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors,
					DeviceType:   types.NetDeviceType,
				}
				deviceAllocated := make(map[string]bool)
				dp := f.GetDeviceProvider(c.DeviceType)
				c.SelectorObjs, err = f.GetDeviceFilter(c)
				Expect(err).NotTo(HaveOccurred())
				filteredDevices, err := dp.GetFilteredDevices(devs, c, 0)
				Expect(err).NotTo(HaveOccurred())

				filteredDevicesTemp := []types.HostDevice{}
				for _, dev := range filteredDevices {
					if !deviceAllocated[dev.GetDeviceID()] {
						deviceAllocated[dev.GetDeviceID()] = true
						filteredDevicesTemp = append(filteredDevicesTemp, dev)
					}
				}
				filteredDevices = filteredDevicesTemp

				rp, err = f.GetResourcePool(c, filteredDevices)
				Expect(err).NotTo(HaveOccurred())

				// Second config definition
				c2 := &types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors2,
					DeviceType:   types.NetDeviceType,
				}

				dp2 := f.GetDeviceProvider(c2.DeviceType)
				c2.SelectorObjs, err = f.GetDeviceFilter(c2)
				Expect(err).NotTo(HaveOccurred())
				filteredDevices, err = dp2.GetFilteredDevices(devs, c2, 0)
				Expect(err).NotTo(HaveOccurred())

				filteredDevicesTemp = []types.HostDevice{}
				for _, dev := range filteredDevices {
					if !deviceAllocated[dev.GetDeviceID()] {
						deviceAllocated[dev.GetDeviceID()] = true
						filteredDevicesTemp = append(filteredDevicesTemp, dev)
					}
				}
				filteredDevices = filteredDevicesTemp

				rp2, err = f.GetResourcePool(c2, filteredDevices)
				Expect(err).NotTo(HaveOccurred())

			})
			It("should return valid exclusive resource pool", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(1))
				Expect(rp.GetDevices()).To(HaveKey("0000:03:02.0"))
				// Check second resource pool to make sure nothing got added to it.
				Expect(rp2).Should(BeNil())
			})
			It("should not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	Describe("getting resource pool for accelerator", func() {
		Context("with all types of selectors used and matching devices found", func() {
			utils.SetDefaultMockNetlinkProvider()
			var (
				rp   types.ResourcePool
				err  error
				devs []types.HostDevice
			)
			BeforeEach(func() {
				f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)

				devs = make([]types.HostDevice, 1)
				vendors := []string{"8086"}
				codes := []string{"1024"}
				drivers := []string{"uio_pci_generic"}
				pciAddr := []string{"0000:04:00.0"}
				for i := range devs {
					d := &mocks.AccelDevice{}
					d.On("GetVendor").Return(vendors[i]).
						On("GetDeviceCode").Return(codes[i]).
						On("GetDriver").Return(drivers[i]).
						On("GetPciAddr").Return(pciAddr[i]).
						On("GetDeviceID").Return(pciAddr[i]).
						On("GetAPIDevice").Return(&pluginapi.Device{})
					devs[i] = d
				}

				var selectors json.RawMessage
				err = selectors.UnmarshalJSON([]byte(`
					[
						{
							"vendors": ["8086"],
							"devices": ["1024"],
							"drivers": ["uio_pci_generic"],
						}
					]`),
				)

				Expect(err).NotTo(HaveOccurred())

				c := types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors,
					DeviceType:   types.AcceleratorType,
				}

				rp, err = f.GetResourcePool(&c, devs)
			})
			It("should return valid resource pool", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(1))
				Expect(rp.GetDevices()).To(HaveKey("0000:04:00.0"))
			})
			It("should not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	Describe("getting resource pool for auxnetdevice", func() {
		Context("with all types of selectors used and matching devices found", func() {
			utils.SetDefaultMockNetlinkProvider()
			var (
				rp   types.ResourcePool
				err  error
				devs []types.HostDevice
			)
			BeforeEach(func() {
				f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)

				devs = make([]types.HostDevice, 4)
				vendors := []string{"8086", "8086", "15b3", "15b3"}
				codes := []string{"1111", "1111", "2222", "2222"}
				drivers := []string{"igb", "igb", "mlx5_core", "mlx5_core"}
				deviceID := []string{"igb.eth.0", "igb.eth.1", "mlx5_core.eth.0", "mlx5_core.sf.1"}
				pfNames := []string{"eno1", "ib0", "eth0", "eth0"}
				rootDevices := []string{"0000:86:00.0", "0000:86:00.0", "0000:05:00.0", "0000:05:00.0"}
				linkTypes := []string{"ether", "infiniband", "ether", "ether"}
				auxTypes := []string{"eth", "eth", "eth", "sf"}
				for i := range devs {
					d := &mocks.AuxNetDevice{}
					d.On("GetVendor").Return(vendors[i]).
						On("GetDeviceCode").Return(codes[i]).
						On("GetDriver").Return(drivers[i]).
						On("GetDeviceID").Return(deviceID[i]).
						On("GetPFName").Return(pfNames[i]).
						On("GetPfPciAddr").Return(rootDevices[i]).
						On("GetAPIDevice").Return(&pluginapi.Device{}).
						On("GetLinkType").Return(linkTypes[i]).
						On("GetFuncID").Return(-1).
						On("GetAuxType").Return(auxTypes[i])
					devs[i] = d
				}

				var selectors json.RawMessage
				err = selectors.UnmarshalJSON([]byte(`
					[
						{
							"vendors": ["15b3"],
							"devices": ["2222"],
							"drivers": ["mlx5_core"],
							"pfNames": ["eth1"],
							"rootDevices": ["0000:05:00.0"],
							"linkTypes": ["ether"],
							"auxTypes": ["eth", "sf"],
						}
					]`),
				)

				Expect(err).NotTo(HaveOccurred())

				c := types.ResourceConfig{
					ResourceName: "fake",
					Selectors:    &selectors,
					DeviceType:   types.AuxNetDeviceType,
				}

				rp, err = f.GetResourcePool(&c, devs)
			})
			It("should return valid resource pool", func() {
				Expect(rp).NotTo(BeNil())
				Expect(rp.GetDevices()).To(HaveLen(4))
				Expect(rp.GetDevices()).To(HaveKey("mlx5_core.sf.1"))
			})
			It("should not fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	DescribeTable("getting device provider",
		func(dt types.DeviceType, shouldSucceed bool) {
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			p := f.GetDeviceProvider(dt)
			if shouldSucceed {
				Expect(p).NotTo(BeNil())
			} else {
				Expect(p).To(BeNil())
			}
		},
		Entry("of a netdevice shouldn't return nil", types.NetDeviceType, true),
		Entry("of an accelerator shouldn't return nil", types.AcceleratorType, true),
		Entry("of an auxnetdevice shouldn't return nil", types.AuxNetDeviceType, true),
		Entry("of unsupported device type should return nil", nil, false),
	)
	DescribeTable("getting device filter",
		func(dt types.DeviceType, sel string, expected interface{}, shouldSucceed bool) {
			// prepare json rawmessage selector
			s := json.RawMessage{}
			err := s.UnmarshalJSON([]byte(sel))
			Expect(err).NotTo(HaveOccurred())

			rc := &types.ResourceConfig{
				DeviceType: dt,
				Selectors:  &s,
			}

			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)

			_, e := f.GetDeviceFilter(rc)
			if shouldSucceed {
				Expect(e).NotTo(HaveOccurred())
			} else {
				Expect(e).To(HaveOccurred())
			}
		},
		Entry("successful netdevice", types.NetDeviceType, `{"PfNames":["eth0"]}`, nil, true),
		Entry("failed netdevice", types.NetDeviceType, `invalid selectors!`, nil, false),
		Entry("successful accelerator", types.AcceleratorType, `{"Vendors": ["8086"]}`, nil, true),
		Entry("failed accelerator", types.AcceleratorType, `invalid selectors!`, nil, false),
		Entry("successful auxnetdevice", types.AuxNetDeviceType, `{"auxTypes": ["foo"]}`, nil, true),
		Entry("failed auxnetdevice", types.AuxNetDeviceType, `invalid selectors!`, nil, false),
		Entry("unsupported type", nil, ``, nil, false),
	)
	Describe("getting rdma spec", func() {
		Context("check c rdma spec", func() {
			mockProvider := &utilmocks.NetlinkProvider{}
			mockProvider.On("HasRdmaParam", mock.AnythingOfType("string"),
				mock.AnythingOfType("string")).Return(false, nil)
			utils.SetNetlinkProviderInst(mockProvider)
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			rs1 := f.GetRdmaSpec(types.NetDeviceType, "0000:00:00.1")
			rs2 := f.GetRdmaSpec(types.AcceleratorType, "0000:00:00.2")
			rs3 := f.GetRdmaSpec(types.AuxNetDeviceType, "foo.bar.3")
			It("shoud return valid rdma spec for netdevice", func() {
				Expect(rs1).ToNot(BeNil())
				Expect(rs1.IsRdma()).ToNot(BeTrue())
				Expect(rs1.GetRdmaDeviceSpec()).To(HaveLen(0))
			})
			It("shoud return nil for accelerator", func() {
				Expect(rs2).To(BeNil())
			})
			It("shoud return valid rdma spec for auxnetdevice", func() {
				Expect(rs3).ToNot(BeNil())
				Expect(rs3.IsRdma()).ToNot(BeTrue())
				Expect(rs3.GetRdmaDeviceSpec()).To(HaveLen(0))
			})
		})
	})
	Describe("getting resource server", func() {
		Context("when resource pool is nil", func() {
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			rs, e := f.GetResourceServer(nil)
			It("should fail", func() {
				Expect(e).To(HaveOccurred())
				Expect(rs).To(BeNil())
			})
		})
		Context("when resource pool uses overridden prefix", func() {
			f := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			rp := mocks.ResourcePool{}
			rp.On("GetResourcePrefix").Return("overridden").
				On("GetResourceName").Return("fake")
			rs, e := f.GetResourceServer(&rp)
			It("should not fail", func() {
				Expect(e).NotTo(HaveOccurred())
				Expect(rs).NotTo(BeNil())
			})
		})
	})
})
