# Chapter 7: Touch Pads and Mousepads

Touch Pads and Mousepads let you control analog inputs by dragging your finger across a surface — just like a joystick or laptop trackpad. This chapter covers both block types and how to configure them.

## Touch Pad Block

### What It Does

A Touch Pad is a virtual joystick. You drag your finger around a circular area, and OmniPanel-go sends axis values to your game based on where your finger is. Think of it like the analog stick on a game controller.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown above the touchpad | "Camera", "Flight Stick", "Turret" |
| **Joystick Index** | Which virtual controller (0-9) | 0 for first controller |
| **Axis ID** | Which pair of axes to use (0-3) | 0 uses X and Y axes |
| **Color** | Color of the touchpad ring and indicator | Any color |
| **Travel Distance** | How far the indicator moves from center (in pixels) | 50 to 150 |
| **Safe Zone** | Dead zone in the center where small movements are ignored | 0 to 30 |

### How It Works

1. Touch the pad and drag your finger
2. A dot moves in the direction you drag
3. The distance from center determines the axis value
4. Release your finger and the dot snaps back to center

### Axis Mapping

Each Touch Pad controls **two axes** at once:

- **Axis ID 0** — uses Axis 0 (horizontal) and Axis 1 (vertical)
- **Axis ID 1** — uses Axis 2 (horizontal) and Axis 3 (vertical)
- **Axis ID 2** — uses Axis 4 (horizontal) and Axis 5 (vertical)
- **Axis ID 3** — uses Axis 6 (horizontal) and Axis 7 (vertical)

### Value Range

- **Center position** — both axes at 127 (neutral)
- **Full left** — horizontal axis at 0
- **Full right** — horizontal axis at 255
- **Full up** — vertical axis at 0
- **Full down** — vertical axis at 255

### Travel Distance

This setting controls how far the indicator dot can move from the center:

- **Smaller values (50-80)** — the dot doesn't move far, good for precise small adjustments
- **Medium values (80-120)** — balanced for general use
- **Larger values (120-150)** — the dot moves a lot, good for full-range control

### Safe Zone (Dead Zone)

The safe zone is a small area in the center where tiny finger movements are ignored. This prevents accidental inputs from shaky hands.

- **0** — no dead zone, every movement is registered
- **10-20** — small dead zone, recommended for most uses
- **20-30** — large dead zone, only intentional movements register

### When to Use a Touch Pad

- Camera control in flight sims
- Turret aiming in vehicle combat games
- Character movement in third-person games
- Any situation where you need smooth analog control in two directions

---

## Star Citizen Themed Touch Pad

### What It Does

Same functionality as the default Touch Pad, but with a sci-fi visual style — glowing rings and angular design elements.

### Settings

Same as the default Touch Pad, with additional visual customization options for the sci-fi aesthetic.

---

## Mousepad Block

### What It Does

A Mousepad simulates a laptop trackpad. Drag your finger to move the mouse cursor on your PC. This is different from a Touch Pad — instead of sending joystick axis values, it sends actual mouse movement.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Label** | Text shown above the mousepad | "Mouse", "Cursor Control" |
| **Mousepad Index** | Which virtual mouse to use (usually 0) | 0 |
| **Sensitivity** | How fast the cursor moves relative to your finger | 1 to 20 |
| **Color** | Color of the mousepad surface | Any color |

### How It Works

1. Touch the mousepad and drag your finger
2. The mouse cursor on your PC moves in the same direction
3. The speed depends on the sensitivity setting
4. Release your finger and the cursor stays where it is (doesn't snap back)

### Sensitivity

- **Low values (1-5)** — cursor moves slowly, good for precise aiming
- **Medium values (5-10)** — balanced for general use
- **High values (10-20)** — cursor moves quickly, good for fast navigation

### Mouse Buttons

The Mousepad block itself only handles movement. For mouse clicks, you'll need to use Command Blocks or Button Blocks configured for mouse input.

### When to Use a Mousepad

- Games that require mouse aiming (FPS games)
- Desktop control from your tablet
- Menu navigation in games that don't support controllers
- Any situation where you need to move the actual mouse cursor

---

## Touch Pad vs Mousepad: Which One?

| Feature | Touch Pad | Mousepad |
|---------|-----------|----------|
| **Input type** | Joystick axis values | Mouse movement |
| **Behavior** | Springs back to center | Cursor stays where you leave it |
| **Best for** | Analog control (throttle, camera) | Pointing and clicking |
| **Game support** | Games with joystick support | Any game that uses a mouse |
| **Precision** | Smooth analog range | Pixel-accurate cursor control |

### Quick Decision Guide

- Need to **steer or aim smoothly**? Use a **Touch Pad**
- Need to **move the mouse cursor**? Use a **Mousepad**
- Playing a **flight sim**? Use a **Touch Pad** for camera control
- Playing an **FPS game**? Use a **Mousepad** for aiming

---

## 3-Finger Swipe: Switch Between Panels

### What It Does

You can switch between panels without going back to the start page by swiping with three fingers on your touch screen.

### How to Use It

1. Place **three fingers** on the screen at the same time
2. Swipe **left** (right to left) to go to the **next panel**
3. Swipe **right** (left to right) to go to the **previous panel**
4. Lift your fingers — the panel switches automatically

### How It Works

- **Panel order**: Panels are sorted alphabetically by name
- **Wrap-around**: Swiping left on the last panel takes you to the first panel, and vice versa
- **Visual feedback**: A semi-transparent overlay appears showing the panel name as you swipe
- **Minimum swipe**: You need to swipe at least 100 pixels (about 2-3 cm on most tablets)
- **Cooldown**: There's a short delay (half a second) between switches to prevent accidental rapid switching

### Tips

- Make sure you're using **exactly three fingers** — one or two fingers won't work
- Swipe **horizontally** — vertical swipes won't trigger panel switching
- The gesture works **anywhere on the screen, including on Touch Pad, Mousepad, and Push-to-Talk blocks** — when you place three fingers down, the swipe gesture takes priority and block interactions are temporarily suppressed
- If the panel doesn't switch, try swiping a bit faster or with more distance
- Pinch-to-zoom is disabled on the client, so you don't need to worry about accidentally zooming

### When to Use 3-Finger Swipe

- You have multiple panels for different games or scenarios
- You want to switch panels quickly during gameplay
- You don't want to minimize your game to access the start page
- You're using a touch device (tablet, touchscreen laptop, or phone)

---

## Tips for Touch Pads and Mousepads

### Size Recommendations

- Make Touch Pads and Mousepads **at least 3x3 grid units** for comfortable use
- Larger pads (4x4 or 5x5) give you more room for precise control
- Don't make them too large — you'll run out of panel space

### Placement Tips

- Place Touch Pads and Mousepads in areas where your thumb or finger naturally rests
- Keep them away from the edges of your panel so you don't accidentally drag off
- Group related controls nearby (e.g., fire button next to a camera Touch Pad)

### Sensitivity Tuning

Start with medium settings and adjust based on feel:

1. Set sensitivity to a middle value
2. Test in your game
3. If it feels too slow, increase sensitivity
4. If it feels too jumpy, decrease sensitivity
5. Repeat until it feels right

### Using Multiple Touch Pads

You can have multiple Touch Pads on the same panel, each controlling different axes:

- Touch Pad 1: Camera control (Axis ID 0)
- Touch Pad 2: Throttle and rudder (Axis ID 1)
- Touch Pad 3: Turret aiming (Axis ID 2)

Just make sure each Touch Pad uses a different Axis ID to avoid conflicts.

## What's Next?

Want to run programs or send web requests from your panel? Learn about [Command Blocks](08-command-blocks.md).
