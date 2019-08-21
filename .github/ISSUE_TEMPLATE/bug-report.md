---
name: Bug Report
about: Report a bug with SR-IOV Network Device Plugin

---
<!-- Please use this template while reporting a bug and provide as much relevant info as possible. Doing so give us the best chance to find a prompt resolution to your issue -->

### What happened?

### What did you expect to happen?

### What are the minimal steps needed to reproduce the bug?

### Anything else we need to know?

### Component Versions
Please fill in the below table with the version numbers of components used.

Component | Version|
------------------------------|--------------------|
|SR-IOV Network Device Plugin |<Input Version Here>|
|SR-IOV CNI Plugin            |<Input Version Here>|
|Multus                       |<Input Version Here>|
|Kubernetes                   |<Input Version Here>| 
|OS                           |<Input Version Here>|

### Config Files
Config file locations may be config dependent.

##### Device pool config file location (Try '/etc/pcidp/config.json')

##### Multus config (Try '/etc/cni/multus/net.d')

##### CNI config (Try '/etc/cni/net.d/')

##### Kubernetes deployment type ( Bare Metal, Kubeadm etc.)

##### Kubeconfig file

##### SR-IOV Network Custom Resource Definition

### Logs
##### SR-IOV Network Device Plugin Logs (use `kubectl logs $PODNAME`)

##### Multus logs (If enabled. Try '/var/log/multus.log' )

##### Kubelet logs (journalctl -u kubelet)
