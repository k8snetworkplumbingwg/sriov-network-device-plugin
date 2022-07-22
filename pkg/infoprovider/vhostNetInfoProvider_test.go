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
)

var _ = Describe("vdpaInfoProvider", func() {
	Describe("creating new vdpaInfoProvider", func() {
		It("should return valid vdpaInfoProvider object", func() {
			dip := infoprovider.NewVhostNetInfoProvider()
			Expect(dip).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(dip)).To(Equal(reflect.TypeOf(&vdpaInfoProvider{})))
		})
	})
	Describe("GetDeviceSpecs", func() {
		It("should return correct specs for vhost net device", func() {
			dip := infoprovider.NewVhostNetInfoProvider()
			Expect(dip.GetDeviceSpecs()).To(Equal([]*pluginapi.DeviceSpec{
				{
					HostPath:      "/dev/vhost-net",
					ContainerPath: "/dev/vhost-net",
					Permissions:   "mrw",
				},
				{
					HostPath:      "/dev/net/tun",
					ContainerPath: "/dev/net/tun",
					Permissions:   "mrw",
				},
			}))
		})
	})
	Describe("GetEnvVal", func() {
		It("should always return an empty string", func() {
			dip := infoprovider.NewVhostNetInfoProvider()
			Expect(dip.GetEnvVal()).To(BeEmpty())
		})
	})
	Describe("GetMounts", func() {
		It("should always return an empty array", func() {
			dip := infoprovider.NewVhostNetInfoProvider()
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
})
