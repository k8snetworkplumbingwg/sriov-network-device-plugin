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
	"syscall"
	"time"

	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/intel/sriov-network-device-plugin/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	netDirectory     = "/sys/class/net/"
	sriov_capabale   = "/sriov_totalvfs"
	sriov_configured = "/sriov_numvfs"

	// Device plugin settings.
	pluginMountPath      = "/var/lib/kubelet/device-plugins"
	kubeletEndpoint      = "kubelet.sock"
	pluginEndpointPrefix = "sriovNet"
	resourceName         = "intel.com/sriov"
)

// sriovManager manages sriov networking devices
type sriovManager struct {
	defaultDevices   []string
	socketFile       string
	devices          map[string]pluginapi.Device   // for Kubelet DP API
	managedDevices   map[string]*api.VfInformation // for internal use (key: pciaddr; value: VfInfo)
	grpcServer       *grpc.Server
	allocatedDevices map[string][]string
}

func NewSriovManager() *sriovManager {

	return &sriovManager{
		devices:          make(map[string]pluginapi.Device),
		allocatedDevices: make(map[string][]string),
		managedDevices:   make(map[string]*api.VfInformation),
		socketFile:       fmt.Sprintf("%s.sock", pluginEndpointPrefix),
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
		sriovFilePath := filepath.Join(netDirectory, dev.Name(), "device", "sriov_numvfs")
		glog.Infof("Checking for file %s ", sriovFilePath)

		if f, err := os.Lstat(sriovFilePath); !os.IsNotExist(err) {
			if f.Mode().IsRegular() { // and its a regular file
				sriovNetDevices = append(sriovNetDevices, dev.Name())
			}
		}
	}

	return sriovNetDevices, nil
}

//Reads DeviceName and gets PCI Addresses of VFs configured
func (sm *sriovManager) discoverNetworks() error {

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
		sriovcapablepath := filepath.Join(netDirectory, dev, "device", sriov_capabale)
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
			sriovconfiguredpath := netDirectory + dev + "/device" + sriov_configured
			vfs, err = ioutil.ReadFile(sriovconfiguredpath)
			if err != nil {
				glog.Errorf("Error. Could not read sriov_numvfs file. SRIOV error. %v", err)
				return err
			}
			configured_vfs := bytes.TrimSpace(vfs)
			numconfiguredvfs, err := strconv.Atoi(string(configured_vfs))
			if err != nil {
				glog.Errorf("Error. Could not parse sriov_numvfs files. Skipping device. Err: %v", err)
				return err
			}
			glog.Infof("Number of Configured VFs for device %v is %v", dev, string(configured_vfs))

			//get PCI IDs for VFs
			for vf := 0; vf < numconfiguredvfs; vf++ {
				vfDir := fmt.Sprintf("/sys/class/net/%s/device/virtfn%d", dev, vf)
				dirInfo, err := os.Lstat(vfDir)
				if err != nil {
					glog.Errorf("Error. Could not get directory information for device: %s, VF: %v. Err: %v", dev, vf, err)
					return err
				}

				if (dirInfo.Mode() & os.ModeSymlink) == 0 {
					glog.Errorf("Error. No symbolic link between virtual function and PCI - Device: %s, VF: %v", dev, vf)
					return fmt.Errorf("Error. No symbolic link between virtual function and PCI - Device: %s, VF: %v", dev, vf)
				}

				pciInfo, err := os.Readlink(vfDir)
				if err != nil {
					glog.Errorf("Error. Cannot read symbolic link between virtual function and PCI - Device: %s, VF: %v. Err: %v", dev, vf, err)
					return err
				}

				pciAddr := pciInfo[len("../"):]
				glog.Infof("PCI Address for device %s, VF %v is %s", dev, vf, pciAddr)

				devName := pciAddr
				sm.devices[devName] = pluginapi.Device{devName, pluginapi.Healthy}
				sm.managedDevices[devName] = &api.VfInformation{Pciaddr: pciAddr, Vfid: int32(vf), Pfname: dev}
			}

		}
	}
	return nil
}

func (sm *sriovManager) GetDeviceState(DeviceName string) string {
	// TODO: Discover device health
	return pluginapi.Healthy
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

	// Register all services
	pluginapi.RegisterDevicePluginServer(sm.grpcServer, sm)
	api.RegisterSendPodInformationServer(sm.grpcServer, sm)

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
	return nil
}

func (sm *sriovManager) Stop() error {
	glog.Infof("Stopping SRIOV Network Device Plugin gRPC server..")
	if sm.grpcServer == nil {
		return nil
	}

	sm.grpcServer.Stop()
	sm.grpcServer = nil

	return sm.cleanup()
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

// Act as a grpc client and register with the kubelet.
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
func (sm *sriovManager) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	changed := true
	for {
		for id, dev := range sm.devices {
			state := sm.GetDeviceState(id)
			if dev.Health != state {
				changed = true
				dev.Health = state
				sm.devices[id] = dev
			}
		}
		if changed {
			resp := new(pluginapi.ListAndWatchResponse)
			for _, dev := range sm.devices {
				resp.Devices = append(resp.Devices, &pluginapi.Device{dev.ID, dev.Health})
			}
			glog.Infof("ListAndWatch: send devices %v\n", resp)
			if err := stream.Send(resp); err != nil {
				glog.Errorf("Error. Cannot update device states: %v\n", err)
				sm.grpcServer.Stop()
				return err
			}
		}
		changed = false
		time.Sleep(5 * time.Second)
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

//API Change: Pod Information passed in here
//Allocate passes the PCI Addr(s) as an env variable to the requesting container
func (sm *sriovManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := new(pluginapi.AllocateResponse)
	pciAddrs := ""
	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)
		glog.Infof("PodUID: %v & Container Name: %v in Allocate", container.PodUID, container.ContName)
		for _, id := range container.DevicesIDs {
			glog.Infof("DeviceID in Allocate: %v", id)
			//PodUID to PCI mapping for CNI Communication
			sm.allocatedDevices[string(container.PodUID)] = append(sm.allocatedDevices[string(container.PodUID)], id)
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

//gRPC Server implementation of CNI/Device communication
//CNI sends Pod Name and Pod Namespace - use inclusterconfig to get PodUID
func (sm *sriovManager) SendPodInformation(ctx context.Context, podInfo *api.PodInformation) (*api.DeviceInformation, error) {
	glog.Infof("SendPodInformation in Device Plugin. Recieved Pod Name: %v Pod Namespace: %v", podInfo.PodName, podInfo.PodNamespace)
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("Error. Could not get InClusterConfig to create K8s Client. %v", err)
		return &api.DeviceInformation{}, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorf("Error. Could not create K8s Client using supplied config. %v", err)
		return &api.DeviceInformation{}, err
	}
	pod, err := clientset.CoreV1().Pods(podInfo.PodNamespace).Get((podInfo.PodName), metav1.GetOptions{})
	if err != nil {
		glog.Errorf("Error. Could not get Pod Information from K8s Client. %v", err)
		return &api.DeviceInformation{}, err
	}
	glog.Infof("Pod UID from API server %v", pod.UID)
	pciForPod := sm.allocatedDevices[string(pod.UID)]
	glog.Infof("PCI for Pod: %v", pciForPod)

	vfInfos := []*api.VfInformation{}
	for _, p := range pciForPod {
		vfInfos = append(vfInfos, sm.managedDevices[p])
	}
	return &api.DeviceInformation{VFs: vfInfos}, nil
}

func main() {
	flag.Parse()
	glog.Infof("SRIOV Network Device Plugin started...")
	sm := NewSriovManager()
	sm.cleanup()

	// respond to syscalls for termination
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start server
	if err := sm.Start(); err != nil {
		glog.Errorf("sriovManager.Start() failed: %v", err)
		return
	}

	// Registers with Kubelet.
	err := Register(path.Join(pluginMountPath, kubeletEndpoint), sm.socketFile, resourceName)
	if err != nil {
		// Stop server
		sm.grpcServer.Stop()
		glog.Fatal(err)
		return
	}
	glog.Infof("SRIOV Network Device Plugin registered with the Kubelet")

	// Catch termination signals
	select {
	case sig := <-sigCh:
		glog.Infof("Received signal \"%v\", shutting down.", sig)
		sm.Stop()
		return
	}
}
