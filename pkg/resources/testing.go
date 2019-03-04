package resources

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
)

var (
	root   *os.File
	tmpDir string
)

// Implementation of pluginapi.RegistrationServer for use in tests.
type fakeRegistrationServer struct {
	grpcServer     *grpc.Server
	sockDir        string
	pluginEndpoint string
	failOnRegister bool
}

func createFakeRegistrationServer(sockDir string, failOnRegister bool) *fakeRegistrationServer {
	return &fakeRegistrationServer{
		sockDir:        sockDir,
		failOnRegister: failOnRegister,
	}
}

func (s *fakeRegistrationServer) Register(ctx context.Context, r *pluginapi.RegisterRequest) (api *pluginapi.Empty, err error) {
	s.pluginEndpoint = r.Endpoint
	api = &pluginapi.Empty{}
	if s.failOnRegister {
		err = fmt.Errorf("fake registering error")
	}
	return
}

func (s *fakeRegistrationServer) start() {
	l, err := net.Listen("unix", path.Join(s.sockDir, types.KubeEndPoint))
	if err != nil {
		panic(err)
	}
	s.grpcServer = grpc.NewServer()
	pluginapi.RegisterRegistrationServer(s.grpcServer, s)
	go s.grpcServer.Serve(l)
	s.waitForServer(5 * time.Second)
}

func (s *fakeRegistrationServer) waitForServer(timeout time.Duration) error {
	maxWaitTime := time.Now().Add(timeout)
	for {
		if time.Now().After(maxWaitTime) {
			return fmt.Errorf("waiting for the fake registration server timed out")
		}
		c, err := net.DialTimeout("unix", path.Join(s.sockDir, types.KubeEndPoint), time.Second)
		if err == nil {
			return c.Close()
		}
	}
}

func (s *fakeRegistrationServer) stop() {
	s.grpcServer.Stop()
	os.Remove(pluginapi.DevicePluginPath)
}

// Implementation of pluginapi.DevicePlugin_ListAndWatchServer for use in tests.
type fakeListAndWatchServer struct {
	resourceServer *resourceServer
	sendCallToFail int // defines which Send() call should return error, set to 0 if none
	sendCalls      int // Send() calls counter
	updates        chan bool
}

// Records that update has been received and fails or not depending on the fake server configuration.
func (s *fakeListAndWatchServer) Send(resp *pluginapi.ListAndWatchResponse) error {
	s.sendCalls++
	if s.sendCallToFail == s.sendCalls {
		return fmt.Errorf("Fake error")
	}
	s.updates <- true
	return nil
}

// Mandatory to implement pluginapi.DevicePlugin_ListAndWatchServer
func (s *fakeListAndWatchServer) Context() context.Context {
	return nil
}

func (s *fakeListAndWatchServer) RecvMsg(m interface{}) error {
	return nil
}

func (s *fakeListAndWatchServer) SendMsg(m interface{}) error {
	return nil
}

func (s *fakeListAndWatchServer) SendHeader(m metadata.MD) error {
	return nil
}

func (s *fakeListAndWatchServer) SetHeader(m metadata.MD) error {
	return nil
}

func (s *fakeListAndWatchServer) SetTrailer(m metadata.MD) {
}
