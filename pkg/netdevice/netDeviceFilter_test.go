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
	"github.com/intel/sriov-network-device-plugin/pkg/factory"
	"github.com/intel/sriov-network-device-plugin/pkg/netdevice"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("netdevice filtering", func() {
	Context("using selectors", func() {
		It("should correctly filter devices", func() {
			rf := factory.NewResourceFactory("fake", "fake", false)
			all := make([]types.PciDevice, 5)
			mocked := make([]mocks.PciNetDevice, 5)

			ve := []string{"8086", "8086", "1111", "2222", "3333"}
			de := []string{"abcd", "123a", "abcd", "2222", "1024"}
			md := []string{"igb_uio", "igb_uio", "igb_uio", "iavf", "vfio-pci"}
			pf := []string{"eth0", "eth0", "eth1", "net0", "net0"}
			lt := []string{"ether", "infiniband", "ether", "ether", "fake"}
			dd := []string{"E710 PPPoE and PPPoL2TPv2", "fake", "fake", "gtp", "profile"}
			rd := []bool{false, true, false, false, true}

			rdmaYes := &mocks.RdmaSpec{}
			rdmaYes.On("IsRdma").Return(true)
			rdmaNo := &mocks.RdmaSpec{}
			rdmaNo.On("IsRdma").Return(false)

			for i, _ := range mocked {
				mocked[i].
					On("GetVendor").Return(ve[i]).
					On("GetDeviceCode").Return(de[i]).
					On("GetDriver").Return(md[i]).
					On("GetPFName").Return(pf[i]).
					On("GetLinkType").Return(lt[i]).
					On("GetDDPProfiles").Return(dd[i])

				if rd[i] {
					mocked[i].On("GetRdmaSpec").Return(rdmaYes)
				} else {
					mocked[i].On("GetRdmaSpec").Return(rdmaNo)
				}

				all[i] = &mocked[i]
			}

			testCases := []struct {
				name     string
				sel      types.NetDeviceSelectors
				rdma     bool
				expected []types.PciDevice
			}{
				{"vendors", types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Vendors: []string{"8086"}}}, false, []types.PciDevice{all[0], all[1]}},
				{"devices", types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Devices: []string{"abcd"}}}, false, []types.PciDevice{all[0], all[2]}},
				{"drivers", types.NetDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Drivers: []string{"igb_uio"}}}, false, []types.PciDevice{all[0], all[1], all[2]}},
				{"pfNames", types.NetDeviceSelectors{PfNames: []string{"net0", "eth1"}}, false, []types.PciDevice{all[2], all[3], all[4]}},
				{"linkTypes", types.NetDeviceSelectors{LinkTypes: []string{"infiniband"}}, false, []types.PciDevice{all[1]}},
				{"linkTypes multi", types.NetDeviceSelectors{LinkTypes: []string{"infiniband", "fake"}}, false, []types.PciDevice{all[1], all[4]}},
				{"ddpProfiles", types.NetDeviceSelectors{DDPProfiles: []string{"E710 PPPoE and PPPoL2TPv2"}}, false, []types.PciDevice{all[0]}},
				{"rdma", types.NetDeviceSelectors{}, true, []types.PciDevice{all[1], all[4]}},
			}

			for _, tc := range testCases {
				By(tc.name)
				filter := netdevice.NewNetDeviceFilter(&tc.sel, rf, tc.rdma)
				actual := filter.GetFilteredDevices(all)
				Expect(actual).To(HaveLen(len(tc.expected)))
				Expect(actual).To(ConsistOf(tc.expected))
			}
		})
	})
})
