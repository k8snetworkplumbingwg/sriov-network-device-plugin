package resources_test

import (
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PKeySelector", func() {
	Describe("PKey selector", func() {
		Context("filtering", func() {
			It("should return devices matching given PKeys", func() {
				pKeys := []string{"0x1", "0x2"}
				sel := resources.NewPKeySelector(pKeys)

				dev0 := mocks.PciNetDevice{}
				dev0.On("GetPKey").Return("0x1")

				dev1 := mocks.PciNetDevice{}
				dev1.On("GetPKey").Return("0x2")

				dev2 := mocks.PciNetDevice{}
				dev2.On("GetPKey").Return("0x3")

				in := []types.HostDevice{&dev0, &dev1, &dev2}
				filtered := sel.Filter(in)

				Expect(filtered).To(ContainElement(&dev0))
				Expect(filtered).To(ContainElement(&dev1))
				Expect(filtered).NotTo(ContainElement(&dev2))
			})
		})
	})
})
