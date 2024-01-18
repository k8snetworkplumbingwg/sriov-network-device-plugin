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

var _ = Describe("vfioInfoProvider", func() {
	Describe("creating new vfioInfoProvider", func() {
		It("should return valid vfioInfoProvider object", func() {
			dip := infoprovider.NewVfioInfoProvider("fakePCIAddr")
			Expect(dip).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(dip)).To(Equal(reflect.TypeOf(&vfioInfoProvider{})))
		})
	})
	DescribeTable("GetDeviceSpecs",
		func(fs *utils.FakeFilesystem, pciAddr string, expected []*pluginapi.DeviceSpec) {
			defer fs.Use()()

			dip := infoprovider.NewVfioInfoProvider(pciAddr)
			specs := dip.GetDeviceSpecs()
			Expect(specs).To(ConsistOf(expected))
		},
		Entry("empty and returning default common vfio device file only",
			&utils.FakeFilesystem{},
			"",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/vfio/vfio", ContainerPath: "/dev/vfio/vfio", Permissions: "rw"},
			},
		),
		Entry("PCI address passed, returns DeviceSpec with paths to its VFIO devices and additional default VFIO path",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:02:00.0", "sys/kernel/iommu_groups/0",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:02:00.0/iommu_group": "../../../../kernel/iommu_groups/0",
				},
			},
			"0000:02:00.0",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/vfio/0", ContainerPath: "/dev/vfio/0", Permissions: "rw"},
				{HostPath: "/dev/vfio/vfio", ContainerPath: "/dev/vfio/vfio", Permissions: "rw"},
			},
		),
	)
	Describe("getting mounts", func() {
		It("should always return empty array of mounts", func() {
			dip := infoprovider.NewVfioInfoProvider("fakeAddr")
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
	Describe("getting env val", func() {
		It("should return passed PCI address and vfio device mount", func() {
			pciAddr := "0000:02:00.0"
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:02:00.0", "sys/kernel/iommu_groups/0",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:02:00.0/iommu_group": "../../../../kernel/iommu_groups/0",
				},
			}
			defer fs.Use()()

			dip := infoprovider.NewVfioInfoProvider(pciAddr)
			dip.GetDeviceSpecs()
			envs := dip.GetEnvVal()
			Expect(len(envs)).To(Equal(2))
			devMount, exist := envs["dev-mount"]
			Expect(exist).To(BeTrue())
			Expect(devMount).To(Equal("/dev/vfio/0"))
			vfioMount, exist := envs["mount"]
			Expect(exist).To(BeTrue())
			Expect(vfioMount).To(Equal("/dev/vfio/vfio"))
		})
	})
})
