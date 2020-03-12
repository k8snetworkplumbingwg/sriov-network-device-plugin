// Copyright 2020 Intel Corp. All Rights Reserved.
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

package accelerator_test

import (
	"github.com/intel/sriov-network-device-plugin/pkg/accelerator"
	"github.com/intel/sriov-network-device-plugin/pkg/factory"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AcceleratorProvider", func() {
	Describe("getting new instance of acceleratorProvider", func() {
		Context("with correct arguments", func() {
			rf := &mocks.ResourceFactory{}
			p := accelerator.NewAccelDeviceProvider(rf)
			It("should return valid instance of the provider", func() {
				Expect(p).ToNot(BeNil())
			})
		})
	})
	Describe("getting devices", func() {
		Context("when there are none", func() {
			rf := &mocks.ResourceFactory{}
			p := accelerator.NewAccelDeviceProvider(rf)
			devs := p.GetDevices()
			It("should return empty slice", func() {
				Expect(devs).To(BeEmpty())
			})
		})
	})
	Describe("adding 4 target devices", func() {
		Context("when 2 are valid devices, but 2 are not", func() {
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/devices/0000:00:00.2",
					"sys/bus/pci/drivers/fake",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/fake",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/fake",
				},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:00:00.1/sriov_numvfs":   []byte("32"),
					"sys/bus/pci/devices/0000:00:00.1/sriov_totalvfs": []byte("64"),
					"sys/bus/pci/devices/0000:00:00.2/sriov_numvfs":   []byte("0"),
				},
			}

			defer fs.Use()()

			rf := factory.NewResourceFactory("fake", "fake", true)
			p := accelerator.NewAccelDeviceProvider(rf)

			dev1 := &ghw.PCIDevice{
				Address: "0000:00:00.1",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb"},
				Product: &pcidb.Product{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbb"},
			}

			dev2 := &ghw.PCIDevice{
				Address: "0000:00:00.2",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbb"},
				Product: &pcidb.Product{Name: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaabbbbbbbbbbbbbbbbbbbbbbbbb"},
			}

			devInvalid := &ghw.PCIDevice{
				Address: "0000:00:00.3",
				Class:   &pcidb.Class{ID: "completely unparsable"},
			}

			devNoSysFs := &ghw.PCIDevice{
				Address: "0000:00:00.4",
				Class:   &pcidb.Class{ID: "1024"},
				Vendor:  &pcidb.Vendor{Name: "Vendor"},
				Product: &pcidb.Product{Name: "Product"},
			}

			devsToAdd := []*ghw.PCIDevice{dev1, dev2, devInvalid, devNoSysFs}

			err := p.AddTargetDevices(devsToAdd, 0x1024)
			It("shouldn't return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return only 1 device on GetDevices()", func() {
				Expect(p.GetDevices()).To(HaveLen(2))
			})
		})
	})
})
