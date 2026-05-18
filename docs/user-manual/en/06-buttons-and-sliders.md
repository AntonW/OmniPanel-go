# Chapter 6: Buttons and Sliders

Buttons and sliders are the most common controls on any panel. This chapter covers every type of button and slider block, all their options, and how to use them.

## Button Block

### What It Does

A button sends a single joystick button press when you tap it. Think of it like a keyboard key or gamepad button — press it, and the game registers a button press. Release it, and the game registers a button release.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown on the button | "Gear", "Fire", "Boost" |
| **Icon URL** | Path to an icon image (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | How to display icon and text | `image-text`, `image-only`, `text-only` |
| **Text Position** | Where the label appears relative to the icon | `bottom`, `top`, `left`, `right` |
| **Joystick Index** | Which virtual controller (0-9) | 0 for first controller |
| **Button ID** | Which button on that controller (0-15) | 0 to 15 |
| **Keyboard Index** | Which virtual keyboard (0-9) for keyboard shortcuts | 0 for first keyboard |
| **Keyboard Key** | Key or combo sent when button is clicked. When set, clicking sends keyboard events instead of joystick button events. | "a", "ctrl+a", "ctrl+shift+escape" |
| **Button Color** | Button fill color | Any color |
| **Button Color Active** | Color when button is pressed | Any color |
| **Label Color** | Label text color | Any color |
| **Border Color** | Button outline color | Any color |
| **Font Size** | Size of the label text | 1 to 48 |
| **Border Radius** | How rounded the corners are | 0 (square) to 100 (circle) |

### How to Use It

1. Drag a Button block onto your panel
2. Click the gear icon
3. Set the label (e.g., "Landing Gear")
4. Set the Joystick Index (usually 0)
5. Set the Button ID (pick an unused number, 0-15)
6. Customize colors to match your panel theme
7. Save the panel

### In Your Game

In your game's control settings, map the corresponding joystick button to the action you want. If your button uses Joystick 0, Button 3, map "Joystick 0 Button 3" to "Toggle Landing Gear" in the game.

---

## Slider Block (Horizontal)

### What It Does

A horizontal slider controls a joystick axis value from 0 to 255 (or a custom max value). Drag the slider thumb left or right to change the value. Games see this as an analog input — like a throttle or steering wheel.

Sliders can also control your computer's system settings directly, like volume or monitor brightness, by running commands on your computer when you move the slider.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown above the slider | "Throttle", "Volume", "Brightness" |
| **Joystick Index** | Which virtual controller (0-9) | 0 for first controller |
| **Slider ID** | Which axis on that controller (0-7) | 0 to 7 |
| **Thumb Color** | Color of the draggable knob | Any color |
| **Background Color** | Color of the slider track | Any color |
| **Progress Color** | Color of the filled portion | Any color |
| **Font Size** | Size of the label text | 12 to 48 |
| **Max Value** | Maximum value the slider can reach | 255 (default), 100 for percentages |
| **System Control** | Label for system control mode (optional) | "volume", "brightness" |
| **System Control Command** | Command to run on your computer when slider moves (optional) | See examples below |

### How to Use It (Joystick Mode)

1. Drag a Slider block onto your panel
2. Click the gear icon
3. Set the label (e.g., "Throttle")
4. Set the Joystick Index (usually 0)
5. Set the Slider ID (pick an unused number, 0-7)
6. Customize colors
7. Save the panel

### How to Use It (System Control Mode)

To make a slider control your computer's volume, brightness, or other settings:

1. Drag a Slider block onto your panel
2. Click the gear icon
3. Set the label (e.g., "Volume")
4. Set **System Control** to a name like "volume"
5. Set **System Control Command** to a command with `{value}` where the slider number should go
6. Set **Max Value** to match your command's expected range (e.g., 100 for percentages)
7. Save the panel

### System Control Command Examples

| What You Want to Control | Command | Max Value |
|--------------------------|---------|-----------|
| System volume (PulseAudio/PipeWire) | `pactl set-sink-volume @DEFAULT_SINK@ {value}%` | 100 |
| System volume (ALSA) | `amixer set Master {value}%` | 100 |
| Monitor brightness (ddcutil, external monitors) | `ddcutil -d 1 setvcp 10 {value}` | 100 |
| Laptop screen brightness | `brightnessctl set {value}%` | 100 |

> **Note:** The `{value}` placeholder is replaced with the current slider position. If your slider goes from 0 to 100 and you set it to 75, the command runs with `75` in place of `{value}`.

### Value Range

- **Leftmost position** = 0
- **Rightmost position** = Max Value (255 by default)
- **Middle position** = Max Value ÷ 2

### In Your Game

Map the corresponding joystick axis in your game. For example, "Joystick 0 Axis 0" could be mapped to "Throttle Control."

---

## Slider Vertical Block

### What It Does

Exactly the same as the horizontal slider, but oriented vertically. Drag the slider thumb up or down to change the value.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown above the slider | "Throttle", "Elevator Trim" |
| **Joystick Index** | Which virtual controller (0-9) | 0 for first controller |
| **Slider ID** | Which axis on that controller (0-7) | 0 to 7 |
| **Thumb Color** | Color of the draggable knob | Any color |
| **Background Color** | Color of the slider track | Any color |
| **Progress Color** | Color of the filled portion | Any color |
| **Font Size** | Size of the label text | 12 to 48 |

### Value Range

- **Bottom position** = 0
- **Top position** = 255
- **Middle position** = 127

### When to Use Vertical vs Horizontal

- **Vertical sliders** — great for throttle controls (push forward for more power, pull back for less)
- **Horizontal sliders** — great for balance controls (left-right adjustments like brake balance)

---

## Star Citizen Themed Button

### What It Does

Same as the default button, but with a sci-fi aesthetic — angled corners and a more futuristic look.

### Additional Settings

| Setting | What It Does |
|---------|-------------|
| **Border Width** | Thickness of the button border |
| **Active Color** | Color when the button is pressed |

---

## Toggle Block

### What It Does

A toggle switch that stays in the on or off position. Unlike a regular button (which is only active while you're pressing it), a toggle remembers its state. The track maintains a 2:1 aspect ratio and the knob stays perfectly circular no matter how you resize the block.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown on the toggle | "Autopilot", "Lights" |
| **Joystick Index** | Which virtual controller (0-9) | 0 |
| **Button ID** | Which button on that controller (0-15) | 0 to 15 |
| **Toggle Mode** | How the button signal behaves | "momentary" (default) or "toggle" |
| **Toggle Color** | Track color when toggle is OFF | Any color |
| **Toggle Color Active** | Track color when toggle is ON | Green, blue, etc. |
| **Label Color** | Label text color | Any color |
| **Border Color** | Track outline color | Any color |
| **Font Size** | Size of the label text | 12 to 48 |

### Toggle Mode Explained

Both modes keep the visual toggle state persistent — the knob stays where you put it. The difference is in what signal gets sent to the game:

- **Momentary** (default): Each click sends a quick button press followed by a release (100ms apart). The visual state toggles and stays. Use this when the game expects a brief button tap to toggle something.
- **Toggle**: Sends button state 1 (pressed) when you turn it ON, and state 0 (released) when you turn it OFF. The signal matches the visual position. Use this when the game reads the button state directly.

### How It Works

- Tap once to turn ON
- Tap again to turn OFF
- The visual state shows whether it's currently on or off
- The track always keeps a 2:1 pill shape; the knob stays circular

### In Your Game

In your game's control settings, map the corresponding joystick button to the action you want. If your toggle uses Joystick 0, Button 5, map "Joystick 0 Button 5" to "Toggle Autopilot" in the game.

---

## Star Citizen Power Slider

### What It Does

A vertical slider with a progress fill bar that shows the current value visually. Great for power levels, shield strength, or energy management displays.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown above the slider | "Shields", "Engines", "Weapons" |
| **Joystick Index** | Which virtual controller (0-9) | 0 |
| **Slider ID** | Which axis on that controller (0-7) | 0 to 7 |
| **Track Color** | Color of the empty track | Dark gray |
| **Progress Color** | Color of the filled portion | Blue, green, etc. |

---

## Sequence Button Block

### What It Does

A sequence button sends a series of keyboard key presses while holding the Ctrl key. When you tap the button, OmniPanel-go holds Ctrl, taps each key in your sequence one by one, then releases Ctrl. This is designed for games that use **Ctrl+WASD** directional inputs instead of arrow keys.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown on the button | "Forward", "Left", "Strafe Right" |
| **Icon URL** | Path to an icon image (optional) | `helldivers2/Reinforce_Stratagem_Icon.svg` |
| **Layout** | How to display icon and text | `image-text`, `image-only`, `text-only` |
| **Text Position** | Where the label appears relative to the icon | `bottom`, `top`, `left`, `right` |
| **Sequence Code** | Comma-separated keys to press (use w, a, s, d) | `d,d,w,s,a` for right-right-up-down-left |
| **Key Delay** | Pause between key presses in milliseconds | 50 to 1000 (default: 100) |
| **Keyboard Index** | Which virtual keyboard (0-9) | 0 for first keyboard |
| **Font Size** | Size of the label text | 1 to 48 |
| **Button Color** | Button fill color | Any color |
| **Button Color Active** | Color when button is pressed/executing | Any color |
| **Label Color** | Label text color | Any color |
| **Border Color** | Button outline color | Any color |

### How Sequence Codes Work

The sequence code uses simple letters for WASD keys:

| Letter | Direction |
|--------|-----------|
| `w` | Up / Forward |
| `a` | Left |
| `s` | Down / Back |
| `d` | Right |

**Example:** `d,d,w,s,a` means: hold Ctrl, tap D, tap D, tap W, tap S, tap A, release Ctrl.

### How to Use It

1. Drag a Sequence Button block onto your panel
2. Click the gear icon
3. Set the label (e.g., "Call Reinforcements")
4. Set the Sequence Code (e.g., `d,d,w,s,a`)
5. Optionally set an icon URL for a visual indicator
6. Adjust the Key Delay if your game needs slower or faster input (default 100ms works for most games)
7. Save the panel

### When to Use

- Games that use Ctrl+WASD for directional commands (e.g., Helldivers 2 stratagem inputs)
- Any situation where you need to send a specific key sequence with one tap
- Quick-access panels for complex keyboard macros

### Tips

- **Key Delay:** If the game misses inputs, increase the delay (try 150-200ms). If it feels too slow, decrease it (try 50-80ms).
- **Sequence length:** There's no hard limit, but very long sequences (10+ keys) may feel sluggish.
- **Icon + text:** Use the `image-text` layout with an icon to make sequence buttons visually distinct from regular buttons.

---

## Star Citizen Emergency Button

### What It Does

A dramatic, red pulsing button for emergency actions. It has a glowing animation to draw attention.

### Settings

| Setting | What It Does |
|---------|-------------|
| **Label** | Text shown on the button | "EJECT", "ABORT", "EMERGENCY" |
| **Joystick Index** | Which virtual controller (0-9) |
| **Button ID** | Which button on that controller (0-15) |
| **Red Color Variants** | Different shades of red for the pulsing effect |

### When to Use

For actions you rarely use but need to access quickly in emergencies:
- Eject seat
- Emergency brake
- Self-destruct (if your game has one)
- Panic button

---

## Tips for Buttons and Sliders

### Keyboard Shortcuts

Any button or slider can be triggered by a keyboard key or combination. Set the **Keyboard Key** setting in the block's gear menu:

- **Single key:** `"a"`, `"b"`, `"f1"`, `"space"`, `"enter"`
- **With modifiers:** `"ctrl+a"`, `"ctrl+shift+a"`, `"alt+f4"`
- **Supported modifiers:** `ctrl`, `shift`, `alt`, `meta` (Windows/Command key)

When a button has a keyboard key set, clicking it sends the keyboard event to the host system instead of a joystick button press. This lets you build on-screen keyboards or quick-access panels that type into any application.

**Physical keyboard triggers:** Pressing the assigned key combination on your physical keyboard also triggers the button/slider — the panel button visually activates and the keyboard event is sent to the host.

### Color Coding

Use colors to group related controls:
- **Green** — navigation and movement
- **Red** — weapons and combat
- **Blue** — systems and utilities
- **Yellow** — warnings and alerts

### Size Matters

- Make frequently used buttons **larger** so they're easier to tap
- Make rarely used buttons **smaller** to save space
- Make sliders **wider/taller** for more precise control

### Label Best Practices

- Keep labels **short** — 1-2 words maximum
- Use **abbreviations** if needed (e.g., "NAV" instead of "Navigation")
- Use **icons or symbols** for universal actions (arrows for direction, etc.)

### Avoiding Input Conflicts

Each button and slider needs a unique joystick input. The editor shows which inputs are already in use. Before adding a new control:

1. Check the input allocation view
2. Pick a Joystick Index and Button/Slider ID that isn't already used
3. If all inputs on one joystick are taken, increase the Joystick Index to use another virtual controller

## What's Next?

Ready to add more advanced controls? Learn about [Touch Pads and Mousepads](07-touch-pads-and-mousepads.md) for analog joystick and mouse input.
