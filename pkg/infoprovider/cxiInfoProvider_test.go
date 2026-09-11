// Copyright 2018 Intel Corp. All Rights Reserved.
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

package infoprovider_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

var _ = Describe("cxiInfoProvider", func() {
	Describe("creating new cxiInfoProvider", func() {
		It("should return valid cxiInfoProvider object", func() {
			dip := infoprovider.NewCxiInfoProvider("fakePCIAddr")
			Expect(dip).NotTo(BeNil())
			Expect(dip.GetName()).To(Equal("cxi"))
		})
	})
	DescribeTable("GetDeviceSpecs",
		func(fs *utils.FakeFilesystem, pciAddr string, expected []*pluginapi.DeviceSpec) {
			defer fs.Use()()

			dip := infoprovider.NewCxiInfoProvider(pciAddr)
			specs := dip.GetDeviceSpecs()
			Expect(specs).To(ConsistOf(expected))
		},
		Entry("no cxi directory returns empty specs",
			&utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:21:00.1"},
			},
			"0000:21:00.1",
			[]*pluginapi.DeviceSpec{},
		),
		Entry("cxi device present returns its char device mount",
			&utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:21:00.1/cxi/cxi4"},
			},
			"0000:21:00.1",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/cxi4", ContainerPath: "/dev/cxi4", Permissions: "rw"},
			},
		),
	)
	Describe("getting mounts", func() {
		It("should always return empty array of mounts", func() {
			dip := infoprovider.NewCxiInfoProvider("fakeAddr")
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
	Describe("getting env val", func() {
		It("should return the cxi device mount", func() {
			pciAddr := "0000:21:00.1"
			fs := &utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:21:00.1/cxi/cxi4"},
			}
			defer fs.Use()()

			dip := infoprovider.NewCxiInfoProvider(pciAddr)
			envs := dip.GetEnvVal()
			Expect(envs).To(HaveKeyWithValue("dev-mount", "/dev/cxi4"))
		})
	})
})
