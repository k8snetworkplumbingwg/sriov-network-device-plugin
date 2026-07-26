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
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/natefinch/lumberjack.v2"
)

type failWriter struct {
	failAfter int
	writes    int
}

func (w *failWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes > w.failAfter {
		return 0, errors.New("primary stderr broken")
	}
	return len(p), nil
}

var _ = Describe("DefaultConfig", func() {
	It("returns production defaults", func() {
		cfg := DefaultConfig()

		Expect(cfg.LogDir).To(Equal("/var/log/sriovdp"))
		Expect(cfg.MaxSizeMB).To(Equal(100))
		Expect(cfg.MaxFiles).To(Equal(5))
		Expect(cfg.MaxAgeDays).To(Equal(30))
		Expect(cfg.Compress).To(BeTrue())
		_, _, err := ResolveConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
	})
})

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
		Entry("valid values remain unchanged",
			Config{LogDir: "/tmp", MaxSizeMB: 50, MaxFiles: 0, MaxAgeDays: 0, Compress: true}, 0, true),
		Entry("valid production config",
			Config{LogDir: "/var/log/sriovdp", MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 30, Compress: true}, 0, true),
	)

	It("applies all fallbacks together", func() {
		cfg := Config{
			LogDir:     "",
			MaxSizeMB:  0,
			MaxFiles:   -1,
			MaxAgeDays: 999,
			Compress:   false,
		}

		normalized, warnings, err := ResolveConfig(cfg)
		def := DefaultConfig()
		Expect(err).NotTo(HaveOccurred())

		Expect(normalized.LogDir).To(Equal(def.LogDir))
		Expect(normalized.MaxSizeMB).To(Equal(def.MaxSizeMB))
		Expect(normalized.MaxFiles).To(Equal(def.MaxFiles))
		Expect(normalized.MaxAgeDays).To(Equal(def.MaxAgeDays))
		Expect(normalized.Compress).To(BeTrue())
		Expect(warnings).To(HaveLen(3))
	})

	It("leaves valid values unchanged", func() {
		cfg := Config{
			LogDir:     "/tmp/sriovdp-test",
			MaxSizeMB:  12,
			MaxFiles:   4,
			MaxAgeDays: 7,
			Compress:   true,
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
			Compress:   true,
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

var _ = Describe("resilientTeeWriter", func() {
	It("keeps draining after primary write failure", func() {
		primary := &failWriter{failAfter: 1}
		var secondary bytes.Buffer
		tee := &resilientTeeWriter{primary: primary, secondary: &secondary}

		n, err := tee.Write([]byte("first\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(6))

		n, err = tee.Write([]byte("second\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(7))

		n, err = tee.Write([]byte("third\n"))
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(6))

		Expect(tee.primary).To(Equal(io.Discard))
		Expect(secondary.String()).To(Equal("first\nsecond\nthird\n"))
		Expect(primary.writes).To(Equal(2))
	})
})

var _ = Describe("CaptureStderr", func() {
	It("rejects a nil writer", func() {
		cleanup, err := CaptureStderr(nil)
		Expect(err).To(HaveOccurred())
		Expect(cleanup).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("writer must not be nil"))
	})

	It("tees stderr to the secondary writer", func() {
		var buf bytes.Buffer
		cleanup, err := CaptureStderr(&buf)
		Expect(err).NotTo(HaveOccurred())
		Expect(cleanup).NotTo(BeNil())

		_, writeErr := os.Stderr.WriteString("tee-test: this should appear in both destinations\n")
		Expect(writeErr).NotTo(HaveOccurred())

		cleanup()
		Expect(buf.String()).To(ContainSubstring("tee-test: this should appear in both destinations"))
	})

	It("restores stderr after cleanup", func() {
		var buf bytes.Buffer
		cleanup, err := CaptureStderr(&buf)
		Expect(err).NotTo(HaveOccurred())

		cleanup()

		_, writeErr := os.Stderr.WriteString("post-restore write\n")
		Expect(writeErr).NotTo(HaveOccurred())
	})

	It("drains buffered data before cleanup returns", func() {
		var buf bytes.Buffer
		cleanup, err := CaptureStderr(&buf)
		Expect(err).NotTo(HaveOccurred())

		payload := strings.Repeat("X", 64*1024) + "\n"
		_, writeErr := os.Stderr.WriteString(payload)
		Expect(writeErr).NotTo(HaveOccurred())

		cleanup()
		Expect(buf.Len()).To(BeNumerically(">=", 64*1024))
	})

	It("supports multiple capture cycles", func() {
		for i := 0; i < 3; i++ {
			var buf bytes.Buffer
			cleanup, err := CaptureStderr(&buf)
			Expect(err).NotTo(HaveOccurred())

			_, writeErr := os.Stderr.WriteString("cycle message\n")
			Expect(writeErr).NotTo(HaveOccurred())

			cleanup()
			Expect(buf.String()).To(ContainSubstring("cycle message"))
		}

		_, writeErr := os.Stderr.WriteString("after all cycles\n")
		Expect(writeErr).NotTo(HaveOccurred())
	})

	It("integrates with a rotating writer", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, MaxAgeDays: 1, Compress: false}
		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		cleanup, err := CaptureStderr(w)
		Expect(err).NotTo(HaveOccurred())

		_, writeErr := os.Stderr.WriteString("integration test: log line via stderr capture\n")
		Expect(writeErr).NotTo(HaveOccurred())

		cleanup()
		Expect(w.Close()).To(Succeed())

		data, err := os.ReadFile(filepath.Join(dir, logFileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("integration test: log line via stderr capture"))
	})
})

var _ = Describe("Rotation", func() {
	It("retains at most MaxFiles backups plus the active file", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, Compress: false}

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

	It("compresses rotated backups when enabled", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 3, Compress: true}

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
	It("preserves logs written before cleanup", func() {
		dir := GinkgoT().TempDir()
		cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, Compress: false}

		w, _, err := NewRotatingWriter(cfg)
		Expect(err).NotTo(HaveOccurred())

		cleanup, err := CaptureStderr(w)
		Expect(err).NotTo(HaveOccurred())

		_, err = fmt.Fprintln(os.Stderr, "pre-shutdown: server stopping")
		Expect(err).NotTo(HaveOccurred())

		cleanup()
		Expect(w.Close()).To(Succeed())

		data, err := os.ReadFile(filepath.Join(dir, logFileName))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(ContainSubstring("pre-shutdown: server stopping"))
	})
})
