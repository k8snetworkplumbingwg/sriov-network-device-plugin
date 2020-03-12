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
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"
	"k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AccelResourcePool", func() {
	Context("getting a new instance of the pool", func() {
		rc := &types.ResourceConfig{ResourceName: "fake", ResourcePrefix: "fake"}
		devs := map[string]*v1beta1.Device{}
		pcis := map[string]types.PciDevice{}

		rp := accelerator.NewAccelResourcePool(rc, devs, pcis)

		It("should return a valid instance of the pool", func() {
			Expect(rp).ToNot(BeNil())
		})
	})
	Describe("getting DeviceSpecs", func() {
		Context("for accelerator devices", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				IsRdma:         false,
			}
			devs := map[string]*v1beta1.Device{}

			// fake1 will have 2 device specs
			fake1 := &mocks.AccelDevice{}
			fake1ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			fake1.On("GetDeviceSpecs").Return(fake1ds)

			// fake2 will have 1 device spec
			fake2 := &mocks.AccelDevice{}
			fake2ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake2"},
			}
			fake2.On("GetDeviceSpecs").Return(fake2ds)

			// fake3 will have 0 device specs
			fake3 := &mocks.AccelDevice{}
			fake3ds := []*pluginapi.DeviceSpec{}
			fake2.On("GetDeviceSpecs").Return(fake3ds)

			pcis := map[string]types.PciDevice{"fake1": fake1, "fake2": fake2, "fake3": fake3}

			rp := accelerator.NewAccelResourcePool(rc, devs, pcis)

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
})
