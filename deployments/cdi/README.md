# SR-IOV Network Device Plugin in CDI mode deployment

SR-IOV Network Device Plugin supports [Container Device Interface (CDI)](https://github.com/container-orchestrated-devices/container-device-interface).

To enable CDI mode, SR-IOV Network Device Plugin should be started with `--use-cdi` CLI argument.
This mode has different deployment requirements: `sriovdp-daemonset.yaml`

```yaml
    - mountPath: /var/run/cdi
      name: dynamic-cdi
```
