// Copyright 2026 Intel Corp. All Rights Reserved.
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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
	"golang.org/x/sys/unix"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	allowedLogBase    = "/var/log"
	defaultLogDir     = "/var/log/sriovdp"
	defaultMaxSizeMB  = 100
	defaultMaxFiles   = 5
	defaultMaxAge     = 30
	logFileName       = "sriovdp.log"
	logDirPerms       = 0750
	maxAllowedSizeMB  = 1024 // 1GB
	maxAllowedFiles   = 100
	maxAllowedAgeDays = 365
)

type Config struct {
	LogDir     string
	MaxSizeMB  int
	MaxFiles   int
	MaxAgeDays int
	Compress   bool
}

type StartupInfo struct {
	ConfigFile      string
	ResourcePrefix  string
	UseCdi          bool
}

func LogStartupHeader(info StartupInfo) {
	separator := strings.Repeat("=", 80)

	podName := os.Getenv("POD_NAME")
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	if podName == "" {
		podName = hostname
	}

	glog.Infof("%s", separator)
	glog.Infof("SR-IOV Network Device Plugin starting")
	glog.Infof("Pod: %s | Start time: %s",
		podName, time.Now().Format(time.RFC3339))
	glog.Infof("Config file: %s | Resource prefix: %s | CDI: %v",
		info.ConfigFile, info.ResourcePrefix, info.UseCdi)
	glog.Infof("%s", separator)
}

// SetupLogRotation enables rotating file logs via stderr capture.
// Prefer glog --log_dir when set. Returns cleanup to defer, or nil on failure.
func SetupLogRotation(cfg Config) func() {
	if f := flag.Lookup("log_dir"); f != nil {
		if v := f.Value.String(); v != "" {
			cfg.LogDir = v
		}
	}

	logDir := cfg.LogDir
	if logDir == "" {
		logDir = DefaultConfig().LogDir
	}
	cleanDir, err := ValidateLogDir(logDir)
	if err != nil {
		glog.Errorf("rejected log directory: %v — continuing without rotation", err)
		return nil
	}
	cfg.LogDir = cleanDir

	resolved, warnings, err := ResolveConfig(cfg)
	for _, w := range warnings {
		glog.Warning(w)
	}
	if err != nil {
		glog.Errorf("failed to resolve log rotation config: %v — continuing without rotation", err)
		return nil
	}

	rotWriter, _, err := NewRotatingWriter(resolved)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sriovdp: log rotation disabled; cannot use log directory %q: %v\n", resolved.LogDir, err) //nolint:errcheck
		glog.Errorf("failed to create log rotation writer: %v — continuing without rotation", err)
		return nil
	}

	cleanup, err := CaptureStderr(rotWriter)
	if err != nil {
		glog.Errorf("failed to capture stderr for log rotation: %v — continuing without rotation", err)
		if closeErr := rotWriter.Close(); closeErr != nil {
			glog.Warningf("failed to close rotation writer after capture failure: %v", closeErr)
		}
		return nil
	}

	glog.Infof("Log rotation enabled: dir=%s maxSize=%dMB maxFiles=%d maxAge=%d compress=%v",
		resolved.LogDir, resolved.MaxSizeMB, resolved.MaxFiles, resolved.MaxAgeDays, resolved.Compress)

	return func() {
		cleanup()
		if err := rotWriter.Close(); err != nil {
			glog.Warningf("failed to close rotation writer: %v", err)
		}
	}
}

// ValidateLogDir ensures dir is under /var/log, creates it, resolves symlinks,
// and returns the real path.
func ValidateLogDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("log directory must not be empty")
	}
	if strings.Contains(dir, "~") {
		return "", fmt.Errorf("log directory %q must not contain '~'", dir)
	}
	for _, part := range strings.Split(filepath.ToSlash(dir), "/") {
		if part == ".." {
			return "", fmt.Errorf("log directory %q must not contain '..'", dir)
		}
	}

	clean := filepath.Clean(dir)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("log directory %q must be an absolute path", dir)
	}
	if err := checkAllowedLogBase(clean); err != nil {
		return "", fmt.Errorf("log directory %q is outside the allowed base %q", dir, allowedLogBase)
	}

	if err := os.MkdirAll(clean, logDirPerms); err != nil {
		return "", fmt.Errorf("cannot create log directory %q: %w", clean, err)
	}

	resolved, err := filepath.EvalSymlinks(clean)
	if err != nil {
		return "", fmt.Errorf("cannot resolve log directory %q: %w", clean, err)
	}
	resolved = filepath.Clean(resolved)
	if err := checkAllowedLogBase(resolved); err != nil {
		return "", fmt.Errorf("log directory %q resolves to %q which is outside the allowed base %q", dir, resolved, allowedLogBase)
	}
	return resolved, nil
}

func checkAllowedLogBase(path string) error {
	if path != allowedLogBase && !strings.HasPrefix(path, allowedLogBase+"/") {
		return fmt.Errorf("outside allowed base")
	}
	return nil
}

func ResolveConfig(cfg Config) (Config, []string, error) {
	def := DefaultConfig()
	normalized := cfg
	warnings := []string{}

	if normalized.LogDir == "" {
		normalized.LogDir = def.LogDir
	}
	if normalized.MaxSizeMB <= 0 || normalized.MaxSizeMB > maxAllowedSizeMB {
		warnings = append(warnings,
			fmt.Sprintf("invalid --log-max-size=%d; using default %d", normalized.MaxSizeMB, def.MaxSizeMB))
		normalized.MaxSizeMB = def.MaxSizeMB
	}
	if normalized.MaxFiles < 0 || normalized.MaxFiles > maxAllowedFiles {
		warnings = append(warnings,
			fmt.Sprintf("invalid --log-max-files=%d; using default %d", normalized.MaxFiles, def.MaxFiles))
		normalized.MaxFiles = def.MaxFiles
	}
	if normalized.MaxAgeDays < 0 || normalized.MaxAgeDays > maxAllowedAgeDays {
		warnings = append(warnings,
			fmt.Sprintf("invalid --log-max-age=%d; using default %d", normalized.MaxAgeDays, def.MaxAgeDays))
		normalized.MaxAgeDays = def.MaxAgeDays
	}

	normalized.Compress = true

	return normalized, warnings, nil
}

// DefaultConfig returns production defaults.
func DefaultConfig() Config {
	return Config{
		LogDir:     defaultLogDir,
		MaxSizeMB:  defaultMaxSizeMB,
		MaxFiles:   defaultMaxFiles,
		MaxAgeDays: defaultMaxAge,
		Compress:   true,
	}
}

// NewRotatingWriter returns a lumberjack writer and any config warnings.
// Caller must Close the writer.
func NewRotatingWriter(cfg Config) (io.WriteCloser, []string, error) {
	resolvedCfg, warnings, err := ResolveConfig(cfg)
	if err != nil {
		return nil, warnings, fmt.Errorf("invalid logging config: %w", err)
	}

	if err := os.MkdirAll(resolvedCfg.LogDir, logDirPerms); err != nil {
		return nil, warnings, fmt.Errorf("cannot create log directory %q: %w", resolvedCfg.LogDir, err)
	}

	probe, err := os.CreateTemp(resolvedCfg.LogDir, ".probe-*")
	if err != nil {
		return nil, warnings, fmt.Errorf("log directory %q is not writable: %w", resolvedCfg.LogDir, err)
	}
	if err := probe.Close(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to close probe file: %v", err))
	}
	if err := os.Remove(probe.Name()); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to remove probe file %q: %v", probe.Name(), err))
	}

	return &lumberjack.Logger{
		Filename:   filepath.Join(resolvedCfg.LogDir, logFileName),
		MaxSize:    resolvedCfg.MaxSizeMB,
		MaxBackups: resolvedCfg.MaxFiles,
		MaxAge:     resolvedCfg.MaxAgeDays,
		Compress:   resolvedCfg.Compress,
	}, warnings, nil
}

// resilientTeeWriter writes to primary + secondary; never fails the drain.
type resilientTeeWriter struct {
	primary   io.Writer // original stderr (or Discard after failure)
	secondary io.Writer // rotating log (best-effort)
}

func (t *resilientTeeWriter) Write(p []byte) (int, error) {
	if _, err := t.primary.Write(p); err != nil {
		t.primary = io.Discard
	}
	_, _ = t.secondary.Write(p)
	return len(p), nil // keep pipe drain alive
}

// CaptureStderr tees fd 2 to original stderr and w. Defer cleanup to restore fd 2.
func CaptureStderr(w io.Writer) (cleanup func(), err error) {
	if w == nil {
		return nil, fmt.Errorf("writer must not be nil")
	}

	origFd, err := unix.Dup(int(os.Stderr.Fd()))
	if err != nil {
		return nil, fmt.Errorf("dup stderr: %w", err)
	}

	r, pw, err := os.Pipe()
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("create pipe: %w", err),
			unix.Close(origFd),
		)
	}

	if err := unix.Dup2(int(pw.Fd()), int(os.Stderr.Fd())); err != nil {
		return nil, errors.Join(
			fmt.Errorf("dup2 pipe to stderr: %w", err),
			r.Close(),
			pw.Close(),
			unix.Close(origFd),
		)
	}

	if err := pw.Close(); err != nil {
		return nil, errors.Join(
			fmt.Errorf("close pipe writer: %w", err),
			unix.Dup2(origFd, int(os.Stderr.Fd())),
			r.Close(),
			unix.Close(origFd),
		)
	}

	origFile := os.NewFile(uintptr(origFd), "original-stderr")
	tee := &resilientTeeWriter{primary: origFile, secondary: w}

	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 32*1024)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				_, _ = tee.Write(buf[:n])
			}
			if readErr != nil {
				if readErr != io.EOF {
					_, _ = fmt.Fprintf(origFile, "sriovdp: stderr tee read error: %v\n", readErr)
				}
				return
			}
		}
	}()

	return func() {
		// Restore fd 2; closes pipe write end → drain gets EOF.
		if err := unix.Dup2(origFd, int(os.Stderr.Fd())); err != nil {
			_, _ = fmt.Fprintf(origFile, "sriovdp: failed to restore stderr: %v\n", err)
		}
		<-done // wait for drain before closing reader
		if err := r.Close(); err != nil {
			_, _ = fmt.Fprintf(origFile, "sriovdp: failed to close pipe reader: %v\n", err)
		}
		if err := origFile.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "sriovdp: failed to close original stderr fd: %v\n", err)
		}
	}, nil
}
