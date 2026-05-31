# Chapter 9: Organizing Panels

As your panels grow more complex, you'll need ways to organize controls into logical groups. This chapter covers Paged Containers and Section Groups — two blocks that help you keep your panels tidy and easy to use.

## Paged Container Block

### What It Does

A Paged Container lets you create multi-page panels with tabs. Instead of cramming everything onto one screen, you can organize controls across multiple pages and switch between them with tabs.

Think of it like browser tabs — each page has its own set of blocks, and you click a tab to switch pages.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Pages** | List of page/tab names | "Flight", "Weapons", "Systems" |
| **Tab Position** | Where the tabs appear | Top, Bottom, Left, Right |
| **Current Page** | Which page is active (1-based index) | 1, 2, 3... |
| **Grid Size** | The internal grid dimensions for each page | 12x12 |
| **Background Image** | Optional image behind the page content | Any image URL |
| **Background Color** | Page background color | Any color |
| **Tab Colors** | Colors for active and inactive tabs | Any colors |

### Creating Pages

1. Drag a **Paged Container** onto your workspace
2. Click the gear icon
3. In the **Pages** setting, enter your page names:
   - Click "Add Page" or type names separated by commas
   - Example: "Flight, Weapons, Systems, Navigation"
4. Choose the tab position (Top is the default and most common)
5. Save the settings

### Using Pages

When you open the panel:

- Tabs appear at the position you chose
- Click a tab to switch to that page
- Each page has its own set of blocks
- Blocks on one page don't overlap with blocks on another page
- The active page is remembered — when you save and reload the panel, it opens on the same page you last viewed

> **Tip:** You can also set the **Current Page** number in the Properties panel to jump to a specific page without clicking tabs.

### Adding Blocks to Pages

1. In the Editor, select the Paged Container
2. Click the tab for the page you want to edit
3. Drag blocks into that page
4. Switch tabs to add blocks to other pages

### Tab Position Options

| Position | Best For |
|----------|----------|
| **Top** | Most common, like browser tabs |
| **Bottom** | Easy thumb access on tablets |
| **Left** | Vertical tab bar, good for many pages |
| **Right** | Vertical tab bar, alternative layout |

### Background Images

You can set a background image for the Paged Container:

1. Place your image in the `user/assets/` folder
2. In the settings, enter the image path: `assets/your-image.png`
3. The image appears behind all blocks on the page

> **Tip:** Use subtle, dark images so your blocks remain visible and readable.
>
> **Browse your assets:** You can view all your uploaded assets by visiting `http://your-server:3000/assets/` in your browser. This shows a directory listing of everything in `user/assets/`, making it easy to find the right filename for your image path.

---

## Section Group Block

### What It Does

A Section Group is a labeled container that groups related blocks together. It draws a border around its contents with a title label, making it clear which controls belong together.

Think of it like a labeled box — everything inside the box is related to the label.

### Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Title** | The label shown at the top of the section | "Flight Controls", "Weapons" |
| **Border Color** | Color of the section border | Any color |
| **Background Color** | Color inside the section | Any color (often slightly different from panel background) |
| **Title Color** | Color of the title text | Any color |

### Creating a Section Group

1. Drag a **Section Group** onto your workspace
2. Click the gear icon
3. Set the title (e.g., "Flight Controls")
4. Customize colors
5. Save the settings

### Adding Blocks to a Section Group

1. In the Editor, drag blocks **inside** the Section Group border
2. The blocks become children of the section
3. Moving the Section Group moves all its blocks together
4. Resizing the Section Group gives more or less room for its blocks

### When to Use Section Groups

- **Group by function:** "Flight Controls," "Weapons," "Navigation"
- **Group by system:** "Engines," "Shields," "Life Support"
- **Group by priority:** "Primary Controls," "Secondary Controls"
- **Visual organization:** Make your panel look structured and professional

---

## Combining Paged Containers and Section Groups

You can use both together for maximum organization:

1. Create a Paged Container with pages: "Flight," "Combat," "Systems"
2. On the "Flight" page, add Section Groups: "Movement," "Camera," "Communications"
3. On the "Combat" page, add Section Groups: "Weapons," "Defense," "Targeting"
4. On the "Systems" page, add Section Groups: "Power," "Diagnostics," "Settings"

This creates a clean hierarchy:
- **Pages** separate major categories
- **Sections** group related controls within each category

---

## Design Tips for Organized Panels

### Plan Before Building

Before you start placing blocks, sketch out your panel layout:

1. What are the main categories of controls?
2. How many controls per category?
3. Which controls do you use most often?
4. Should frequently used controls be on the first page?

### Keep It Simple

- Don't create too many pages — 3 to 5 pages is usually enough
- Don't overcrowd sections — leave space between blocks
- Use consistent section sizes for a clean look

### Color Consistency

- Use the same border color for all Section Groups
- Use different background colors to distinguish sections
- Make tab colors match your overall panel theme

### Naming Conventions

Use clear, consistent names for pages and sections:

- Good: "Flight," "Combat," "Systems"
- Less ideal: "Page 1," "Page 2," "Page 3"
- Good: "Engine Controls," "Weapon Systems"
- Less ideal: "Group 1," "Group 2"

### Testing Your Layout

After organizing your panel:

1. Open it on your tablet
2. Try switching between pages
3. Make sure all blocks are visible and accessible
4. Check that you can tap buttons without accidentally hitting the wrong one
5. Adjust sizes and positions as needed

---

## Practical Examples

### Example 1: Flight Simulator Panel

**Pages:**
- "Flight" — throttle, flaps, landing gear, trim
- "Navigation" — autopilot, waypoints, compass
- "Systems" — fuel, engine status, electrical

**Section Groups on "Flight" page:**
- "Thrust" — throttle slider, afterburner button
- "Aerodynamics" — flaps slider, spoiler buttons
- "Landing" — gear toggle, brake button

### Example 2: Racing Game Panel

**Pages:**
- "Driving" — gear selector, boost, brake balance
- "Setup" — tire pressure, suspension, wing angle
- "Info" — lap times, speed, position

**Section Groups on "Driving" page:**
- "Transmission" — gear up/down buttons, clutch slider
- "Performance" — boost button, DRS toggle
- "Braking" — brake balance slider, brake bias buttons

### Example 3: Space Game Panel

**Pages:**
- "Flight" — navigation, speed, docking
- "Combat" — weapons, shields, countermeasures
- "Power" — energy distribution, reactor status

**Section Groups on "Combat" page:**
- "Weapons" — weapon group buttons, fire toggle
- "Defense" — shield controls, countermeasure launcher
- "Targeting" — target lock, scan, lock-on indicator

---

## Layer Tree

The layer tree (usually on the right side of the editor) shows the hierarchy of all blocks on your panel:

- Top-level blocks appear at the root
- Blocks inside Section Groups are indented under the group
- Pages in a Paged Container show their child blocks

Use the layer tree to:

- **Reorder blocks** — drag to change stacking order
- **Duplicate blocks** — click the duplicate icon
- **Select blocks** — click to select on the workspace
- **See hierarchy** — understand which blocks are inside which containers

## What's Next?

Want to display live data on your panel? Learn about [Live Data Display](10-live-data-display.md) — showing CPU usage, custom metrics, and more.
