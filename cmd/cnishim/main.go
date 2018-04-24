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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	pb "github.com/intel/sriov-network-device-plugin/api"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// DelegateCmd is enum type used for defining delegateExec commands
type DelegateCmd int

// constants defining Device Plugin socket path, default data dir, DelegateCmd enum
const (
	dpMountPath                = "/var/lib/kubelet/device-plugins"
	defaultDataDir             = "/var/lib/cni/cnishim"
	Add            DelegateCmd = 0 // Add command
	Del            DelegateCmd = 1 // Delete command
)

// PluginConf holds configurations taken from stdin as json.
type PluginConf struct {
	types.NetConf
	DataDir      string                 `json:"dataDir"`
	DevicePlugin string                 `json:"deviceplugin"`
	Delegate     map[string]interface{} `json:"delegate"`
}

// K8sArgs is the valid CNI_ARGS used by Kubernetes to add Pod info
type K8sArgs struct {
	types.CommonArgs
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

func init() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()
}

// parseConfig parses the given configuration from stdin.
func parseConfig(stdin []byte) (*PluginConf, error) {
	conf := &PluginConf{}

	if err := json.Unmarshal(stdin, conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %+v", err)
	}

	// config validation
	if conf.DevicePlugin == "" {
		return nil, fmt.Errorf("deviceplugin must be specified")
	}

	if conf.DataDir == "" {
		conf.DataDir = defaultDataDir
	}

	if conf.Delegate == nil {
		return nil, fmt.Errorf("delegate must be specified")
	}

	if !keyExist(conf.Delegate, "type") && !isValidString(conf.Delegate["type"]) {
		return conf, fmt.Errorf("'delegate' dictionary must have a 'type' field (string)")
	}
	if keyExist(conf.Delegate, "name") {
		return conf, fmt.Errorf("'delegate' dictionary must not have 'name' field, it'll be set by cnishim")
	}

	return conf, nil
}

// Establishes a gRPC connection with the device plugin
func dialToDP(dpEndPointPath string) (pb.SendPodInformationClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(dpEndPointPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))

	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial to grpc server %v", err)
	}

	return pb.NewSendPodInformationClient(conn), conn, nil
}

func getDeviceInfo(k *K8sArgs, dpEndpointPrefix string) (*pb.VfInformation, error) {

	// Set up a connection to the gRPC server.
	dpEndPoint := fmt.Sprintf("%s.sock", dpEndpointPrefix)
	dpEndPointPath := filepath.Join(dpMountPath, dpEndPoint)

	client, conn, err := dialToDP(dpEndPointPath)
	if err != nil {
		fmt.Printf("error making connection to grpc server using %v : %v", dpEndPointPath, err)
		return nil, err
	}
	defer conn.Close()

	req := &pb.PodInformation{PodName: string(k.K8S_POD_NAME), PodNamespace: string(k.K8S_POD_NAMESPACE)}
	resp, err := client.SendPodInformation(context.Background(), req)
	if err != nil {
		fmt.Printf("error getting response from device plugin server: %q", err)
		return nil, err
	}

	vfInfo := resp.GetVFs()[0] // For now just work on the first VF
	return vfInfo, nil
}

// Adapted from flannel-cni
// Ref: [https://github.com/containernetworking/plugins/blob/master/plugins/meta/flannel/flannel.go]
func keyExist(m map[string]interface{}, k string) bool {
	_, ok := m[k]
	return ok
}

func isValidString(i interface{}) bool {
	_, ok := i.(string)
	return ok
}

// Save delegate config
func saveDelegateConf(cid, dataDir, ifName string, conf []byte) error {

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("unable to create cnishim data directory %v", err)
	}
	confFileName := fmt.Sprintf("%s-%s", cid, ifName)
	path := filepath.Join(dataDir, confFileName)
	if err := ioutil.WriteFile(path, conf, 0600); err != nil {
		return fmt.Errorf("unable to write delegate config file %v", err)
	}
	return nil
}

// Load delegate conf from file
func loadDelegateConf(cid, dataDir, ifName string) ([]byte, error) {
	confFileName := fmt.Sprintf("%s-%s", cid, ifName)
	path := filepath.Join(dataDir, confFileName)
	defer os.Remove(path)

	return ioutil.ReadFile(path)
}

// Adapted from flannel-cni
// Ref: [https://github.com/containernetworking/plugins/blob/master/plugins/meta/flannel/flannel.go]
func delegateExec(cid, dataDir, ifName string, netconf map[string]interface{}, cmd DelegateCmd) error {
	switch cmd {
	case Add:
		// Save delegate conf
		confAsBytes, err := json.Marshal(netconf)
		if err != nil {
			return fmt.Errorf("error serializing delegate conf: %v", err)
		}
		if err = saveDelegateConf(cid, dataDir, ifName, confAsBytes); err != nil {
			return err
		}
		result, err := invoke.DelegateAdd(netconf["type"].(string), confAsBytes)
		if err != nil {
			return err
		}
		return result.Print()
	case Del:
		// Read delegate conf from file
		delegateConfBytes, err := loadDelegateConf(cid, dataDir, ifName)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Try to parse json bytes to NetConf{} to check if required configs exist
		n := &types.NetConf{}
		if err = json.Unmarshal(delegateConfBytes, n); err != nil {
			return fmt.Errorf("failed to parse netconf: %v", err)
		}

		return invoke.DelegateDel(n.Type, delegateConfBytes)
	default:
		return fmt.Errorf("invalid delegate command")
	}

}

func cmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return fmt.Errorf("Error in loading the CNI-Shim args %v", err)
	}

	// Get Pod info from CNI_Args
	k8sArgs := K8sArgs{}
	err = types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return fmt.Errorf("Error in loading the k8s args %v", err)
	}

	vfInfo, err := getDeviceInfo(&k8sArgs, conf.DevicePlugin)
	if err != nil {
		return err
	}

	conf.Delegate["name"] = conf.Name
	conf.Delegate["deviceid"] = vfInfo.GetPciaddr() // Add in device information
	conf.Delegate["if0"] = vfInfo.GetPfname()
	conf.Delegate["vfid"] = int(vfInfo.GetVfid())

	// TODO: use conf.Args to pass additional device specific info
	// args.Args = args.Args + fmt.Sprintf("deviceid=%s;pfname=%s;vfnum=%d",vfInfo.GetPfname(), vfInfo.GetPciaddr(), int(vfInfo.GetVfid()))

	return delegateExec(args.ContainerID, conf.DataDir, args.IfName, conf.Delegate, Add)
}

func cmdDel(args *skel.CmdArgs) error {
	conf, err := parseConfig(args.StdinData)
	if err != nil {
		return fmt.Errorf("Error in loading the CNI-Shim args %v", err)
	}

	// conf.Delegate is not being used for deletion as configs are read from a file.
	// It's just a dummy parameter for the function call
	return delegateExec(args.ContainerID, conf.DataDir, args.IfName, conf.Delegate, Del)
}

func main() {
	skel.PluginMain(cmdAdd, cmdDel, version.All)
}
