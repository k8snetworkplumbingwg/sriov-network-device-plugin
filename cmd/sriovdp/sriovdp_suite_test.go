package main

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSriovdp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sriovdp Suite")
}
