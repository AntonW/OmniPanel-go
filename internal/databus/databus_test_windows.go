//go:build windows

package databus

import (
	"syscall"
	"testing"
)

func TestCollectDisk_FailurePath(t *testing.T) {
	db := New()
	old := procGetDiskFreeSpaceExW
	// RemoveDirectoryW("C:\\") reliably fails and returns 0 (BOOL false),
	// which exercises the ret==0 early return branch in collectDisk.
	procGetDiskFreeSpaceExW = syscall.NewLazyDLL("kernel32.dll").NewProc("RemoveDirectoryW")
	defer func() { procGetDiskFreeSpaceExW = old }()

	db.collectDisk()
}

