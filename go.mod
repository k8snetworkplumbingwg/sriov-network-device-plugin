module github.com/k8snetworkplumbingwg/sriov-network-device-plugin

go 1.13

require (
	github.com/Mellanox/rdmamap v1.0.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/jaypipes/ghw v0.6.0
	github.com/jaypipes/pcidb v0.5.0
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.1-0.20201119153432-9d213757d22d
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	github.com/vishvananda/netlink v1.1.0
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/grpc v1.28.1
	k8s.io/kubelet v0.18.1
)
