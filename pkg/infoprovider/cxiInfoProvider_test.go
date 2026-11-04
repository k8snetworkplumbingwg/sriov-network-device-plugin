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

package infoprovider_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
)

var _ = Describe("cxiInfoProvider", func() {
	Describe("creating new cxiInfoProvider", func() {
		It("should return valid cxiInfoProvider object", func() {
			dip := infoprovider.NewCxiInfoProvider(&mocks.CxiSpec{})
			Expect(dip).NotTo(BeNil())
			Expect(dip.GetName()).To(Equal("cxi"))
		})
	})
	Describe("GetDeviceSpecs", func() {
		It("should return nil for non-cxi device", func() {
			cxi := &mocks.CxiSpec{}
			cxi.On("IsCxi").Return(false)
			dip := infoprovider.NewCxiInfoProvider(cxi)
			Expect(dip.GetDeviceSpecs()).To(BeNil())
		})
		It("should return the cxi char device spec", func() {
			cxi := &mocks.CxiSpec{}
			cxiSpecs := []*pluginapi.DeviceSpec{
				{HostPath: "/dev/cxi4", ContainerPath: "/dev/cxi4", Permissions: "rw"},
			}
			cxi.On("IsCxi").Return(true).
				On("GetCxiDeviceSpec").Return(cxiSpecs)
			dip := infoprovider.NewCxiInfoProvider(cxi)
			Expect(dip.GetDeviceSpecs()).To(Equal(cxiSpecs))
		})
	})
	Describe("getting mounts", func() {
		It("should always return empty array of mounts", func() {
			dip := infoprovider.NewCxiInfoProvider(&mocks.CxiSpec{})
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
	Describe("getting env val", func() {
		It("should return the cxi device mount", func() {
			cxi := &mocks.CxiSpec{}
			cxi.On("GetCxiDeviceSpec").Return([]*pluginapi.DeviceSpec{
				{HostPath: "/dev/cxi4", ContainerPath: "/dev/cxi4", Permissions: "rw"},
			})
			dip := infoprovider.NewCxiInfoProvider(cxi)
			envs := dip.GetEnvVal()
			Expect(envs).To(HaveKeyWithValue("dev-mount", "/dev/cxi4"))
		})
		It("should return empty env when no cxi device is found", func() {
			cxi := &mocks.CxiSpec{}
			cxi.On("GetCxiDeviceSpec").Return([]*pluginapi.DeviceSpec{})
			dip := infoprovider.NewCxiInfoProvider(cxi)
			Expect(dip.GetEnvVal()).To(BeEmpty())
		})
	})
})
