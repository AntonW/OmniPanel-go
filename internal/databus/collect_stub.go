//go:build !linux && !windows

package databus

// collectCPU is a no-op on unsupported platforms.
func (db *DataBus) collectCPU() {}

// collectMemory is a no-op on unsupported platforms.
func (db *DataBus) collectMemory() {}

// collectDisk is a no-op on unsupported platforms.
func (db *DataBus) collectDisk() {}

// collectNetwork is a no-op on unsupported platforms.
func (db *DataBus) collectNetwork() {}
