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
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/stretchr/testify/mock"
)

var _ = Describe("NetResourcePool", func() {
	Context("getting a new instance of the pool", func() {
		rf := factory.NewResourceFactory("fake", "fake", true)
		nadutils := rf.GetNadUtils()
		rc := &types.ResourceConfig{
			ResourceName:   "fake",
			ResourcePrefix: "fake",
			SelectorObj:    &types.NetDeviceSelectors{},
		}
		devs := map[string]*v1beta1.Device{}
		pcis := map[string]types.PciDevice{}

		rp := netdevice.NewNetResourcePool(nadutils, rc, devs, pcis)

		It("should return a valid instance of the pool", func() {
			Expect(rp).ToNot(BeNil())
		})
	})
	Describe("getting DeviceSpecs", func() {
		Context("for non-RDMA devices", func() {
			rf := factory.NewResourceFactory("fake", "fake", true)
			nadutils := rf.GetNadUtils()
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObj: &types.NetDeviceSelectors{
					IsRdma: false,
				},
			}
			devs := map[string]*v1beta1.Device{}

			rdmaNo := &mocks.RdmaSpec{}
			rdmaNo.On("IsRdma").Return(false)

			// fake1 will have 2 device specs
			fake1 := &mocks.PciNetDevice{}
			fake1ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			fake1.On("GetDeviceSpecs").Return(fake1ds).On("GetRdmaSpec").Return(rdmaNo)

			// fake2 will have 1 device spec
			fake2 := &mocks.PciNetDevice{}
			fake2ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake2"},
			}
			fake2.On("GetDeviceSpecs").Return(fake2ds).
				On("GetRdmaSpec").Return(rdmaNo)

			// fake3 will have 0 device specs
			fake3 := &mocks.PciNetDevice{}
			fake3ds := []*pluginapi.DeviceSpec{}
			fake2.On("GetDeviceSpecs").Return(fake3ds).On("GetRdmaSpec").Return(rdmaNo)

			pcis := map[string]types.PciDevice{"fake1": fake1, "fake2": fake2, "fake3": fake3}

			rp := netdevice.NewNetResourcePool(nadutils, rc, devs, pcis)

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
		Context("for RDMA devices", func() {
			rf := factory.NewResourceFactory("fake", "fake", true)
			nadutils := rf.GetNadUtils()
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				SelectorObj: &types.NetDeviceSelectors{
					IsRdma: true,
				},
			}
			devs := map[string]*v1beta1.Device{}
			rdma1 := &mocks.RdmaSpec{}
			rdma2 := &mocks.RdmaSpec{}

			// fake1 will have 2 RDMA device specs and 0 regular device specs
			fake1 := &mocks.PciNetDevice{}
			fake1ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			rdma1.On("IsRdma").Return(true).On("GetRdmaDeviceSpec").Return(fake1ds)
			fake1.On("GetRdmaSpec").Return(rdma1).
				On("GetDeviceSpecs").Return(make([]*pluginapi.DeviceSpec, 0))

			// fake2 will have 0 rdma device specs to trigger error msg
			fake2 := &mocks.PciNetDevice{}
			fake2ds := []*pluginapi.DeviceSpec{}
			rdma2.On("IsRdma").Return(false).On("GetRdmaDeviceSpec").Return(fake2ds)
			fake2.On("GetDeviceSpecs").Return(make([]*pluginapi.DeviceSpec, 0)).
				On("GetRdmaSpec").Return(rdma2)

			pcis := map[string]types.PciDevice{"fake1": fake1, "fake2": fake2}

			rp := netdevice.NewNetResourcePool(nadutils, rc, devs, pcis)

			devIDs := []string{"fake1", "fake2"}

			actual := rp.GetDeviceSpecs(devIDs)

			It("should return valid slice of device specs", func() {
				Expect(actual).ToNot(BeNil())
				Expect(actual).To(HaveLen(2)) // fake1 => 2 rdma devices
				Expect(actual).To(ContainElement(fake1ds[0]))
				Expect(actual).To(ContainElement(fake1ds[1]))
			})
		})
	})
	Describe("Saving and Cleaning DevInfo files ", func() {
		t := GinkgoT()
		Context("for valid pci devices", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fakeResource",
				ResourcePrefix: "fakeOrg.io",
				SelectorObj: &types.NetDeviceSelectors{
					IsRdma: true,
				},
			}

			devs := map[string]*v1beta1.Device{}
			fake1 := &mocks.PciNetDevice{}
			fake1.On("GetPciAddr").Return("0000:01:00.1")
			fake2 := &mocks.PciNetDevice{}
			fake2.On("GetPciAddr").Return("0000:01:00.2")
			pcis := map[string]types.PciDevice{"fake1": fake1, "fake2": fake2}

			It("should call nadutils to create a well formatted DeviceInfo object", func() {
				nadutils := &mocks.NadUtils{}
				nadutils.On("SaveDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1", Anything).
					Return(func(rName, id string, devInfo *nettypes.DeviceInfo) error {
						if devInfo.Type != nettypes.DeviceInfoTypePCI || devInfo.Pci == nil || devInfo.Pci.PciAddress != "0000:01:00.1" {
							return fmt.Errorf("wrong device info")
						}
						return nil
					})
				nadutils.On("SaveDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2", Anything).
					Return(func(rName, id string, devInfo *nettypes.DeviceInfo) error {
						if devInfo.Type != nettypes.DeviceInfoTypePCI || devInfo.Pci == nil || devInfo.Pci.PciAddress != "0000:01:00.2" {
							return fmt.Errorf("wrong device info")
						}
						return nil
					})
				rp := netdevice.NewNetResourcePool(nadutils, rc, devs, pcis)
				err := rp.StoreDeviceInfoFile("fakeOrg.io")
				nadutils.AssertExpectations(t)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should call nadutils to clean the DeviceInfo objects", func() {
				nadutils := &mocks.NadUtils{}
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake1").Return(nil)
				nadutils.On("CleanDeviceInfoFile", "fakeOrg.io/fakeResource", "fake2").Return(nil)
				rp := netdevice.NewNetResourcePool(nadutils, rc, devs, pcis)
				err := rp.CleanDeviceInfoFile("fakeOrg.io")
				nadutils.AssertExpectations(t)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
