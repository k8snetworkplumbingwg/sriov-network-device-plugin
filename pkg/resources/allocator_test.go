package resources_test

import (
	"reflect"

	"github.com/jaypipes/pcidb"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/netdevice"

	"github.com/jaypipes/ghw"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/resources"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Allocator", func() {
	var (
		f  types.ResourceFactory
		rc *types.ResourceConfig
	)
	newPciDeviceFn := func(pciAddr string) *ghw.PCIDevice {
		return &ghw.PCIDevice{
			Address: pciAddr,
			Vendor:  &pcidb.Vendor{ID: ""},
			Product: &pcidb.Product{ID: ""},
		}
	}

	BeforeEach(func() {
	})
	Describe("creating new packed allocator", func() {
		Context("with valid policy", func() {
			It("should return valid allocator", func() {
				packedAllocator := resources.NewPackedAllocator()
				expected := &resources.PackedAllocator{}
				Expect(reflect.TypeOf(packedAllocator)).To(Equal(reflect.TypeOf(expected)))
			})
		})
	})
	Describe("creating new device set", func() {
		Context("with no element", func() {
			It("should return valid device set", func() {
				ds := resources.NewDeviceSet()
				expected := make(resources.DeviceSet)
				Expect(reflect.TypeOf(ds)).To(Equal(reflect.TypeOf(expected)))
			})
		})
	})
	Describe("manipulating device set", func() {
		Context("by inserting and deleting elements", func() {
			It("should return no error and valid device set", func() {
				f = factory.NewResourceFactory("fake", "fake", true)
				rc = &types.ResourceConfig{SelectorObjs: []interface{}{types.NetDeviceSelectors{}}}
				ds := resources.NewDeviceSet()
				d1, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.1"), f, rc, 0)
				d2, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:af.0"), f, rc, 0)
				d3, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:1b.2"), f, rc, 0)
				d4, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:1b.0"), f, rc, 0)

				ds.Insert("0000:00:00.1", d1)
				ds.Insert("0000:00:af.0", d2)
				ds.Insert("0000:00:1b.2", d3)
				ds.Insert("0000:00:1b.0", d4)
				expectedSet := resources.DeviceSet{
					"0000:00:00.1": d1,
					"0000:00:af.0": d2,
					"0000:00:1b.2": d3,
					"0000:00:1b.0": d4,
				}
				Expect(ds).To(HaveLen(4))
				Expect(reflect.DeepEqual(ds, expectedSet)).To(Equal(true))

				sortedKeys := ds.AsSortedStrings()
				expectedSlice := []string{
					"0000:00:00.1",
					"0000:00:1b.0",
					"0000:00:1b.2",
					"0000:00:af.0",
				}
				Expect(sortedKeys).To(Equal(expectedSlice))

				ds.Delete("0000:00:00.1")
				ds.Delete("0000:00:af.0")
				ds.Delete("0000:00:1b.2")
				ds.Delete("0000:00:1b.0")
				Expect(ds).To(HaveLen(0))
			})
		})
	})
	DescribeTable("allocating with packed allocator",
		func(rqt *pluginapi.ContainerPreferredAllocationRequest, expected []string) {
			rc = &types.ResourceConfig{SelectorObjs: []interface{}{types.NetDeviceSelectors{}}}
			f = factory.NewResourceFactory("fake", "fake", true)
			d1, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:00.1"), f, rc, 0)
			d2, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:af.0"), f, rc, 0)
			d3, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:1b.2"), f, rc, 0)
			d4, _ := netdevice.NewPciNetDevice(newPciDeviceFn("0000:00:1b.0"), f, rc, 0)
			rp := resources.NewResourcePool(rc,
				map[string]types.HostDevice{
					"0000:00:00.1": d1,
					"0000:00:af.0": d2,
					"0000:00:1b.2": d3,
					"0000:00:1b.0": d4,
				},
			)
			pa := resources.NewPackedAllocator()
			sortedKeys := pa.Allocate(rqt, rp)
			Expect(sortedKeys).To(Equal(expected))
		},
		Entry("allocating successfully with 3 device IDs",
			&pluginapi.ContainerPreferredAllocationRequest{
				AvailableDeviceIDs: []string{
					"0000:00:00.1",
					"0000:00:af.0",
					"0000:00:1b.2",
					"0000:00:1b.0",
				},
				MustIncludeDeviceIDs: []string{},
				AllocationSize:       int32(3),
			},
			[]string{
				"0000:00:00.1",
				"0000:00:1b.0",
				"0000:00:1b.2",
			},
		),
		Entry("allocating with invalid available device IDs",
			&pluginapi.ContainerPreferredAllocationRequest{
				AvailableDeviceIDs: []string{
					"0000:00:00.2",
					"0000:00:af.1",
				},
				MustIncludeDeviceIDs: []string{},
				AllocationSize:       int32(1),
			},
			[]string{},
		),
		Entry("allocating with invalid must include device IDs",
			&pluginapi.ContainerPreferredAllocationRequest{
				AvailableDeviceIDs: []string{
					"0000:00:00.1",
					"0000:00:af.0",
					"0000:00:1b.2",
				},
				MustIncludeDeviceIDs: []string{
					"0000:00:00.5",
					"0000:00:00.6",
				},
				AllocationSize: int32(2),
			},
			[]string{},
		),
		Entry("allocating with invalid size 1",
			&pluginapi.ContainerPreferredAllocationRequest{
				AvailableDeviceIDs: []string{
					"0000:00:00.2",
					"0000:00:af.1",
				},
				MustIncludeDeviceIDs: []string{},
				AllocationSize:       int32(3),
			},
			[]string{},
		),
		Entry("allocating with invalid size 2",
			&pluginapi.ContainerPreferredAllocationRequest{
				AvailableDeviceIDs: []string{
					"0000:00:00.2",
					"0000:00:af.1",
				},
				MustIncludeDeviceIDs: []string{},
				AllocationSize:       int32(-1),
			},
			[]string{},
		),
		Entry("allocating with invalid size 3",
			&pluginapi.ContainerPreferredAllocationRequest{
				AvailableDeviceIDs: []string{
					"0000:00:00.2",
					"0000:00:af.0",
					"0000:00:1b.2",
				},
				MustIncludeDeviceIDs: []string{
					"0000:00:00.2",
					"0000:00:af.1",
				},
				AllocationSize: int32(3),
			},
			[]string{},
		),
	)
})
