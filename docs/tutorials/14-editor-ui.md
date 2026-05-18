# Chapter 14: The Editor UI

## What This UI Does

The Editor is a visual panel designer. It lets you:
- Drag blocks from a categorized library onto a workspace
- Position and resize blocks on a configurable grid
- Configure block settings via a context-aware properties panel
- Organize blocks into nested pages/tabs
- Save, load, duplicate, import, and export panel definitions
- View all input bindings (joystick, keyboard, mousepad, command, speech)
- Manage speech triggers with conflict detection
- Undo/redo changes with a 100-step history

## HTML Layout (`static/editor/editor.html`)

```html
<body>
    <!-- Top menu bar: New/Load/Save/Duplicate/Delete, Undo/Redo, Export/Import -->
    <div id="menubar">
        <div class="menu-group">...</div>
    </div>

    <!-- Three-panel layout -->
    <div id="editor-layout">
        <!-- Left sidebar: categorized block library with search -->
        <div id="sidebar-left">
            <div class="sidebar-header">
                <input id="block-search" placeholder="Search blocks...">
            </div>
            <div id="block-library"></div>
        </div>

        <!-- Center: workspace with zoom/pan toolbar -->
        <div id="workspace-container">
            <div id="workspace-toolbar">
                <!-- Zoom controls, grid toggle, grid size, snap, guides -->
            </div>
            <div id="workspace-viewport">
                <div id="workspace-canvas">
                    <div id="maincontainer"></div>
                </div>
            </div>
        </div>

        <!-- Right sidebar: tabs for properties, layers, bindings, speech -->
        <div id="sidebar-right">
            <div class="sidebar-tabs">
                <button data-tab="properties">Properties</button>
                <button data-tab="layers">Layers</button>
                <button data-tab="bindings">Bindings</button>
                <button data-tab="speech">Speech</button>
            </div>
            <div id="tab-properties">...</div>
            <div id="tab-layers">...</div>
            <div id="tab-bindings">...</div>
            <div id="tab-speech">...</div>
            <div id="workspace-settings">
                <!-- Panel theme selector, aspect ratio, background color, background image -->
            </div>
        </div>
    </div>

    <!-- Modals: panel load dialog, delete confirmation -->
    <div id="modal-overlay">...</div>
    <div id="delete-modal">...</div>
    <div id="toast-container"></div>
</body>
```

## Modular JavaScript Architecture

The editor is split into 12 JavaScript modules in `static/editor/js/`:

| File | Purpose |
|------|---------|
| `utils.js` | Helper functions (escapeHtml, debounce, showToast, createModal) |
| `state.js` | EditorState class with 100-step undo/redo history |
| `workspace.js` | Zoom/pan, grid system, aspect ratio, background image |
| `block-library.js` | Categorized block palette with search |
| `block-renderer.js` | Block creation, template rendering, serialization |
| `drag-drop.js` | Drag & drop with ghost block and alignment guides |
| `selection.js` | Multi-select via shift-click and drag rectangle |
| `properties-panel.js` | Context-aware grouped settings editor |
| `panel-manager.js` | New/Load/Save/Duplicate/Delete/Export/Import |
| `binding-manager.js` | Unified view of all input/output bindings |
| `speech-center.js` | Speech command list with conflict detection |
| `main.js` | Entry point, initialization, keyboard shortcuts |

> **Concept: Module Pattern**
> Each file defines a class that receives dependencies via its constructor. The `main.js` file creates all instances and wires them together. This avoids global state and makes each module independently testable.

```javascript
// main.js — entry point
class Editor {
    constructor() {
        this.state = new EditorState();
        this.workspace = new Workspace(this.state);
        this.blockLibrary = new BlockLibrary();
        this.blockRenderer = new BlockRenderer(this.state, this.workspace);
        this.dragDrop = new DragDropManager(this.state, this.workspace, this.blockRenderer);
        this.selection = new SelectionManager(this.state, this.workspace);
        this.propertiesPanel = new PropertiesPanel(this.state);
        this.panelManager = new PanelManager(this.state);
        this.bindingManager = new BindingManager(this.state);
        this.speechCenter = new SpeechCenter(this.state);
        this.layersPanel = new LayersPanel(this.state);
    }

    async init() {
        this.workspace.init();
        this.dragDrop.init();
        this.selection.init();
        this.panelManager.init();
        this.bindingManager.init();
        this.speechCenter.init();
        this.layersPanel.init();
        await this.blockLibrary.init();
        await this.propertiesPanel.init();
        this.initTabs();
        this.initUndoRedo();
        // ...
    }
}
```

## State Management with Undo/Redo (`state.js`)

```javascript
class EditorState {
    constructor() {
        this.history = [];          // Array of { action, timestamp }
        this.historyIndex = -1;     // Current position in history
        this.maxHistory = 100;      // Maximum history size
        this.selectedBlocks = new Set();
        this.isDirty = false;
    }

    pushState(action) {
        // Trim future states if we're in the middle of history
        if (this.historyIndex < this.history.length - 1) {
            this.history = this.history.slice(0, this.historyIndex + 1);
        }
        this.history.push({ action: { ...action }, timestamp: Date.now() });
        // ...
    }

    undo() {
        const state = this.history[this.historyIndex];
        this.applyReverse(state.action);
        this.historyIndex--;
    }

    redo() {
        this.historyIndex++;
        const state = this.history[this.historyIndex];
        this.applyForward(state.action);
    }

    clearSelection() {
        this.selectedBlocks.clear();
        document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected'));
        document.dispatchEvent(new CustomEvent('selection-cleared'));
    }
}
```

> **Key Pattern: Command Pattern for Undo/Redo**
> Each action records its type and relevant data (blockId, old/new values). `applyReverse` undoes the action, `applyForward` redoes it. This avoids storing full snapshots — only the minimal diff is kept.

> **Concept: Custom DOM Events for State Changes**
> The `clearSelection()` method dispatches a `selection-cleared` event so other UI components (properties panel, binding manager, speech center) can react without tight coupling. This follows the observer pattern using the DOM's built-in event system.

Supported action types:
- `block-added` / `block-removed`
- `block-moved` (oldPos/newPos)
- `block-resized` (oldSize/newSize)
- `setting-changed` (key/oldValue/newValue)
- `block-duplicated` (originalId/newId)

## Block Library with Categories (`block-library.js`)

Blocks are fetched from `/api/blocks` and categorized by filename patterns:

```javascript
this.categories = {
    'Input Controls': ['button', 'toggle', 'emergency_button'],
    'Sliders': ['slider', 'slider_vertical', 'power_slider'],
    'Pointing': ['mousepad', 'touch_pad'],
    'Commands': ['command_block'],
    'Speech': ['speech_command', 'push_to_talk'],
    'Data Display': ['data_display'],
    'Containers': ['paged_container', 'section_group'],
};
```

Each category renders as a collapsible section with block cards showing an icon and name. Blocks are stored as consolidated HTML files in `user/blocks/` (one per block type, no theme subdirectories). The search input filters blocks by name or category.

## Context-Aware Properties Panel (`properties-panel.js`)

Settings are grouped by category, with the Theme group appearing first:

```javascript
this.settingGroups = {
    'Identity': ['label'],
    'Appearance': ['button_color', 'button_color_active', 'label_color', ...],
    'Input Mapping': ['joystick', 'button', 'slider', 'keyboard_index', ...],
    'Command': ['command_type', 'command', 'http_method', 'http_url', 'http_body',
        'hold_repeat', 'hold_interval'],
    'Speech': ['speech_trigger', 'speech_aliases', 'speech_type', ...],
    'Layout': ['gridx', 'gridy', 'pages', 'currentPage']
};
```

The panel also renders a **Parameters** section dynamically. It scans `block.settings` for keys starting with `param_` and displays each as an editable card with type selector (text/number/toggle), default value, and optional min/max fields. A "+ Add Parameter" button creates new parameters via a name prompt.

The Theme dropdown is populated from `/api/themes` and allows per-block theme overrides. Changing the theme re-renders the block with the new CSS.

The panel subscribes to editor events during `init()`:

```javascript
async init() {
    this.assets = await window.editor.blockRenderer.loadAssets();
    document.addEventListener('block-selected', (e) => {
        this.showProperties(e.detail.block);
    });
    document.addEventListener('selection-cleared', () => {
        this.clear();
    });
    document.addEventListener('blocks-changed', () => {
        this.refresh();
    });
}
```

> **Key Pattern: Event-Driven UI Updates**
> Rather than polling or manual calls, the properties panel reacts to custom DOM events. When a block is clicked, `block-selected` fires with the block element in `detail`. When selection clears, `selection-cleared` fires. This decouples the panel from the selection logic — any component can dispatch these events and the panel responds automatically.

Each setting type renders the appropriate input: color picker with alpha, dropdown, toggle, textarea, asset selector, or number input. Changes are pushed to the undo history.

## Workspace with Zoom/Pan (`workspace.js`)

```javascript
class Workspace {
    constructor(state) {
        this.zoom = 1;
        this.panX = 0;
        this.panY = 0;
        this.gridSize = 12;
        this.gridEnabled = true;
        this.snapEnabled = true;
        this.guidesEnabled = true;
    }

    initZoom() {
        // Ctrl+wheel zooms, middle-click pans
        viewport.addEventListener('wheel', (e) => {
            if (e.ctrlKey) {
                e.preventDefault();
                const delta = e.deltaY > 0 ? -0.05 : 0.05;
                this.setZoom(this.zoom + delta);
            }
        }, { passive: false });
    }

    applyTransform() {
        canvas.style.transform = `translate(calc(-50% + ${this.panX}px), calc(-50% + ${this.panY}px)) scale(${this.zoom})`;
    }

    /** Fits the panel to the viewport by calculating the optimal zoom and resetting pan to center. */
    fitToScreen() {
        const viewport = document.getElementById('workspace-viewport');
        const mainContainer = document.getElementById('maincontainer');
        const scaleX = viewport.clientWidth / mainContainer.offsetWidth;
        const scaleY = viewport.clientHeight / mainContainer.offsetHeight;
        this.setZoom(Math.min(scaleX, scaleY) * 0.95);
        this.panX = 0;
        this.panY = 0;
        this.applyTransform();
    }
}
```

> **Key Pattern: Auto-fit on Initialization**
> The `initAspect()` method calls `requestAnimationFrame(() => this.fitToScreen())` after calculating the panel size. This ensures the DOM layout is complete before fitting, so the panel is always centered and properly scaled when the editor opens.

## Block Interaction and Capture-Phase Clicks (`block-renderer.js`)

Blocks use capture-phase event listeners (`addEventListener(..., true)`) to intercept clicks before the block's internal content consumes them. This solves the common problem where clicking inside a block (on buttons, inputs, etc.) fails to select the block.

```javascript
attachBlockListeners(blockWrapper) {
    // Capture-phase mousedown: select block when clicking wrapper or content area
    // Tab buttons are excluded so they can receive drag/selection events
    blockWrapper.addEventListener('mousedown', (e) => {
        if (e.target === blockWrapper ||
            e.target.classList.contains('block-content-area') ||
            e.target.classList.contains('tab-btn-editor')) {
            e.stopPropagation();
            this.state.selectBlock(blockWrapper.id);
        }
    }, true);

    // Capture-phase click: select block, but let delete/settings/tab buttons through
    blockWrapper.addEventListener('click', (e) => {
        if (e.target.classList.contains('delete-button') ||
            e.target.classList.contains('settings-button') ||
            e.target.classList.contains('tab-btn-editor')) {
            return; // Let the button's own listener handle it
        }
        e.stopPropagation();
        this.state.selectBlock(blockWrapper.id);
    }, true);

    // Delete button: stops propagation, calls deleteBlock
    deleteBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        this.deleteBlock(blockWrapper);
    });

    // Settings button: stops propagation, opens properties panel
    settingsBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        window.editor.propertiesPanel.showProperties(blockWrapper);
    });
}
```

> **Concept: Event Propagation Phases**
> JavaScript events travel in two phases: capture (outer → inner) and bubble (inner → outer). By using capture phase (`true` as third argument), the block wrapper intercepts clicks before they reach internal elements. However, we must explicitly skip this interception for the delete button, settings button, and tab buttons (`tab-btn-editor`) so their own click handlers fire. This is critical for paged containers — without excluding `tab-btn-editor`, clicking a tab would select the block instead of switching pages.

## Paged Container Page State (`block-renderer.js`)

The `renderPages()` method builds tab buttons and page wrappers for paged container blocks. Page state is stored in `blockWrapper.settings.currentPage` (1-based index), which is read during render and updated on tab click:

```javascript
async renderPages(blockWrapper, contentArea, rescuedBlocks) {
    const pagesSetting = blockWrapper.settings.pages.toString();
    // Parse pages: comma-separated names, a number, or single name
    let pageNames = pagesSetting.includes(',')
        ? pagesSetting.split(',').map(s => s.trim())
        : [pagesSetting];
    let numPages = pageNames.length;

    const currentPage = parseInt(blockWrapper.settings.currentPage) || 1;

    for (let i = 1; i <= numPages; i++) {
        const btn = document.createElement('button');
        btn.className = `tab-btn-editor ${i === currentPage ? 'active' : ''}`;
        btn.innerText = pageNames[i - 1] || `Page ${i}`;

        const page = document.createElement('div');
        page.className = `page-wrapper nested-dropzone ${i === currentPage ? 'active' : ''}`;
        page.dataset.pageIndex = i;

        // Tab click: update currentPage setting so it persists across re-renders
        btn.onclick = (e) => {
            e.stopPropagation();
            blockWrapper.settings.currentPage = i;
            header.querySelectorAll(':scope > .tab-btn-editor')
                .forEach(b => b.classList.remove('active'));
            pagesContainer.querySelectorAll(':scope > .page-wrapper')
                .forEach(p => p.classList.remove('active'));
            btn.classList.add('active');
            page.classList.add('active');
        };
    }
}
```

> **Key Pattern: Settings as State**
> Rather than storing page state on a separate property or DOM attribute, `currentPage` lives in `blockWrapper.settings`. This means it is automatically serialized when the panel is saved (via `serializeBlock()`), appears in the Properties panel under the Layout group, and participates in undo/redo through the standard `setBlockSetting()` flow.

> **Concept: Re-render Stability**
> When any setting changes on a block, `renderBlock()` is called, which rebuilds the entire content area including tabs. By reading `currentPage` from settings at the start of `renderPages()`, the correct tab is re-activated after every re-render. Without this, the block would always reset to page 1.

## Theme System

Blocks no longer contain `<style>` tags. Instead, each block type is a single HTML file in `user/blocks/` with a `<settings>` tag and HTML structure. Theme CSS files live in `user/themes/`:

```
user/
├── blocks/
│   ├── button.html          # No <style> tag — just settings + HTML
│   ├── slider.html
│   └── ...
└── themes/
    ├── default.css          # Neutral, light-on-dark
    ├── star_citizen.css     # Sci-fi: Rajdhani font, cyan accents, clip-path
    ├── kde_breeze.css       # KDE Plasma: Noto Sans, flat, solid colors
    └── windows_11.css       # Fluent Design: Segoe UI, flat surfaces
```

### Theme Resolution

When rendering a block, the theme is resolved in this order:

1. `block.theme` — per-block override set in the Properties panel
2. `panel.theme` — panel-level default set in Workspace settings
3. `'default'` — fallback if neither is set

### CSS Scoping

Theme CSS is loaded dynamically and scoped to prevent conflicts between blocks using different themes on the same panel:

```javascript
async renderBlock(blockWrapper) {
    const theme = blockWrapper.theme || this.currentPanelTheme || 'default';
    const cssResponse = await fetch(`/themes/${theme}.css`);
    let themeCSS = await cssResponse.text();
    
    // Remove CSS comments, then scope all selectors
    themeCSS = themeCSS.replace(/\/\*[\s\S]*?\*\//g, '');
    const scopedCSS = themeCSS.replace(/(^|}|;)\s*([^{};@]+?)\s*\{/g, (match, p1, p2) => {
        const selector = p2.trim();
        if (!selector || selector.startsWith('@')) return match;
        return `${p1} ${selector.split(',').map(sel => `#${blockId} ${sel.trim()}`).join(', ')} {`;
    });
    
    finalHtml = `<style>${scopedCSS}</style>` + finalHtml;
}
```

> **Key Pattern: Dynamic CSS Scoping**
> Instead of using `<link>` tags (which apply globally), the editor fetches theme CSS as text, removes comments, prefixes every selector with the block's DOM element ID (`#block_xyz .simple-button`), and injects the result as a `<style>` tag. This ensures each block has isolated styling — two blocks on the same panel can use different themes without conflicts.

> **Concept: CSS Selector Scoping**
> The regex `/(^|}|;)\s*([^{};@]+?)\s*\{/` matches CSS rule selectors. `p1` captures the preceding `}`, `;`, or start-of-string. `p2` captures the selector text. Each selector is split by commas (for grouped selectors like `.foo, .bar`) and prefixed with `#blockId`. At-rules like `@keyframes` are skipped.

## Alignment Guides (`drag-drop.js`)

When dragging a block, dashed lines appear at edges that align with other blocks:

```javascript
calculateGuides(blockRect, otherBlocks) {
    const guides = [];
    const threshold = this.snapThreshold; // 3 pixels

    for (const other of otherBlocks) {
        const otherRect = other.getBoundingClientRect();
        // Check left, right, top, bottom edge alignment
        if (Math.abs(blockRect.left - otherRect.left) < threshold) {
            guides.push({ type: 'vertical', x: otherRect.left });
        }
        // ... center alignment, horizontal edges
    }
    return guides;
}
```

## Panel Manager (`panel-manager.js`)

Replaces the old `prompt()`-based save/load with a proper UI:

```javascript
async showLoadDialog() {
    const panels = await fetch('/api/panels').then(r => r.json());
    const modal = createModal({
        title: 'Load Panel',
        content: panels.allPanels.map(p => `
            <div class="panel-item">
                <span class="panel-name">${p}</span>
                <button class="btn-load" data-panel="${p}">Load</button>
            </div>
        `).join(''),
    });
    // ...
}
```

### Panel JSON v2 Format

```json
{
    "version": 2,
    "name": "Flight Controls",
    "theme": "star_citizen",
    "aspectRatio": "16/9",
    "grid": { "x": 12, "y": 12 },
    "backgroundColor": "#e6e6e6",
    "backgroundImage": "cockpit.jpg",
    "backgroundFit": "cover",
    "blocks": [...],
    "metadata": {
        "created": "2026-05-11T...",
        "modified": "2026-05-11T..."
    }
}
```

The top-level `theme` field sets the default theme for all new blocks. Each block in the `blocks` array can override this with its own `theme` field:

```json
{
    "id": "block_xyz",
    "path": "user/blocks/button.html",
    "theme": "kde_breeze",
    "left": "0%",
    "top": "0%",
    "width": "25%",
    "height": "16.67%",
    "zIndex": "100",
    "settings": { "label": "My Button", "button_color": "#31363bff", ... },
    "children": []
}
```

Block paths are simplified — no theme subdirectory. Each block stores setting values and dynamically-added parameter metadata in the JSON. Template metadata (`type-*`, `min-*`, `max-*`, `options-*` from the HTML `<settings>` tag) is re-parsed from the block's HTML template each time the panel loads, then merged with saved `settingsMeta`. This means:

- Template-defined settings always get their types from the HTML
- User-added parameters (via "Add Parameter") persist their custom types (e.g., toggle) because `settingsMeta` is serialized to JSON
- If a block template is updated, its setting types update automatically without migrating saved panels

> **Key Pattern: Settings vs Metadata Separation with Merge**
> Block HTML files define both values and metadata in a single `<settings>` tag. The editor separates them at load time: values go into `block.settings` (serialized to JSON), metadata goes into `block.settingsMeta`. On reload, template metadata is parsed first, then saved `settingsMeta` is merged in. This means dynamically added parameters keep their types across saves, while template-defined settings always reflect the current HTML.

> **Key Pattern: V2 Panel Structure**
> All panels now use v2 format exclusively. The top-level object contains `version: 2`, a `blocks` array for all block definitions, workspace settings (`backgroundColor`, `backgroundImage`, `backgroundFit`, `grid`, `aspectRatio`), and `metadata` with creation/modification timestamps. The Go backend (`speech.go`) and client viewer (`client.js`) both expect this structure.

## Binding Manager (`binding-manager.js`)

The Bindings tab shows all input allocations in one view:

```javascript
refresh() {
    const bindings = {
        joysticks: {},     // { jsIndex: { buttons: {}, sliders: {} } }
        keyboards: {},     // { kbIndex: [{ key, label }] }
        mousepads: {},     // { mpIndex: label }
        commands: [],      // [{ label, type, command, httpMethod, httpUrl }]
        speech: []         // [{ label, trigger, aliases, type }]
    };
    // ... scan all blocks and populate
}
```

## Speech Center (`speech-center.js`)

The Speech tab lists all speech triggers and detects conflicts:

```javascript
checkConflicts(speechBlocks) {
    const conflicts = [];
    for (let i = 0; i < speechBlocks.length; i++) {
        for (let j = i + 1; j < speechBlocks.length; j++) {
            if (isSimilar(speechBlocks[i].trigger, speechBlocks[j].trigger)) {
                conflicts.push({ a: speechBlocks[i].trigger, b: speechBlocks[j].trigger });
            }
        }
    }
    // Render conflict warnings
}
```

> **Key Pattern: Levenshtein Distance for Fuzzy Matching**
> The `isSimilar` function normalizes strings (lowercase, strip non-alphanumeric) and checks if the Levenshtein distance is less than 20% of the longer string. This catches near-duplicates like "gear up" vs "gear upp".

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+Z` | Undo |
| `Ctrl+Shift+Z` / `Ctrl+Y` | Redo |
| `Ctrl+S` | Save panel |
| `Delete` | Delete selected block(s) |
| `Escape` | Clear selection |

## Key Takeaways

- Modular architecture: 12 JavaScript files, each with a single responsibility
- Undo/redo via command pattern with 100-step history
- Categorized block library with search and icons (flat structure, no theme subdirectories)
- Context-aware properties panel with Theme group, grouped settings, and dynamic Parameters section
- Dynamic parameter management: add/remove parameters via UI, each with type (text/number/toggle), default value, and optional min/max constraints
- Parameter metadata (`settingsMeta`) is serialized to JSON for user-added parameters, merged with template metadata on reload
- Theme system: panel-level default + per-block override, CSS loaded dynamically and scoped to block DOM element ID
- Zoom/pan with Ctrl+wheel and middle-click drag; auto-fit on editor open
- Alignment guides with snap-to-block edges
- Multi-select via shift-click and drag rectangle
- Panel JSON v2 format with `theme` field, `settingsMeta`, metadata, grid config, background image
- Binding manager shows all input types in one view
- Speech center with conflict detection using Levenshtein distance
- Keyboard shortcuts for common operations
- Capture-phase click listeners for block selection, with explicit passthrough for delete/settings buttons

[← Back: Chapter 13](13-panel-ui.md) · [Next: Chapter 15 →](15-host-ui.md)
