# Maintainers

## Project maintainers

The project maintainers own the overall codebase, review and merge pull
requests, and cut releases.

| Name                  | GitHub                                           |
|-----------------------|--------------------------------------------------|
| Adrian Chiris         | [@adrianchiris](https://github.com/adrianchiris) |
| Sebastian Scheinkman  | [@SchSeba](https://github.com/SchSeba)           |
| Andrea Panattoni      | [@zeeke](https://github.com/zeeke)               |
| Ivan Kolodyazhny      | [@e0ne](https://github.com/e0ne)                 |

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
