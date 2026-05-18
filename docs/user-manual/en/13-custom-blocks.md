# Chapter 13: Custom Blocks

Don't like the look of the built-in blocks? Want to create a control that doesn't exist yet? This chapter shows you how to create your own custom blocks.

## What Are Custom Blocks?

Custom blocks are HTML files you create that define new UI elements for your panels. They work exactly like the built-in blocks — you drag them from the library, configure their settings, and place them on your panel.

## Where to Put Custom Blocks

Custom blocks go in the `user/blocks/` folder:

```
OmniPanel-go/
├── user/
│   └── blocks/
│       ├── my-button.html
│       ├── my-slider.html
│       └── my-display.html
```

You can organize them into subfolders:

```
OmniPanel-go/
├── user/
│   └── blocks/
│       ├── my-theme/
│       │   ├── button.html
│       │   └── slider.html
│       └── custom/
│           └── special-control.html
```

## Creating a Simple Custom Block

### Step 1: Create an HTML File

Create a new file in `user/blocks/` with an `.html` extension. Let's call it `my-button.html`.

### Step 2: Write the HTML

A custom block is just an HTML file with special placeholders for settings:

```html
<div class="my-button" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  font-size: settings-fontsizepx;
  border-radius: settings-borderradiuspx;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
">
  settings-label
</div>
```

### Step 3: Define Settings

At the top of your HTML file, add a `<settings>` tag that lists the configurable options:

```html
<settings>
  label:My Button:text
  backgroundcolor:#333333:color
  textcolor:#ffffff:color
  fontsize:16:number
  borderradius:8:number
</settings>

<div class="my-button" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  font-size: settings-fontsizepx;
  border-radius: settings-borderradiuspx;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
">
  settings-label
</div>
```

### Step 4: Restart OmniPanel-go

Restart OmniPanel-go (or refresh the editor) and your custom block will appear in the block library.

---

## How Settings Work

### The `<settings>` Tag

The `<settings>` tag defines what options appear in the gear icon settings panel. Each line follows this format:

```
name:default_value:type
```

| Part | Description | Example |
|------|-------------|---------|
| `name` | The setting name (used in placeholders) | `label`, `backgroundcolor` |
| `default_value` | The starting value | `My Button`, `#333333` |
| `type` | The input type | `text`, `color`, `number` |

### Setting Types

| Type | What It Shows | Example |
|------|--------------|---------|
| `text` | Text input field | `label:My Button:text` |
| `color` | Color picker | `backgroundcolor:#333333:color` |
| `number` | Number input | `fontsize:16:number` |

### Using Settings in HTML

Use `settings-name` placeholders in your HTML. OmniPanel-go replaces these with the actual values when rendering the block:

```html
<!-- This placeholder... -->
settings-label

<!-- ...becomes this when the label is "Fire": -->
Fire

<!-- This placeholder... -->
settings-backgroundcolor

<!-- ...becomes this when the color is "#FF0000": -->
#FF0000
```

---

## Adding JavaScript to Custom Blocks

Custom blocks can include JavaScript to handle interactions like button presses, slider movements, and data updates.

### Example: Interactive Button

```html
<settings>
  label:Fire:text
  backgroundcolor:#cc0000:color
  textcolor:#ffffff:color
  fontsize:18:number
  joystick:0:number
  button:0:number
</settings>

<div class="my-button"
     data-joystick="settings-joystick"
     data-button="settings-button"
     style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  font-size: settings-fontsizepx;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
  border-radius: 8px;
">
  settings-label
</div>

<script>
  const button = document.currentScript.previousElementSibling;

  button.addEventListener('mousedown', () => {
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    // Send button press to OmniPanel-go
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: true
    }, '*');
  });

  button.addEventListener('mouseup', () => {
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    // Send button release to OmniPanel-go
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: false
    }, '*');
  });

  // Touch support for mobile devices
  button.addEventListener('touchstart', (e) => {
    e.preventDefault();
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: true
    }, '*');
  });

  button.addEventListener('touchend', (e) => {
    e.preventDefault();
    const joystick = parseInt(button.dataset.joystick);
    const btn = parseInt(button.dataset.button);
    window.parent.postMessage({
      type: 'joystick-button',
      joystick: joystick,
      button: btn,
      pressed: false
    }, '*');
  });
</script>
```

---

## Adding CSS to Custom Blocks

You can include CSS to style your blocks. Use a `<style>` tag:

```html
<settings>
  label:My Button:text
  backgroundcolor:#333333:color
  textcolor:#ffffff:color
</settings>

<style>
  .my-button {
    transition: all 0.2s ease;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.3);
  }

  .my-button:hover {
    transform: scale(1.05);
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
  }

  .my-button:active {
    transform: scale(0.95);
  }
</style>

<div class="my-button" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  padding: 10px 20px;
  text-align: center;
  cursor: pointer;
  user-select: none;
  border-radius: 8px;
">
  settings-label
</div>
```

---

## Advanced: Subscribing to Data

Custom blocks can display live data by subscribing to the data bus.

### Example: Data Display Block

```html
<settings>
  title:CPU:text
  datakey:cpu_usage:text
  unit:%:text
  decimals:1:number
  textcolor:#ffffff:color
  backgroundcolor:#1a1a1a:color
</settings>

<div class="data-display" style="
  background-color: settings-backgroundcolor;
  color: settings-textcolor;
  padding: 15px;
  border-radius: 8px;
  text-align: center;
">
  <div class="title">settings-title</div>
  <div class="value" id="value">--</div>
  <div class="unit">settings-unit</div>
</div>

<script>
  const display = document.currentScript.previousElementSibling;
  const valueEl = display.querySelector('#value');
  const dataKey = 'settings-datakey';
  const decimals = parseInt('settings-decimals');

  // Subscribe to data updates
  window.parent.postMessage({
    type: 'subscribe',
    key: dataKey
  }, '*');

  // Listen for data updates
  window.addEventListener('message', (e) => {
    if (e.data.type === 'data-update' && e.data.key === dataKey) {
      const value = parseFloat(e.data.value).toFixed(decimals);
      valueEl.textContent = value;
    }
  });
</script>
```

---

## Tips for Custom Blocks

### Start Simple

Begin with a basic static block (just HTML and CSS) before adding JavaScript interactions.

### Use Unique Class Names

Prefix your CSS class names to avoid conflicts with other blocks:

```css
/* Good */
.my-theme-button { }

/* Bad - might conflict with other blocks */
.button { }
```

### Test in the Editor

After creating a custom block:

1. Restart OmniPanel-go
2. Open the Editor
3. Drag your block onto the workspace
4. Test all settings in the gear icon panel
5. Open the panel on your tablet to test interactions

### File Naming

Use descriptive names for your block files:

- Good: `sci-fi-button.html`, `volume-slider.html`
- Less ideal: `block1.html`, `test.html`

### Backing Up Custom Blocks

Your custom blocks are in the `user/blocks/` folder. Back up this folder if you want to preserve your custom blocks when updating OmniPanel-go.

---

## Example: Toggle Switch Block

Here's a complete custom toggle switch using the standard `.toggle-track` structure:

```html
<settings>
  label:Autopilot:text
  joystick:0:number
  button:5:number
  toggle_color:#666666:color
  toggle_color_active:#00cc00:color
  label_color:#ffffff:color
  border_color:#00cc00:color
</settings>

<style>
  .toggle-switch {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5cqmin;
    cursor: pointer;
    user-select: none;
  }

  .toggle-track {
    height: var(--height);
    aspect-ratio: 2 / 1;
    border-radius: 999px;
    position: relative;
    transition: background-color 0.2s;
    background: var(--toggle-color);
    border: 1px solid var(--border-color);
  }

  .toggle-track::after {
    content: '';
    position: absolute;
    top: 50%;
    left: 3px;
    transform: translateY(-50%);
    width: 40%;
    height: 80%;
    border-radius: 50%;
    background: #ffffff;
    transition: left 0.2s;
  }

  .toggle-track.active {
    background-color: var(--toggle-color-active);
  }

  .toggle-track.active::after {
    left: calc(100% - 80% - 3px);
  }
</style>

<div class="toggle-switch" style="--toggle-color: settings-toggle_color; --toggle-color-active: settings-toggle_color_active; --border-color: settings-border_color;">
  <p style="color: settings-label_color;">settings-label</p>
  <div class="toggle-track" virtual-joystick="settings-joystick" emulate-button="settings-button"></div>
</div>
```

### Key Points for Toggle Blocks

- Use a `<div class="toggle-track">` element (not a `<button>`) with `virtual-joystick` and `emulate-button` attributes
- The track must have `aspect-ratio: 2 / 1` to maintain its pill shape when resized
- The knob is created via `::after` pseudo-element with `width: 40%` and `height: 80%` — this math ensures a perfect circle because 40% of the 2x-width track equals 80% of the 1x-height track
- Center the knob vertically with `top: 50%` and `transform: translateY(-50%)`
- Slide the knob to the right in the active state with `left: calc(100% - 80% - 3px)`
- The `active` class is toggled by the framework when the virtual button state changes

## Example: Sequence Button Block

Here's a complete sequence button that sends WASD keys while holding Ctrl:

```html
<settings>
label:Stratagem:text
icon_url::text
layout:image-text:select:image-only,text-only,image-text
text_position:bottom:select:top,bottom,left,right
sequence_code:d,d,w,s,a:text
key_delay:100:number
keyboard_index:0:number
font_size:14:number
button_color:#1a1a1a:color
button_color_active:#ffcc00:color
label_color:#ffffff:color
border_color:#ffcc00:color
</settings>

<style>
  .sequence-button {
    width: 100%;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    user-select: none;
    background: var(--button-color);
    color: var(--label-color);
    font-size: var(--font-size);
    border: 2px solid var(--border-color);
    clip-path: polygon(10px 0, 100% 0, 100% calc(100% - 10px), calc(100% - 10px) 100%, 0 100%, 0 10px);
    transition: background-color 0.15s, transform 0.1s;
  }

  .sequence-button:active,
  .sequence-button.executing {
    background: var(--button-color-active);
    color: #000;
    transform: scale(0.97);
  }

  .sequence-button .icon {
    width: 32px;
    height: 32px;
    object-fit: contain;
  }

  .sequence-button.layout-image-text {
    flex-direction: var(--text-position, column);
  }

  .sequence-button.layout-image-text.text-position-top {
    flex-direction: column-reverse;
  }

  .sequence-button.layout-image-text.text-position-left {
    flex-direction: row-reverse;
  }

  .sequence-button.layout-image-text.text-position-right {
    flex-direction: row;
  }
</style>

<div class="sequence-button layout-settings-layout text-position-settings-text_position"
     style="--button-color: settings-button_color; --button-color-active: settings-button_color_active; --label-color: settings-label_color; --border-color: settings-border_color; --font-size: settings-font_sizepx;"
     data-icon-url="settings-icon_url"
     data-sequence-code="settings-sequence_code"
     data-key-delay="settings-key_delay"
     data-keyboard-index="settings-keyboard_index">
  <img class="icon" data-icon-url="settings-icon_url" alt="">
  <span class="label">settings-label</span>
</div>
```

### Key Points for Sequence Button Blocks

- Use `data-sequence-code` to store the comma-separated WASD keys (e.g., `d,d,w,s,a`)
- Use `data-key-delay` for the pause between key presses in milliseconds
- Use `data-keyboard-index` to select which virtual keyboard to use
- Use `data-icon-url` instead of `src` to prevent browser 404s before JS initializes
- The `initSequenceButton` function in `client.js` handles icon loading and layout configuration
- The `executeSequence` function holds Ctrl, taps each key, then releases Ctrl
- Add the `executing` class during sequence playback for visual feedback

### Shared Button Classes for Custom Blocks

All built-in button blocks (`button.html`, `command_block.html`, `sequence_button.html`) use a
shared set of CSS classes for icon + text layouts. You can use these in your custom blocks too:

```html
<button ...>
    <div class="btn-content layout-image-text text-bottom">
        <img class="btn-icon" data-icon-url="settings-icon_url" alt="" />
        <span class="btn-label">settings-label</span>
    </div>
</button>
```

The `initButtonLayout()` function in `client.js` automatically configures these classes when it
detects `.btn-content` inside a button. Supported layouts: `image-only`, `text-only`, `image-text`
with text positions: `top`, `bottom`, `left`, `right`.

## What's Next?

Wrap up with [Tips and Troubleshooting](14-tips-and-troubleshooting.md) for best practices and common fixes.
