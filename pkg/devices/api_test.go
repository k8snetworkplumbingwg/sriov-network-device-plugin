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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
)

var _ = Describe("ApiDevice", func() {
	t := GinkgoT()
	Context("Create new ApiDevice", func() {
		It("with populated fields", func() {
			mockInfo1 := &mocks.DeviceInfoProvider{}
			mockSpec1 := []*v1beta1.DeviceSpec{
				{HostPath: "/mock/spec/1"},
			}
			mockEnv := types.AdditionalInfo{"deviceID": "0000:00:00.1"}
			mockInfo1.On("GetName").Return("generic")
			mockInfo1.On("GetEnvVal").Return(mockEnv)
			mockInfo1.On("GetDeviceSpecs").Return(mockSpec1)
			mockInfo1.On("GetMounts").Return(nil)
			mockInfo2 := &mocks.DeviceInfoProvider{}
			mockSpec2 := []*v1beta1.DeviceSpec{
				{HostPath: "/mock/spec/2"},
			}
			mockInfo2.On("GetName").Return("generic")
			mockInfo2.On("GetEnvVal").Return(mockEnv)
			mockInfo2.On("GetDeviceSpecs").Return(mockSpec2)
			mockInfo2.On("GetMounts").Return(nil)

			infoProviders := []types.DeviceInfoProvider{mockInfo1, mockInfo2}
			dev := devices.NewAPIDeviceImpl("0000:00:00.1", infoProviders, -1)

			envs := dev.GetEnvVal()
			Expect(len(envs)).To(Equal(1))
			_, exist := envs["generic"]
			Expect(exist).To(BeTrue())
			pci, exist := envs["generic"]["deviceID"]
			Expect(exist).To(BeTrue())
			Expect(pci).To(Equal("0000:00:00.1"))

			Expect(dev.GetDeviceSpecs()).To(HaveLen(2))
			Expect(dev.GetMounts()).To(HaveLen(0))
			Expect(dev.GetAPIDevice()).NotTo(BeNil())
			Expect(dev.GetAPIDevice().ID).To(Equal("0000:00:00.1"))
			Expect(dev.GetAPIDevice().Topology).To(BeNil())
			mockInfo1.AssertExpectations(t)
			mockInfo2.AssertExpectations(t)
		})
		It("with populated API device topology", func() {
			infoProviders := []types.DeviceInfoProvider{}
			dev := devices.NewAPIDeviceImpl("0000:00:00.1", infoProviders, 0)

			Expect(dev.GetAPIDevice()).NotTo(BeNil())
			Expect(dev.GetAPIDevice().Topology).NotTo(BeNil())
			Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
		})
	})
})
