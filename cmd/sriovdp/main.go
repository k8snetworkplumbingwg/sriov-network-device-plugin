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
	"syscall"

	"github.com/golang/glog"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/logging"
)

const (
	defaultConfig = "/etc/pcidp/config.json"
	defaultLogMaxSize = 100
	defaultLogMaxFiles = 5
	defaultLogMaxAge = 30
)

// flagInit parse command line flags
func flagInit(cp *cliParams) {
	flag.StringVar(&cp.configFile, "config-file", defaultConfig,
		"JSON device pool config file location")
	flag.StringVar(&cp.resourcePrefix, "resource-prefix", "intel.com",
		"resource name prefix used for K8s extended resource")
	flag.BoolVar(&cp.useCdi, "use-cdi", false,
		"Use Container Device Interface to expose devices in containers")
	flag.IntVar(&cp.logMaxSize, "log-max-size", defaultLogMaxSize,
		"Maximum size in MB of a log file before rotation")
	flag.IntVar(&cp.logMaxFiles, "log-max-files", defaultLogMaxFiles,
		"Maximum number of old rotated log files to retain")
	flag.IntVar(&cp.logMaxAge, "log-max-age", defaultLogMaxAge,
		"Maximum number of days to retain old log files")
}

func configureGlogDefaults() {
	if f := flag.Lookup("logtostderr"); f != nil && f.Value.String() != "true" {
		if err := flag.Set("logtostderr", "true"); err != nil {
			glog.Warningf("failed to set logtostderr=true: %v", err)
		}
	}
	if f := flag.Lookup("alsologtostderr"); f != nil && f.Value.String() != "false" {
		if err := flag.Set("alsologtostderr", "false"); err != nil {
			glog.Warningf("failed to set alsologtostderr=false: %v", err)
		}
	}
}

func main() {
	cp := &cliParams{}
	flagInit(cp)
	flag.Parse()
	configureGlogDefaults()

	if cleanupLog := setupLogRotation(cp); cleanupLog != nil {
		defer cleanupLog()
	}
	defer glog.Flush()

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

// setupLogRotation configures stderr-based log rotation. It uses the
// glog --log_dir path when set, otherwise falls back to the default
// log directory. Returns a cleanup function that must be deferred.
func setupLogRotation(cp *cliParams) func() {
	logDir := logging.DefaultConfig().LogDir
	if f := flag.Lookup("log_dir"); f != nil {
		if v := f.Value.String(); v != "" {
			logDir = v
		}
	}

	rawCfg := logging.Config{
		LogDir:     logDir,
		MaxSizeMB:  cp.logMaxSize,
		MaxFiles:   cp.logMaxFiles,
		MaxAgeDays: cp.logMaxAge,
		Compress:   true,
	}
	cfg, warnings, err := logging.ResolveConfig(rawCfg)
	for _, w := range warnings {
		glog.Warning(w)
	}
	if err != nil {
		glog.Errorf("failed to resolve log rotation config: %v — continuing without rotation", err)
		return nil
	}

	rotWriter, _, err := logging.NewRotatingWriter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sriovdp: log rotation disabled; cannot use log directory %q: %v\n", cfg.LogDir, err) //nolint:errcheck
		glog.Errorf("failed to create log rotation writer: %v — continuing without rotation", err)
		return nil
	}

	cleanup, err := logging.CaptureStderr(rotWriter)
	if err != nil {
		glog.Errorf("failed to capture stderr for log rotation: %v — continuing without rotation", err)
		if closeErr := rotWriter.Close(); closeErr != nil {
			glog.Warningf("failed to close rotation writer after capture failure: %v", closeErr)
		}
		return nil
	}

	glog.Infof("Log rotation enabled: dir=%s maxSize=%dMB maxFiles=%d maxAge=%d compress=%v",
		cfg.LogDir, cfg.MaxSizeMB, cfg.MaxFiles, cfg.MaxAgeDays, cfg.Compress)

	return func() {
		cleanup()
		if err := rotWriter.Close(); err != nil {
			glog.Warningf("failed to close rotation writer: %v", err)
		}
	}
}
