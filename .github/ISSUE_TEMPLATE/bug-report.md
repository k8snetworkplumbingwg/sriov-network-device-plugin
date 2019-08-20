---
name: Bug Report
about: Report a bug with SR-IOV Device Plugin

---
<!-- Please use this template while reporting a bug and provide as much relevant info as possible. Doing so give us the best chance to find a prompt resolution to your issue -->


**What happened?**:

**What did you expect to happen?**:

**What are the minimal steps needed to reproduce the bug?**:

**Anything else we need to know?**:

**Environment**:

- SR-IOV Device Plugin Version
- Device pool config file location (default "/etc/pcidp/config.json")
- SR-IOV CNI Version
- Multus version (If applicable)
- Multus config ('/etc/cni/multus/net.d)'
- CNI config ('/etc/cni/net.d/')
- Kubernetes version (use `kubectl version`):
- Kubernetes deployment type ( Bare Metal, Kubeadm etc.)
- OS (e.g. from /etc/os-release):
- Kubeconfig file
- SR-IOV Network Custom Resource Definition

**Useful logs**
- SR-IOV Device Plugin Logs (use `kubectl logs <Name of relevant device plugin pod>)
- Multus logs (default "/var/log/multus.log" )
- Kubelet logs (journalctl -u kubelet)
