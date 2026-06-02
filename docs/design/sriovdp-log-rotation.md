---
title: Persistent Log Storage for SR-IOV Device Plugin
authors:
  - gavrielg1
reviewers:
  - SchSeba
creation-date: 01-06-2026
last-updated: 01-06-2026
---

# Persistent Log Storage for SR-IOV Device Plugin

## Summary

Add persistent, host-based log storage with rotation for the sriov-network-device-plugin so that logs survive pod deletions triggered by the sriov-network-operator. Today, when the operator deletes the device plugin pod after a configuration change, all container logs are lost. This makes it difficult to debug device discovery, resource allocation, and Kubelet registration issues.

## Motivation

The sriov-network-operator manages the lifecycle of the sriov-network-device-plugin DaemonSet pods. On every SR-IOV configuration change (applied via SriovNetworkNodeState), the operator's config daemon calls restartDevicePluginPod() which deletes the device plugin pod on the affected node. The DaemonSet controller then recreates it. This deletion destroys all container logs.

The device plugin uses glog for logging. While glog supports file-based logging via --log_dir, the standalone deployment manifests already use this feature (writing to /var/log/sriovdp), but the operator-managed deployment does not — it logs only to stderr with no --log-dir flag and no /var/log volume mount.

Even when file logging is enabled (standalone mode), glog does not implement log rotation. Log files can grow indefinitely, eventually consuming significant host disk space. Furthermore, glog creates multiple files per severity level (INFO, WARNING, ERROR, FATAL) with PID-stamped filenames, and each pod restart creates a new set of files — compounding the disk usage problem.

## Use Cases

- **Post-restart debugging:** After the operator deletes and recreates the device plugin pod, the admin needs to inspect pre-restart logs to understand what resources were being served, what devices were discovered, and whether any allocation errors occurred.
- **Device discovery issues:** When VFs are not advertised correctly, historical logs from before the last restart provide crucial context.
- **Kubelet registration problems:** Registration with the Kubelet happens at startup. If the pod is repeatedly restarted, logs from previous registrations are needed to diagnose failures.
- **Long-running stability:** For standalone deployments using --log_dir, unbounded log growth can fill the host's /var/log partition.

## Goals

- Persist device plugin logs to the host filesystem so they survive operator-triggered pod deletions.
- Implement log rotation with configurable maximum file size and number of retained files.
- Ensure the operator-managed deployment includes file-based logging (matching standalone behavior).
- Maintain existing stderr logging behavior (logs still appear in kubectl logs).
- Work with the existing glog logging framework without requiring a full logging migration.

## Non-Goals

- Migrating from glog to a different logging framework (e.g., klog, zap).
- Implementing log persistence for other components (CNI plugins, etc.).
- Implementing centralized log aggregation.

## Proposal

### Overview

Implement log rotation using the lumberjack library (gopkg.in/natefinch/lumberjack.v2) as a wrapper around glog's file output, and update the operator-managed deployment manifest to include the --log-dir flag and a host volume mount for logs.

Two complementary changes are proposed:

1. **Operator-side:** Update the device plugin DaemonSet manifest (bindata/manifests/plugins/sriov-device-plugin.yaml) to add --log-dir=sriovdp and a /var/log host volume mount — matching the standalone deployment.

2. **Device plugin-side:** Add a log rotation wrapper that manages glog's output files, preventing unbounded growth.

### Workflow Description

#### Approach: Log Rotation Sidecar / Wrapper

`glog` v1.2.5 writes files directly and does **not** support pluggable writers. Specifically, `glog` v1.2.5 does **not** export `SetOutput()`, `SetLogger()`, or any other output-redirection API. It is configured exclusively via `flag` registration (`-log_dir`, `-logtostderr`, `-alsologtostderr`, `-v`, etc.). This constraint limits the available approaches:

**Option A: External log rotation via logrotate**

Add a logrotate configuration that rotates glog's output files via a sidecar container in the DaemonSet. This approach is non-invasive (no Go code changes) but introduces operational complexity:

- Requires an additional sidecar container image to build, publish, and maintain across all architectures (amd64, arm64, ppc64le, s390x).
- `copytruncate` has a race window — log lines written between the copy and truncate are lost.
- glog's per-severity, per-PID file explosion is not solved — stale files from previous restarts still accumulate and require separate cleanup configuration.
- Polling-based rotation (every N seconds) cannot react instantly to burst log output exceeding the size limit.

**Option B: Stderr capture with lumberjack (recommended)**

Intercept the process's stderr output at the OS level:

1. Set `--logtostderr=true` (disable glog's native file output).
2. In `main.go`, use `os.Pipe()` to capture stderr and tee it to both the original stderr and a lumberjack-managed file.

This approach bypasses glog's file writer entirely, writing a single consolidated log file (`sriovdp.log`) instead of glog's per-severity, per-PID files.

**Why Option B is recommended:**

- **Single container** — no sidecar image to maintain, simpler pod spec, no extra scheduling overhead across large clusters.
- **Immediate, atomic rotation** — lumberjack rotates the instant the file hits max size; no polling delay.
- **No data-loss race** — unlike `copytruncate`, lumberjack owns the file exclusively and rotation is atomic.
- **Eliminates stale file problem** — lumberjack manages its own retention (MaxBackups + MaxAge); no separate cleanup config needed.
- **Single consolidated log file** — operationally simpler for admins (`tail -f sriovdp.log` instead of figuring out which PID-stamped file is current).
- **Configurable via CLI flags** — rotation parameters are passed directly to the binary, no external config files to mount.

**Option A remains a viable fallback** for environments where modifying the device plugin binary is not possible (e.g., using an upstream image as-is and only controlling the DaemonSet manifest).

### Implementation Detail

#### Option B: Stderr capture with lumberjack (recommended)

```go
// pkg/logging/logging.go

package logging

import (
    "io"
    "os"
    "path/filepath"
    "syscall"

    "gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
    LogDir     string
    MaxSizeMB  int
    MaxFiles   int
    MaxAgeDays int
    Compress   bool
}

func DefaultConfig() Config {
    return Config{
        LogDir:     "/var/log/sriovdp",
        MaxSizeMB:  100,
        MaxFiles:   5,
        MaxAgeDays: 30,
        Compress:   true,
    }
}

func NewRotatingWriter(cfg Config) io.WriteCloser {
    return &lumberjack.Logger{
        Filename:   filepath.Join(cfg.LogDir, "sriovdp.log"),
        MaxSize:    cfg.MaxSizeMB,
        MaxBackups: cfg.MaxFiles,
        MaxAge:     cfg.MaxAgeDays,
        Compress:   cfg.Compress,
    }
}

// CaptureStderr redirects stderr to both the original stderr and the
// rotating writer. Returns a cleanup function that must be called
// before process exit to flush remaining data.
func CaptureStderr(w io.Writer) (cleanup func(), err error) {
    origStderr, err := syscall.Dup(int(os.Stderr.Fd()))
    if err != nil {
        return nil, err
    }

    r, pw, err := os.Pipe()
    if err != nil {
        syscall.Close(origStderr)
        return nil, err
    }

    // Replace stderr fd with the write end of the pipe
    if err := syscall.Dup2(int(pw.Fd()), int(os.Stderr.Fd())); err != nil {
        r.Close()
        pw.Close()
        syscall.Close(origStderr)
        return nil, err
    }
    pw.Close()

    origFile := os.NewFile(uintptr(origStderr), "original-stderr")
    multi := io.MultiWriter(origFile, w)

    done := make(chan struct{})
    go func() {
        io.Copy(multi, r)
        close(done)
    }()

    return func() {
        // Restore original stderr so no new writes enter the pipe
        syscall.Dup2(origStderr, int(os.Stderr.Fd()))
        // Wait for io.Copy to drain all remaining data and hit EOF
        <-done
        r.Close()
        origFile.Close()
    }, nil
}
```

In cmd/sriovdp/main.go:

```go
func main() {
    cp := &cliParams{}
    flagInit(cp)
    flag.Parse()

    // Flush glog buffers on exit (glog buffers INFO for ~30 seconds)
    defer glog.Flush()

    // Set up rotated file logging.
    // glog's --log_dir flag is registered by the glog package; retrieve
    // the value via flag.Lookup after flag.Parse(). If not set, fall back
    // to the default log directory.
    logDir := flag.Lookup("log_dir").Value.String()
    if logDir == "" {
        logDir = logging.DefaultConfig().LogDir
    }

    rotWriter := logging.NewRotatingWriter(logging.Config{
        LogDir:     logDir,
        MaxSizeMB:  cp.logMaxSize,
        MaxFiles:   cp.logMaxFiles,
        MaxAgeDays: cp.logMaxAge,
        Compress:   cp.logCompress,
    })
    defer rotWriter.Close()

    // glog v1.2.5 does NOT support SetOutput() or SetLogger().
    // Use --logtostderr and capture stderr to tee into lumberjack.
    cleanup, err := logging.CaptureStderr(rotWriter)
    if err != nil {
        glog.Errorf("failed to set up log rotation: %v", err)
    } else {
        defer cleanup()
    }

    rm := newResourceManager(cp)
    // ... rest of main ...
}
```

> **Important:** `defer glog.Flush()` must be added near the top of `main()`. glog buffers INFO-level messages and only flushes periodically (~30 seconds). Without this, the last ~30 seconds of logs are silently lost on every SIGTERM shutdown.

**Disk usage with Option B:**

| Parameter | Value |
|-----------|-------|
| Current active file | 1 x 100 MB |
| Backup files (MaxBackups) | 5 x 100 MB |
| **Maximum disk usage** | **~600 MB per node** (1 active + 5 backups) |

Note: lumberjack retains MaxBackups + 1 (the active file), so 5 backups = 6 total files.

#### Option A: External logrotate (fallback)

For environments where the device plugin binary cannot be modified, an external logrotate sidecar can be used instead. No Go code changes are required:

```
# /etc/logrotate.d/sriovdp (embedded in sidecar image)
/var/log/sriovdp/*.log {
    size 100M
    rotate 5
    compress
    missingok
    notifempty
    copytruncate
    daily
}
```

The `copytruncate` directive copies the current log file and truncates it in place, avoiding the need to signal the process to reopen file handles.

Note that glog creates per-severity files with PID-stamped names (e.g., `sriovdp.<hostname>.<user>.log.INFO.<date>-<time>.<pid>`). Old files from previous pod restarts (different PIDs) accumulate and require a separate `maxage` directive or startup cleanup script to prune stale files.

## Operator-Side Changes

### Update device plugin DaemonSet manifest

In sriov-network-operator/bindata/manifests/plugins/sriov-device-plugin.yaml:

```yaml
spec:
  containers:
    - name: sriov-device-plugin
      args:
        - --log-dir=sriovdp
        - --log-level=10
        - --resource-prefix={{.ResourcePrefix}}
        - --config-file=/etc/pcidp/$(NODE_NAME)
      volumeMounts:
        # ... existing mounts ...
        - name: log
          mountPath: /var/log
  volumes:
    # ... existing volumes ...
    - name: log
      hostPath:
        path: /var/log
        type: DirectoryOrCreate
```

This matches the existing standalone deployment (deployments/sriovdp-daemonset.yaml) which already has these settings.

### Optional: Expose log configuration via SriovOperatorConfig

Similar to the config daemon proposal, add log configuration to the operator config:

```yaml
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovOperatorConfig
metadata:
  name: default
  namespace: sriov-network-operator
spec:
  devicePluginLogConfig:
    enabled: true
    maxSizeMB: 100
    maxFiles: 5
```

The operator would pass these as additional args or environment variables to the device plugin pod.

## API Extensions

### New CLI flags for the device plugin

```
--log-max-size       Maximum size in MB of a log file before rotation (default: 100)
--log-max-files      Maximum number of old log files to retain (default: 5)
--log-max-age        Maximum number of days to retain old log files (default: 30)
--log-compress       Compress rotated log files (default: true)
```

These flags are added to cmd/sriovdp/main.go alongside existing glog flags.

### Entrypoint script changes

Update images/entrypoint.sh to pass rotation flags:

```bash
if [ "$LOG_DIR" != "" ]; then
    mkdir -p "/var/log/$LOG_DIR"
    CLI_PARAMS="$CLI_PARAMS --log_dir /var/log/$LOG_DIR --alsologtostderr"
    CLI_PARAMS="$CLI_PARAMS --log-max-size ${LOG_MAX_SIZE:-100}"
    CLI_PARAMS="$CLI_PARAMS --log-max-files ${LOG_MAX_FILES:-5}"
fi
```

## Implementation Details/Notes/Constraints

### 1. glog v1.2.5 compatibility

`glog` v1.2.5 (the version in `go.mod`) does **not** export `SetOutput()`, `SetLogger()`, or any output-redirection API. The only way to control glog's behavior is through its registered flags (`-log_dir`, `-logtostderr`, `-alsologtostderr`, `-v`, etc.). This is the key constraint that makes stderr capture (Option B) the most viable in-process approach.

With Option B, glog is configured with `--logtostderr=true` so all log output goes to stderr. The `CaptureStderr()` function intercepts stderr at the file-descriptor level and tees it to both the original stderr (preserving `kubectl logs`) and a lumberjack-managed rotating file.

glog writes log files in its own format: `<program>.<hostname>.<user>.log.<severity>.<date>-<time>.<pid>`. Because `hostNetwork: true` is used in all DaemonSet manifests, the hostname in filenames is the node hostname (not the pod name). Each pod restart creates a new set of 4 severity files (INFO, WARNING, ERROR, FATAL) plus 4 symlinks. Option B avoids this file explosion entirely by consolidating all output into a single `sriovdp.log`.

If the logrotate sidecar approach (Option A) is used as a fallback, the following config handles native glog files:

```
# logrotate config
/var/log/sriovdp/*.log {
    size 100M
    rotate 5
    compress
    missingok
    notifempty
    copytruncate
}
```

The `copytruncate` directive copies the current log file and then truncates it, avoiding the need to signal the process to reopen its file handles.

### 2. glog.Flush() and log buffer behavior

glog buffers INFO-level messages and only flushes them periodically (~30 seconds). The current codebase has **zero** calls to `glog.Flush()`. On SIGTERM, the signal handler calls `stopAllServers()` and `cleanupCDISpecs()` then returns — but never flushes glog buffers. This means the last ~30 seconds of INFO-level logs are silently lost on every operator-triggered pod deletion.

Any implementation must add `defer glog.Flush()` near the top of `main()`.

### 3. glog.Fatal exit paths

The codebase has two `glog.Fatalf` call sites in `pkg/resources/server.go`:

1. Deprecated registration failure
2. Server restart failure in `Watch()` loop

`glog.Fatalf` calls `glog.Flush()` internally and then calls `os.Exit(255)`. Since `os.Exit` bypasses all `defer` statements, `defer`-based cleanup (lumberjack writer close, stderr capture restore) will NOT execute on Fatal paths. However, this is acceptable because:

- `glog.Fatalf` flushes its own buffers internally, so the Fatal message itself is always captured.
- The Fatal paths represent unrecoverable errors where the process is dying regardless.
- lumberjack's file state is consistent on disk even without explicit `Close()` — only the final partial write (if any) might be lost.

For additional safety, consider converting the two `glog.Fatal` calls to `glog.Error` + controlled shutdown to ensure all cleanup runs.

### 4. glog multi-file output and stale file accumulation

When using `--log_dir` with glog's native file output, glog creates files per severity level per process invocation. With frequent operator-triggered restarts:

| Restarts | Severity files created | Symlinks |
|----------|----------------------|----------|
| 1 | 4 | 4 |
| 10 | 40 | 4 (updated) |
| 100 | 400 | 4 (updated) |

Old files from previous PIDs are never cleaned up by glog.

**Option B eliminates this problem entirely** — by using `--logtostderr=true` and capturing stderr to a single lumberjack-managed file (`sriovdp.log`), there is only ever one active file plus up to MaxBackups rotated copies. No PID-stamped files are created, and lumberjack handles its own retention automatically.

If Option A (logrotate) is used as a fallback, a startup cleanup routine or the logrotate `maxage` directive should be used to prune stale files.

### 5. Disk space considerations

With Option B (lumberjack) and default configuration:

| Parameter | Value |
|-----------|-------|
| Current active file | 1 x 100 MB |
| Backup files (MaxBackups=5) | 5 x 100 MB |
| **Maximum disk usage per node** | **~600 MB** |

Note: lumberjack retains MaxBackups + 1 (the active file), so 5 backups = 6 total files.

This is predictable and bounded — unlike glog's native file output which grows unboundedly with restarts. The /var/log partition on most nodes has several GB available.

### 6. Permissions

The device plugin container runs as privileged with the /var/log hostPath mount. Log files will be created with standard permissions (0644) under /var/log/sriovdp/.

### 7. Backward compatibility with standalone deployments

Standalone deployments that already use --log-dir will benefit from log rotation automatically once the binary is updated. No manifest changes are needed for standalone users — only the Go binary needs to be updated.

## Upgrade & Downgrade considerations

- **Upgrade (operator-managed):** After upgrade, the device plugin DaemonSet will include the new --log-dir arg and volume mount. Logs will start being persisted. No user action required.
- **Upgrade (standalone):** Users with existing --log-dir will get log rotation for free. Users without --log-dir see no change.
- **Downgrade:** Downgrading the device plugin binary removes rotation. Existing rotated log files remain on the host. New logs will be written by glog in its default format (no rotation). An admin may want to clean up old files manually.
- **Operator manifest rollback:** If the operator is downgraded, the --log-dir and volume mount will be removed from the DaemonSet. Logging reverts to stderr-only for operator-managed deployments. Existing log files on the host are not cleaned up.

## Test Plan

### Unit tests

- Verify that the rotating writer correctly wraps lumberjack with the expected configuration.
- Verify default configuration values.
- Verify that CLI flags for rotation are parsed correctly by both the Go binary and `entrypoint.sh`.
- Verify `CaptureStderr` correctly tees output to both original stderr and the rotating writer.
- Verify that `flag.Lookup("log_dir")` correctly retrieves the glog-registered flag value.

### Integration / E2E tests

- Deploy the device plugin with log persistence enabled (via operator).
- Trigger a configuration change that causes the operator to restart the device plugin pod.
- After restart, verify that log files from the previous pod exist on the host at /var/log/sriovdp/.
- Verify that the new pod also writes to the same log directory.
- Verify log rotation by generating enough log output to exceed the configured max size.
- Verify that the number of retained files does not exceed the configured maximum.
- Verify that `kubectl logs` still shows current output (stderr preserved).

### Manual testing

- Verify logs persist across operator-triggered pod deletions.
- Verify kubectl logs still shows current logs (stderr output is preserved).
- Verify disk usage stays within configured limits after prolonged operation.
- Test standalone deployment with and without --log-dir to ensure backward compatibility.
- Verify that `glog.Flush()` is called on clean shutdown (last log lines are not lost).
- Test `glog.Fatal` paths — verify that Fatal messages are captured even though deferred cleanup is bypassed.
