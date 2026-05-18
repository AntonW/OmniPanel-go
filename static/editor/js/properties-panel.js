/**
 * PropertiesPanel renders a context-aware settings editor for selected blocks.
 * Settings are grouped by category (Theme, Identity, Appearance, Input Mapping,
 * Command, Speech, Layout). The Theme group appears first and contains a dropdown
 * populated from /api/themes, allowing per-block theme overrides. Each setting type
 * (color, select, toggle, textarea, asset, number) renders the appropriate input widget.
 * Changes are pushed to undo history and trigger a re-render of the block. The panel
 * subscribes to block-selected, selection-cleared, and blocks-changed events during init().
 */
class PropertiesPanel {
    constructor(state) {
        this.state = state;
        this.currentBlock = null;
        this.assets = [];
        /** Array of theme names loaded from /api/themes */
        this.themes = [];
        this.settingGroups = {
            'Identity': ['label'],
            'Appearance': ['button_color', 'button_color_active', 'label_color', 'bg_color',
                'border_color', 'text_color', 'icon_color', 'color', 'font_size',
                'width', 'height', 'border_radius', 'background_image'],
            'Input Mapping': ['joystick', 'button', 'slider', 'keyboard_index', 'keyboard_key',
                'mousepad', 'sensitivity', 'toggle_mode'],
            'Command': ['command_type', 'command', 'http_method', 'http_url', 'http_body',
                'hold_repeat', 'hold_interval'],
            'Speech': ['speech_trigger', 'speech_aliases', 'speech_type', 'display_mode'],
            'Media': ['control_mode', 'show_cover', 'cover_image', 'show_title', 'title', 'show_artist', 'artist',
                'show_progress', 'progress_value'],
            'RSS': ['feed_urls', 'refresh_interval', 'max_entries', 'show_date', 'show_feed_label',
                'show_description', 'description_max_length', 'new_entry_color', 'feed_label_color'],
            'Layout': ['gridx', 'gridy', 'pages', 'currentPage', 'icon_url', 'layout', 'text_position']
        };
    }

    /**
     * Initializes asset list and subscribes to editor events.
     * Listens for block-selected, selection-cleared, and blocks-changed.
     */
    async init() {
        if (window.editor && window.editor.blockRenderer) {
            this.assets = await window.editor.blockRenderer.loadAssets();
        }
        
        // Load themes
        try {
            const res = await fetch('/api/themes');
            this.themes = await res.json();
        } catch (e) {
            console.error('Failed to load themes:', e);
            this.themes = ['default'];
        }
        
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

    /**
     * Renders the properties panel for a selected block.
     * Groups settings by category and creates appropriate input widgets.
     * @param {HTMLElement} block - The selected block element with settings and settingsMeta.
     */
    async showProperties(block) {
        this.currentBlock = block;
        const panel = document.getElementById('properties-panel');
        const meta = block.settingsMeta || {};
        const settings = block.settings || {};
        const blockTheme = block.theme || window.editor?.blockRenderer?.currentPanelTheme || 'default';

        await this.migrateMissingSettings(block, settings, meta);

        const grouped = this.groupSettings(settings, meta);
        const params = this.extractParameters(settings, meta);

        panel.innerHTML = `
            <div class="properties-header">
                <h4>${escapeHtml(settings.label || block.id)}</h4>
                <span class="block-id">${escapeHtml(block.id)}</span>
            </div>
            <div class="property-group">
                <div class="group-header">
                    <span class="group-toggle">\u25BC</span>
                    <span class="group-name">Theme</span>
                </div>
                <div class="group-fields">
                    <div class="property-field">
                        <label>Block Theme</label>
                        <select data-key="__theme__">
                            ${this.themes.map(t => `<option value="${escapeHtml(t)}" ${blockTheme === t ? 'selected' : ''}>${escapeHtml(t.replace(/_/g, ' '))}</option>`).join('')}
                        </select>
                    </div>
                </div>
            </div>
            ${Object.entries(grouped).map(([group, fields]) =>
            this.renderGroup(group, fields, block)
        ).join('')}
            ${this.renderRSSFeedsSection(settings, block)}
            ${this.renderParametersSection(params, block)}
        `;

        this.attachListeners(block);
    }

    /** Re-renders the current block's properties if it still exists in the DOM. */
    async refresh() {
        if (this.currentBlock && document.getElementById(this.currentBlock.id)) {
            await this.showProperties(this.currentBlock);
        }
    }

    /** Clears the panel and shows the empty state message. */
    clear() {
        this.currentBlock = null;
        document.getElementById('properties-panel').innerHTML =
            '<div class="empty-state">Select a block to edit its properties</div>';
    }

    /**
     * Migrates existing blocks by injecting missing settings from the template.
     * When block HTML templates are updated with new settings (e.g., icon_url,
     * layout, text_position), existing blocks on the canvas won't have those
     * settings until they're recreated. This method fetches the current template,
     * parses its <settings> tag, and injects any missing defaults into the block's
     * settings and settingsMeta. After migration, the block is re-rendered so the
     * new settings take effect immediately.
     *
     * Called on every showProperties() so that clicking any existing block triggers
     * a one-time migration if needed.
     *
     * @param {HTMLElement} block - The selected block element.
     * @param {Object} settings - Block settings (mutated in place).
     * @param {Object} meta - Block settingsMeta (mutated in place).
     */
    async migrateMissingSettings(block, settings, meta) {
        const sourcePath = block.dataset.sourcePath;
        if (!sourcePath) return;

        try {
            const response = await fetch(sourcePath);
            const rawHtml = await response.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(rawHtml, 'text/html');
            const settingsTag = doc.querySelector('settings');
            if (!settingsTag) return;

            let migrated = false;
            for (let attr of settingsTag.attributes) {
                const name = attr.name;
                const value = attr.value;
                if (name.startsWith('type-') || name.startsWith('min-') ||
                    name.startsWith('max-') || name.startsWith('options-')) continue;

                if (settings[name] === undefined) {
                    settings[name] = value;
                    if (!meta[name]) meta[name] = {};
                    const typeAttr = settingsTag.getAttribute(`type-${name}`);
                    if (typeAttr) meta[name].type = typeAttr;
                    const optionsAttr = settingsTag.getAttribute(`options-${name}`);
                    if (optionsAttr) meta[name].options = optionsAttr;
                    migrated = true;
                }
            }

            if (migrated) {
                if (window.editor?.blockRenderer) {
                    await window.editor.blockRenderer.renderBlock(block);
                }
            }
        } catch (e) {
            console.warn('Migration check failed for block:', block.id, e);
        }
    }

    /**
     * Groups block settings by predefined category keys.
     * Settings not in any category are placed in an "Other" group.
     * Each field includes its value and metadata (type, min, max, options).
     * @param {Object} settings - Block settings key-value pairs.
     * @param {Object} meta - Block settings metadata (types, constraints).
     * @returns {Object} Grouped settings keyed by category name.
     */
    groupSettings(settings, meta) {
        const grouped = {};
        const used = new Set();

        for (const [groupName, keys] of Object.entries(this.settingGroups)) {
            const fields = [];
            for (const key of keys) {
                if (key in settings) {
                    fields.push({ key, value: settings[key], meta: meta[key] || { type: 'text' } });
                    used.add(key);
                }
            }
            if (fields.length > 0) {
                grouped[groupName] = fields;
            }
        }

        const remaining = [];
        for (const [key, value] of Object.entries(settings)) {
            if (!used.has(key)) {
                remaining.push({ key, value, meta: meta[key] || { type: 'text' } });
            }
        }
        if (remaining.length > 0) {
            grouped['Other'] = remaining;
        }

        return grouped;
    }

    renderGroup(groupName, fields, block) {
        return `
            <div class="property-group">
                <div class="group-header">
                    <span class="group-toggle">\u25BC</span>
                    <span class="group-name">${escapeHtml(groupName)}</span>
                </div>
                <div class="group-fields">
                    ${fields.map(f => this.renderField(f, block)).join('')}
                </div>
            </div>
        `;
    }

    renderField(field, block) {
        const { key, value, meta } = field;
        const label = key.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
        let inputHtml = '';

        switch (meta.type) {
            case 'color':
                inputHtml = this.renderColorField(key, value);
                break;
            case 'select':
                inputHtml = this.renderSelectField(key, value, meta.options);
                break;
            case 'toggle':
                inputHtml = this.renderToggleField(key, value);
                break;
            case 'textarea':
                inputHtml = this.renderTextareaField(key, value);
                break;
            case 'asset':
                inputHtml = this.renderAssetField(key, value);
                break;
            case 'number':
            case 'percentage':
                inputHtml = this.renderNumberField(key, value, meta);
                break;
            default:
                inputHtml = `<input type="text" data-key="${escapeHtml(key)}" value="${escapeHtml(value)}">`;
        }

        return `
            <div class="property-field">
                <label>${escapeHtml(label)}</label>
                ${inputHtml}
            </div>
        `;
    }

    renderColorField(key, value) {
        let hex = value.substring(0, 7);
        let alphaInt = value.length === 9 ? parseInt(value.substring(7, 9), 16) : 255;

        return `
            <div class="color-field-container">
                <div class="color-row">
                    <input type="color" data-key="${escapeHtml(key)}" value="${hex}">
                    <input type="range" data-key="${escapeHtml(key)}-alpha" min="0" max="255" value="${alphaInt}">
                </div>
                <input type="text" class="color-hex-input" data-key="${escapeHtml(key)}-hex" value="${value.toUpperCase()}">
            </div>
        `;
    }

    renderSelectField(key, value, options) {
        const opts = options.split(',').map(o => o.trim());
        return `
            <select data-key="${escapeHtml(key)}">
                ${opts.map(opt => `<option value="${escapeHtml(opt)}" ${value === opt ? 'selected' : ''}>${escapeHtml(opt)}</option>`).join('')}
            </select>
        `;
    }

    renderToggleField(key, value) {
        const checked = value === 'true' || value === true;
        return `
            <div class="toggle-row">
                <input type="checkbox" data-key="${escapeHtml(key)}" ${checked ? 'checked' : ''}>
                <span class="toggle-label">${checked ? 'On' : 'Off'}</span>
            </div>
        `;
    }

    renderTextareaField(key, value) {
        return `<textarea data-key="${escapeHtml(key)}" rows="3">${escapeHtml(value)}</textarea>`;
    }

    renderAssetField(key, value) {
        return `
            <select data-key="${escapeHtml(key)}">
                <option value="none">None</option>
                ${this.assets.map(asset => `<option value="${escapeHtml(asset)}" ${value === asset ? 'selected' : ''}>${escapeHtml(asset)}</option>`).join('')}
            </select>
        `;
    }

    renderNumberField(key, value, meta) {
        const val = meta.type === 'percentage' ? value.replace('%', '') : value;
        let attrs = `type="number" data-key="${escapeHtml(key)}" value="${escapeHtml(val)}"`;
        if (meta.min !== undefined) attrs += ` min="${meta.min}"`;
        if (meta.max !== undefined) attrs += ` max="${meta.max}"`;
        if (meta.type === 'percentage') attrs += ` step="1"`;
        return `<input ${attrs}>`;
    }

    /**
     * Attaches change listeners to all inputs in the properties panel.
     * Handles color alpha/hex compositing, checkbox boolean conversion,
     * parameter metadata changes (type, min, max), parameter value changes,
     * and pushes changes to undo history before re-rendering the block.
     * Parameter value inputs are identified by data-param-key (without data-param-meta).
     * Parameter metadata inputs are identified by both data-param-key and data-param-meta.
     * @param {HTMLElement} block - The block element whose settings are being edited.
     */
    attachListeners(block) {
        const panel = document.getElementById('properties-panel');

        panel.querySelectorAll('.group-header').forEach(header => {
            header.addEventListener('click', () => {
                header.classList.toggle('collapsed');
            });
        });

        const addParamBtn = panel.querySelector('.add-param-btn');
        if (addParamBtn) {
            addParamBtn.addEventListener('click', () => {
                this.addParameter(block);
            });
        }

        const addFeedBtn = panel.querySelector('.add-feed-btn');
        if (addFeedBtn) {
            addFeedBtn.addEventListener('click', () => {
                this.addFeed(block);
            });
        }

        panel.querySelectorAll('.remove-feed-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const feedUrl = btn.dataset.feedUrl;
                this.removeFeed(block, feedUrl);
            });
        });

        panel.querySelectorAll('.remove-param-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const paramKey = btn.dataset.paramKey;
                this.removeParameter(block, paramKey);
            });
        });

        panel.querySelectorAll('input, select, textarea').forEach(input => {
            const key = input.dataset.key;
            const paramKey = input.dataset.paramKey;
            const paramMeta = input.dataset.paramMeta;

            if (!key && !paramKey) return;

            const eventType = input.type === 'checkbox' ? 'change' : 'input';

            input.addEventListener(eventType, () => {
                let newValue;
                
                // Handle theme selector
                if (key === '__theme__') {
                    newValue = input.value;
                    block.theme = newValue;
                    
                    const oldValue = this.themes.includes(block.theme) ? block.theme : 'default';
                    
                    this.state.pushState({
                        type: 'setting-changed',
                        blockId: block.id,
                        key: '__theme__',
                        oldValue,
                        newValue
                    });
                    
                    if (window.editor && window.editor.blockRenderer) {
                        window.editor.blockRenderer.renderBlock(block);
                    }
                    return;
                }

                // Handle parameter metadata changes (type, min, max)
                if (paramKey && paramMeta) {
                    if (paramMeta === 'type') {
                        newValue = input.value;
                        const oldType = block.settingsMeta[paramKey]?.type || 'text';
                        if (!block.settingsMeta) block.settingsMeta = {};
                        if (!block.settingsMeta[paramKey]) block.settingsMeta[paramKey] = {};
                        block.settingsMeta[paramKey].type = newValue;

                        const currentVal = block.settings[paramKey];
                        if (newValue === 'toggle') {
                            block.settings[paramKey] = (currentVal === 'true' || currentVal === true || currentVal === '1') ? 'true' : 'false';
                        } else if (newValue === 'number') {
                            const num = parseFloat(currentVal);
                            block.settings[paramKey] = isNaN(num) ? '0' : String(num);
                        } else {
                            block.settings[paramKey] = String(currentVal);
                        }

                        this.state.pushState({
                            type: 'param-meta-changed',
                            blockId: block.id,
                            paramKey: paramKey,
                            metaKey: 'type',
                            oldValue: oldType,
                            newValue: newValue
                        });

                        this.showProperties(block);
                        if (window.editor && window.editor.blockRenderer) {
                            window.editor.blockRenderer.renderBlock(block);
                        }
                        return;
                    }

                    if (paramMeta === 'min' || paramMeta === 'max') {
                        newValue = input.value;
                        if (!block.settingsMeta) block.settingsMeta = {};
                        if (!block.settingsMeta[paramKey]) block.settingsMeta[paramKey] = {};
                        const oldVal = block.settingsMeta[paramKey][paramMeta];
                        block.settingsMeta[paramKey][paramMeta] = newValue;

                        this.state.pushState({
                            type: 'param-meta-changed',
                            blockId: block.id,
                            paramKey: paramKey,
                            metaKey: paramMeta,
                            oldValue: oldVal,
                            newValue: newValue
                        });

                        if (window.editor && window.editor.blockRenderer) {
                            window.editor.blockRenderer.renderBlock(block);
                        }
                        return;
                    }
                }

                // Handle parameter value changes (checkbox, text, number inputs for params)
                if (paramKey && !paramMeta) {
                    if (input.type === 'checkbox') {
                        newValue = input.checked ? 'true' : 'false';
                    } else {
                        newValue = input.value;
                    }

                    const oldValue = block.settings[paramKey];
                    block.settings[paramKey] = newValue;

                    this.state.pushState({
                        type: 'setting-changed',
                        blockId: block.id,
                        key: paramKey,
                        oldValue,
                        newValue
                    });

                    if (window.editor && window.editor.blockRenderer) {
                        window.editor.blockRenderer.renderBlock(block);
                    }
                    return;
                }

                if (key.endsWith('-alpha')) {
                    const baseKey = key.replace('-alpha', '');
                    const colorInput = panel.querySelector(`input[type="color"][data-key="${baseKey}"]`);
                    const alphaHex = parseInt(input.value).toString(16).padStart(2, '0');
                    newValue = (colorInput.value + alphaHex).toUpperCase();
                    const hexInput = panel.querySelector(`input[data-key="${baseKey}-hex"]`);
                    if (hexInput) hexInput.value = newValue;
                } else if (key.endsWith('-hex')) {
                    let val = input.value.trim();
                    if (!val.startsWith('#')) val = '#' + val;
                    if (/^#([A-Fa-f0-9]{3}|[A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$/.test(val)) {
                        newValue = val;
                        const colorInput = panel.querySelector(`input[type="color"][data-key="${key.replace('-hex', '')}"]`);
                        const alphaInput = panel.querySelector(`input[data-key="${key.replace('-hex', '')}-alpha"]`);
                        if (colorInput && val.length >= 7) colorInput.value = val.substring(0, 7);
                        if (alphaInput && val.length === 9) alphaInput.value = parseInt(val.substring(7, 9), 16);
                    } else {
                        return;
                    }
                } else if (input.type === 'checkbox') {
                    newValue = input.checked ? 'true' : 'false';
                    const label = input.parentElement.querySelector('.toggle-label');
                    if (label) label.textContent = input.checked ? 'On' : 'Off';
                } else {
                    newValue = input.value;
                }

                const oldValue = block.settings[key.replace(/-alpha|-hex/g, '')];
                const actualKey = key.replace(/-alpha|-hex/g, '');

                block.settings[actualKey] = newValue;

                this.state.pushState({
                    type: 'setting-changed',
                    blockId: block.id,
                    key: actualKey,
                    oldValue,
                    newValue
                });

                if (window.editor && window.editor.blockRenderer) {
                    window.editor.blockRenderer.renderBlock(block);
                }
            });
        });
    }

    /**
     * Extracts all param_* settings from block settings into a structured array.
     * Parameters are identified by the "param_" prefix in block.settings. Their
     * type metadata (text, number, toggle) and constraints (min, max) are read
     * from block.settingsMeta. The returned array is sorted alphabetically by name.
     * @param {Object} settings - Block settings key-value pairs.
     * @param {Object} meta - Block settings metadata (types, constraints).
     * @returns {Array} Array of parameter objects with name, value, and meta.
     */
    extractParameters(settings, meta) {
        const params = [];
        for (const [key, value] of Object.entries(settings)) {
            if (key.startsWith('param_')) {
                const paramName = key.replace('param_', '');
                const paramMeta = meta[key] || { type: 'text' };
                const minKey = `min-${paramName}`
                const maxKey = `max-${paramName}`;
                if (meta[minKey]) paramMeta.min = meta[minKey].min;
                if (meta[maxKey]) paramMeta.max = meta[maxKey].max;
                params.push({
                    key: key,
                    name: paramName,
                    value: value,
                    meta: paramMeta
                });
            }
        }
        return params.sort((a, b) => a.name.localeCompare(b.name));
    }

    /**
     * Renders the Parameters section with an Add button and existing parameter cards.
     * This section appears after all predefined setting groups. Each parameter is
     * displayed as a collapsible card with type selector, default value input, and
     * optional min/max fields for number types. An empty state message is shown when
     * no parameters exist.
     * @param {Array} params - Array of parameter objects from extractParameters().
     * @param {HTMLElement} block - The block element.
     * @returns {string} HTML string for the parameters section.
     */
    renderParametersSection(params, block) {
        const paramsHtml = params.map(p => this.renderParameterCard(p, block)).join('');
        return `
            <div class="property-group">
                <div class="group-header">
                    <span class="group-toggle">\u25BC</span>
                    <span class="group-name">Parameters</span>
                </div>
                <div class="group-fields">
                    <button class="add-param-btn" data-block-id="${escapeHtml(block.id)}">+ Add Parameter</button>
                    ${paramsHtml}
                    ${params.length === 0 ? '<div class="empty-params">No parameters defined. Click "Add Parameter" to create one.</div>' : ''}
                </div>
            </div>
        `;
    }

    /**
     * Renders a single parameter card with editable fields for type, default value,
     * and optional min/max constraints. The card adapts its value input based on the
     * parameter type: text input for text, number input for number types, and checkbox
     * for toggle. Min/max fields only appear for number types.
     * @param {Object} param - Parameter object with name, value, and meta.
     * @param {HTMLElement} block - The block element.
     * @returns {string} HTML string for the parameter card.
     */
    renderParameterCard(param, block) {
        const isNumber = param.meta.type === 'number' || param.meta.type === 'percentage';
        const isToggle = param.meta.type === 'toggle';
        const checked = isToggle && (param.value === 'true' || param.value === true);

        let valueInput = '';
        if (isNumber) {
            valueInput = `<input type="number" data-param-key="${escapeHtml(param.key)}" value="${escapeHtml(param.value)}" ${param.meta.min !== undefined ? `min="${param.meta.min}"` : ''} ${param.meta.max !== undefined ? `max="${param.meta.max}"` : ''}>`;
        } else if (isToggle) {
            valueInput = `<input type="checkbox" data-param-key="${escapeHtml(param.key)}" ${checked ? 'checked' : ''}>`;
        } else {
            valueInput = `<input type="text" data-param-key="${escapeHtml(param.key)}" value="${escapeHtml(param.value)}">`;
        }

        return `
            <div class="param-card" data-param-name="${escapeHtml(param.name)}">
                <div class="param-card-header">
                    <span class="param-name-display">${escapeHtml(param.name)}</span>
                    <button class="remove-param-btn" data-param-key="${escapeHtml(param.key)}" title="Remove parameter">\u2716</button>
                </div>
                <div class="param-card-body">
                    <div class="param-field">
                        <label>Type</label>
                        <select data-param-meta="type" data-param-key="${escapeHtml(param.key)}">
                            <option value="text" ${param.meta.type === 'text' ? 'selected' : ''}>Text</option>
                            <option value="number" ${param.meta.type === 'number' ? 'selected' : ''}>Number</option>
                            <option value="toggle" ${param.meta.type === 'toggle' ? 'selected' : ''}>Toggle</option>
                        </select>
                    </div>
                    <div class="param-field">
                        <label>Default Value</label>
                        ${valueInput}
                    </div>
                    ${isNumber ? `
                    <div class="param-field param-range-fields">
                        <div>
                            <label>Min</label>
                            <input type="number" data-param-meta="min" data-param-key="${escapeHtml(param.key)}" value="${param.meta.min !== undefined ? param.meta.min : ''}">
                        </div>
                        <div>
                            <label>Max</label>
                            <input type="number" data-param-meta="max" data-param-key="${escapeHtml(param.key)}" value="${param.meta.max !== undefined ? param.meta.max : ''}">
                        </div>
                    </div>
                    ` : ''}
                </div>
            </div>
        `;
    }

    /**
     * Adds a new parameter to the block after prompting for a name. The name is
     * sanitized to only allow alphanumeric characters and underscores, and must
     * start with a letter. Duplicate names are rejected. The new parameter is
     * created with type "text" and an empty default value. A "param-added" state
     * is pushed to undo history, and the block is re-rendered.
     * @param {HTMLElement} block - The block element.
     */
    addParameter(block) {
        const name = prompt('Parameter name (letters, numbers, underscores only):');
        if (!name) return;

        const sanitizedName = name.trim().replace(/[^a-zA-Z0-9_]/g, '_');
        if (!sanitizedName || /^[0-9]/.test(sanitizedName)) {
            alert('Invalid parameter name. Must start with a letter and contain only letters, numbers, and underscores.');
            return;
        }

        const key = `param_${sanitizedName}`;
        if (block.settings[key] !== undefined) {
            alert(`Parameter "${sanitizedName}" already exists.`);
            return;
        }

        block.settings[key] = '';
        if (!block.settingsMeta) block.settingsMeta = {};
        block.settingsMeta[key] = { type: 'text' };

        this.state.pushState({
            type: 'param-added',
            blockId: block.id,
            paramName: sanitizedName
        });

        this.showProperties(block);

        if (window.editor && window.editor.blockRenderer) {
            window.editor.blockRenderer.renderBlock(block);
        }
    }

    /**
     * Removes a parameter from the block by deleting its settings entry and metadata.
     * A "param-removed" state is pushed to undo history with the old value and metadata
     * for potential restoration. The properties panel and block preview are re-rendered.
     * @param {HTMLElement} block - The block element.
     * @param {string} paramKey - The parameter key (e.g., "param_volume").
     */
    removeParameter(block, paramKey) {
        if (block.settings[paramKey] === undefined) return;

        const paramName = paramKey.replace('param_', '');
        const oldValue = block.settings[paramKey];
        const oldMeta = block.settingsMeta ? { ...block.settingsMeta[paramKey] } : null;

        delete block.settings[paramKey];
        if (block.settingsMeta) {
            delete block.settingsMeta[paramKey];
        }

        this.state.pushState({
            type: 'param-removed',
            blockId: block.id,
            paramName: paramName,
            oldValue: oldValue,
            oldMeta: oldMeta
        });

        this.showProperties(block);

        if (window.editor && window.editor.blockRenderer) {
            window.editor.blockRenderer.renderBlock(block);
        }
    }

    /**
     * Renders the RSS Feeds section with an Add button and existing feed URLs.
     * This section appears after all predefined setting groups for RSS blocks.
     * Each feed URL is displayed as a card with a remove button.
     * @param {Object} settings - Block settings key-value pairs.
     * @param {HTMLElement} block - The block element.
     * @returns {string} HTML string for the RSS feeds section.
     */
    renderRSSFeedsSection(settings, block) {
        if (!settings['feed_urls'] && settings['feed_urls'] !== '') return '';

        const feedUrlsRaw = settings['feed_urls'] || '';
        const feeds = this.parseFeedEntries(feedUrlsRaw);

        const feedsHtml = feeds.map(f => {
            const displayLabel = f.label ? `<span class="feed-label-display">${escapeHtml(f.label)}</span>` : '';
            return `
                <div class="feed-card">
                    <span class="feed-url" title="${escapeHtml(f.url)}">${escapeHtml(f.url)}</span>
                    ${displayLabel}
                    <button class="remove-feed-btn" data-feed-url="${escapeHtml(f.url)}" title="Remove feed">\u2716</button>
                </div>
            `;
        }).join('');

        return `
            <div class="property-group">
                <div class="group-header">
                    <span class="group-toggle">\u25BC</span>
                    <span class="group-name">RSS Feeds</span>
                </div>
                <div class="group-fields">
                    <button class="add-feed-btn" data-block-id="${escapeHtml(block.id)}">+ Add Feed</button>
                    ${feedsHtml}
                    ${feeds.length === 0 ? '<div class="empty-feeds">No feeds added. Click "Add Feed" to add an RSS feed URL.</div>' : ''}
                </div>
            </div>
        `;
    }

    /**
     * Parses feed entries from the feed_urls setting.
     * Each line can be "URL" or "URL|Label".
     * @param {string} raw - The raw feed_urls setting value.
     * @returns {Array} Array of {url, label} objects.
     */
    parseFeedEntries(raw) {
        return raw.split('\n')
            .map(line => line.trim())
            .filter(line => line)
            .map(line => {
                const parts = line.split('|');
                return { url: parts[0].trim(), label: parts.length > 1 ? parts.slice(1).join('|').trim() : '' };
            });
    }

    /**
     * Adds a new feed URL to the block after prompting for the URL and optional label.
     * The entry is stored as "URL|Label" (newline-separated).
     * Duplicate URLs are rejected. The block is re-rendered.
     * @param {HTMLElement} block - The block element.
     */
    addFeed(block) {
        const url = prompt('Enter RSS/Atom feed URL:');
        if (!url) return;

        const trimmedUrl = url.trim();
        if (!trimmedUrl) return;

        const existingFeeds = this.parseFeedEntries(block.settings['feed_urls'] || '');
        if (existingFeeds.some(f => f.url === trimmedUrl)) {
            alert('This feed URL already exists.');
            return;
        }

        const label = prompt('Enter a display label for this feed (optional, leave empty to use feed title):');

        const feedUrlsRaw = block.settings['feed_urls'] || '';
        const lines = feedUrlsRaw.split('\n').filter(l => l.trim());
        const newLine = label && label.trim() ? `${trimmedUrl}|${label.trim()}` : trimmedUrl;
        lines.push(newLine);
        block.settings['feed_urls'] = lines.join('\n');

        this.state.pushState({
            type: 'setting-changed',
            blockId: block.id,
            key: 'feed_urls',
            oldValue: feedUrlsRaw,
            newValue: block.settings['feed_urls']
        });

        this.showProperties(block);

        if (window.editor && window.editor.blockRenderer) {
            window.editor.blockRenderer.renderBlock(block);
        }
    }

    /**
     * Removes a feed URL from the block by deleting it from the feed_urls setting.
     * The properties panel and block preview are re-rendered.
     * @param {HTMLElement} block - The block element.
     * @param {string} url - The feed URL to remove.
     */
    removeFeed(block, url) {
        const feedUrlsRaw = block.settings['feed_urls'] || '';
        const feeds = this.parseFeedEntries(feedUrlsRaw);
        const filteredFeeds = feeds.filter(f => f.url !== url);

        block.settings['feed_urls'] = filteredFeeds.map(f => f.label ? `${f.url}|${f.label}` : f.url).join('\n');

        this.state.pushState({
            type: 'setting-changed',
            blockId: block.id,
            key: 'feed_urls',
            oldValue: feedUrlsRaw,
            newValue: block.settings['feed_urls']
        });

        this.showProperties(block);

        if (window.editor && window.editor.blockRenderer) {
            window.editor.blockRenderer.renderBlock(block);
        }
    }
}
