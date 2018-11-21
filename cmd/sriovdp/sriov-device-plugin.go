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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	netDirectory    = "/sys/class/net/"
	sriovCapable    = "/sriov_totalvfs"
	sriovConfigured = "/sriov_numvfs"

	// Device plugin settings.
	pluginMountPath      = "/var/lib/kubelet/device-plugins"
	kubeletEndpoint      = "kubelet.sock"
	pluginEndpointPrefix = "sriovNet"
	resourceName         = "intel.com/sriov"
)

// sriovManager manages sriov networking devices
type sriovManager struct {
	socketFile  string
	devices     map[string]pluginapi.Device // for Kubelet DP API
	rootDevices []string
	grpcServer  *grpc.Server
	termSignal  chan bool
	stopWatcher chan bool
}

func newSriovManager() *sriovManager {

	return &sriovManager{
		devices:     make(map[string]pluginapi.Device),
		socketFile:  fmt.Sprintf("%s.sock", pluginEndpointPrefix),
		termSignal:  make(chan bool, 1),
		stopWatcher: make(chan bool),
	}
}

// Returns a list of SRIOV capable PF names as string
func getSriovPfList() ([]string, error) {

	sriovNetDevices := []string{}

	netDevices, err := ioutil.ReadDir(netDirectory)
	if err != nil {
		glog.Errorf("Error. Cannot read %s for network device names. Err: %v", netDirectory, err)
		return sriovNetDevices, err
	}

	if len(netDevices) < 1 {
		glog.Errorf("Error. No network device found in %s directory", netDirectory)
		return sriovNetDevices, err
	}

	for _, dev := range netDevices {
		sriovDirPath := filepath.Join(netDirectory, dev.Name())
		glog.Infof("Checking inside dir %s", sriovDirPath)
		dir, err := os.Stat(sriovDirPath)
		if err != nil {
			continue
		}
		if !dir.Mode().IsDir() {
			// Could be e.g. bonding_masters file
			continue
		}

		sriovFilePath := filepath.Join(sriovDirPath, "device", "sriov_numvfs")
		glog.Infof("Checking for file %s", sriovFilePath)

		if f, err := os.Lstat(sriovFilePath); !os.IsNotExist(err) {
			if f.Mode().IsRegular() { // and its a regular file
				sriovNetDevices = append(sriovNetDevices, dev.Name())
			}
		}
	}

	return sriovNetDevices, nil
}

// GetVFList returns a List containing PCI addr for all VF discovered in a given PF
func GetVFList(pf string) ([]string, error) {
	vfList := make([]string, 0)
	pfDir := filepath.Join(netDirectory, pf, "device")
	_, err := os.Lstat(pfDir)
	if err != nil {
		glog.Errorf("Error. Could not get PF directory information for device: %s, Err: %v", pf, err)
		return vfList, err
	}

	vfDirs, err := filepath.Glob(filepath.Join(pfDir, "virtfn*"))
	if err != nil {
		glog.Errorf("Error. Could not read VF directories, Err: %v", err)
		return vfList, err
	}

	//Read all VF directory and get add VF PCI addr to the vfList
	for _, dir := range vfDirs {
		dirInfo, err := os.Lstat(dir)
		if err == nil && (dirInfo.Mode()&os.ModeSymlink != 0) {
			linkName, err := filepath.EvalSymlinks(dir)
			if err == nil {
				vfLink := filepath.Base(linkName)
				vfList = append(vfList, vfLink)
			}
		}
	}
	return vfList, err
}

//Reads DeviceName and gets PCI Addresses of VFs configured
func (sm *sriovManager) discoverNetworks() error {

	var healthValue string
	sm.rootDevices = []string{}

	// Get a list of SRIOV capable NICs in the host
	pfList, err := getSriovPfList()

	if err != nil {
		return err
	}

	if len(pfList) < 1 {
		glog.Errorf("Error. No SRIOV network device found")
		return fmt.Errorf("Error. No SRIOV network device found")
	}

	for _, dev := range pfList {
		sriovcapablepath := filepath.Join(netDirectory, dev, "device", sriovCapable)
		glog.Infof("Sriov Capable Path: %v", sriovcapablepath)
		vfs, err := ioutil.ReadFile(sriovcapablepath)
		if err != nil {
			glog.Errorf("Error. Could not read sriov_totalvfs in device folder. SRIOV not supported. Err: %v", err)
			return err
		}
		totalvfs := bytes.TrimSpace(vfs)
		numvfs, err := strconv.Atoi(string(totalvfs))
		if err != nil {
			glog.Errorf("Error. Could not parse sriov_capable file. Err: %v", err)
			return err
		}
		glog.Infof("Total number of VFs for device %v is %v", dev, numvfs)
		if numvfs > 0 {
			glog.Infof("SRIOV capable device discovered: %v", dev)
			sriovconfiguredpath := netDirectory + dev + "/device" + sriovConfigured
			vfs, err = ioutil.ReadFile(sriovconfiguredpath)
			if err != nil {
				glog.Errorf("Error. Could not read sriov_numvfs file. SRIOV error. %v", err)
				return err
			}
			configuredVFs := bytes.TrimSpace(vfs)
			numconfiguredvfs, err := strconv.Atoi(string(configuredVFs))
			if err != nil {
				glog.Errorf("Error. Could not parse sriov_numvfs files. Skipping device. Err: %v", err)
				return err
			}
			glog.Infof("Number of Configured VFs for device %v is %v", dev, string(configuredVFs))

			if numconfiguredvfs > 0 {
				sm.rootDevices = append(sm.rootDevices, dev)
				if IsNetlinkStatusUp(dev) {
					healthValue = pluginapi.Healthy
				} else {
					healthValue = "Unhealthy"
				}
				if vfList, err := GetVFList(dev); err == nil {
					for _, vfDev := range vfList {
						sm.devices[vfDev] = pluginapi.Device{ID: vfDev, Health: healthValue}
					}
				}
			}
		}
	}
	glog.Infof("Discovered SR-IOV PF devices: %v", sm.rootDevices)
	return nil
}

// IsNetlinkStatusUp returns 'false' if 'operstate' is not "up" for a Linux netowrk device
func IsNetlinkStatusUp(dev string) bool {
	opsFile := filepath.Join(netDirectory, dev, "operstate")
	bytes, err := ioutil.ReadFile(opsFile)
	if err != nil || strings.TrimSpace(string(bytes)) != "up" {
		return false
	}
	return true
}

// Probe returns 'true' if device health changes 'false' otherwise
func (sm *sriovManager) Probe() bool {
	// Network device should check link status for each physical port and update health status for
	// all associated VFs if there is any
	changed := false
	var healthValue string
	for _, pf := range sm.rootDevices {
		// If the PF link is up = "Healthy"
		if IsNetlinkStatusUp(pf) {
			healthValue = pluginapi.Healthy
		} else {
			healthValue = "Unhealthy"
		}

		// Get VFs associated with this device
		if vfs, err := GetVFList(pf); err == nil {
			for _, vf := range vfs {
				device := sm.devices[vf]
				if device.Health != healthValue {
					sm.devices[vf] = pluginapi.Device{ID: vf, Health: healthValue}
					changed = true
				}
			}
		}
	}
	return changed
}

// Discovers SRIOV capabable NIC devices.
func (sm *sriovManager) Start() error {
	glog.Infof("Discovering SRIOV network device[s]")
	if err := sm.discoverNetworks(); err != nil {
		return err
	}
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, sm.socketFile)
	glog.Infof("Starting SRIOV Network Device Plugin server at: %s\n", pluginEndpoint)
	lis, err := net.Listen("unix", pluginEndpoint)
	if err != nil {
		glog.Errorf("Error. Starting SRIOV Network Device Plugin server failed: %v", err)
	}
	sm.grpcServer = grpc.NewServer()

	// Register SRIOV device plugin service
	pluginapi.RegisterDevicePluginServer(sm.grpcServer, sm)

	go sm.grpcServer.Serve(lis)

	// Wait for server to start by launching a blocking connection
	conn, err := grpc.Dial(pluginEndpoint, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		glog.Errorf("Error. Could not establish connection with gRPC server: %v", err)
		return err
	}
	glog.Infoln("SRIOV Network Device Plugin server started serving")
	conn.Close()

	// Registers with Kubelet.
	err = Register(path.Join(pluginMountPath, kubeletEndpoint), sm.socketFile, resourceName)
	if err != nil {
		// Stop server
		sm.grpcServer.Stop()
		glog.Fatal(err)
		return err
	}
	glog.Infof("SRIOV Network Device Plugin registered with the Kubelet")
	return nil
}

func (sm *sriovManager) restart() error {
	glog.Infof("Restarting SRIOV Network Device Plugin server..")
	if sm.grpcServer == nil {
		return nil
	}
	// Send terminate signal to ListAndWatch()
	sm.termSignal <- true

	sm.grpcServer.Stop()
	sm.grpcServer = nil

	return sm.Start()
}

func (sm *sriovManager) Stop() error {
	glog.Infof("Stopping SRIOV Network Device Plugin server..")
	if sm.grpcServer == nil {
		return nil
	}
	// Send terminate signal to ListAndWatch()
	sm.termSignal <- true
	sm.stopWatcher <- true

	sm.grpcServer.Stop()
	sm.grpcServer = nil

	return sm.cleanup()
}

func (sm *sriovManager) Watch() {
	// Watch for socket file; if not present restart server
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, sm.socketFile)
	for {
		select {
		case stop := <-sm.stopWatcher:
			if stop {
				return
			}
		default:
			_, err := os.Lstat(pluginEndpoint)
			if err != nil {
				// Socket file not found; restart server
				glog.Warningf("Server endpoint not found %s", sm.socketFile)
				glog.Warningf("Most likely Kubelet restarted")
				if err := sm.restart(); err != nil {
					glog.Fatalf("Unable to restart server %v", err)
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
}

// Removes existing socket if exists
// [adpoted from https://github.com/redhat-nfvpe/k8s-dummy-device-plugin/blob/master/dummy.go ]
func (sm *sriovManager) cleanup() error {
	pluginEndpoint := filepath.Join(pluginapi.DevicePluginPath, sm.socketFile)
	if err := os.Remove(pluginEndpoint); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Register registers as a grpc client with the kubelet.
func Register(kubeletEndpoint, pluginEndpoint, resourceName string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		glog.Errorf("SRIOV Network Device Plugin cannot connect to Kubelet service: %v", err)
		return err
	}
	defer conn.Close()
	client := pluginapi.NewRegistrationClient(conn)

	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	if _, err = client.Register(context.Background(), request); err != nil {
		glog.Errorf("SRIOV Network Device Plugin cannot register to Kubelet service: %v", err)
		return err
	}
	return nil
}

// Implements DevicePlugin service functions
func (sm *sriovManager) ListAndWatch(empty *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	// Send initial list of devices
	resp := new(pluginapi.ListAndWatchResponse)
	for _, dev := range sm.devices {
		resp.Devices = append(resp.Devices, &pluginapi.Device{ID: dev.ID, Health: dev.Health})
	}
	glog.Infof("ListAndWatch: send initial devices %v\n", resp)
	if err := stream.Send(resp); err != nil {
		glog.Errorf("Error. Cannot send initial device states: %v\n", err)
		sm.grpcServer.Stop()
		return err
	}

	// Probes device state every 10 seconds and updates if changed.
	// Terminates when termSignal received.
	for {
		if sm.Probe() {
			resp := new(pluginapi.ListAndWatchResponse)
			for _, dev := range sm.devices {
				resp.Devices = append(resp.Devices, &pluginapi.Device{ID: dev.ID, Health: dev.Health})
			}
			glog.Infof("ListAndWatch: send devices %v\n", resp)
			if err := stream.Send(resp); err != nil {
				glog.Errorf("Error. Cannot update device states: %v\n", err)
				sm.grpcServer.Stop()
				return err
			}
		}

		select {
		case <-time.After(10 * time.Second):
			continue
		case <-sm.termSignal:
			glog.Infof("Terminate signal received, exiting ListAndWatch.")
			return nil
		}
	}
	return nil
}

func (sm *sriovManager) PreStartContainer(ctx context.Context, psRqt *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (sm *sriovManager) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired: false,
	}, nil
}

//Allocate passes the PCI Addr(s) as an env variable to the requesting container
func (sm *sriovManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := new(pluginapi.AllocateResponse)
	pciAddrs := ""
	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)
		for _, id := range container.DevicesIDs {
			glog.Infof("DeviceID in Allocate: %v", id)
			dev, ok := sm.devices[id]
			if !ok {
				glog.Errorf("Error. Invalid allocation request with non-existing device %s", id)
				return nil, fmt.Errorf("Error. Invalid allocation request with non-existing device %s", id)
			}
			if dev.Health != pluginapi.Healthy {
				glog.Errorf("Error. Invalid allocation request with unhealthy device %s", id)
				return nil, fmt.Errorf("Error. Invalid allocation request with unhealthy device %s", id)
			}

			pciAddrs = pciAddrs + id + ","
		}

		glog.Infof("PCI Addrs allocated: %s", pciAddrs)
		envmap := make(map[string]string)
		envmap["SRIOV-VF-PCI-ADDR"] = pciAddrs

		containerResp.Envs = envmap
		resp.ContainerResponses = append(resp.ContainerResponses, containerResp)
	}
	return resp, nil
}

func main() {
	flag.Parse()
	defer glog.Flush()
	glog.Infof("Starting SRIOV Network Device Plugin...")
	sm := newSriovManager()
	if sm == nil {
		glog.Errorf("Unable to get instance of a SRIOV-Manager")
		return
	}
	sm.cleanup()

	// respond to syscalls for termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start server
	if err := sm.Start(); err != nil {
		glog.Errorf("sriovManager.Start() failed: %v", err)
		return
	}

	// Start plugin endpoint watcher
	go sm.Watch()

	// Catch termination signals
	select {
	case sig := <-sigCh:
		glog.Infof("Received signal \"%v\", shutting down.", sig)
		sm.Stop()
		return
	}
}
