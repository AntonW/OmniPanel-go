# Chapter 15: The Start Page

## What This Page Does

The Start Page (served at `/`) is the central hub of OmniPanel-go. It combines what used to be a separate host dashboard into a single page that:
- Lists all available panels with clickable cards
- Provides a link to the Panel Editor
- Displays the connection URL for clients
- Controls fullscreen mode on connected clients
- Shows a live log of WebSocket events (client connect/disconnect)
- Supports dark/light theme toggle

## HTML Layout (`static/index.html`)

```html
<main class="main">
    <section class="tools-section">
        <h2 class="section-title">Tools</h2>
        <div class="tools-grid">
            <a href="/editor" class="tool-card">
                <span class="tool-icon">🛠️</span>
                <span class="tool-name">Panel Editor</span>
                <span class="tool-desc">Design and edit panels</span>
            </a>
            <a href="/editor?new=true" class="tool-card">
                <span class="tool-icon">➕</span>
                <span class="tool-name">New Panel</span>
                <span class="tool-desc">Create a blank panel</span>
            </a>
        </div>
    </section>

    <section class="host-section">
        <h2 class="section-title">Host Controls</h2>
        <div class="host-controls">
            <div class="control-row">
                <label>Connection string:</label>
                <span id="url-display">loading...</span>
            </div>
            <div class="control-row">
                <button id="enter-fullscreen">Enter fullscreen</button>
                <button id="exit-fullscreen">Exit fullscreen</button>
            </div>
        </div>
        <div id="log-container">
            <div class="log-entry">Waiting for connection...</div>
        </div>
    </section>

    <section class="panels-section">
        <h2 class="section-title">Available Panels</h2>
        <div id="panels-grid" class="panels-grid">
            <div class="loading">Loading panels...</div>
        </div>
    </section>
</main>
```

The page is organized into three sections: Tools, Host Controls, and Available Panels.

The `<head>` section includes a favicon link for browser tab icon discovery:

```html
<link rel="icon" href="/omnipanel-go-logo.svg" type="image/svg+xml">
```

> **Concept: Favicon via `<link>` tag**
> All HTML pages (`static/index.html`, `static/client/index.html`, `static/editor/editor.html`, `static/login.html`) include this `<link>` tag so browsers can discover the SVG favicon directly. Additionally, `internal/relay/server.go` registers a `/favicon.ico` route that redirects to `/omnipanel-go-logo.svg` as a fallback for browsers that request the legacy `.ico` path.

## CSS (`static/index.css`)

The start page supports both light and dark themes via CSS custom properties and a `data-theme` attribute:

```css
:root {
    --bg-primary: #f8f9fa;
    --bg-card: #ffffff;
    --text-primary: #1a1a2e;
    --accent: #0cf574;
}

[data-theme="dark"] {
    --bg-primary: #0d1117;
    --bg-card: #161b22;
    --text-primary: #e6edf3;
}
```

Key design features:
- **Theme toggle**: A button in the header switches between light and dark modes, with the preference saved to `localStorage`
- **Card grid**: Tools and panels use `grid-template-columns: repeat(auto-fill, minmax(...))` for responsive layouts
- **Host controls**: Connection string displayed in monospace font, buttons with hover glow effects
- **Log container**: Monospace font, max-height with scroll, colored log type indicators (error=red, warning=yellow, info=blue)
- **Responsive**: On screens < 600px, grids collapse and host controls stack vertically

## JavaScript (inline in `static/index.html`)

### Theme Toggle

```javascript
const savedTheme = localStorage.getItem('theme') || 'dark';
html.setAttribute('data-theme', savedTheme);

themeToggle.addEventListener('click', () => {
    const current = html.getAttribute('data-theme');
    const next = current === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
});
```

> **Concept: `localStorage`**
> A browser API for persisting key-value data across page reloads. `localStorage.getItem('theme')` returns the saved preference, defaulting to `'dark'` if nothing is stored.

### Loading Panels

```javascript
async function loadPanels() {
    const grid = document.getElementById('panels-grid');
    const res = await fetch('/api/panels');
    const data = await res.json();

    grid.innerHTML = '';
    data.allPanels.forEach(panel => {
        const card = document.createElement('a');
        card.href = `/panel?name=${encodeURIComponent(panel)}`;
        card.target = '_blank';
        card.className = 'panel-card';
        card.innerHTML = `
            <span class="panel-icon">📟</span>
            <span class="panel-name">${panel}</span>
        `;
        grid.appendChild(card);
    });
}
```

Fetches the panel list from the REST API and creates clickable cards. Each card opens the panel in a new tab.

### Displaying the Connection URL

```javascript
function getLocalIP() {
    const a = document.createElement('a');
    a.href = window.location.href;
    return a.hostname;
}

async function loadConfig() {
    const res = await fetch('/api/config');
    const config = await res.json();
    const ip = getLocalIP();
    document.getElementById('url-display').innerText = `http://${ip}:${config.port}`;
}
```

> **Concept: URL parsing with `<a>` element**
> Creating an `<a>` element and setting its `href` lets the browser parse the URL. `a.hostname` extracts just the hostname part. This is a clever trick to avoid manual string parsing.

### WebSocket Connection and Logging

```javascript
let ws;
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws = new WebSocket(`${protocol}//${window.location.host}/ws`);

    ws.onopen = () => {
        addLogEntry(new Date().toLocaleTimeString(), 'System', 'Host UI connected');
    };

    ws.onmessage = (event) => {
        try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'log-event') {
                addLogEntry(msg.timestamp, 'System', msg.data);
            }
        } catch (e) {
            // ignore non-JSON messages
        }
    };

    ws.onclose = () => {
        addLogEntry(new Date().toLocaleTimeString(), 'System', 'Host UI disconnected, reconnecting...');
        setTimeout(connectWebSocket, 3000);
    };
}
```

The WebSocket connection serves two purposes:
1. **Log events**: When clients connect or disconnect, the server broadcasts `log-event` messages with the client IP and timestamp
2. **Reconnection**: If the connection drops, it automatically reconnects after 3 seconds

### Log Entry Management

```javascript
function addLogEntry(timestamp, type, data) {
    const container = document.getElementById('log-container');
    const entry = document.createElement('div');
    entry.className = 'log-entry';
    entry.innerHTML = `
        <span class="log-timestamp">[${timestamp}]</span>
        <span class="log-type">${type}:</span>
        <span class="log-data">${data}</span>
    `;
    container.prepend(entry);
    while (container.children.length > 500) {
        container.lastElementChild.remove();
    }
}
```

> **Concept: `container.prepend()`**
> `prepend()` inserts a new element at the beginning of the container, so the newest log entries appear at the top. The `while` loop caps the log at 500 entries to prevent memory issues.

### Fullscreen Control

```javascript
document.getElementById('enter-fullscreen').addEventListener('click', () => {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'enter-fullscreen' }));
    }
});

document.getElementById('exit-fullscreen').addEventListener('click', () => {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'exit-fullscreen' }));
    }
    if (document.fullscreenElement) {
        document.exitFullscreen();
    }
});
```

Sends fullscreen commands via WebSocket to all connected clients. The exit button also exits fullscreen on the host machine itself.

### Startup

```javascript
document.addEventListener('DOMContentLoaded', () => {
    loadPanels();
    loadConfig();
    connectWebSocket();
});
```

On page load: load panel list, fetch config for the connection URL, and connect the WebSocket.

## Key Takeaways

- The Start Page combines panel list, editor links, host controls, and live logging into one page
- Theme preference is persisted in `localStorage` and applied on page load
- `<a>` element URL parsing is a clever browser trick
- `container.prepend()` + capped length creates a bounded log
- `window.open('/editor?new=true')` passes state via URL parameters
- The same WebSocket endpoint serves all UIs with different message handling
- The server broadcasts `log-event` messages with client IP on connect/disconnect
- Timestamps use 24-hour format with date: `2006-01-02 15:04:05`

[← Back: Chapter 14](14-editor-ui.md) · [Next: Chapter 16 →](16-data-flow.md)
