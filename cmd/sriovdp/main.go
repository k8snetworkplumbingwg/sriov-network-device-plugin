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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/golang/glog"
)

const (
	defaultConfig   = "/etc/pcidp/config.json"
	defaultLogLevel = 10
)

func usage() {
	fmt.Fprintf(os.Stderr,
		"SR-IOV Network Device Plugin\n\n"+
			"%s\n"+
			"\t-h -help\n"+
			"\t--log-dir=\n"+
			"\t--log-level=%d\n"+
			"\t--resource-prefix=\n"+
			"\t--config-file=\n"+
			"\t--use-cdi\n",
		os.Args[0], defaultLogLevel)
}

// flagInit registers CLI flags (same surface as the former cmd/entrypoint wrapper).
func flagInit(cp *cliParams, logDir *string, logLevel *int) {
	flag.Usage = usage
	flag.StringVar(logDir, "log-dir", "", "Log directory under /var/log/")
	flag.IntVar(logLevel, "log-level", defaultLogLevel, "Log verbosity level")
	flag.StringVar(&cp.configFile, "config-file", defaultConfig,
		"JSON device pool config file location")
	flag.StringVar(&cp.resourcePrefix, "resource-prefix", "intel.com",
		"resource name prefix used for K8s extended resource")
	flag.BoolVar(&cp.useCdi, "use-cdi", false,
		"Use Container Device Interface to expose devices in containers")
}

// applyLogFlags configures glog after flag.Parse() so --log-dir / --log-level are honored.
func applyLogFlags(logDir string, logLevel int) error {
	if err := flag.Set("v", strconv.Itoa(logLevel)); err != nil {
		return err
	}
	if logDir != "" {
		logPath := filepath.Join("/var/log", logDir)
		if err := os.MkdirAll(logPath, 0o755); err != nil {
			return fmt.Errorf("failed to create log dir %q: %w", logPath, err)
		}
		if err := flag.Set("log_dir", logPath); err != nil {
			return err
		}
	}
	if err := flag.Set("logtostderr", "true"); err != nil {
		return err
	}

	return nil
}

func main() {
	cp := &cliParams{}
	var logDir string
	logLevel := defaultLogLevel
	flagInit(cp, &logDir, &logLevel)
	flag.Parse()
	if err := applyLogFlags(logDir, logLevel); err != nil {
		fmt.Fprintf(os.Stderr, "error configuring logging: %v\n", err)
		return
	}
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
