# Chapter 12: Virtual Joysticks

This chapter explains how OmniPanel-go creates virtual game controllers that your games can detect and use. Understanding this helps you configure your panels correctly and troubleshoot input issues.

## What Are Virtual Joysticks?

When you tap a button or move a slider on your panel, OmniPanel-go doesn't send that input directly to your game. Instead, it creates a **virtual joystick** — a fake game controller that your PC sees as a real device. Your game then receives inputs from this virtual joystick just like it would from a physical gamepad.

## How Many Inputs Per Joystick?

Each virtual joystick provides:

- **8 analog axes** — for sliders, touchpads, and continuous controls
- **16 buttons** — for buttons, toggles, and on/off controls

### The 8 Axes

| Axis | Name | Typical Use |
|------|------|-------------|
| 0 | X | Horizontal movement (left-right) |
| 1 | Y | Vertical movement (up-down) |
| 2 | Z | Depth or third dimension |
| 3 | RX | Rotation around X axis (roll) |
| 4 | RY | Rotation around Y axis (yaw) |
| 5 | RZ | Rotation around Z axis (pitch) |
| 6 | Throttle | Throttle control |
| 7 | Rudder | Rudder control |

All axes have a value range of **0 to 255**:
- **0** = minimum (full left, full up, etc.)
- **127** = center/neutral
- **255** = maximum (full right, full down, etc.)

### The 16 Buttons

Buttons are numbered 0 through 15. Each button can be:
- **Pressed** (active)
- **Released** (inactive)

---

## Platform Differences

### Linux

On Linux, OmniPanel-go uses the built-in `uinput` kernel module. This is part of the Linux kernel, so:

- **No extra software needed** — it works out of the box
- **Requires root/admin permissions** — OmniPanel-go needs access to `/dev/uinput`

If you get permission errors, run OmniPanel-go with `sudo`:
```bash
sudo ./omnipanel-go
```

Or add your user to the `input` group:
```bash
sudo usermod -aG input $USER
```
(You'll need to log out and back in for this to take effect.)

### Windows

On Windows, OmniPanel-go uses the **vJoy driver** to create virtual joysticks.

**Installation:**
1. Download vJoy version **2.2.2.0** from the [official vJoy releases page](https://github.com/BrunnerInnovation/vJoy/releases/tag/v2.2.2.0)
2. Run the installer
3. During installation, make sure to enable the number of devices you want (at least 4 is recommended)
4. Restart your computer if prompted

**Getting vJoyInterface.dll:**
After installing vJoy, OmniPanel-go will automatically find the `vJoyInterface.dll` file it needs. It searches in these locations:
- Next to the OmniPanel-go executable (if bundled)
- Standard vJoy installation directories: `C:\Program Files\vJoy\` or `C:\Program Files (x86)\vJoy\`
- Your system PATH

If you built OmniPanel-go from source using the `scripts/build-with-vosk.ps1` script, the DLL will be automatically copied next to your executable. Otherwise, OmniPanel-go will find it in the vJoy installation directory the first time it runs.

**Note:** vJoy is only needed for virtual joysticks. Virtual mouse and keyboard input use Windows' built-in `SendInput` function and don't require any extra drivers.

### macOS

macOS is **not supported** for virtual joystick or mouse input. You can still run the OmniPanel-go web server and design panels on a Mac, but the input simulation won't work.

---

## Configuring the Number of Joysticks

### In config.json

Open `config.json` and set `numJoysticks`:

```json
{
  "numJoysticks": 5
}
```

This creates 5 virtual joysticks (indices 0, 1, 2, 3, 4).

### Default Value

The default is **5 joysticks**, which gives you:
- 32 buttons per joystick × 5 = 80 total buttons
- 8 axes per joystick × 5 = 40 total axes

This is enough for most panels. If you need more inputs, increase the number.

### Changing at Runtime

You can also change the joystick count from the Start Page without restarting OmniPanel-go. Look for the joystick count control and adjust it there.

---

## Virtual Mouse Input

In addition to virtual joysticks, OmniPanel-go can simulate mouse input.

### Capabilities

- **Mouse movement** — move the cursor relative to its current position
- **Left click** — simulate a left mouse button press
- **Right click** — simulate a right mouse button press
- **Middle click** — simulate a middle mouse button press (scroll wheel click)
- **Scroll wheel** — simulate scrolling up or down

### Platform Support

- **Linux:** Uses `uinput` (same as virtual joysticks)
- **Windows:** Uses built-in `SendInput` (no extra driver needed)
- **macOS:** Not supported

### Using Mouse Input

Mouse input is primarily used through **Mousepad blocks** on your panel. Configure the mousepad's sensitivity to control how fast the cursor moves.

---

## Virtual Keyboard Input

OmniPanel-go can also simulate keyboard input, letting you send key presses and combinations (like Ctrl+A) to any application.

### Capabilities

- **Single key press** — any letter, number, function key, or special key
- **Key combinations** — modifier + key (e.g., Ctrl+A, Ctrl+Shift+Esc, Alt+F4)
- **Supported modifiers:** Ctrl, Shift, Alt, Meta (Windows/Command key)

### Platform Support

- **Linux:** Uses `uinput` (same as virtual joysticks and mouse)
- **Windows:** Uses built-in `SendInput` (no extra driver needed)
- **macOS:** Not supported

### Using Keyboard Input

Keyboard input is configured through the **Keyboard Key** setting on any button or slider block:

1. Click the gear icon on a button or slider
2. Set **Keyboard Key** to the key or combination you want (e.g., `"a"`, `"ctrl+a"`)
3. Set **Keyboard Index** to choose which virtual keyboard (usually 0)

When you click the button on your panel, the key press is sent to the host system. This works with any application that accepts keyboard input.

### Building an On-Screen Keyboard

You can create a full on-screen keyboard panel by placing button blocks for each key and setting their Keyboard Key values. The included **Keyboard** panel template demonstrates this layout.

### Available Key Names

Use these names in the Keyboard Key setting:

| Category | Examples |
|----------|----------|
| Letters | `a` through `z` |
| Numbers | `0` through `9` |
| Function keys | `f1` through `f12` |
| Modifiers | `ctrl`, `shift`, `alt`, `meta` |
| Special keys | `space`, `enter`, `escape`, `tab`, `backspace`, `capslock` |
| Punctuation | `minus`, `equal`, `comma`, `dot`, `slash`, `semicolon`, `apostrophe`, `grave`, `backslash`, `leftbracket`, `rightbracket` |

Combine modifiers with `+`: `ctrl+a`, `ctrl+shift+a`, `alt+f4`, `meta+e`.

---

## Mapping Panel Controls to Joystick Inputs

### Button Blocks

Each button block needs:
- **Joystick Index** — which virtual controller (0-9)
- **Button ID** — which button on that controller (0-15)

**Example:** A button with Joystick Index 0 and Button ID 3 controls the 4th button on the 1st virtual joystick.

### Slider Blocks

Each slider block needs:
- **Joystick Index** — which virtual controller (0-9)
- **Slider ID** — which axis on that controller (0-7)

**Example:** A slider with Joystick Index 0 and Slider ID 0 controls the X axis (first axis) on the 1st virtual joystick.

### Touch Pad Blocks

Each touchpad block needs:
- **Joystick Index** — which virtual controller (0-9)
- **Axis ID** — which pair of axes to use (0-3)

Axis ID mapping:
- **Axis ID 0** — uses Axis 0 (X) and Axis 1 (Y)
- **Axis ID 1** — uses Axis 2 (Z) and Axis 3 (RX)
- **Axis ID 2** — uses Axis 4 (RY) and Axis 5 (RZ)
- **Axis ID 3** — uses Axis 6 (Throttle) and Axis 7 (Rudder)

---

## Setting Up Games to Use Virtual Joysticks

### Step 1: Start OmniPanel-go

Make sure OmniPanel-go is running before launching your game. The virtual joysticks are created when OmniPanel-go starts.

### Step 2: Launch Your Game

Start your game as usual.

### Step 3: Configure Game Controls

In your game's control settings:

1. Go to the input/controller configuration
2. Look for joystick or gamepad settings
3. Select the virtual joystick (it may appear as "vJoy Device" on Windows or "OmniPanel-go Joystick" on Linux)
4. Map the buttons and axes to game actions

### Step 4: Test Inputs

1. Open your panel on your tablet
2. Tap a button or move a slider
3. Check if the game responds
4. If not, verify the joystick index and button/axis IDs match your game's configuration

---

## Troubleshooting Virtual Joysticks

### Game Doesn't Detect the Virtual Joystick

**Windows:**
- Make sure vJoy is installed and configured
- Open the vJoy configuration tool and verify devices are enabled
- Ensure `vJoyInterface.dll` is in one of these locations:
  - Next to `omnipanel-go.exe`
  - In your vJoy installation directory
  - In your system PATH
- Restart OmniPanel-go after installing vJoy
- Check that the DLL is not quarantined by antivirus software

**Linux:**
- Check that you have permission to access `/dev/uinput`
- Try running OmniPanel-go with `sudo`
- Verify the `uinput` kernel module is loaded: `lsmod | grep uinput`

### Inputs Aren't Working

- Verify the **Joystick Index** in your block settings matches the joystick your game is listening to
- Check the **Button ID** or **Slider ID** matches the correct input
- Use the **input allocation view** in the editor to see which inputs are already in use
- Make sure no two blocks are using the same input (unless you want them to do the same thing)

### Too Few Inputs

If you run out of buttons or axes:

1. Increase `numJoysticks` in `config.json`
2. Restart OmniPanel-go
3. Use higher Joystick Index values in your blocks (1, 2, 3, etc.)

---

## Tips for Virtual Joysticks

### Organize Your Inputs

Keep a record of which inputs you've assigned:

```
Joystick 0:
  Button 0: Landing Gear
  Button 1: Flaps Up
  Button 2: Flaps Down
  Axis 0: Throttle (Slider)
  Axis 1: Camera X/Y (Touch Pad)

Joystick 1:
  Button 0: Weapon Group 1
  Button 1: Weapon Group 2
  ...
```

### Consistent Numbering

Use a consistent numbering scheme across your panels:

- Joystick 0: Primary flight controls
- Joystick 1: Weapons and combat
- Joystick 2: Systems and utilities
- Joystick 3: Navigation and communication

This makes it easier to remember which input does what.

### Testing Without a Game

You can test virtual joystick inputs without launching a game:

1. On Windows, use the "Set up USB game controllers" tool in Control Panel
2. On Linux, use `jstest` or `evtest` to monitor joystick events
3. Tap buttons and move sliders on your panel — you should see the inputs register

## What's Next?

Want to create your own custom block designs? Learn about [Custom Blocks](13-custom-blocks.md).
