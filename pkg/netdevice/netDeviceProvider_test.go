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

package netdevice_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetDeviceProvider", func() {
	Describe("getting new instance of netDeviceProvider", func() {
		Context("with correct arguments", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			It("should return valid instance of the provider", func() {
				Expect(p).ToNot(BeNil())
			})
		})
	})
	Describe("getting devices", func() {
		Context("when there are none", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			config := &types.ResourceConfig{}
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config, 0)
			It("should return empty slice", func() {
				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})
		})
		Context("with an invalid selector index", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			config := &types.ResourceConfig{}
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config, 0)
			It("should return empty slice when SelectorObjs is nil", func() {
				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})

			It("should return empty slice when selector index is out of range", func() {
				netDeviceSelector := types.NetDeviceSelectors{}
				config = &types.ResourceConfig{SelectorObjs: []interface{}{&netDeviceSelector}}
				devs = p.GetDevices(config, 1)

				Expect(dDevs).To(BeEmpty())
				Expect(devs).To(BeEmpty())
			})
		})
	})
	Describe("adding 3 target devices", func() {
		Context("when 2 are valid devices, but 1 is a PF with SRIOV configured and 1 is invalid", func() {
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

			rf := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", true, false)
			p := netdevice.NewNetDeviceProvider(rf)
			config := &types.ResourceConfig{
				DeviceType: types.NetDeviceType,
				SelectorObjs: []interface{}{types.NetDeviceSelectors{
					DeviceSelectors: types.DeviceSelectors{},
				},
				}}
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
				Class:   &pcidb.Class{ID: "completely unparsable"},
				Vendor:  &pcidb.Vendor{},
				Product: &pcidb.Product{},
			}

			devsToAdd := []*ghw.PCIDevice{dev1, dev2, devInvalid}

			err := p.AddTargetDevices(devsToAdd, 0x1024)
			dDevs := p.GetDiscoveredDevices()
			devs := p.GetDevices(config, 0)
			It("shouldn't return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return only 1 device on GetDiscoveredDevices()", func() {
				Expect(dDevs).To(HaveLen(1))
			})
			It("should return only 1 device on GetDevices()", func() {
				Expect(devs).To(HaveLen(1))
			})
		})
	})
	Describe("getting Filtered devices", func() {
		Context("using selectors", func() {
			It("should correctly filter devices", func() {
				rf := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", false, false)
				p := netdevice.NewNetDeviceProvider(rf)
				all := make([]types.HostDevice, 5)
				mocked := make([]mocks.PciNetDevice, 5)

				ve := []string{"8086", "8086", "1111", "2222", "3333"}
				de := []string{"abcd", "123a", "abcd", "2222", "1024"}
				md := []string{"igb_uio", "igb_uio", "igb_uio", "iavf", "vfio-pci"}
				pa := []string{"0000:03:02.0", "0000:03:02.1", "0000:03:02.2", "0000:03:02.3", "0000:03:02.4"}
				pf := []string{"eth0", "eth0", "eth1", "net0", "net0"}
				ro := []string{"0000:86:00.0", "0000:86:00.1", "0000:86:00.2", "0000:86:00.3", "0000:86:00.4"}
				lt := []string{"ether", "infiniband", "ether", "ether", "fake"}
				dd := []string{"E710 PPPoE and PPPoL2TPv2", "fake", "fake", "gtp", "profile"}
				rd := []bool{false, true, false, false, true}
				vd := []string{"vhost", "vhost", "", "", "virtio"}

				rdmaYes := &mocks.RdmaSpec{}
				rdmaYes.On("IsRdma").Return(true)
				rdmaNo := &mocks.RdmaSpec{}
				rdmaNo.On("IsRdma").Return(false)

				vdpaVhost := &mocks.VdpaDevice{}
				vdpaVhost.On("GetDriver").Return("vhost_vdpa").
					On("GetType").Return(types.VdpaVhostType)
				vdpaVirtio := &mocks.VdpaDevice{}
				vdpaVirtio.On("GetDriver").Return("virtio_vdpa").
					On("GetType").Return(types.VdpaVirtioType)

				for i := range mocked {
					mocked[i].
						On("GetVendor").Return(ve[i]).
						On("GetDeviceCode").Return(de[i]).
						On("GetDriver").Return(md[i]).
						On("GetPciAddr").Return(pa[i]).
						On("GetPfNetName").Return(pf[i]).
						On("GetPfPciAddr").Return(ro[i]).
						On("GetLinkType").Return(lt[i]).
						On("GetDDPProfiles").Return(dd[i]).
						On("GetFuncID").Return(-1).
						On("IsRdma").Return(rd[i])

					switch vd[i] {
					case "vhost":
						mocked[i].On("GetVdpaDevice").Return(vdpaVhost)
					case "virtio":
						mocked[i].On("GetVdpaDevice").Return(vdpaVirtio)
					default:
						mocked[i].On("GetVdpaDevice").Return(nil)
					}

					all[i] = &mocked[i]
				}

				testCases := []struct {
					name     string
					sel      *types.NetDeviceSelectors
					expected []types.HostDevice
				}{
					{"vendors", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Vendors: []string{"8086"}}}, []types.HostDevice{all[0], all[1]}},
					{"devices", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Devices: []string{"abcd"}}}, []types.HostDevice{all[0], all[2]}},
					{"drivers", &types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Drivers: []string{"igb_uio"}}}, []types.HostDevice{all[0], all[1], all[2]}},
					{"pciAddresses", &types.NetDeviceSelectors{GenericPciDeviceSelectors: types.GenericPciDeviceSelectors{PciAddresses: []string{"0000:03:02.0", "0000:03:02.3"}}}, []types.HostDevice{all[0], all[3]}},
					{"pfNames", &types.NetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{PfNames: []string{"net0", "eth1"}}}, []types.HostDevice{all[2], all[3], all[4]}},
					{"rootDevices", &types.NetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{RootDevices: []string{"0000:86:00.0", "0000:86:00.4"}}}, []types.HostDevice{all[0], all[4]}},
					{"linkTypes", &types.NetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{LinkTypes: []string{"infiniband"}}}, []types.HostDevice{all[1]}},
					{"linkTypes multi", &types.NetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{LinkTypes: []string{"infiniband", "fake"}}}, []types.HostDevice{all[1], all[4]}},
					{"ddpProfiles", &types.NetDeviceSelectors{DDPProfiles: []string{"E710 PPPoE and PPPoL2TPv2"}}, []types.HostDevice{all[0]}},
					{"rdma", &types.NetDeviceSelectors{GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{IsRdma: true}}, []types.HostDevice{all[1], all[4]}},
					{"vdpa-vhost", &types.NetDeviceSelectors{VdpaType: "vhost"}, []types.HostDevice{all[0], all[1]}},
					{"vdpa-virtio", &types.NetDeviceSelectors{VdpaType: "virtio"}, []types.HostDevice{all[4]}},
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
				selectors := []*types.NetDeviceSelectors{
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
				rf := factory.NewResourceFactory("fake", "fake", "/var/lib/kubelet", false, false)
				p := netdevice.NewNetDeviceProvider(rf)
				devs := make([]types.HostDevice, 0)

				rc := &types.ResourceConfig{}

				_, err := p.GetFilteredDevices(devs, rc, 0)

				Expect(err).To(HaveOccurred())

				rc = &types.ResourceConfig{
					SelectorObjs: []interface{}{
						&types.NetDeviceSelectors{},
					},
				}
				_, err = p.GetFilteredDevices(devs, rc, 1)

				Expect(err).To(HaveOccurred())
			})
		})
	})
})
