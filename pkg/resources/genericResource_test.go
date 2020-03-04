package resources

import (
	"reflect"

	"github.com/intel/sriov-network-device-plugin/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("genericResource", func() {
	Describe("creating new genericResource", func() {
		var pool types.DeviceInfoProvider
		BeforeEach(func() {
			pool = NewGenericResource()
		})
		It("should return valid genericResource object", func() {
			Expect(pool).NotTo(Equal(nil))
			Expect(reflect.TypeOf(pool)).To(Equal(reflect.TypeOf(&genericResource{})))
		})
	})
	Describe("getting mounts", func() {
		It("should always return an empty array", func() {
			pool := genericResource{}
			Expect(pool.GetMounts("fakePCIAddr")).To(BeEmpty())
		})
	})
	Describe("getting device specs", func() {
		It("should always return an empty map", func() {
			pool := genericResource{}
			Expect(pool.GetDeviceSpecs("fakePCIAddr")).To(BeEmpty())
		})
	})
})
