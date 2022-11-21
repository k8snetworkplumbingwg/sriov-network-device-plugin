package utils

import (
	"fmt"

	"github.com/golang/glog"
	vdpa "github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
)

// VdpaProvider is a wrapper type over go-vdpa library
type VdpaProvider interface {
	GetVdpaDeviceByPci(pciAddr string) (vdpa.VdpaDevice, error)
}

type defaultVdpaProvider struct {
}

var vdpaProvider VdpaProvider = &defaultVdpaProvider{}

// SetVdpaProviderInst method would be used by unit tests in other packages
func SetVdpaProviderInst(inst VdpaProvider) {
	vdpaProvider = inst
}

// GetVdpaProvider will be invoked by functions in other packages that would need access to the vdpa library methods.
func GetVdpaProvider() VdpaProvider {
	return vdpaProvider
}

func (defaultVdpaProvider) GetVdpaDeviceByPci(pciAddr string) (vdpa.VdpaDevice, error) {
	// the govdpa library requires the pci address to include the "pci/" prefix
	fullPciAddr := "pci/" + pciAddr
	vdpaDevices, err := vdpa.GetVdpaDevicesByPciAddress(fullPciAddr)
	if err != nil {
		return nil, err
	}
	numVdpaDevices := len(vdpaDevices)
	if numVdpaDevices == 0 {
		return nil, fmt.Errorf("no vdpa device associated to pciAddress %s", pciAddr)
	}
	if numVdpaDevices > 1 {
		glog.Infof("More than one vDPA device found for pciAddress %s, returning the first one", pciAddr)
	}
	return vdpaDevices[0], nil
}
