package resources

import (
	"os"
	"time"

	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"

	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	Describe("creating new instance of resource server", func() {
		Context("valid arguments are passed", func() {
			var rs *resourceServer
			BeforeEach(func() {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fakename")
				obj := newResourceServer("fakeprefix", "fakesuffix", &rp)
				rs = obj.(*resourceServer)
				rs.sockDir = fs.RootDir
			})
			It("should have the properties correctly assigned", func() {
				Expect(rs.resourcePool.GetResourceName()).To(Equal("fakename"))
				Expect(rs.resourceNamePrefix).To(Equal("fakeprefix"))
				Expect(rs.endPoint).To(Equal("fakename.fakesuffix"))
			})
		})
	})
	DescribeTable("registering with Kubelet",
		func(shouldRunServer, shouldServerFail, shouldFail bool) {
			fs := &utils.FakeFilesystem{}
			defer fs.Use()()
			rp := mocks.ResourcePool{}
			rp.On("GetResourceName").Return("fakename")
			obj := newResourceServer("fakeprefix", "fakesuffix", &rp)
			rs := obj.(*resourceServer)
			rs.sockDir = fs.RootDir
			registrationServer := createFakeRegistrationServer(fs.RootDir, shouldServerFail)
			if shouldRunServer {
				os.MkdirAll(pluginapi.DevicePluginPath, 0755)
				registrationServer.start()
			}
			err := rs.register()
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			if shouldRunServer {
				registrationServer.stop()
			}
		},
		Entry("when can't connect to Kubelet should fail", false, true, true),
		Entry("when device plugin unable to register with Kubelet should fail", true, true, true),
		Entry("succesfully shouldn't fail", true, false, false),
	)
	Describe("initializating server", func() {
		Context("in all scenarios", func() {
			var err error
			BeforeEach(func() {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com")
				rs := newResourceServer("fakeprefix", "fakesuffix", &rp).(*resourceServer)
				rs.sockDir = fs.RootDir
				err = rs.Init()
			})
			It("should never fail", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
	Describe("resource server lifecycle", func() {
		// integration-like test for the resource server (positive cases)
		var (
			fakeConf *types.ResourceConfig
			rp       mocks.ResourcePool
			fs       *utils.FakeFilesystem
		)
		BeforeEach(func() {
			fakeConf = &types.ResourceConfig{
				ResourceName: "fake",
				Selectors: struct {
					Vendors []string `json:"vendors,omitempty"`
					Devices []string `json:"devices,omitempty"`
					Drivers []string `json:"drivers,omitempty"`
				}{[]string{}, []string{"fakeid"}, []string{}},
			}
			rp = mocks.ResourcePool{}
			rp.On("GetConfig").Return(fakeConf).
				On("GetResourceName").Return("fake.com").
				On("DiscoverDevices").Return(nil).
				On("GetDevices").Return(map[string]*pluginapi.Device{}).
				On("Probe").Return(true)
			fs = &utils.FakeFilesystem{}
		})
		Context("starting, restarting and stopping the resource server", func() {
			It("should not fail and messages should be received on the channels", func(done Done) {
				defer fs.Use()()
				rs := newResourceServer("fake.com", "fake", &rp).(*resourceServer)
				rs.sockDir = fs.RootDir

				registrationServer := createFakeRegistrationServer(fs.RootDir, false)
				os.MkdirAll(pluginapi.DevicePluginPath, 0755)
				registrationServer.start()
				defer registrationServer.stop()

				err := rs.Start()
				Expect(err).NotTo(HaveOccurred())

				err = rs.restart()
				Expect(err).NotTo(HaveOccurred())
				Eventually(rs.termSignal, time.Second*10).Should(Receive())

				go func() {
					err := rs.Stop()
					Expect(err).NotTo(HaveOccurred())
				}()
				Eventually(rs.termSignal, time.Second*10).Should(Receive())
				Eventually(rs.stopWatcher, time.Second*10).Should(Receive())

				close(done)
			}, 12.0)
		})
		Context("starting, watching and stopping the resource server", func() {
			It("should not fail and messages should be received on the channels", func(done Done) {
				defer fs.Use()()
				rs := newResourceServer("fake.com", "fake", &rp).(*resourceServer)
				rs.sockDir = fs.RootDir

				registrationServer := createFakeRegistrationServer(fs.RootDir, false)
				os.MkdirAll(pluginapi.DevicePluginPath, 0755)
				registrationServer.start()
				defer registrationServer.stop()

				// start server to register with fake kubelet
				err := rs.Start()
				Expect(err).NotTo(HaveOccurred())

				// run socket watcher in background as in real-life
				go rs.Watch()

				// sleep 1 second to let watcher perform at least a single socket-file check
				time.Sleep(time.Second)
				err = rs.Stop()
				Expect(err).NotTo(HaveOccurred())

				Eventually(rs.termSignal, time.Second*10).Should(Receive())

				close(done)
			}, 12.0)
		})
	})

	DescribeTable("allocating",
		func(req *pluginapi.AllocateRequest, expectedRespLength int, shouldFail bool) {
			rp := mocks.ResourcePool{}
			rp.On("GetResourceName").
				Return("fake.com").
				On("GetDeviceFiles").
				Return(map[string]string{"00:00.01": "/dev/fake"}).
				On("GetDeviceSpecs", []string{"00:00.01"}).
				Return([]*pluginapi.DeviceSpec{{ContainerPath: "/dev/fake", HostPath: "/dev/fake", Permissions: "rw"}}).
				On("GetEnvs", []string{"00:00.01"}).
				Return([]string{"00:00.01"}).
				On("GetMounts", []string{"00:00.01"}).
				Return([]*pluginapi.Mount{{ContainerPath: "/dev/fake", HostPath: "/dev/fake", ReadOnly: false}})

			rs := newResourceServer("fake.com", "fake", &rp).(*resourceServer)

			resp, err := rs.Allocate(nil, req)

			Expect(len(resp.GetContainerResponses())).To(Equal(expectedRespLength))

			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("allocating succesfully 1 deviceID",
			&pluginapi.AllocateRequest{
				ContainerRequests: []*pluginapi.ContainerAllocateRequest{{DevicesIDs: []string{"00:00.01"}}},
			},
			1,
			false,
		),
		PEntry("allocating deviceID that does not exist",
			&pluginapi.AllocateRequest{
				ContainerRequests: []*pluginapi.ContainerAllocateRequest{{DevicesIDs: []string{"00:00.02"}}},
			},
			0,
			true,
		),
		Entry("empty AllocateRequest", &pluginapi.AllocateRequest{}, 0, false),
	)
	Describe("running PreStartContainer", func() {
		It("should not fail", func() {
			rs := &resourceServer{}
			resp, err := rs.PreStartContainer(nil, nil)
			Expect(resp).NotTo(Equal(nil))
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("running GetDevicePluginOptions", func() {
		It("should not fail", func() {
			rs := &resourceServer{}
			resp, err := rs.GetDevicePluginOptions(nil, nil)
			Expect(resp).NotTo(Equal(nil))
			Expect(err).NotTo(HaveOccurred())
		})
	})
	DescribeTable("getting env variables",
		func(in []string, expected map[string]string) {
			fs := &utils.FakeFilesystem{}
			defer fs.Use()()
			deviceIDs := []string{"fakeid"}
			rp := mocks.ResourcePool{}
			rp.
				On("GetResourceName").Return("fake").
				On("GetEnvs", deviceIDs).Return(in)

			obj := newResourceServer("fake.com", "fake", &rp)
			rs := *obj.(*resourceServer)
			rs.sockDir = fs.RootDir
			actual := rs.getEnvs(deviceIDs)
			for k, v := range expected {
				Expect(actual).To(HaveKeyWithValue(k, v))
			}
		},
		Entry("some values",
			[]string{"fakeId0", "fakeId1"},
			map[string]string{
				"PCIDEVICE_FAKE_COM_FAKE": "fakeId0,fakeId1",
			},
		),
	)

	Describe("ListAndWatch", func() {
		Context("when first Send call in DevicePlugin_ListAndWatch failed", func() {
			It("should fail", func() {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com").
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.01": {ID: "00:00.01", Health: "Healthy"}}).Once()

				rs := newResourceServer("fake.com", "fake", &rp).(*resourceServer)
				rs.sockDir = fs.RootDir

				lwSrv := &fakeListAndWatchServer{
					resourceServer: rs,
					sendCallToFail: 1,
				}

				err := rs.ListAndWatch(&pluginapi.Empty{}, lwSrv)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("when Send call in DevicePlugin_ListAndWatch breaks", func() {
			It("should receive not fail", func(done Done) {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com").
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.01": {ID: "00:00.01", Health: "Healthy"}}).Once().
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.02": {ID: "00:00.02", Health: "Healthy"}}).Once()

				rs := newResourceServer("fake.com", "fake", &rp).(*resourceServer)
				rs.sockDir = fs.RootDir

				lwSrv := &fakeListAndWatchServer{
					resourceServer: rs,
					sendCallToFail: 2,
					updates:        make(chan bool),
				}

				// run ListAndWatch which will send initial update
				var err error
				go func() {
					err = rs.ListAndWatch(&pluginapi.Empty{}, lwSrv)
					// because DevicePlugin_ListAndWatch breaks...
					Expect(err).To(HaveOccurred())
				}()

				// wait for the initial update to reach ListAndWatchServer
				Eventually(lwSrv.updates, time.Second*30).Should(Receive())
				// this time it should break
				rs.updateSignal <- true
				Eventually(lwSrv.updates, time.Second*30).ShouldNot(Receive())

				close(done)
			}, 60.0)
		})
		Context("when received multiple update requests and then the term signal", func() {
			It("should receive not fail", func(done Done) {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com").
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.01": {ID: "00:00.01", Health: "Healthy"}}).Once().
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.02": {ID: "00:00.02", Health: "Healthy"}}).Once()

				rs := newResourceServer("fake.com", "fake", &rp).(*resourceServer)
				rs.sockDir = fs.RootDir

				lwSrv := &fakeListAndWatchServer{
					resourceServer: rs,
					sendCallToFail: 0, // no failures on purpose
					updates:        make(chan bool),
				}

				// run ListAndWatch which will send initial update
				var err error
				go func() {
					err = rs.ListAndWatch(&pluginapi.Empty{}, lwSrv)
					Expect(err).NotTo(HaveOccurred())
				}()

				// wait for the initial update to reach ListAndWatchServer
				Eventually(lwSrv.updates, time.Second*10).Should(Receive())

				// send another set of updates and wait for the ListAndWatchServer
				rs.updateSignal <- true
				Eventually(lwSrv.updates, time.Second*10).Should(Receive())

				// finally send term signal
				rs.termSignal <- true

				close(done)
			}, 30.0)
		})
	})
})
