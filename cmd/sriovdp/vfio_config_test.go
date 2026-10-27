// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/factory"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

func TestNativeVFIOConfig(t *testing.T) {
	for _, tc := range []struct {
		name, options string
		valid         bool
	}{
		{"native", `"vfioMode":"iommufd","needIommu":true,`, true},
		{"legacy default", "", true},
		{"unknown mode", `"vfioMode":"auto",`, false},
		{"invalid control", `"needIommu":true,`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := filepath.Join(t.TempDir(), "config.json")
			data := `{"resourceList":[{"resourceName":"native",` + tc.options + `"selectors":[{"drivers":["vfio-pci"]}]}]}`
			if err := os.WriteFile(config, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
			rm := &resourceManager{cliParams: cliParams{configFile: config, resourcePrefix: "example.com", hostDevDir: "/host-dev"}, rFactory: factory.NewResourceFactory("example.com", "test", false, false)}
			rm.deviceProviders = map[types.DeviceType]types.DeviceProvider{types.NetDeviceType: rm.rFactory.GetDeviceProvider(types.NetDeviceType)}
			if err := rm.readConfig(); err != nil {
				t.Fatal(err)
			}
			if len(rm.configList) != 1 || rm.configList[0].HostDevDir != "/host-dev" || rm.configList[0].DeviceType != types.NetDeviceType {
				t.Fatalf("configuration not retained: %+v", rm.configList)
			}
			if rm.validConfigs() != tc.valid {
				t.Fatalf("validation should be %v", tc.valid)
			}
		})
	}
}
