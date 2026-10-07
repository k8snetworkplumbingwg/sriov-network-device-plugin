# SR-IOV Network Device Plugin Health Checks

## Overview

The SR-IOV Network Device Plugin includes built-in health check capabilities that allow Kubernetes to monitor the daemon's liveness and readiness. These checks help ensure:

- **Liveness**: The daemon process is running and responsive
- **Readiness**: At least one SR-IOV device is detected, configured, and available for pod allocation

Health checks communicate via a simple JSON-based protocol over Unix domain sockets in `/tmp`.

## Quick Start

### Help

For command-line help:

```bash
./sriovdp -h              # Shows daemon flags
./sriovdp health -h       # Shows health check help
```

### Basic Usage

Start the daemon with health check support (enabled by default):

```bash
./sriovdp -config-file=/etc/pcidp/config.json
```

### Checking Daemon Health

Use the integrated health check command:

```bash
# Check if daemon is alive
./sriovdp health liveness
# Output: OK: daemon is alive
# Exit code: 0

# Check if any devices are ready
./sriovdp health readiness
# Output: OK: ready (5/8 devices healthy)
# Exit code: 0

# Check readiness when no devices available
./sriovdp health readiness
# Output: NOT READY: no healthy devices (0/5)
# Exit code: 1

# Check with custom socket directory
./sriovdp health -healthcheck-socket-dir=/var/run readiness
# Uses /var/run/sriovdp.sock instead of /tmp/sriovdp.sock

# Note: flags must come before the check type
./sriovdp health -healthcheck-socket-dir=/var/run liveness
```

Exit codes:
- `0`: Check passed
- `1`: Check failed (readiness: no healthy devices)
- `2`: Error (socket error, invalid response, bad arguments)

## Kubernetes Integration

### DaemonSet Probes Configuration

Add health checks to your SR-IOV Device Plugin DaemonSet:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: sriov-device-plugin
  namespace: kube-system
spec:
  template:
    spec:
       containers:
       - name: sriovdp
         image: sriov-network-device-plugin:latest
         args:
         - -config-file=/etc/pcidp/config.json
         - -healthcheck-socket-dir=/tmp

          # Verify daemon is running
          livenessProbe:
            exec:
              command:
              - /usr/bin/sriovdp
              - health
              - -healthcheck-socket-dir=/tmp
              - liveness
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 2
            failureThreshold: 3

          # Verify devices are detected and ready
          readinessProbe:
            exec:
              command:
              - /usr/bin/sriovdp
              - health
              - -healthcheck-socket-dir=/tmp
              - readiness
            initialDelaySeconds: 30
            periodSeconds: 300
            timeoutSeconds: 2
            failureThreshold: 1

          # Wait for daemon startup
          startupProbe:
            exec:
              command:
              - /usr/bin/sriovdp
              - health
              - -healthcheck-socket-dir=/tmp
              - liveness
            initialDelaySeconds: 0
            periodSeconds: 5
            timeoutSeconds: 2
            failureThreshold: 30  # 150 seconds max startup time
```

### Probe Recommendations

**Liveness Probe**:
- **Purpose**: Restart pod if daemon becomes unresponsive
- **Recommended Settings**:
  - `initialDelaySeconds: 10` (give daemon time to start)
  - `periodSeconds: 10` (check frequently)
  - `failureThreshold: 3` (3 failures = restart)

**Readiness Probe**:
- **Purpose**: Remove pod from service if devices unavailable
- **Recommended Settings**:
  - `initialDelaySeconds: 30` (allow device detection)
  - `periodSeconds: 30` (check periodically)
  - `failureThreshold: 1` (immediate removal if no devices)

**Startup Probe**:
- **Purpose**: Wait for initial device detection on pod startup
- **Recommended Settings**:
  - `periodSeconds: 5` (fast polling during startup)
  - `failureThreshold: 30` (allow up to 150 seconds for boot)

## Socket Protocol

### Socket Location

Default: `/tmp/sriovdp.sock`

Override with:
```bash
./sriovdp -healthcheck-socket-dir=/var/run
# Creates: /var/run/sriovdp.sock
```

Or environment variable:
```bash
export SRIOV_HEALTHCHECK_SOCKET_DIR=/var/run
./sriovdp -config-file=/etc/pcidp/config.json
```

### Request/Response Format

Communications use line-delimited JSON over Unix domain sockets.

#### Liveness Check

**Request**:
```json
{"check":"liveness"}
```

**Response**:
```json
{"status":"alive"}
```

**Example** (using netcat):
```bash
$ echo '{"check":"liveness"}' | nc -U /tmp/sriovdp.sock
{"status":"alive"}
```

#### Readiness Check

**Request**:
```json
{"check":"readiness"}
```

**Response**:
```json
{"ready":true,"healthy_count":5,"total_count":8,"healthy_servers":2}
```

or

```json
{"ready":false,"healthy_count":0,"total_count":5,"healthy_servers":0}
```

**Fields**:
- `ready`: (boolean) True if any healthy device detected
- `healthy_count`: (integer) Number of healthy/available devices across all servers
- `total_count`: (integer) Total number of devices across all servers
- `healthy_servers`: (integer) Number of resource servers with >= 1 healthy device

**Example**:
```bash
$ echo '{"check":"readiness"}' | nc -U /tmp/sriovdp.sock
{"ready":true,"healthy_count":5,"total_count":8,"healthy_servers":2}
```

## Device Health Determination

The daemon performs periodic probes (every 20 seconds by default) to detect device health status.

### Health Check Criteria

A device is considered **healthy** if:

1. **Device exists**: The PCI device directory is accessible at `/sys/bus/pci/devices/{pciAddr}`
2. **Actively monitored**: Checks run automatically based on resource server configuration
3. **Not removed**: Device hasn't been hot-unplugged after startup

### Readiness Status

The plugin is considered **ready** if:
- **At least one** SR-IOV device is detected AND
- **At least one** device is marked as healthy (found in sysfs)

### Health Check Details

Health status is updated automatically every 20 seconds as the daemon runs device probes. The checks verify:

- Device PCI address exists in `/sys/bus/pci/devices/`
- Device is accessible by the system
- No hardware errors detected (indicated by missing sysfs entries)

**Note**: Link status is NOT checked. A physical interface with "no carrier" is still available for container allocation.

## Troubleshooting

### Socket Not Found

**Error**: `Error: failed to connect to socket /tmp/sriovdp.sock: no such file or directory`

**Causes**:
1. Daemon not running
2. Socket directory doesn't exist or isn't writable
3. Wrong socket directory

**Solution**:
```bash
# Verify daemon is running
ps aux | grep sriovdp

# Check socket exists
ls -la /tmp/sriovdp.sock

# Override socket directory if needed
./sriovdp -healthcheck-socket-dir=/var/run health readiness
```

### Not Ready Status

**Output**: `NOT READY: no healthy devices (0/5)`

**Causes**:
1. SR-IOV VFs not created yet
2. Device drivers not loaded
3. Devices configured but not accessible
4. Hardware issues

**Solution**:
```bash
# Check device configuration
cat /etc/pcidp/config.json

# Verify SR-IOV VFs created
ls /sys/class/net/*/device/sriov_numvfs

# Check device accessibility
ls /sys/bus/pci/devices/ | grep -E "^[0-9a-f]"

# Wait longer for device detection (startup probe may help)
```

### Daemon Unresponsive

**Error**: `Error: failed to send request: broken pipe` or timeout

**Causes**:
1. Daemon crashed
2. Daemon stuck in startup sequence
3. Resource contention

**Solution**:
```bash
# Check daemon status and logs
systemctl status sriov-device-plugin
journalctl -u sriov-device-plugin -n 50

# Manually restart
systemctl restart sriov-device-plugin

# Increase liveness probe failure threshold if transient
```

## Advanced Configuration

### Custom Socket Directory

For restricted `/tmp` or Kubernetes security policies:

```bash
./sriovdp \
  -config-file=/etc/pcidp/config.json \
  -healthcheck-socket-dir=/var/run/sriov
```

### Disabling Health Checks

Health checks run automatically and cannot be disabled, but Kubernetes probes can be omitted from the DaemonSet if not desired.

### Integration with Container Runtimes

Health checks work with any standard Kubernetes container runtime (Docker, containerd, CRI-O, etc.) since they use standard exec probes.

## Performance Impact

- **Minimal**: Health checks run in background with no impact on device allocation
- **Probe Interval**: 20 seconds (configurable per resource server)
- **Socket Communication**: Sub-millisecond latency for JSON over Unix sockets
- **Memory**: Negligible (<1MB for socket listener)

## Monitoring and Observability

### Kubernetes Events

Monitor probe failures:
```bash
kubectl describe pod <sriov-pod> -n kube-system
kubectl get events -n kube-system --field-selector involvedObject.name=<sriov-pod>
```

### Daemon Logs

Verify health check operation:
```bash
# Enable detailed logging
./sriovdp -v=4 -config-file=/etc/pcidp/config.json

# Check for device probe messages
journalctl -u sriov-device-plugin | grep -i "device\|health\|probe"
```

### Manual Testing

Test full probe cycle:
```bash
# Terminal 1: Start daemon with verbose logging
./sriovdp -v=4 -config-file=/etc/pcidp/config.json

# Terminal 2: Run health checks in a loop
while true; do
  ./sriovdp health readiness
  sleep 5
done
```

## References

- Kubernetes Probe Documentation: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
