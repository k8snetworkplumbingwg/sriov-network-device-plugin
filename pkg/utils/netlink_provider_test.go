package utils

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
)

const (
	fakeFwAppName = "fakeFwAppName"
)

var errFakeNetlink = errors.New("fake netlink error")

var _ = Describe("NetlinkProvider Functions", func() {
	var (
		mockProvider *mocks.NetlinkProvider
	)

	BeforeEach(func() {
		mockProvider = &mocks.NetlinkProvider{}
		SetNetlinkProviderInst(mockProvider)
	})

	Describe("IsDevlinkDDPSupportedByDevice", func() {
		It("should return true when device is supported", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(map[string]string{fwAppNameKey: fakeFwAppName}, nil)

			result := IsDevlinkDDPSupportedByDevice("fakeDevice")
			Expect(result).To(BeTrue())
		})

		It("should return false when device is not supported", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
				Return(nil, errFakeNetlink)

			result := IsDevlinkDDPSupportedByDevice("fakeDevice")
			Expect(result).To(BeFalse())
		})
	})

	Describe("DevlinkGetDDPProfiles", func() {
		It("should return DDP profiles when no error occurs", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", pciBus, "fakeDevice").
				Return(map[string]string{fwAppNameKey: fakeFwAppName}, nil)

			profile, err := DevlinkGetDDPProfiles("fakeDevice")
			Expect(err).ToNot(HaveOccurred())
			Expect(profile).To(Equal(fakeFwAppName))
		})

		It("should return an error when fetching DDP profiles fails", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", pciBus, "fakeDevice").
				Return(nil, errFakeNetlink)

			profile, err := DevlinkGetDDPProfiles("fakeDevice")
			Expect(err).To(HaveOccurred())
			Expect(profile).To(BeEmpty())
		})
	})

	Describe("DevlinkGetDeviceInfoByNameAndKeys", func() {
		It("should return device info when key is found", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", pciBus, "fakeDevice").
				Return(map[string]string{fwAppNameKey: fakeFwAppName}, nil)

			info, err := DevlinkGetDeviceInfoByNameAndKeys("fakeDevice", []string{fwAppNameKey})
			Expect(err).ToNot(HaveOccurred())
			Expect(info[fwAppNameKey]).To(Equal(fakeFwAppName))
		})

		It("should return an error when key is not found", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", pciBus, "fakeDevice").
				Return(map[string]string{}, nil)

			info, err := DevlinkGetDeviceInfoByNameAndKeys("fakeDevice", []string{fwAppNameKey})
			Expect(err).To(HaveOccurred())
			Expect(info).To(BeNil())
		})

		It("should return an error when fetching device info fails", func() {
			mockProvider.
				On("GetDevlinkGetDeviceInfoByNameAsMap", pciBus, "fakeDevice").
				Return(nil, errFakeNetlink)

			info, err := DevlinkGetDeviceInfoByNameAndKeys("fakeDevice", []string{fwAppNameKey})
			Expect(err).To(HaveOccurred())
			Expect(info).To(BeNil())
		})
	})
})

func init() {
	SetDefaultMockNetlinkProvider()
}
