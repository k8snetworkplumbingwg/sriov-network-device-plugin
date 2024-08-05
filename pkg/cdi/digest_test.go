/*
 * SPDX-FileCopyrightText: Copyright (c) 2024 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/cdi"
)

var _ = Describe("Digest", func() {
	Context("successfully create digest", func() {
		It("should return digest", func() {
			spec := specs.Spec{
				Version: specs.CurrentVersion,
				Kind:    "nvidia.com/net-pci",
				Devices: []specs.Device{
					{
						ContainerEdits: specs.ContainerEdits{
							DeviceNodes: []*specs.DeviceNode{
								{
									Path:        "/dev/infiniband/issm0",
									HostPath:    "/dev/infiniband/issm0",
									Permissions: "rw",
								},
							},
						},
					},
				},
			}
			got := cdi.Digest(spec)
			Expect(got.String()).To(Equal("sha256:250959bdd6c927d5eb06860bdbbd974011c3c26f06980aef3c0621a7c027140d"))
		})
	})
})
