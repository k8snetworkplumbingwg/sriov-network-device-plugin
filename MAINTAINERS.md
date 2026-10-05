# Maintainers

## Vendor-specific code

Vendor-specific features are maintained by the vendor contributing them. The
contacts listed below are responsible for reviewing changes, triaging issues and
keeping the feature working with future releases of the device plugin.

### HPE Slingshot (Cassini / CXI)

| Name            | GitHub                                   | Company |
|-----------------|------------------------------------------|---------|
| Cody D'Ambrosio | [@codambro](https://github.com/codambro) | HPE     |

Scope:

- `pkg/infoprovider/cxiInfoProvider.go`
- `pkg/utils/utils.go` (`HasCxiDevice`, `GetCxiDeviceFile`)
- `isCxi` selector handling in `pkg/netdevice/`
