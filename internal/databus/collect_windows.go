//go:build windows

package databus

import (
	"syscall"
	"unsafe"
)

var (
	modKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceExW = modKernel32.NewProc("GetDiskFreeSpaceExW")
)

// collectCPU returns placeholder on Windows.
// Full Windows CPU collection would require Performance Counters or WMI.
func (db *DataBus) collectCPU() {
	// Placeholder: Windows CPU collection requires WMI or Performance Counters
	// which would add external dependencies. Stubbed for now.
}

// collectMemory returns placeholder on Windows.
// Full Windows memory collection would require GlobalMemoryStatusEx.
func (db *DataBus) collectMemory() {
	// Placeholder: Windows memory collection requires GlobalMemoryStatusEx
	// Stubbed for now to avoid external dependencies.
}

// collectDisk uses GetDiskFreeSpaceExW to get root filesystem usage.
func (db *DataBus) collectDisk() {
	var freeBytes, totalBytes, availFree int64
	root := syscall.StringToUTF16Ptr("C:\\")

	ret, _, _ := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(root)),
		uintptr(unsafe.Pointer(&availFree)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&freeBytes)),
	)
	if ret == 0 {
		return
	}

	total := uint64(totalBytes)
	free := uint64(freeBytes)
	if total > 0 {
		used := total - free
		usage := float64(used) / float64(total) * 100.0
		db.mu.Lock()
		db.data["disk_usage"] = DataValue{Value: usage, Unit: "%", Source: "system"}
		db.mu.Unlock()
	}
}

// collectNetwork returns placeholder on Windows.
// Full Windows network collection would require GetIfTable or IPHelper API.
func (db *DataBus) collectNetwork() {
	// Placeholder: Windows network collection requires IPHelper API
	// Stubbed for now to avoid external dependencies.
}
