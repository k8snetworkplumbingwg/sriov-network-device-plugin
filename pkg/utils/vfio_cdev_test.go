// Copyright 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// A real harmless character device supplies node metadata without privileged mknod.
func cdevFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	pci := filepath.Join(root, "pci", "0000:af:02.0")
	host := filepath.Join(root, "host-dev")
	for _, p := range []string{filepath.Join(pci, "vfio-dev", "vfio2"), filepath.Join(host, "vfio", "devices")} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	var stat unix.Stat_t
	if err := unix.Stat("/dev/null", &stat); err != nil {
		t.Fatal(err)
	}
	number := fmt.Sprintf("%d:%d\n", unix.Major(stat.Rdev), unix.Minor(stat.Rdev))
	if err := os.WriteFile(filepath.Join(pci, "vfio-dev", "vfio2", "dev"), []byte(number), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/null", filepath.Join(host, "vfio", "devices", "vfio2")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/sys/bus/pci/drivers/vfio-pci", filepath.Join(pci, "driver")); err != nil {
		t.Fatal(err)
	}
	return pci, host
}

func TestResolveVFIOCdev(t *testing.T) {
	for _, tc := range []struct {
		name    string
		change  func(*testing.T, string, string)
		wantErr bool
	}{
		{name: "cdev without legacy nodes"},
		{name: "no IOMMU", change: func(t *testing.T, p, h string) {
			if err := os.MkdirAll(filepath.Join(p, "iommu_group"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p, "iommu_group/name"), []byte("vfio-noiommu\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true},
		{name: "missing sysfs identity", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.Remove(filepath.Join(p, "vfio-dev/vfio2/dev")))
		}, wantErr: true},
		{name: "missing node", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.Remove(filepath.Join(h, "vfio/devices/vfio2")))
		}, wantErr: true},
		{name: "regular file", change: func(t *testing.T, p, h string) {
			n := filepath.Join(h, "vfio/devices/vfio2")
			requireCdevChange(t, os.Remove(n))
			requireCdevChange(t, os.WriteFile(n, []byte(""), 0600))
		}, wantErr: true},
		{name: "wrong device number", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.WriteFile(filepath.Join(p, "vfio-dev/vfio2/dev"), []byte("999:999"), 0600))
		}, wantErr: true},
		{name: "malformed device number", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.WriteFile(filepath.Join(p, "vfio-dev/vfio2/dev"), []byte("1:3 garbage"), 0600))
		}, wantErr: true},
		{name: "ambiguous cdev", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.Mkdir(filepath.Join(p, "vfio-dev/vfio3"), 0755))
		}, wantErr: true},
		{name: "invalid cdev name", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.Rename(filepath.Join(p, "vfio-dev/vfio2"), filepath.Join(p, "vfio-dev/vfio-not-a-number")))
		}, wantErr: true},
		{name: "wrong driver", change: func(t *testing.T, p, h string) {
			requireCdevChange(t, os.Remove(filepath.Join(p, "driver")))
			requireCdevChange(t, os.Symlink("/sys/bus/pci/drivers/mlx5_core", filepath.Join(p, "driver")))
		}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pci, host := cdevFixture(t)
			if tc.change != nil {
				tc.change(t, pci, host)
			}
			got, err := ResolveVFIOCdev(pci, host)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected failure, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != "/dev/vfio/devices/vfio2" {
				t.Fatalf("got %q; inspection root must not leak into device spec", got)
			}
		})
	}
}

func requireCdevChange(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
