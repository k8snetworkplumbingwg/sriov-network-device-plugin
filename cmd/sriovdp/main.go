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

	logCfg := logging.Config{
		LogDir:     logging.DefaultConfig().LogDir,
		MaxSizeMB:  cp.logMaxSize,
		MaxFiles:   cp.logMaxFiles,
		MaxAgeDays: cp.logMaxAge,
		Compress:   true,
	}
	if cleanupLog := logging.SetupLogRotation(logCfg); cleanupLog != nil {
		defer cleanupLog()
	}
	defer glog.Flush()

	logging.LogStartupHeader(logging.StartupInfo{
		ConfigFile:     cp.configFile,
		ResourcePrefix: cp.resourcePrefix,
		UseCdi:         cp.useCdi,
	})

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
