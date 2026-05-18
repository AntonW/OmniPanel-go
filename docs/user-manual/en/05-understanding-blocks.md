# Chapter 5: Understanding Blocks

Blocks are the building blocks of your panels — literally. Every button, slider, touchpad, and display on your panel is a block. This chapter explains how blocks work and how to use them effectively.

## What Is a Block?

A block is a single UI element on your panel. Think of blocks like LEGO pieces — each one does something specific, and you combine them to build your control panel.

## Block Themes

Every block can be styled with one of four themes. Themes control the look and feel — fonts, colors, gradients, and special effects. You can set a **default theme for the entire panel** and **override it per block**.

### Available Themes

**Default** — Clean, neutral design that works with any game. Uses system fonts and simple shadows.

**Star Citizen** — Sci-fi styled with angled corners, cyan glowing accents, and the Rajdhani font. Perfect for space sims.

**KDE Breeze** — Flat, clean design based on the KDE Plasma Breeze Dark desktop theme. Solid colors, no glow effects, subtle hover changes, and the Breeze blue accent. Uses the Noto Sans font.

**Windows 11** — Fluent Design with flat surfaces, rounded corners, the Segoe UI font, and the Windows accent color.

### Setting the Panel Theme

In the workspace settings (bottom-right of the editor), you'll find a **Panel Theme** dropdown. This sets the default theme for all new blocks you add to the panel.

### Changing a Block's Theme

1. Select a block on the workspace
2. Open the **Properties** tab (right sidebar)
3. Find the **Theme** group at the top
4. Choose a different theme from the dropdown
5. The block updates immediately

You can mix and match themes within the same panel. For example, use Star Citizen for your flight controls and KDE Breeze for media controls.

## The Block Library

The block library is in the left sidebar of the Editor. It shows all available block types you can add to your panel, organized by category:

- **Input Controls** — Button, Toggle, Emergency Button
- **Sliders** — Slider, Slider Vertical, Power Slider
- **Pointing** — Mousepad, Touch Pad
- **Commands** — Command Block
- **Speech** — Speech Command, Push to Talk
- **Data Display** — Data Display
- **Containers** — Paged Container, Section Group

Each block type appears once — you choose its theme in the Properties panel after adding it.

## Adding Blocks to Your Panel

1. Find the block you want in the left sidebar
2. Click and drag it onto the workspace
3. Release to place it
4. The block snaps to the grid

## Configuring Blocks

Every block has a **gear icon** that opens its settings. Settings vary by block type, but common options include:

### Common Settings

- **Label** — Text displayed on or above the block
- **Colors** — Background, text, border, and accent colors
- **Font Size** — Size of the label text
- **Joystick Index** — Which virtual controller to use (0 to 9)

### Joystick Index and IDs

These settings tell OmniPanel-go which virtual controller input to use:

- **Joystick Index (0-9)** — Which virtual controller. If you have 4 joysticks, use 0, 1, 2, or 3
- **Button ID (0-15)** — Which button on that controller (each joystick has 16 buttons)
- **Slider/Axis ID (0-7)** — Which axis on that controller (each joystick has 8 axes)

> **Example:** A button with Joystick Index 0 and Button ID 3 controls the 4th button on the 1st virtual joystick.

## Moving and Resizing Blocks

### Moving

- Grab the **cross icon** (move handle) on the block
- Drag to a new position
- Release to snap to the grid

### Resizing

- Grab the **bottom-right corner** handle
- Drag to resize
- Snaps to grid increments

## Duplicating Blocks

Need multiple similar buttons? Instead of creating each one from scratch:

1. Find the block in the **layer tree** (usually on the right side of the editor)
2. Click the **duplicate icon** next to the block
3. A copy appears on the workspace
4. Configure the copy with different settings

## Deleting Blocks

To remove a block:

1. Select the block on the workspace
2. Press the **Delete** key on your keyboard
3. Or use the delete option in the layer tree

## The Grid System

The workspace uses a **12x12 percentage-based grid**:

- 12 columns across (horizontal)
- 12 rows down (vertical)
- Each block's position and size is measured in grid units

### Why a Grid?

- Keeps everything aligned and tidy
- Makes panels look professional
- Ensures blocks scale properly on different screen sizes
- Prevents overlapping controls

### Grid Tips

- Small blocks (1-2 grid units) — good for simple buttons
- Medium blocks (3-4 grid units) — good for labeled controls
- Large blocks (5+ grid units) — good for touchpads and displays

## Input Allocation View

The editor shows which joystick buttons and axes are already in use. This helps you avoid assigning the same input to multiple blocks.

Look for the **input allocations** section in the editor — it shows a list of used joystick inputs so you can pick unused ones for new blocks.

## Block Settings Deep Dive

When you click the gear icon on a block, you'll see settings specific to that block type. Here's what the most common settings mean:

### Colors

Most blocks let you customize:

- **Background color** — the main fill color
- **Text color** — the label text color
- **Border color** — the outline color
- **Accent color** — special effects (progress bars, active states, etc.)

Click any color swatch to open a color picker. You can:

- Choose from preset colors
- Enter a hex code (like `#FF0000` for red)
- Use the color picker to select any color

### Labels

The label is the text shown on or above the block. Keep labels short and clear:

- Good: "Gear", "Throttle", "Flaps"
- Less ideal: "Landing Gear Control Button", "Engine Throttle Slider"

## Saving Block Configurations

Block settings are saved as part of the panel. When you save the panel, all block configurations are preserved. You don't need to save individual blocks separately.

## What's Next?

Now that you understand blocks, let's look at the most common ones in detail: [Buttons and Sliders](06-buttons-and-sliders.md).
