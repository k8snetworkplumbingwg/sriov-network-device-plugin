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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	// Socket directory permissions - restrictive for security
	socketDirPermissions = 0o750
	healthCheckDeadline  = 2 * time.Second
)

// Request represents a health check request
type Request struct {
	Check string `json:"check"` // "liveness" or "readiness"
}

// LivenessResponse represents a liveness check response
type LivenessResponse struct {
	Status string `json:"status"` // "alive"
}

// ReadinessResponse represents a readiness check response
type ReadinessResponse struct {
	Ready          bool `json:"ready"`
	HealthyCount   int  `json:"healthy_count"`
	TotalCount     int  `json:"total_count"`
	HealthyServers int  `json:"healthy_servers"`
}

// HealthCheckServer defines the interface for health check server
type HealthCheckServer interface {
	Start() error
	Stop() error
	UpdateDeviceStatus(serverIndex int, devices map[string]*pluginapi.Device)
}

type healthCheckServerImpl struct {
	socketPath    string
	listener      net.Listener
	stopCh        chan struct{}
	mu            sync.RWMutex
	serverDevices map[int]map[string]*pluginapi.Device
}

// NewHealthCheckServer creates a new HealthCheckServer instance
func NewHealthCheckServer(socketDir string) HealthCheckServer {
	socketPath := filepath.Join(socketDir, "sriovdp.sock")
	return &healthCheckServerImpl{
		socketPath:    socketPath,
		stopCh:        make(chan struct{}),
		serverDevices: make(map[int]map[string]*pluginapi.Device),
	}
}

// Start starts the health check server
func (h *healthCheckServerImpl) Start() error {
	// Ensure directory exists
	socketDir := filepath.Dir(h.socketPath)
	if err := os.MkdirAll(socketDir, socketDirPermissions); err != nil {
		return fmt.Errorf("failed to create socket directory %s: %v", socketDir, err)
	}

	// Clean up old socket if it exists
	if err := os.Remove(h.socketPath); err != nil && !os.IsNotExist(err) {
		glog.Warningf("failed to remove old socket %s: %v", h.socketPath, err)
	}

	lis, err := net.Listen("unix", h.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", h.socketPath, err)
	}

	h.listener = lis
	glog.Infof("HealthCheckServer listening on %s", h.socketPath)

	go h.acceptLoop()
	return nil
}

// acceptLoop accepts incoming connections and routes them to handlers
func (h *healthCheckServerImpl) acceptLoop() {
	for {
		select {
		case <-h.stopCh:
			return
		default:
			conn, err := h.listener.Accept()
			if err != nil {
				select {
				case <-h.stopCh:
					return
				default:
					glog.Errorf("Accept error: %v", err)
					continue
				}
			}
			go h.handleConnection(conn)
		}
	}
}

// handleConnection handles a single client connection
func (h *healthCheckServerImpl) handleConnection(conn net.Conn) {
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(healthCheckDeadline)); err != nil {
		glog.Warningf("failed to set deadline: %v", err)
		return
	}

	decoder := json.NewDecoder(conn)
	var req Request

	if err := decoder.Decode(&req); err != nil {
		glog.Warningf("Failed to decode request: %v", err)
		return
	}

	encoder := json.NewEncoder(conn)

	switch req.Check {
	case checkTypeLiveness:
		if err := encoder.Encode(LivenessResponse{Status: "alive"}); err != nil {
			glog.Warningf("failed to encode liveness response: %v", err)
		}
	case checkTypeReadiness:
		h.mu.RLock()
		healthyCount, totalCount, healthyServers := h.computeStatus()
		h.mu.RUnlock()

		if err := encoder.Encode(ReadinessResponse{
			Ready:          healthyCount > 0,
			HealthyCount:   healthyCount,
			TotalCount:     totalCount,
			HealthyServers: healthyServers,
		}); err != nil {
			glog.Warningf("failed to encode readiness response: %v", err)
		}
	default:
		glog.Warningf("Unknown check type: %s", req.Check)
	}
}

// computeStatus computes the aggregated health status across all servers
func (h *healthCheckServerImpl) computeStatus() (healthyCount, totalCount, healthyServers int) {
	for _, devices := range h.serverDevices {
		healthyInServer := 0
		for _, dev := range devices {
			if dev == nil {
				continue
			}
			totalCount++
			if dev.Health == pluginapi.Healthy {
				healthyCount++
				healthyInServer++
			}
		}
		if healthyInServer > 0 {
			healthyServers++
		}
	}
	return
}

// UpdateDeviceStatus updates the device status for a specific server
func (h *healthCheckServerImpl) UpdateDeviceStatus(serverIndex int, devices map[string]*pluginapi.Device) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Create a copy of the devices map to avoid external mutations
	devicesCopy := make(map[string]*pluginapi.Device)
	for k, v := range devices {
		devicesCopy[k] = v
	}
	h.serverDevices[serverIndex] = devicesCopy
}

// Stop stops the health check server and cleans up
func (h *healthCheckServerImpl) Stop() error {
	close(h.stopCh)
	if h.listener != nil {
		if err := h.listener.Close(); err != nil {
			glog.Warningf("failed to close listener: %v", err)
		}
	}
	if err := os.Remove(h.socketPath); err != nil && !os.IsNotExist(err) {
		glog.Warningf("failed to remove socket %s: %v", h.socketPath, err)
	}
	glog.Infof("HealthCheckServer stopped")
	return nil
}
