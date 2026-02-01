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

package resources

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"

	cdiPkg "github.com/rebellions-sw/rbln-k8s-device-plugin/pkg/cdi"
	"github.com/rebellions-sw/rbln-k8s-device-plugin/pkg/infoprovider"
	"github.com/rebellions-sw/rbln-k8s-device-plugin/pkg/types"
)

type resourceServer struct {
	resourcePool       types.ResourcePool
	resourceConfig     *types.ResourceConfig
	pluginWatch        bool
	endPoint           string // Socket file
	sockPath           string // Socket file path
	resourceNamePrefix string
	grpcServer         *grpc.Server
	termSignal         chan bool
	updateSignal       chan bool
	stopWatcher        chan bool
	checkIntervals     int // health check intervals in seconds
	useCdi             bool
	cdi                cdiPkg.CDI
}

const (
	rsWatchInterval = 5 * time.Second
	unix            = "unix"
	sysBusPci       = "/sys/bus/pci/devices"
)

// NewResourceServer returns an instance of ResourceServer
func NewResourceServer(prefix, suffix string, pluginWatch, useCdi bool, rp types.ResourcePool,
	rc *types.ResourceConfig) types.ResourceServer {
	sockName := fmt.Sprintf("%s_%s.%s", prefix, rp.GetResourceName(), suffix)
	sockPath := filepath.Join(types.SockDir, sockName)
	if !pluginWatch {
		sockPath = filepath.Join(types.DeprecatedSockDir, sockName)
	}
	return &resourceServer{
		resourcePool:       rp,
		resourceConfig:     rc,
		pluginWatch:        pluginWatch,
		endPoint:           sockName,
		sockPath:           sockPath,
		resourceNamePrefix: prefix,
		useCdi:             useCdi,
		grpcServer:         grpc.NewServer(),
		termSignal:         make(chan bool, 1),
		updateSignal:       make(chan bool),
		stopWatcher:        make(chan bool),
		checkIntervals:     20, // updates every 20 seconds
		cdi:                cdiPkg.New(),
	}
}

func (rs *resourceServer) register() error {
	kubeletEndpoint := unix + ":" + filepath.Join(types.DeprecatedSockDir, types.KubeEndPoint)
	conn, err := grpc.NewClient(kubeletEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		glog.Errorf("%s device plugin unable connect to Kubelet : %v", rs.resourcePool.GetResourceName(), err)
		return err
	}
	defer conn.Close()
	client := pluginapi.NewRegistrationClient(conn)

	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     rs.endPoint,
		ResourceName: fmt.Sprintf("%s/%s", rs.resourceNamePrefix, rs.resourcePool.GetResourceName()),
	}

	if _, err = client.Register(context.Background(), request); err != nil {
		glog.Errorf("%s device plugin unable to register with Kubelet : %v", rs.resourcePool.GetResourceName(), err)
		return err
	}
	glog.Infof("%s device plugin registered with Kubelet", rs.resourcePool.GetResourceName())
	return nil
}

func (rs *resourceServer) GetInfo(ctx context.Context, rqt *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	pluginInfoResponse := &registerapi.PluginInfo{
		Type:              registerapi.DevicePlugin,
		Name:              fmt.Sprintf("%s/%s", rs.resourceNamePrefix, rs.resourcePool.GetResourceName()),
		Endpoint:          filepath.Join(types.SockDir, rs.endPoint),
		SupportedVersions: []string{"v1alpha1", "v1beta1"},
	}
	return pluginInfoResponse, nil
}

func (rs *resourceServer) NotifyRegistrationStatus(ctx context.Context,
	regstat *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	if regstat.PluginRegistered {
		glog.Infof("Plugin: %s gets registered successfully at Kubelet\n", rs.endPoint)
	} else {
		glog.Infof("Plugin: %s failed to be registered at Kubelet: %v; restarting.\n", rs.endPoint, regstat.Error)
		rs.grpcServer.Stop()
	}
	return &registerapi.RegistrationStatusResponse{}, nil
}

func (rs *resourceServer) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	glog.Infof("Allocate() called with %+v", rqt)
	resp := new(pluginapi.AllocateResponse)

	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)

		sort.Strings(container.DevicesIDs)

		envs, err := rs.getEnvs(container.DevicesIDs)
		if err != nil {
			glog.Errorf("failed to get environment variables for device IDs %v: %v", container.DevicesIDs, err)
			return nil, err
		}

		if rs.useCdi {
			containerResp.Annotations, err = cdiPkg.CreateRuntimeAnnotations(rs.resourceNamePrefix, rs.resourcePool.GetCDIName())
			if err != nil {
				return nil, fmt.Errorf("can't create container annotation: %s", err)
			}
			containerResp.Devices = rs.resourcePool.GetDeviceSpecs(container.DevicesIDs)
		} else {
			containerResp.Devices = rs.resourcePool.GetDeviceSpecs(container.DevicesIDs)
			containerResp.Mounts = rs.resourcePool.GetMounts(container.DevicesIDs)
		}

		err = rs.resourcePool.StoreDeviceInfoFile(rs.resourceNamePrefix, container.DevicesIDs)
		if err != nil {
			glog.Errorf("failed to store device info file for device IDs %v: %v", container.DevicesIDs, err)
			return nil, err
		}

		containerResp.Envs = envs
		resp.ContainerResponses = append(resp.ContainerResponses, containerResp)
	}
	glog.Infof("AllocateResponse send: %+v", resp)
	return resp, nil
}

func (rs *resourceServer) ListAndWatch(empty *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	methodID := fmt.Sprintf("ListAndWatch(%s)", rs.resourcePool.GetResourceName()) // for logging purpose
	glog.Infof("%s invoked", methodID)
	// Send initial list of devices
	devs := make([]*pluginapi.Device, 0)
	resp := new(pluginapi.ListAndWatchResponse)
	for _, dev := range rs.resourcePool.GetDevices() {
		devs = append(devs, dev)
	}
	resp.Devices = devs
	glog.Infof("%s: send devices %v\n", methodID, resp)
	if err := stream.Send(resp); err != nil {
		glog.Errorf("%s: error: cannot update device states: %v\n", methodID, err)
		rs.grpcServer.Stop()
		return err
	}

	// listen for events: if updateSignal send new list of devices
	for {
		select {
		case <-rs.termSignal:
			// Terminate signal received; return from mehtod call
			glog.Infof("%s: terminate signal received", methodID)
			return nil
		case <-rs.updateSignal:
			// Device health changed; so send new device list
			glog.Infof("%s: device health changed!\n", methodID)
			newDevs := make([]*pluginapi.Device, 0)
			for _, dev := range rs.resourcePool.GetDevices() {
				newDevs = append(newDevs, dev)
			}
			resp.Devices = newDevs
			glog.Infof("%s: send updated devices %v", methodID, resp)

			if err := stream.Send(resp); err != nil {
				glog.Errorf("%s: error: cannot update device states: %v\n", methodID, err)
				return err
			}
		}
	}
}

// TODO: (SchSeba) check if we want to use this function
func (rs *resourceServer) GetPreferredAllocation(ctx context.Context,
	request *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	glog.Infof("GetPreferredAllocation() called with %+v", request)
	response := &pluginapi.PreferredAllocationResponse{}
	for _, req := range request.ContainerRequests {
		var resp *pluginapi.ContainerPreferredAllocationResponse
		devices, err := rs.selectPreferredDevices(req.AvailableDeviceIDs, req.MustIncludeDeviceIDs, int(req.AllocationSize))
		if err != nil || devices == nil {
			// For non-critical errors (like NonRebellionsDeviceError), fall back to kubelet
			glog.Warningf(
				"Could not determine preferred allocation for container, falling back to kubelet's default logic. Request: %+v, Error: %v",
				req, err,
			)
			resp = &pluginapi.ContainerPreferredAllocationResponse{
				DeviceIDs: []string{},
			}
		} else {
			resp = &pluginapi.ContainerPreferredAllocationResponse{
				DeviceIDs: devices,
			}
		}

		response.ContainerResponses = append(response.ContainerResponses, resp)
	}
	return response, nil
}

func (rs *resourceServer) PreStartContainer(ctx context.Context,
	psRqt *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (rs *resourceServer) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: true,
	}, nil
}

func (rs *resourceServer) Init() error {
	return nil
}

// gRPC server related
func (rs *resourceServer) Start() error {
	resourceName := rs.resourcePool.GetResourceName()
	_ = rs.cleanUp() // try tp clean up and continue

	glog.Infof("starting %s device plugin endpoint at: %s\n", resourceName, rs.endPoint)
	lis, err := net.Listen(unix, rs.sockPath)
	if err != nil {
		glog.Errorf("error starting %s device plugin endpoint: %v", resourceName, err)
		return err
	}

	// Register all services
	if rs.pluginWatch {
		registerapi.RegisterRegistrationServer(rs.grpcServer, rs)
	}
	pluginapi.RegisterDevicePluginServer(rs.grpcServer, rs)

	// start serving from grpcServer
	go func() {
		err := rs.grpcServer.Serve(lis)
		if err != nil {
			glog.Errorf("serving incoming requests failed: %s", err.Error())
		}
	}()

	rs.triggerUpdate()

	if !rs.pluginWatch {
		// Register with Kubelet.
		err = rs.register()
		if err != nil {
			// Stop server
			rs.grpcServer.Stop()
			glog.Fatal(err)
			return err
		}
	}

	return nil
}

func (rs *resourceServer) restart() error {
	resourceName := rs.resourcePool.GetResourceName()
	glog.Infof("restarting %s device plugin server...", resourceName)
	if rs.grpcServer == nil {
		return fmt.Errorf("grpc server instance not found for %s", resourceName)
	}
	rs.grpcServer.Stop()
	rs.grpcServer = nil
	// Send terminate signal to ListAndWatch()
	rs.termSignal <- true

	rs.grpcServer = grpc.NewServer() // new instance of a grpc server
	return rs.Start()
}

func (rs *resourceServer) Stop() error {
	resourceName := rs.resourcePool.GetResourceName()
	glog.Infof("stopping %s device plugin server...", resourceName)
	if rs.grpcServer == nil {
		return nil
	}
	// Send terminate signal to ListAndWatch()
	rs.termSignal <- true
	if !rs.pluginWatch {
		rs.stopWatcher <- true
	}

	rs.grpcServer.Stop()
	rs.grpcServer = nil

	return rs.cleanUp()
}

func (rs *resourceServer) Watch() {
	// Watch for Kubelet socket file; if not present restart server
	for {
		select {
		case stop := <-rs.stopWatcher:
			if stop {
				glog.Infof("kubelet watcher stopped for server %s", rs.resourcePool.GetResourceName())
				return
			}
		default:
			_, err := os.Lstat(rs.sockPath)
			if err != nil {
				// Socket file not found; restart server
				glog.Warningf("server endpoint not found %s", rs.endPoint)
				glog.Warningf("most likely Kubelet restarted")
				if err := rs.restart(); err != nil {
					glog.Fatalf("unable to restart server %v", err)
				}
			}
		}
		// Sleep for some intervals; TODO: investigate on suggested interval
		time.Sleep(rsWatchInterval)
	}
}

func (rs *resourceServer) cleanUp() error {
	errors := make([]string, 0)
	if err := os.Remove(rs.sockPath); err != nil && !os.IsNotExist(err) {
		errors = append(errors, err.Error())
	}
	if err := rs.resourcePool.CleanDeviceInfoFile(rs.resourceNamePrefix); err != nil {
		errors = append(errors, err.Error())
	}
	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, ","))
	}
	return nil
}

func (rs *resourceServer) triggerUpdate() {
	rp := rs.resourcePool
	if rs.checkIntervals > 0 {
		go func() {
			for {
				changed := rp.Probe()
				if changed {
					rs.updateSignal <- true
				}
				time.Sleep(time.Second * time.Duration(rs.checkIntervals))
			}
		}()
	}
}

func (rs *resourceServer) getEnvs(deviceIDs []string) (map[string]string, error) {
	return rs.resourcePool.GetEnvs(rs.resourceNamePrefix, deviceIDs)
}

func (rs *resourceServer) selectPreferredDevices(availableDeviceIDs, mustIncludeDeviceIDs []string, allocationSize int) ([]string, error) {
	// Check if this is a Rebellions NPU device pool
	if !rs.isRebellionsNPUPool(availableDeviceIDs) {
		glog.Infof("Non-Rebellions device pool detected, delegating to kubelet for allocation")
		return nil, fmt.Errorf("non-Rebellions device pool, kubelet should handle allocation")
	}

	glog.Infof("Rebellions NPU device pool detected, applying topology-aware allocation")

	// Get product ID of the first device to determine device type
	productID, err := rs.getDeviceProductID(availableDeviceIDs[0])
	if err != nil {
		glog.Errorf("Failed to get product ID for device %s: %v", availableDeviceIDs[0], err)
		return nil, fmt.Errorf("failed to get product ID for device %s: %v", availableDeviceIDs[0], err)
	}

	// Create allocator using factory
	allocator, err := CreateAllocator(availableDeviceIDs, productID, rs.resourceConfig)
	if err != nil {
		glog.Errorf("Failed to create allocator: %v", err)
		return nil, err
	}

	result := allocator.SelectDevices(mustIncludeDeviceIDs, allocationSize)
	return result, nil
}

// isRebellionsNPUPool checks if the device pool contains Rebellions NPU devices
func (rs *resourceServer) isRebellionsNPUPool(availableDeviceIDs []string) bool {
	// Use the first device to check vendor ID
	deviceID := availableDeviceIDs[0]
	vendorID, err := rs.getDeviceVendorID(deviceID)
	if err != nil {
		glog.Warningf("Failed to get vendor ID for device %s: %v", deviceID, err)
		return false
	}

	return vendorID == infoprovider.RebellionsVendorID
}

// getDeviceVendorID returns the vendor ID for a given PCI device
func (rs *resourceServer) getDeviceVendorID(pciAddr string) (string, error) {
	vendorPath := filepath.Join(sysBusPci, pciAddr, "vendor")
	vendorData, err := os.ReadFile(vendorPath)
	if err != nil {
		return "", fmt.Errorf("failed to read vendor ID: %w", err)
	}

	vendorStr := strings.TrimSpace(string(vendorData))
	vendorStr = strings.TrimPrefix(vendorStr, "0x")

	return vendorStr, nil
}

// getDeviceProductID returns the product ID for a given PCI device
func (rs *resourceServer) getDeviceProductID(pciAddr string) (string, error) {
	devicePath := filepath.Join(sysBusPci, pciAddr, "device")
	deviceData, err := os.ReadFile(devicePath)
	if err != nil {
		return "", fmt.Errorf("failed to read device/product ID: %w", err)
	}

	deviceStr := strings.TrimSpace(string(deviceData))
	deviceStr = strings.TrimPrefix(deviceStr, "0x")

	return deviceStr, nil
}
