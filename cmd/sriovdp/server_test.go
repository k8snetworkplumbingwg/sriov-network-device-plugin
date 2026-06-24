// Copyright 2018 Intel Corp. All Rights Reserved.
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
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func TestNewHealthCheckServer(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)
	require.NotNil(t, hcs)

	// Verify socket path is constructed correctly
	expectedPath := filepath.Join(tmpDir, "sriovdp.sock")
	impl := hcs.(*healthCheckServerImpl)
	assert.Equal(t, expectedPath, impl.socketPath)
}

func TestHealthCheckServerStart(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	err := hcs.Start()
	require.NoError(t, err)
	defer hcs.Stop() //nolint:errcheck

	// Verify socket file was created
	socketPath := filepath.Join(tmpDir, "sriovdp.sock")
	_, err = os.Stat(socketPath)
	require.NoError(t, err, "socket file should exist")
}

func TestHealthCheckServerStop(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	err := hcs.Start()
	require.NoError(t, err)

	// Verify socket exists
	socketPath := filepath.Join(tmpDir, "sriovdp.sock")
	_, err = os.Stat(socketPath)
	require.NoError(t, err)

	// Stop server
	err = hcs.Stop()
	require.NoError(t, err)

	// Verify socket is removed
	_, err = os.Stat(socketPath)
	assert.Error(t, err, "socket should be removed after stop")
}

func TestHealthCheckServerLivenessCheck(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	err := hcs.Start()
	require.NoError(t, err)
	defer hcs.Stop() //nolint:errcheck

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect and send liveness request
	socketPath := filepath.Join(tmpDir, "sriovdp.sock")
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Send request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(map[string]string{"check": "liveness"})
	require.NoError(t, err)

	// Read response
	decoder := json.NewDecoder(conn)
	var resp LivenessResponse
	err = decoder.Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "alive", resp.Status)
}

func TestHealthCheckServerReadinessCheckEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	err := hcs.Start()
	require.NoError(t, err)
	defer hcs.Stop() //nolint:errcheck

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Connect and send readiness request
	socketPath := filepath.Join(tmpDir, "sriovdp.sock")
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Send request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(map[string]string{"check": "readiness"})
	require.NoError(t, err)

	// Read response
	decoder := json.NewDecoder(conn)
	var resp ReadinessResponse
	err = decoder.Decode(&resp)
	require.NoError(t, err)

	// No devices added yet
	assert.False(t, resp.Ready)
	assert.Equal(t, 0, resp.HealthyCount)
	assert.Equal(t, 0, resp.TotalCount)
	assert.Equal(t, 0, resp.HealthyServers)
}

func TestHealthCheckServerReadinessCheckWithDevices(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	err := hcs.Start()
	require.NoError(t, err)
	defer hcs.Stop() //nolint:errcheck

	// Update device status for server 0
	devices := map[string]*pluginapi.Device{
		"dev0": {
			ID:     "dev0",
			Health: pluginapi.Healthy,
		},
		"dev1": {
			ID:     "dev1",
			Health: pluginapi.Unhealthy,
		},
	}
	hcs.UpdateDeviceStatus(0, devices)

	// Give server time to process
	time.Sleep(100 * time.Millisecond)

	// Connect and send readiness request
	socketPath := filepath.Join(tmpDir, "sriovdp.sock")
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	require.NoError(t, err)
	defer conn.Close()

	// Send request
	encoder := json.NewEncoder(conn)
	err = encoder.Encode(map[string]string{"check": "readiness"})
	require.NoError(t, err)

	// Read response
	decoder := json.NewDecoder(conn)
	var resp ReadinessResponse
	err = decoder.Decode(&resp)
	require.NoError(t, err)

	// Should be ready with 1 healthy device
	assert.True(t, resp.Ready)
	assert.Equal(t, 1, resp.HealthyCount)
	assert.Equal(t, 2, resp.TotalCount)
	assert.Equal(t, 1, resp.HealthyServers)
}

func TestHealthCheckServerConcurrentConnections(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	err := hcs.Start()
	require.NoError(t, err)
	defer hcs.Stop() //nolint:errcheck

	socketPath := filepath.Join(tmpDir, "sriovdp.sock")

	// Test 5 concurrent connections
	for i := 0; i < 5; i++ {
		go func(_ int) {
			conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
			require.NoError(t, err)
			defer conn.Close()

			encoder := json.NewEncoder(conn)
			err = encoder.Encode(map[string]string{"check": "liveness"})
			require.NoError(t, err)

			decoder := json.NewDecoder(conn)
			var resp LivenessResponse
			err = decoder.Decode(&resp)
			require.NoError(t, err)
			assert.Equal(t, "alive", resp.Status)
		}(i)
	}

	// Wait for all goroutines
	time.Sleep(500 * time.Millisecond)
}

func TestHealthCheckServerUpdateDeviceStatus(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)

	devices := map[string]*pluginapi.Device{
		"dev0": {
			ID:     "dev0",
			Health: pluginapi.Healthy,
		},
	}

	// Update status for server 0
	hcs.UpdateDeviceStatus(0, devices)

	// Verify status was updated
	impl := hcs.(*healthCheckServerImpl)
	impl.mu.RLock()
	defer impl.mu.RUnlock()

	assert.Contains(t, impl.serverDevices, 0)
	assert.Contains(t, impl.serverDevices[0], "dev0")
	assert.Equal(t, pluginapi.Healthy, impl.serverDevices[0]["dev0"].Health)
}

func TestHealthCheckServerComputeStatus(t *testing.T) {
	tmpDir := t.TempDir()
	hcs := NewHealthCheckServer(tmpDir)
	impl := hcs.(*healthCheckServerImpl)

	// Add devices for server 0
	impl.UpdateDeviceStatus(0, map[string]*pluginapi.Device{
		"dev0": {ID: "dev0", Health: pluginapi.Healthy},
		"dev1": {ID: "dev1", Health: pluginapi.Unhealthy},
	})

	// Add devices for server 1
	impl.UpdateDeviceStatus(1, map[string]*pluginapi.Device{
		"dev2": {ID: "dev2", Health: pluginapi.Healthy},
	})

	// Compute status
	impl.mu.RLock()
	healthyCount, totalCount, healthyServers := impl.computeStatus()
	impl.mu.RUnlock()

	assert.Equal(t, 2, healthyCount, "should have 2 healthy devices")
	assert.Equal(t, 3, totalCount, "should have 3 total devices")
	assert.Equal(t, 2, healthyServers, "should have 2 healthy servers")
}
