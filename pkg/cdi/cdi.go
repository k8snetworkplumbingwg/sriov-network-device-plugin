/*
 * SPDX-FileCopyrightText: Copyright (c) 2022 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cdi

import (
	"os"
	"path/filepath"

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	"github.com/golang/glog"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
)

const cdiSpecPrefix = "sriov-dp-"

// CDI represents CDI API required by Device plugin
type CDI interface {
	CreateCDISpecForPool(resourcePrefix string, rPool types.ResourcePool) error
	CreateContainerAnnotations(devicesIDs []string, resourcePrefix, resourceKind string) (map[string]string, error)
	CleanupSpecs() error
}

// impl implements CDI interface
type impl struct {
}

// New returns an instance of CDI interface implementation
func New() CDI {
	return &impl{}
}

// CreateCDISpecForPool creates CDI spec file with specified devices
func (c *impl) CreateCDISpecForPool(resourcePrefix string, rPool types.ResourcePool) error {
	err := c.CleanupSpecs()
	if err != nil {
		glog.Errorf("CreateCDISpecForPool(): can not cleanup old spec files: %v", err)
		return err
	}
	cdiDevices := make([]cdiSpecs.Device, 0)
	cdiSpec := cdiSpecs.Spec{
		Version: cdiSpecs.CurrentVersion,
		Kind:    resourcePrefix + "/" + rPool.GetCDIName(),
		Devices: cdiDevices,
	}

	for _, dev := range rPool.GetDevices() {
		containerEdit := cdiSpecs.ContainerEdits{
			DeviceNodes: make([]*cdiSpecs.DeviceNode, 0),
		}

		for _, spec := range rPool.GetDeviceSpecs([]string{dev.GetID()}) {
			deviceNode := cdiSpecs.DeviceNode{
				Path:        spec.ContainerPath,
				HostPath:    spec.HostPath,
				Permissions: "rw",
			}
			containerEdit.DeviceNodes = append(containerEdit.DeviceNodes, &deviceNode)
		}
		device := cdiSpecs.Device{
			Name:           dev.GetID(),
			ContainerEdits: containerEdit,
		}
		cdiSpec.Devices = append(cdiSpec.Devices, device)
	}

	err = cdi.GetRegistry().SpecDB().WriteSpec(&cdiSpec, cdiSpecPrefix+resourcePrefix)
	if err != nil {
		glog.Errorf("CreateCDISpecForPool(): can not create CDI json: %v", err)
		return err
	}

	return nil
}

// CreateContainerAnnotations creates container annotations based on CDI spec for a container runtime
func (c *impl) CreateContainerAnnotations(devicesIDs []string, resourcePrefix, resourceKind string) (map[string]string, error) {
	annotations := make(map[string]string, 0)
	annoKey, err := cdi.AnnotationKey(resourcePrefix, resourceKind)
	if err != nil {
		glog.Errorf("CreateContainerAnnotations(): can't create container annotation: %v", err)
		return nil, err
	}
	devices := make([]string, 0)
	for _, id := range devicesIDs {
		devices = append(devices, cdi.QualifiedName(resourcePrefix, resourceKind, id))
	}
	annoValue, err := cdi.AnnotationValue(devices)
	if err != nil {
		glog.Errorf("CreateContainerAnnotations(): can't create container annotation: %v", err)
		return nil, err
	}
	annotations[annoKey] = annoValue

	return annotations, nil
}

// CleanupSpecs removes previously-created CDI specs
func (c *impl) CleanupSpecs() error {
	for _, dir := range cdi.GetRegistry().GetSpecDirectories() {
		specs, err := filepath.Glob(filepath.Join(dir, cdiSpecPrefix+"*"))
		if err != nil {
			return err
		}
		for _, spec := range specs {
			if err := os.Remove(spec); err != nil {
				return err
			}
		}
	}

	return nil
}
