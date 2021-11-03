package utils

import (
	"github.com/Mellanox/sriovnet"
)

// SriovnetProvider is a wrapper type over sriovnet library
type SriovnetProvider interface {
	GetUplinkRepresentor(vfPciAddress string) (string, error)
}

type defaultSriovnetProvider struct {
}

var sriovnetProvider SriovnetProvider = &defaultSriovnetProvider{}

// SetSriovnetProviderInst method would be used by unit tests in other packages
func SetSriovnetProviderInst(inst SriovnetProvider) {
	sriovnetProvider = inst
}

// GetSriovnetProvider will be invoked by functions in other packages that would need access to the sriovnet library methods.
func GetSriovnetProvider() SriovnetProvider {
	return sriovnetProvider
}

func (defaultSriovnetProvider) GetUplinkRepresentor(vfPciAddress string) (string, error) {
	return sriovnet.GetUplinkRepresentor(vfPciAddress)
}
