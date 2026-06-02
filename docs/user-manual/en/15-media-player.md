# Chapter 15: Media Player

Want to see what music is playing and control it from your panel? This chapter covers the Media Player block and how to connect it to your music apps on Linux and Windows.

## What Is the Media Player Block?

The Media Player block shows information about the currently playing media (music, video, podcast) and lets you control playback with buttons. In automatic mode, it works with Linux players through MPRIS and Windows players through SMTC (System Media Transport Controls) — including Spotify, VLC, browser media sessions, and many more.

### What It Shows

- **Cover art** — album artwork from the current track
- **Title** — song or video name
- **Artist** — performer or channel name
- **Progress bar** — how far through the track you are
- **Volume** — current player volume (reflected in the volume buttons)
- **Control buttons** — previous, play/pause, next, volume down, volume up

---

## Control Modes

The Media Player block has two control modes. Choose the one that fits your setup:

### MPRIS Mode (Automatic Media Mode)

This is the automatic mode. OmniPanel-go automatically connects to the platform media system:
- **Linux:** MPRIS over D-Bus
- **Windows:** SMTC (System Media Transport Controls)

**What you get:**
- Real-time updates when the song changes
- Cover art from the media player
- Play/pause/next/previous buttons that control the actual app
- Progress bar that moves as the track plays

**Requirements:**
- Linux desktop (KDE Plasma, GNOME, etc.) with an MPRIS-compatible player, or Windows 10/11 with an app that exposes media controls
- Media player integration enabled in `config.json` (see below)

### Keyboard Mode (Any Platform)

This mode sends keyboard media keys (like the Play/Pause button on your keyboard) to control media. It works on any operating system but requires the media app to respond to keyboard shortcuts.

**What you get:**
- Buttons that simulate keyboard media keys
- Works on Windows, macOS, and Linux
- No automatic data updates — you set title/artist/cover manually

---

## Setting Up Automatic Media Mode

### Step 1: Enable Media Player Integration in Config

Open `config.json` and add or update the `media_player` section:

```json
{
  "media_player": {
    "enabled": true,
    "poll_interval": 1000
  }
}
```

- `enabled`: Set to `true` to turn on media player monitoring
- `poll_interval`: How often to check for updates, in milliseconds (1000 = 1 second). Minimum is 500ms.

Restart OmniPanel-go after changing this setting.

### Step 2: Start a Media Player App

Open a media app that supports platform media controls:
- **Spotify** (desktop app)
- **VLC** (any platform)
- **Firefox**, **Chrome**, or **Edge** (with media playing)
- **Rhythmbox**, **Audacious**, **Clementine**, and many others

### Step 3: Add the Media Player Block

1. Open the Editor
2. Find the **Media** category in the block library (🎵 icon)
3. Drag the **Media Player** block onto your workspace
4. Click the gear icon to open settings
5. Set **Control Mode** to `mediacontrol`
6. Toggle on the features you want (cover art, title, artist, progress bar)
7. Save the panel

### Step 4: Test It

Open your panel in a browser. When you play music in your media player, the block should automatically show the title, artist, and cover art. The control buttons should work to play, pause, skip, and adjust volume.

> **Note for distributed setups (serve + connect mode):** Automatic media mode works the same way — the media player block communicates with the host agent through the relay server. Enable `media_player` in the host agent's `config.json` (not the server config). The host agent must be connected to the relay server.

---

## Switching Between Media Players

If you have multiple media players running at the same time (for example, Spotify and VLC), OmniPanel-go shows small tabs overlaid on the cover art area. These tabs let you choose which player the block displays and controls.

**How it works:**
- When only one player is running, no tabs are shown — the block just shows that player
- When two or more players are running, tabs appear at the top of the cover art
- Each tab shows the player's name (e.g., "Spotify", "VLC media player")
- Click a tab to switch to that player — the cover art, title, artist, and progress bar will update immediately
- The active tab is highlighted so you can see which player is currently selected
- If you close a player that was selected, OmniPanel-go automatically switches to another available player

**Example:** You have Spotify playing music and VLC playing a video. The tabs show "Spotify" and "VLC media player". Click "VLC media player" to see and control the video instead. The control buttons (play, pause, next, etc.) will now affect VLC instead of Spotify.

---

## Media Player Block Settings

| Setting | What It Does | Example Values |
|---------|-------------|----------------|
| **Control Mode** | How the block gets data and sends commands | `mediacontrol` (auto), `keyboard` (manual) |
| **Show Cover** | Show or hide the cover art area | On / Off |
| **Cover Image** | Manual image URL (keyboard mode only) | `https://...` or leave empty |
| **Show Title** | Show or hide the track title | On / Off |
| **Title** | Manual title text (keyboard mode only) | "My Song" |
| **Show Artist** | Show or hide the artist name | On / Off |
| **Artist** | Manual artist text (keyboard mode only) | "Artist Name" |
| **Show Progress** | Show or hide the progress bar | On / Off |
| **Progress Value** | Manual progress 0-100% (keyboard mode only) | 0 to 100 |
| **Font Size** | Text size | 8 to 24 |
| **Background Color** | Block background | Any color |
| **Button Color** | Control button color | Any color |
| **Button Active Color** | Button hover/active color | Any color |
| **Text Color** | Title and artist text color | Any color |
| **Border Color** | Block border color | Any color |
| **Border Radius** | Corner roundness (0-100) | 0 (sharp) to 100 (round) |

---

## Troubleshooting

### No cover art showing

Some browsers (Chrome, Firefox) store cover art in temporary files that the block can access through a proxy. If cover art isn't showing:
1. Make sure MPRIS is enabled in `config.json`
2. Check that your media player is actually playing something
3. Try a different media player — some don't expose cover art via MPRIS
4. **In serve/connect mode:** Make sure the host agent is connected to the relay server. Cover art is sent through the relay server, so a disconnected host means no cover art.
5. **With authentication enabled:** The cover art endpoint requires the auth token. If you see a 401 error, check that your token is valid and the browser has stored it (look for the login page redirect).

### Buttons don't control my music

In automatic media mode (`Control Mode = mediacontrol`):
1. Make sure your media player is running and playing
2. Check the server logs for MPRIS errors
3. Some players need to be "active" (have a window open) to accept commands
4. **In serve/connect mode:** A 500 error on button clicks usually means the host agent is disconnected or MPRIS is not enabled on the host. Check the host agent logs for "MPRIS watcher failed to start" messages.

In Keyboard mode:
1. Make sure the media app responds to keyboard media keys
2. Check that the keyboard keys are mapped correctly in the block settings

### "No media players connected"

This means OmniPanel-go can't find active media sessions on your system:
1. Make sure `media_player.enabled` is `true` in `config.json`
2. Restart OmniPanel-go after enabling media player integration
3. Start a media player (Spotify, VLC, etc.)
4. On KDE Plasma, make sure the "Media Player" widget can see your player
5. **In serve/connect mode:** This error appears when the relay server can't reach the host agent. Verify the host agent is running and connected (check the start page for the host IP indicator).

---

## Supported Media Players

Any app that exposes system media controls will work. On Linux this means MPRIS2 apps; on Windows this means SMTC-enabled apps. Common ones include:

| Player | Cover Art | Controls | Notes |
|--------|-----------|----------|-------|
| Spotify (desktop) | Yes | Full | Best experience |
| VLC | Yes | Full | Works great |
| Firefox | Yes | Full | Via Plasma Browser Integration |
| Chrome/Chromium | Partial | Full | Streaming services show art; YouTube does not |
| Rhythmbox | Yes | Full | GNOME default |
| Audacious | Yes | Full | Lightweight player |
| Clementine | Yes | Full | Feature-rich player |

> **Note about YouTube on Chrome/Edge:** YouTube videos don't show cover art because:
> - The media information comes from the browser's MediaSession API, which doesn't provide artwork for YouTube
> - Windows SMTC (System Media Transport Controls) doesn't receive thumbnails from YouTube in Chrome/Edge
> - You'll see the music note placeholder instead of a video thumbnail
>
> This is a limitation of how the browser and YouTube share media information. Other streaming services (Spotify, Apple Music, etc.) work within browsers because they provide proper MediaSession artwork.

---

[← Back: Tips and Troubleshooting](14-tips-and-troubleshooting.md) · [Next: RSS Feeds →](16-rss-feeds.md)
