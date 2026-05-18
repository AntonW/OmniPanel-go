# Chapter 3: The Start Page

The Start Page is your dashboard — the first thing you see when you open OmniPanel-go. It's your control center for managing panels, monitoring connections, and adjusting settings.

## How to Access the Start Page

Open your browser and go to:
```
http://YOUR-PC-IP:3000
```

## What You'll See

### Panel List

The main area shows cards for each saved panel. Each card displays:

- **Panel name** — the name you gave it when saving
- **Click the card** — opens that panel on your device

If you haven't created any panels yet, this area will be empty.

### Quick Links

At the top of the page, you'll find shortcuts:

- **Panel Editor** — opens the visual editor where you design panels
- **New Panel** — opens the editor with a blank canvas

### Connection URL

This section shows the web address you should enter on your tablet or phone to access OmniPanel-go. It looks like:

```
http://192.168.1.100:3000
```

You can copy this address and share it with anyone on your network who needs to connect.

### Fullscreen Controls

Buttons to toggle fullscreen mode on all connected devices:

- **Enter Fullscreen** — makes all connected tablets go fullscreen (hides browser bars)
- **Exit Fullscreen** — returns all devices to normal browser view

> **Tip:** Fullscreen mode is great for immersion — your panel will look like a dedicated app rather than a webpage.

### Connection Log

At the bottom of the page, you'll see a live log showing:

- **When devices connect** — with a timestamp
- **When devices disconnect** — with a timestamp
- **The IP address** of each connecting device

This is useful for:

- Checking if your tablet successfully connected
- Seeing how many devices are currently connected
- Troubleshooting connection issues

Example log entry:
```
[14:32:15] Device connected from 192.168.1.50
[14:35:42] Device disconnected from 192.168.1.50
```

### Theme Toggle

In the top-right corner, you'll find a button to switch between:

- **Dark theme** — darker colors, easier on the eyes in dim rooms
- **Light theme** — brighter colors, better for well-lit environments

Your preference is saved automatically, so OmniPanel-go remembers your choice next time you open it.

## Using the Start Page

### Opening a Panel on Your Tablet

1. On your tablet, open the Start Page
2. Tap the panel card you want to use
3. The panel opens with all your buttons, sliders, and controls

### Opening a Panel on Your PC

You can also open panels on your PC browser for testing:

1. Click a panel card on the Start Page
2. The panel opens in your browser
3. You can click buttons with your mouse to test them

### Managing Multiple Devices

You can have different panels open on different devices at the same time:

- Tablet 1: Flight controls panel
- Tablet 2: Navigation and systems panel
- Phone: Quick access panel for emergency functions

Each device opens its own panel independently.

## What's Next?

Ready to build your first panel? Head to [Chapter 4: Creating Your First Panel](04-creating-your-first-panel.md) to learn how to use the visual editor.
