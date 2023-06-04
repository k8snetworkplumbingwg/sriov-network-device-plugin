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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
			annotations, err := cdi.CreateContainerAnnotations([]string{deviceId}, "example.com", "net")
			Expect(err).NotTo(HaveOccurred())
			Expect(len(annotations)).To(Equal(1))
			annoKey := "cdi.k8s.io/example.com_net"
			annoVal := "example.com/net=0000:00:00.1"
			Expect(annotations[annoKey]).To(Equal(annoVal))
		})
	})
})
