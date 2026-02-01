package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang/glog"
	"golang.org/x/sys/unix"
)

type RblnSmi struct {
	KMDVersion string   `json:"KMD_version"`
	Devices    []Device `json:"devices"`
}

type Device struct {
	GroupID string  `json:"group_id"`
	Npu     int     `json:"npu"`
	Name    string  `json:"name"`
	SID     string  `json:"sid"`
	Device  string  `json:"device"`
	PCI     PCIInfo `json:"pci"`
}

type PCIInfo struct {
	Dev      string `json:"dev"`
	BusID    string `json:"bus_id"`
	NUMANode string `json:"numa_node"`
}

const (
	rblnSmiCommand     = "rbln-smi"
	rblnSmiGroupOption = "group"
	defaultExecTimeout = 5 * time.Second
	defaultGroupID     = "0"
	rblnSmiLockFile    = "/var/run/rbln-device-plugin.lock"
	rsdDevice          = "/dev/rsd"
	defaultRsdDevice   = rsdDevice + "0"
	lockFilePerm       = 0o644
	lockPollInterval   = 100 * time.Millisecond
)

func getRblnSmiAbsolutePath() (string, error) {
	// Prefer explicit driver/installed locations before falling back to PATH lookup.
	candidates := []string{
		"/usr/bin/rbln-smi",
		"/run/rbln/driver/usr/bin/rbln-smi",
		"/host/usr/bin/rbln-smi",
		"/host/driver/usr/bin/rbln-smi",
	}

	for _, p := range candidates {
		info, err := os.Stat(p)
		if err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return p, nil
		}
	}

	abs, err := exec.LookPath(rblnSmiCommand)
	if err != nil {
		return "", fmt.Errorf("cannot find %s command in predefined paths or PATH: %w", rblnSmiCommand, err)
	}
	return abs, nil
}

func runRblnSmiCommand(args ...string) ([]byte, error) {
	abs, err := getRblnSmiAbsolutePath()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultExecTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, abs, args...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			status := ee.Sys().(syscall.WaitStatus)
			return nil, fmt.Errorf("command failed with exit code %d: %s", status.ExitStatus(), string(out))
		}
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	return out, nil
}

func getDeviceInfoFromRblnSmi() (*RblnSmi, error) {
	out, err := runRblnSmiCommand("-g", "-j")
	if err != nil {
		return nil, err
	}
	var result RblnSmi
	if err = json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rbln-smi output: %w", err)
	}
	return &result, nil
}

func collectRsdGroupIDs(smiInfo *RblnSmi, devices []string) []string {
	var filterdDevices []Device

	if len(devices) == 0 {
		filterdDevices = smiInfo.Devices
	} else {
		deviceSet := make(map[string]struct{}, len(devices))
		for _, device := range devices {
			deviceSet[device] = struct{}{}
		}

		for _, v := range smiInfo.Devices {
			if _, ok := deviceSet[v.PCI.BusID]; ok {
				filterdDevices = append(filterdDevices, v)
			}
		}
	}

	seen := make(map[string]struct{})
	groupIDs := make([]string, 0, len(filterdDevices))
	for _, fd := range filterdDevices {
		if fd.GroupID == defaultGroupID {
			continue
		}
		if _, exists := seen[fd.GroupID]; !exists {
			seen[fd.GroupID] = struct{}{}
			groupIDs = append(groupIDs, fd.GroupID)
		}
	}
	return groupIDs
}

func getRsdGroupIDs(smiInfo *RblnSmi, devices []string) []string {
	return collectRsdGroupIDs(smiInfo, devices)
}

func getAllRsdGroupIDs(smiInfo *RblnSmi) []string {
	return collectRsdGroupIDs(smiInfo, nil)
}

func getNpuIDs(smiInfo *RblnSmi, devices []string) []int {
	var npuIDs []int

	for _, device := range devices {
		for _, v := range smiInfo.Devices {
			if v.PCI.BusID == device {
				npuIDs = append(npuIDs, v.Npu)
			}
		}
	}
	return npuIDs
}

func nextRsdGroupID(smiInfo *RblnSmi) (string, error) {
	groupIDs := getAllRsdGroupIDs(smiInfo)

	seen := make(map[int]struct{}, len(groupIDs))

	for _, s := range groupIDs {
		groupID, err := strconv.Atoi(s)
		if err != nil {
			return "", fmt.Errorf("invalid group id: %d", groupID)
		}
		seen[groupID] = struct{}{}
	}

	for i := 1; ; i++ {
		if _, ok := seen[i]; !ok {
			return strconv.Itoa(i), nil
		}
	}
}

func acquireFileLock(ctx context.Context, lockPath string) (release func() error, err error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, lockFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	unlock := func() error {
		_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
		return f.Close()
	}

	ticker := time.NewTicker(lockPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = f.Close()
			return nil, fmt.Errorf("acquire lock timeout/canceled: %w", ctx.Err())
		default:
			if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
				<-ticker.C
				continue
			}
			return unlock, nil
		}
	}
}

func destroyRsdGroup(deviceIDs []string) error {
	smiInfo, err := getDeviceInfoFromRblnSmi()
	if err != nil {
		return err
	}

	groupIDs := getRsdGroupIDs(smiInfo, deviceIDs)
	if err != nil {
		return err
	}
	if len(groupIDs) > 0 {
		_, err = runRblnSmiCommand(rblnSmiGroupOption, "-d", strings.Join(groupIDs, ","))
		if err != nil {
			return err
		}
		glog.Infof("Destroyed RSD groups: %s", strings.Join(groupIDs, ","))
	}
	return nil
}

func createRsdGroup(devices []string) (groupID string, err error) {
	smiInfo, err := getDeviceInfoFromRblnSmi()
	if err != nil {
		return "", err
	}

	groupID, err = nextRsdGroupID(smiInfo)
	if err != nil {
		return "", err
	}

	npuIDs := getNpuIDs(smiInfo, devices)

	strIDs := make([]string, len(npuIDs))
	for i, id := range npuIDs {
		strIDs[i] = strconv.Itoa(id)
	}

	_, err = runRblnSmiCommand(
		rblnSmiGroupOption,
		"-c", groupID,
		"-a", strings.Join(strIDs, ","),
	)

	if err != nil {
		return "", err
	}

	glog.Infof("Created RSD group %s for devices: %s", groupID, strings.Join(devices, ","))
	return groupID, nil
}

func withRsdLock(ctx context.Context, fn func() (string, error)) (string, error) {
	release, err := acquireFileLock(ctx, rblnSmiLockFile)
	if err != nil {
		return "", err
	}
	defer func() {
		if relErr := release(); err == nil && relErr != nil {
			err = relErr
		}
	}()
	return fn()
}

func RecreateRsdGroup(deviceIDs []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), defaultExecTimeout)
	defer cancel()

	groupID, err := withRsdLock(ctx, func() (string, error) {
		if err := destroyRsdGroup(deviceIDs); err != nil {
			glog.Errorf("Failed to destroy RSD groups: %q", err)
			return "", err
		}
		return createRsdGroup(deviceIDs)
	})

	if err != nil {
		glog.Errorf("Failed to create RSD groups: %q", err)
		return defaultRsdDevice
	}
	return rsdDevice + groupID
}
