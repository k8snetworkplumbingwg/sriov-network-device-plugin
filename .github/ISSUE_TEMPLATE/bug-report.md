---
name: Bug Report
about: Report a bug with SR-IOV Device Plugin

---
<!-- Please use this template while reporting a bug and provide as much relevant info as possible. Doing so give us the best chance to find a prompt resolution to your issue -->

**What happened?**

**What did you expect to happen?**

**What are the minimal steps needed to reproduce the bug?**

**Anything else we need to know?**

**Environment**

Please fill in the below table with the version numbers of components used.

Component | Version|
----------------------------|------------|
|SR-IOV Device Plugin |       <input type="text" id="sriovDPVersion"/>    |
|SR-IOV CNI Plugin |  <input type="text" id="sriovCNIVersion"/> 
|Multus |<input type="text" id="multusVersion"/>
| Kubernetes|<input type="text" id="k8sVersion"/>| 
| OS|<input type="text" id="OSVersion"/>|

Please paste config files below:
-Device pool config file location (default "/etc/pcidp/config.json")
- Multus config ('/etc/cni/multus/net.d)'

- CNI config ('/etc/cni/net.d/')
- Kubernetes deployment type ( Bare Metal, Kubeadm etc.)
- Kubeconfig file
- SR-IOV Network Custom Resource Definition

**Useful logs**
- SR-IOV Device Plugin Logs (use `kubectl logs $PODNAME')
- Multus logs (default "/var/log/multus.log" )
- Kubelet logs (journalctl -u kubelet)
