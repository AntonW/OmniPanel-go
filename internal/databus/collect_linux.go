//go:build linux

package databus

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// collectCPU reads /proc/stat and calculates CPU usage percentage.
// CPU usage = (total_delta - idle_delta) / total_delta * 100
// Skips the first call (no baseline yet).
func (db *DataBus) collectCPU() {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	fields := strings.Fields(lines[0]) // "cpu  user nice system idle iowait ..."
	if len(fields) < 5 {
		return
	}

	var idle, total uint64
	for i := 1; i < len(fields); i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += val
		if i == 4 || i == 5 { // idle + iowait
			idle += val
		}
	}

	if db.lastCPUTotal > 0 {
		deltaTotal := total - db.lastCPUTotal
		deltaIdle := idle - db.lastCPUIdle
		if deltaTotal > 0 {
			usage := float64(deltaTotal-deltaIdle) / float64(deltaTotal) * 100.0
			db.mu.Lock()
			db.data["cpu_usage"] = DataValue{Value: usage, Unit: "%", Source: "system"}
			db.mu.Unlock()
		}
	}

	db.lastCPUIdle = idle
	db.lastCPUTotal = total
}

// collectMemory reads /proc/meminfo and calculates memory usage percentage.
// Uses bufio.Scanner for memory-efficient line-by-line reading.
func (db *DataBus) collectMemory() {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer file.Close()

	var total, available uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			total, _ = strconv.ParseUint(fields[1], 10, 64)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			available, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}

	if total > 0 {
		used := total - available
		usage := float64(used) / float64(total) * 100.0
		db.mu.Lock()
		db.data["memory_usage"] = DataValue{Value: usage, Unit: "%", Source: "system"}
		db.mu.Unlock()
	}
}

// collectDisk uses syscall.Statfs to get root filesystem usage.
// Usage = (total - free) / total * 100
func (db *DataBus) collectDisk() {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total > 0 {
		used := total - free
		usage := float64(used) / float64(total) * 100.0
		db.mu.Lock()
		db.data["disk_usage"] = DataValue{Value: usage, Unit: "%", Source: "system"}
		db.mu.Unlock()
	}
}

// collectNetwork reads /proc/net/dev and calculates network throughput (KB/s).
// Skips the loopback interface (lo). Uses delta-based rate calculation.
func (db *DataBus) collectNetwork() {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer file.Close()

	var totalRx, totalTx uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip header lines and lines without interface names
		if !strings.Contains(line, ":") || strings.HasPrefix(line, "Inter-") || strings.HasPrefix(line, "face") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue // skip loopback
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 9 {
			continue
		}

		rx, _ := strconv.ParseUint(fields[0], 10, 64) // received bytes
		tx, _ := strconv.ParseUint(fields[8], 10, 64) // transmitted bytes
		totalRx += rx
		totalTx += tx
	}

	now := time.Now()
	if !db.lastNetTime.IsZero() {
		elapsed := now.Sub(db.lastNetTime).Seconds()
		if elapsed > 0 {
			db.mu.RLock()
			lastRx := db.lastNetRx["total"]
			lastTx := db.lastNetTx["total"]
			db.mu.RUnlock()

			rxRate := float64(totalRx-lastRx) / elapsed / 1024.0
			txRate := float64(totalTx-lastTx) / elapsed / 1024.0

			db.mu.Lock()
			db.data["network_rx"] = DataValue{Value: rxRate, Unit: "KB/s", Source: "system"}
			db.data["network_tx"] = DataValue{Value: txRate, Unit: "KB/s", Source: "system"}
			db.mu.Unlock()
		}
	}

	db.mu.Lock()
	db.lastNetRx["total"] = totalRx
	db.lastNetTx["total"] = totalTx
	db.lastNetTime = now
	db.mu.Unlock()
}
