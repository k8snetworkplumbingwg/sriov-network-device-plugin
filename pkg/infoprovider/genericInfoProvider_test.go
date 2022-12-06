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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/infoprovider"
)

var _ = Describe("genericInfoProvider", func() {
	Describe("creating new genericInfoProvider", func() {
		It("should return valid genericInfoProvider object", func() {
			dip := infoprovider.NewGenericInfoProvider("fakePCIAddr")
			Expect(dip).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(dip)).To(Equal(reflect.TypeOf(&genericInfoProvider{})))
		})
	})
	Describe("getting mounts", func() {
		It("should always return an empty array", func() {
			dip := infoprovider.NewGenericInfoProvider("fakePCIAddr")
			Expect(dip.GetMounts()).To(BeEmpty())
		})
	})
	Describe("getting device specs", func() {
		It("should always return an empty map", func() {
			dip := infoprovider.NewGenericInfoProvider("fakePCIAddr")
			Expect(dip.GetDeviceSpecs()).To(BeEmpty())
		})
	})
})
