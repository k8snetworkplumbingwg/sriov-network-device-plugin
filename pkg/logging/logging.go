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
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
	"k8s.io/klog/v2"
)

const (
	allowedLogBase = "/var/log"

	DefaultLogDir    = allowedLogBase + "/sriovdp"
	DefaultMaxSizeMB = 100
	DefaultMaxFiles  = 5
	DefaultMaxAge    = 30

	logFileName        = "sriovdp.log"
	logDirPerms        = 0750
	maxAllowedSizeMB   = 1024 // 1GB
	maxAllowedFiles    = 100
	maxAllowedAgeDays  = 365
	startupHeaderWidth = 80
)

type Config struct {
	LogDir     string
	MaxSizeMB  int
	MaxFiles   int
	MaxAgeDays int
}

type StartupInfo struct {
	ConfigFile     string
	ResourcePrefix string
	UseCdi         bool
	Log            Config
}

// LogStartupHeader prints a banner with node identity and config summary.
func LogStartupHeader(info StartupInfo) error {
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return fmt.Errorf("NODE_NAME environment variable is not set")
	}

	separator := strings.Repeat("=", startupHeaderWidth)

	klog.Infof("%s", separator)
	klog.Infof("SR-IOV Network Device Plugin starting")
	klog.Infof("Node: %s | Start time: %s",
		nodeName, time.Now().Format(time.RFC3339))
	klog.Infof("Config file: %s | Resource prefix: %s | CDI: %v",
		info.ConfigFile, info.ResourcePrefix, info.UseCdi)

	if info.Log.LogDir != "" {
		klog.Infof("sriovdp/persistent-file-logging: persistent file logging enabled")
		klog.Infof("Path: %s | maxSizeMB=%d maxFiles=%d maxAgeDays=%d compress=true",
			info.Log.LogDir, info.Log.MaxSizeMB, info.Log.MaxFiles, info.Log.MaxAgeDays)
	}
	klog.Infof("%s", separator)
	return nil
}

// teeWriter writes every log line to both console (stderr) and the rotating
// file. File-write errors are reported to stderr so they are never silent.
type teeWriter struct {
	console io.Writer
	file    io.WriteCloser
}

func (t *teeWriter) Write(p []byte) (int, error) {
	n, _ := t.console.Write(p)
	if _, err := t.file.Write(p); err != nil {
		fmt.Fprintf(t.console, "sriovdp: file-log write error: %v\n", err) //nolint:errcheck
	}
	return n, nil
}

// SetupLogRotation connects klog output to both stderr and a rotating log file
// via klog.SetOutput. cfg.LogDir must be under /var/log (the DaemonSet hostPath
// mount). Returns cleanup to defer, or an error when configuration is invalid.
func SetupLogRotation(cfg Config) (func(), error) {
	logDir := cfg.LogDir
	if logDir == "" {
		logDir = DefaultLogDir
	}
	cleanDir, err := ValidateLogDir(logDir)
	if err != nil {
		return nil, fmt.Errorf("rejected log directory: %w", err)
	}
	cfg.LogDir = cleanDir

	resolved, _, err := ResolveConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve log rotation config: %w", err)
	}

	rotWriter, _, err := NewRotatingWriter(resolved)
	if err != nil {
		return nil, fmt.Errorf("cannot use log directory %q: %w", resolved.LogDir, err)
	}

	tee := &teeWriter{console: os.Stderr, file: rotWriter}
	klog.SetOutput(tee)
	klog.LogToStderr(false)
	if f := flag.Lookup("one_output"); f != nil {
		_ = f.Value.Set("true")
	}

	return func() {
		klog.Flush()
		klog.LogToStderr(true)
		if f := flag.Lookup("one_output"); f != nil {
			_ = f.Value.Set("false")
		}
		klog.SetOutput(os.Stderr)
		if err := rotWriter.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "sriovdp: failed to close rotation writer: %v\n", err) //nolint:errcheck
		}
	}, nil
}

var validLogDirRegex = regexp.MustCompile(`^[a-zA-Z0-9_/\-.]+$`)

func validateLogDirInput(dir string) error {
	if dir == "" {
		return fmt.Errorf("log directory must not be empty")
	}
	if !validLogDirRegex.MatchString(dir) {
		return fmt.Errorf("log directory contains invalid characters; only alphanumeric, _, -, /, and . are allowed")
	}
	if strings.HasPrefix(dir, "-") {
		return fmt.Errorf("log directory %q must not start with '-'", dir)
	}
	return nil
}

// ValidateLogDir ensures dir is under /var/log, creates it, resolves symlinks,
// and returns the real path.
func ValidateLogDir(dir string) (string, error) {
	if err := validateLogDirInput(dir); err != nil {
		return "", err
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

	resolved, err := resolveExistingPrefix(clean)
	if err != nil {
		return "", fmt.Errorf("cannot resolve log directory %q: %w", clean, err)
	}
	resolved = filepath.Clean(resolved)
	if err := checkAllowedLogBase(resolved); err != nil {
		return "", fmt.Errorf("log directory %q resolves to %q which is outside the allowed base %q", dir, resolved, allowedLogBase)
	}

	if err := os.MkdirAll(resolved, logDirPerms); err != nil {
		return "", fmt.Errorf("cannot create log directory %q: %w", resolved, err)
	}

	if err := os.Chmod(resolved, logDirPerms); err != nil {
		return "", fmt.Errorf("cannot set permissions on log directory %q: %w", resolved, err)
	}

	return resolved, nil
}

func resolveExistingPrefix(path string) (string, error) {
	path = filepath.Clean(path)

	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved), nil
	}

	parent := filepath.Dir(path)
	if parent == path {
		return "", fmt.Errorf("cannot resolve any component of %q", path)
	}

	resolvedParent, err := resolveExistingPrefix(parent)
	if err != nil {
		return "", err
	}

	return filepath.Join(resolvedParent, filepath.Base(path)), nil
}

func checkAllowedLogBase(path string) error {
	if path != allowedLogBase && !strings.HasPrefix(path, allowedLogBase+"/") {
		return fmt.Errorf("outside allowed base")
	}
	return nil
}

// ResolveConfig normalizes and clamps Config values, returning warnings for
// any fields that were corrected.
func ResolveConfig(cfg Config) (Config, []string, error) {
	normalized := cfg
	warnings := []string{}

	if normalized.LogDir == "" {
		normalized.LogDir = DefaultLogDir
	}
	if normalized.MaxSizeMB <= 0 || normalized.MaxSizeMB > maxAllowedSizeMB {
		warnings = append(warnings,
			fmt.Sprintf("invalid --log-max-size=%d; using default %d", normalized.MaxSizeMB, DefaultMaxSizeMB))
		normalized.MaxSizeMB = DefaultMaxSizeMB
	}
	if normalized.MaxFiles <= 0 || normalized.MaxFiles > maxAllowedFiles {
		warnings = append(warnings,
			fmt.Sprintf("invalid --log-max-files=%d; using default %d", normalized.MaxFiles, DefaultMaxFiles))
		normalized.MaxFiles = DefaultMaxFiles
	}
	if normalized.MaxAgeDays < 0 || normalized.MaxAgeDays > maxAllowedAgeDays {
		warnings = append(warnings,
			fmt.Sprintf("invalid --log-max-age=%d; using default %d", normalized.MaxAgeDays, DefaultMaxAge))
		normalized.MaxAgeDays = DefaultMaxAge
	}

	return normalized, warnings, nil
}

// NewRotatingWriter returns a lumberjack writer. Caller must Close it.
func NewRotatingWriter(cfg Config) (io.WriteCloser, []string, error) {
	var warnings []string

	if err := os.MkdirAll(cfg.LogDir, logDirPerms); err != nil {
		return nil, warnings, fmt.Errorf("cannot create log directory %q: %w", cfg.LogDir, err)
	}

	probe, err := os.CreateTemp(cfg.LogDir, ".probe-*")
	if err != nil {
		return nil, warnings, fmt.Errorf("log directory %q is not writable: %w", cfg.LogDir, err)
	}
	if err := probe.Close(); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to close probe file: %v", err))
	}
	if err := os.Remove(probe.Name()); err != nil {
		warnings = append(warnings, fmt.Sprintf("failed to remove probe file %q: %v", probe.Name(), err))
	}

	return &lumberjack.Logger{
		Filename:   filepath.Join(cfg.LogDir, logFileName),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxFiles,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   true,
	}, warnings, nil
}
