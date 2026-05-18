# Chapter 10: Live Data Display

Want to see your PC's CPU usage, RAM usage, or custom data right on your panel? This chapter covers how to display live, updating data on your control panels.

## What Is Live Data?

Live data is information that updates in real-time on your panel. Instead of static buttons and sliders, you can have displays that show:

- CPU usage percentage
- Memory (RAM) usage
- Disk usage
- Network download/upload speed
- Any custom data you want to push

## System Metrics (Built-In)

OmniPanel-go automatically collects system metrics every 500 milliseconds (twice per second). These metrics are always available — you don't need to configure anything to use them.

### Available System Metrics

| Data Key | What It Shows | Example Value |
|----------|--------------|---------------|
| `cpu_usage` | CPU utilization percentage | 45.2 |
| `memory_usage` | RAM utilization percentage | 62.8 |
| `disk_usage` | Hard drive usage percentage | 71.3 |
| `network_rx` | Network download speed (KB/s) | 1250.5 |
| `network_tx` | Network upload speed (KB/s) | 342.1 |

> **Note:** On Windows, only `disk_usage` is available for system metrics. Full system metrics (CPU, RAM, network) are available on Linux.

---

## Data Display Block

### What It Does

The Data Display Block shows a live data value on your panel. It updates automatically whenever the data changes.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Data Key** | The name of the data to display | `cpu_usage`, `memory_usage` |
| **Title** | Label shown above the value | "CPU", "RAM", "Network Down" |
| **Unit** | Text shown after the value | "%", "KB/s", "°C" |
| **Decimal Places** | How many digits after the decimal point | 0, 1, 2 |
| **Text Color** | Color of the value text | Any color |
| **Background Color** | Block background color | Any color |
| **Size** | How large the display is | Small, Medium, Large |

### Adding a Data Display Block

1. Drag a **Data Display** block onto your workspace
2. Click the gear icon
3. Set the **Data Key** to one of the system metrics (e.g., `cpu_usage`)
4. Set the **Title** (e.g., "CPU")
5. Set the **Unit** (e.g., "%")
6. Set **Decimal Places** to 1 for one digit after the decimal
7. Customize colors and size
8. Save the settings

### Example Configurations

**CPU Usage Display:**
- Data Key: `cpu_usage`
- Title: `CPU`
- Unit: `%`
- Decimal Places: `1`
- Result: Shows "CPU: 45.2%"

**Network Download Speed:**
- Data Key: `network_rx`
- Title: `Download`
- Unit: `KB/s`
- Decimal Places: `0`
- Result: Shows "Download: 1250 KB/s"

**Memory Usage:**
- Data Key: `memory_usage`
- Title: `RAM`
- Unit: `%`
- Decimal Places: `1`
- Result: Shows "RAM: 62.8%"

### Visual Feedback

When a data value changes, the display **flashes briefly** to draw your attention. This helps you notice changes at a glance without staring at the numbers.

---

## Pushing Custom Data

You can send your own data to OmniPanel-go from external programs, scripts, or web applications. This data then appears on your panel just like system metrics.

### Method 1: HTTP API

Send a POST request to OmniPanel-go's data endpoint:

```
POST http://YOUR-PC-IP:3000/api/data/push
Content-Type: application/json

{
  "key": "server_temp",
  "value": 42.5,
  "unit": "C",
  "source": "sensors"
}
```

| Field | What It Does | Required |
|-------|-------------|----------|
| `key` | The data name (used in Data Display blocks) | Yes |
| `value` | The data value (number or text) | Yes |
| `unit` | Unit label (optional, shown on display) | No |
| `source` | Where the data came from (for organization) | No |

### Example: Using curl (Linux/Mac)

```bash
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "gpu_temp", "value": 65, "unit": "°C"}'
```

### Example: Using PowerShell (Windows)

```powershell
Invoke-RestMethod -Uri "http://192.168.1.100:3000/api/data/push" -Method POST -ContentType "application/json" -Body '{"key": "gpu_temp", "value": 65, "unit": "°C"}'
```

### Example: Using Python

```python
import requests

requests.post("http://192.168.1.100:3000/api/data/push", json={
    "key": "gpu_temp",
    "value": 65,
    "unit": "°C"
})
```

### Displaying Custom Data

Once you push data with a key (like `gpu_temp`), you can display it on your panel:

1. Add a Data Display block
2. Set the Data Key to `gpu_temp`
3. Set the Title to "GPU Temp"
4. Set the Unit to "°C"
5. The block now shows your custom data

### Method 2: WebSocket

For applications that need to push data continuously, you can use WebSocket connections. This is more advanced and typically used by developers building integrations.

---

## Star Citizen Themed Data Display

### What It Does

Same functionality as the default Data Display, but with a sci-fi visual style — glowing text and angular design.

### Settings

Same as the default Data Display block, with additional visual customization for the sci-fi aesthetic.

---

## Practical Examples

### Example 1: System Monitor Panel

Create a panel that shows your PC's health:

- **CPU Usage** — Data Key: `cpu_usage`, Unit: `%`
- **RAM Usage** — Data Key: `memory_usage`, Unit: `%`
- **Disk Usage** — Data Key: `disk_usage`, Unit: `%`
- **Download Speed** — Data Key: `network_rx`, Unit: `KB/s`
- **Upload Speed** — Data Key: `network_tx`, Unit: `KB/s`

### Example 2: Game Server Monitor

If you run a game server, push server data to your panel:

```bash
# Push player count
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "players", "value": 24, "unit": "players"}'

# Push server FPS
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "server_fps", "value": 60, "unit": "FPS"}'
```

Then display them on your panel with Data Display blocks.

### Example 3: Smart Home Dashboard

Push sensor data from smart home devices:

```bash
# Temperature
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "living_room_temp", "value": 22.5, "unit": "°C"}'

# Humidity
curl -X POST http://192.168.1.100:3000/api/data/push \
  -H "Content-Type: application/json" \
  -d '{"key": "living_room_humidity", "value": 45, "unit": "%"}'
```

---

## Tips for Data Displays

### Update Frequency

- System metrics update every **500ms** (twice per second)
- Custom data updates **immediately** when you push it
- Your panel reflects changes as soon as they arrive

### Decimal Places

- Use **0 decimal places** for whole numbers (player counts, FPS)
- Use **1 decimal place** for percentages and temperatures
- Use **2 decimal places** for precise measurements

### Color Coding

Use colors to indicate status:

- **Green** — normal values (CPU under 50%)
- **Yellow** — caution values (CPU 50-80%)
- **Red** — warning values (CPU over 80%)

> **Note:** The Data Display block doesn't automatically change colors based on values. You'd need to create multiple displays or use custom blocks for dynamic coloring.

### Sizing

- **Small** — good for compact panels with many displays
- **Medium** — balanced for most uses
- **Large** — good for important metrics you want to see at a glance

## What's Next?

Want to control your panel with your voice? Learn about [Speech Commands](11-speech-commands.md).
