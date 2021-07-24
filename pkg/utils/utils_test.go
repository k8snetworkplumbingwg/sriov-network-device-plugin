package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func assertShouldFail(err error, shouldFail bool) {
	if shouldFail {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).NotTo(HaveOccurred())
	}
}

var _ = Describe("In the utils package", func() {
	DescribeTable("getting PF PCI address",
		func(fs *FakeFilesystem, addr string, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetPfAddr(addr)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("address of a PF is passed", &FakeFilesystem{}, "0000:00:00.0", "0000:00:00.0", false),
		Entry("physfn is not a symlink",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.0/physfn": []byte("invalid content")},
			},
			"0000:00:00.0", "", true,
		),
		Entry("getting PF address of a VF",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:00:00.0", "sys/bus/pci/devices/0000:00:00.1"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:00:00.1/physfn": "../0000:00:00.0"},
			},
			"0000:00:00.1", "0000:00:00.0", false,
		),
	)

	DescribeTable("checking whether device is a SRIOV PF",
		func(fs *FakeFilesystem, addr string, expected bool) {
			defer fs.Use()()
			actual := IsSriovPF(addr)
			Expect(actual).To(Equal(expected))
		},
		Entry("sriov_totalvfs file exists",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.0/sriov_totalvfs": []byte("0")},
			},
			"0000:00:00.0", true,
		),
		Entry("sriov_totalvfs file doesn't exist",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.0"}}, "0000:00:00.0", false,
		),
	)

	DescribeTable("checking whether device is a SRIOV VF",
		func(fs *FakeFilesystem, addr string, expected bool) {
			defer fs.Use()()
			actual := IsSriovVF(addr)
			Expect(actual).To(Equal(expected))
		},
		Entry("physfn file exists and is a symlink to PF",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:00:00.0", "sys/bus/pci/devices/0000:00:00.1"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:00:00.1/physfn": "../0000:00:00.0"},
			},
			"0000:00:00.1", true,
		),
		Entry("physfn file doesn't exist",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:00:00.1"}}, "0000:00:00.1", false,
		),
	)

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
			assertShouldFail(err, shouldFail)
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
			assertShouldFail(err, shouldFail)
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

	DescribeTable("getting NUMA node of device",
		func(fs *FakeFilesystem, pciAddr string, expected int) {
			defer fs.Use()()
			Expect(GetDevNode(pciAddr)).To(Equal(expected))
		},
		Entry("reading the device path fails", &FakeFilesystem{}, "0000:00:00.0", -1),
		Entry("converting the NUMA node to integer fails",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("invalid content")},
			},
			"0000:00:00.1", -1,
		),
		Entry("finding positive NUMA node",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("1")},
			},
			"0000:00:00.1", 1,
		),
		Entry("finding zero NUMA node",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("0")},
			},
			"0000:00:00.1", 0,
		),
		Entry("finding negative NUMA node",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:00:00.1"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:00:00.1/numa_node": []byte("-1")},
			},
			"0000:00:00.1", -1,
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

	DescribeTable("checking that device PCI address is valid and device exists",
		func(fs *FakeFilesystem, addr, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := ValidPciAddr(addr)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
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
			//TODO: adapt test to running in a virtual environment
			actualHost, _, err := GetVFIODeviceFile(device)
			Expect(actualHost).To(Equal(expected))
			assertShouldFail(err, shouldFail)
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
			assertShouldFail(err, shouldFail)
		},
		Entry("could not get directory information for device",
			&FakeFilesystem{},
			"0000:01:10.0", "", true,
		),
		Entry("uio is not a dir",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:10.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:10.0/uio": []byte("junk")},
			},
			"0000:01:10.0", "", true,
		),
		Entry("uio device file path is returned",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/uio/uio1"},
			},
			"0000:01:10.0", "/dev/uio1", false,
		),
	)

	DescribeTable("getting driver name",
		func(fs *FakeFilesystem, device, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetDriverName(device)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("driver link doesn't exist",
			&FakeFilesystem{},
			"0000:01:10.0", "", true,
		),
		Entry("correct driver name is returned",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/drivers/fake"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/driver": "../../../../bus/pci/drivers/fake"},
			},
			"0000:01:10.0", "fake", false,
		),
	)

	DescribeTable("getting interface names",
		func(fs *FakeFilesystem, device string, expected []string, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetNetNames(device)
			Expect(actual).To(ConsistOf(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("device doesn't exist", &FakeFilesystem{}, "0000:01:10.0", nil, true),
		Entry("net is not a directory",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:10.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:10.0/net": []byte("junk")},
			},
			"0000:01:10.0", nil, true,
		),
		Entry("single network interface",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/net/fake0"}},
			"0000:01:10.0", []string{"fake0"}, false,
		),
		Entry("multiple network interfaces for single device",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/net/fake0", "sys/bus/pci/devices/0000:01:10.0/net/fake1"},
			},
			"0000:01:10.0", []string{"fake0", "fake1"}, false,
		),
	)

	DescribeTable("getting PF names",
		func(fs *FakeFilesystem, device string, expected string, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetPfName(device)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("device doesn't exist", &FakeFilesystem{}, "0000:01:10.0", nil, true),
		Entry("device is a VF and interface name exists",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/physfn/net/fakePF"}},
			"0000:01:10.0", "fakePF", false,
		),
		Entry("device is a VF and interface name does not exist",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/physfn/net/"}},
			"0000:01:10.0", "", true,
		),
		Entry("device is a PF and interface name exists",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/net/fakeIF"}},
			"0000:01:10.0", "fakeIF", false,
		),
		Entry("device is a PF interface name does not exist",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/net/fakeIF"}},
			"0000:01:10.0", "fakeIF", false,
		),
		Entry("net is not a directory at all",
			&FakeFilesystem{
				Dirs:  []string{"sys/bus/pci/devices/0000:01:10.0"},
				Files: map[string][]byte{"sys/bus/pci/devices/0000:01:10.0/net": []byte("junk")},
			},
			"0000:01:10.0", "", true,
		),
	)

	DescribeTable("getting ID of VF",
		func(fs *FakeFilesystem, device string, expected int, shouldFail bool) {
			defer fs.Use()()
			actual, err := GetVFID(device)
			Expect(actual).To(Equal(expected))
			assertShouldFail(err, shouldFail)
		},
		Entry("device doesn't exist",
			&FakeFilesystem{},
			"0000:01:10.0", -1, false),
		Entry("device has no link to PF",
			&FakeFilesystem{Dirs: []string{"sys/bus/pci/devices/0000:01:10.0"}},
			"0000:01:10.0", -1, false),
		Entry("PF has no VF links",
			&FakeFilesystem{
				Dirs:     []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0"},
			},
			"0000:01:10.0", -1, false),
		Entry("VF not found in PF",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
				},
			},
			"0000:01:10.0", -1, false),
		Entry("VF found in PF",
			&FakeFilesystem{
				Dirs: []string{"sys/bus/pci/devices/0000:01:10.0/", "sys/bus/pci/devices/0000:01:00.0/"},
				Symlinks: map[string]string{"sys/bus/pci/devices/0000:01:10.0/physfn": "../0000:01:00.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn0": "../0000:01:08.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn1": "../0000:01:09.0",
					"sys/bus/pci/devices/0000:01:00.0/virtfn2": "../0000:01:10.0",
				},
			},
			"0000:01:10.0", 2, false),
	)
})
