package resources

import (
	"reflect"

	"github.com/intel/sriov-network-device-plugin/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetDevicePool", func() {
	Describe("creating new newNetDevicePool", func() {
		var pool types.DeviceInfoProvider
		BeforeEach(func() {
			pool = newNetDevicePool()
		})
		It("should return valid netDevicePool object", func() {
			Expect(pool).NotTo(Equal(nil))
			Expect(reflect.TypeOf(pool)).To(Equal(reflect.TypeOf(&netDevicePool{})))
		})
	})
	Describe("getting mounts", func() {
		It("should always return an empty array", func() {
			pool := netDevicePool{}
			Expect(pool.GetMounts("fakePCIAddr")).To(BeEmpty())
		})
	})
	Describe("getting device specs", func() {
		It("should always return an empty map", func() {
			pool := netDevicePool{}
			Expect(pool.GetDeviceSpecs("fakePCIAddr")).To(BeEmpty())
		})
	})
})
