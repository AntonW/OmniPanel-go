/**
 * Editor renderer - handles block rendering, workspace management, and panel save/load.
 *
 * Authentication:
 * - getAuthToken() retrieves token from URL params, localStorage, or sessionStorage
 * - checkAuthRequired() probes /api/config to detect if auth is enabled
 * - If no token and auth is required, redirects to /login
 * - All API fetch() calls use addAuthHeaders() for Bearer token injection
 */

class Block {
    constructor(name, type, absolutePath, children = []) {
        this.name = name;
        this.type = type;
        this.path = absolutePath;
        this.children = children;
    }
}

/**
 * Retrieves the authentication token from URL, localStorage, or sessionStorage.
 * Priority: URL query param > localStorage > sessionStorage.
 * @returns {string|null} The auth token or null.
 */
function getAuthToken() {
    const urlParams = new URLSearchParams(window.location.search);
    const urlToken = urlParams.get('token');
    if (urlToken) return urlToken;
    return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
}

/**
 * Adds Authorization: Bearer header to fetch headers.
 * @param {Object} headers - Existing headers
 * @returns {Object} Headers with auth added (unchanged if no token)
 */
function addAuthHeaders(headers = {}) {
    const token = getAuthToken();
    if (!token) return headers;
    return { ...headers, 'Authorization': `Bearer ${token}` };
}

/**
 * Appends token as query parameter to a URL.
 * @param {string} url - Target URL
 * @returns {string} URL with token param (unchanged if no token)
 */
function addTokenToUrl(url) {
    const token = getAuthToken();
    if (!token) return url;
    const separator = url.includes('?') ? '&' : '?';
    return `${url}${separator}token=${encodeURIComponent(token)}`;
}

/**
 * Checks if authentication is required by probing /api/config.
 * @returns {Promise<boolean>} True if server returns 401
 */
async function checkAuthRequired() {
    try {
        const res = await fetch('/api/config');
        if (res.ok) return false;
        if (res.status === 401) return true;
    } catch { }
    return false;
}

/**
 * Redirects to /login if the response is 401 Unauthorized.
 * Clears stored tokens to prevent redirect loops.
 * @param {Response} res - The fetch response to check
 * @returns {boolean} True if redirected (caller should return early)
 */
function handleUnauthorized(res) {
    if (res.status === 401) {
        localStorage.removeItem('auth_token');
        sessionStorage.removeItem('auth_token');
        window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname + window.location.search);
        return true;
    }
    return false;
}

let highestZ = 100;
const WORKSPACE_GRID_X = 12;
const WORKSPACE_GRID_Y = 12;
const DEFAULT_NESTED_GRID = 10;
let blockToDelete = null;
let ghostBlock = null;
let currentPanelName = null;

const generateId = () => `block_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

function pathToBlockUrl(filePath) {
    const idx = filePath.indexOf('blocks/');
    if (idx === -1) return filePath;
    return '/blocks/' + filePath.substring(idx + 7);
}

document.addEventListener('DOMContentLoaded', async () => {
    const authToken = getAuthToken();
    if (!authToken) {
        const authRequired = await checkAuthRequired();
        if (authRequired) {
            window.location.href = '/login?redirect=' + encodeURIComponent(window.location.pathname + window.location.search);
            return;
        }
    }

    console.log("Renderer loaded");

    const urlParams = new URLSearchParams(window.location.search);
    const isNewPanel = urlParams.get('new') === 'true' || urlParams.get('isNew') === 'true';

    initWorkspaceSettings();
    initAspectController();
    getAvailableAssets();
    BuildEditorArea();
    initWorkspaceGrid();
    initModalListeners();
    initEditorSelection();

    if (!isNewPanel) {
        loadWorkspace(true);
    }
});

function OrderZIndex(targetBlock) {
    if (!targetBlock) return;
    const allBlocks = Array.from(document.querySelectorAll('.loaded-block'));
    const otherBlocks = allBlocks.filter(b => b !== targetBlock);
    otherBlocks.sort((a, b) => (parseInt(a.style.zIndex) || 0) - (parseInt(b.style.zIndex) || 0));
    let currentZIndex = 100;
    for (const block of otherBlocks) {
        block.style.zIndex = currentZIndex;
        currentZIndex++;
    }
    targetBlock.style.zIndex = currentZIndex;
    if (targetBlock.settings) targetBlock.settings.zIndex = currentZIndex;
}

async function GetBlocks() {
    const blocksconstainer = document.getElementById("blocksconstainer");
    try {
        const res = await fetch('/api/blocks', { headers: addAuthHeaders() });
        if (handleUnauthorized(res)) return;
        const blocksRaw = await res.json();
        const blocks = blocksRaw.map(item => new Block(item.name, item.type, item.path, item.children));
        buildHtmlTree(blocks, blocksconstainer);
    } catch (e) {
        console.error("Failed to load blocks:", e);
    }
}

function initWorkspaceGrid() {
    const mainContainer = document.querySelector('#maincontainer');
    if (!mainContainer) return;
    const cellWidth = 100 / WORKSPACE_GRID_X;
    const cellHeight = 100 / WORKSPACE_GRID_Y;
    mainContainer.style.setProperty('--grid-w', `${cellWidth}%`);
    mainContainer.style.setProperty('--grid-h', `${cellHeight}%`);
}

function buildHtmlTree(blocksArray, parentElement) {
    const ul = document.createElement('ul');
    ul.classList.add('block-list');
    blocksArray.forEach(block => {
        const li = document.createElement('li');
        const label = document.createElement('span');
        label.classList.add('tree-label');
        label.textContent = block.name.replace('.html', '').replace('_', ' ');
        li.appendChild(label);
        if (block.type === 'folder') {
            li.classList.add('block-folder');
            label.addEventListener('click', (e) => { e.stopPropagation(); li.classList.toggle('collapsed'); });
            if (block.children && block.children.length > 0) buildHtmlTree(block.children, li);
        } else {
            li.classList.add('block-file');
            if (block.name.endsWith('.html')) {
                li.draggable = true;
                li.addEventListener('dragstart', (event) => {
                    event.dataTransfer.setData('text/plain', block.path);
                    event.dataTransfer.setData('block-name', block.name);
                    event.dataTransfer.effectAllowed = 'copy';
                });
            }
        }
        ul.appendChild(li);
    });
    parentElement.appendChild(ul);
}

function BuildEditorArea() {
    console.log("Editor area initializing...");
    GetBlocks();
    const mainContainer = document.querySelector('#maincontainer');

    mainContainer.addEventListener('mousedown', (e) => {
        if (e.target === mainContainer) {
            document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected'));
        }
    });

    mainContainer.addEventListener('dragover', (event) => {
        event.preventDefault();
        let targetDropZone = event.target.closest('.nested-dropzone') || mainContainer;
        const draggedBlock = window.draggedElement;
        if (draggedBlock) {
            if (!ghostBlock) {
                ghostBlock = document.createElement('div');
                ghostBlock.className = 'block-placeholder';
                targetDropZone.appendChild(ghostBlock);
            }
            if (ghostBlock.parentElement !== targetDropZone) targetDropZone.appendChild(ghostBlock);
            const rect = targetDropZone.getBoundingClientRect();
            const gX = parseInt(targetDropZone.dataset.gridx) || WORKSPACE_GRID_X;
            const gY = parseInt(targetDropZone.dataset.gridy) || WORKSPACE_GRID_Y;
            const sX = 100 / gX;
            const sY = 100 / gY;
            let wPct = (draggedBlock.offsetWidth / rect.width) * 100;
            let hPct = (draggedBlock.offsetHeight / rect.height) * 100;
            let snappedW = Math.max(sX, Math.round(wPct / sX) * sX);
            let snappedH = Math.max(sY, Math.round(hPct / sY) * sY);
            const offset = JSON.parse(event.dataTransfer.getData('offset') || '{"x":0,"y":0}');
            let lPct = ((event.clientX - rect.left - offset.x) / rect.width) * 100;
            let tPct = ((event.clientY - rect.top - offset.y) / rect.height) * 100;
            let snappedL = Math.floor(lPct / sX) * sX;
            let snappedT = Math.floor(tPct / sY) * sY;
            snappedL = Math.max(0, Math.min(snappedL, 100 - snappedW));
            snappedT = Math.max(0, Math.min(snappedT, 100 - snappedH));
            ghostBlock.style.width = `${snappedW}%`;
            ghostBlock.style.height = `${snappedH}%`;
            ghostBlock.style.left = `${snappedL}%`;
            ghostBlock.style.top = `${snappedT}%`;
        }
    });

    mainContainer.addEventListener('dragleave', (event) => {
        const targetDropZone = event.target.closest('.nested-dropzone');
        if (targetDropZone) { targetDropZone.style.outline = 'none'; targetDropZone.style.backgroundColor = ''; }
    });

    mainContainer.addEventListener('drop', async (event) => {
        event.preventDefault();
        const targetDropZone = event.target.closest('.nested-dropzone') || mainContainer;
        targetDropZone.style.outline = 'none';
        targetDropZone.style.backgroundColor = '';
        const rect = targetDropZone.getBoundingClientRect();
        let currentGridX, currentGridY;
        if (targetDropZone === mainContainer) { currentGridX = WORKSPACE_GRID_X; currentGridY = WORKSPACE_GRID_Y; }
        else { currentGridX = targetDropZone.dataset.gridx ? parseInt(targetDropZone.dataset.gridx) : DEFAULT_NESTED_GRID; currentGridY = targetDropZone.dataset.gridy ? parseInt(targetDropZone.dataset.gridy) : DEFAULT_NESTED_GRID; }
        const snapStepX = 100 / currentGridX;
        const snapStepY = 100 / currentGridY;
        const action = event.dataTransfer.getData('action');
        if (action === 'move' && window.draggedElement) {
            const block = window.draggedElement;
            let finalTarget = targetDropZone;
            if (block.contains(finalTarget)) finalTarget = block.parentElement.closest('.nested-dropzone') || mainContainer;
            const newParent = finalTarget;
            const pRect = newParent.getBoundingClientRect();
            const offset = JSON.parse(event.dataTransfer.getData('offset'));
            const physWidth = block.offsetWidth;
            const physHeight = block.offsetHeight;
            let wPct = (physWidth / pRect.width) * 100;
            let hPct = (physHeight / pRect.height) * 100;
            const gX = parseInt(newParent.dataset.gridx) || 12;
            const gY = parseInt(newParent.dataset.gridy) || 12;
            const sX = 100 / gX;
            const sY = 100 / gY;
            let snappedW = Math.min(100, Math.round(wPct / sX) * sX);
            let snappedH = Math.min(100, Math.round(hPct / sY) * sY);
            if (snappedW < sX) snappedW = sX;
            if (snappedH < sY) snappedH = sY;
            const rawXPixels = event.clientX - pRect.left - offset.x;
            const rawYPixels = event.clientY - pRect.top - offset.y;
            let lPct = (rawXPixels / pRect.width) * 100;
            let tPct = (rawYPixels / pRect.height) * 100;
            let snappedL = Math.floor(lPct / sX) * sX;
            let snappedT = Math.floor(tPct / sY) * sY;
            snappedL = Math.max(0, Math.min(snappedL, 100 - snappedW));
            snappedT = Math.max(0, Math.min(snappedT, 100 - snappedH));
            Object.assign(block.style, { width: `${snappedW}%`, height: `${snappedH}%`, left: `${snappedL}%`, top: `${snappedT}%` });
            if (block.parentElement !== newParent) newParent.appendChild(block);
            updateLayersTree();
            setTimeout(() => {
                const bRect = block.getBoundingClientRect();
                const target = block.querySelector('.move-handle') || block;
                const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window, clientX: bRect.left + bRect.width / 2, clientY: bRect.top + bRect.height / 2 });
                target.dispatchEvent(clickEvent);
                document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected'));
                block.classList.add('selected');
            }, 20);
        } else {
            const filePath = event.dataTransfer.getData('text/plain');
            if (filePath && filePath.endsWith('.html')) {
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
                            if (name.startsWith('type-')) { const key = name.replace('type-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].type = value; }
                            else if (name.startsWith('min-')) { const key = name.replace('min-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].min = value; }
                            else if (name.startsWith('max-')) { const key = name.replace('max-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].max = value; }
                            else if (name.startsWith('options-')) { const key = name.replace('options-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].options = value; }
                            else { initialSettings[name] = value; }
                        }
                        settingsTag.remove();
                    }
                    const blockWrapper = document.createElement('div');
                    blockWrapper.settings = initialSettings;
                    blockWrapper.settingsMeta = settingsMeta;
                    blockWrapper.classList.add('loaded-block');
                    blockWrapper.id = generateId();
                    blockWrapper.style.position = 'absolute';
                    blockWrapper.style.width = `${snapStepX * 2}%`;
                    blockWrapper.style.height = `${snapStepY * 2}%`;
                    blockWrapper.htmlTemplate = doc.body.innerHTML;
                    let blockWidthPercent = parseFloat(blockWrapper.style.width) || snapStepX;
                    let blockHeightPercent = parseFloat(blockWrapper.style.height) || snapStepY;
                    let xPercent = (((event.clientX - rect.left) / rect.width) * 100) - snapStepX;
                    let yPercent = (((event.clientY - rect.top) / rect.height) * 100) - snapStepY;
                    let snappedX = Math.round(xPercent / snapStepX) * snapStepX;
                    let snappedY = Math.round(yPercent / snapStepY) * snapStepY;
                    snappedX = Math.max(0, Math.min(snappedX, 100 - blockWidthPercent));
                    snappedY = Math.max(0, Math.min(snappedY, 100 - blockHeightPercent));
                    blockWrapper.style.left = `${snappedX}%`;
                    blockWrapper.style.top = `${snappedY}%`;
                    const moveHandle = document.createElement('div'); moveHandle.classList.add('move-handle'); moveHandle.innerHTML = '☩'; moveHandle.draggable = true; blockWrapper.appendChild(moveHandle);
                    const settingsBtn = document.createElement('div'); settingsBtn.classList.add('settings-button'); settingsBtn.innerHTML = '⚙'; blockWrapper.appendChild(settingsBtn);
                    const resizeHandle = document.createElement('div'); resizeHandle.classList.add('resize-handle'); blockWrapper.appendChild(resizeHandle);
                    const contentArea = document.createElement('div'); contentArea.classList.add('block-content-area'); contentArea.style.height = '100%'; blockWrapper.appendChild(contentArea);
                    const deleteBtn = document.createElement('div'); deleteBtn.classList.add('delete-button'); deleteBtn.innerHTML = '🗑'; blockWrapper.appendChild(deleteBtn);
                    blockWrapper.dataset.sourcePath = filePath;
                    renderBlockFromTemplate(blockWrapper);
                    addSelectionListeners(blockWrapper);
                    addWorkspaceDragListeners(blockWrapper, moveHandle);
                    addResizeListeners(blockWrapper, resizeHandle);
                    addDeleteFunctionality(blockWrapper, deleteBtn);
                    settingsBtn.addEventListener('mousedown', (e) => e.stopPropagation());
                    settingsBtn.addEventListener('click', (e) => { e.stopPropagation(); openSettingsModal(blockWrapper); });
                    targetDropZone.appendChild(blockWrapper);
                    updateLayersTree();
                    setTimeout(() => {
                        const bRect = blockWrapper.getBoundingClientRect();
                        const target = blockWrapper.querySelector('.move-handle') || blockWrapper;
                        const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window, clientX: bRect.left + bRect.width / 2, clientY: bRect.top + bRect.height / 2 });
                        target.dispatchEvent(clickEvent);
                        document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected'));
                        blockWrapper.classList.add('selected');
                    }, 20);
                } catch (error) { console.error("Failed to load/parse the block:", error); }
            }
        }
    });
}

function addWorkspaceDragListeners(blockWrapper, moveHandle) {
    moveHandle.addEventListener('dragstart', (e) => {
        e.dataTransfer.setData('action', 'move');
        const rect = blockWrapper.getBoundingClientRect();
        e.dataTransfer.setData('offset', JSON.stringify({ x: e.clientX - rect.left, y: e.clientY - rect.top }));
        e.dataTransfer.setDragImage(blockWrapper, e.clientX - rect.left, e.clientY - rect.top);
        window.draggedElement = blockWrapper;
        setTimeout(() => { blockWrapper.style.pointerEvents = 'none'; }, 0);
    });
    moveHandle.addEventListener('dragend', (e) => {
        blockWrapper.style.pointerEvents = 'all';
        window.draggedElement = null;
        if (ghostBlock) { ghostBlock.remove(); ghostBlock = null; }
        updateLayersTree();
    });
}

function addSelectionListeners(blockWrapper) {
    blockWrapper.addEventListener('mousedown', (e) => {
        e.stopPropagation();
        OrderZIndex(blockWrapper);
        document.querySelectorAll('.loaded-block.selected').forEach(el => { if (el !== blockWrapper) el.classList.remove('selected'); });
        blockWrapper.classList.add('selected');
    });
}

function addResizeListeners(block, handle) {
    handle.addEventListener('mousedown', (e) => {
        e.stopPropagation(); e.preventDefault();
        const startX = e.clientX; const startY = e.clientY;
        const parentRect = block.parentElement.getBoundingClientRect();
        const startWidth = block.offsetWidth; const startHeight = block.offsetHeight;
        const gX = block.parentElement.dataset.gridx ? parseInt(block.parentElement.dataset.gridx) : WORKSPACE_GRID_X;
        const gY = block.parentElement.dataset.gridy ? parseInt(block.parentElement.dataset.gridy) : WORKSPACE_GRID_Y;
        const snapStepX = 100 / gX; const snapStepY = 100 / gY;
        let hasMoved = false;
        const onMouseMove = (moveEvent) => {
            hasMoved = true;
            let newWidthPct = ((startWidth + moveEvent.clientX - startX) / parentRect.width) * 100;
            let newHeightPct = ((startHeight + moveEvent.clientY - startY) / parentRect.height) * 100;
            const currentLeft = parseFloat(block.style.left) || 0; const currentTop = parseFloat(block.style.top) || 0;
            newWidthPct = Math.max(snapStepX, Math.min(newWidthPct, 100 - currentLeft));
            newHeightPct = Math.max(snapStepY, Math.min(newHeightPct, 100 - currentTop));
            newWidthPct = Math.round(newWidthPct / snapStepX) * snapStepX;
            newHeightPct = Math.round(newHeightPct / snapStepY) * snapStepY;
            block.style.width = `${newWidthPct}%`; block.style.height = `${newHeightPct}%`;
        };
        const onMouseUp = () => {
            document.removeEventListener('mousemove', onMouseMove); document.removeEventListener('mouseup', onMouseUp);
            if (hasMoved) {
                const capturePhantomClick = (ce) => { ce.stopPropagation(); ce.preventDefault(); };
                document.addEventListener('click', capturePhantomClick, { capture: true, once: true });
                setTimeout(() => { document.removeEventListener('click', capturePhantomClick, { capture: true }); }, 50);
            }
        };
        document.addEventListener('mousemove', onMouseMove); document.addEventListener('mouseup', onMouseUp);
    });
}

function renderBlockFromTemplate(blockWrapper) {
    const contentArea = blockWrapper.querySelector('.block-content-area');
    if (!contentArea) return;
    const rescuedBlocks = [];
    const existingPagesContainer = contentArea.querySelector('.pages-container');
    if (existingPagesContainer) {
        existingPagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(page => {
            const pIndex = parseInt(page.dataset.pageIndex);
            Array.from(page.querySelectorAll(':scope > .loaded-block')).forEach(b => rescuedBlocks.push({ pageIndex: pIndex, element: b }));
        });
    }
    let finalHtml = blockWrapper.htmlTemplate;
    const blockId = blockWrapper.id;
    Object.keys(blockWrapper.settings).forEach(key => {
        const value = blockWrapper.settings[key];
        const placeholder = new RegExp(`settings-${key}(?![a-zA-Z0-9_])`, 'g');
        finalHtml = finalHtml.replace(placeholder, value);
    });
    const tempDiv = document.createElement('div'); tempDiv.innerHTML = finalHtml;
    tempDiv.querySelectorAll('style').forEach(style => {
        style.innerHTML = style.innerHTML.replace(/(^|}|;)\s*([^{};]+)\s*\{/g, (match, p1, p2) => {
            return `${p1} ${p2.split(',').map(sel => `#${blockId} ${sel.trim()}`).join(', ')} {`;
        });
    });
    contentArea.innerHTML = tempDiv.innerHTML;
    if (blockWrapper.settings.pages) {
        const pagesSetting = blockWrapper.settings.pages.toString();
        let pageNames = []; let numPages = 0;
        if (pagesSetting.includes(',')) { pageNames = pagesSetting.split(',').map(s => s.trim()); numPages = pageNames.length; }
        else if (!isNaN(pagesSetting) && pagesSetting.trim() !== "") { numPages = parseInt(pagesSetting); for (let i = 1; i <= numPages; i++) pageNames.push(`Page ${i}`); }
        else { pageNames = [pagesSetting]; numPages = 1; }
        const gX = blockWrapper.settings.gridx || 10; const gY = blockWrapper.settings.gridy || 10;
        const header = contentArea.querySelector('.tab-header'); const pagesContainer = contentArea.querySelector('.pages-container');
        if (header && pagesContainer) {
            header.innerHTML = ''; pagesContainer.innerHTML = '';
            for (let i = 1; i <= numPages; i++) {
                const btn = document.createElement('button'); btn.className = `tab-btn ${i === 1 ? 'active' : ''}`; btn.innerText = pageNames[i - 1] || `Page ${i}`;
                const page = document.createElement('div'); page.className = `page-wrapper nested-dropzone ${i === 1 ? 'active' : ''}`; page.dataset.pageIndex = i; page.dataset.gridx = gX; page.dataset.gridy = gY;
                const cellWidth = 100 / gX; const cellHeight = 100 / gY;
                page.style.backgroundSize = `${cellWidth}% ${cellHeight}%`;
                page.style.backgroundImage = `linear-gradient(to right, rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(to bottom, rgba(255,255,255,0.05) 1px, transparent 1px)`;
                applyBackground(blockWrapper, contentArea);
                rescuedBlocks.forEach(rescue => { if (rescue.pageIndex === i) { page.appendChild(rescue.element); if (rescue.element.settings && rescue.element.settings.pages) renderBlockFromTemplate(rescue.element); } });
                btn.onclick = (e) => { e.stopPropagation(); header.querySelectorAll(':scope > .tab-btn').forEach(b => b.classList.remove('active')); pagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(p => p.classList.remove('active')); btn.classList.add('active'); page.classList.add('active'); page.querySelectorAll(':scope > .loaded-block').forEach(child => { if (child.settings && child.settings.pages) renderBlockFromTemplate(child); }); };
                header.appendChild(btn); pagesContainer.appendChild(page);
            }
        }
    }
}

function openSettingsModal(blockWrapper) {
    const modal = document.querySelector('#settings-modal');
    const fieldsContainer = document.querySelector('#modal-fields');
    fieldsContainer.innerHTML = '';

    const speechSection = document.createElement('div');
    speechSection.className = 'setting-section';
    speechSection.innerHTML = '<h4 style="margin:0.5em 0;color:#1ccad8;">Speech Settings</h4>';
    const speechTriggerRow = document.createElement('div'); speechTriggerRow.className = 'setting-row';
    const speechTriggerLabel = document.createElement('label'); speechTriggerLabel.innerText = 'Speech Trigger'; speechTriggerRow.appendChild(speechTriggerLabel);
    const speechTriggerInput = document.createElement('input'); speechTriggerInput.type = 'text';
    speechTriggerInput.value = blockWrapper.settings.speech_trigger || '';
    speechTriggerInput.placeholder = 'e.g. "gear up"';
    speechTriggerInput.oninput = () => { blockWrapper.settings.speech_trigger = speechTriggerInput.value; };
    speechTriggerRow.appendChild(speechTriggerInput);
    speechSection.appendChild(speechTriggerRow);

    const speechAliasesRow = document.createElement('div'); speechAliasesRow.className = 'setting-row';
    const speechAliasesLabel = document.createElement('label'); speechAliasesLabel.innerText = 'Speech Aliases'; speechAliasesRow.appendChild(speechAliasesLabel);
    const speechAliasesInput = document.createElement('input'); speechAliasesInput.type = 'text';
    speechAliasesInput.value = blockWrapper.settings.speech_aliases || '';
    speechAliasesInput.placeholder = 'e.g. "landing gear up, gear up please"';
    speechAliasesInput.oninput = () => { blockWrapper.settings.speech_aliases = speechAliasesInput.value; };
    speechAliasesRow.appendChild(speechAliasesInput);
    speechSection.appendChild(speechAliasesRow);

    const speechTypeRow = document.createElement('div'); speechTypeRow.className = 'setting-row';
    const speechTypeLabel = document.createElement('label'); speechTypeLabel.innerText = 'Trigger Type'; speechTypeRow.appendChild(speechTypeLabel);
    const speechTypeSelect = document.createElement('select');
    ['button', 'slider'].forEach(type => {
        const opt = document.createElement('option'); opt.value = type; opt.innerText = type;
        if ((blockWrapper.settings.speech_type || 'button') === type) opt.selected = true;
        speechTypeSelect.appendChild(opt);
    });
    speechTypeSelect.onchange = () => { blockWrapper.settings.speech_type = speechTypeSelect.value; };
    speechTypeRow.appendChild(speechTypeSelect);
    speechSection.appendChild(speechTypeRow);

    fieldsContainer.appendChild(speechSection);

    const divider = document.createElement('hr');
    divider.style.cssText = 'border:none;border-top:1px solid #444;margin:0.8em 0;';
    fieldsContainer.appendChild(divider);

    Object.keys(blockWrapper.settings).forEach(async key => {
        if (key.startsWith('speech_')) return;
        const value = blockWrapper.settings[key];
        const meta = (blockWrapper.settingsMeta && blockWrapper.settingsMeta[key]) ? blockWrapper.settingsMeta[key] : { type: 'string' };
        const fieldRow = document.createElement('div'); fieldRow.className = 'setting-row';
        const label = document.createElement('label'); label.innerText = (key.charAt(0).toUpperCase() + key.slice(1)).replace(/_/g, ' '); fieldRow.appendChild(label);
        if (meta.type === 'asset') {
            const select = document.createElement('select');
            const assets = await getAvailableAssets();
            select.innerHTML = `<option value="none">None</option>`;
            assets.forEach(asset => { const opt = document.createElement('option'); opt.value = asset; opt.innerText = asset; if (blockWrapper.settings[key] === asset) opt.selected = true; select.appendChild(opt); });
            select.onchange = () => { blockWrapper.settings[key] = select.value; renderBlockFromTemplate(blockWrapper); };
            fieldRow.appendChild(select);
        } else if (meta.type === 'color') {
            const colorContainer = document.createElement('div'); colorContainer.className = 'color-field-container';
            const topRow = document.createElement('div'); topRow.className = 'color-row';
            const colorInput = document.createElement('input'); colorInput.type = 'color';
            const alphaInput = document.createElement('input'); alphaInput.type = 'range'; alphaInput.min = 0; alphaInput.max = 255;
            const hexInput = document.createElement('input'); hexInput.type = 'text'; hexInput.placeholder = '#RRGGBBAA';
            let hex = value.substring(0, 7); let alphaInt = value.length === 9 ? parseInt(value.substring(7, 9), 16) : 255;
            colorInput.value = hex; alphaInput.value = alphaInt; hexInput.value = value.toUpperCase();
            const updateFromControls = () => { const aHex = parseInt(alphaInput.value).toString(16).padStart(2, '0'); const fullHex = (colorInput.value + aHex).toUpperCase(); hexInput.value = fullHex; blockWrapper.settings[key] = fullHex; renderBlockFromTemplate(blockWrapper); };
            const updateFromText = () => { let val = hexInput.value.trim(); if (!val.startsWith('#')) val = '#' + val; if (/^#([A-Fa-f0-9]{3}|[A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$/.test(val)) { if (val.length === 9) { colorInput.value = val.substring(0, 7); alphaInput.value = parseInt(val.substring(7, 9), 16); } else if (val.length === 7) { colorInput.value = val; alphaInput.value = 255; } blockWrapper.settings[key] = val; renderBlockFromTemplate(blockWrapper); } };
            colorInput.oninput = updateFromControls; alphaInput.oninput = updateFromControls; hexInput.oninput = updateFromText;
            topRow.append(colorInput, alphaInput); colorContainer.append(topRow, hexInput); fieldRow.appendChild(colorContainer);
        } else if (meta.type === 'select' && meta.options) {
            const select = document.createElement('select');
            const options = meta.options.split(',').map(o => o.trim());
            options.forEach(opt => { const option = document.createElement('option'); option.value = opt; option.innerText = opt; if (blockWrapper.settings[key] === opt) option.selected = true; select.appendChild(option); });
            select.onchange = () => { blockWrapper.settings[key] = select.value; renderBlockFromTemplate(blockWrapper); };
            fieldRow.appendChild(select);
        } else if (meta.type === 'textarea') {
            const textarea = document.createElement('textarea');
            textarea.value = value;
            textarea.rows = 3;
            textarea.style.width = '100%';
            textarea.style.fontFamily = 'monospace';
            textarea.style.fontSize = '0.8em';
            textarea.oninput = () => { blockWrapper.settings[key] = textarea.value; renderBlockFromTemplate(blockWrapper); };
            fieldRow.appendChild(textarea);
        } else if (meta.type === 'toggle') {
            const toggleContainer = document.createElement('div');
            toggleContainer.style.display = 'flex';
            toggleContainer.style.alignItems = 'center';
            toggleContainer.style.gap = '0.5em';
            const toggle = document.createElement('input');
            toggle.type = 'checkbox';
            toggle.checked = value === 'true' || value === true;
            const toggleLabel = document.createElement('span');
            toggleLabel.innerText = toggle.checked ? 'On' : 'Off';
            toggle.onchange = () => { blockWrapper.settings[key] = toggle.checked ? 'true' : 'false'; toggleLabel.innerText = toggle.checked ? 'On' : 'Off'; renderBlockFromTemplate(blockWrapper); };
            toggleContainer.append(toggle, toggleLabel);
            fieldRow.appendChild(toggleContainer);
        } else {
            const input = document.createElement('input');
            if (meta.type === 'number' || meta.type === 'percentage') { input.type = 'number'; if (meta.min !== undefined) input.min = meta.min; if (meta.max !== undefined) input.max = meta.max; input.value = meta.type === 'percentage' ? value.replace('%', '') : value; }
            else { input.type = 'text'; input.value = value; }
            input.oninput = () => { let newValue = input.value; if (meta.type === 'number' || meta.type === 'percentage') { let num = parseInt(newValue); if (!isNaN(num)) { if (meta.min !== undefined && num < meta.min) num = meta.min; if (meta.max !== undefined && num > meta.max) num = meta.max; newValue = num; } } if (meta.type === 'percentage') newValue += '%'; blockWrapper.settings[key] = newValue; if (key === 'gridx' || key === 'gridy') { renderBlockFromTemplate(blockWrapper); resnapChildrenToGrid(blockWrapper); } else { renderBlockFromTemplate(blockWrapper); } };
            fieldRow.appendChild(input);
        }
        fieldsContainer.appendChild(fieldRow);
    });
    modal.style.display = 'block';
}

function closeSettingsModal() { document.getElementById('settings-modal').style.display = 'none'; }

/**
 * Serializes the current workspace into a v2 panel JSON and saves it to the server.
 * Collects all top-level blocks (with nested children for paged containers),
 * wraps them in a v2 panel object with version, metadata, grid config, and
 * background settings, then POSTs to /api/panel/save.
 */
async function saveWorkspace() {
    const mainContainer = document.querySelector('#maincontainer');
    const bgColorInput = document.getElementById('workspace-bg-color');
    const topLevelElements = mainContainer.querySelectorAll(':scope > .loaded-block');
    const blocks = [];
    topLevelElements.forEach(el => blocks.push(getBlockDataRecursive(el)));

    const now = new Date().toISOString();
    const panelData = {
        version: 2,
        name: "new_panel",
        aspectRatio: "16/9",
        grid: { x: 12, y: 12 },
        backgroundColor: bgColorInput.value,
        backgroundImage: "none",
        backgroundFit: "cover",
        blocks: blocks,
        metadata: { created: now, modified: now }
    };

    let fileName = prompt("Enter panel name:", "new_panel");
    if (!fileName) return;
    fileName = fileName.replace(/\.json$/, '');
    panelData.name = fileName;

    try {
        const res = await fetch('/api/panel/save', {
            method: 'POST',
            headers: addAuthHeaders({ 'Content-Type': 'application/json' }),
            body: JSON.stringify({ fileName, content: panelData })
        });
        if (handleUnauthorized(res)) return;
        const result = await res.json();
        if (result.success) alert("Workspace saved successfully!");
        else alert("Save failed: " + result.error);
    } catch (e) { console.error("Save failed:", e); alert("Save failed"); }
}

function getBlockDataRecursive(block) {
    const blockData = { id: block.id, left: block.style.left, top: block.style.top, width: block.style.width, height: block.style.height, zIndex: block.style.zIndex || 100, settings: block.settings, path: block.dataset.sourcePath, children: [] };
    const pagesContainer = block.querySelector('.pages-container');
    if (!pagesContainer) return blockData;
    pagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(page => {
        const pageIndex = page.dataset.pageIndex;
        const childBlocks = page.querySelectorAll(':scope > .loaded-block');
        if (childBlocks.length > 0) {
            const pageGroup = { pageIndex: parseInt(pageIndex), blocks: [] };
            childBlocks.forEach(child => pageGroup.blocks.push(getBlockDataRecursive(child)));
            blockData.children.push(pageGroup);
        }
    });
    return blockData;
}

/**
 * Prompts the user to select a panel, then loads its v2 JSON definition into the editor.
 * Clears existing blocks, applies background color and settings from the panel data,
 * then recursively renders each block via createBlockRecursive.
 * @param {boolean} firstTime - If true, skips the prompt and loads the default panel.
 */
async function loadWorkspace(firstTime = false) {
    let data = null;
    try {
        const panelsRes = await fetch('/api/panels', { headers: addAuthHeaders() });
        if (handleUnauthorized(panelsRes)) return;
        const panels = await panelsRes.json();
        const panelName = prompt("Select panel to load:", panels.allPanels[0] || "");
        if (!panelName) return;
        currentPanelName = panelName;
        const res = await fetch(`/api/panel/content?name=${encodeURIComponent(panelName)}`, { headers: addAuthHeaders() });
        if (handleUnauthorized(res)) return;
        if (res.ok) data = await res.json();
    } catch (e) { console.error("Failed to load panel:", e); }
    if (!data || !data.blocks) return;
    const mainContainer = document.querySelector('#maincontainer');
    const bgColorInput = document.getElementById('workspace-bg-color');
    const bgColorInputText = document.getElementById('workspace-bg-color-text');
    mainContainer.querySelectorAll(':scope > .loaded-block').forEach(el => el.remove());
    if (data.backgroundColor) { bgColorInput.value = data.backgroundColor; bgColorInputText.value = data.backgroundColor; mainContainer.style.setProperty('--workspace-bg', data.backgroundColor); initWorkspaceSettings(); }
    for (const blockData of data.blocks) {
        await createBlockRecursive(blockData, mainContainer);
    }
    updateLayersTree();
}

async function createBlockRecursive(blockData, parentElement) {
    try {
        const blockUrl = pathToBlockUrl(blockData.path);
        const response = await fetch(blockUrl);
        const rawHtml = await response.text();
        const parser = new DOMParser();
        const doc = parser.parseFromString(rawHtml, 'text/html');
        const settingsTag = doc.querySelector('settings');
        const settingsMeta = {};
        if (!blockData.settings) blockData.settings = {};
        if (settingsTag) {
            for (let attr of settingsTag.attributes) {
                const name = attr.name; const value = attr.value;
                if (name.startsWith('type-')) { const key = name.replace('type-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].type = value; }
                else if (name.startsWith('min-')) { const key = name.replace('min-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].min = value; }
                else if (name.startsWith('max-')) { const key = name.replace('max-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].max = value; }
                else { if (blockData.settings[name] === undefined) blockData.settings[name] = value; }
            }
            settingsTag.remove();
        }
        const blockWrapper = document.createElement('div');
        blockWrapper.classList.add('loaded-block'); blockWrapper.id = blockData.id; blockWrapper.dataset.sourcePath = blockData.path;
        Object.assign(blockWrapper.style, { position: 'absolute', left: blockData.left, top: blockData.top, width: blockData.width, height: blockData.height, zIndex: blockData.zIndex || 100 });
        blockWrapper.settings = blockData.settings; blockWrapper.settingsMeta = settingsMeta; blockWrapper.htmlTemplate = doc.head.innerHTML + doc.body.innerHTML;
        const moveHandle = document.createElement('div'); moveHandle.classList.add('move-handle'); moveHandle.innerHTML = '☩'; moveHandle.draggable = true; blockWrapper.appendChild(moveHandle);
        const settingsBtn = document.createElement('div'); settingsBtn.classList.add('settings-button'); settingsBtn.innerHTML = '⚙'; blockWrapper.appendChild(settingsBtn);
        const resizeHandle = document.createElement('div'); resizeHandle.classList.add('resize-handle'); blockWrapper.appendChild(resizeHandle);
        const contentArea = document.createElement('div'); contentArea.classList.add('block-content-area'); contentArea.style.height = '100%'; blockWrapper.appendChild(contentArea);
        const deleteBtn = document.createElement('div'); deleteBtn.classList.add('delete-button'); deleteBtn.innerHTML = '🗑'; blockWrapper.appendChild(deleteBtn);
        renderBlockFromTemplate(blockWrapper);
        addSelectionListeners(blockWrapper); addWorkspaceDragListeners(blockWrapper, blockWrapper.querySelector('.move-handle')); addResizeListeners(blockWrapper, blockWrapper.querySelector('.resize-handle')); addDeleteFunctionality(blockWrapper, blockWrapper.querySelector('.delete-button'));
        settingsBtn.addEventListener('mousedown', (e) => e.stopPropagation());
        settingsBtn.addEventListener('click', (e) => { e.stopPropagation(); openSettingsModal(blockWrapper); });
        if (blockData.children && blockData.children.length > 0) {
            const pagesContainer = blockWrapper.querySelector('.pages-container');
            if (pagesContainer) {
                for (const pageGroup of blockData.children) {
                    const targetPage = pagesContainer.querySelector(`:scope > .page-wrapper[data-page-index="${pageGroup.pageIndex}"]`);
                    if (targetPage) { for (const childBlockData of pageGroup.blocks) await createBlockRecursive(childBlockData, targetPage); }
                }
            }
        }
        parentElement.appendChild(blockWrapper);
    } catch (error) { console.error(`Failed to reload block:`, error); }
}

function addDeleteFunctionality(blockWrapper, deleteBtn) {
    deleteBtn.addEventListener('mousedown', (e) => e.stopPropagation());
    deleteBtn.addEventListener('click', (e) => { e.stopPropagation(); document.querySelector('#delete-modal').style.display = 'flex'; blockToDelete = blockWrapper; });
}

function initModalListeners() {
    const modal = document.querySelector('#delete-modal');
    document.querySelector('#modal-cancel').onclick = () => { modal.style.display = 'none'; blockToDelete = null; };
    document.querySelector('#modal-confirm').onclick = () => {
        if (blockToDelete) { blockToDelete.style.transition = "all 0.15s ease"; blockToDelete.style.opacity = "0"; blockToDelete.style.transform = "scale(0.95)"; const target = blockToDelete; setTimeout(() => { target.remove(); updateLayersTree(); }, 150); }
        modal.style.display = 'none'; blockToDelete = null;
    };
}

async function getAvailableAssets() {
    try {
        const res = await fetch('/assets/');
        if (!res.ok) return [];
        const text = await res.text();
        const parser = new DOMParser();
        const doc = parser.parseFromString(text, 'text/html');
        const links = doc.querySelectorAll('a[href]');
        return Array.from(links).map(a => a.getAttribute('href')).filter(h => /\.(png|jpg|jpeg|svg|webp|gif)$/i.test(h)).map(h => h.split('/').pop());
    } catch (err) { console.error("Could not read assets", err); return []; }
}

async function applyBackground(blockWrapper, contentArea) {
    const bgFile = blockWrapper.settings['background_image'];
    const target = contentArea.querySelector('.ui-container') || contentArea;
    if (bgFile && bgFile !== 'none') { target.style.backgroundImage = `url('/assets/${bgFile}')`; target.style.backgroundSize = 'cover'; target.style.backgroundPosition = 'center'; }
    else { target.style.backgroundImage = 'none'; }
}

function resnapChildrenToGrid(blockWrapper) {
    const gX = parseInt(blockWrapper.settings.gridx) || 12; const gY = parseInt(blockWrapper.settings.gridy) || 12;
    const stepX = 100 / gX; const stepY = 100 / gY;
    blockWrapper.querySelectorAll('.page-wrapper').forEach(page => {
        page.querySelectorAll(':scope > .loaded-block').forEach(child => {
            const currL = parseFloat(child.style.left); const currT = parseFloat(child.style.top); const currW = parseFloat(child.style.width); const currH = parseFloat(child.style.height);
            let newL = Math.round(currL / stepX) * stepX; let newT = Math.round(currT / stepY) * stepY; let newW = Math.round(currW / stepX) * stepX; let newH = Math.round(currH / stepY) * stepY;
            if (newW < stepX) newW = stepX; if (newH < stepY) newH = stepY;
            if (newL + newW > 100) newL = 100 - newW; if (newT + newH > 100) newT = 100 - newH;
            Object.assign(child.style, { left: `${newL}%`, top: `${newT}%`, width: `${newW}%`, height: `${newH}%` });
        });
    });
}

function updateLayersTree() {
    const toolsMenu = document.querySelector('.tools'); if (!toolsMenu) return;
    toolsMenu.innerHTML = '<h3>Blocks</h3>';
    const rootList = document.createElement('ul'); rootList.className = 'block-list';
    const mainContainer = document.getElementById('maincontainer');
    function getDirectBlockChildren(parentElement) { return Array.from(parentElement.querySelectorAll('.loaded-block')).filter(block => { let parentBlock = block.parentElement.closest('.loaded-block'); return parentElement.id === 'maincontainer' ? !parentBlock : parentBlock === parentElement; }); }
    function buildTree(parentDOMNode, parentUL) {
        getDirectBlockChildren(parentDOMNode).forEach((block) => {
            const li = document.createElement('li'); const label = document.createElement('span'); label.className = 'tree-label';
            if (!block.id) block.id = 'block-' + Math.random().toString(36).substr(2, 9);
            label.dataset.blockTarget = block.id;
            const settings = block.settings || {}; label.textContent = (settings['label'] || block.id || "Unnamed Block");
            label.addEventListener('click', (e) => { e.stopPropagation(); document.querySelectorAll('.loaded-block.selected').forEach(b => b.classList.remove('selected')); document.querySelectorAll('.tree-label.active-layer').forEach(l => l.classList.remove('active-layer')); block.classList.add('selected'); OrderZIndex(block); label.classList.add('active-layer'); let current = block; while (current && current !== mainContainer) { const pageWrapper = current.closest('.page-wrapper'); if (!pageWrapper) break; const pagedBlock = pageWrapper.closest('.loaded-block'); const header = pagedBlock?.querySelector('.tab-header'); const pagesContainer = pageWrapper.parentElement; if (pagedBlock && header && pagesContainer) { const pageIndex = pageWrapper.dataset.pageIndex; header.querySelectorAll(':scope > .tab-btn').forEach(btn => btn.classList.remove('active')); pagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(pw => pw.classList.remove('active')); const buttons = header.querySelectorAll(':scope > .tab-btn'); const targetBtn = buttons[parseInt(pageIndex) - 1]; if (targetBtn) targetBtn.classList.add('active'); pageWrapper.classList.add('active'); pagedBlock.settings.currentPage = parseInt(pageIndex); } current = pagedBlock.parentElement; } if (li.classList.contains('block-folder')) li.classList.toggle('collapsed'); });
            const dupBtn = document.createElement('button'); dupBtn.className = 'layer-dup-btn'; dupBtn.innerHTML = '⧉'; dupBtn.title = "Duplicate Layer"; dupBtn.addEventListener('click', (e) => { e.stopPropagation(); duplicateBlock(block); });
            const subChildren = getDirectBlockChildren(block);
            if (subChildren.length > 0) { li.className = 'block-folder collapsed'; li.appendChild(label); const subUl = document.createElement('ul'); subUl.className = 'block-list'; buildTree(block, subUl); li.appendChild(subUl); li.appendChild(dupBtn); }
            else { li.className = 'block-file'; li.appendChild(label); li.appendChild(dupBtn); }
            parentUL.appendChild(li);
        });
    }
    buildTree(mainContainer, rootList); toolsMenu.appendChild(rootList);
}

function initEditorSelection() {
    const mainContainer = document.getElementById('maincontainer'); if (!mainContainer) return;
    mainContainer.addEventListener('click', (e) => {
        const clickedBlock = e.target.closest('.loaded-block');
        if (clickedBlock) { e.stopPropagation(); const correspondingLabel = document.querySelector(`.tree-label[data-block-target="${clickedBlock.id}"]`); if (correspondingLabel) { correspondingLabel.click(); let parentFolder = correspondingLabel.closest('.block-folder'); while (parentFolder) { parentFolder.classList.remove('collapsed'); parentFolder = parentFolder.parentElement.closest('.block-folder'); } correspondingLabel.scrollIntoView({ behavior: 'smooth', block: 'nearest' }); } }
        else { document.querySelectorAll('.loaded-block.selected').forEach(b => b.classList.remove('selected')); document.querySelectorAll('.tree-label.active-layer').forEach(l => l.classList.remove('active-layer')); }
    });
}

async function duplicateBlock(sourceBlock) {
    try {
        const blockData = { id: 'block-' + Date.now() + Math.random().toString(36).substr(2, 5), path: sourceBlock.dataset.sourcePath, left: (parseInt(sourceBlock.style.left) + 1) + '%', top: (parseInt(sourceBlock.style.top) + 1) + '%', width: sourceBlock.style.width, height: sourceBlock.style.height, zIndex: sourceBlock.style.zIndex, settings: JSON.parse(JSON.stringify(sourceBlock.settings || {})) };
        const settingsMeta = JSON.parse(JSON.stringify(sourceBlock.settingsMeta || {})); const htmlTemplate = sourceBlock.htmlTemplate;
        const blockWrapper = document.createElement('div'); blockWrapper.classList.add('loaded-block'); blockWrapper.id = blockData.id; blockWrapper.dataset.sourcePath = blockData.path;
        Object.assign(blockWrapper.style, { position: 'absolute', left: blockData.left, top: blockData.top, width: blockData.width, height: blockData.height, zIndex: blockData.zIndex });
        blockWrapper.settings = blockData.settings; blockWrapper.settingsMeta = settingsMeta; blockWrapper.htmlTemplate = htmlTemplate;
        const moveHandle = document.createElement('div'); moveHandle.classList.add('move-handle'); moveHandle.innerHTML = '☩'; moveHandle.draggable = true; blockWrapper.appendChild(moveHandle);
        const settingsBtn = document.createElement('div'); settingsBtn.classList.add('settings-button'); settingsBtn.innerHTML = '⚙'; blockWrapper.appendChild(settingsBtn);
        const resizeHandle = document.createElement('div'); resizeHandle.classList.add('resize-handle'); blockWrapper.appendChild(resizeHandle);
        const contentArea = document.createElement('div'); contentArea.classList.add('block-content-area'); contentArea.style.height = '100%'; blockWrapper.appendChild(contentArea);
        const deleteBtn = document.createElement('div'); deleteBtn.classList.add('delete-button'); deleteBtn.innerHTML = '🗑'; blockWrapper.appendChild(deleteBtn);
        renderBlockFromTemplate(blockWrapper); addSelectionListeners(blockWrapper); addWorkspaceDragListeners(blockWrapper, moveHandle); addResizeListeners(blockWrapper, resizeHandle); addDeleteFunctionality(blockWrapper, deleteBtn);
        settingsBtn.addEventListener('mousedown', (e) => e.stopPropagation()); settingsBtn.addEventListener('click', (e) => { e.stopPropagation(); openSettingsModal(blockWrapper); });
        sourceBlock.after(blockWrapper); updateLayersTree();
        setTimeout(() => { const bRect = blockWrapper.getBoundingClientRect(); const target = blockWrapper.querySelector('.move-handle') || blockWrapper; const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, view: window, clientX: bRect.left + bRect.width / 2, clientY: bRect.top + bRect.height / 2 }); target.dispatchEvent(clickEvent); document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected')); blockWrapper.classList.add('selected'); }, 20);
        return blockWrapper;
    } catch (error) { console.error(`Failed to duplicate block:`, error); }
}

function initAspectController() {
    const dropdown = document.getElementById('aspect-preset'); const workspace = document.getElementById('editor-workspace'); const mainContainer = document.getElementById('maincontainer');
    if (!dropdown || !workspace || !mainContainer) return;
    let currentRatio = 16 / 9; let resizeTimer;
    const calculateAndApplySize = () => { const availableW = workspace.clientWidth * 0.95; const availableH = workspace.clientHeight * 0.95; let targetW = availableW; let targetH = targetW / currentRatio; if (targetH > availableH) { targetH = availableH; targetW = targetH * currentRatio; } mainContainer.style.width = `${targetW}px`; mainContainer.style.height = `${targetH}px`; };
    dropdown.addEventListener('change', (e) => { mainContainer.classList.remove('is-resizing'); const [w, h] = e.target.value.split('/').map(Number); currentRatio = w / h; calculateAndApplySize(); });
    const observer = new ResizeObserver(() => { mainContainer.classList.add('is-resizing'); calculateAndApplySize(); clearTimeout(resizeTimer); resizeTimer = setTimeout(() => { mainContainer.classList.remove('is-resizing'); }, 150); });
    observer.observe(workspace);
    const [startW, startH] = dropdown.value.split('/').map(Number); currentRatio = startW / startH; calculateAndApplySize();
}

function initWorkspaceSettings() {
    const bgColorInput = document.getElementById('workspace-bg-color'); const bgColorInputText = document.getElementById('workspace-bg-color-text'); const mainContainer = document.getElementById('maincontainer');
    if (!bgColorInput || !mainContainer || !bgColorInputText) return;
    bgColorInput.addEventListener('input', (e) => { const color = e.target.value; mainContainer.style.setProperty('--workspace-bg', color); bgColorInputText.value = color; });
    bgColorInputText.addEventListener('input', (e) => { const color = e.target.value; mainContainer.style.setProperty('--workspace-bg', color); bgColorInput.value = color; });
}

function showBindingsPopup() {
    const blocks = document.querySelectorAll('.loaded-block');
    const joystickMap = {};
    blocks.forEach(block => {
        if (!block.settings) return;
        let jsIndex = block.settings.joystick;
        if (jsIndex !== undefined && jsIndex !== '') {
            jsIndex = parseInt(jsIndex);
            if (!joystickMap[jsIndex]) joystickMap[jsIndex] = { buttons: {}, sliders: {} };
            const displayName = block.settings.label ? block.settings.label : block.id;
            let btnIndex = block.settings.button;
            if (btnIndex !== undefined && btnIndex !== '') {
                btnIndex = parseInt(btnIndex);
                if (joystickMap[jsIndex].buttons[btnIndex]) joystickMap[jsIndex].buttons[btnIndex] += `, ${displayName}`;
                else joystickMap[jsIndex].buttons[btnIndex] = displayName;
            }
            let sldIndex = block.settings.slider;
            if (sldIndex !== undefined && sldIndex !== '') {
                sldIndex = parseInt(sldIndex);
                if (joystickMap[jsIndex].sliders[sldIndex]) joystickMap[jsIndex].sliders[sldIndex] += `, ${displayName}`;
                else joystickMap[jsIndex].sliders[sldIndex] = displayName;
            }
        }
    });
    const overlay = document.createElement('div'); overlay.id = 'bindings-overlay';
    let bodyHtml = ''; const activeJoysticks = Object.keys(joystickMap).map(Number).sort((a, b) => a - b);
    if (activeJoysticks.length === 0) bodyHtml = '<p style="text-align:center; color:#aaa;">No inputs are currently allocated in this panel.</p>';
    else { for (const jsIndex of activeJoysticks) { bodyHtml += `<div class="joystick-section"><h3>Virtual Joystick ${jsIndex}</h3>`; bodyHtml += `<div class="binding-group-title">Sliders / Axes</div><div class="button-grid">`; for (let i = 0; i < 8; i++) { const isUsed = joystickMap[jsIndex].sliders[i] !== undefined; const slotClass = isUsed ? 'used slider-used' : 'unallocated'; const displayName = isUsed ? joystickMap[jsIndex].sliders[i] : 'Unallocated'; bodyHtml += `<div class="binding-slot ${slotClass}"><span class="btn-num">Slider ${i}</span><span class="btn-name">${displayName}</span></div>`; } bodyHtml += `</div>`; bodyHtml += `<div class="binding-group-title">Buttons</div><div class="button-grid">`; for (let i = 0; i < 16; i++) { const isUsed = joystickMap[jsIndex].buttons[i] !== undefined; const slotClass = isUsed ? 'used' : 'unallocated'; const displayName = isUsed ? joystickMap[jsIndex].buttons[i] : 'Unallocated'; bodyHtml += `<div class="binding-slot ${slotClass}"><span class="btn-num">Button ${i}</span><span class="btn-name">${displayName}</span></div>`; } bodyHtml += `</div></div>`; } }
    overlay.innerHTML = `<div class="bindings-modal" onclick="event.stopPropagation()"><div class="bindings-modal-header"><h2>Input Allocations</h2><button class="close-bindings-btn" id="close-bindings-modal">✖</button></div><div class="bindings-modal-body">${bodyHtml}</div></div>`;
    overlay.querySelector('#close-bindings-modal').addEventListener('click', () => overlay.remove());
    overlay.addEventListener('click', () => overlay.remove());
    document.body.appendChild(overlay);
}
