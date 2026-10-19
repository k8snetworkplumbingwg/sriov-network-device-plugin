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
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
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

var _ = Describe("validateLogDirInput", func() {
	DescribeTable("rejects invalid input",
		func(dir string, errSubstr string) {
			err := validateLogDirInput(dir)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(errSubstr))
		},
		Entry("empty", "", "must not be empty"),
		Entry("control character", "/var/log/sriovdp\x01", "invalid characters"),
		Entry("leading dash", "-/var/log/sriovdp", "must not start with '-'"),
		Entry("whitespace", "/var/log/sriov dp", "invalid characters"),
		Entry("tab", "/var/log/sriov\tdp", "invalid characters"),
		Entry("single quote", "/var/log/sriov'dp", "invalid characters"),
		Entry("double quote", "/var/log/sriov\"dp", "invalid characters"),
		Entry("shell glob", "/var/log/sriov*dp", "invalid characters"),
		Entry("shell pipe", "/var/log/sriov|dp", "invalid characters"),
		Entry("shell semicolon", "/var/log/sriov;dp", "invalid characters"),
		Entry("shell redirection", "/var/log/sriov>dp", "invalid characters"),
		Entry("tilde", "/var/log/~/blabla", "invalid characters"),
	)

	DescribeTable("accepts valid input",
		func(dir string) {
			Expect(validateLogDirInput(dir)).NotTo(HaveOccurred())
		},
		Entry("default path", "/var/log/sriovdp"),
		Entry("hyphen and underscore", "/var/log/sriov-dp_test"),
		Entry("dot segment", "/var/log/sriovdp/logs.v1"),
	)
})

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
		Entry("control character", "/var/log/sriovdp\x01"),
		Entry("leading dash", "-/var/log/sriovdp"),
		Entry("whitespace", "/var/log/sriov dp"),
		Entry("tab", "/var/log/sriov\tdp"),
		Entry("single quote", "/var/log/sriov'dp"),
		Entry("double quote", "/var/log/sriov\"dp"),
		Entry("shell glob", "/var/log/sriov*dp"),
		Entry("shell pipe", "/var/log/sriov|dp"),
		Entry("shell semicolon", "/var/log/sriov;dp"),
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

	It("does not change permissions on an existing directory", func() {
		dir := varLogTempDir()
		const wantMode = os.FileMode(0o755)
		Expect(os.Chmod(dir, wantMode)).To(Succeed())

		got, err := ValidateLogDir(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).NotTo(BeEmpty())

		info, err := os.Stat(dir)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.Mode().Perm()).To(Equal(wantMode))
	})
})

var _ = Describe("SetupLogRotation", func() {
	BeforeEach(func() {
		if flag.Lookup("stderrthreshold") == nil {
			klog.InitFlags(nil)
		}
	})

	AfterEach(func() {
		klog.LogToStderr(true)
		klog.SetOutput(os.Stderr)
	})

	It("uses the default log dir when unset", func() {
		cfg := Config{MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 30}
		cleanup, effective, warnings, err := SetupLogRotation(cfg)
		if err != nil {
			Skip(fmt.Sprintf("default log dir %q not usable in this environment: %v", DefaultLogDir, err))
		}
		Expect(cleanup).NotTo(BeNil())
		Expect(effective.LogDir).To(Equal(DefaultLogDir))
		Expect(warnings).To(BeEmpty())
		cleanup()
	})

	It("enables rotation for a writable /var/log path", func() {
		dir := varLogTempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
		cleanup, effective, warnings, err := SetupLogRotation(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(cleanup).NotTo(BeNil())
		Expect(effective).To(Equal(cfg))
		Expect(warnings).To(BeEmpty())

		klog.Info("setup-test: logged line via klog")
		cleanup()

		data, err := os.ReadFile(filepath.Join(dir, "sriovdp.log"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("setup-test: logged line via klog"))
	})

	It("rejects a path outside /var/log", func() {
		dir := GinkgoT().TempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
		cleanup, _, _, err := SetupLogRotation(cfg)
		Expect(err).To(HaveOccurred())
		Expect(cleanup).To(BeNil())
	})

	It("normalizes invalid limits to defaults", func() {
		dir := varLogTempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 0, MaxFiles: 5, MaxAgeDays: 30}
		cleanup, effective, warnings, err := SetupLogRotation(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(cleanup).NotTo(BeNil())
		Expect(effective.MaxSizeMB).To(Equal(DefaultMaxSizeMB))
		Expect(warnings).To(HaveLen(1))
		Expect(warnings[0]).To(ContainSubstring("invalid --log-max-size=0"))

		data, readErr := os.ReadFile(filepath.Join(dir, "sriovdp.log"))
		Expect(readErr).NotTo(HaveOccurred())
		logText := string(data)
		Expect(logText).To(ContainSubstring("invalid --log-max-size=0"))
		Expect(logText).NotTo(ContainSubstring("maxSizeMB=0"))

		cleanup()
	})

	It("writes the log file at the configured directory", func() {
		dir := varLogTempDir()

		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, MaxAgeDays: 3}
		cleanup, _, _, err := SetupLogRotation(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(cleanup).NotTo(BeNil())

		klog.Info("config passthrough test")
		cleanup()

		_, statErr := os.Stat(filepath.Join(dir, "sriovdp.log"))
		Expect(statErr).NotTo(HaveOccurred())
	})

	It("emits each error once on stderr when tee owns console logging", func() {
		dir := varLogTempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}

		oldStderr := os.Stderr
		reader, writer, err := os.Pipe()
		Expect(err).NotTo(HaveOccurred())
		os.Stderr = writer

		cleanup, _, _, setupErr := SetupLogRotation(cfg)
		Expect(setupErr).NotTo(HaveOccurred())

		const marker = "dedup-console-test-marker"
		klog.Error(marker)
		klog.Flush()

		Expect(writer.Close()).To(Succeed())
		os.Stderr = oldStderr

		var captured bytes.Buffer
		_, err = io.Copy(&captured, reader)
		Expect(err).NotTo(HaveOccurred())
		Expect(strings.Count(captured.String(), marker)).To(Equal(1))

		cleanup()
	})
})

var _ = Describe("StartupInfo", func() {
	It("should contain all required fields for startup header", func() {
		info := StartupInfo{
			ConfigFile:     "/etc/pcidp/config.json",
			ResourcePrefix: "intel.com",
			UseCdi:         true,
			Log: Config{
				LogDir:     "/var/log/sriovdp",
				MaxSizeMB:  100,
				MaxFiles:   5,
				MaxAgeDays: 30,
			},
		}

		Expect(info.ConfigFile).To(Equal("/etc/pcidp/config.json"))
		Expect(info.ResourcePrefix).To(Equal("intel.com"))
		Expect(info.UseCdi).To(BeTrue())
		Expect(info.Log.LogDir).To(Equal("/var/log/sriovdp"))
		Expect(info.Log.MaxSizeMB).To(Equal(100))
		Expect(info.Log.MaxFiles).To(Equal(5))
		Expect(info.Log.MaxAgeDays).To(Equal(30))
	})

	It("requires NODE_NAME to be set", func() {
		os.Unsetenv("NODE_NAME")
		err := LogStartupHeader(StartupInfo{
			ConfigFile:     "/etc/pcidp/config.json",
			ResourcePrefix: "intel.com",
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("NODE_NAME"))
	})

	It("prints effective rotation limits in the banner, not pre-normalization zeros", func() {
		dir := varLogTempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 0, MaxFiles: 5, MaxAgeDays: 30}

		cleanup, effective, _, err := SetupLogRotation(cfg)
		Expect(err).NotTo(HaveOccurred())
		defer cleanup()

		Expect(os.Setenv("NODE_NAME", "test-node.example")).To(Succeed())
		defer os.Unsetenv("NODE_NAME")

		err = LogStartupHeader(StartupInfo{
			ConfigFile:     "/etc/pcidp/config.json",
			ResourcePrefix: "intel.com",
			Log:            effective,
		})
		Expect(err).NotTo(HaveOccurred())
		klog.Flush()

		data, readErr := os.ReadFile(filepath.Join(dir, "sriovdp.log"))
		Expect(readErr).NotTo(HaveOccurred())
		logText := string(data)
		Expect(logText).To(ContainSubstring(fmt.Sprintf("maxSizeMB=%d", DefaultMaxSizeMB)))
		Expect(logText).NotTo(ContainSubstring("maxSizeMB=0"))
	})
})
