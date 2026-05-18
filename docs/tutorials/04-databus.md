# Chapter 4: DataBus & System Metrics

## What This Package Does

The `DataBus` is a thread-safe key-value store that collects system metrics. It provides a unified interface for both system-collected metrics (CPU, memory, disk, network) and externally-pushed data (from the web UI or API).

> **Platform Note:** Metric collection is platform-specific. Linux reads from `/proc` and uses `syscall.Statfs`. Windows uses `GetDiskFreeSpaceExW` for disk (CPU, memory, and network are stubbed). Other platforms are no-ops. The collector implementations live in `collect_linux.go`, `collect_windows.go`, and `collect_stub.go`, while `databus.go` contains the shared thread-safe store and public API.

## The Data Structures

```go
// internal/databus/databus.go (shared) + collect_linux.go (Linux collectors)
type DataValue struct {
    Value  any    `json:"value"`
    Unit   string `json:"unit"`
    Source string `json:"source,omitempty"`
}

type DataBus struct {
    mu         sync.RWMutex
    data       map[string]DataValue

    lastCPUIdle   uint64
    lastCPUTotal  uint64

    lastNetRx     map[string]uint64
    lastNetTx     map[string]uint64
    lastNetTime   time.Time
}
```

`DataValue` wraps any value with a unit label and an optional source tag. The `Source` field has `omitempty`, meaning it won't appear in JSON output if empty.

The "last" fields store previous readings for **delta calculations**. CPU usage and network speed can't be read directly — you need two samples and compute the difference.

## Initialization

```go
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
```

The first call to each `collect*` method seeds the "last" values. The first CPU and network readings won't produce output (no previous sample to compare against), but they establish the baseline.

## Thread-Safe Get/Set

```go
func (db *DataBus) Set(key string, value any, unit string) {
    db.mu.Lock()
    defer db.mu.Unlock()
    db.data[key] = DataValue{Value: value, Unit: unit, Source: "api"}
}

func (db *DataBus) Get(key string) (DataValue, bool) {
    db.mu.RLock()
    defer db.mu.RUnlock()
    v, ok := db.data[key]
    return v, ok
}

func (db *DataBus) Snapshot() map[string]DataValue {
    db.mu.RLock()
    defer db.mu.RUnlock()
    result := make(map[string]DataValue)
    for k, v := range db.data {
        result[k] = v
    }
    return result
}
```

> **Key Pattern: Defensive copy in Snapshot**
> `Snapshot()` returns a **copy** of the map, not the original. If it returned `db.data` directly, callers could modify the internal state without holding the mutex. The copy is made under `RLock` to ensure consistency.

## Collecting CPU Usage

> **Linux-specific:** The collectors below are from `collect_linux.go`. On Windows, CPU/memory/network collection is stubbed (see `collect_windows.go`), and on other platforms all collectors are no-ops (`collect_stub.go`).

```go
func (db *DataBus) collectCPU() {
    data, err := os.ReadFile("/proc/stat")
    if err != nil {
        return
    }

    lines := strings.Split(string(data), "\n")
    fields := strings.Fields(lines[0])
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
        if i == 4 || i == 5 {
            idle += val
        }
    }
```

`/proc/stat` contains a line like:
```
cpu  123456 789 12345 987654321 45678 1234 5678 0 0
```

The fields (after `cpu`) are: user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice. All values are cumulative since boot.

- `total` = sum of all fields
- `idle` = idle (index 4) + iowait (index 5)

```go
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
```

> **Concept: Delta-based measurement**
> Since `/proc/stat` values are cumulative, CPU usage = (total_delta - idle_delta) / total_delta × 100. The `if db.lastCPUTotal > 0` check skips the first call (no baseline yet).

## Collecting Memory Usage

```go
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
```

> **Concept: `bufio.Scanner`**
> For reading line-by-line from a file, `bufio.Scanner` is more memory-efficient than `os.ReadFile` because it doesn't load the entire file at once. `/proc/meminfo` can be large, so streaming is better.

`/proc/meminfo` format:
```
MemTotal:       16384000 kB
MemAvailable:   12288000 kB
```

## Collecting Disk Usage

> **Linux-specific:** The code below is from `collect_linux.go`. Windows uses `GetDiskFreeSpaceExW` (see `collect_windows.go`), and other platforms are no-ops (`collect_stub.go`).

```go
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
```

> **Concept: `syscall.Statfs`**
> This is a direct system call that queries filesystem statistics. `stat.Blocks` is the total number of blocks, `stat.Bfree` is free blocks, and `stat.Bsize` is the block size in bytes. Multiplying gives bytes.

## Collecting Network Throughput

```go
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
        if !strings.Contains(line, ":") || strings.HasPrefix(line, "Inter-") || strings.HasPrefix(line, "face") {
            continue
        }

        parts := strings.SplitN(line, ":", 2)
        if len(parts) != 2 {
            continue
        }

        iface := strings.TrimSpace(parts[0])
        if iface == "lo" {
            continue
        }

        fields := strings.Fields(parts[1])
        if len(fields) < 9 {
            continue
        }

        rx, _ := strconv.ParseUint(fields[0], 10, 64)
        tx, _ := strconv.ParseUint(fields[8], 10, 64)
        totalRx += rx
        totalTx += tx
    }
```

`/proc/net/dev` format:
```
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 123456    789    0    0    0     0          0         0  123456    789    0    0    0     0       0          0
  eth0: 987654321 456789  0    0    0     0          0         0  123456789 321654  0    0    0     0       0          0
```

The code skips the header lines and the loopback interface (`lo`), then sums RX (field 0) and TX (field 8) across all real interfaces.

```go
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
```

Network speed = bytes_delta / seconds_elapsed / 1024 (to get KB/s).

## The Metrics Collection Loop

```go
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
}
```

Called from `AppState.StartDataBroadcast()`. Runs all four collectors on a timer in a background goroutine.

## Key Takeaways

- Linux exposes system info through `/proc` files — no special libraries needed
- CPU and network metrics require delta calculations (two samples over time)
- `bufio.Scanner` is memory-efficient for line-by-line file reading
- `syscall.Statfs` provides direct access to filesystem statistics
- Always copy data under lock when returning internal state
- The `Source` field distinguishes system-collected data from API-pushed data

[← Back: Chapter 3](03-state-and-concurrency.md) · [Next: Chapter 5 →](05-http-routing.md)
