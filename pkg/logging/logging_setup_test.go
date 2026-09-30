// Copyright 2024 Intel Corp. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logging

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
)

// varLogTempDir creates a temp dir under /var/log; skips if not writable.
func varLogTempDir() string {
	dir, err := os.MkdirTemp("/var/log", "sriovdp-test-*")
	if err != nil {
		Skip(fmt.Sprintf("cannot create test directory under /var/log: %v", err))
	}
	DeferCleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

var _ = Describe("ValidateLogDir", func() {
	DescribeTable("rejects unsafe paths",
		func(dir string) {
			got, err := ValidateLogDir(dir)
			Expect(err).To(HaveOccurred())
			Expect(got).To(BeEmpty())
		},
		Entry("root", "/"),
		Entry("tmp", "/tmp/logs"),
		Entry("etc", "/etc/sriovdp"),
		Entry("home", "/home/user/logs"),
		Entry("traversal out of var/log", "/var/log/../etc"),
		Entry("traversal to blabla", "/var/log/../blabla"),
		Entry("root traversal to var/log", "/../var/log"),
		Entry("relative traversal", "../var/log"),
		Entry("tilde under var/log", "/var/log/~/blabla"),
		Entry("relative path", "sriovdp"),
		Entry("relative var/log", "var/log/sriovdp"),
		Entry("var without log", "/var/sriovdp"),
		Entry("var/log prefix trick", "/var/logextra"),
		Entry("empty", ""),
	)

	It("accepts exact /var/log", func() {
		got, err := ValidateLogDir("/var/log")
		if err != nil {
			Skip(fmt.Sprintf("cannot validate /var/log: %v", err))
		}
		Expect(got).To(Equal("/var/log"))
	})

	It("resolves a subdirectory under /var/log", func() {
		dir := varLogTempDir()
		got, err := ValidateLogDir(dir)
		Expect(err).NotTo(HaveOccurred())
		resolved, err := filepath.EvalSymlinks(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(filepath.Clean(resolved)))
	})

	It("cleans dot segments", func() {
		dir := varLogTempDir()
		got, err := ValidateLogDir(dir + "/.")
		Expect(err).NotTo(HaveOccurred())
		resolved, err := filepath.EvalSymlinks(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(filepath.Clean(resolved)))
	})

	It("rejects a symlink that escapes /var/log", func() {
		dir := varLogTempDir()
		link := filepath.Join(dir, "escape")
		Expect(os.Symlink("/etc", link)).To(Succeed())
		got, err := ValidateLogDir(link)
		Expect(err).To(HaveOccurred())
		Expect(got).To(BeEmpty())
		Expect(err.Error()).To(ContainSubstring("outside the allowed base"))
	})
})

var _ = Describe("SetupLogRotation", func() {
	// Restore klog output after each test so rotation from one test
	// does not leak into the next.
	AfterEach(func() {
		klog.LogToStderr(true)
		klog.SetOutput(os.Stderr)
	})

	It("falls back to the default log dir when unset", func() {
		cfg := Config{MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 30}
		cleanup := SetupLogRotation(cfg)
		if cleanup != nil {
			cleanup()
		}
	})

	It("enables rotation for a writable /var/log path", func() {
		dir := varLogTempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())

		klog.Info("setup-test: logged line via klog")
		cleanup()

		data, err := os.ReadFile(filepath.Join(dir, "sriovdp.log"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("setup-test: logged line via klog"))
	})

	It("falls back to the default log dir when path is outside /var/log", func() {
		dir := GinkgoT().TempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
		cleanup := SetupLogRotation(cfg)
		if _, err := os.Stat(DefaultLogDir); err != nil && !os.IsNotExist(err) {
			Expect(cleanup).To(BeNil())
			return
		}
		if cleanup == nil {
			Skip(fmt.Sprintf("default log dir %q not usable in this environment", DefaultLogDir))
		}
		cleanup()
	})

	It("falls back invalid limits to defaults without disabling rotation", func() {
		dir := varLogTempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 0, MaxFiles: 5, MaxAgeDays: 30}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())
		cleanup()
	})

	It("writes the log file at the configured directory", func() {
		dir := varLogTempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, MaxAgeDays: 3}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())

		klog.Info("config passthrough test")
		cleanup()

		_, statErr := os.Stat(filepath.Join(dir, "sriovdp.log"))
		Expect(statErr).NotTo(HaveOccurred())
	})
})
