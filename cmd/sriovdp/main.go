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

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/config"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/features"

	"github.com/golang/glog"
)

const (
	defaultConfig = "/etc/pcidp/config.json"
)

// Parse Command line flags
func flagInit(cp *cliParams) {
	flag.StringVar(&cp.configFile, "config-file", defaultConfig,
		"JSON device pool config file location")
	flag.StringVar(&cp.resourcePrefix, "resource-prefix", "intel.com",
		"resource name prefix used for K8s extended resource")
	flag.StringVar(&cp.featureGates, "feature-gates", "",
		"enables or disables selected features")
}

func main() {
	cp := &cliParams{}
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

	config.NewConfig()

	cfg, err := config.GetConfig()
	if err != nil {
		glog.Fatalf("error while getting config: %v", err)
		return
	}

	// Read global config
	if err := cfg.ReadConfig(cp.configFile); err != nil {
		glog.Error(err)
		return
	}

	fg, err := prepareFeaturegates()
	if err != nil {
		glog.Fatalf("error while getting feature gate: %v", err)
		return
	}

	// Set FeatureGates with ConfigMap
	if err := fg.SetFromMap(cfg.FeatureGates); err != nil {
		glog.Error(err)
		return
	}

	// Set FeatureGates with CLI arguments
	if err := fg.SetFromString(cp.featureGates); err != nil {
		glog.Error(err)
		return
	}

	glog.Infof("Discovering host devices")
	if err := rm.discoverHostDevices(); err != nil {
		glog.Errorf("error discovering host devices%v", err)
		return
	}

	if err := startServers(rm); err != nil {
		glog.Error(err)
		return
	}

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
}

func prepareFeaturegates() (*features.FeatureGate, error) {
	features.NewFeatureGate()
	return features.GetFeatureGate()
}

func startServers(rm *resourceManager) error {
	glog.Infof("Initializing resource servers")
	if err := rm.initServers(); err != nil {
		return fmt.Errorf("error initializing resource servers %w", err)
	}

	glog.Infof("Starting all servers...")
	if err := rm.startAllServers(); err != nil {
		return fmt.Errorf("error starting resource servers %w", err)
	}
	glog.Infof("All servers started.")
	return nil
}
