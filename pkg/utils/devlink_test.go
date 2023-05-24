package utils

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

const (
	fakeFwAppName = "fakeFwAppName"
)

var fakeNetlinkError error

var _ = BeforeSuite(func() {
	fakeNetlinkError = errors.New("Fake netlink error")
})

var _ = Describe("In the devlink related functions", func() {
	DescribeTable("IsDevlinkDDPSupportedByPCIDevice",
		func(nf NetlinkFunction, device string, expectedValue bool) {
			netlinkDevlinkGetDeviceInfoByNameAsMap = nf
			result := IsDevlinkDDPSupportedByPCIDevice(device)
			Expect(result).To(Equal(expectedValue))
		},
		Entry("should return true", FakeNetlinkSuccess, "fakeDevice", true),
		Entry("should return true", FakeNetlinkError, "fakeDevice", false),
	)
	DescribeTable("DevlinkGetDDPProfiles",
		func(nf NetlinkFunction, device, expectedValue string, errorValue func() error) {
			netlinkDevlinkGetDeviceInfoByNameAsMap = nf
			profile, err := DevlinkGetDDPProfiles(device)
			if errorValue() != nil {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(profile).To(Equal(expectedValue))
		},
		Entry("should return fakeFwAppName and no error", FakeNetlinkSuccess, "fakeDevice", fakeFwAppName, func() error { return nil }),
		Entry("should return empty string and error", FakeNetlinkError, "fakeDevice", "", getFakeNetlinkError),
	)
	DescribeTable("DevlinkGetDeviceInfoByNameAndKeys",
		func(nf NetlinkFunction, device, keyToGet string, isSuccessful bool) {
			netlinkDevlinkGetDeviceInfoByNameAsMap = nf
			_, err := DevlinkGetDeviceInfoByNameAndKeys(device, []string{keyToGet})
			if isSuccessful {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
			}
		},
		Entry("should return no error", FakeNetlinkSuccess, "fakeDevice", fwAppNameKey, true),
		Entry("should return error", FakeNetlinkSuccess, "fakeDevice", "fakeKey", false),
	)
	DescribeTable("DDPNotSupportedError",
		func(device string) {
			err := DDPNotSupportedError(device)
			expectedErrorMessage := fmt.Errorf("%w: %s", ErrDDPNotSupported, device)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(expectedErrorMessage.Error()))
		},
		Entry("should return no error", "fakeDevice"),
	)
})

func FakeNetlinkSuccess(bus, device string) (map[string]string, error) {
	values := make(map[string]string)
	values[fwAppNameKey] = fakeFwAppName
	return values, nil
}

func FakeNetlinkError(bus, device string) (map[string]string, error) {
	return nil, fakeNetlinkError
}

func getFakeNetlinkError() error {
	return fakeNetlinkError
}
