package netdevice_test

import (
	"github.com/intel/sriov-network-device-plugin/pkg/netdevice"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"
	"k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NetResourcePool", func() {
	Context("getting a new instance of the pool", func() {
		rc := &types.ResourceConfig{ResourceName: "fake", ResourcePrefix: "fake"}
		devs := map[string]*v1beta1.Device{}
		pcis := map[string]types.PciDevice{}

		rp := netdevice.NewNetResourcePool(rc, devs, pcis)

		It("should return a valid instance of the pool", func() {
			Expect(rp).ToNot(BeNil())
		})
	})
	Describe("getting DeviceSpecs", func() {
		Context("for non-RDMA devices", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				IsRdma:         false,
			}
			devs := map[string]*v1beta1.Device{}

			rdmaNo := &mocks.RdmaSpec{}
			rdmaNo.On("IsRdma").Return(false)

			// fake1 will have 2 device specs
			fake1 := &mocks.PciNetDevice{}
			fake1ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			fake1.On("GetDeviceSpecs").Return(fake1ds).On("GetRdmaSpec").Return(rdmaNo)

			// fake2 will have 1 device spec
			fake2 := &mocks.PciNetDevice{}
			fake2ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake2"},
			}
			fake2.On("GetDeviceSpecs").Return(fake2ds).
				On("GetRdmaSpec").Return(rdmaNo)

			// fake3 will have 0 device specs
			fake3 := &mocks.PciNetDevice{}
			fake3ds := []*pluginapi.DeviceSpec{}
			fake2.On("GetDeviceSpecs").Return(fake3ds).On("GetRdmaSpec").Return(rdmaNo)

			pcis := map[string]types.PciDevice{"fake1": fake1, "fake2": fake2, "fake3": fake3}

			rp := netdevice.NewNetResourcePool(rc, devs, pcis)

			devIDs := []string{"fake1", "fake2"}

			actual := rp.GetDeviceSpecs(devIDs)

			It("should return valid slice of device specs", func() {
				Expect(actual).ToNot(BeNil())
				Expect(actual).To(HaveLen(3)) // fake1 + fake2 => 3 devices
				Expect(actual).To(ContainElement(fake1ds[0]))
				Expect(actual).To(ContainElement(fake1ds[1]))
				Expect(actual).To(ContainElement(fake2ds[0]))
			})
		})
		Context("for RDMA devices", func() {
			rc := &types.ResourceConfig{
				ResourceName:   "fake",
				ResourcePrefix: "fake",
				IsRdma:         true,
			}
			devs := map[string]*v1beta1.Device{}
			rdma1 := &mocks.RdmaSpec{}
			rdma2 := &mocks.RdmaSpec{}

			// fake1 will have 2 RDMA device specs and 0 regular device specs
			fake1 := &mocks.PciNetDevice{}
			fake1ds := []*pluginapi.DeviceSpec{
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1a"},
				&pluginapi.DeviceSpec{ContainerPath: "/fake/path", HostPath: "/dev/fake1b"},
			}
			rdma1.On("IsRdma").Return(true).On("GetRdmaDeviceSpec").Return(fake1ds)
			fake1.On("GetRdmaSpec").Return(rdma1).
				On("GetDeviceSpecs").Return(make([]*pluginapi.DeviceSpec, 0))

			// fake2 will have 0 rdma device specs to trigger error msg
			fake2 := &mocks.PciNetDevice{}
			fake2ds := []*pluginapi.DeviceSpec{}
			rdma2.On("IsRdma").Return(false).On("GetRdmaDeviceSpec").Return(fake2ds)
			fake2.On("GetDeviceSpecs").Return(make([]*pluginapi.DeviceSpec, 0)).
				On("GetRdmaSpec").Return(rdma2)

			pcis := map[string]types.PciDevice{"fake1": fake1, "fake2": fake2}

			rp := netdevice.NewNetResourcePool(rc, devs, pcis)

			devIDs := []string{"fake1", "fake2"}

			actual := rp.GetDeviceSpecs(devIDs)

			It("should return valid slice of device specs", func() {
				Expect(actual).ToNot(BeNil())
				Expect(actual).To(HaveLen(2)) // fake1 => 2 rdma devices
				Expect(actual).To(ContainElement(fake1ds[0]))
				Expect(actual).To(ContainElement(fake1ds[1]))
			})
		})
	})
})
