package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/stretchr/testify/mock"
	nl "github.com/vishvananda/netlink"

	"github.com/k8snetworkplumbingwg/sriov-network-device-plugin/pkg/utils/mocks"
)

// FakeFilesystem allows to setup isolated fake files structure used for the tests.
type FakeFilesystem struct {
	RootDir  string
	Dirs     []string
	Files    map[string][]byte
	Symlinks map[string]string
}

// Use function creates entire files structure and returns a function to tear it down. Example usage: defer fs.Use()()
func (fs *FakeFilesystem) Use() func() {
	// create the new fake fs root dir in /tmp/sriov...
	tmpDir, err := ioutil.TempDir("", "sriov")
	if err != nil {
		panic(fmt.Errorf("error creating fake root dir: %s", err.Error()))
	}
	fs.RootDir = tmpDir

	for _, dir := range fs.Dirs {
		//nolint: gomnd
		err := os.MkdirAll(path.Join(fs.RootDir, dir), 0755)
		if err != nil {
			panic(fmt.Errorf("error creating fake directory: %s", err.Error()))
		}
	}
	for filename, body := range fs.Files {
		//nolint: gomnd
		err := os.WriteFile(path.Join(fs.RootDir, filename), body, 0600)
		if err != nil {
			panic(fmt.Errorf("error creating fake file: %s", err.Error()))
		}
	}
	//nolint: gomnd
	err = os.MkdirAll(path.Join(fs.RootDir, "usr/share/hwdata"), 0755)
	if err != nil {
		panic(fmt.Errorf("error creating fake directory: %s", err.Error()))
	}

	// TODO: Remove writing pci.ids file once ghw is mocked
	// This is to fix the CI failure where ghw lib fails to
	// unzip pci.ids file downloaded from internet.
	pciData, err := os.ReadFile("/usr/share/hwdata/pci.ids")
	if err != nil {
		panic(fmt.Errorf("error reading file: %s", err.Error()))
	}
	//nolint: gomnd
	err = os.WriteFile(path.Join(fs.RootDir, "usr/share/hwdata/pci.ids"), pciData, 0600)
	if err != nil {
		panic(fmt.Errorf("error creating fake file: %s", err.Error()))
	}

	for link, target := range fs.Symlinks {
		err = os.Symlink(target, path.Join(fs.RootDir, link))
		if err != nil {
			panic(fmt.Errorf("error creating fake symlink: %s", err.Error()))
		}
	}

	sysBusPci = path.Join(fs.RootDir, "/sys/bus/pci/devices")

	return func() {
		// remove temporary fake fs
		err := os.RemoveAll(fs.RootDir)
		if err != nil {
			panic(fmt.Errorf("error tearing down fake filesystem: %s", err.Error()))
		}
	}
}

// SetDefaultMockNetlinkProvider sets a mocked instance of NetlinkProvider to be used by unit test in other packages
func SetDefaultMockNetlinkProvider() {
	mockProvider := mocks.NetlinkProvider{}

	mockProvider.
		On("GetLinkAttrs", mock.AnythingOfType("string")).
		Return(&nl.LinkAttrs{EncapType: "fakeLinkType"}, nil)
	mockProvider.
		On("GetDevLinkDeviceEswitchAttrs", mock.AnythingOfType("string")).
		Return(&nl.DevlinkDevEswitchAttr{Mode: "fakeMode"}, nil)

	SetNetlinkProviderInst(&mockProvider)
}
