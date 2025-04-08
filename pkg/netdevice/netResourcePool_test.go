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
	"fmt"

	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	utilsmocks "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/stretchr/testify/mock"
)

var _ = Describe("NetResourcePool", func() {
	Context("getting a new instance of the pool", func() {
		rf := factory.NewResourceFactory("fake", "fake", true, false)
		nadutils := rf.GetNadUtils()
		rc := &types.ResourceConfig{
			ResourceName:   "fake",
			ResourcePrefix: "fake",
			SelectorObjs:   []interface{}{&types.NetDeviceSelectors{}},
		}
		pcis := map[string]types.HostDevice{}

		rp := netdevice.NewNetResourcePool(nadutils, rc, pcis)

		It("should return a valid instance of the pool", func() {
			Expect(rp).ToNot(BeNil())
		})
	})
	Describe("getting DeviceSpecs", func() {
		Context("for multiple devices", func() {
			rf := factory.NewResourceFactory("fake", "fake", true, false)
			nadutils := rf.GetNadUtils()
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObjs: []interface{}{&types.NetDeviceSelectors{
					GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{
						IsRdma: false,
					},
				},
				}}

			// fake1 will have 2 device specs
			fake1 := &mocks.PciNetDevice{}
			fake1ds := []*pluginapi.DeviceSpec{
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			fake1.On("GetDeviceSpecs").Return(fake1ds)

			// fake2 will have 1 device spec
			fake2 := &mocks.PciNetDevice{}
			fake2ds := []*pluginapi.DeviceSpec{
				{ContainerPath: "/fake/path", HostPath: "/dev/fake2"},
			}
			fake2.On("GetDeviceSpecs").Return(fake2ds)

			// fake3 will have 0 device specs
			fake3 := &mocks.PciNetDevice{}
			fake3ds := []*pluginapi.DeviceSpec{}
			fake3.On("GetDeviceSpecs").Return(fake3ds)

			pcis := map[string]types.HostDevice{"fake1": fake1, "fake2": fake2, "fake3": fake3}

			rp := netdevice.NewNetResourcePool(nadutils, rc, pcis)

			devIDs := []string{"fake1", "fake2"}

			actual := rp.GetDeviceSpecs(devIDs)

			It("should return valid slice of device specs", func() {
				Expect(actual).ToNot(BeNil())
				Expect(actual).To(HaveLen(3)) // fake1 + fake2 => 3 devices
				Expect(actual).To(ContainElement(fake1ds[0]))
				Expect(actual).To(ContainElement(fake1ds[1]))
				Expect(actual).To(ContainElement(fake2ds[0]))
			})
		})
	})
	Describe("Saving and Cleaning DevInfo files ", func() {
		t := GinkgoT()
		Context("for valid pci devices", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fakeResource",
				ResourcePrefix: "fakeOrg.io",
				SelectorObjs: []interface{}{&types.NetDeviceSelectors{
					GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{
						IsRdma: true,
					},
				},
				}}

			fake1 := &mocks.PciNetDevice{}
			fake1.On("GetPciAddr").Return("0000:01:00.1").
				On("GetVdpaDevice").Return(nil).
				On("IsRdma").Return(true)
			fake2 := &mocks.PciNetDevice{}
			fake2.On("GetPciAddr").Return("0000:01:00.2").
				On("GetVdpaDevice").Return(nil).
				On("IsRdma").Return(false)
			pcis := map[string]types.HostDevice{"fake1": fake1, "fake2": fake2}

			fakeRdmaProvider := utilsmocks.RdmaProvider{}
			fakeRdmaProvider.On("GetRdmaDevicesForPcidev", "0000:01:00.1").Return([]string{"rdmadevice1"})
			utils.SetRdmaProviderInst(&fakeRdmaProvider)

			It("should call nadutils to create a well formatted DeviceInfo object", func() {
				nadutils := &mocks.NadUtils{}
				nadutils.On("SaveDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1", Anything).
					Return(func(rName, id string, devInfo *nettypes.DeviceInfo) error {
						if devInfo.Type != nettypes.DeviceInfoTypePCI || devInfo.Pci == nil || devInfo.Pci.PciAddress != "0000:01:00.1" {
							return fmt.Errorf("wrong device info")
						}
						if devInfo.Pci.RdmaDevice != "rdmadevice1" {
							return fmt.Errorf("wrong rdma device")
						}
						return nil
					})
				nadutils.On("SaveDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2", Anything).
					Return(func(rName, id string, devInfo *nettypes.DeviceInfo) error {
						if devInfo.Type != nettypes.DeviceInfoTypePCI || devInfo.Pci == nil || devInfo.Pci.PciAddress != "0000:01:00.2" {
							return fmt.Errorf("wrong device info")
						}
						if devInfo.Pci.RdmaDevice != "" {
							return fmt.Errorf("wrong rdma device")
						}
						return nil
					})
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1").Return(nil)
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2").Return(nil)
				rp := netdevice.NewNetResourcePool(nadutils, rc, pcis)
				err := rp.StoreDeviceInfoFile("fakeOrg.io", []string{"fake1", "fake2"})
				nadutils.AssertExpectations(t)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should call nadutils to clean the DeviceInfo objects", func() {
				nadutils := &mocks.NadUtils{}
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1").Return(nil)
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2").Return(nil)
				rp := netdevice.NewNetResourcePool(nadutils, rc, pcis)
				err := rp.CleanDeviceInfoFile("fakeOrg.io")
				nadutils.AssertExpectations(t)
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("for vdpa devices devices", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fakeResource",
				ResourcePrefix: "fakeOrg.io",
				SelectorObjs: []interface{}{&types.NetDeviceSelectors{
					VdpaType: "vhost",
				},
				}}

			fakeVdpa1 := &mocks.VdpaDevice{}
			fakeVdpa1.On("GetParent").Return("vdpa1").
				On("GetPath").Return("/dev/vhost-vdpa5", nil).
				On("GetType").Return(types.VdpaVhostType)

			fakeVdpa2 := &mocks.VdpaDevice{}
			fakeVdpa2.On("GetParent").Return("vdpa2").
				On("GetPath").Return("/dev/vhost-vdpa6", nil).
				On("GetType").Return(types.VdpaVhostType)

			fake1 := &mocks.PciNetDevice{}
			fake2 := &mocks.PciNetDevice{}
			fake1.On("GetPciAddr").Return("0000:01:00.1").
				On("GetVdpaDevice").Return(fakeVdpa1)
			fake2.On("GetPciAddr").Return("0000:01:00.2").
				On("GetVdpaDevice").Return(fakeVdpa2)

			pcis := map[string]types.HostDevice{"fake1": fake1, "fake2": fake2}

			It("should call nadutils to create a well formatted DeviceInfo object", func() {
				nadutils := &mocks.NadUtils{}
				nadutils.On("SaveDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1", Anything).
					Return(func(rName, id string, devInfo *nettypes.DeviceInfo) error {
						if devInfo.Type != nettypes.DeviceInfoTypeVDPA ||
							devInfo.Vdpa == nil ||
							devInfo.Vdpa.ParentDevice != "vdpa1" ||
							devInfo.Vdpa.Driver != "vhost" ||
							devInfo.Vdpa.Path != "/dev/vhost-vdpa5" {
							return fmt.Errorf("wrong device info %+v", devInfo)
						}
						return nil
					})
				nadutils.On("SaveDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2", Anything).
					Return(func(rName, id string, devInfo *nettypes.DeviceInfo) error {
						if devInfo.Type != nettypes.DeviceInfoTypeVDPA ||
							devInfo.Vdpa == nil ||
							devInfo.Vdpa.ParentDevice != "vdpa2" ||
							devInfo.Vdpa.Driver != "vhost" ||
							devInfo.Vdpa.Path != "/dev/vhost-vdpa6" {
							return fmt.Errorf("wrong device info %+v", devInfo)
						}
						return nil
					})
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1").Return(nil)
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2").Return(nil)

				rp := netdevice.NewNetResourcePool(nadutils, rc, pcis)
				err := rp.StoreDeviceInfoFile("fakeOrg.io", []string{"fake1", "fake2"})
				Expect(err).ToNot(HaveOccurred())
				nadutils.AssertExpectations(t)
			})
			It("should call nadutils to clean the DeviceInfo objects", func() {
				nadutils := &mocks.NadUtils{}
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1").Return(nil)
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2").Return(nil)
				rp := netdevice.NewNetResourcePool(nadutils, rc, pcis)
				err := rp.CleanDeviceInfoFile("fakeOrg.io")
				Expect(err).ToNot(HaveOccurred())
				nadutils.AssertExpectations(t)
			})
		})
	})
	Describe("Health Checking on Net Devices", func() {
		Context("PF link status and device existence for VFs with Same PF", func() {
			var fake1health bool
			var fake2health bool
			var fake1 *mocks.PciNetDevice
			var fake2 *mocks.PciNetDevice
			var rp types.ResourcePool

			BeforeEach(func() {
				rf := factory.NewResourceFactory("fake", "fake", true, false)
				nadutils := rf.GetNadUtils()
				rc := &types.ResourceConfig{
					ResourceName:             "fake",
					ResourcePrefix:           "fake",
					CheckHealthOnPf:          true,
					CheckHealthOnDeviceExist: true,
					SelectorObjs:             []interface{}{&types.NetDeviceSelectors{}},
				}

				fake1 = &mocks.PciNetDevice{}
				fake1.On("GetPfNetName").Return("fakepf1")
				fake1.On("GetPciAddr").Return("0000:01:00.1")
				fake1.On("SetHealth", Anything).Run(func(args Arguments) {
					fake1health = args.Bool(0)
				}).Return()
				fake1health = true

				fake2 = &mocks.PciNetDevice{}
				fake2.On("GetPfNetName").Return("fakepf1")
				fake2.On("GetPciAddr").Return("0000:01:00.2")
				fake2.On("SetHealth", Anything).Run(func(args Arguments) {
					fake2health = args.Bool(0)
				}).Return()
				fake2health = true

				pcis := map[string]types.HostDevice{"fake1": fake1, "fake2": fake2}

				rp = netdevice.NewNetResourcePool(nadutils, rc, pcis)
			})

			SetCurrentHealth := func(health bool) {
				fake1health = health
				fake2health = health
				fake1.On("GetHealth").Return(health).Once()
				fake2.On("GetHealth").Return(health).Once()
			}

			SetCurrentLinkState := func(up bool) {
				fake1.On("IsPfLinkUp").Return(up, nil).Once()
				fake2.On("IsPfLinkUp").Return(up, nil).Once()
			}

			SetCurrentDeviceExistance := func(exist bool) {
				fake1.On("DeviceExists").Return(exist, nil).Once()
				fake2.On("DeviceExists").Return(exist, nil).Once()
			}

			SetupHealthProbeTest := func(health, link, exist bool) {
				SetCurrentHealth(health)
				SetCurrentLinkState(link)
				SetCurrentDeviceExistance(exist)
			}

			It("Currently Device Healthy, PF Link Up, Device exists", func() {
				SetupHealthProbeTest(true, true, true)
				Expect(rp.Probe()).To(BeFalse())
				Expect(fake1health).To(BeTrue())
				Expect(fake2health).To(BeTrue())
			})
			It("Currently Device Healthy, PF Link Up, Device missing", func() {
				SetupHealthProbeTest(true, true, false)
				Expect(rp.Probe()).To(BeTrue())
				Expect(fake1health).To(BeFalse())
				Expect(fake2health).To(BeFalse())
			})
			It("Currently Device Healthy, PF Link Down, Device exists", func() {
				SetupHealthProbeTest(true, false, true)
				Expect(rp.Probe()).To(BeTrue())
				Expect(fake1health).To(BeFalse())
				Expect(fake2health).To(BeFalse())
			})
			It("Currently Device Healthy, PF Link Down, Device missing", func() {
				SetupHealthProbeTest(true, false, false)
				Expect(rp.Probe()).To(BeTrue())
				Expect(fake1health).To(BeFalse())
				Expect(fake2health).To(BeFalse())
			})
			It("Currently Device Unhealthy, PF Link Up, Device exists", func() {
				SetupHealthProbeTest(false, true, true)
				Expect(rp.Probe()).To(BeTrue())
				Expect(fake1health).To(BeTrue())
				Expect(fake2health).To(BeTrue())
			})
			It("Currently Device Unhealthy, PF Link Up, Device missing", func() {
				SetupHealthProbeTest(false, true, false)
				Expect(rp.Probe()).To(BeFalse())
				Expect(fake1health).To(BeFalse())
				Expect(fake2health).To(BeFalse())
			})
			It("Currently Device Unhealthy, PF Link Down, Device exists", func() {
				SetupHealthProbeTest(false, false, true)
				Expect(rp.Probe()).To(BeFalse())
				Expect(fake1health).To(BeFalse())
				Expect(fake2health).To(BeFalse())
			})
			It("Currently Device Unhealthy, PF Link Down, Device missing", func() {
				SetupHealthProbeTest(false, false, false)
				Expect(rp.Probe()).To(BeFalse())
				Expect(fake1health).To(BeFalse())
				Expect(fake2health).To(BeFalse())
			})
		})
	})
})
