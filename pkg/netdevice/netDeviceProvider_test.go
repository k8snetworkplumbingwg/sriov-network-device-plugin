package netdevice_test

import (
	"github.com/intel/sriov-network-device-plugin/pkg/factory"
	"github.com/intel/sriov-network-device-plugin/pkg/netdevice"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	"github.com/jaypipes/ghw"
	"github.com/jaypipes/pcidb"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetDeviceProvider", func() {
	Describe("getting new instance of netDeviceProvide", func() {
		Context("with correct arguments", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			It("should return valid instance of the provider", func() {
				Expect(p).ToNot(BeNil())
			})
		})
	})
	Describe("getting devices", func() {
		Context("when there are none", func() {
			rf := &mocks.ResourceFactory{}
			p := netdevice.NewNetDeviceProvider(rf)
			devs := p.GetDevices()
			It("should return empty slice", func() {
				Expect(devs).To(BeEmpty())
			})
		})
	})
	Describe("adding 3 target devices", func() {
		Context("when 2 are valid devices, but 1 is a PF with SRIOV configured and 1 is invalid", func() {
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
			p := netdevice.NewNetDeviceProvider(rf)

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
				Class: &pcidb.Class{ID: "completely unparsable"},
			}

			devsToAdd := []*ghw.PCIDevice{dev1, dev2, devInvalid}

			err := p.AddTargetDevices(devsToAdd, 0x1024)
			It("shouldn't return an error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("should return only 1 device on GetDevices()", func() {
				Expect(p.GetDevices()).To(HaveLen(1))
			})
		})
	})
	/*DescribeTable("checking whether device has default route",
		func(fs *utils.FakeFilesystem, addr string, expected, shouldFail bool) {
			defer fs.Use()()

			//actual, err := hasDefaultRoute(addr)
			//Expect(actual).To(Equal(expected))
			//assertShouldFail(err, shouldFail)
		},
		Entry("device doesn't exist", &utils.FakeFilesystem{}, "0000:00:00.0", false, true),
		Entry("has interface in sys fs but netlink lib returns nil",
			&utils.FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.0/net/invalid0"}},
			"0000:00:00.0",
			true, false,
		),
	)*/
})
