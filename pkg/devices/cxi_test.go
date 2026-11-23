// Copyright 2026 Hewlett Packard Enterprise. All Rights Reserved.
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

package devices_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/devices"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
)

var _ = Describe("CxiSpec", func() {
	DescribeTable("creating new CxiSpec",
		func(fs *utils.FakeFilesystem, isCxi bool, expected []*pluginapi.DeviceSpec) {
			defer fs.Use()()

			spec := devices.NewCxiSpec("0000:21:00.1")
			Expect(spec.IsCxi()).To(Equal(isCxi))
			Expect(spec.GetCxiDeviceSpec()).To(ConsistOf(expected))
		},
		Entry("no cxi directory",
			&utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:21:00.1"},
			},
			false,
			[]*pluginapi.DeviceSpec{},
		),
		Entry("cxi device present",
			&utils.FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:21:00.1/cxi/cxi4"},
			},
			true,
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/cxi4", ContainerPath: "/dev/cxi4", Permissions: "rw"},
			},
		),
		Entry("multiple cxi devices",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:21:00.1/cxi/cxi4",
					"sys/bus/pci/devices/0000:21:00.1/cxi/cxi5",
				},
			},
			false,
			[]*pluginapi.DeviceSpec{},
		),
	)
})
