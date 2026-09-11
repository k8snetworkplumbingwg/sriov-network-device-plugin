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
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang/glog"
)

const (
	defaultConfig           = "/etc/pcidp/config.json"
	checkTypeHelp           = "help"
	checkTypeLiveness       = "liveness"
	checkTypeReadiness      = "readiness"
	healthCheckConnTimeout  = 2 * time.Second
	healthCheckReadTimeout  = 5 * time.Second
	healthCheckErrorCode    = 2
	readinessNotReadyCode   = 1
	logVerbosityLevel       = 4
)

// flagInit parse command line flags
func flagInit(cp *cliParams) {
	flag.StringVar(&cp.configFile, "config-file", defaultConfig,
		"JSON device pool config file location")
	flag.StringVar(&cp.resourcePrefix, "resource-prefix", "intel.com",
		"resource name prefix used for K8s extended resource")
	flag.BoolVar(&cp.useCdi, "use-cdi", false,
		"Use Container Device Interface to expose devices in containers")
	flag.StringVar(&cp.healthcheckSocketDir, "healthcheck-socket-dir", "/tmp",
		"Unix socket directory for health/readiness checks")

	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s [flags]              - Start the device plugin daemon\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s health <check>      - Perform health checks\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nFor health check help:\n")
		fmt.Fprintf(os.Stderr, "  %s health -h\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nDaemon flags:\n")
		flag.PrintDefaults()
	}
}

func main() {
	// Apply environment variable override once at startup
	healthcheckSocketDir := "/tmp"
	if envDir := os.Getenv("SRIOV_HEALTHCHECK_SOCKET_DIR"); envDir != "" {
		healthcheckSocketDir = envDir
	}

	// Check if this is a health check invocation (before flag parsing)
	if len(os.Args) > 1 && os.Args[1] == "health" {
		os.Exit(runHealthCheck(os.Args[2:], healthcheckSocketDir))
	}

	cp := &cliParams{}
	cp.healthcheckSocketDir = healthcheckSocketDir
	flagInit(cp)
	flag.Parse()

	rm := newResourceManager(cp)

	glog.Infof("resource manager reading configs")
	if err := rm.readConfig(); err != nil {
		glog.Errorf("error getting resources from file %v", err)
		return
	}

	if len(rm.configList) < 1 {
		glog.Errorf("no resource configuration; exiting")
		return // No config found
	}

	// Validate configs
	if !rm.validConfigs() {
		glog.Fatalf("Exiting.. one or more invalid configuration(s) given")
		return
	}
	glog.Infof("Discovering host devices")
	if err := rm.discoverHostDevices(); err != nil {
		glog.Errorf("error discovering host devices%v", err)
		return
	}

	glog.Infof("Initializing resource servers")
	if err := rm.initServers(); err != nil {
		glog.Errorf("error initializing resource servers %v", err)
		return
	}

	glog.Infof("Starting all servers...")
	if err := rm.startAllServers(); err != nil {
		glog.Errorf("error starting resource servers %v\n", err)
		return
	}
	glog.Infof("All servers started.")
	glog.Infof("Listening for term signals")
	// respond to syscalls for termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Catch termination signals
	sig := <-sigCh
	glog.Infof("Received signal \"%v\", shutting down.", sig)
	if err := rm.stopAllServers(); err != nil {
		glog.Errorf("stopping servers produced error: %s", err.Error())
	}
	if err := rm.cleanupCDISpecs(); err != nil {
		glog.Errorf("cleaning up CDI Specs produced error: %s", err.Error())
	}
}

// printHealthCheckUsage prints the help text for the health check subcommand
func printHealthCheckUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s health [-healthcheck-socket-dir=<dir>] <liveness|readiness>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\nChecks:\n")
	fmt.Fprintf(os.Stderr, "  liveness   - Check if daemon is running and responsive\n")
	fmt.Fprintf(os.Stderr, "  readiness  - Check if devices are detected and ready for allocation\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	fmt.Fprintf(os.Stderr, "  -healthcheck-socket-dir  Directory where health check socket is located (default: /tmp)\n")
	fmt.Fprintf(os.Stderr, "\nEnvironment:\n")
	fmt.Fprintf(os.Stderr, "  SRIOV_HEALTHCHECK_SOCKET_DIR  Override socket directory\n")
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s health liveness\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s health -healthcheck-socket-dir=/var/run readiness\n", os.Args[0])
}

// validateHealthCheckArgs validates and extracts check type and socket directory from args
func validateHealthCheckArgs(args []string) (string, string, error) {
	healthFs := flag.NewFlagSet("health", flag.ContinueOnError)
	healthFs.SetOutput(io.Discard)
	healthFs.Usage = func() {}
	socketDir := ""
	healthFs.StringVar(&socketDir, "healthcheck-socket-dir", "",
		"Directory where health check socket is located")

	if err := healthFs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return "", "", fmt.Errorf("help")
		}
		return "", "", err
	}

	remaining := healthFs.Args()
	if len(remaining) == 0 {
		return "", "", fmt.Errorf("no check type provided")
	}

	checkType := remaining[0]
	if checkType == checkTypeHelp {
		return "", "", fmt.Errorf("help")
	}

	if len(remaining) > 1 {
		for _, arg := range remaining[1:] {
			if arg == "--help" || arg == "-h" || arg == checkTypeHelp {
				return "", "", fmt.Errorf("help")
			}
		}
		return "", "", fmt.Errorf("unexpected arguments: %v", remaining[1:])
	}

	if checkType != checkTypeLiveness && checkType != checkTypeReadiness {
		return "", "", fmt.Errorf("invalid check type %q (expected '%s' or '%s')", checkType, checkTypeLiveness, checkTypeReadiness)
	}

	return checkType, socketDir, nil
}

// performHealthCheck sends health check request and processes response
func performHealthCheck(socketPath, checkType string) int {
	conn, err := net.DialTimeout("unix", socketPath, healthCheckConnTimeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to connect to socket %s: %v\n", socketPath, err)
		return healthCheckErrorCode
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(healthCheckReadTimeout))

	// Send request
	req := map[string]string{"check": checkType}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to send request: %v\n", err)
		return healthCheckErrorCode
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var response interface{}
	if err := decoder.Decode(&response); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read response: %v\n", err)
		return healthCheckErrorCode
	}

	respMap, ok := response.(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: invalid response type\n")
		return healthCheckErrorCode
	}

	switch checkType {
	case checkTypeLiveness:
		return handleLivenessCheck(respMap)
	case checkTypeReadiness:
		return handleReadinessCheck(respMap)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown check type: %s\n", checkType)
		return healthCheckErrorCode
	}
}

// handleLivenessCheck processes liveness check response
func handleLivenessCheck(respMap map[string]interface{}) int {
	if status, ok := respMap["status"].(string); ok && status == "alive" {
		fmt.Println("OK: daemon is alive")
		return 0
	}
	fmt.Fprintf(os.Stderr, "ERROR: daemon not responding correctly\n")
	return healthCheckErrorCode
}

// handleReadinessCheck processes readiness check response
func handleReadinessCheck(respMap map[string]interface{}) int {
	if ready, ok := respMap["ready"].(bool); ok {
		healthyCount := int(respMap["healthy_count"].(float64))
		totalCount := int(respMap["total_count"].(float64))

		if ready {
			fmt.Printf("OK: ready (%d/%d devices healthy)\n", healthyCount, totalCount)
			return 0
		}
		fmt.Printf("NOT READY: no healthy devices (%d/%d)\n", healthyCount, totalCount)
		return readinessNotReadyCode
	}
	fmt.Fprintf(os.Stderr, "Error: invalid readiness response\n")
	return healthCheckErrorCode
}

// runHealthCheck performs a health check against the health check socket
func runHealthCheck(args []string, defaultSocketDir string) int {
	checkType, socketDir, err := validateHealthCheckArgs(args)
	if err != nil {
		if err.Error() == "help" {
			printHealthCheckUsage()
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			printHealthCheckUsage()
		}
		return healthCheckErrorCode
	}

	// Resolve socket directory: CLI flag > default
	if socketDir == "" {
		socketDir = defaultSocketDir
	}

	socketPath := filepath.Join(socketDir, "sriovdp.sock")
	return performHealthCheck(socketPath, checkType)
}
