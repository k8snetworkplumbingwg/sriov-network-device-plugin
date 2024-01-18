package resources

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	CDImocks "github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/cdi/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/types/mocks"
	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	Describe("creating new instance of resource server", func() {
		Context("valid arguments are passed", func() {
			var rs *resourceServer
			rp := mocks.ResourcePool{}
			BeforeEach(func() {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp.On("GetResourceName").Return("fakename")
			})
			It("should have the properties correctly assigned when plugin watcher enabled", func() {
				// Create ResourceServer with plugin watch mode enabled
				obj := NewResourceServer("fakeprefix", "fakesuffix", true, false, &rp)
				rs = obj.(*resourceServer)
				Expect(rs.resourcePool.GetResourceName()).To(Equal("fakename"))
				Expect(rs.resourceNamePrefix).To(Equal("fakeprefix"))
				Expect(rs.endPoint).To(Equal("fakeprefix_fakename.fakesuffix"))
				Expect(rs.pluginWatch).To(Equal(true))
				Expect(rs.sockPath).To(Equal(filepath.Join(types.SockDir, "fakeprefix_fakename.fakesuffix")))
			})
			It("should have the properties correctly assigned when plugin watcher disabled", func() {
				// Create ResourceServer with plugin watch mode disabled
				obj := NewResourceServer("fakeprefix", "fakesuffix", false, false, &rp)
				rs = obj.(*resourceServer)
				Expect(rs.resourcePool.GetResourceName()).To(Equal("fakename"))
				Expect(rs.resourceNamePrefix).To(Equal("fakeprefix"))
				Expect(rs.endPoint).To(Equal("fakeprefix_fakename.fakesuffix"))
				Expect(rs.pluginWatch).To(Equal(false))
				Expect(rs.sockPath).To(Equal(filepath.Join(types.DeprecatedSockDir,
					"fakeprefix_fakename.fakesuffix")))
			})
		})
	})
	DescribeTable("registering with Kubelet",
		func(shouldRunServer, shouldEnablePluginWatch, shouldServerFail, shouldFail bool) {
			t := GinkgoT()
			var err error
			fs := &utils.FakeFilesystem{}
			defer fs.Use()()
			rp := mocks.ResourcePool{}
			rp.On("Probe").Return(false)
			rp.On("GetResourceName").Return("fakename")
			rp.On("StoreDeviceInfoFile", "fakeprefix").Return(nil)
			rp.On("CleanDeviceInfoFile", "fakeprefix").Return(nil)

			// Use faked dir as socket dir
			types.SockDir = fs.RootDir
			types.DeprecatedSockDir = fs.RootDir

			obj := NewResourceServer("fakeprefix", "fakesuffix", shouldEnablePluginWatch, false, &rp)
			rs := obj.(*resourceServer)

			registrationServer := createFakeRegistrationServer(fs.RootDir,
				"fakeprefix_fakename.fakesuffix", shouldServerFail, shouldEnablePluginWatch)

			if shouldRunServer {
				if shouldEnablePluginWatch {
					_ = rs.Start()
					rp.AssertCalled(t, "CleanDeviceInfoFile", "fakeprefix")
					rp.AssertCalled(t, "StoreDeviceInfoFile", "fakeprefix")
				} else {
					_ = os.MkdirAll(pluginapi.DevicePluginPath, 0755)
					registrationServer.start()
				}
			}
			if shouldEnablePluginWatch {
				err = registrationServer.registerPlugin()
			} else {
				err = rs.register()
			}
			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
			if shouldRunServer {
				if shouldEnablePluginWatch {
					rs.grpcServer.Stop()
				} else {
					registrationServer.stop()
				}
			}
		},
		Entry("when can't connect to Kubelet should fail", false, false, true, true),
		Entry("when device plugin unable to register with Kubelet should fail", true, false, true, true),
		Entry("when Kubelet unable to register with device plugin should fail", true, true, true, true),
		Entry("successfully shouldn't fail", true, false, false, false),
		Entry("successfully shouldn't fail with plugin watcher enabled", true, true, false, false),
	)
	Describe("initializating server", func() {
		Context("in all scenarios", func() {
			var err error
			BeforeEach(func() {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com")
				rs := NewResourceServer("fakeprefix", "fakesuffix", true, false, &rp).(*resourceServer)
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
			fs       *utils.FakeFilesystem
		)
		t := GinkgoT()
		BeforeEach(func() {
			var selectors json.RawMessage
			err := selectors.UnmarshalJSON([]byte(`[{"devices": ["fakeid"]}]`))
			Expect(err).NotTo(HaveOccurred())

			fakeConf = &types.ResourceConfig{
				ResourceName: "fake",
				Selectors:    &selectors,
			}
			fs = &utils.FakeFilesystem{}
		})
		Context("starting, restarting and stopping the resource server", func() {
			It("should not fail and messages should be received on the channels", func(done Done) {
				defer fs.Use()()
				// Use faked dir as socket dir
				types.DeprecatedSockDir = fs.RootDir
				rp := mocks.ResourcePool{}
				rp.On("GetConfig").Return(fakeConf).
					On("GetResourceName").Return("fake.com").
					On("DiscoverDevices").Return(nil).
					On("GetDevices").Return(map[string]*pluginapi.Device{}).
					On("Probe").Return(true).
					On("StoreDeviceInfoFile", "fake").Return(nil).
					On("CleanDeviceInfoFile", "fake").Return(nil)

				// Create ResourceServer with plugin watch mode disabled
				rs := NewResourceServer("fake", "fake", false, false, &rp).(*resourceServer)

				registrationServer := createFakeRegistrationServer(fs.RootDir,
					"fake_fake.com.fake", false, false)
				_ = os.MkdirAll(pluginapi.DevicePluginPath, 0755)

				registrationServer.start()
				defer registrationServer.stop()

				err := rs.Start()
				Expect(err).NotTo(HaveOccurred())

				err = rs.restart()
				Expect(err).NotTo(HaveOccurred())
				Eventually(rs.termSignal, time.Second*10).Should(Receive())

				go func() {
					rp.On("CleanDeviceInfoFile", "fake").Return(nil)
					err := rs.Stop()
					Expect(err).NotTo(HaveOccurred())
					rp.AssertCalled(t, "CleanDeviceInfoFile", "fake")
				}()
				Eventually(rs.termSignal, time.Second*10).Should(Receive())
				Eventually(rs.stopWatcher, time.Second*10).Should(Receive())

				close(done)
			}, 12.0)
			It("should not fail and messages should be received on the channels", func(done Done) {
				defer fs.Use()()
				// Use faked dir as socket dir
				types.SockDir = fs.RootDir

				rp := mocks.ResourcePool{}
				rp.On("GetConfig").Return(fakeConf).
					On("GetResourceName").Return("fake.com").
					On("DiscoverDevices").Return(nil).
					On("GetDevices").Return(map[string]*pluginapi.Device{}).
					On("Probe").Return(true).
					On("StoreDeviceInfoFile", "fake").Return(nil).
					On("CleanDeviceInfoFile", "fake").Return(nil)
				// Create ResourceServer with plugin watch mode enabled
				rs := NewResourceServer("fake", "fake", true, false, &rp).(*resourceServer)

				registrationServer := createFakeRegistrationServer(fs.RootDir,
					"fake_fake.com.fake", false, true)
				err := rs.Start()
				Expect(err).NotTo(HaveOccurred())

				err = registrationServer.registerPlugin()
				Expect(err).NotTo(HaveOccurred())

				go func() {
					rp.On("CleanDeviceInfoFile", "fake").Return(nil)
					err := rs.Stop()
					Expect(err).NotTo(HaveOccurred())
					rp.AssertCalled(t, "CleanDeviceInfoFile", "fake")
				}()
				Eventually(rs.termSignal, time.Second*10).Should(Receive())

				close(done)
			}, 12.0)
		})
		Context("starting, watching and stopping the resource server", func() {
			It("should not fail and messages should be received on the channels", func(done Done) {
				defer fs.Use()()
				// Use faked dir as socket dir
				types.DeprecatedSockDir = fs.RootDir
				rp := mocks.ResourcePool{}
				rp.On("GetConfig").Return(fakeConf).
					On("GetResourceName").Return("fake.com").
					On("DiscoverDevices").Return(nil).
					On("GetDevices").Return(map[string]*pluginapi.Device{}).
					On("Probe").Return(true).
					On("StoreDeviceInfoFile", "fake").Return(nil).
					On("CleanDeviceInfoFile", "fake").Return(nil)

				// Create ResourceServer with plugin watch mode disabled
				rs := NewResourceServer("fake", "fake", false, false, &rp).(*resourceServer)

				registrationServer := createFakeRegistrationServer(fs.RootDir,
					"fake_fake.com.fake", false, false)
				_ = os.MkdirAll(pluginapi.DevicePluginPath, 0755)

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
				On("GetEnvs", "fake.com", []string{"00:00.01"}).
				Return(map[string]string{"PCIDEVICE_FAKE_COM_FAKE_INFO": "{\"00:00.01\":{\"netdevice\":{\"pci\":\"00:00.01\"}}}"}, nil).
				On("GetMounts", []string{"00:00.01"}).
				Return([]*pluginapi.Mount{{ContainerPath: "/dev/fake", HostPath: "/dev/fake", ReadOnly: false}})

			rs := NewResourceServer("fake.com", "fake", true, false, &rp).(*resourceServer)

			resp, err := rs.Allocate(context.TODO(), req)

			Expect(len(resp.GetContainerResponses())).To(Equal(expectedRespLength))

			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("allocating successfully 1 deviceID",
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
	DescribeTable("allocating with CDI",
		func(req *pluginapi.AllocateRequest, expectedRespLength int, shouldFail bool) {
			rp := mocks.ResourcePool{}
			rp.On("GetResourceName").
				Return("fake.com").
				On("GetDeviceFiles").
				Return(map[string]string{"00:00.01": "/dev/fake"}).
				On("GetDeviceSpecs", []string{"00:00.01"}).
				Return([]*pluginapi.DeviceSpec{{ContainerPath: "/dev/fake", HostPath: "/dev/fake", Permissions: "rw"}}).
				On("GetEnvs", "fake.com", []string{"00:00.01"}).
				Return(map[string]string{"PCIDEVICE_FAKE_COM_FAKE_INFO": "{\"00:00.01\":{\"netdevice\":{\"pci\":\"00:00.01\"}}}"}, nil).
				On("GetMounts", []string{"00:00.01"}).
				Return([]*pluginapi.Mount{{ContainerPath: "/dev/fake", HostPath: "/dev/fake", ReadOnly: false}}).
				On("GetCDIName").
				Return("fake.com")

			rs := NewResourceServer("fake.com", "fake", true, true, &rp).(*resourceServer)

			cdi := &CDImocks.CDI{}
			cdi.On("CreateCDISpecForPool", "fake.com", &rp).Return(nil).Twice().
				On("CreateContainerAnnotations", []string{"00:00.01"}, "fake.com", "fake.com").
				Return(map[string]string{"00:00.01": "fake.com/net=00:00.01"}, nil)
			rs.cdi = cdi

			resp, err := rs.Allocate(context.TODO(), req)

			Expect(len(resp.GetContainerResponses())).To(Equal(expectedRespLength))

			if shouldFail {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("allocating successfully 1 deviceID",
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
			resp, err := rs.PreStartContainer(context.TODO(), nil)
			Expect(resp).NotTo(Equal(nil))
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("running GetDevicePluginOptions", func() {
		It("should not fail", func() {
			rs := &resourceServer{}
			resp, err := rs.GetDevicePluginOptions(context.TODO(), nil)
			Expect(resp).NotTo(Equal(nil))
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Describe("ListAndWatch", func() {
		Context("when first Send call in DevicePlugin_ListAndWatch failed", func() {
			It("should fail", func() {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com").
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.01": {ID: "00:00.01", Health: "Healthy"}}).Once()

				rs := NewResourceServer("fake.com", "fake", true, false, &rp).(*resourceServer)
				rs.sockPath = fs.RootDir

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

				rs := NewResourceServer("fake.com", "fake", true, false, &rp).(*resourceServer)
				rs.sockPath = fs.RootDir

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

				rs := NewResourceServer("fake.com", "fake", true, false, &rp).(*resourceServer)
				rs.sockPath = fs.RootDir

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
		Context("when CDI is enabled", func() {
			It("should not fail", func(done Done) {
				fs := &utils.FakeFilesystem{}
				defer fs.Use()()
				rp := mocks.ResourcePool{}
				rp.On("GetResourceName").Return("fake.com").
					On("GetDevices").Return(map[string]*pluginapi.Device{"00:00.01": {ID: "00:00.01", Health: "Healthy"}}).Twice().
					On("GetResourcePrefix").Return("fake.com").Twice()

				rs := NewResourceServer("fake.com", "fake", true, true, &rp).(*resourceServer)
				rs.sockPath = fs.RootDir

				cdi := &CDImocks.CDI{}
				cdi.On("CreateCDISpecForPool", "fake.com", &rp).Return(nil).Twice()
				rs.cdi = cdi

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
			}, 30)
		})
	})
})
