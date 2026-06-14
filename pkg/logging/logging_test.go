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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/natefinch/lumberjack.v2"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "/var/log/sriovdp", cfg.LogDir)
	assert.Equal(t, 100, cfg.MaxSizeMB)
	assert.Equal(t, 5, cfg.MaxFiles)
	assert.Equal(t, 30, cfg.MaxAgeDays)
	assert.True(t, cfg.Compress)
	_, _, err := ResolveConfig(cfg)
	assert.NoError(t, err)
}

func TestResolveConfig(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantErr    string
		wantWarns  int
		expectSame bool
	}{
		{
			name:      "empty LogDir falls back to default",
			cfg:       Config{LogDir: "", MaxSizeMB: 100, MaxFiles: 5},
			wantWarns: 0,
		},
		{
			name:      "zero MaxSizeMB falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: 0, MaxFiles: 5},
			wantWarns: 1,
		},
		{
			name:      "negative MaxSizeMB falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: -1, MaxFiles: 5},
			wantWarns: 1,
		},
		{
			name:      "MaxSizeMB exceeds limit falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: 2000, MaxFiles: 5},
			wantWarns: 1,
		},
		{
			name:      "negative MaxFiles falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: -1},
			wantWarns: 1,
		},
		{
			name:      "MaxFiles exceeds limit falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 200},
			wantWarns: 1,
		},
		{
			name:      "negative MaxAgeDays falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: -1},
			wantWarns: 1,
		},
		{
			name:      "MaxAgeDays exceeds limit falls back to default",
			cfg:       Config{LogDir: "/tmp", MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 500},
			wantWarns: 1,
		},
		{
			name:       "valid values remain unchanged",
			cfg:        Config{LogDir: "/tmp", MaxSizeMB: 50, MaxFiles: 0, MaxAgeDays: 0, Compress: true},
			expectSame: true,
		},
		{
			name:       "valid production config",
			cfg:        Config{LogDir: "/var/log/sriovdp", MaxSizeMB: 100, MaxFiles: 5, MaxAgeDays: 30, Compress: true},
			expectSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, warnings, err := ResolveConfig(tt.cfg)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
				assert.Len(t, warnings, tt.wantWarns)
				if tt.expectSame {
					assert.Equal(t, tt.cfg, got)
				}
			}
		})
	}
}

func TestResolveConfig_AppliesFallbacks(t *testing.T) {
	cfg := Config{
		LogDir:     "",
		MaxSizeMB:  0,
		MaxFiles:   -1,
		MaxAgeDays: 999,
		Compress:   false,
	}

	normalized, warnings, err := ResolveConfig(cfg)
	def := DefaultConfig()
	require.NoError(t, err)

	assert.Equal(t, def.LogDir, normalized.LogDir)
	assert.Equal(t, def.MaxSizeMB, normalized.MaxSizeMB)
	assert.Equal(t, def.MaxFiles, normalized.MaxFiles)
	assert.Equal(t, def.MaxAgeDays, normalized.MaxAgeDays)
	assert.True(t, normalized.Compress)
	assert.Len(t, warnings, 3)
}

func TestResolveConfig_LeavesValidValues(t *testing.T) {
	cfg := Config{
		LogDir:     "/tmp/sriovdp-test",
		MaxSizeMB:  12,
		MaxFiles:   4,
		MaxAgeDays: 7,
		Compress:   true,
	}

	normalized, warnings, err := ResolveConfig(cfg)
	require.NoError(t, err)

	assert.Equal(t, cfg, normalized)
	assert.Empty(t, warnings)
}

func TestNewRotatingWriter_LumberjackConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		LogDir:     dir,
		MaxSizeMB:  42,
		MaxFiles:   7,
		MaxAgeDays: 14,
		Compress:   true,
	}

	w, _, err := NewRotatingWriter(cfg)
	require.NoError(t, err)
	require.NotNil(t, w)

	lj, ok := w.(*lumberjack.Logger)
	require.True(t, ok, "writer must be a *lumberjack.Logger")

	assert.Equal(t, filepath.Join(dir, logFileName), lj.Filename)
	assert.Equal(t, 42, lj.MaxSize)
	assert.Equal(t, 7, lj.MaxBackups)
	assert.Equal(t, 14, lj.MaxAge)
	assert.True(t, lj.Compress)
}

func TestNewRotatingWriter_CreatesFileOnWrite(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}

	w, _, err := NewRotatingWriter(cfg)
	require.NoError(t, err)

	_, writeErr := w.Write([]byte("hello from rotating writer\n"))
	assert.NoError(t, writeErr)
	assert.NoError(t, w.Close())

	logPath := filepath.Join(dir, logFileName)
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "hello from rotating writer")
}

func TestNewRotatingWriter_InvalidConfig(t *testing.T) {
	cfg := Config{LogDir: "/proc/1/fdinfo", MaxSizeMB: 10, MaxFiles: 3, MaxAgeDays: 7}
	w, _, err := NewRotatingWriter(cfg)
	require.Error(t, err)
	assert.Nil(t, w)
	assert.Contains(t, err.Error(), "log directory")
}

func TestNewRotatingWriter_UnwritableDir(t *testing.T) {
	cfg := Config{LogDir: "/proc/1/fdinfo", MaxSizeMB: 10, MaxFiles: 3}

	w, _, err := NewRotatingWriter(cfg)
	assert.Error(t, err, "unwritable directory must fail at creation time")
	assert.Nil(t, w)
}

func TestCaptureStderr_NilWriter(t *testing.T) {
	cleanup, err := CaptureStderr(nil)
	assert.Error(t, err)
	assert.Nil(t, cleanup)
	assert.Contains(t, err.Error(), "writer must not be nil")
}

func TestCaptureStderr_TeesToBothDestinations(t *testing.T) {
	var buf bytes.Buffer
	cleanup, err := CaptureStderr(&buf)
	require.NoError(t, err)
	require.NotNil(t, cleanup)

	msg := "tee-test: this should appear in both destinations\n"
	_, writeErr := os.Stderr.WriteString(msg)
	require.NoError(t, writeErr)

	cleanup()

	assert.Contains(t, buf.String(), "tee-test: this should appear in both destinations")
}

func TestCaptureStderr_RestoresStderrAfterCleanup(t *testing.T) {
	var buf bytes.Buffer
	cleanup, err := CaptureStderr(&buf)
	require.NoError(t, err)

	cleanup()

	_, writeErr := os.Stderr.WriteString("post-restore write\n")
	assert.NoError(t, writeErr, "stderr must be writable after cleanup")
}

func TestCaptureStderr_CleanupDrainsBufferedData(t *testing.T) {
	var buf bytes.Buffer
	cleanup, err := CaptureStderr(&buf)
	require.NoError(t, err)

	payload := strings.Repeat("X", 64*1024) + "\n"
	_, writeErr := os.Stderr.WriteString(payload)
	require.NoError(t, writeErr)

	cleanup()

	assert.GreaterOrEqual(t, buf.Len(), 64*1024,
		"all buffered data must be drained before cleanup returns")
}

func TestCaptureStderr_MultipleCycles(t *testing.T) {
	for i := 0; i < 3; i++ {
		var buf bytes.Buffer
		cleanup, err := CaptureStderr(&buf)
		require.NoError(t, err, "cycle %d", i)

		_, writeErr := os.Stderr.WriteString("cycle message\n")
		require.NoError(t, writeErr)

		cleanup()

		assert.Contains(t, buf.String(), "cycle message", "cycle %d", i)
	}

	_, writeErr := os.Stderr.WriteString("after all cycles\n")
	assert.NoError(t, writeErr)
}

func TestCaptureStderr_IntegrationWithRotatingWriter(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, MaxAgeDays: 1, Compress: false}
	w, _, err := NewRotatingWriter(cfg)
	require.NoError(t, err)

	cleanup, err := CaptureStderr(w)
	require.NoError(t, err)

	msg := "integration test: log line via stderr capture\n"
	_, writeErr := os.Stderr.WriteString(msg)
	require.NoError(t, writeErr)

	cleanup()
	assert.NoError(t, w.Close())

	logPath := filepath.Join(dir, logFileName)
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "integration test: log line via stderr capture")
}

func TestRotation_RetainsMaxFiles(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 2, Compress: false}

	w, _, err := NewRotatingWriter(cfg)
	require.NoError(t, err)

	chunk := strings.Repeat("B", 1024) + "\n"
	for i := 0; i < 4096; i++ {
		_, writeErr := w.Write([]byte(chunk))
		require.NoError(t, writeErr)
	}
	require.NoError(t, w.Close())

	var fileCount int
	for attempts := 0; attempts < 20; attempts++ {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		fileCount = 0
		for _, e := range entries {
			if !e.IsDir() {
				fileCount++
			}
		}
		if fileCount <= 3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	assert.LessOrEqual(t, fileCount, 3, "MaxBackups(2) + 1 active = 3 max")
	assert.GreaterOrEqual(t, fileCount, 2, "at least one backup must exist")
}

func TestRotation_Compressed(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{LogDir: dir, MaxSizeMB: 1, MaxFiles: 3, Compress: true}

	w, _, err := NewRotatingWriter(cfg)
	require.NoError(t, err)

	chunk := strings.Repeat("C", 1024) + "\n"
	for i := 0; i < 2048; i++ {
		_, writeErr := w.Write([]byte(chunk))
		require.NoError(t, writeErr)
	}
	require.NoError(t, w.Close())

	var hasGz bool
	for attempts := 0; attempts < 20; attempts++ {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".gz") {
				hasGz = true
				break
			}
		}
		if hasGz {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	assert.True(t, hasGz, "compressed backup file (.gz) must exist")
}

func TestGracefulShutdown_PreservesLogs(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{LogDir: dir, MaxSizeMB: 10, MaxFiles: 3, Compress: false}

	w, _, err := NewRotatingWriter(cfg)
	require.NoError(t, err)

	cleanup, err := CaptureStderr(w)
	require.NoError(t, err)

	_, err = fmt.Fprintln(os.Stderr, "pre-shutdown: server stopping")
	require.NoError(t, err)

	cleanup()
	require.NoError(t, w.Close())

	logPath := filepath.Join(dir, logFileName)
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "pre-shutdown: server stopping")
}
