# Chapter 4: Creating Your First Panel

This chapter walks you through creating your first control panel from scratch. By the end, you'll have a working panel with buttons and sliders that you can open on your tablet.

## Opening the Editor

From the Start Page, click **Panel Editor** or **New Panel**. You'll see the editor interface with:

- **Top menu bar** — New, Load, Save, Duplicate, Delete, Undo, Redo, Export, Import
- **Left sidebar** — the block library organized by category with search
- **Center workspace** — where you arrange your blocks (with zoom/pan controls)
- **Right sidebar** — tabs for Properties, Layers, Bindings, and Speech

## The Workspace

The workspace is a canvas where you place and arrange blocks. It uses a **12x12 grid system** — imagine the workspace divided into 12 columns and 12 rows. This helps you align blocks neatly.

### Grid Snapping

When you drag a block, it automatically snaps to the grid. You can toggle the grid visibility, change grid size (8×8, 12×12, 16×16, 24×24), and toggle snap behavior using the toolbar above the workspace.

### Zoom and Pan

- **Zoom**: Hold `Ctrl` and scroll the mouse wheel, or use the +/− buttons in the toolbar
- **Pan**: Middle-click and drag, or use the Fit button to reset view
- **Auto-fit**: The panel automatically centers and scales to fit your screen when you open the editor
- **Zoom level** is shown in the toolbar (e.g., 100%)

### Aspect Ratio

In the right sidebar under Workspace, you can choose an aspect ratio preset:

- **9:16** — Slim portrait phones
- **16:9** — widescreen (most tablets and phones)
- **4:3** — older tablets, some iPads
- And more...

This preview helps you see how your panel will look on different devices.

## Step 1: Add Your First Button

1. In the left sidebar, find the **Input Controls** category
2. Find the **Button** block (you can use the search bar to filter)
3. Click and drag it onto the workspace
4. Release it where you want the button to appear
5. The block snaps to the grid

## Step 2: Configure the Button

Every block has settings you can customize:

1. Click the block to select it — the **Properties** tab in the right sidebar automatically opens with the block's settings
2. Settings are grouped by category:
   - **Identity** — Label text
   - **Appearance** — Colors, font size, border radius
   - **Input Mapping** — Joystick index, button ID, keyboard shortcuts
   - **Command** — Command type, shell command, HTTP method/URL/body (for command blocks)
   - **Speech** — Voice trigger phrase and aliases (if applicable)
3. Change the label to "Gear"
4. Leave the joystick settings as they are for now
5. Click outside the block to deselect (the Properties panel clears)

## Step 3: Add a Slider

1. Drag a **Slider** block from the **Sliders** category onto the workspace
2. Place it below your button
3. Select it and configure in the Properties tab:
   - **Label** — change to "Throttle"
   - **Joystick Index** — leave at 0
   - **Slider ID** — which axis to control (0 to 7)
   - **Colors** — customize the slider appearance

## Step 4: Resize and Position Blocks

### Moving Blocks

- Grab the **cross icon** (move handle) on any block
- Drag it to a new position
- Release to snap it to the grid
- **Alignment guides** (cyan dashed lines) appear when edges align with other blocks

### Resizing Blocks

- Grab the **bottom-right corner** handle of any block
- Drag to make it bigger or smaller
- The block snaps to grid sizes

### Tips for Layout

- Leave some space between blocks so you don't accidentally tap the wrong one
- Group related controls together (all flight controls in one area, weapons in another)
- Make frequently used buttons larger for easier tapping

## Step 5: Save Your Panel

1. Click the **Save** button in the top menu bar
2. If this is a new panel, a dialog asks for a panel name
3. Enter a name like "Flight Controls"
4. The panel is saved — you'll see a confirmation toast notification

Your panel is now saved and ready to use.

## Step 6: Open Your Panel

### On Your PC

1. Go back to the Start Page
2. You'll see your "Flight Controls" panel card
3. Click it to open the panel

### On Your Tablet

1. Open the Start Page on your tablet
2. Tap the "Flight Controls" panel card
3. Your panel opens with the button and slider

## Testing Your Panel

- **Tap the button** — it should flash briefly, confirming it sent input
- **Drag the slider** — move it up and down to see it respond
- **Check your game** — if a game is running and listening for joystick input, it should receive these inputs

> **Note:** If no game is running, the inputs are still being sent — there's just nothing to receive them. You can verify inputs are working in [Chapter 12: Virtual Joysticks](12-virtual-joysticks.md).

## Loading a Saved Panel

To edit a panel you've already created:

1. Open the Editor
2. Click the **Load** button in the top menu bar
3. A dialog shows all your saved panels
4. Click the panel you want to edit
5. The panel loads into the workspace

## Undo and Redo

The editor tracks up to 100 changes:

- **Undo**: `Ctrl+Z` or the Undo button in the menu bar
- **Redo**: `Ctrl+Shift+Z` or `Ctrl+Y`, or the Redo button
- Works for: adding blocks, moving, resizing, changing settings, duplicating, deleting

## Deleting Blocks

- Select a block and press the **Delete** key on your keyboard
- Or click the **trash icon** 🗑 in the top-right corner of the block
- You can also click the **gear icon** ⚙ in the top-left to open the Properties panel

## Duplicating Blocks

- In the **Layers** tab (right sidebar), click the **duplicate icon** (⧉) next to any block
- A copy appears offset slightly from the original

## What's Next?

Now that you know the basics, let's dive deeper into [Understanding Blocks](05-understanding-blocks.md) — all the different types of controls you can add to your panels.
