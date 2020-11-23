package resources_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("genericInfoProvider", func() {
	Describe("creating new genericInfoProvider", func() {
		var pool types.DeviceInfoProvider
		BeforeEach(func() {
			pool = resources.NewGenericInfoProvider("fakePCIAddr")
		})
		It("should return valid genericInfoProvider object", func() {
			Expect(pool).NotTo(Equal(nil))
			// FIXME: Expect(reflect.TypeOf(pool)).To(Equal(reflect.TypeOf(&genericInfoProvider{})))
		})
	})
	Describe("getting mounts", func() {
		It("should always return an empty array", func() {
			pool := resources.NewGenericInfoProvider("fakePCIAddr")
			Expect(pool.GetMounts()).To(BeEmpty())
		})
	})
	Describe("getting device specs", func() {
		It("should always return an empty map", func() {
			pool := resources.NewGenericInfoProvider("fakePCIAddr")
			Expect(pool.GetDeviceSpecs()).To(BeEmpty())
		})
	})
})
