# Chapter 8: Command Blocks

Command Blocks let you run programs on your PC or send web requests by tapping a button on your panel. This chapter covers everything you can do with Command Blocks.

## What Is a Command Block?

A Command Block is a button that, when pressed, executes a command instead of sending a joystick input. You can use it to:

- Launch games or applications
- Control system settings (volume, brightness)
- Send data to other programs
- Trigger web services
- Run scripts

## Two Modes

Command Blocks work in two modes:

### Shell Mode

Runs a command directly on your PC, just like typing it in a terminal or command prompt.

**Examples:**
- `notepad` — opens Notepad on Windows
- `echo Hello` — prints "Hello" to the console
- `ls -la` — lists files on Linux
- `volume set 50` — sets system volume (if you have a volume control tool)

### HTTP Mode

Sends a web request to a server or API.

**Examples:**
- Send a GET request to check a server status
- Send a POST request to control a smart home device
- Send data to a web application

## Creating a Command Block

1. Drag a **Command Block** from the library onto your workspace
2. Click the gear icon to open settings
3. Choose the command type: **Shell** or **HTTP**

---

## Shell Mode Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown on the button | "Launch Game", "Screenshot" |
| **Icon URL** | Path to an icon image (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | How to display icon and text | `image-text`, `image-only`, `text-only` |
| **Text Position** | Where the label appears relative to the icon | `bottom`, `top`, `left`, `right` |
| **Command Type** | Set to "Shell" | Shell |
| **Command** | The command to run | `notepad`, `echo Hello`, `calc` |
| **Button Color** | Button fill color | Any color |
| **Button Color Active** | Color when button is pressed | Any color |
| **Label Color** | Label text color | Any color |
| **Border Color** | Button outline color | Any color |
| **Hold Repeat** | Enable hold-to-repeat (see below) | On or Off |
| **Repeat Interval** | How often to repeat (in milliseconds) | 500, 1000, 2000 |

### Command Examples

**Windows:**
```
notepad          - Opens Notepad
calc             - Opens Calculator
mspaint          - Opens Paint
explorer .       - Opens File Explorer in current folder
```

**Linux:**
```
gedit            - Opens Gedit text editor
gnome-calculator - Opens Calculator
xdg-open .       - Opens file manager in current folder
```

---

## HTTP Mode Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown on the button | "Toggle Light", "Send Data" |
| **Icon URL** | Path to an icon image (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | How to display icon and text | `image-text`, `image-only`, `text-only` |
| **Text Position** | Where the label appears relative to the icon | `bottom`, `top`, `left`, `right` |
| **Command Type** | Set to "HTTP" | HTTP |
| **Method** | HTTP request type | GET, POST, PUT, DELETE, PATCH |
| **URL** | The web address to send the request to | `http://192.168.1.50/api/light` |
| **Body** | Data to send with the request (for POST/PUT) | `{"on": true}` |
| **Button Color** | Button fill color | Any color |
| **Button Color Active** | Color when button is pressed | Any color |
| **Label Color** | Label text color | Any color |
| **Border Color** | Button outline color | Any color |
| **Hold Repeat** | Enable hold-to-repeat | On or Off |
| **Repeat Interval** | How often to repeat (in milliseconds) | 500, 1000, 2000 |

### HTTP Method Types

| Method | When to Use | Example |
|--------|-------------|---------|
| **GET** | Request data from a server | Get server status |
| **POST** | Send new data to a server | Create a new record |
| **PUT** | Update existing data | Change a setting |
| **DELETE** | Remove data | Delete a file |
| **PATCH** | Partially update data | Modify one field |

### HTTP Body

For POST, PUT, and PATCH requests, you can include a body with the request. This is typically JSON data:

```json
{"action": "turn_on", "brightness": 75}
```

---

## Parameters: Making Commands Adjustable

Parameters let you add adjustable inputs above your Command Block button. Instead of hardcoding values in your command, you can use parameters to change them on the fly.

### Adding Parameters

1. In the Command Block settings, find the **Parameters** section
2. Click **Add Parameter**
3. Configure the parameter:

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Name** | The parameter name (used in the command) | `volume`, `brightness` |
| **Type** | Input type | Number, Text, Toggle |
| **Default Value** | Starting value | 50, "hello", false |
| **Min/Max** | For number type: allowed range | 0 to 100 |
| **Step** | For number type: increment size | 1, 5, 10 |

### Using Parameters in Commands

In your command, use `{name}` where you want the parameter value to appear.

**Shell Example:**
```
echo Volume is set to {volume}
```

If the volume parameter is set to 75, the command becomes:
```
echo Volume is set to 75
```

**HTTP Example:**
```json
{"brightness": {brightness}, "color": "{color}"}
```

If brightness is 80 and color is "red", the body becomes:
```json
{"brightness": 80, "color": "red"}
```

### Parameter Types

**Number:**
- Shows a number input with up/down arrows
- Set min, max, and step values
- Good for: volume levels, brightness, counts

**Text:**
- Shows a text input field
- Good for: names, messages, file paths

**Toggle:**
- Shows an on/off switch
- Sends `true` or `false`
- Good for: enable/disable flags

---

## Hold-to-Repeat

Hold-to-repeat lets you hold down the Command Block button to repeatedly execute the command at set intervals.

### When to Use Hold-to-Repeat

- **Volume control** — hold to continuously increase/decrease volume
- **Scrolling** — hold to keep scrolling
- **Rapid commands** — any command you want to fire repeatedly

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Hold Repeat** | Enable or disable repeat | On or Off |
| **Repeat Interval** | Time between repeats (milliseconds) | 100, 500, 1000 |

### Repeat Interval Guide

- **100ms** — very fast repeat (10 times per second)
- **500ms** — medium repeat (2 times per second)
- **1000ms** — slow repeat (once per second)
- **2000ms** — very slow repeat (once every 2 seconds)

---

## Visual Feedback

Command Blocks provide visual feedback so you know what's happening:

### While Executing

The button shows a **glowing animation** while the command is running.

### On Success

The button border flashes **green** briefly.

### On Error

The button border flashes **red** briefly.

### Toast Notification

A message appears showing the command output. This message:

- Shows what the command returned
- Auto-dismisses after 5 seconds
- Appears at the top of the panel

---

## Practical Examples

### Example 1: Volume Control

**Goal:** Create buttons to increase and decrease system volume.

**Increase Volume Button:**
- Label: "Volume Up"
- Command Type: Shell
- Command: `pactl set-sink-volume @DEFAULT_SINK@ +5%` (Linux) or a Windows volume utility command
- Hold Repeat: On
- Repeat Interval: 200

**Decrease Volume Button:**
- Label: "Volume Down"
- Command Type: Shell
- Command: `pactl set-sink-volume @DEFAULT_SINK@ -5%` (Linux)
- Hold Repeat: On
- Repeat Interval: 200

### Example 2: Launch Game

**Goal:** A button to launch your favorite game.

**Button Settings:**
- Label: "Launch Star Citizen"
- Command Type: Shell
- Command: `C:\Program Files\Roberts Space Industries\StarCitizen\LIVE\StarCitizen.exe` (Windows)

### Example 3: Smart Home Control

**Goal:** Turn on a smart light from your panel.

**Button Settings:**
- Label: "Living Room Light"
- Command Type: HTTP
- Method: POST
- URL: `http://192.168.1.100/api/lights/1`
- Body: `{"on": true}`

### Example 4: Screenshot Button

**Goal:** Take a screenshot with one tap.

**Button Settings:**
- Label: "Screenshot"
- Command Type: Shell
- Command: `gnome-screenshot` (Linux) or a Windows screenshot utility

---

## Tips for Command Blocks

### Security Note

Shell commands run with the same permissions as OmniPanel-go. Be careful with commands that:

- Delete files
- Modify system settings
- Run with administrator/root privileges

### Testing Commands

Before adding a command to your panel, test it in a terminal or command prompt first. This ensures the command works and helps you see what output to expect.

### Error Handling

If a command fails:

1. Check the red flash and error toast notification
2. Read the error message
3. Test the command in a terminal to see the full error
4. Fix the command and try again

### Path Issues

On Windows, if your command includes spaces in the path, wrap it in quotes:

```
"C:\Program Files\My App\app.exe"
```

## What's Next?

Learn how to organize complex panels with [Pages, Tabs, and Sections](09-organizing-panels.md).
