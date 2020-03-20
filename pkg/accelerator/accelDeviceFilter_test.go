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
	"github.com/intel/sriov-network-device-plugin/pkg/accelerator"
	"github.com/intel/sriov-network-device-plugin/pkg/factory"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("accelerator filtering", func() {
	Context("using selectors", func() {
		It("should correctly filter devices", func() {
			rf := factory.NewResourceFactory("fake", "fake", false)
			all := make([]types.PciDevice, 5)
			mocked := make([]mocks.AccelDevice, 5)

			ve := []string{"8086", "8086", "1111", "2222", "3333"}
			de := []string{"abcd", "123a", "abcd", "2222", "1024"}
			md := []string{"igb_uio", "igb_uio", "igb_uio", "iavf", "vfio-pci"}

			for i, _ := range mocked {
				mocked[i].
					On("GetVendor").Return(ve[i]).
					On("GetDeviceCode").Return(de[i]).
					On("GetDriver").Return(md[i])

				all[i] = &mocked[i]
			}

			testCases := []struct {
				name     string
				sel      types.AccelDeviceSelectors
				expected []types.PciDevice
			}{
				{"vendors", types.AccelDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Vendors: []string{"8086"}}}, []types.PciDevice{all[0], all[1]}},
				{"devices", types.AccelDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Devices: []string{"abcd"}}}, []types.PciDevice{all[0], all[2]}},
				{"drivers", types.AccelDeviceSelectors{DeviceSelectors: types.DeviceSelectors{Drivers: []string{"igb_uio"}}}, []types.PciDevice{all[0], all[1], all[2]}},
			}

			for _, tc := range testCases {
				By(tc.name)
				filter := accelerator.NewAccelDeviceFilter(&tc.sel, rf)
				actual := filter.GetFilteredDevices(all)
				Expect(actual).To(HaveLen(len(tc.expected)))
				Expect(actual).To(ConsistOf(tc.expected))
			}
		})
	})
})
