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

package auxnetdevice_test

import (
	"testing"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/auxnetdevice"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	tmocks "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
)

func newPciDevice(pciAddr string) *ghw.PCIDevice {
	return &ghw.PCIDevice{
		Address: pciAddr,
		Vendor:  &pcidb.Vendor{ID: "15b3"},
		Product: &pcidb.Product{ID: "a2d6"},
	}
}

func TestAuxnetdevice(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auxnetdevice Suite")
}

var _ = Describe("AuxNetDevice", func() {
	Describe("creating new AuxNetDevice", func() {
		t := GinkgoT()
		Context("successfully", func() {
			It("should populate fields", func() {
				fs := &utils.FakeFilesystem{
					Dirs: []string{
						"sys/bus/pci/devices/0000:00:00.1/net/net0",
						"sys/bus/pci/drivers/mlx5_core",
					},
					Symlinks: map[string]string{
						"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/mlx5_core",
					},
					Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0")},
				}
				defer fs.Use()()
				utils.SetDefaultMockNetlinkProvider()
				auxDevID := "mlx5_core.sf.0"
				fakeSriovnetProvider := mocks.SriovnetProvider{}
				fakeSriovnetProvider.
					On("GetUplinkRepresentorFromAux", auxDevID).Return("net0", nil).
					On("GetPfPciFromAux", auxDevID).Return("0000:00:00.1", nil).
					On("GetSfIndexByAuxDev", auxDevID).Return(1, nil).
					On("GetNetDevicesFromAux", auxDevID).Return([]string{"eth0"}, nil)
				utils.SetSriovnetProviderInst(&fakeSriovnetProvider)

				f := factory.NewResourceFactory("fake", "fake", true)
				in := newPciDevice("0000:00:00.1")
				rc := &types.ResourceConfig{}

				dev, err := auxnetdevice.NewAuxNetDevice(in, auxDevID, f, rc)
				Expect(err).NotTo(HaveOccurred())
				Expect(dev).NotTo(BeNil())
				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				Expect(dev.GetNetName()).To(Equal("eth0"))
				Expect(dev.GetEnvVal()).To(Equal(auxDevID))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev.IsRdma()).To(BeFalse())
				Expect(dev.GetLinkType()).To(Equal("fakeLinkType"))
				Expect(dev.GetFuncID()).To(Equal(1))
				Expect(dev.GetAPIDevice().Topology.Nodes[0].ID).To(Equal(int64(0)))
				Expect(dev.GetAuxType()).To(Equal("sf"))
				fakeSriovnetProvider.AssertExpectations(t)
			})
		})
		Context("with two devices but only one of them being RDMA", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				DeviceType:     types.AuxNetDeviceType,
				SelectorObj: &types.AuxNetDeviceSelectors{
					GenericNetDeviceSelectors: types.GenericNetDeviceSelectors{IsRdma: true},
				},
			}
			fs := &utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:00:00.1",
					"sys/bus/pci/devices/0000:00:00.2",
					"sys/bus/pci/drivers/mlx5_core",
				},
				Symlinks: map[string]string{
					"sys/bus/pci/devices/0000:00:00.1/driver": "../../../../bus/pci/drivers/mlx5_core",
					"sys/bus/pci/devices/0000:00:00.2/driver": "../../../../bus/pci/drivers/mlx5_core",
				},
			}

			rdma1 := &tmocks.RdmaSpec{}
			// fake1 will have 2 RDMA device specs
			fake1ds := []*pluginapi.DeviceSpec{
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			rdma1.On("IsRdma").Return(true).On("GetRdmaDeviceSpec").Return(fake1ds)

			rdma2 := &tmocks.RdmaSpec{}
			rdma2.On("IsRdma").Return(false)

			f := &tmocks.ResourceFactory{}

			pciAddr1 := "0000:00:00.1"
			pciAddr2 := "0000:00:00.2"
			auxDevName1 := "mlx5_core.eth.1"
			auxDevName2 := "mlx5_core.eth-rep.2"
			mockInfo1 := &tmocks.DeviceInfoProvider{}
			mockInfo1.On("GetEnvVal").Return(auxDevName1)
			mockInfo1.On("GetDeviceSpecs").Return(nil)
			mockInfo1.On("GetMounts").Return(nil)
			mockInfo2 := &tmocks.DeviceInfoProvider{}
			mockInfo2.On("GetEnvVal").Return(auxDevName2)
			mockInfo2.On("GetDeviceSpecs").Return(nil)
			mockInfo2.On("GetMounts").Return(nil)
			f.On("GetDefaultInfoProvider", auxDevName1, "mlx5_core").Return(mockInfo1).
				On("GetDefaultInfoProvider", auxDevName2, "mlx5_core").Return(mockInfo2).
				On("GetRdmaSpec", types.AuxNetDeviceType, auxDevName1).Return(rdma1).
				On("GetRdmaSpec", types.AuxNetDeviceType, auxDevName2).Return(rdma2)

			fakeSriovnetProvider := mocks.SriovnetProvider{}
			fakeSriovnetProvider.On("GetUplinkRepresentorFromAux", auxDevName1).Return("net0", nil).
				On("GetPfPciFromAux", auxDevName1).Return("0000:00:00.0", nil).
				On("GetSfIndexByAuxDev", auxDevName1).Return(1, nil).
				On("GetNetDevicesFromAux", auxDevName1).Return([]string{"eth0"}, nil).
				On("GetUplinkRepresentorFromAux", auxDevName2).Return("net0", nil).
				On("GetPfPciFromAux", auxDevName2).Return("0000:00:00.0", nil).
				On("GetSfIndexByAuxDev", auxDevName2).Return(2, nil).
				On("GetNetDevicesFromAux", auxDevName2).Return([]string{"eth1"}, nil)

			in1 := newPciDevice(pciAddr1)
			in2 := newPciDevice(pciAddr2)

			It("should populate Rdma device specs if isRdma", func() {
				defer fs.Use()()
				utils.SetSriovnetProviderInst(&fakeSriovnetProvider)
				dev, err := auxnetdevice.NewAuxNetDevice(in1, auxDevName1, f, rc)

				Expect(err).NotTo(HaveOccurred())
				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				Expect(dev.IsRdma()).To(BeTrue())
				Expect(dev.GetEnvVal()).To(Equal(auxDevName1))
				Expect(dev.GetDeviceSpecs()).To(HaveLen(2)) // 2x Rdma devs
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(dev.GetAuxType()).To(Equal("eth"))
				mockInfo1.AssertExpectations(t)
				rdma1.AssertExpectations(t)
			})
			It("but not otherwise", func() {
				defer fs.Use()()
				utils.SetSriovnetProviderInst(&fakeSriovnetProvider)
				dev, err := auxnetdevice.NewAuxNetDevice(in2, auxDevName2, f, rc)

				Expect(err).NotTo(HaveOccurred())
				Expect(dev.GetDriver()).To(Equal("mlx5_core"))
				Expect(dev.GetEnvVal()).To(Equal(auxDevName2))
				Expect(dev.IsRdma()).To(BeFalse())
				Expect(dev.GetDeviceSpecs()).To(HaveLen(0))
				Expect(dev.GetMounts()).To(HaveLen(0))
				Expect(dev.GetAuxType()).To(Equal("eth-rep"))
				mockInfo2.AssertExpectations(t)
				rdma2.AssertExpectations(t)
			})
		})
	})
})
