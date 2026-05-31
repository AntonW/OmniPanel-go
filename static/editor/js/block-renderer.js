/**
 * BlockRenderer handles block creation, template rendering, serialization, and user interaction.
 * It fetches block HTML, parses settings tags, loads theme CSS dynamically, and scopes all
 * selectors to the block's DOM element ID to prevent cross-block style conflicts.
 *
 * Theme resolution: block.theme → panel.theme (currentPanelTheme) → 'default'
 * Each block fetches its theme CSS from /themes/<name>.css, scopes all selectors with
 * `#<blockId>`, and injects the result as a <style> tag before the block's HTML.
 */
class BlockRenderer {
    constructor(state, workspace) {
        this.state = state;
        this.workspace = workspace;
        this.assets = [];
        /** Default theme applied to new blocks when no block-level theme is set */
        this.currentPanelTheme = 'default';
    }

    /**
     * Fetches available assets from /assets/ for background image selectors.
     * The server must return an HTML directory listing (Browse: true in Fiber)
     * which is parsed to extract image filenames (png, jpg, jpeg, svg, webp, gif).
     * Populates this.assets array for use in background image dropdowns.
     */
    async loadAssets() {
        try {
            const res = await fetch('/assets/');
            if (!res.ok) return [];
            const text = await res.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(text, 'text/html');
            const links = doc.querySelectorAll('a[href]');
            this.assets = Array.from(links)
                .map(a => a.getAttribute('href'))
                .filter(h => /\.(png|jpg|jpeg|svg|webp|gif)$/i.test(h))
                .map(h => h.split('/').pop());
        } catch (err) {
            console.error('Could not read assets', err);
            this.assets = [];
        }
        return this.assets;
    }

    /**
     * Creates a new block from a drag-and-drop event. Parses the HTML template's
     * <settings> tag to extract initial values and metadata (types, min/max, options),
     * positions the block at the drop location snapped to the grid, and attaches
     * handles and event listeners.
     * @param {string} filePath - Path to the block HTML file.
     * @param {DragEvent} event - The drop event containing client coordinates.
     * @param {HTMLElement} targetDropZone - The workspace or page element receiving the drop.
     */
    async createFromDrop(filePath, event, targetDropZone) {
        try {
            const blockUrl = pathToBlockUrl(filePath);
            const response = await fetch(blockUrl);
            const rawHtml = await response.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(rawHtml, 'text/html');

            const settingsTag = doc.querySelector('settings');
            let initialSettings = {};
            let settingsMeta = {};

            if (settingsTag) {
                for (let attr of settingsTag.attributes) {
                    const name = attr.name;
                    const value = attr.value;
                    if (name.startsWith('type-')) {
                        const key = name.replace('type-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].type = value;
                    } else if (name.startsWith('min-')) {
                        const key = name.replace('min-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].min = value;
                    } else if (name.startsWith('max-')) {
                        const key = name.replace('max-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].max = value;
                    } else if (name.startsWith('options-')) {
                        const key = name.replace('options-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].options = value;
                    } else {
                        initialSettings[name] = value;
                    }
                }
                settingsTag.remove();
            }

            const rect = targetDropZone.getBoundingClientRect();
            const gX = parseInt(targetDropZone.dataset.gridx) || this.workspace.getGridSize();
            const gY = parseInt(targetDropZone.dataset.gridy) || this.workspace.getGridSize();
            const snapStepX = 100 / gX;
            const snapStepY = 100 / gY;

            const blockWrapper = document.createElement('div');
            blockWrapper.settings = initialSettings;
            blockWrapper.settingsMeta = settingsMeta;
            blockWrapper.classList.add('loaded-block');
            blockWrapper.id = generateId();
            blockWrapper.theme = this.currentPanelTheme || 'default';
            blockWrapper.style.position = 'absolute';
            blockWrapper.style.width = `${snapStepX * 2}%`;
            blockWrapper.style.height = `${snapStepY * 2}%`;
            blockWrapper.htmlTemplate = doc.body.innerHTML;
            blockWrapper.dataset.sourcePath = filePath;

            const xPercent = ((event.clientX - rect.left) / rect.width) * 100 - snapStepX;
            const yPercent = ((event.clientY - rect.top) / rect.height) * 100 - snapStepY;
            let snappedX = Math.round(xPercent / snapStepX) * snapStepX;
            let snappedY = Math.round(yPercent / snapStepY) * snapStepY;
            snappedX = clamp(snappedX, 0, 100 - snapStepX * 2);
            snappedY = clamp(snappedY, 0, 100 - snapStepY * 2);

            blockWrapper.style.left = `${snappedX}%`;
            blockWrapper.style.top = `${snappedY}%`;

            this.addHandles(blockWrapper);
            targetDropZone.appendChild(blockWrapper);

            await this.renderBlock(blockWrapper);
            this.attachBlockListeners(blockWrapper);

            this.state.pushState({
                type: 'block-added',
                blockId: blockWrapper.id,
                blockData: this.serializeBlock(blockWrapper),
                parentId: targetDropZone.id || targetDropZone.closest('.loaded-block')?.id
            });

            this.selectBlock(blockWrapper);
            this.notifyBlocksChanged();

            return blockWrapper;
        } catch (error) {
            console.error('Failed to load/parse the block:', error);
            showToast('Failed to create block', 'error');
        }
    }

    /**
     * Recreates a block from saved JSON data. Fetches the HTML template, parses the
     * <settings> tag to rebuild settingsMeta (types, min/max, options), and fills in
     * any missing setting values from template defaults. Saved settingsMeta from the
     * JSON is merged with template metadata, allowing dynamically added parameters
     * (with custom types like toggle) to persist across panel reloads. Template metadata
     * takes precedence for keys that exist in both, while saved metadata fills in entries
     * not defined in the template (e.g., user-added parameters).
     * @param {Object} blockData - Saved block JSON with id, path, settings, settingsMeta, position, size.
     * @param {HTMLElement} parentElement - DOM element to append the block to.
     */
    async createFromData(blockData, parentElement) {
        try {
            const blockUrl = pathToBlockUrl(blockData.path);
            const response = await fetch(blockUrl);
            const rawHtml = await response.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(rawHtml, 'text/html');

            const settingsMeta = {};
            if (!blockData.settings) blockData.settings = {};

            const settingsTag = doc.querySelector('settings');
            if (settingsTag) {
                for (let attr of settingsTag.attributes) {
                    const name = attr.name;
                    const value = attr.value;
                    if (name.startsWith('type-')) {
                        const key = name.replace('type-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].type = value;
                    } else if (name.startsWith('min-')) {
                        const key = name.replace('min-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].min = value;
                    } else if (name.startsWith('max-')) {
                        const key = name.replace('max-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].max = value;
                    } else if (name.startsWith('options-')) {
                        const key = name.replace('options-', '');
                        if (!settingsMeta[key]) settingsMeta[key] = {};
                        settingsMeta[key].options = value;
                    } else {
                        if (blockData.settings[name] === undefined) {
                            blockData.settings[name] = value;
                        }
                    }
                }
                settingsTag.remove();
            }

            if (blockData.settingsMeta) {
                for (const [key, meta] of Object.entries(blockData.settingsMeta)) {
                    if (!settingsMeta[key]) {
                        settingsMeta[key] = {};
                    }
                    if (meta.type) settingsMeta[key].type = meta.type;
                    if (meta.min !== undefined) settingsMeta[key].min = meta.min;
                    if (meta.max !== undefined) settingsMeta[key].max = meta.max;
                    if (meta.options) settingsMeta[key].options = meta.options;
                }
            }

            const blockWrapper = document.createElement('div');
            blockWrapper.classList.add('loaded-block');
            blockWrapper.id = blockData.id || generateId();
            blockWrapper.dataset.sourcePath = blockData.path;
            blockWrapper.theme = blockData.theme || this.currentPanelTheme || 'default';
            Object.assign(blockWrapper.style, {
                position: 'absolute',
                left: blockData.left,
                top: blockData.top,
                width: blockData.width,
                height: blockData.height,
                zIndex: blockData.zIndex || 100
            });
            blockWrapper.settings = blockData.settings;
            blockWrapper.settingsMeta = settingsMeta;
            blockWrapper.htmlTemplate = doc.head.innerHTML + doc.body.innerHTML;

            this.addHandles(blockWrapper);
            parentElement.appendChild(blockWrapper);

            await this.renderBlock(blockWrapper);
            this.attachBlockListeners(blockWrapper);

            if (blockData.children && blockData.children.length > 0) {
                const pagesContainer = blockWrapper.querySelector('.pages-container');
                if (pagesContainer) {
                    for (const pageGroup of blockData.children) {
                        const targetPage = pagesContainer.querySelector(`:scope > .page-wrapper[data-page-index="${pageGroup.pageIndex}"]`);
                        if (targetPage) {
                            for (const childBlockData of pageGroup.blocks) {
                                await this.createFromData(childBlockData, targetPage);
                            }
                        }
                    }
                }
            }

            return blockWrapper;
        } catch (error) {
            console.error(`Failed to reload block:`, error);
        }
    }

    addHandles(blockWrapper) {
        const moveHandle = document.createElement('div');
        moveHandle.classList.add('move-handle');
        moveHandle.innerHTML = '\u2629';
        moveHandle.draggable = true;
        blockWrapper.appendChild(moveHandle);

        const settingsBtn = document.createElement('div');
        settingsBtn.classList.add('settings-button');
        settingsBtn.innerHTML = '\u2699';
        blockWrapper.appendChild(settingsBtn);

        const resizeHandle = document.createElement('div');
        resizeHandle.classList.add('resize-handle');
        blockWrapper.appendChild(resizeHandle);

        const contentArea = document.createElement('div');
        contentArea.classList.add('block-content-area');
        contentArea.style.height = '100%';
        blockWrapper.appendChild(contentArea);

        const deleteBtn = document.createElement('div');
        deleteBtn.classList.add('delete-button');
        deleteBtn.innerHTML = '\u{1F5D1}';
        blockWrapper.appendChild(deleteBtn);
    }

    /**
     * Renders a block by loading its theme CSS, scoping all selectors to the block's
     * DOM element ID, replacing settings placeholders, and injecting the result into
     * the block's content area. Theme CSS is fetched asynchronously and injected as
     * a <style> tag (not <link>) to ensure proper scoping.
     * @param {HTMLElement} blockWrapper - The block DOM element with settings, theme, and htmlTemplate.
     */
    async renderBlock(blockWrapper) {
        const contentArea = blockWrapper.querySelector('.block-content-area');
        if (!contentArea) return;

        const rescuedBlocks = [];
        const existingPagesContainer = contentArea.querySelector('.pages-container');
        if (existingPagesContainer) {
            existingPagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(page => {
                const pIndex = parseInt(page.dataset.pageIndex);
                Array.from(page.querySelectorAll(':scope > .loaded-block')).forEach(b => {
                    rescuedBlocks.push({ pageIndex: pIndex, element: b });
                });
            });
        }

        let finalHtml = blockWrapper.htmlTemplate;
        const blockId = blockWrapper.id;
        
        // Determine theme: block-level > panel-level > default
        const theme = blockWrapper.theme || this.currentPanelTheme || 'default';
        
        // Load theme CSS and inject as scoped style
        try {
            const cssResponse = await fetch(`/themes/${theme}.css`);
            let themeCSS = await cssResponse.text();
            
            // Remove CSS comments first
            themeCSS = themeCSS.replace(/\/\*[\s\S]*?\*\//g, '');
            
            // Scope all CSS selectors to this block
            const scopedCSS = themeCSS.replace(/(^|}|;)\s*([^{};@]+?)\s*\{/g, (match, p1, p2) => {
                const selector = p2.trim();
                if (!selector || selector.startsWith('@')) return match;
                return `${p1} ${selector.split(',').map(sel => `#${blockId} ${sel.trim()}`).join(', ')} {`;
            });
            
            // Inject as style tag
            finalHtml = `<style>${scopedCSS}</style>` + finalHtml;
        } catch (e) {
            console.error(`Failed to load theme ${theme}:`, e);
        }

        Object.keys(blockWrapper.settings).forEach(key => {
            const value = blockWrapper.settings[key];
            const placeholder = new RegExp(`settings-${key}(?![a-zA-Z0-9_])`, 'g');
            finalHtml = finalHtml.replace(placeholder, value);
        });

        finalHtml = finalHtml.replace(/settings-__block_id__/g, blockId);

        const tempDiv = document.createElement('div');
        tempDiv.innerHTML = finalHtml;

        contentArea.innerHTML = tempDiv.innerHTML;

        if (blockWrapper.settings.pages) {
            await this.renderPages(blockWrapper, contentArea, rescuedBlocks);
        }

        this.applyBackground(blockWrapper, contentArea);
        this.applyConditionalDisplay(blockWrapper, contentArea);
    }

    applyConditionalDisplay(blockWrapper, contentArea) {
        const settings = blockWrapper.settings || {};
        
        const coverEl = contentArea.querySelector('[data-show-cover]');
        if (coverEl) {
            coverEl.style.display = settings.show_cover === 'true' ? 'flex' : 'none';
        }
        
        const titleEl = contentArea.querySelector('[data-show-title]');
        if (titleEl) {
            titleEl.style.display = settings.show_title === 'true' ? 'block' : 'none';
        }
        
        const artistEl = contentArea.querySelector('[data-show-artist]');
        if (artistEl) {
            artistEl.style.display = settings.show_artist === 'true' ? 'block' : 'none';
        }
        
        const progressEl = contentArea.querySelector('[data-show-progress]');
        if (progressEl) {
            progressEl.style.display = settings.show_progress === 'true' ? 'flex' : 'none';
        }
    }

    /**
     * Renders tab buttons and page wrappers for a paged container block.
     * Parses the `pages` setting (comma-separated names, a number, or a single name)
     * and creates corresponding tab buttons and page wrapper divs.
     *
     * The active page is determined by `blockWrapper.settings.currentPage`, defaulting to 1.
     * When a tab is clicked, the `currentPage` setting is updated so the selection persists
     * across re-renders and is serialized when the panel is saved.
     *
     * @param {HTMLElement} blockWrapper - The paged container block element.
     * @param {HTMLElement} contentArea - The content area containing .tab-header and .pages-container.
     * @param {Array} rescuedBlocks - Blocks to restore into pages during re-render (e.g., after drag).
     */
    async renderPages(blockWrapper, contentArea, rescuedBlocks) {
        const pagesSetting = blockWrapper.settings.pages.toString();
        let pageNames = [];
        let numPages = 0;

        if (pagesSetting.includes(',')) {
            pageNames = pagesSetting.split(',').map(s => s.trim());
            numPages = pageNames.length;
        } else if (!isNaN(pagesSetting) && pagesSetting.trim() !== '') {
            numPages = parseInt(pagesSetting);
            for (let i = 1; i <= numPages; i++) pageNames.push(`Page ${i}`);
        } else {
            pageNames = [pagesSetting];
            numPages = 1;
        }

        const gX = blockWrapper.settings.gridx || 10;
        const gY = blockWrapper.settings.gridy || 10;

        const header = contentArea.querySelector('.tab-header');
        const pagesContainer = contentArea.querySelector('.pages-container');

        if (header && pagesContainer) {
            header.innerHTML = '';
            pagesContainer.innerHTML = '';

            const currentPage = parseInt(blockWrapper.settings.currentPage) || 1;

            for (let i = 1; i <= numPages; i++) {
                const btn = document.createElement('button');
                btn.className = `tab-btn-editor ${i === currentPage ? 'active' : ''}`;
                btn.innerText = pageNames[i - 1] || `Page ${i}`;

                const page = document.createElement('div');
                page.className = `page-wrapper nested-dropzone ${i === currentPage ? 'active' : ''}`;
                page.dataset.pageIndex = i;
                page.dataset.gridx = gX;
                page.dataset.gridy = gY;

                const cellWidth = 100 / gX;
                const cellHeight = 100 / gY;
                page.style.backgroundSize = `${cellWidth}% ${cellHeight}%`;
                page.style.backgroundImage = `linear-gradient(to right, rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.05) 1px, transparent 1px)`;

                rescuedBlocks.forEach(async (rescue) => {
                    if (rescue.pageIndex === i) {
                        page.appendChild(rescue.element);
                        if (rescue.element.settings && rescue.element.settings.pages) {
                            await this.renderBlock(rescue.element);
                        }
                    }
                });

                btn.onclick = (e) => {
                    e.stopPropagation();
                    blockWrapper.settings.currentPage = i;
                    header.querySelectorAll(':scope > .tab-btn-editor').forEach(b => b.classList.remove('active'));
                    pagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(p => p.classList.remove('active'));
                    btn.classList.add('active');
                    page.classList.add('active');
                };

                header.appendChild(btn);
                pagesContainer.appendChild(page);
            }
        }
    }

    applyBackground(blockWrapper, contentArea) {
        const bgFile = blockWrapper.settings['background_image'];
        const target = contentArea.querySelector('.ui-container') || contentArea;
        if (bgFile && bgFile !== 'none') {
            target.style.backgroundImage = `url('/assets/${bgFile}')`;
            target.style.backgroundSize = 'cover';
            target.style.backgroundPosition = 'center';
        } else {
            target.style.backgroundImage = 'none';
        }
    }

    /**
     * Attaches selection, drag, resize, delete, and settings listeners to a block.
     * Uses capture-phase (true) for mousedown/click to intercept events before block internals.
     * Delete button, settings button, and tab buttons (tab-btn-editor) are excluded from
     * capture-phase handling so their own click listeners fire normally.
     */
    attachBlockListeners(blockWrapper) {
        const moveHandle = blockWrapper.querySelector('.move-handle');
        const resizeHandle = blockWrapper.querySelector('.resize-handle');
        const deleteBtn = blockWrapper.querySelector('.delete-button');
        const settingsBtn = blockWrapper.querySelector('.settings-button');

        blockWrapper.addEventListener('mousedown', (e) => {
            if (e.target === blockWrapper || e.target.classList.contains('block-content-area') || e.target.classList.contains('tab-btn-editor')) {
                e.stopPropagation();
                if (!e.shiftKey) {
                    this.state.clearSelection();
                }
                this.state.addSelection(blockWrapper.id);
                this.state.selectBlock(blockWrapper.id);
                document.dispatchEvent(new CustomEvent('block-selected', { detail: { block: blockWrapper } }));
            }
        }, true);

        blockWrapper.addEventListener('click', (e) => {
            if (e.target.classList.contains('delete-button') || e.target.classList.contains('settings-button') || e.target.classList.contains('tab-btn-editor')) {
                return;
            }
            e.stopPropagation();
            if (!e.shiftKey) {
                this.state.clearSelection();
            }
            this.state.addSelection(blockWrapper.id);
            this.state.selectBlock(blockWrapper.id);
            document.dispatchEvent(new CustomEvent('block-selected', { detail: { block: blockWrapper } }));
        }, true);

        if (moveHandle) {
            moveHandle.addEventListener('dragstart', (e) => {
                e.dataTransfer.setData('action', 'move');
                const rect = blockWrapper.getBoundingClientRect();
                e.dataTransfer.setData('offset', JSON.stringify({
                    x: e.clientX - rect.left,
                    y: e.clientY - rect.top
                }));
                e.dataTransfer.setDragImage(blockWrapper, e.clientX - rect.left, e.clientY - rect.top);
                window.draggedElement = blockWrapper;
                blockWrapper.classList.add('dragging');
                setTimeout(() => { blockWrapper.style.pointerEvents = 'none'; }, 0);
            });

            moveHandle.addEventListener('dragend', () => {
                blockWrapper.style.pointerEvents = 'all';
                blockWrapper.classList.remove('dragging');
                window.draggedElement = null;
                const ghostBlock = document.querySelector('.block-placeholder');
                if (ghostBlock) ghostBlock.remove();
                this.notifyBlocksChanged();
            });
        }

        if (resizeHandle) {
            this.addResizeListeners(blockWrapper, resizeHandle);
        }

        if (deleteBtn) {
            deleteBtn.addEventListener('mousedown', (e) => e.stopPropagation());
            deleteBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                this.deleteBlock(blockWrapper);
            });
        }

        if (settingsBtn) {
            settingsBtn.addEventListener('mousedown', (e) => e.stopPropagation());
            settingsBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                if (window.editor && window.editor.propertiesPanel) {
                    window.editor.propertiesPanel.showProperties(blockWrapper);
                    document.querySelector('[data-tab="properties"]').click();
                }
            });
        }
    }

    addResizeListeners(block, handle) {
        handle.addEventListener('mousedown', (e) => {
            e.stopPropagation();
            e.preventDefault();

            const startX = e.clientX;
            const startY = e.clientY;
            const parentRect = block.parentElement.getBoundingClientRect();
            const startWidth = block.offsetWidth;
            const startHeight = block.offsetHeight;
            const gX = block.parentElement.dataset.gridx ? parseInt(block.parentElement.dataset.gridx) : this.workspace.getGridSize();
            const gY = block.parentElement.dataset.gridy ? parseInt(block.parentElement.dataset.gridy) : this.workspace.getGridSize();
            const snapStepX = 100 / gX;
            const snapStepY = 100 / gY;

            const oldSize = { width: block.style.width, height: block.style.height };
            let hasMoved = false;

            const onMouseMove = (moveEvent) => {
                hasMoved = true;
                let newWidthPct = ((startWidth + moveEvent.clientX - startX) / parentRect.width) * 100;
                let newHeightPct = ((startHeight + moveEvent.clientY - startY) / parentRect.height) * 100;
                const currentLeft = parseFloat(block.style.left) || 0;
                const currentTop = parseFloat(block.style.top) || 0;
                newWidthPct = clamp(newWidthPct, snapStepX, 100 - currentLeft);
                newHeightPct = clamp(newHeightPct, snapStepY, 100 - currentTop);
                newWidthPct = Math.round(newWidthPct / snapStepX) * snapStepX;
                newHeightPct = Math.round(newHeightPct / snapStepY) * snapStepY;
                block.style.width = `${newWidthPct}%`;
                block.style.height = `${newHeightPct}%`;
            };

            const onMouseUp = () => {
                document.removeEventListener('mousemove', onMouseMove);
                document.removeEventListener('mouseup', onMouseUp);
                if (hasMoved) {
                    const newSize = { width: block.style.width, height: block.style.height };
                    window.editor.state.pushState({
                        type: 'block-resized',
                        blockId: block.id,
                        oldSize,
                        newSize
                    });
                    this.notifyBlocksChanged();
                }
            };

            document.addEventListener('mousemove', onMouseMove);
            document.addEventListener('mouseup', onMouseUp);
        });
    }

    deleteBlock(blockWrapper) {
        const blockData = this.serializeBlock(blockWrapper);
        const parentId = blockWrapper.parentElement.id || blockWrapper.parentElement.closest('.loaded-block')?.id;

        blockWrapper.style.transition = 'all 0.15s ease';
        blockWrapper.style.opacity = '0';
        blockWrapper.style.transform = 'scale(0.95)';

        setTimeout(() => {
            blockWrapper.remove();
            this.state.pushState({
                type: 'block-removed',
                blockId: blockWrapper.id,
                blockData,
                parentId
            });
            this.state.clearSelection();
            this.notifyBlocksChanged();
        }, 150);
    }

    duplicateBlock(sourceBlock) {
        const newId = generateId();
        const blockData = {
            id: newId,
            path: sourceBlock.dataset.sourcePath,
            left: (parseFloat(sourceBlock.style.left) + 2) + '%',
            top: (parseFloat(sourceBlock.style.top) + 2) + '%',
            width: sourceBlock.style.width,
            height: sourceBlock.style.height,
            zIndex: sourceBlock.style.zIndex,
            settings: JSON.parse(JSON.stringify(sourceBlock.settings || {}))
        };

        const parentId = sourceBlock.parentElement.id || sourceBlock.parentElement.closest('.loaded-block')?.id;

        this.state.pushState({
            type: 'block-duplicated',
            originalId: sourceBlock.id,
            newId
        });

        this.createFromData(blockData, parentId ? document.getElementById(parentId) : document.getElementById('maincontainer'))
            .then(newBlock => {
                if (newBlock) {
                    this.selectBlock(newBlock);
                    this.notifyBlocksChanged();
                }
            });
    }

    /**
     * Serializes a block into v2 panel JSON format. Includes the block's theme
     * (resolved from block.theme → currentPanelTheme → 'default'), path (simplified
     * without theme subdirectory), position, size, settings, settingsMeta, and nested
     * children. The settingsMeta field stores type information for dynamically added
     * parameters (e.g., toggle type) that are not defined in the HTML template.
     * @param {HTMLElement} block - The block DOM element.
     * @returns {Object} Block JSON with id, path, theme, left, top, width, height, zIndex, settings, settingsMeta, children.
     */
    serializeBlock(block) {
        const blockData = {
            id: block.id,
            left: block.style.left,
            top: block.style.top,
            width: block.style.width,
            height: block.style.height,
            zIndex: block.style.zIndex || 100,
            settings: { ...block.settings },
            settingsMeta: JSON.parse(JSON.stringify(block.settingsMeta || {})),
            path: block.dataset.sourcePath,
            theme: block.theme || this.currentPanelTheme || 'default',
            children: []
        };

        const pagesContainer = block.querySelector('.pages-container');
        if (pagesContainer) {
            pagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(page => {
                const pageIndex = page.dataset.pageIndex;
                const childBlocks = page.querySelectorAll(':scope > .loaded-block');
                if (childBlocks.length > 0) {
                    blockData.children.push({
                        pageIndex: parseInt(pageIndex),
                        blocks: Array.from(childBlocks).map(child => this.serializeBlock(child))
                    });
                }
            });
        }

        return blockData;
    }

    selectBlock(block) {
        this.state.selectBlock(block.id);
        document.dispatchEvent(new CustomEvent('block-selected', { detail: { block } }));
    }

    notifyBlocksChanged() {
        document.dispatchEvent(new CustomEvent('blocks-changed'));
        if (window.editor && window.editor.layersPanel) {
            window.editor.layersPanel.update();
        }
    }
}
