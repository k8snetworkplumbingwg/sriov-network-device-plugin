package utils

import (
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
	return vdpa.GetVdpaDeviceByPci(pciAddr)
}
