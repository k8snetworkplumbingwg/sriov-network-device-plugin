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
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
)

var _ = Describe("vdpaInfoProvider", func() {
	Describe("creating new vdpaInfoProvider", func() {
		It("should return valid vdpaInfoProvider object", func() {
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, nil)
			Expect(dip).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(dip)).To(Equal(reflect.TypeOf(&vdpaInfoProvider{})))
		})
	})
	Describe("GetDeviceSpecs", func() {
		It("should return nil if no vdpa device provided", func() {
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, nil)
			Expect(dip.GetDeviceSpecs()).To(BeNil())
		})
		It("should return nil if vdpa type is not supported", func() {
			vdpa := &mocks.VdpaDevice{}
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaInvalidType, vdpa)
			Expect(dip.GetDeviceSpecs()).To(BeNil())
		})
		It("should return nil if vdpa device has invalid type", func() {
			vdpa := &mocks.VdpaDevice{}
			vdpa.On("GetType").Return(types.VdpaInvalidType)
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, vdpa)
			Expect(dip.GetDeviceSpecs()).To(BeNil())
		})
		It("should return nil if vdpa device type doesn't match", func() {
			vdpa := &mocks.VdpaDevice{}
			vdpa.On("GetType").Return(types.VdpaVhostType)
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, vdpa)
			Expect(dip.GetDeviceSpecs()).To(BeNil())
		})
		It("should return empty map if device type is not a vdpa vhost", func() {
			vdpa := &mocks.VdpaDevice{}
			vdpa.On("GetType").Return(types.VdpaVirtioType)
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, vdpa)
			Expect(dip.GetDeviceSpecs()).To(HaveLen(0))
		})
		It("should return correct spec for vdpa vhost type device", func() {
			vdpa := &mocks.VdpaDevice{}
			vdpa.On("GetType").Return(types.VdpaVhostType).
				On("GetPath").Return("/dev/vhost-vdpa1", nil)
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVhostType, vdpa)
			Expect(dip.GetDeviceSpecs()).To(Equal([]*pluginapi.DeviceSpec{{
				HostPath:      "/dev/vhost-vdpa1",
				ContainerPath: "/dev/vhost-vdpa1",
				Permissions:   "rw",
			}}))
		})
	})
	Describe("GetEnvVal", func() {
		It("should return an empty list if there are no mounts", func() {
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, nil)
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(0))
		})
		It("should return object with the mounts", func() {
			vdpa := &mocks.VdpaDevice{}
			vdpa.On("GetType").Return(types.VdpaVhostType).
				On("GetPath").Return("/dev/vhost-vdpa1", nil)
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVhostType, vdpa)
			dip.GetDeviceSpecs()
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(1))
			mount, exist := envs["mount"]
			Expect(exist).To(BeTrue())
			Expect(mount).To(Equal("/dev/vhost-vdpa1"))
		})
	})
	Describe("GetMounts", func() {
		It("should always return an empty array", func() {
			dip := infoprovider.NewVdpaInfoProvider(types.VdpaVirtioType, nil)
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
})
