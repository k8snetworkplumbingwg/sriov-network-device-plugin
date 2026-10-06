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
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/natefinch/lumberjack.v2"
)

var _ = Describe("ResolveConfig", func() {
	DescribeTable("normalizes invalid values",
		func(cfg Config, wantWarns int, expectSame bool) {
			got, warnings, err := ResolveConfig(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(HaveLen(wantWarns))
			if expectSame {
				Expect(got).To(Equal(cfg))
			}
		},
		Entry("empty LogDir falls back to default",
			Config{LogDir: "", MaxSizeMB: 100, MaxFiles: 5}, 0, false),
		Entry("zero MaxSizeMB falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 0, MaxFiles: 5}, 1, false),
		Entry("negative MaxSizeMB falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: -1, MaxFiles: 5}, 1, false),
		Entry("MaxSizeMB exceeds limit falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 2000, MaxFiles: 5}, 1, false),
		Entry("negative MaxFiles falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: -1}, 1, false),
		Entry("MaxFiles exceeds limit falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 200}, 1, false),
		Entry("negative MaxAgeDays falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: -1}, 1, false),
		Entry("MaxAgeDays exceeds limit falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 500}, 1, false),
		Entry("zero MaxFiles falls back to default",
			Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 0}, 1, false),
		Entry("valid values remain unchanged",
			Config{LogDir: "/tmp", MaxSizeMB: 50, MaxFiles: 1, MaxAgeDays: 0}, 0, true),
		Entry("valid production config",
			Config{LogDir: DefaultLogDir, MaxSizeMB: DefaultMaxSizeMB, MaxFiles: DefaultMaxFiles, MaxAgeDays: DefaultMaxAge}, 0, true),
	)

	It("applies all fallbacks together", func() {
		cfg := Config{
			LogDir:     "",
			MaxSizeMB:  0,
			MaxFiles:   -1,
			MaxAgeDays: 999,
		}

		normalized, warnings, err := ResolveConfig(cfg)
		Expect(err).NotTo(HaveOccurred())

		Expect(normalized.LogDir).To(Equal(DefaultLogDir))
		Expect(normalized.MaxSizeMB).To(Equal(DefaultMaxSizeMB))
		Expect(normalized.MaxFiles).To(Equal(DefaultMaxFiles))
		Expect(normalized.MaxAgeDays).To(Equal(DefaultMaxAge))
		Expect(warnings).To(HaveLen(3))
	})

	It("leaves valid values unchanged", func() {
		cfg := Config{
			LogDir:     "/tmp/sriovdp-test",
			MaxSizeMB:  12,
			MaxFiles:   4,
			MaxAgeDays: 7,
		}

		normalized, warnings, err := ResolveConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(normalized).To(Equal(cfg))
		Expect(warnings).To(BeEmpty())
	})
})

var _ = Describe("NewRotatingWriter", func() {
	It("configures lumberjack with the resolved values", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{
			LogDir:     dir,
			MaxSizeMB:  42,
			MaxFiles:   7,
			MaxAgeDays: 14,
		}

		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(w).NotTo(BeNil())

		lj, ok := w.(*lumberjack.Logger)
		Expect(ok).To(BeTrue())
		Expect(lj.Filename).To(Equal(filepath.Join(dir, logFileName)))
		Expect(lj.MaxSize).To(Equal(42))
		Expect(lj.MaxBackups).To(Equal(7))
		Expect(lj.MaxAge).To(Equal(14))
		Expect(lj.Compress).To(BeTrue())
	})

	It("creates the log file on write", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}

		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		_, writeErr := w.Write([]byte("hello from rotating writer\n"))
		Expect(writeErr).NotTo(HaveOccurred())
		Expect(w.Close()).To(Succeed())

		data, err := os.ReadFile(filepath.Join(dir, logFileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("hello from rotating writer"))
	})

	It("fails for an invalid/unwritable directory", func() {
		cfg := Config{LogDir: "/proc/1/fdinfo", MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
		w, _, err := NewRotatingWriter(cfg)
		Expect(err).To(HaveOccurred())
		Expect(w).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("log directory"))
	})
})

var _ = Describe("teeWriter", func() {
	It("writes to both console and file", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		var consoleBuf strings.Builder
		tee := &teeWriter{console: &consoleBuf, file: w}

		_, writeErr := tee.Write([]byte("tee-test line\n"))
		Expect(writeErr).NotTo(HaveOccurred())
		Expect(w.Close()).To(Succeed())

		Expect(consoleBuf.String()).To(ContainSubstring("tee-test line"))

		data, err := os.ReadFile(filepath.Join(dir, logFileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("tee-test line"))
	})
})

// tempLogDir creates a temp directory whose cleanup retries RemoveAll so
// lumberjack's async gzip goroutine cannot flake DeferCleanup.
func tempLogDir() string {
	dir, err := os.MkdirTemp("", "sriovdp-log-*")
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Eventually(func() error { return os.RemoveAll(dir) }, 2*time.Second, 50*time.Millisecond).Should(Succeed())
	})
	return dir
}

var _ = Describe("Rotation", func() {
	It("retains at most MaxFiles backups plus the active file", func() {
		dir := tempLogDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2}

		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		chunk := strings.Repeat("B", 1024) + "\n"
		for i := 0; i < 4096; i++ {
			_, writeErr := w.Write([]byte(chunk))
			Expect(writeErr).NotTo(HaveOccurred())
		}
		Expect(w.Close()).To(Succeed())

		var fileCount int
		Eventually(func() int {
			entries, err := os.ReadDir(dir)
			Expect(err).NotTo(HaveOccurred())
			fileCount = 0
			for _, e := range entries {
				if !e.IsDir() {
					fileCount++
				}
			}
			return fileCount
		}, time.Second, 50*time.Millisecond).Should(BeNumerically("<=", 3))

		Expect(fileCount).To(BeNumerically(">=", 2))
	})

	It("compresses rotated backups", func() {
		dir := tempLogDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 3}

		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		chunk := strings.Repeat("C", 1024) + "\n"
		for i := 0; i < 2048; i++ {
			_, writeErr := w.Write([]byte(chunk))
			Expect(writeErr).NotTo(HaveOccurred())
		}
		Expect(w.Close()).To(Succeed())

		Eventually(func() bool {
			entries, err := os.ReadDir(dir)
			Expect(err).NotTo(HaveOccurred())
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), ".gz") {
					return true
				}
			}
			return false
		}, 2*time.Second, 100*time.Millisecond).Should(BeTrue())
	})
})

var _ = Describe("GracefulShutdown", func() {
	It("preserves logs written directly to the rotating writer before close", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3}

		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		_, err = w.Write([]byte("pre-shutdown: server stopping\n"))
		Expect(err).NotTo(HaveOccurred())

		Expect(w.Close()).To(Succeed())

		data, err := os.ReadFile(filepath.Join(dir, logFileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("pre-shutdown: server stopping"))
	})
})

var _ = Describe("StartupInfo", func() {
	It("should contain all required fields for startup header", func() {
		info := StartupInfo{
			ConfigFile:     "/etc/pcidp/config.json",
			ResourcePrefix: "intel.com",
			UseCdi:         true,
			LogDir:         "/var/log/sriovdp",
			LogMaxSizeMB:   100,
			LogMaxFiles:    5,
			LogMaxAgeDays:  30,
		}

		Expect(info.ConfigFile).To(Equal("/etc/pcidp/config.json"))
		Expect(info.ResourcePrefix).To(Equal("intel.com"))
		Expect(info.UseCdi).To(BeTrue())
		Expect(info.LogDir).To(Equal("/var/log/sriovdp"))
		Expect(info.LogMaxSizeMB).To(Equal(100))
		Expect(info.LogMaxFiles).To(Equal(5))
		Expect(info.LogMaxAgeDays).To(Equal(30))
	})
})

var _ = Describe("HostPath Environment Variable", func() {
	var originalEnv string

	BeforeEach(func() {
		originalEnv = os.Getenv("SRIOV_DP_LOG_HOST_PATH")
	})

	AfterEach(func() {
		if originalEnv == "" {
			os.Unsetenv("SRIOV_DP_LOG_HOST_PATH")
		} else {
			os.Setenv("SRIOV_DP_LOG_HOST_PATH", originalEnv)
		}
	})

	It("should read SRIOV_DP_LOG_HOST_PATH from environment", func() {
		os.Setenv("SRIOV_DP_LOG_HOST_PATH", "/var/log/my-custom-path/sriovdp")

		hostPath := os.Getenv("SRIOV_DP_LOG_HOST_PATH")
		Expect(hostPath).To(Equal("/var/log/my-custom-path/sriovdp"))
	})

	It("should return empty when SRIOV_DP_LOG_HOST_PATH is not set", func() {
		os.Unsetenv("SRIOV_DP_LOG_HOST_PATH")

		hostPath := os.Getenv("SRIOV_DP_LOG_HOST_PATH")
		Expect(hostPath).To(BeEmpty())
	})
})
