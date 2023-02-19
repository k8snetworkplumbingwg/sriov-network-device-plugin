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

package infoprovider_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
)

var _ = Describe("rdmaInfoProvider", func() {
	Describe("creating new rdmaInfoProvider", func() {
		It("should return valid rdmaInfoProvider object", func() {
			rdma := &mocks.RdmaSpec{}
			dip := infoprovider.NewRdmaInfoProvider(rdma)
			Expect(dip).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(dip)).To(Equal(reflect.TypeOf(&rdmaInfoProvider{})))
		})
	})
	Describe("GetDeviceSpecs", func() {
		It("should return an empty map for non-rdma device", func() {
			rdma := &mocks.RdmaSpec{}
			rdma.On("IsRdma").Return(false)
			dip := infoprovider.NewRdmaInfoProvider(rdma)
			Expect(dip.GetDeviceSpecs()).To(BeNil())
		})
		It("should return non empty map for rdma device", func() {
			rdma := &mocks.RdmaSpec{}
			rdmaSpecs := []*pluginapi.DeviceSpec{
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			rdma.On("IsRdma").Return(true).
				On("GetRdmaDeviceSpec").Return(rdmaSpecs)

			dip := infoprovider.NewRdmaInfoProvider(rdma)
			Expect(dip.GetDeviceSpecs()).To(HaveLen(2))
		})
	})
	Describe("GetEnvVal", func() {
		It("should the rdma mounts from deviceSpecs", func() {
			rdma := &mocks.RdmaSpec{}
			rdma.On("IsRdma").Return(true).
				On("GetRdmaDeviceSpec").Return([]*pluginapi.DeviceSpec{
				{ContainerPath: "/dev/infiniband/issm4"},
				{ContainerPath: "/dev/infiniband/umad4"},
				{ContainerPath: "/dev/infiniband/uverbs4"},
				{ContainerPath: "/dev/infiniband/rdma_cm"}})
			dip := infoprovider.NewRdmaInfoProvider(rdma)
			dip.GetDeviceSpecs()
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(4))
			mount, exist := envs["rdma_cm"]
			Expect(exist).To(BeTrue())
			Expect(mount).To(Equal("/dev/infiniband/rdma_cm"))
			mount, exist = envs["uverbs"]
			Expect(exist).To(BeTrue())
			Expect(mount).To(Equal("/dev/infiniband/uverbs4"))
			mount, exist = envs["umad"]
			Expect(exist).To(BeTrue())
			Expect(mount).To(Equal("/dev/infiniband/umad4"))
			mount, exist = envs["issm"]
			Expect(exist).To(BeTrue())
			Expect(mount).To(Equal("/dev/infiniband/issm4"))
		})
	})
	Describe("GetMounts", func() {
		It("should always return an empty array", func() {
			rdma := &mocks.RdmaSpec{}
			dip := infoprovider.NewRdmaInfoProvider(rdma)
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
})
