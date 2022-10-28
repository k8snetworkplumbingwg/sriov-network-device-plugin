package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

// https://npf.io/2015/06/testing-exec-command
func FakeExecCommand(outs, exitCode string) func(string, ...string) *exec.Cmd {
	return func(command string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", command}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
			fmt.Sprintf("EXIT_CODE=%s", exitCode),
			fmt.Sprintf("OUTS=%s", outs)}
		return cmd
	}
}

// https://npf.io/2015/06/testing-exec-command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	exitCode := 0
	if val, ok := os.LookupEnv("EXIT_CODE"); ok {
		if code, err := strconv.ParseInt(val, 10, 0); err == nil && code >= 0 && code < 256 {
			exitCode = int(code)
		}
	}

	fmt.Printf("%s", os.Getenv("OUTS"))
	os.Exit(exitCode)
}

var _ = Describe("In the utils package", func() {
	DescribeTable("Get DDP Profiles",
		// int value(0-255) in string quote(e.g "0", "127")
		func(dev, ddpToolOut, want, exitCode string, shouldSucceed bool) {
			ddpExecCommand = FakeExecCommand(ddpToolOut, exitCode)
			defer func() { ddpExecCommand = exec.Command }()

			got, err := GetDDPProfiles(dev)

			if shouldSucceed {
				Expect(err).NotTo(HaveOccurred())
				Expect(want).To(Equal(got))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(want).To(Equal(got))
			}
		},
		Entry("valid device id and valid output from ddp tool",
			"0000:af:09.0",
			`
			{
				"DDPInventory": {
						"device": "154C",
						"address": "0000:03:02.0",
						"name": "enp3s2",
						"display": "Intel(R) Ethernet Virtual Function 700 Series",
						"DDPpackage": {
								"track_id": "80000008",
								"version": "1.0.3.0",
								"name": "GTPv1-C/U IPv4/IPv6 payload"
						}
				}
		}
			`,
			"GTPv1-C/U IPv4/IPv6 payload",
			"0",
			true,
		),
		Entry("valid device id and unexpected output from ddp tool",
			"0000:af:09.0",
			"What ever output",
			"",
			"0",
			false,
		),
		Entry("ddp tool command not found",
			"0000:af:09.0",
			"",
			"",
			"127", // Command not found
			false,
		),
		Entry("ddp tool command not found but stdout matches expected string",
			"0000:af:09.0",
			`
			{
				"DDPInventory": {
						"device": "154C",
						"address": "0000:03:02.0",
						"name": "enp3s2",
						"display": "Intel(R) Ethernet Virtual Function 700 Series",
						"DDPpackage": {
								"track_id": "80000008",
								"version": "1.0.3.0",
								"name": "GTPv1-C/U IPv4/IPv6 payload"
						}
				}
		}
			`,
			"",
			"127",
			false,
		),
		Entry("ddp tool output in alien format",
			"0000:af:09.0",
			"02:00.0    enp1s0f0       abcdece      1.0.3.a  GTPv1-C/U IPv4/IPv6 payload",
			"",
			"0",
			false,
		),
	)

	DescribeTable("getDDPNameFromStdout",
		func(ddpToolOut []byte, want string, shouldSucceed bool) {

			got, err := getDDPNameFromStdout(ddpToolOut)

			if shouldSucceed {
				Expect(err).NotTo(HaveOccurred())
				Expect(want).To(Equal(got))
			} else {
				Expect(err).To(HaveOccurred())
				Expect(want).To(Equal(got))
			}
		},
		Entry("valid output from ddp tool",
			[]byte(`
			{
				"DDPInventory": {
						"device": "154C",
						"address": "0000:03:02.0",
						"name": "enp3s2",
						"display": "Intel(R) Ethernet Virtual Function 700 Series",
						"DDPpackage": {
								"track_id": "80000008",
								"version": "1.0.3.0",
								"name": "GTPv1-C/U IPv4/IPv6 payload"
						}
				}
		}
			`),
			"GTPv1-C/U IPv4/IPv6 payload",
			true,
		),
		Entry("valid output from ddp tool but DDPpackage field is missing",
			[]byte(`
		{
			"DDPInventory": {
					"device": "154C",
					"address": "0000:03:02.0",
					"name": "enp3s2",
					"display": "Intel(R) Ethernet Virtual Function 700 Series"
			}
	}
		`),
			"",
			false,
		),
		Entry("ddp tool returns error object in json",
			[]byte(`
		{"DDPInventory":{
			"error": "2",
			"message": "An internal error has occurred"
		}
	}
	`),
			"",
			false,
		),
		Entry("invalid json output from ddp tool",
			[]byte(`
			{"DDPInventory":{
                "error": "2"
                "message": "An internal error has occurred"
        	}
		}
		`),
			"",
			false,
		),
		Entry("random text from ddp tool output",
			[]byte(``),
			"",
			false,
		),
	)
})
