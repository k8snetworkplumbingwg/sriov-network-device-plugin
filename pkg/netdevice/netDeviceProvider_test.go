package netdevice_test

import (
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	//. "github.com/onsi/gomega"
)

var _ = Describe("NetDeviceProvider", func() {
	PDescribeTable("checking whether device has default route",
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
	)
})
