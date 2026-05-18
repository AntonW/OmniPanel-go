# Chapter 13: The Panel UI (client)

## What This UI Does

The Panel UI is served at `/panel?name=<panel-name>` and is what end users see on their tablet, phone, or browser. It:
- Loads a specific panel via URL query parameter using the REST API
- Connects to the server via WebSocket for real-time input/output
- Sends touch/mouse input as joystick/mouse events
- Displays live system metrics
- Executes commands when buttons are pressed

## HTML Shell (`static/client/index.html`)

The HTML is minimal — just a skeleton with embedded CSS:

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Omnipanel</title>
    <script src="/client/client.js"></script>
    <style>
        /* CSS for fullscreen prompt, command toast, etc. */
    </style>
</head>
```

The body shows an error message by default. If JavaScript loads correctly and connects to the server, it replaces the entire body content with the panel.

> **Concept: `viewport` meta tag**
> `width=device-width, initial-scale=1.0` tells mobile browsers to render at the device's actual width, not zoomed out. Essential for touch interfaces.

## JavaScript Architecture (`static/client/client.js`)

### Global State

```javascript
let socket;
let reconnectInterval;
let commandHoldIntervals = {};
```

- `socket`: The WebSocket connection
- `reconnectInterval`: Timer for the watchdog that checks connection health
- `commandHoldIntervals`: Tracks hold-repeat timers for command buttons

### WebSocket Connection

```javascript
function connect() {
    if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
        return;
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

    socket.onopen = () => {
        console.log("Connected to OmniPanel-go Host");
    };

    socket.onmessage = (event) => {
        const msg = JSON.parse(event.data);

        if (msg.type === 'force-reload') {
            location.reload();
        }

        if (msg.type === 'load-panel') {
            loadPanel(msg.data);
        }

        if (msg.type === 'enter-fullscreen') {
            showFullscreenPopup();
        } else if (msg.type === 'exit-fullscreen') {
            if (document.fullscreenElement) {
                document.exitFullscreen();
            }
        }

        if (msg.type === 'command-result') {
            handleCommandResult(msg.data);
        }

        if (msg.type === 'data-update') {
            handleDataUpdate(msg.data);
        }
    };
}
```

The message router dispatches based on the `type` field:
- `force-reload` → reload the page (panel changed)
- `enter-fullscreen` / `exit-fullscreen` → fullscreen control
- `command-result` → show button feedback
- `data-update` → update metric displays

Note: The server no longer sends `load-panel` on WebSocket connect. Panels are loaded via the REST API using the `?name=` query parameter.

### Reconnection Watchdog

```javascript
function startWatchdog() {
    if (reconnectInterval) clearInterval(reconnectInterval);

    reconnectInterval = setInterval(() => {
        if (!socket || socket.readyState === WebSocket.CLOSED || socket.readyState === WebSocket.CLOSING) {
            console.warn("Watchdog detected closed connection. Reconnecting...");
            connect();
        }
    }, 3000);
}
```

Every 3 seconds, check if the WebSocket is still alive. If not, reconnect. This handles network drops and server restarts gracefully.

### Loading and Rendering Panels

```javascript
async function loadPanel(data) {
    const mainContainer = document.querySelector('body');
    mainContainer.innerHTML = "";

    for (const blockData of data) {
        if (blockData.backgroundColor) {
            mainContainer.style.setProperty('--workspace-bg', blockData.backgroundColor);
            continue;
        }
        await renderBlockRecursive(blockData, mainContainer);
    }

    enableInputs();
}
```

The panel data is an array of block definitions. The first item might be a background color setting; the rest are blocks. After all blocks are rendered, `enableInputs()` attaches event listeners.

### Recursive Block Rendering

```javascript
async function renderBlockRecursive(blockData, parentElement) {
    // 1. Fetch the block's HTML template from /blocks/...
    const fileName = blockData.path.split('blocks').pop().replace(/\\/g, '/');
    const url = `/blocks/${fileName}`;
    const response = await fetch(url);
    const rawHtml = await response.text();

    // 2. Parse the HTML and extract <settings> metadata
    const parser = new DOMParser();
    const doc = parser.parseFromString(rawHtml, 'text/html');
    const settingsTag = doc.querySelector('settings');

    const settingsMeta = {};
    if (settingsTag) {
        for (let attr of settingsTag.attributes) {
            const name = attr.name;
            const value = attr.value;
            if (name.startsWith('type-')) { /* ... */ }
            else if (name.startsWith('min-')) { /* ... */ }
            // ... extract type, min, max, options for each setting
        }
        settingsTag.remove();
    }
```

> **Concept: `DOMParser`**
> Parses an HTML string into a DOM document. This lets us inspect and manipulate the block's HTML before inserting it into the page.

> **Concept: `<settings>` custom element**
> Blocks define their configurable properties as attributes on a `<settings>` tag:
> ```html
> <settings label="My Button" button="0" type-button="select" options-button="0,1,2,3">
> ```
> The renderer extracts these into `settingsMeta` for the editor's settings modal.

```javascript
    // 3. Create the block wrapper
    const blockWrapper = document.createElement('div');
    blockWrapper.classList.add('loaded-block');
    blockWrapper.id = blockData.id;
    blockWrapper.settingsMeta = settingsMeta;

    Object.assign(blockWrapper.style, {
        position: 'absolute',
        left: blockData.left,
        top: blockData.top,
        width: blockData.width,
        height: blockData.height,
        zIndex: blockData.zIndex || 1
    });

    blockWrapper.settings = blockData.settings;
    blockWrapper.htmlTemplate = doc.head.innerHTML + doc.body.innerHTML;

    // 4. Render the template with settings substituted
    renderBlockFromTemplate(blockWrapper);

    // 5. Recursively render child blocks (for nested pages)
    if (blockData.children && blockData.children.length > 0) {
        for (const pageGroup of blockData.children) {
            const targetPage = contentArea.querySelector(
                `:scope > .page-wrapper[data-page-index="${pageGroup.pageIndex}"]`
            );
            if (targetPage) {
                for (const childBlock of pageGroup.blocks) {
                    await renderBlockRecursive(childBlock, targetPage);
                }
            }
        }
    }

    parentElement.appendChild(blockWrapper);
}
```

### Template Rendering

```javascript
function renderBlockFromTemplate(blockWrapper) {
    const contentArea = blockWrapper.querySelector('.block-content-area');
    if (!contentArea) return;

    let finalHtml = blockWrapper.htmlTemplate;
    const blockId = blockWrapper.id;

    // Replace settings placeholders: settings-label → "My Button"
    Object.keys(blockWrapper.settings).forEach(key => {
        const value = blockWrapper.settings[key];
        const placeholder = new RegExp(`settings-${key}(?![a-zA-Z0-9_])`, 'g');
        finalHtml = finalHtml.replace(placeholder, value);
    });

    // Replace block ID placeholder
    finalHtml = finalHtml.replace(/settings-__block_id__/g, blockId);

    contentArea.innerHTML = finalHtml;

    renderCommandParams(blockWrapper);

    // Initialize interactive elements
    requestAnimationFrame(() => {
        const hitbox = blockWrapper.querySelector('.joy-hitbox');
        if (hitbox) initJoystick(blockWrapper);

        const mousepadHitbox = blockWrapper.querySelector('.mousepad-hitbox');
        if (mousepadHitbox) initMousepad(blockWrapper);

        const metricBlock = blockWrapper.querySelector('[data-key]');
        if (metricBlock) {
            metricBlock.dataset.decimalPlaces = blockWrapper.settings['decimal_places'] || '1';
        }
    });
}
```

> **Concept: Negative lookahead regex**
> `(?![a-zA-Z0-9_])` ensures `settings-label` doesn't match `settings-label_extra`. The regex only matches when the placeholder is followed by a non-word character.

### Joystick Input

```javascript
function initJoystick(blockWrapper) {
    const hitbox = blockWrapper.querySelector('.joy-hitbox');
    const base = blockWrapper.querySelector('.joy-base');
    const thumb = blockWrapper.querySelector('.joy-thumb');

    const maxTravel = parseInt(blockWrapper.settings.travel) || 60;

    let active = false;
    let centerX, centerY;
    let activePointerId = null;

    hitbox.style.touchAction = "none";

    function handlePointerStart(e) {
        e.preventDefault();
        const rect = hitbox.getBoundingClientRect();
        active = true;
        activePointerId = e.pointerId;
        centerX = e.clientX - rect.left;
        centerY = e.clientY - rect.top;
        base.style.display = 'block';
    }

    function handlePointerMove(e) {
        if (!active || e.pointerId !== activePointerId) return;
        e.preventDefault();

        const rect = hitbox.getBoundingClientRect();
        const dx = Math.max(-maxTravel, Math.min(maxTravel, (e.clientX - rect.left) - centerX));
        const dy = Math.max(-maxTravel, Math.min(maxTravel, (e.clientY - rect.top) - centerY));

        thumb.style.transform = `translate(calc(-50% + ${dx}px), calc(-50% + ${dy}px))`;

        // Convert -maxTravel..+maxTravel to 0..255
        blockWrapper.dataset.joyX = Math.round(((dx / maxTravel + 1) / 2) * 255);
        blockWrapper.dataset.joyY = Math.round((((dy / maxTravel) + 1) / 2) * 255);

        socket.send(JSON.stringify({
            type: 'simulate-joystick',
            data: {
                js: blockWrapper.settings['joystick'],
                id: blockWrapper.settings['slider'],
                value: { x: parseInt(blockWrapper.dataset.joyX), y: parseInt(blockWrapper.dataset.joyY) }
            }
        }));
    }

    function handlePointerEnd(e) {
        if (!active) return;
        active = false;
        blockWrapper.dataset.joyX = 127;
        blockWrapper.dataset.joyY = 127;
        thumb.style.transform = `translate(-50%, -50%)`;
        // Send center position
        socket.send(JSON.stringify({ /* ... */ }));
    }

    hitbox.addEventListener('pointerdown', handlePointerStart);
    hitbox.addEventListener('pointermove', handlePointerMove);
    hitbox.addEventListener('pointerup', handlePointerEnd);
    hitbox.addEventListener('pointercancel', handlePointerEnd);
}
```

> **Concept: Pointer Events**
> Pointer Events unify mouse, touch, and pen input. `pointerdown`, `pointermove`, `pointerup` work for all input types. `e.pointerId` tracks individual touches for multi-touch support. `touchAction: "none"` prevents the browser's default scroll/zoom behavior.

> **Concept: Coordinate mapping**
> The joystick converts pixel offset (-60 to +60) to a 0–255 range:
> - -60 → 0 (full left/up)
> - 0 → 128 (center)
> - +60 → 255 (full right/down)

### Mousepad Input

```javascript
function initMousepad(blockWrapper) {
    const sensitivity = parseInt(blockWrapper.settings.sensitivity) || 5;
    const mousepadIndex = parseInt(blockWrapper.settings.mousepad) || 0;

    let active = false;
    let lastX = 0, lastY = 0;

    function handlePointerMove(e) {
        if (!active || e.pointerId !== activePointerId) return;

        const dx = Math.round((e.clientX - lastX) * sensitivity);
        const dy = Math.round((e.clientY - lastY) * sensitivity);

        lastX = e.clientX;
        lastY = e.clientY;

        if (dx !== 0 || dy !== 0) {
            socket.send(JSON.stringify({
                type: 'simulate-mousepad',
                data: { js: mousepadIndex, value: { x: dx, y: dy } }
            }));
        }
    }

    // Wheel event
    hitbox.addEventListener('wheel', (e) => {
        e.preventDefault();
        const delta = Math.sign(e.deltaY) * -1;
        socket.send(JSON.stringify({
            type: 'simulate-mousewheel',
            data: { js: mousepadIndex, delta: delta }
        }));
    }, { passive: false });

    // Mouse buttons
    leftBtn.addEventListener('pointerdown', (e) => {
        socket.send(JSON.stringify({
            type: 'simulate-mousebtn',
            data: { js: mousepadIndex, btn: 'left', state: 1 }
        }));
    });
}
```

Mouse movement is **relative** — each event reports the delta since the last event. The `sensitivity` multiplier controls how fast the cursor moves.

### 3-Finger Swipe Gesture for Panel Navigation

```javascript
let availablePanels = [];
let currentPanelIndex = -1;
let swipeState = {
    activePointers: new Map(),
    startX: 0,
    isTracking: false,
    lastSwitchTime: 0,
};

const SWIPE_THRESHOLD = 100;
const SWIPE_COOLDOWN = 500;
```

> **Concept: Event capture phase**
> The `pointerdown`, `pointermove`, `pointerup`, and `pointercancel` listeners are registered with `capture: true` (the third parameter). This means they fire during the capture phase — before events bubble down to child elements. This is critical because joystick and mousepad blocks call `e.stopPropagation()`, which would prevent the swipe detector from seeing the events if we used the default bubble phase.

```javascript
function enableThreeFingerSwipe() {
    loadPanelList();
    
    let cumulativeDeltaX = 0;
    let lastCenterX = 0;
    
    document.addEventListener('pointerdown', (e) => {
        if (e.target.closest('.joy-hitbox, .mousepad-hitbox, .push-to-talk-btn')) return;
        
        swipeState.activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
        
        if (swipeState.activePointers.size === 3 && !swipeState.isTracking) {
            swipeState.isTracking = true;
            cumulativeDeltaX = 0;
            const pointers = Array.from(swipeState.activePointers.values());
            lastCenterX = pointers.reduce((sum, p) => sum + p.x, 0) / 3;
            swipeState.startX = lastCenterX;
        }
    }, true);
```

The gesture system:
1. Tracks all active pointers in a `Map` keyed by `pointerId`
2. Activates when exactly 3 fingers are on screen
3. Calculates the center point of all 3 fingers by averaging their X coordinates
4. Ignores touches on joystick, mousepad, or push-to-talk blocks

```javascript
    document.addEventListener('pointermove', (e) => {
        if (!swipeState.isTracking || swipeState.activePointers.size !== 3) return;
        if (!swipeState.activePointers.has(e.pointerId)) return;
        
        e.preventDefault();
        
        swipeState.activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
        
        const pointers = Array.from(swipeState.activePointers.values());
        const currentCenterX = pointers.reduce((sum, p) => sum + p.x, 0) / 3;
        
        cumulativeDeltaX += (currentCenterX - lastCenterX);
        lastCenterX = currentCenterX;
        
        if (Math.abs(cumulativeDeltaX) > 30) {
            const direction = cumulativeDeltaX > 0 ? 'right' : 'left';
            const opacity = Math.min(0.8, Math.abs(cumulativeDeltaX) / SWIPE_THRESHOLD);
            showSwipeFeedback(direction, '', opacity);
        }
    }, { passive: false, capture: true });
```

> **Key Pattern (JavaScript): Cumulative delta tracking**
> Instead of comparing start position to end position, the code accumulates delta changes on every `pointermove` event. This is more accurate because:
> - It handles cases where fingers lift at slightly different times
> - It provides real-time feedback during the swipe
> - It avoids issues with touch events being interrupted or reordered
> 
> The `e.preventDefault()` call stops the browser from interpreting the 3-finger gesture as a system action (like task view on Windows or app switch on macOS).

```javascript
    function handlePointerEnd(e) {
        if (!swipeState.isTracking) return;
        
        swipeState.activePointers.delete(e.pointerId);
        
        if (swipeState.activePointers.size < 2) {
            if (Math.abs(cumulativeDeltaX) > SWIPE_THRESHOLD) {
                if (cumulativeDeltaX > 0) {
                    switchPanel('right');
                } else {
                    switchPanel('left');
                }
            } else {
                hideSwipeFeedback();
            }
            
            swipeState.isTracking = false;
            swipeState.activePointers.clear();
            cumulativeDeltaX = 0;
        }
    }
}
```

The gesture triggers when:
- At least 2 fingers have lifted (gesture is ending)
- Cumulative horizontal movement exceeds `SWIPE_THRESHOLD` (100px)
- Positive delta = swipe right (previous panel), negative = swipe left (next panel)

```javascript
function switchPanel(direction) {
    if (availablePanels.length === 0) return;
    
    const now = Date.now();
    if (now - swipeState.lastSwitchTime < SWIPE_COOLDOWN) return;
    
    let targetIndex;
    if (direction === 'left') {
        targetIndex = (currentPanelIndex - 1 + availablePanels.length) % availablePanels.length;
    } else {
        targetIndex = (currentPanelIndex + 1) % availablePanels.length;
    }
    
    // ... fetch and load panel ...
    window.history.replaceState({}, '', `/panel?name=${encodeURIComponent(targetPanel)}`);
}
```

> **Concept: Wrap-around indexing with modulo**
> The expression `(currentPanelIndex - 1 + availablePanels.length) % availablePanels.length` handles wrap-around for the previous panel. Adding `availablePanels.length` before the modulo ensures the result is always positive (JavaScript's `%` operator can return negative values for negative operands).
> 
> For 5 panels (indices 0-4):
> - At index 0, swipe left → `(0 - 1 + 5) % 5 = 4` (last panel)
> - At index 4, swipe right → `(4 + 1) % 5 = 0` (first panel)

The panel list is fetched once from `/api/panels` on initialization. Panels are ordered alphabetically by the server. The browser URL is updated with `replaceState` so the back button doesn't accumulate history entries.

### Declarative Input Elements

```javascript
function enableInputs() {
    // Buttons with emulate-button attribute
    const buttons = document.querySelectorAll('[emulate-button]');
    buttons.forEach(button => {
        const btnId = button.getAttribute('emulate-button');
        const toggleMode = button.getAttribute('toggle-mode');
        const isToggleMode = toggleMode === 'toggle';

        if (isToggleMode) {
            // Toggle mode: click toggles state, persists visually
            button.addEventListener('pointerup', (e) => {
                const newState = !button.classList.contains('active');
                button.classList.toggle('active', newState);
                button.dataset.toggledState = newState ? 'true' : 'false';
                socket.send(JSON.stringify({
                    type: 'simulate-button',
                    data: { js: jsIndex, id: btnId, state: newState ? 1 : 0 }
                }));
            });
        } else {
            // Momentary mode: click sends press/release pulse, persists visually
            button.addEventListener('pointerup', (e) => {
                const newState = !button.classList.contains('active');
                button.classList.toggle('active', newState);
                button.dataset.toggledState = newState ? 'true' : 'false';
                socket.send(JSON.stringify({
                    type: 'simulate-button',
                    data: { js: jsIndex, id: btnId, state: 1 }
                }));
                setTimeout(() => {
                    socket.send(JSON.stringify({
                        type: 'simulate-button',
                        data: { js: jsIndex, id: btnId, state: 0 }
                    }));
                }, 100);
            });
        }
    });

    // Sliders with emulate-slider attribute
    // Two modes: joystick axis emulation (default) or system control (shell commands)
    const sliders = document.querySelectorAll('[emulate-slider]');
    sliders.forEach(slider => {
        const systemControl = slider.getAttribute('system-control');
        const systemControlCommand = slider.getAttribute('system-control-command');
        const useSystemControl = systemControl && systemControl.trim() !== ''
            && !systemControl.startsWith('settings-')
            && systemControlCommand && systemControlCommand.trim() !== ''
            && !systemControlCommand.startsWith('settings-');

        slider.addEventListener('input', (event) => {
            const value = parseInt(event.target.value);

            if (useSystemControl) {
                // System control mode: replace {value} and execute as shell command
                const command = systemControlCommand.replace('{value}', value.toString());
                socket.send(JSON.stringify({
                    type: 'execute-command',
                    data: { block_id: blockId, command_type: 'shell', command: command }
                }));
            } else {
                // Joystick mode: send axis value to virtual joystick
                socket.send(JSON.stringify({
                    type: 'simulate-slider',
                    data: { js: jsIndex, id: axisId, value: value }
                }));
            }
        });
    });
}
```

> **Concept: Slider dual-mode behavior**
> Sliders operate in two mutually exclusive modes determined by the presence of `system-control` and `system-control-command` attributes:
> - **Joystick mode** (default): Sends `simulate-slider` WebSocket messages. The backend routes these to the virtual joystick driver as axis events. Games see analog input.
> - **System control mode**: Sends `execute-command` WebSocket messages with a shell command. The `{value}` placeholder is replaced with the current slider position (0 to `max_value`). This lets sliders control host system settings like volume (`pactl`) or monitor brightness (`ddcutil`).
>
> **Key Pattern (JavaScript):** The mode check uses `!systemControl.startsWith('settings-')` to detect unresolved template placeholders. When the editor hasn't set a real value, the attribute contains `settings-system_control` literally — this guard prevents accidental command execution.

Block HTML can declare interactive elements with simple attributes:
```html
<button emulate-button="0">Press Me</button>
<input type="range" emulate-slider="2" min="0" max="255">
<!-- System control slider: executes "pactl set-sink-volume @DEFAULT_SINK@ 75%" when value is 75 -->
<input type="range" emulate-slider="0" system-control="volume" system-control-command="pactl set-sink-volume @DEFAULT_SINK@ {value}%">
<div class="toggle-track" emulate-button="0" toggle-mode="toggle"></div>
```

> **Concept: Toggle mode vs Momentary mode**
> Both modes persist the visual `.active` class on the element. The difference is in the signal sent to the virtual joystick:
> - **Momentary** (default): Sends state:1 then state:0 after 100ms — simulates a quick button press. The UI toggles and stays in its new state.
> - **Toggle**: Sends state:1 when turned on, state:0 when turned off — the signal matches the visual position.
>
> Use `toggle-mode="toggle"` on any element with `emulate-button` to enable toggle behavior. The state is persisted in `dataset.toggledState` so it survives re-renders.

### Command Buttons

```javascript
const commandButtons = document.querySelectorAll('.command-button');
commandButtons.forEach(btn => {
    const blockWrapper = btn.closest('.loaded-block');
    const blockId = blockWrapper.id;

    btn.addEventListener('pointerdown', (e) => {
        executeCommand(btn, blockWrapper, blockId);

        // Hold-repeat: execute command repeatedly while held
        const holdRepeat = btn.dataset.holdRepeat === 'true';
        const holdInterval = parseInt(btn.dataset.holdInterval) || 200;

        if (holdRepeat) {
            commandHoldIntervals[blockId] = setInterval(() => {
                executeCommand(btn, blockWrapper, blockId);
            }, holdInterval);
        }
    });

    btn.addEventListener('pointerup', () => {
        clearInterval(commandHoldIntervals[blockId]);
        delete commandHoldIntervals[blockId];
    });
});
```

Command buttons support a "hold to repeat" feature — useful for things like volume up/down.

### Shared Button Content Layout

All button-type blocks (`button.html`, `command_block.html`, `sequence_button.html`) share a common
HTML structure for icon + text display:

```html
<button ...>
    <div class="btn-content layout-image-text text-bottom">
        <img class="btn-icon" data-icon-url="settings-icon_url" alt="" />
        <span class="btn-label">settings-label</span>
    </div>
</button>
```

Three layout modes are supported via the `layout` setting:
- `image-only` — icon visible, label hidden
- `text-only` — label visible, icon hidden
- `image-text` — both visible, position controlled by `text_position` (`top`, `bottom`, `left`, `right`)

The client-side `initButtonLayout()` function (`static/client/client.js`) resolves the icon URL
and configures the flexbox layout at render time:

```javascript
function initButtonLayout(blockWrapper, btn) {
    const settings = blockWrapper.settings;
    const iconUrl = settings['icon_url'];
    const layout = settings['layout'] || 'text-only';
    const textPosition = settings['text_position'] || 'bottom';
    const contentEl = blockWrapper.querySelector('.btn-content');

    if (contentEl) {
        contentEl.className = `btn-content layout-${layout} text-${textPosition}`;
    }
    // ... icon URL resolution and visibility toggling
}
```

> **Key Pattern (CSS):** Layout is controlled entirely through CSS class names on `.btn-content`.
> The `layout-*` class sets the flex direction, and `text-*` classes handle ordering via
> `flex-direction: column-reverse` or `row-reverse`. This means themes only need to style
> `.btn-content`, `.btn-icon`, and `.btn-label` — no per-block overrides required.

> **Concept: Template migration**
> When a block HTML template is updated with new settings (e.g., `icon_url`, `layout`), existing
> blocks on the canvas won't have those settings in their saved JSON. The editor's
> `migrateMissingSettings()` method (`static/editor/js/properties-panel.js`) fetches the current
> template on block selection, compares its `<settings>` tag against the block's existing settings,
> and injects any missing defaults. This ensures old panels automatically gain new features without
> manual recreation.

### Live Data Updates

```javascript
function handleDataUpdate(data) {
    const blocks = document.querySelectorAll('[data-key]');
    blocks.forEach(block => {
        const dataKey = block.getAttribute('data-key');
        const dataItem = data[dataKey];
        if (!dataItem) return;

        const valueEl = block.querySelector('.data-value');
        const unitEl = block.querySelector('.data-unit');

        if (valueEl) {
            const decimalPlaces = parseInt(block.dataset.decimalPlaces) || 1;
            const rawValue = dataItem.value;
            let newValue;
            if (typeof rawValue === 'number') {
                newValue = parseFloat(rawValue).toFixed(decimalPlaces);
            } else {
                newValue = String(rawValue);
            }
            if (valueEl.textContent !== newValue) {
                valueEl.textContent = newValue;
                valueEl.classList.add('data-flash');
                setTimeout(() => valueEl.classList.remove('data-flash'), 300);
            }
        }

        if (unitEl && dataItem.unit) {
            unitEl.textContent = dataItem.unit;
        }
    });
}
```

Blocks with a `data-key` attribute receive live updates. The value is formatted to the configured decimal places, and a CSS flash animation highlights changes.

### Command Result Feedback

```javascript
function handleCommandResult(data) {
    const block = document.getElementById(data.block_id);
    if (!block) return;

    const btn = block.querySelector('.command-button');
    if (!btn) return;

    btn.classList.remove('success', 'error');
    btn.classList.add(data.success ? 'success' : 'error');

    setTimeout(() => btn.classList.remove('success', 'error'), 1500);

    if (data.output.trim()) {
        showCommandToast(data.output.trim(), data.success);
    }
}
```

The button briefly turns green (success) or red (error), and a toast notification shows the command output.

### Wake Lock

```javascript
async function keepScreenAlive() {
    if ('wakeLock' in navigator) {
        try {
            await navigator.wakeLock.request('screen');
            document.addEventListener('visibilitychange', async () => {
                if (document.visibilityState === 'visible') {
                    await navigator.wakeLock.request('screen');
                }
            });
        } catch (err) {
            console.error(`Wake Lock failed: ${err.name}, ${err.message}`);
        }
    }
}
```

> **Concept: Wake Lock API**
> Prevents the device screen from turning off. Essential for a control panel that should stay on. The `visibilitychange` listener re-acquires the lock when the tab becomes visible again.

### Startup

```javascript
window.addEventListener('DOMContentLoaded', () => {
    const urlParams = new URLSearchParams(window.location.search);
    const panelName = urlParams.get('name');

    if (panelName) {
        fetch(`/api/panel/content?name=${encodeURIComponent(panelName)}`)
            .then(res => {
                if (!res.ok) throw new Error(`Panel "${panelName}" not found`);
                return res.json();
            })
            .then(data => loadPanel(data))
            .catch(err => {
                document.body.innerHTML = `<h1>Error</h1><p>${err.message}</p>`;
            });
    } else {
        document.body.innerHTML = `<h1>No Panel Specified</h1><p><a href="/">Go to start page</a></p>`;
    }

    connect();
    startWatchdog();
    keepScreenAlive();
});
```

When the DOM is ready, the client reads the `name` query parameter, fetches the specified panel via REST API, then connects to WebSocket for real-time features.

## Key Takeaways

- `DOMParser` parses HTML strings into inspectable DOM trees
- Pointer Events unify mouse, touch, and pen input
- Event capture phase (`capture: true`) intercepts events before child elements consume them
- 3-finger swipe gesture enables panel navigation without interfering with block inputs
- Cumulative delta tracking provides accurate gesture detection with real-time feedback
- Wrap-around indexing with modulo arithmetic enables circular panel navigation
- `data-*` attributes store metadata on DOM elements
- CSS custom properties (`--workspace-bg`) enable runtime theming
- The Wake Lock API keeps the screen on for panel displays
- Declarative attributes (`emulate-button`, `data-key`) connect HTML to behavior
- Toggle mode (`toggle-mode="toggle"`) persists visual state and sends position-based signals
- Momentary mode sends a 100ms press/release pulse while also persisting visual state
- `dataset.toggledState` persists toggle state across re-renders
- Hold-repeat uses `setInterval` / `clearInterval` for command repetition
- Negative lookahead regex prevents partial placeholder matches

[← Back: Chapter 12](12-audio-recording.md) · [Next: Chapter 14 →](14-editor-ui.md)
