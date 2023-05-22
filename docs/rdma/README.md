# Running RDMA applications in Kubernetes

RDMA supports zero-copy networking by enabling the network adapter to transfer data from the wire directly to application memory or from application memory directly to the wire, eliminating the need to copy data between application memory and the data buffers in the operating system. Such transfers require no work to be done by CPUs, caches, or context switches, and transfers continue in parallel with other system operations. This reduces latency in message transfer.

## Supported NICs:
* Mellanox ConnectX®-4 Lx Adapter
* Mellanox ConnectX®-5 Adapter
* Intel E810-C Adapter

## RDMA Capable Hardware:
* Mellanox ConnectX®-4 Lx Adapter
* Mellanox ConnectX®-5 Adapter
* Intel E810-C Adapter

## RDMA modules:
* Mellanox ConnectX®-4 Lx, ConnectX®-5 Adapters mlx5_core or mlx5_ib
* Intel E810-C Adapter ice and iavf

## Privileges
IPC_LOCK capability privilege is required for RMA application to function properly in Kubernetes Pod.

## Rdma Mounts:
Using Rdma requires mounting special files from `/dev/infiniband` in the container:
```
# ls /dev/infiniband
issm2  rdma_cm  ucm2  umad1  uverbs2
```
__Note__: rdma character devices mounted under `/dev/infiniband` may vary depending on the vendor and loaded kernel modules.
