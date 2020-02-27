package resources

import (
	"github.com/golang/glog"
	"github.com/intel/sriov-network-device-plugin/pkg/types"
	"github.com/intel/sriov-network-device-plugin/pkg/utils"
	"github.com/jaypipes/ghw"
)

// pciNetDevice extends pciDevice
type pciNetDevice struct {
	types.PciDevice
	ifName    string
	pfName    string
	linkSpeed string
	rdmaSpec  types.RdmaSpec
	linkType  string
}

// NewPciNetDevice returns an instance of PciNetDevice interface
func NewPciNetDevice(dev *ghw.PCIDevice, rFactory types.ResourceFactory) (types.PciNetDevice, error) {

	var ifName string
	pciDev, err := NewPciDevice(dev, rFactory)
	if err != nil {
		return nil, err
	}

	pciAddr := pciDev.GetPciAddr()
	netDevs, _ := utils.GetNetNames(pciAddr)
	if len(netDevs) == 0 {
		ifName = ""
	} else {
		ifName = netDevs[0]
	}
	pfName, err := utils.GetPfName(pciAddr)
	if err != nil {
		glog.Warningf("unable to get PF name %q", err.Error())
	}

	rdmaSpec := rFactory.GetRdmaSpec(dev.Address)

	linkType := ""
	if len(ifName) > 0 {
		la, err := utils.GetLinkAttrs(ifName)
		if err != nil {
			return nil, err
		}
		linkType = la.EncapType
	}

	return &pciNetDevice{
		PciDevice: pciDev,
		ifName:    ifName,
		pfName:    pfName,
		linkSpeed: "", // TO-DO: Get this using utils pkg
		rdmaSpec:  rdmaSpec,
		linkType:  linkType,
	}, nil
}

func (nd *pciNetDevice) GetPFName() string {
	return nd.pfName
}

func (nd *pciNetDevice) GetNetName() string {
	return nd.ifName
}

func (nd *pciNetDevice) GetLinkSpeed() string {
	return nd.linkSpeed
}

func (nd *pciNetDevice) GetRdmaSpec() types.RdmaSpec {
	return nd.rdmaSpec
}

func (nd *pciNetDevice) GetLinkType() string {
	return nd.linkType
}

func (nd *pciNetDevice) GetDDPProfiles() string {
	pciAddr := nd.GetPciAddr()
	ddpProfile, err := utils.GetDDPProfiles(pciAddr)
	if err != nil {
		glog.Infof("GetDDPProfiles(): unable to get ddp profiles for device %s : %q", pciAddr, err)
		return ""
	}
	return ddpProfile
}
