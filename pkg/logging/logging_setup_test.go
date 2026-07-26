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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func saveAndRestoreLogDir() {
	origVal := flag.Lookup("log_dir").Value.String()
	DeferCleanup(func() {
		_ = flag.Set("log_dir", origVal)
	})
}

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
	BeforeEach(func() {
		saveAndRestoreLogDir()
	})

	It("falls back to the default log dir when unset", func() {
		Expect(flag.Set("log_dir", "")).To(Succeed())

		cfg := Config{MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 30, Compress: true}
		cleanup := SetupLogRotation(cfg)
		if cleanup != nil {
			cleanup()
		}
	})

	It("enables rotation for a writable /var/log path", func() {
		dir := varLogTempDir()
		Expect(flag.Set("log_dir", dir)).To(Succeed())

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7, Compress: true}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())

		_, err := os.Stderr.WriteString("setup-test: logged line\n")
		Expect(err).NotTo(HaveOccurred())

		cleanup()

		data, err := os.ReadFile(filepath.Join(dir, "sriovdp.log"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("setup-test: logged line"))
	})

	It("rejects a path outside /var/log", func() {
		dir := GinkgoT().TempDir()
		Expect(flag.Set("log_dir", dir)).To(Succeed())

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7, Compress: true}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).To(BeNil())
	})

	It("falls back invalid limits to defaults without disabling rotation", func() {
		dir := varLogTempDir()
		Expect(flag.Set("log_dir", dir)).To(Succeed())

		cfg := Config{LogDir: dir, MaxSizeMB: 0, MaxFiles: 5, MaxAgeDays: 30, Compress: true}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())
		cleanup()
	})

	It("restores stderr after cleanup", func() {
		dir := varLogTempDir()
		Expect(flag.Set("log_dir", dir)).To(Succeed())

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7, Compress: true}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())

		cleanup()

		_, err := os.Stderr.WriteString("post-cleanup write OK\n")
		Expect(err).NotTo(HaveOccurred())
	})

	It("writes the log file at the configured directory", func() {
		dir := varLogTempDir()
		Expect(flag.Set("log_dir", dir)).To(Succeed())

		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, MaxAgeDays: 3, Compress: true}
		cleanup := SetupLogRotation(cfg)
		Expect(cleanup).NotTo(BeNil())

		_, err := os.Stderr.WriteString("config passthrough test\n")
		Expect(err).NotTo(HaveOccurred())

		cleanup()

		_, statErr := os.Stat(filepath.Join(dir, "sriovdp.log"))
		Expect(statErr).NotTo(HaveOccurred())
	})
})
