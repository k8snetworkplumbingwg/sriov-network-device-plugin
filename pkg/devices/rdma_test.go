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
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
)

var _ = Describe("RdmaSpec", func() {
	Describe("creating new RdmaSpec", func() {
		t := GinkgoT()
		Context("successfully", func() {
			It("without device specs", func() {
				fakeRdmaProvider := mocks.RdmaProvider{}
				fakeRdmaProvider.On("GetRdmaDevicesForPcidev", "0000:00:00.0").Return([]string{})
				utils.SetRdmaProviderInst(&fakeRdmaProvider)
				spec := devices.NewRdmaSpec("0000:00:00.0")

				Expect(spec.IsRdma()).To(BeFalse())
				Expect(spec.GetRdmaDeviceSpec()).To(HaveLen(0))
				fakeRdmaProvider.AssertExpectations(t)
			})
			It("with device specs", func() {
				fakeRdmaProvider := mocks.RdmaProvider{}
				fakeRdmaProvider.On("GetRdmaDevicesForPcidev", "0000:00:00.0").
					Return([]string{"fake_0", "fake_1"})
				fakeRdmaProvider.On("GetRdmaCharDevices", "fake_0").Return([]string{
					"/dev/infiniband/issm0", "/dev/infiniband/umad0",
					"/dev/infiniband/uverbs0", "/dev/infiniband/rdma_cm",
				}).On("GetRdmaCharDevices", "fake_1").Return([]string{"/dev/infiniband/rdma_cm"})
				utils.SetRdmaProviderInst(&fakeRdmaProvider)
				spec := devices.NewRdmaSpec("0000:00:00.0")

				Expect(spec.IsRdma()).To(BeTrue())
				Expect(spec.GetRdmaDeviceSpec()).To(Equal([]*pluginapi.DeviceSpec{
					{
						ContainerPath: "/dev/infiniband/issm0",
						HostPath:      "/dev/infiniband/issm0",
						Permissions:   "rw",
					},
					{
						ContainerPath: "/dev/infiniband/umad0",
						HostPath:      "/dev/infiniband/umad0",
						Permissions:   "rw",
					},
					{
						ContainerPath: "/dev/infiniband/uverbs0",
						HostPath:      "/dev/infiniband/uverbs0",
						Permissions:   "rw",
					},
					{
						ContainerPath: "/dev/infiniband/rdma_cm",
						HostPath:      "/dev/infiniband/rdma_cm",
						Permissions:   "rw",
					},
					{
						ContainerPath: "/dev/infiniband/rdma_cm",
						HostPath:      "/dev/infiniband/rdma_cm",
						Permissions:   "rw",
					},
				}))
				fakeRdmaProvider.AssertExpectations(t)
			})
		})
	})
})
