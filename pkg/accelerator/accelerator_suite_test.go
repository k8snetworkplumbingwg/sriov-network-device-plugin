package accelerator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAccelerator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Accelerator Suite")
}
