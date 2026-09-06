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

package cdi_test

import (
	"strings"
	"testing"

	cdilib "github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation"

	cdiPkg "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/cdi"
)

func TestCdi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CDI Suite")
}

var _ = Describe("Cdi", func() {
	Context("successfully create container annotation", func() {
		It("should return container annotation", func() {
			deviceId := "0000:00:00.1"
			cdi := cdiPkg.New()
			annotations, err := cdi.CreateContainerAnnotations([]string{deviceId}, "example.com", "net", "pool")
			Expect(err).NotTo(HaveOccurred())
			Expect(annotations).To(HaveLen(1))
			_, devices, err := cdilib.ParseAnnotations(annotations)
			Expect(err).NotTo(HaveOccurred())
			annoVal := "example.com/net=0000:00:00.1"
			Expect(devices).To(Equal([]string{annoVal}))
		})
		It("supports long resource names with stable, valid annotation keys", func() {
			cdi := cdiPkg.New()
			prefix := strings.Repeat("a", 63) + ".example.com"
			name := strings.Repeat("b", 63)
			first, err := cdi.CreateContainerAnnotations([]string{"0000:00:00.1"}, prefix, "net-pci", name)
			Expect(err).NotTo(HaveOccurred())
			second, err := cdi.CreateContainerAnnotations([]string{"0000:00:00.2"}, prefix, "net-pci", name)
			Expect(err).NotTo(HaveOccurred())
			Expect(first).To(HaveLen(1))
			Expect(second).To(HaveLen(1))
			for key := range first {
				Expect(validation.IsQualifiedName(key)).To(BeEmpty())
				Expect(second).To(HaveKey(key))
			}
		})
	})
})
