package utils

import (
	"github.com/k8snetworkplumbingwg/sriovnet"
)

// SriovnetProvider is a wrapper type over sriovnet library
type SriovnetProvider interface {
	GetUplinkRepresentor(vfPciAddress string) (string, error)
	GetUplinkRepresentorFromAux(auxDev string) (string, error)
	GetPfPciFromAux(auxDev string) (string, error)
	GetSfIndexByAuxDev(auxDev string) (int, error)
	GetNetDevicesFromAux(auxDev string) ([]string, error)
	GetAuxNetDevicesFromPci(pciAddr string) ([]string, error)
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

func (defaultSriovnetProvider) GetUplinkRepresentorFromAux(auxDev string) (string, error) {
	return sriovnet.GetUplinkRepresentorFromAux(auxDev)
}

func (defaultSriovnetProvider) GetPfPciFromAux(auxDev string) (string, error) {
	return sriovnet.GetPfPciFromAux(auxDev)
}

func (defaultSriovnetProvider) GetSfIndexByAuxDev(auxDev string) (int, error) {
	return sriovnet.GetSfIndexByAuxDev(auxDev)
}

func (defaultSriovnetProvider) GetNetDevicesFromAux(auxDev string) ([]string, error) {
	return sriovnet.GetNetDevicesFromAux(auxDev)
}

func (defaultSriovnetProvider) GetAuxNetDevicesFromPci(pciAddr string) ([]string, error) {
	return sriovnet.GetAuxNetDevicesFromPci(pciAddr)
}
