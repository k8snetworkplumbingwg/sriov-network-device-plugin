package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("In the utils package", func() {
	DescribeTable("getting number of configured VFs",
		func(fs *FakeFilesystem, pf string, expected int) {
			defer fs.Use()()
			Expect(GetVFconfigured(pf)).To(Equal(expected))
		},
		Entry("reading the VF path fails", &FakeFilesystem{}, "0000:00:00.0", 0),
		Entry("converting the VFs number to integer fails",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/sriov_numvfs": []byte("invalid content")},
			},
			"0000:00:00.1", 0,
		),
		Entry("finding positive number of VFs",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/sriov_numvfs": []byte("32")},
			},
			"0000:00:00.1", 32,
		),
	)
	DescribeTable("getting VFs list",
		func(fs *FakeFilesystem, pf string, expected []string, shouldFail bool) {
			defer fs.Use()()
			vfList, err := GetVFList(pf)
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(vfList).To(Equal(expected))
		},
		Entry("reading the PF path fails", &FakeFilesystem{}, "0000:00:00.1", []string{}, true),
		Entry("VF list is correctly returned",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:01:00.0", "sys/bus/pci/devices/0000:01:10.0"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:10.0"},
			},
			"0000:01:00.0", []string{"0000:01:10.0"}, false,
		),
		Entry("empty VFs list is returned",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:00:00.3"},
			},
			"0000:00:00.3", []string{}, false,
		),
	)

	DescribeTable("getting PCI address from VFID",
		func(fs *FakeFilesystem, pf string, vf int, expected string, shouldFail bool) {
			defer fs.Use()()
			pciAddr, err := GetPciAddrFromVFID(pf, vf)
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			Expect(pciAddr).To(Equal(expected))
		},
		Entry(
			"PCI address is successfully returned",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:01:00.0", "sys/bus/pci/devices/0000:01:10.0"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:10.0"},
			},
			"0000:01:00.0", 0, "0000:01:10.0", false,
		),
		Entry("could not get directory information for the PF", &FakeFilesystem{}, "0000:01:00.0", 0, "", true),
		Entry("there's no symbolic link between virtual function and PCI",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:00.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:00.0/virtfn0": []byte("junk")},
			},
			"0000:01:00.0", 0, "", true,
		),
	)

	DescribeTable("getting SR-IOV VFs capacity for the PF",
		func(fs *FakeFilesystem, pf string, expected int) {
			defer fs.Use()()
			Expect(GetSriovVFcapacity(pf)).To(Equal(expected))
		},
		Entry("positive number of total VFs is returned",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:00.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:00.0/sriov_totalvfs": []byte("32")},
			},
			"0000:01:00.0", 32,
		),
		Entry("total vfs file doesn't exist",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:00.0"},
			},
			"0000:01:00.0", 0,
		),
		Entry("cannot convert junk from totalvfs to int",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:00.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:00.0/sriov_totalvfs": []byte("junk")},
			},
			"0000:01:00.0", 0,
		),
	)

	DescribeTable("checking that device status is up",
		func(fs *FakeFilesystem, dev string, expected bool) {
			defer fs.Use()()
			Expect(IsNetlinkStatusUp(dev)).To(Equal(expected))
		},
		Entry("all devices operstates are up should return true",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:00.0/net/eth0", "sys/bus/pci/devices/0000:01:00.0/net/eth1"},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:01:00.0/net/eth0/operstate": []byte("up"),
					"sys/bus/pci/devices/0000:01:00.0/net/eth1/operstate": []byte("up"),
				},
			},
			"0000:01:00.0", true,
		),
		Entry("at least one device operstate is down should return false",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:00.0/net/eth0", "sys/bus/pci/devices/0000:01:00.0/net/eth1"},
				Files: map[string][]byte{
					"sys/bus/pci/devices/0000:01:00.0/net/eth0/operstate": []byte("up"),
					"sys/bus/pci/devices/0000:01:00.0/net/eth1/operstate": []byte("down"),
				},
			},
			"0000:01:00.0", false,
		),
		PEntry("when operstate file doesn't exist should return false",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:00.0"},
			},
			"0000:01:00.0", false,
		),
	)

	DescribeTable("checking that device PCI addres is valid and device exists",
		func(fs *FakeFilesystem, addr, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := ValidPciAddr(addr)
			Expect(actual).To(Equal(expected))
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("long id is submitted and device exists",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:00.0"}},
			"0000:01:00.0", "0000:01:00.0", false,
		),
		Entry("short id is submitted and device exists",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:00.0"}},
			"01:00.0", "0000:01:00.0", false,
		),
		Entry("valid long id is submitted, but device doesn't exist",
			&FakeFilesystem{},
			"0000:01:00.0", "0000:01:00.0", true,
		),
		Entry("valid short id is submitted, but device doesn't exist",
			&FakeFilesystem{},
			"01:00.0", "0000:01:00.0", true,
		),
		Entry("invalid id is submitted",
			&FakeFilesystem{},
			"junk", "", true,
		),
	)

	DescribeTable("checking whether SR-IOV is configured",
		func(fs *FakeFilesystem, addr string, expected bool) {
			defer fs.Use()()
			Expect(SriovConfigured(addr)).To(Equal(expected))
		},
		Entry("SR-IOV is not configured", &FakeFilesystem{}, "0000:01:00.0", false),
		Entry("SR-IOV is configured",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:00.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:00.0/sriov_numvfs": []byte("32")},
			},
			"0000:01:00.0", true,
		),
	)

	DescribeTable("validating resource name",
		func(name string, expected bool) {
			Expect(ValidResourceName(name)).To(Equal(expected))
		},
		Entry("resource name is valid", "sriov_net_0", true),
		Entry("resource name is invalid", "junk-net-0", false),
	)

	DescribeTable("getting VFIO device file",
		func(fs *FakeFilesystem, device, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetVFIODeviceFile(device)
			Expect(actual).To(Equal(expected))
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("could not get directory information for device",
			&FakeFilesystem{},
			"0000:01:00.0", "", true,
		),
		Entry("finding iommuDir file fails",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0"}},
			"0000:01:10.0", "", true,
		),
		Entry("symlink to iommu group is invalid (normal dir instead of symlink)",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/iommu_group"}},
			"0000:01:10.0", "", true,
		),
		Entry("vfio device file path is returned",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:01:10.0", "sys/kernel/iommu_groups/1"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/iommu_group": "../../../../kernel/iommu_groups/1"},
			},
			"0000:01:10.0", "/dev/vfio/1", false,
		),
	)

	DescribeTable("getting UIO device file",
		func(fs *FakeFilesystem, device, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetUIODeviceFile(device)
			Expect(actual).To(Equal(expected))
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("could not get directory information for device",
			&FakeFilesystem{},
			"0000:01:10.0", "", true,
		),
		Entry("uio device file path is returned",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/uio/uio1"},
			},
			"0000:01:10.0", "/dev/uio1", false,
		),
	)
})
