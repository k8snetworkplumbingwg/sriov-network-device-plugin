package resources_test

import (
	"github.com/intel/sriov-network-device-plugin/pkg/resources"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("UioPool", func() {
	Describe("creating new UIO resource pool", func() {
		var uioPool types.DeviceInfoProvider
		BeforeEach(func() {
			uioPool = resources.NewUioResource()
		})
		It("should return valid uioResource object", func() {
			Expect(uioPool).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(uioPool)).To(Equal(reflect.TypeOf(&uioResource{})))
		})
	})
	DescribeTable("getting device specs",
		func(fs *utils.FakeFilesystem, pciAddr string, expected []*pluginapi.DeviceSpec) {
			defer fs.Use()()

			pool := resources.NewUioResource()
			specs := pool.GetDeviceSpecs(pciAddr)
			Expect(specs).To(ConsistOf(expected))
		},
		Entry("empty", &utils.FakeFilesystem{}, "", []*pluginapi.DeviceSpec{}),
		Entry("PCI address passed, returns DeviceSpec with paths to its UIO devices",
			&utils.FakeFilesystem{
				Dirs: []string{
					"sys/bus/pci/devices/0000:02:00.0/uio/uio0",
				},
			},
			"0000:02:00.0",
			[]*pluginapi.DeviceSpec{
				{HostPath: "/dev/uio0", ContainerPath: "/dev/uio0", Permissions: "mrw"},
			},
		),
	)
	Describe("getting mounts", func() {
		It("should always return empty array of mounts", func() {
			pool := resources.NewUioResource()
			result := pool.GetMounts("fakePCIAddr")
			Expect(result).To(BeEmpty())
		})
	})
	Describe("getting env val", func() {
		It("should always return passed PCI address", func() {
			in := "00:02.0"
			pool := resources.NewUioResource()
			out := pool.GetEnvVal(in)
			Expect(out).To(Equal(in))
		})
	})
})
