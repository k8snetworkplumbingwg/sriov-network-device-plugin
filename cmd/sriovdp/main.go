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
	"os"
	"os/signal"
	"syscall"

	"k8s.io/klog/v2"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/logging"
)

const (
	defaultConfig = "/etc/pcidp/config.json"
)

// flagInit parse command line flags
func flagInit(cp *cliParams) {
	flag.StringVar(&cp.configFile, "config-file", defaultConfig,
		"JSON device pool config file location")
	flag.StringVar(&cp.resourcePrefix, "resource-prefix", "intel.com",
		"resource name prefix used for K8s extended resource")
	flag.BoolVar(&cp.useCdi, "use-cdi", false,
		"Use Container Device Interface to expose devices in containers")
	flag.IntVar(&cp.logMaxSize, "log-max-size", logging.DefaultMaxSizeMB,
		"Maximum size in MB of a log file before rotation")
	flag.IntVar(&cp.logMaxFiles, "log-max-files", logging.DefaultMaxFiles,
		"Maximum number of old rotated log files to retain")
	flag.IntVar(&cp.logMaxAge, "log-max-age", logging.DefaultMaxAge,
		"Maximum number of days to retain old log files")
}

func resolveLogDir() string {
	if f := flag.Lookup("log_dir"); f != nil {
		if v := f.Value.String(); v != "" {
			return v
		}
	}
	return logging.DefaultLogDir
}

func main() {
	cp := &cliParams{}
	klog.InitFlags(nil)
	flagInit(cp)
	flag.Parse()

	logCfg := logging.Config{
		LogDir:     resolveLogDir(),
		MaxSizeMB:  cp.logMaxSize,
		MaxFiles:   cp.logMaxFiles,
		MaxAgeDays: cp.logMaxAge,
	}
	if cleanupLog := logging.SetupLogRotation(logCfg); cleanupLog != nil {
		defer cleanupLog()
	}
	defer klog.Flush()

	logging.LogStartupHeader(logging.StartupInfo{
		ConfigFile:     cp.configFile,
		ResourcePrefix: cp.resourcePrefix,
		UseCdi:         cp.useCdi,
		LogDir:         logCfg.LogDir,
		LogMaxSizeMB:   logCfg.MaxSizeMB,
		LogMaxFiles:    logCfg.MaxFiles,
		LogMaxAgeDays:  logCfg.MaxAgeDays,
	})

	rm := newResourceManager(cp)

	klog.Infof("resource manager reading configs")
	if err := rm.readConfig(); err != nil {
		klog.Errorf("error getting resources from file %v", err)
		return
	}

	if len(rm.configList) < 1 {
		klog.Errorf("no resource configuration; exiting")
		return // No config found
	}

	// Validate configs
	if !rm.validConfigs() {
		klog.Fatalf("Exiting.. one or more invalid configuration(s) given")
		return
	}
	klog.Infof("Discovering host devices")
	if err := rm.discoverHostDevices(); err != nil {
		klog.Errorf("error discovering host devices%v", err)
		return
	}

	klog.Infof("Initializing resource servers")
	if err := rm.initServers(); err != nil {
		klog.Errorf("error initializing resource servers %v", err)
		return
	}

	klog.Infof("Starting all servers...")
	if err := rm.startAllServers(); err != nil {
		klog.Errorf("error starting resource servers %v\n", err)
		return
	}
	klog.Infof("All servers started.")
	klog.Infof("Listening for term signals")
	// respond to syscalls for termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Catch termination signals
	sig := <-sigCh
	klog.Infof("Received signal \"%v\", shutting down.", sig)
	if err := rm.stopAllServers(); err != nil {
		klog.Errorf("stopping servers produced error: %s", err.Error())
	}
	if err := rm.cleanupCDISpecs(); err != nil {
		klog.Errorf("cleaning up CDI Specs produced error: %s", err.Error())
	}
}
