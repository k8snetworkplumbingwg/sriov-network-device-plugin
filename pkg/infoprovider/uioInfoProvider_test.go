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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

var _ = Describe("uioInfoProvider", func() {
	Describe("creating new uioInfoProvider", func() {
		It("should return valid uioInfoProvider object", func() {
			dip := infoprovider.NewUioInfoProvider("fakePCIAddr")
			Expect(dip).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(dip)).To(Equal(reflect.TypeOf(&uioInfoProvider{})))
		})
	})
	DescribeTable("getting device specs",
		func(fs *utils.FakeFilesystem, pciAddr string, expected []*pluginapi.DeviceSpec) {
			defer fs.Use()()

			dip := infoprovider.NewUioInfoProvider(pciAddr)
			specs := dip.GetDeviceSpecs()
			Expect(specs).To(ConsistOf(expected))
		},
		Entry("empty", &utils.FakeFilesystem{}, "", []*pluginapi.DeviceSpec{}),
		Entry("PCI address passed, returns DeviceSpec with paths to its UIO devices",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:02:00.0/uio/uio0",
				},
			},
			"0000:02:00.0",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/uio0", ContainerPath: "/dev/uio0", Permissions: "rw"},
			},
		),
	)
	Describe("getting mounts", func() {
		It("should always return empty array of mounts", func() {
			dip := infoprovider.NewUioInfoProvider("fakePCIAddr")
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
	Describe("getting env val", func() {
		It("should return passed PCI address and mounts for device", func() {
			pciAddr := "0000:02:00.0"
			fs := utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:02:00.0/uio/uio0",
				},
			}
			defer fs.Use()()
			dip := infoprovider.NewUioInfoProvider(pciAddr)
			dip.GetDeviceSpecs()
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(1))
			mount, exist := envs["mount"]
			Expect(exist).To(BeTrue())
			Expect(mount).To(Equal("/dev/uio0"))
		})
	})
})
