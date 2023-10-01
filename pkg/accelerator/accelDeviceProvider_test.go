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

package accelerator_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/accelerator"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AcceleratorProvider", func() {
	Describe("getting new instance of acceleratorProvider", func() {
		Context("with correct arguments", func() {
			rf := &mocks.ResourceFactory{}
			p := accelerator.NewAccelDeviceProvider(rf)
			It("should return valid instance of the provider", func() {
				Expect(p).ToNot(BeNil())
			})
		})
	})
	Describe("getting devices", func() {
		Context("when there are none", func() {
			rf := &mocks.ResourceFactory{}
			p := accelerator.NewAccelDeviceProvider(rf)
			config := &types.ResourceConfig{}
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config, 0)
			It("should return empty slice", func() {
				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})
		})
		Context("when the selector index is invalid", func() {
			rf := &mocks.ResourceFactory{}
			p := accelerator.NewAccelDeviceProvider(rf)
			config := &types.ResourceConfig{}
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config, 0)
			It("should return empty slice when SelectorObjs is nil", func() {
				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})

			It("should return empty slice when selector index is out of range", func() {
				accelDeviceSelector := types.AccelDeviceSelectors{}
				config = &types.ResourceConfig{SelectorObjs: []interface{}{&accelDeviceSelector}}
				devs = p.GetDevices(config, 1)

				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})
		})
	})
	Describe("adding 4 target devices", func() {
		Context("when 2 are valid devices, but 2 are not", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/devices/0000:00:00.2",
					"sys/bus/pci/drivers/fake",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/fake",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/fake",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.1/sriov_numvfs":   []byte("32"),
					"sys/bus/pci/devices/0000:00:00.1/sriov_totalvfs": []byte("64"),
					"sys/bus/pci/devices/0000:00:00.2/sriov_numvfs":   []byte("0"),
				},
			}

			defer fs.Use()()

			rf := factory.NewResourceFactory("fake", "fake", true, false)
			p := accelerator.NewAccelDeviceProvider(rf)
			config := &types.ResourceConfig{
				DeviceType: types.AcceleratorType,
				SelectorObjs: []interface{}{types.AccelDeviceSelectors{
					DeviceSelectors: types.DeviceSelectors{},
				}},
			}

			dev1 := &ghw.PCIDevice{
				Address: "0000:00:00.1",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb"},
				Product: &pcidb.Product{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbb"},
			}

			dev2 := &ghw.PCIDevice{
				Address: "0000:00:00.2",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb"},
				Product: &pcidb.Product{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbb"},
			}

			devInvalid := &ghw.PCIDevice{
				Address: "0000:00:00.3",
				Class:   &pcidb.Class{ID: "completely unparsable"},
				Vendor:  &pcidb.Vendor{},
				Product: &pcidb.Product{},
			}

			devNoSysFs := &ghw.PCIDevice{
				Address: "0000:00:00.4",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "Vendor"},
				Product: &pcidb.Product{Name: "Product"},
			}

			devsToAdd := []*ghw.PCIDevice{dev1, dev2, devInvalid, devNoSysFs}

			err := p.AddTargetDevices(devsToAdd, 0x1024)
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config, 0)
			It("shouldn't return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return 3 devices on GetDiscoveredDevices()", func() {
				Expect(dDevs).To(HaveLen(3))
			})
			It("should return 2 devices on GetDevices()", func() {
				Expect(devs).To(HaveLen(2))
			})
		})
	})
	Describe("getting Filtered devices", func() {
		Context("using selectors", func() {
			It("should correctly filter devices", func() {
				rf := factory.NewResourceFactory("fake", "fake", false, false)
				p := accelerator.NewAccelDeviceProvider(rf)
				all := make([]types.HostDevice, 5)
				mocked := make([]mocks.AccelDevice, 5)

				ve := []string{"8086", "8086", "1111", "2222", "3333"}
				de := []string{"abcd", "123a", "abcd", "2222", "1024"}
				md := []string{"igb_uio", "igb_uio", "igb_uio", "iavf", "vfio-pci"}
				pa := []string{"0000:03:02.0", "0000:03:02.1", "0000:03:02.2", "0000:03:02.3", "0000:03:02.4"}

				for i := range mocked {
					mocked[i].
						On("GetVendor").Return(ve[i]).
						On("GetDeviceCode").Return(de[i]).
						On("GetDriver").Return(md[i]).
						On("GetPciAddr").Return(pa[i])

					all[i] = &mocked[i]
				}

				testCases := []struct {
					name     string
					sel      *types.AccelDeviceSelectors
					expected []types.HostDevice
				}{
					{"vendors", &types.AccelDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Vendors: []string{"8086"}}}, []types.HostDevice{all[0], all[1]}},
					{"devices", &types.AccelDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Devices: []string{"abcd"}}}, []types.HostDevice{all[0], all[2]}},
					{"drivers", &types.AccelDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Drivers: []string{"igb_uio"}}}, []types.HostDevice{all[0], all[1], all[2]}},
					{"pciAddresses", &types.AccelDeviceSelectors{GenericPciDeviceSelectors: types.GenericPciDeviceSelectors{PciAddresses: []string{"0000:03:02.0", "0000:03:02.3"}}},
						[]types.HostDevice{all[0], all[3]}},
				}

				for _, tc := range testCases {
					By(tc.name)
					config := &types.ResourceConfig{SelectorObjs: []interface{}{tc.sel}}
					actual, err := p.GetFilteredDevices(all, config, 0)
					Expect(err).NotTo(HaveOccurred())
					Expect(actual).To(HaveLen(len(tc.expected)))
					Expect(actual).To(ConsistOf(tc.expected))
				}
				By("GetFilteredDevices uses the specified selector index")
				selectors := []*types.AccelDeviceSelectors{
					{
						DeviceSelectors: types.DeviceSelectors{
							Vendors: []string{"None"},
						},
					}, {
						DeviceSelectors: types.DeviceSelectors{
							Vendors: []string{"8086"},
						},
					},
				}
				matchingDevices := []types.HostDevice{all[0], all[1]}

				selectorObjs := make([]interface{}, len(selectors))
				for index := range selectors {
					selectorObjs[index] = selectors[index]
				}
				config := &types.ResourceConfig{SelectorObjs: selectorObjs}

				actual, err := p.GetFilteredDevices(all, config, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(BeEmpty())

				actual, err = p.GetFilteredDevices(all, config, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(actual).To(HaveLen(len(matchingDevices)))
				Expect(actual).To(ConsistOf(matchingDevices))
			})
			It("should error if the selector index is out of bounds", func() {
				rf := factory.NewResourceFactory("fake", "fake", false, false)
				p := accelerator.NewAccelDeviceProvider(rf)
				devs := make([]types.HostDevice, 0)

				rc := &types.ResourceConfig{}

				_, err := p.GetFilteredDevices(devs, rc, 0)

				Expect(err).To(HaveOccurred())

				rc = &types.ResourceConfig{
					SelectorObjs: []interface{}{
						&types.AccelDeviceSelectors{},
					},
				}
				_, err = p.GetFilteredDevices(devs, rc, 1)

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
