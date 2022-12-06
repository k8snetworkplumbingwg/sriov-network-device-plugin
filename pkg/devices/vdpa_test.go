/*
 * SPDX-FileCopyrightText: Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package devices_test

import (
	"fmt"

	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
)

type fakeKvdpaDevice struct {
	driver string
}

func (v *fakeKvdpaDevice) Driver() string {
	return v.driver
}
func (v *fakeKvdpaDevice) Name() string {
	return ""
}
func (v *fakeKvdpaDevice) MgmtDev() kvdpa.MgmtDev {
	return nil
}
func (v *fakeKvdpaDevice) VirtioNet() kvdpa.VirtioNet {
	return nil
}
func (v *fakeKvdpaDevice) VhostVdpa() kvdpa.VhostVdpa {
	return nil
}
func (v *fakeKvdpaDevice) ParentDevicePath() (string, error) {
	return "", nil
}

var _ = Describe("VdpaDevice", func() {
	t := GinkgoT()
	Context("getting new device", func() {
		It("no valid vdpa device for pci address", func() {
			fakeVdpaProvider := mocks.VdpaProvider{}
			fakeVdpaProvider.On("GetVdpaDeviceByPci", "0000:00:00.0").Return(nil, fmt.Errorf("ERROR"))
			utils.SetVdpaProviderInst(&fakeVdpaProvider)
			dev := devices.GetVdpaDevice("0000:00:00.0")

			Expect(dev).To(BeNil())
			fakeVdpaProvider.AssertExpectations(t)
		})
		It("unsupported vdpa type", func() {
			fakeKvdpaDev := &fakeKvdpaDevice{driver: "not supported"}
			fakeVdpaProvider := mocks.VdpaProvider{}
			fakeVdpaProvider.On("GetVdpaDeviceByPci", "0000:00:00.0").Return(fakeKvdpaDev, nil)
			utils.SetVdpaProviderInst(&fakeVdpaProvider)
			dev := devices.GetVdpaDevice("0000:00:00.0")

			Expect(dev).NotTo(BeNil())
			Expect(dev.GetType()).To(Equal(types.VdpaInvalidType))
			fakeVdpaProvider.AssertExpectations(t)
		})
		It("supported vdpa type", func() {
			fakeKvdpaDev := &fakeKvdpaDevice{driver: types.SupportedVdpaTypes[types.VdpaVirtioType]}
			fakeVdpaProvider := mocks.VdpaProvider{}
			fakeVdpaProvider.On("GetVdpaDeviceByPci", "0000:00:00.0").Return(fakeKvdpaDev, nil)
			utils.SetVdpaProviderInst(&fakeVdpaProvider)
			dev := devices.GetVdpaDevice("0000:00:00.0")

			Expect(dev).NotTo(BeNil())
			Expect(dev.GetType()).To(Equal(types.VdpaVirtioType))
			fakeVdpaProvider.AssertExpectations(t)
		})
	})
})
