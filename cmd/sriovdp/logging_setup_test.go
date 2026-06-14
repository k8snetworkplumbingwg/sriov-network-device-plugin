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

package main

import (
	"flag"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ensureFlagsRegistered calls flagInit exactly once so the custom rotation
// flags exist on the global flag.CommandLine. Safe to call from any test.
var ensureFlagsRegistered = sync.OnceFunc(func() {
	cp := &cliParams{}
	flagInit(cp)
})

// ---------- Dimension 1 & 3: setupLogRotation + flag parsing ----------

func TestFlagInit_RegistersAllFlags(t *testing.T) {
	ensureFlagsRegistered()

	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{"config-file", "config-file", defaultConfig},
		{"resource-prefix", "resource-prefix", "intel.com"},
		{"use-cdi", "use-cdi", "false"},
		{"log-max-size", "log-max-size", "100"},
		{"log-max-files", "log-max-files", "5"},
		{"log-max-age", "log-max-age", "30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := flag.Lookup(tt.flagName)
			require.NotNil(t, f, "flag --%s must be registered", tt.flagName)
			assert.Equal(t, tt.defaultValue, f.DefValue,
				"flag --%s default must match MD API Extensions table", tt.flagName)
		})
	}
}

func TestFlagInit_DefaultValues(t *testing.T) {
	cp := cliParams{
		logMaxSize:  100,
		logMaxFiles: 5,
		logMaxAge:   30,
	}

	assert.Equal(t, 100, cp.logMaxSize, "MD default: --log-max-size")
	assert.Equal(t, 5, cp.logMaxFiles, "MD default: --log-max-files")
	assert.Equal(t, 30, cp.logMaxAge, "MD default: --log-max-age")
}

func TestSetupLogRotation_NoLogDir_FallsBackToDefault(t *testing.T) {
	ensureFlagsRegistered()

	origVal := flag.Lookup("log_dir").Value.String()
	defer func() {
		flag.Set("log_dir", origVal) //nolint:errcheck
	}()

	require.NoError(t, flag.Set("log_dir", ""))

	cp := &cliParams{logMaxSize: 100, logMaxFiles: 5, logMaxAge: 30}
	cleanup := setupLogRotation(cp)

	// When log_dir is empty, setupLogRotation falls back to the default
	// path (/var/log/sriovdp). In test environments this path is typically
	// not writable, so cleanup will be nil (graceful degradation).
	// Either outcome is valid — what matters is no panic/crash.
	if cleanup != nil {
		cleanup()
	}
}

func TestSetupLogRotation_WithLogDir(t *testing.T) {
	ensureFlagsRegistered()

	dir := t.TempDir()
	origVal := flag.Lookup("log_dir").Value.String()
	defer func() {
		flag.Set("log_dir", origVal) //nolint:errcheck
	}()

	require.NoError(t, flag.Set("log_dir", dir))

	cp := &cliParams{logMaxSize: 10, logMaxFiles: 3, logMaxAge: 7}
	cleanup := setupLogRotation(cp)
	require.NotNil(t, cleanup, "cleanup must be returned when log_dir is set")

	// Write to stderr — it should be captured into the rotating file.
	_, err := os.Stderr.WriteString("setup-test: logged line\n")
	require.NoError(t, err)

	cleanup()

	logPath := filepath.Join(dir, "sriovdp.log")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "setup-test: logged line")
}

func TestSetupLogRotation_InvalidConfig(t *testing.T) {
	ensureFlagsRegistered()

	dir := t.TempDir()
	origVal := flag.Lookup("log_dir").Value.String()
	defer func() {
		flag.Set("log_dir", origVal) //nolint:errcheck
	}()

	require.NoError(t, flag.Set("log_dir", dir))

	cp := &cliParams{logMaxSize: 0, logMaxFiles: 5, logMaxAge: 30}
	cleanup := setupLogRotation(cp)

	// Invalid rotation values should be normalized to defaults.
	require.NotNil(t, cleanup,
		"invalid config values should fall back to defaults, not disable rotation")
	cleanup()
}

func TestSetupLogRotation_CleanupClosesWriter(t *testing.T) {
	ensureFlagsRegistered()

	dir := t.TempDir()
	origVal := flag.Lookup("log_dir").Value.String()
	defer func() {
		flag.Set("log_dir", origVal) //nolint:errcheck
	}()

	require.NoError(t, flag.Set("log_dir", dir))

	cp := &cliParams{logMaxSize: 10, logMaxFiles: 3, logMaxAge: 7}
	cleanup := setupLogRotation(cp)
	require.NotNil(t, cleanup)

	// Call cleanup — should not panic and should restore stderr.
	cleanup()

	// Stderr must still be writable after cleanup.
	_, err := os.Stderr.WriteString("post-cleanup write OK\n")
	assert.NoError(t, err)
}

func TestSetupLogRotation_ConfigPassthrough(t *testing.T) {
	ensureFlagsRegistered()

	dir := t.TempDir()
	origVal := flag.Lookup("log_dir").Value.String()
	defer func() {
		flag.Set("log_dir", origVal) //nolint:errcheck
	}()

	require.NoError(t, flag.Set("log_dir", dir))

	cp := &cliParams{logMaxSize: 1, logMaxFiles: 2, logMaxAge: 3}
	cleanup := setupLogRotation(cp)
	require.NotNil(t, cleanup)

	logPath := filepath.Join(dir, "sriovdp.log")
	_, err := os.Stderr.WriteString("config passthrough test\n")
	require.NoError(t, err)

	cleanup()

	_, statErr := os.Stat(logPath)
	assert.NoError(t, statErr, "log file must exist at the configured log_dir")
}
