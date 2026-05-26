/**
 * PanelManager handles panel CRUD operations (new, load, save, duplicate, delete,
 * export, import) and manages the panel-level theme setting. The panel theme is the
 * default applied to new blocks; existing blocks retain their individual theme unless
 * the user explicitly changes them. Panel JSON includes a top-level "theme" field.
 */

/**
 * Retrieves the auth token from URL params, localStorage, or sessionStorage.
 * @returns {string|null} The auth token or null.
 */
function getEditorAuthToken() {
    const urlParams = new URLSearchParams(window.location.search);
    const urlToken = urlParams.get('token');
    if (urlToken) return urlToken;
    return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
}

/**
 * Adds Authorization: Bearer header to fetch headers for API calls.
 * @param {Object} headers - Existing headers
 * @returns {Object} Headers with auth added (unchanged if no token)
 */
function addEditorAuthHeaders(headers = {}) {
    const token = getEditorAuthToken();
    if (!token) return headers;
    return { ...headers, 'Authorization': `Bearer ${token}` };
}

/**
 * Redirects to /login if the response is 401 Unauthorized.
 * Clears stored tokens to prevent redirect loops.
 * @param {Response} res - The fetch response to check
 * @returns {boolean} True if redirected (caller should return early)
 */
function handleEditorUnauthorized(res) {
    if (res.status === 401) {
        localStorage.removeItem('auth_token');
        sessionStorage.removeItem('auth_token');
        window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname + window.location.search);
        return true;
    }
    return false;
}

class PanelManager {
    constructor(state) {
        this.state = state;
        this.currentPanelName = null;
        /** Default theme for new blocks; loaded from panel JSON or set via workspace selector */
        this.currentPanelTheme = 'default';
    }

    init() {
        document.getElementById('btn-new-panel').onclick = () => this.newPanel();
        document.getElementById('btn-load-panel').onclick = () => this.showLoadDialog();
        document.getElementById('btn-save-panel').onclick = () => this.savePanel();
        document.getElementById('btn-duplicate-panel').onclick = () => this.duplicatePanel();
        document.getElementById('btn-delete-panel').onclick = () => this.deletePanel();
        document.getElementById('btn-export').onclick = () => this.exportPanel();
        document.getElementById('btn-import').onclick = () => this.importPanel();
        
        // Initialize theme selector
        this.initThemeSelector();
    }
    
    async initThemeSelector() {
        try {
            const res = await fetch('/api/themes', { headers: addEditorAuthHeaders() });
            if (handleEditorUnauthorized(res)) return;
            const themes = await res.json();
            
            const selector = document.getElementById('panel-theme-selector');
            if (selector) {
                selector.innerHTML = themes.map(t => 
                    `<option value="${escapeHtml(t)}">${escapeHtml(t.replace(/_/g, ' '))}</option>`
                ).join('');
                
                selector.addEventListener('change', (e) => {
                    this.currentPanelTheme = e.target.value;
                    if (window.editor && window.editor.blockRenderer) {
                        window.editor.blockRenderer.currentPanelTheme = e.target.value;
                    }
                    // Do NOT re-render existing blocks - panel theme only affects NEW blocks
                    // Existing blocks keep their individual theme settings
                });
            }
        } catch (e) {
            console.error('Failed to load themes:', e);
        }
    }

    newPanel() {
        if (this.state.isDirty) {
            if (!confirm('You have unsaved changes. Create a new panel anyway?')) return;
        }
        document.getElementById('maincontainer').querySelectorAll(':scope > .loaded-block').forEach(el => el.remove());
        this.currentPanelName = null;
        document.getElementById('current-panel-name').textContent = 'New Panel';
        this.state.clearHistory();
        this.state.clearSelection();
        if (window.editor) {
            if (window.editor.propertiesPanel) window.editor.propertiesPanel.clear();
            if (window.editor.layersPanel) window.editor.layersPanel.update();
            if (window.editor.bindingManager) window.editor.bindingManager.refresh();
            if (window.editor.speechCenter) window.editor.speechCenter.refresh();
        }
        showToast('New panel created');
    }

    async showLoadDialog() {
        try {
            const panelsRes = await fetch('/api/panels', { headers: addEditorAuthHeaders() });
            if (handleEditorUnauthorized(panelsRes)) return;
            const panels = await panelsRes.json();

            if (!panels.allPanels || panels.allPanels.length === 0) {
                showToast('No panels found', 'warning');
                return;
            }

            const modal = createModal({
                title: 'Load Panel',
                content: `
                    <div class="panel-list">
                        ${panels.allPanels.map(p => `
                            <div class="panel-item">
                                <span class="panel-name">${escapeHtml(p)}</span>
                                <button class="btn-load" data-panel="${escapeHtml(p)}">Load</button>
                            </div>
                        `).join('')}
                    </div>
                `,
                actions: [{ label: 'Cancel', class: 'secondary' }]
            });

            modal.element.querySelectorAll('.btn-load').forEach(btn => {
                btn.onclick = () => {
                    this.loadPanel(btn.dataset.panel);
                    modal.close();
                };
            });
        } catch (e) {
            console.error('Failed to load panel list:', e);
            showToast('Failed to load panel list', 'error');
        }
    }

    async savePanel() {
        let name = this.currentPanelName;
        if (!name) {
            name = prompt('Enter panel name:');
            if (!name) return;
        }

        const data = this.serializeWorkspace();

        try {
            const res = await fetch('/api/panel/save', {
                method: 'POST',
                headers: addEditorAuthHeaders({ 'Content-Type': 'application/json' }),
                body: JSON.stringify({ fileName: name, content: data })
            });
            if (handleEditorUnauthorized(res)) return;
            const result = await res.json();
            if (result.success) {
                this.currentPanelName = name;
                document.getElementById('current-panel-name').textContent = name;
                this.state.isDirty = false;
                showToast('Panel saved successfully', 'success');
            } else {
                showToast('Save failed: ' + result.error, 'error');
            }
        } catch (e) {
            console.error('Save failed:', e);
            showToast('Save failed', 'error');
        }
    }

    /**
     * Loads a v2 panel JSON into the editor workspace. Fetches the panel from
     * /api/panel/content, clears existing blocks, applies workspace settings
     * (background color/image, grid size, aspect ratio), then recreates each
     * block via BlockRenderer.createFromData.
     * @param {string} name - Panel name (without .json extension).
     */
    async loadPanel(name) {
        try {
            const res = await fetch(`/api/panel/content?name=${encodeURIComponent(name)}`, { headers: addEditorAuthHeaders() });
            if (handleEditorUnauthorized(res)) return;
            if (!res.ok) {
                showToast('Failed to load panel', 'error');
                return;
            }

            const data = await res.json();
            this.currentPanelName = name;
            document.getElementById('current-panel-name').textContent = name;

            document.getElementById('maincontainer').querySelectorAll(':scope > .loaded-block').forEach(el => el.remove());

            const blocks = data.blocks || [];

            if (data.backgroundColor) {
                document.getElementById('workspace-bg-color').value = data.backgroundColor;
                document.getElementById('workspace-bg-text').value = data.backgroundColor;
                document.getElementById('maincontainer').style.setProperty('--workspace-bg', data.backgroundColor);
            }

            if (data.backgroundImage) {
                document.getElementById('workspace-bg-image').value = data.backgroundImage;
                const fit = data.backgroundFit || 'cover';
                document.getElementById('workspace-bg-fit').value = fit;
                if (window.editor && window.editor.workspace) {
                    window.editor.workspace.applyBackground(
                        document.getElementById('maincontainer'),
                        data.backgroundImage,
                        fit
                    );
                }
            }

            if (data.grid) {
                document.getElementById('grid-size').value = data.grid.x;
                if (window.editor && window.editor.workspace) {
                    window.editor.workspace.gridSize = data.grid.x;
                    window.editor.workspace.updateGrid();
                }
            }

            if (data.aspectRatio) {
                document.getElementById('aspect-preset').value = data.aspectRatio;
            }
            
            // Load panel theme
            if (data.theme) {
                this.currentPanelTheme = data.theme;
                if (window.editor && window.editor.blockRenderer) {
                    window.editor.blockRenderer.currentPanelTheme = data.theme;
                }
                const themeSelector = document.getElementById('panel-theme-selector');
                if (themeSelector) themeSelector.value = data.theme;
            }

            for (const blockData of blocks) {
                if (blockData.backgroundColor) continue;
                await window.editor.blockRenderer.createFromData(blockData, document.getElementById('maincontainer'));
            }

            this.state.clearHistory();
            this.state.isDirty = false;
            if (window.editor.layersPanel) window.editor.layersPanel.update();
            if (window.editor.bindingManager) window.editor.bindingManager.refresh();
            if (window.editor.speechCenter) window.editor.speechCenter.refresh();

            showToast(`Panel "${name}" loaded`, 'success');
        } catch (e) {
            console.error('Failed to load panel:', e);
            showToast('Failed to load panel', 'error');
        }
    }

    async duplicatePanel() {
        if (!this.currentPanelName) {
            showToast('No panel loaded to duplicate', 'warning');
            return;
        }

        const newName = prompt('Enter name for duplicate:', this.currentPanelName + '_copy');
        if (!newName) return;

        try {
            const res = await fetch(`/api/panel/content?name=${encodeURIComponent(this.currentPanelName)}`, { headers: addEditorAuthHeaders() });
            if (handleEditorUnauthorized(res)) return;
            if (!res.ok) {
                showToast('Failed to load panel for duplication', 'error');
                return;
            }

            const data = await res.json();
            const saveRes = await fetch('/api/panel/save', {
                method: 'POST',
                headers: addEditorAuthHeaders({ 'Content-Type': 'application/json' }),
                body: JSON.stringify({ fileName: newName, content: data })
            });
            if (handleEditorUnauthorized(saveRes)) return;

            const result = await saveRes.json();
            if (result.success) {
                showToast(`Panel duplicated as "${newName}"`, 'success');
            } else {
                showToast('Duplicate failed: ' + result.error, 'error');
            }
        } catch (e) {
            console.error('Duplicate failed:', e);
            showToast('Duplicate failed', 'error');
        }
    }

    async deletePanel() {
        if (!this.currentPanelName) {
            showToast('No panel loaded to delete', 'warning');
            return;
        }

        if (!confirm(`Delete panel "${this.currentPanelName}"? This cannot be undone.`)) return;

        try {
            const res = await fetch(`/api/panel/delete?name=${encodeURIComponent(this.currentPanelName)}`, {
                method: 'DELETE',
                headers: addEditorAuthHeaders()
            });
            if (handleEditorUnauthorized(res)) return;

            if (res.ok) {
                showToast(`Panel "${this.currentPanelName}" deleted`, 'success');
                this.newPanel();
            } else {
                showToast('Delete failed', 'error');
            }
        } catch (e) {
            console.error('Delete failed:', e);
            showToast('Delete failed', 'error');
        }
    }

    exportPanel() {
        const data = this.serializeWorkspace();
        const json = JSON.stringify(data, null, 2);
        const blob = new Blob([json], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${this.currentPanelName || 'panel'}.json`;
        a.click();
        URL.revokeObjectURL(url);
        showToast('Panel exported', 'success');
    }

    importPanel() {
        const input = document.createElement('input');
        input.type = 'file';
        input.accept = '.json';
        input.onchange = async (e) => {
            const file = e.target.files[0];
            if (!file) return;

            try {
                const text = await file.text();
                const data = JSON.parse(text);

                const name = prompt('Enter panel name for import:', file.name.replace('.json', ''));
                if (!name) return;

                const res = await fetch('/api/panel/save', {
                    method: 'POST',
                    headers: addEditorAuthHeaders({ 'Content-Type': 'application/json' }),
                    body: JSON.stringify({ fileName: name, content: data })
                });
                if (handleEditorUnauthorized(res)) return;

                const result = await res.json();
                if (result.success) {
                    showToast(`Panel imported as "${name}"`, 'success');
                    this.loadPanel(name);
                } else {
                    showToast('Import failed: ' + result.error, 'error');
                }
            } catch (err) {
                console.error('Import failed:', err);
                showToast('Import failed: invalid JSON', 'error');
            }
        };
        input.click();
    }

    /**
     * Serializes the current editor workspace into a v2 panel JSON object.
     * Collects all top-level blocks with their serialized children (for paged
     * containers), and wraps them with version, name, grid config, background
     * settings, and metadata timestamps.
     * @returns {Object} V2 panel JSON ready for saving or export.
     */
    serializeWorkspace() {
        const mainContainer = document.getElementById('maincontainer');
        const topLevel = mainContainer.querySelectorAll(':scope > .loaded-block');
        const blocks = Array.from(topLevel).map(block =>
            window.editor.blockRenderer.serializeBlock(block)
        );

        return {
            version: 2,
            name: this.currentPanelName || 'untitled',
            theme: this.currentPanelTheme || 'default',
            aspectRatio: document.getElementById('aspect-preset').value,
            grid: {
                x: parseInt(document.getElementById('grid-size').value),
                y: parseInt(document.getElementById('grid-size').value)
            },
            backgroundColor: document.getElementById('workspace-bg-color').value,
            backgroundImage: document.getElementById('workspace-bg-image').value,
            backgroundFit: document.getElementById('workspace-bg-fit').value,
            blocks,
            metadata: {
                created: this.state.currentPanel?.metadata?.created || new Date().toISOString(),
                modified: new Date().toISOString()
            }
        };
    }
}
