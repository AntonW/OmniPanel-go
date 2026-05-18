// Package databus provides a thread-safe key-value store for system metrics.
// It collects CPU, memory, disk, and network stats from the OS using
// platform-specific collectors (see collect_linux.go, collect_windows.go,
// collect_stub.go).
//
// Metrics are collected via delta-based calculations — two samples over time
// are needed to compute rates (CPU usage %, network KB/s). The first reading
// seeds the baseline; subsequent readings produce output.
//
// External systems can also push custom metrics via Set/SetSource, which are
// then included in the periodic Snapshot broadcasts to WebSocket clients.
//
// See docs/tutorials/04-databus.md for a detailed walkthrough.
package databus

import (
	"log/slog"
	"sync"
	"time"
)

// DataValue wraps a metric value with its unit and source label.
// Source distinguishes system-collected metrics from API-pushed data.
type DataValue struct {
	Value  any    `json:"value"`
	Unit   string `json:"unit"`
	Source string `json:"source,omitempty"` // omitted if empty
}

// DataBus is a thread-safe key-value store for system metrics.
// It collects CPU, memory, disk, and network stats from the OS.
// The "last" fields store previous readings for delta-based calculations.
type DataBus struct {
	mu   sync.RWMutex
	data map[string]DataValue

	lastCPUIdle  uint64
	lastCPUTotal uint64

	lastNetRx   map[string]uint64
	lastNetTx   map[string]uint64
	lastNetTime time.Time
}

// New creates a DataBus and seeds initial metric readings.
// The first readings establish baselines for delta calculations.
func New() *DataBus {
	db := &DataBus{
		data:      make(map[string]DataValue),
		lastNetRx: make(map[string]uint64),
		lastNetTx: make(map[string]uint64),
	}
	db.collectCPU()
	db.collectMemory()
	db.collectDisk()
	db.collectNetwork()
	return db
}

// StartMetrics launches a background goroutine that collects system metrics
// at the specified interval.
func (db *DataBus) StartMetrics(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			db.collectCPU()
			db.collectMemory()
			db.collectDisk()
			db.collectNetwork()
		}
	}()
	slog.Info("DataBus system metrics started", "interval", interval)
}

// Set stores a metric value with the source "api".
func (db *DataBus) Set(key string, value any, unit string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = DataValue{Value: value, Unit: unit, Source: "api"}
}

// SetSource stores a metric value with a custom source label.
func (db *DataBus) SetSource(key string, value any, unit string, source string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = DataValue{Value: value, Unit: unit, Source: source}
}

// Get retrieves a metric value. Returns (value, true) or (zero, false) if not found.
func (db *DataBus) Get(key string) (DataValue, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	v, ok := db.data[key]
	return v, ok
}

// Snapshot returns a defensive copy of all metrics.
// Copying under the lock ensures callers can't mutate internal state.
func (db *DataBus) Snapshot() map[string]DataValue {
	db.mu.RLock()
	defer db.mu.RUnlock()
	result := make(map[string]DataValue)
	for k, v := range db.data {
		result[k] = v
	}
	return result
}
