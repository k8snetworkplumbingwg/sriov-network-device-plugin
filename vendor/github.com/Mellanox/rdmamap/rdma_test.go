package rdmamap

import (
	"fmt"
	"testing"
)

func TestGetRdmaDevices(t *testing.T) {
	rdmaDevices := GetRdmaDeviceList()
	t.Log("Devices: ", rdmaDevices)
}

func TestRdmaCharDevices(t *testing.T) {
	rdmaDevices := GetRdmaDeviceList()
	t.Log("Devices: ", rdmaDevices)

	for _, dev := range rdmaDevices {
		charDevices := GetRdmaCharDevices(dev)
		fmt.Printf("Rdma device: = %s", dev)
		t.Log(" Char devices: = ", charDevices)
	}
}

func TestRdmaDeviceForNetdevice(t *testing.T) {

	netdev := "ib0"
	rdmaDev, err := GetRdmaDeviceForNetdevice(netdev)
	if err == nil {
		fmt.Printf("netdev = %s, rdmadev = %s\n", netdev, rdmaDev)
	} else {
		fmt.Printf("rdma device not found for netdev = %s\n", netdev)
	}

	found := IsRDmaDeviceForNetdevice(netdev)
	fmt.Printf("rdma device %t for netdev = %s\n", found, netdev)

	netdev = "ens1f0"
	found = IsRDmaDeviceForNetdevice(netdev)
	fmt.Printf("rdma device %t for netdev = %s\n", found, netdev)

	netdev = "lo"
	found = IsRDmaDeviceForNetdevice(netdev)
	fmt.Printf("rdma device %t for netdev = %s\n", found, netdev)
}

func TestRdmaDeviceStats(t *testing.T) {

	stats, err := GetRdmaSysfsAllPortsStats("mlx5_1")
	if err == nil {
		t.Log(stats)
	} else {
		t.Log("error is: ", err)
	}
}

func TestRdmaDeviceForPcidev(t *testing.T) {
	devs := GetRdmaDevicesForPcidev("0000:05:00.0")
	t.Log("rdma devs :", devs)
}
