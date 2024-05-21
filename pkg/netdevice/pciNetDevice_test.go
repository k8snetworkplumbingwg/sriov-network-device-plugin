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
	"path/filepath"
	"testing"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestNetdevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Netdevice Suite")
}

var _ = Describe("PciNetDevice", func() {
	Describe("creating new PciNetDevice", func() {
		t := GinkgoT()
		newPciDeviceFn := func(pciAddr string) *ghw.PCIDevice {
			return &ghw.PCIDevice{
				Address: pciAddr,
				Vendor:  &pcidb.Vendor{ID: ""},
				Product: &pcidb.Product{ID: ""},
			}
		}
		Context("successfully", func() {
			It("should populate fields", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/eth0",
						"sys/kernel/iommu_groups/0",
						"sys/bus/pci/drivers/vfio-pci",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
					Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0")},
				}
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn("0000:00:00.1")
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc, 0)

				Expect(dev.GetDriver()).To(Equal("vfio-pci"))
				Expect(dev.GetNetName()).To(Equal("eth0"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // /dev/vfio/vfio0 and default /dev/vfio/vfio
				envs := dev.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				vfioMap, exist := envs["vfio"]
				Expect(exist).To(BeTrue())
				Expect(len(vfioMap)).To(Equal(2))
				vfio, exist := envs["vfio"]["mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/vfio"))
				vfio, exist = envs["vfio"]["dev-mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/0"))
				genericMap, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				Expect(len(genericMap)).To(Equal(1))
				generic, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(generic).To(Equal("0000:00:00.1"))

				Expect(dev.IsRdma()).To(BeFalse())
				Expect(dev.GetLinkType()).To(Equal("fakeLinkType"))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("with two devices but only one of them being RDMA", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObjs: []interface{}{&types.NetDeviceSelectors{
					GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{
						IsRdma: true,
					},
				}},
			}
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/devices/0000:00:00.2/net/eth1",
					"sys/bus/pci/drivers/mlx5_core",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/mlx5_core",
				},
			}

			rdma1 := &mocks.RdmaSpec{}
			// fake1 will have 2 RDMA device specs
			fake1ds := []*pluginapi.DeviceSpec{
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			rdma1.On("IsRdma").Return(true).On("GetRdmaDeviceSpec").Return(fake1ds)

			rdma2 := &mocks.RdmaSpec{}
			rdma2.On("IsRdma").Return(false)

			f := &mocks.ResourceFactory{}

			pciAddr1 := "0000:00:00.1"
			mockInfo1 := &mocks.DeviceInfoProvider{}
			mockEnv1 := types.AdditionalInfo{"deviceID": pciAddr1}
			mockInfo1.On("GetName").Return("generic")
			mockInfo1.On("GetEnvVal").Return(mockEnv1)
			mockInfo1.On("GetDeviceSpecs").Return(nil)
			mockInfo1.On("GetMounts").Return(nil)
			pciAddr2 := "0000:00:00.2"
			mockInfo2 := &mocks.DeviceInfoProvider{}
			mockEnv2 := types.AdditionalInfo{"deviceID": pciAddr2}
			mockInfo2.On("GetName").Return("generic")
			mockInfo2.On("GetEnvVal").Return(mockEnv2)
			mockInfo2.On("GetDeviceSpecs").Return(nil)
			mockInfo2.On("GetMounts").Return(nil)
			f.On("GetDefaultInfoProvider", pciAddr1, "mlx5_core").Return([]types.DeviceInfoProvider{mockInfo1}).
				On("GetDefaultInfoProvider", pciAddr2, "mlx5_core").Return([]types.DeviceInfoProvider{mockInfo2}).
				On("GetRdmaSpec", types.NetDeviceType, pciAddr1).Return(rdma1).
				On("GetRdmaSpec", types.NetDeviceType, pciAddr2).Return(rdma2)

			in1 := newPciDeviceFn(pciAddr1)
			in2 := newPciDeviceFn(pciAddr2)

			It("should populate Rdma device specs if isRdma", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()
				dev, err := netdevice.NewPciNetDevice(in1, f, rc, 0)

				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				Expect(dev.IsRdma()).To(BeTrue())
				envs := dev.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				_, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["rdma"]
				Expect(exist).To(BeTrue())
				pci, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal(pciAddr1))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // 2x Rdma devs
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(err).NotTo(HaveOccurred())
				mockInfo1.AssertExpectations(t)
				rdma1.AssertExpectations(t)
			})
			It("but not otherwise", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()
				dev, err := netdevice.NewPciNetDevice(in2, f, rc, 0)

				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				envs := dev.GetEnvVal()
				Expect(len(envs)).To(Equal(1))
				_, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["rdma"]
				Expect(exist).To(BeFalse())
				pci, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal(pciAddr2))
				Expect(dev.IsRdma()).To(BeFalse())
				Expect(dev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(err).NotTo(HaveOccurred())
				mockInfo2.AssertExpectations(t)
				rdma2.AssertExpectations(t)
			})
			It("should use isRdma from the selector chosen by selector index", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()
				rc := &types.ResourceConfig{
					ResourceName:   "fake",
					ResourcePrefix: "fake",
					SelectorObjs: []interface{}{&types.NetDeviceSelectors{
						GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{
							IsRdma: true,
						},
					}, &types.NetDeviceSelectors{
						GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{
							IsRdma: false,
						},
					},
					}}

				// passing an RDMA capable device, but selector index chooses the IsRdma: false selector
				dev, err := netdevice.NewPciNetDevice(in1, f, rc, 1)

				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				envs := dev.GetEnvVal()
				Expect(envs).To(HaveLen(1))
				_, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["rdma"]
				Expect(exist).To(BeFalse())
				pci, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal(pciAddr1))
				Expect(dev.IsRdma()).To(BeFalse())
				Expect(dev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(err).NotTo(HaveOccurred())
				mockInfo1.AssertExpectations(t)
				rdma1.AssertExpectations(t)
			})
		})
		Context("With needsVhostNet", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObjs: []interface{}{&types.NetDeviceSelectors{},
					&types.NetDeviceSelectors{
						NeedVhostNet: true,
					},
				}}
			no_vhost_net_selector_index := 0
			vhost_net_selector_index := 1

			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/kernel/iommu_groups/0",
					"sys/bus/pci/drivers/vfio-pci",
					"dev/net",
				},
				Files: map[string][]byte{
					"/dev/vhost-net": nil,
					"/dev/net/tun":   nil,
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
					"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
				},
			}

			f := factory.NewResourceFactory("fake", "fake", true, false)
			in := newPciDeviceFn("0000:00:00.1")
			It("should add the vhost-net deviceSpec", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				//nolint: gocritic
				infoprovider.HostNet = filepath.Join(fs.RootDir, "/dev/vhost-net")
				//nolint: gocritic
				infoprovider.HostTun = filepath.Join(fs.RootDir, "/dev/net/tun")

				dev, err := netdevice.NewPciNetDevice(in, f, rc, vhost_net_selector_index)

				Expect(dev.GetDriver()).To(Equal("vfio-pci"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(4)) // /dev/vfio/0 and default /dev/vfio/vfio + vhost-net + tun
				envs := dev.GetEnvVal()
				Expect(len(envs)).To(Equal(3))
				_, exist := envs["vfio"]
				Expect(exist).To(BeTrue())
				vfio, exist := envs["vfio"]["mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/vfio"))
				vfio, exist = envs["vfio"]["dev-mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/0"))
				_, exist = envs["vhost"]
				Expect(exist).To(BeTrue())
				vhost, exist := envs["vhost"]["net-mount"]
				Expect(exist).To(BeTrue())
				Expect(vhost).To(Equal("/dev/vhost-net"))
				vhost, exist = envs["vhost"]["tun-mount"]
				Expect(exist).To(BeTrue())
				Expect(vhost).To(Equal("/dev/net/tun"))
				genericMap, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				Expect(len(genericMap)).To(Equal(1))
				generic, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(generic).To(Equal("0000:00:00.1"))

				Expect(dev.IsRdma()).To(BeFalse())
				Expect(err).NotTo(HaveOccurred())
			})
			It("but only if the selector index has the selector set", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				dev, err := netdevice.NewPciNetDevice(in, f, rc, no_vhost_net_selector_index)

				Expect(dev.GetDriver()).To(Equal("vfio-pci"))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // /dev/vfio/0 and default /dev/vfio/vfio + vhost-net + tun
				envs := dev.GetEnvVal()
				Expect(envs).To(HaveLen(2))
				_, exist := envs["vfio"]
				Expect(exist).To(BeTrue())
				vfio, exist := envs["vfio"]["mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/vfio"))
				vfio, exist = envs["vfio"]["dev-mount"]
				Expect(exist).To(BeTrue())
				Expect(vfio).To(Equal("/dev/vfio/0"))
				_, exist = envs["vhost"]
				Expect(exist).To(BeFalse())
				genericMap, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				Expect(genericMap).To(HaveLen(1))
				generic, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(generic).To(Equal("0000:00:00.1"))

				Expect(dev.IsRdma()).To(BeFalse())
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("device's PF name is not available", func() {
			It("device should be added", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{"sys/bus/pci/devices/0000:00:00.1"},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/iommu_group": "../../../../kernel/iommu_groups/0",
						"sys/bus/pci/devices/0000:00:00.1/driver":      "../../../../bus/pci/drivers/vfio-pci",
					},
				}
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				f := factory.NewResourceFactory("fake", "fake", true, false)
				in := newPciDeviceFn("0000:00:00.1")
				rc := &types.ResourceConfig{}

				dev, err := netdevice.NewPciNetDevice(in, f, rc, 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(dev).NotTo(BeNil())

				envs := dev.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
			})
		})
		Context("With Vdpa-capable devices", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1/",
					"sys/bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:00:00.2/",
					"sys/bus/pci/drivers/ifcvf",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/ifcvf",
				},
			}

			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObjs: []interface{}{&types.NetDeviceSelectors{
					VdpaType: "vhost",
				}, &types.NetDeviceSelectors{
					VdpaType: "virtio",
				},
				}}
			vhost_selector_index := 0
			virtio_selector_index := 1

			defaultInfo1 := &mocks.DeviceInfoProvider{}
			mockEnv1 := types.AdditionalInfo{"deviceID": "0000:00:00.1"}
			defaultInfo1.On("GetName").Return("generic")
			defaultInfo1.On("GetEnvVal").Return(mockEnv1)
			defaultInfo1.On("GetDeviceSpecs").Return(nil)
			defaultInfo1.On("GetMounts").Return(nil)
			defaultInfo2 := &mocks.DeviceInfoProvider{}
			mockEnv2 := types.AdditionalInfo{"deviceID": "0000:00:00.2"}
			defaultInfo2.On("GetName").Return("generic")
			defaultInfo2.On("GetEnvVal").Return(mockEnv2)
			defaultInfo2.On("GetDeviceSpecs").Return(nil)
			defaultInfo2.On("GetMounts").Return(nil)

			fakeVdpaVhost := &mocks.VdpaDevice{}
			fakeVdpaVhost.On("GetDriver").Return("vhost_vdpa").
				On("GetPath").Return("/dev/vhost-vdpa1", nil).
				On("GetType").Return(types.VdpaVhostType)

			fakeVdpaVirtio := &mocks.VdpaDevice{}
			fakeVdpaVirtio.On("GetDriver").Return("virtio_vdpa").
				On("GetPath").Return("/sys/bus/virtio/devices/virtio2", nil).
				On("GetType").Return(types.VdpaVirtioType)

			// 0000:00:00.1 -> vhost
			// 0000:00:00.2 -> virtio
			f := &mocks.ResourceFactory{}
			f.On("GetVdpaDevice", "0000:00:00.1").Return(fakeVdpaVhost).
				On("GetVdpaDevice", "0000:00:00.2").Return(fakeVdpaVirtio).
				On("GetDefaultInfoProvider", "0000:00:00.1", "mlx5_core").Return([]types.DeviceInfoProvider{defaultInfo1}).
				On("GetDefaultInfoProvider", "0000:00:00.2", "ifcvf").Return([]types.DeviceInfoProvider{defaultInfo2})

			in1 := newPciDeviceFn("0000:00:00.1")
			in2 := newPciDeviceFn("0000:00:00.2")

			It("should add the correct deviceSpec if the ResourceConfig type matches the vdpa driver", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				dev1, err1 := netdevice.NewPciNetDevice(in1, f, rc, vhost_selector_index)
				dev2, err2 := netdevice.NewPciNetDevice(in2, f, rc, virtio_selector_index)

				Expect(dev1.GetDriver()).To(Equal("mlx5_core"))
				envs := dev1.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				_, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["vdpa"]
				Expect(exist).To(BeTrue())
				pci, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal("0000:00:00.1"))
				Expect(dev1.GetDeviceSpecs()).To(Equal([]*pluginapi.DeviceSpec{
					{
						HostPath:      "/dev/vhost-vdpa1",
						ContainerPath: "/dev/vhost-vdpa1",
						Permissions:   "rw",
					}}))
				Expect(dev1.GetMounts()).To(HaveLen(0))
				Expect(dev1.GetVdpaDevice()).To(Equal(fakeVdpaVhost))
				Expect(err1).NotTo(HaveOccurred())

				Expect(dev2.GetDriver()).To(Equal("ifcvf"))
				envs = dev2.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				_, exist = envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["vdpa"]
				Expect(exist).To(BeTrue())
				pci, exist = envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal("0000:00:00.2"))
				Expect(dev2.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev2.GetMounts()).To(HaveLen(0))
				Expect(dev2.GetVdpaDevice()).To(Equal(fakeVdpaVirtio))
				Expect(err2).NotTo(HaveOccurred())

				f.AssertExpectations(t)
				defaultInfo1.AssertExpectations(t)
				defaultInfo2.AssertExpectations(t)
			})
			It("should generate empty deviceSpecs if ResourceConfig type does not match vdpa driver", func() {
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()

				dev1, err1 := netdevice.NewPciNetDevice(in1, f, rc, virtio_selector_index)
				dev2, err2 := netdevice.NewPciNetDevice(in2, f, rc, vhost_selector_index)

				envs := dev1.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				_, exist := envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["vdpa"]
				Expect(exist).To(BeTrue())
				pci, exist := envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal("0000:00:00.1"))
				Expect(dev1.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev1.GetMounts()).To(HaveLen(0))
				Expect(err1).NotTo(HaveOccurred())
				envs = dev2.GetEnvVal()
				Expect(len(envs)).To(Equal(2))
				_, exist = envs["generic"]
				Expect(exist).To(BeTrue())
				_, exist = envs["vdpa"]
				Expect(exist).To(BeTrue())
				pci, exist = envs["generic"]["deviceID"]
				Expect(exist).To(BeTrue())
				Expect(pci).To(Equal("0000:00:00.2"))
				Expect(dev2.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev2.GetMounts()).To(HaveLen(0))
				Expect(err2).NotTo(HaveOccurred())

				f.AssertExpectations(t)
				defaultInfo1.AssertExpectations(t)
				defaultInfo2.AssertExpectations(t)
			})
		})
	})
})
