/*
 * SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package resources

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakePCIDriverOps struct {
	current      map[string]string
	driverExists bool
	inUse        map[string]bool
	owner        string
	errors       map[string]error
	calls        []string
}

type serialPCIDriverOps struct {
	currentCalls atomic.Int32
	entered      chan int32
	releaseFirst chan struct{}
}

type changingPCIDriverOps struct {
	*fakePCIDriverOps
	sequence []string
}

func (o *changingPCIDriverOps) CurrentDriver(pciAddr string) (string, error) {
	if len(o.sequence) == 0 {
		return o.fakePCIDriverOps.CurrentDriver(pciAddr)
	}
	driver := o.sequence[0]
	o.sequence = o.sequence[1:]
	o.calls = append(o.calls, "current:"+pciAddr)
	o.current[pciAddr] = driver
	return driver, nil
}

func (o *serialPCIDriverOps) CurrentDriver(string) (string, error) {
	call := o.currentCalls.Add(1)
	o.entered <- call
	if call == 1 {
		<-o.releaseFirst
	}
	return "mlx5_core", nil
}

func (*serialPCIDriverOps) DriverExists(string) bool                 { return true }
func (*serialPCIDriverOps) DeviceInUse(string) (bool, string, error) { return false, "", nil }
func (*serialPCIDriverOps) SetDriverOverride(string, string) error   { return nil }
func (*serialPCIDriverOps) Unbind(string, string) error              { return nil }
func (*serialPCIDriverOps) Bind(string, string) error                { return nil }

func (f *fakePCIDriverOps) CurrentDriver(pciAddr string) (string, error) {
	f.calls = append(f.calls, "current:"+pciAddr)
	if err := f.errors["current:"+pciAddr]; err != nil {
		return "", err
	}
	return f.current[pciAddr], nil
}

func (f *fakePCIDriverOps) DriverExists(driver string) bool {
	f.calls = append(f.calls, "exists:"+driver)
	return f.driverExists
}

func (f *fakePCIDriverOps) DeviceInUse(pciAddr string) (bool, string, error) {
	f.calls = append(f.calls, "in-use:"+pciAddr)
	return f.inUse[pciAddr], f.owner, f.errors["in-use:"+pciAddr]
}

func (f *fakePCIDriverOps) SetDriverOverride(pciAddr, driver string) error {
	f.calls = append(f.calls, "override:"+pciAddr+":"+driver)
	return f.errors["override:"+driver]
}

func (f *fakePCIDriverOps) Unbind(pciAddr, driver string) error {
	f.calls = append(f.calls, "unbind:"+pciAddr+":"+driver)
	return f.errors["unbind:"+driver]
}

func (f *fakePCIDriverOps) Bind(pciAddr, driver string) error {
	f.calls = append(f.calls, "bind:"+pciAddr+":"+driver)
	if err := f.errors["bind:"+driver]; err != nil {
		return err
	}
	f.current[pciAddr] = driver
	return nil
}

var _ = Describe("PCI driver recovery", func() {
	const (
		pciAddr       = "0000:03:00.1"
		desiredDriver = "mlx5_core"
	)

	newManager := func(ops *fakePCIDriverOps) *hostPCIDriverManager {
		return &hostPCIDriverManager{ops: ops}
	}
	newOps := func(driver string) *fakePCIDriverOps {
		return &fakePCIDriverOps{
			current:      map[string]string{pciAddr: driver},
			driverExists: true,
			inUse:        map[string]bool{},
			errors:       map[string]error{},
		}
	}

	It("is idempotent and clears a stale driver override", func() {
		ops := newOps(desiredDriver)
		err := newManager(ops).EnsureDrivers([]string{pciAddr}, desiredDriver)

		Expect(err).NotTo(HaveOccurred())
		Expect(ops.calls).To(Equal([]string{
			"current:" + pciAddr,
			"current:" + pciAddr,
			"override:" + pciAddr + ":",
			"current:" + pciAddr,
		}))
	})

	It("rebinds an unowned VFIO device and verifies the result", func() {
		ops := newOps(vfioPCIDriver)
		err := newManager(ops).EnsureDrivers([]string{pciAddr}, desiredDriver)

		Expect(err).NotTo(HaveOccurred())
		Expect(ops.calls).To(Equal([]string{
			"current:" + pciAddr,
			"exists:" + desiredDriver,
			"in-use:" + pciAddr,
			"current:" + pciAddr,
			"exists:" + desiredDriver,
			"in-use:" + pciAddr,
			"override:" + pciAddr + ":" + desiredDriver,
			"in-use:" + pciAddr,
			"unbind:" + pciAddr + ":" + vfioPCIDriver,
			"bind:" + pciAddr + ":" + desiredDriver,
			"override:" + pciAddr + ":",
			"current:" + pciAddr,
		}))
	})

	It("revalidates a device that changes to VFIO after batch preflight", func() {
		ops := &changingPCIDriverOps{
			fakePCIDriverOps: newOps(desiredDriver),
			sequence:         []string{desiredDriver, vfioPCIDriver},
		}

		err := (&hostPCIDriverManager{ops: ops}).EnsureDrivers([]string{pciAddr}, desiredDriver)

		Expect(err).NotTo(HaveOccurred())
		Expect(ops.calls).To(ContainElement("unbind:" + pciAddr + ":" + vfioPCIDriver))
		Expect(ops.current[pciAddr]).To(Equal(desiredDriver))
	})

	It("rejects an owned VFIO device without changing sysfs", func() {
		ops := newOps(vfioPCIDriver)
		ops.inUse[pciAddr] = true
		ops.owner = "process 42 (qemu)"

		err := newManager(ops).EnsureDrivers([]string{pciAddr}, desiredDriver)

		Expect(err).To(MatchError(ContainSubstring("still in use by process 42 (qemu)")))
		Expect(ops.calls).NotTo(ContainElement(HavePrefix("override:")))
		Expect(ops.calls).NotTo(ContainElement(HavePrefix("unbind:")))
	})

	It("preflights all devices before changing any of them", func() {
		secondAddr := "0000:03:00.2"
		ops := newOps(vfioPCIDriver)
		ops.current[secondAddr] = vfioPCIDriver
		ops.inUse[secondAddr] = true
		ops.owner = "process 42 (qemu)"

		err := newManager(ops).EnsureDrivers([]string{pciAddr, secondAddr}, desiredDriver)

		Expect(err).To(HaveOccurred())
		Expect(ops.calls).NotTo(ContainElement(HavePrefix("override:")))
		Expect(ops.calls).NotTo(ContainElement(HavePrefix("unbind:")))
	})

	It("rolls back to VFIO when the desired driver bind fails", func() {
		ops := newOps(vfioPCIDriver)
		ops.errors["bind:"+desiredDriver] = errors.New("bind failed")

		err := newManager(ops).EnsureDrivers([]string{pciAddr}, desiredDriver)

		Expect(err).To(MatchError(ContainSubstring("bind failed")))
		Expect(ops.calls).To(ContainElements(
			"override:"+pciAddr+":"+vfioPCIDriver,
			"bind:"+pciAddr+":"+vfioPCIDriver,
			"override:"+pciAddr+":",
			"current:"+pciAddr,
		))
		Expect(ops.current[pciAddr]).To(Equal(vfioPCIDriver))
	})

	It("serializes concurrent recovery for the same PCI device", func() {
		ops := &serialPCIDriverOps{
			entered:      make(chan int32, 6),
			releaseFirst: make(chan struct{}),
		}
		manager := &hostPCIDriverManager{ops: ops}
		errors := make(chan error, 2)

		go func() { errors <- manager.EnsureDrivers([]string{pciAddr}, desiredDriver) }()
		Eventually(ops.entered).Should(Receive(Equal(int32(1))))
		go func() { errors <- manager.EnsureDrivers([]string{pciAddr}, desiredDriver) }()
		Consistently(ops.entered, 100*time.Millisecond).ShouldNot(Receive())

		close(ops.releaseFirst)
		for expected := int32(2); expected <= 6; expected++ {
			Eventually(ops.entered).Should(Receive(Equal(expected)))
		}
		Eventually(errors).Should(Receive(Succeed()))
		Eventually(errors).Should(Receive(Succeed()))
	})

	DescribeTable("fails closed before rebinding",
		func(configure func(*fakePCIDriverOps), expected string) {
			ops := newOps(vfioPCIDriver)
			configure(ops)
			err := newManager(ops).EnsureDrivers([]string{pciAddr}, desiredDriver)
			Expect(err).To(MatchError(ContainSubstring(expected)))
			Expect(ops.calls).NotTo(ContainElement(HavePrefix("unbind:")))
		},
		Entry("when current driver cannot be read", func(ops *fakePCIDriverOps) {
			ops.errors["current:"+pciAddr] = errors.New("read failed")
		}, "read failed"),
		Entry("when desired driver is not loaded", func(ops *fakePCIDriverOps) {
			ops.driverExists = false
		}, "is not loaded"),
		Entry("when ownership cannot be determined", func(ops *fakePCIDriverOps) {
			ops.errors["in-use:"+pciAddr] = errors.New("permission denied")
		}, "permission denied"),
		Entry("when current driver is unexpected", func(ops *fakePCIDriverOps) {
			ops.current[pciAddr] = "iavf"
		}, "uses driver \"iavf\""),
	)

	Describe("host VFIO owner detection", func() {
		var root string
		BeforeEach(func() {
			var err error
			root, err = os.MkdirTemp("", "sriov-driver-recovery")
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(os.RemoveAll, root)
		})

		setup := func(useCdev, owned bool) *hostPCIDriverOps {
			pciBusPath := filepath.Join(root, "sys", "bus", "pci")
			procPath := filepath.Join(root, "proc")
			vfioPath := filepath.Join(root, "dev", "vfio")
			devicePath := filepath.Join(pciBusPath, "devices", pciAddr)
			Expect(os.MkdirAll(devicePath, 0o755)).To(Succeed())
			Expect(os.Symlink("../../../../kernel/iommu_groups/17", filepath.Join(devicePath, "iommu_group"))).To(Succeed())

			var node string
			if useCdev {
				Expect(os.MkdirAll(filepath.Join(devicePath, "vfio-dev", "vfio9"), 0o755)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(vfioPath, "devices"), 0o755)).To(Succeed())
				node = filepath.Join(vfioPath, "devices", "vfio9")
			} else {
				Expect(os.MkdirAll(vfioPath, 0o755)).To(Succeed())
				node = filepath.Join(vfioPath, "17")
			}
			Expect(os.WriteFile(node, []byte("vfio"), 0o600)).To(Succeed())

			fdPath := filepath.Join(procPath, "42", "fd")
			Expect(os.MkdirAll(fdPath, 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(procPath, "42", "comm"), []byte("qemu\n"), 0o600)).To(Succeed())
			if owned {
				Expect(os.Symlink(node, filepath.Join(fdPath, "8"))).To(Succeed())
			}

			return &hostPCIDriverOps{pciBusPath: pciBusPath, procPath: procPath, vfioDevPath: vfioPath}
		}

		DescribeTable("finds legacy and cdev VFIO owners",
			func(useCdev bool) {
				inUse, owner, err := setup(useCdev, true).DeviceInUse(pciAddr)
				Expect(err).NotTo(HaveOccurred())
				Expect(inUse).To(BeTrue())
				Expect(owner).To(Equal("process 42 (qemu)"))
			},
			Entry("legacy group device", false),
			Entry("VFIO cdev", true),
		)

		It("reports an unowned VFIO group", func() {
			inUse, owner, err := setup(false, false).DeviceInUse(pciAddr)
			Expect(err).NotTo(HaveOccurred())
			Expect(inUse).To(BeFalse())
			Expect(owner).To(BeEmpty())
		})

		It("fails closed when no VFIO device node can be identified", func() {
			ops := setup(false, false)
			Expect(os.Remove(filepath.Join(ops.vfioDevPath, "17"))).To(Succeed())
			_, _, err := ops.DeviceInUse(pciAddr)
			Expect(err).To(MatchError(ContainSubstring("no VFIO device node")))
		})

		It("finds a retained anonymous legacy VFIO device descriptor", func() {
			ops := setup(false, false)
			fdDir := filepath.Join(ops.procPath, "42", "fd")
			fdInfoDir := filepath.Join(ops.procPath, "42", "fdinfo")
			Expect(os.MkdirAll(fdInfoDir, 0o755)).To(Succeed())
			Expect(os.Symlink("anon_inode:[vfio-device]", filepath.Join(fdDir, "9"))).To(Succeed())
			fdInfo := "pos:\t0\nvfio-device-syspath: /sys/devices/pci0000:00/" + pciAddr + "\n"
			Expect(os.WriteFile(filepath.Join(fdInfoDir, "9"), []byte(fdInfo), 0o600)).To(Succeed())

			inUse, owner, err := ops.DeviceInUse(pciAddr)

			Expect(err).NotTo(HaveOccurred())
			Expect(inUse).To(BeTrue())
			Expect(owner).To(Equal("process 42 (qemu)"))
		})

		It("fails closed for an anonymous VFIO descriptor on kernels without VFIO fdinfo", func() {
			ops := setup(false, false)
			fdDir := filepath.Join(ops.procPath, "42", "fd")
			Expect(os.Symlink("anon_inode:[vfio-device]", filepath.Join(fdDir, "9"))).To(Succeed())

			inUse, _, err := ops.DeviceInUse(pciAddr)

			Expect(err).NotTo(HaveOccurred())
			Expect(inUse).To(BeTrue())
		})
	})

	It("writes driver operations through the mounted host PCI sysfs", func() {
		root, err := os.MkdirTemp("", "sriov-driver-sysfs")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(os.RemoveAll, root)

		devicePath := filepath.Join(root, "devices", pciAddr)
		vfioDriverPath := filepath.Join(root, "drivers", vfioPCIDriver)
		desiredDriverPath := filepath.Join(root, "drivers", desiredDriver)
		for _, path := range []string{devicePath, vfioDriverPath, desiredDriverPath} {
			Expect(os.MkdirAll(path, 0o755)).To(Succeed())
		}
		for _, path := range []string{
			filepath.Join(devicePath, "driver_override"),
			filepath.Join(vfioDriverPath, "unbind"),
			filepath.Join(desiredDriverPath, "bind"),
		} {
			Expect(os.WriteFile(path, nil, 0o600)).To(Succeed())
		}
		ops := &hostPCIDriverOps{pciBusPath: root}

		Expect(ops.SetDriverOverride(pciAddr, desiredDriver)).To(Succeed())
		data, err := os.ReadFile(filepath.Join(devicePath, "driver_override"))
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal([]byte(desiredDriver)))
		Expect(ops.SetDriverOverride(pciAddr, "")).To(Succeed())
		data, err = os.ReadFile(filepath.Join(devicePath, "driver_override"))
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal([]byte("\n")))

		Expect(ops.Unbind(pciAddr, vfioPCIDriver)).To(Succeed())
		data, err = os.ReadFile(filepath.Join(vfioDriverPath, "unbind"))
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal([]byte(pciAddr)))
		Expect(ops.Bind(pciAddr, desiredDriver)).To(Succeed())
		data, err = os.ReadFile(filepath.Join(desiredDriverPath, "bind"))
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(Equal([]byte(pciAddr)))
	})

	It("recognizes decimal process directory names only", func() {
		Expect(isDecimal("123")).To(BeTrue())
		Expect(isDecimal("self")).To(BeFalse())
		Expect(isDecimal(strings.Repeat("1", 8))).To(BeTrue())
	})
})
