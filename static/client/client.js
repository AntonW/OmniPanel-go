// OmniPanel-go Panel Client — Speech Recognition Module
//
// Handles client-side speech recognition features:
// - Push-to-talk recording: captures audio from browser microphone, sends PCM
//   chunks to server via WebSocket binary frames
// - Wake word detection: uses Web Speech API (Chrome/Edge only) to listen for
//   a wake word, then records follow-up speech
// - Host recording control: sends start/stop signals to server when
//   recording_location is "host" (server handles microphone capture)
// - Speech trigger registration: sends block phrases/aliases to server so
//   the Vosk grammar can be updated dynamically
// - TTS feedback: speaks confirmation text after command execution
// - Visual feedback: toast notifications and block flash on match

let socket;
let reconnectInterval;
let commandHoldIntervals = {};
/**
 * Global map tracking which RSS entry GUIDs each client has seen.
 * Keys are block IDs, values are Sets of entry GUIDs.
 * Used to determine which entries should be highlighted as "new".
 */
let rssSeenEntries = {};

// Tracks available MPRIS players for source selection tabs.
let mprisAvailablePlayers = [];
let mprisSelectedPlayer = '';

// Maps keyboard shortcut strings (e.g., "ctrl+a", "q") to block actions.
// Populated by enableInputs() for any button/slider with a keyboard-key setting.
let keyboardShortcutMap = {};

// Tracks which keyboard shortcuts are currently held down to prevent repeat events.
let activeKeyboardKeys = new Set();

// Speech configuration received from server on connect.
// Controls recording behavior, trigger mode, and TTS.
let speechConfig = {
    enabled: false,
    recordingLocation: 'client',
    triggerMode: 'push-to-talk',
    wakeWord: 'omnipanel-go',
    ttsEnabled: true,
    wakeWordListenSec: 8,
};

let mediaRecorder = null;
let audioChunks = [];
let isRecording = false;
let speechRecognition = null;
let wakeWordActive = false;
let recordingLocation = 'client';

let audioContext = null;
let audioStream = null;
let scriptProcessor = null;
let pcmBuffer = [];

/**
 * Loads a v2 panel definition and renders all blocks into the client viewport.
 * Applies the panel background color and theme, then iterates over the blocks array
 * to fetch each block's HTML template, parse its <settings> tag, load the block's
 * theme CSS (scoped to the block's DOM element ID), and render it with saved settings
 * and position data. Theme resolution: block.theme → data.theme → 'default'.
 * @param {Object} data - V2 panel JSON with blocks, backgroundColor, theme, etc.
 */
async function loadPanel(data) {
    const mainContainer = document.querySelector('body');
    mainContainer.innerHTML = "";

    const blocks = data.blocks || [];
    if (data.backgroundColor) {
        mainContainer.style.setProperty('--workspace-bg', data.backgroundColor);
    }
    
    // Store panel theme globally for blocks to use
    window.currentPanelTheme = data.theme || 'default';
    
    for (const blockData of blocks) {
        await renderBlockRecursive(blockData, mainContainer);
    }

    enableInputs();
}

/**
 * Recursively renders a block and its children from saved JSON data.
 * Fetches the block HTML template from /blocks/, parses the <settings> tag
 * to rebuild settingsMeta (types, min/max, options), then merges any saved
 * settingsMeta from the JSON (for dynamically added parameters with custom
 * types like toggle). Applies saved settings and position data, then calls
 * renderBlockFromTemplate to load theme CSS and inject scoped styles.
 * Processes nested children in paged containers.
 * @param {Object} blockData - Saved block JSON with id, path, theme, settings, settingsMeta, position, size, children.
 * @param {HTMLElement} parentElement - DOM element to append the rendered block to.
 */
async function renderBlockRecursive(blockData, parentElement) {
    try {
        const fileName = blockData.path.split('blocks').pop().replace(/\\/g, '/');
        const url = `/blocks/${fileName}`;
        const response = await fetch(url);
        const rawHtml = await response.text();

        const parser = new DOMParser();
        const doc = parser.parseFromString(rawHtml, 'text/html');
        const settingsTag = doc.querySelector('settings');

        const settingsMeta = {};
        if (settingsTag) {
            for (let attr of settingsTag.attributes) {
                const name = attr.name;
                const value = attr.value;
                if (name.startsWith('type-')) { const key = name.replace('type-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].type = value; }
                else if (name.startsWith('min-')) { const key = name.replace('min-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].min = value; }
                else if (name.startsWith('max-')) { const key = name.replace('max-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].max = value; }
                else if (name.startsWith('options-')) { const key = name.replace('options-', ''); if (!settingsMeta[key]) settingsMeta[key] = {}; settingsMeta[key].options = value; }
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
        blockWrapper.id = blockData.id;
        blockWrapper.settingsMeta = settingsMeta;
        blockWrapper.theme = blockData.theme || window.currentPanelTheme || 'default';

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

        const contentArea = document.createElement('div');
        contentArea.classList.add('block-content-area');
        contentArea.style.width = '100%';
        contentArea.style.height = '100%';
        blockWrapper.appendChild(contentArea);

        await renderBlockFromTemplate(blockWrapper);

        if (blockData.children && blockData.children.length > 0) {
            for (const pageGroup of blockData.children) {
                const pagesContainer = contentArea.querySelector('.pages-container');
                if (pagesContainer) {
                    const targetPage = pagesContainer.querySelector(`:scope > .page-wrapper[data-page-index="${pageGroup.pageIndex}"]`);

                    if (targetPage) {
                        for (const childBlock of pageGroup.blocks) {
                            await renderBlockRecursive(childBlock, targetPage);
                        }
                    }
                }
            }
        }

        parentElement.appendChild(blockWrapper);

    } catch (e) {
        console.error("Recursive Load Error:", e);
    }
}

/**
 * Renders a block from its parsed HTML template. Loads the block's theme CSS
 * from /themes/<name>.css, removes comments, scopes all selectors to the block's
 * DOM element ID, and injects the result as a <style> tag before the block HTML.
 * This ensures each block has isolated styling that doesn't conflict with other blocks.
 * @param {HTMLElement} blockWrapper - The block DOM element with settings, theme, and htmlTemplate.
 */
async function renderBlockFromTemplate(blockWrapper) {
    const contentArea = blockWrapper.querySelector('.block-content-area');
    if (!contentArea) return;

    const rescuedBlocks = [];
    const existingPagesContainer = contentArea.querySelector('.pages-container');
    if (existingPagesContainer) {
        existingPagesContainer.querySelectorAll(':scope > .page-wrapper').forEach(page => {
            const pIndex = parseInt(page.dataset.pageIndex);
            const blocks = Array.from(page.querySelectorAll(':scope > .loaded-block'));
            blocks.forEach(b => {
                rescuedBlocks.push({ pageIndex: pIndex, element: b });
            });
        });
    }

    let finalHtml = blockWrapper.htmlTemplate;
    const blockId = blockWrapper.id;
    
    // Load and inject theme CSS with proper scoping
    const theme = blockWrapper.theme || window.currentPanelTheme || 'default';
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

    contentArea.innerHTML = finalHtml;

    renderCommandParams(blockWrapper);
    applyConditionalDisplay(blockWrapper, contentArea);

    if (blockWrapper.settings.pages) {
        applyBackground(blockWrapper, contentArea);

        const header = contentArea.querySelector('.tab-header');
        const container = contentArea.querySelector('.pages-container');

        if (header && container) {
            rebuildPages(blockWrapper, header, container, blockWrapper.settings.pages);
        }
    }

    requestAnimationFrame(() => {
        const hitbox = blockWrapper.querySelector('.joy-hitbox');
        if (hitbox) {
            initJoystick(blockWrapper);
        }
        const mousepadHitbox = blockWrapper.querySelector('.mousepad-hitbox');
        if (mousepadHitbox) {
            initMousepad(blockWrapper);
        }

        const metricBlock = blockWrapper.querySelector('[data-key]');
        if (metricBlock) {
            metricBlock.dataset.decimalPlaces = blockWrapper.settings['decimal_places'] || '1';
        }

        if (blockWrapper.settings.speech_trigger) {
            registerBlockSpeechTrigger(blockWrapper);
        }

        const pttBlock = blockWrapper.querySelector('[data-ptt]');
        if (pttBlock) {
            initPushToTalk(pttBlock);
        }

        const rssFeedBlock = blockWrapper.querySelector('.rss-feed');
        if (rssFeedBlock) {
            initRSSFeed(blockWrapper);
        }

        if (blockWrapper.settings.control_mode === 'mpris') {
            renderMediaSourceTabs();
        }

        const stratagemBtn = blockWrapper.querySelector('.sequence-button button');
        if (stratagemBtn) {
            initSequenceButton(blockWrapper, stratagemBtn);
        }

        const simpleBtn = blockWrapper.querySelector('.simple-button button');
        if (simpleBtn) {
            initButtonLayout(blockWrapper, simpleBtn);
        }

        const cmdBtn = blockWrapper.querySelector('.command-button');
        if (cmdBtn) {
            initButtonLayout(blockWrapper, cmdBtn);
        }
    });
}

/**
 * Renders source selection tabs on media player blocks when multiple MPRIS players
 * are available. Called during block rendering and whenever the available players
 * list changes via data-update.
 *
 * When there is only one (or zero) players, the overlay is hidden. When there are
 * two or more, pill-style tabs are rendered inside the .media-source-tabs-overlay
 * element on the cover art. Clicking a tab sends POST /api/mpris/select to switch
 * the active player, which triggers a data-update with the new player's state.
 */
function renderMediaSourceTabs() {
    const mediaBlocks = document.querySelectorAll('.media-player[data-control-mode="mpris"]');
    mediaBlocks.forEach(block => {
        const overlay = block.querySelector('.media-source-tabs-overlay');
        if (!overlay) return;

        if (mprisAvailablePlayers.length <= 1) {
            overlay.innerHTML = '';
            overlay.style.display = 'none';
            return;
        }

        overlay.style.display = 'flex';
        overlay.innerHTML = '';

        mprisAvailablePlayers.forEach(player => {
            const tab = document.createElement('button');
            tab.className = 'media-source-tab';
            tab.textContent = player.identity || player.name;
            tab.dataset.player = player.name;

            if (player.name === mprisSelectedPlayer) {
                tab.classList.add('active');
            }

            tab.addEventListener('click', async (e) => {
                e.stopPropagation();
                e.preventDefault();

                try {
                    await fetch('/api/mpris/select', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ player: player.name })
                    });
                    mprisSelectedPlayer = player.name;

                    overlay.querySelectorAll('.media-source-tab').forEach(t => t.classList.remove('active'));
                    tab.classList.add('active');
                } catch (err) {
                    console.error('Failed to select player:', err);
                }
            });

            overlay.appendChild(tab);
        });
    });
}

/**
 * Registers a block's speech trigger with the server.
 * Sends the phrase, aliases, trigger type, joystick index, button ID, and
 * axis ID so the server can update the Vosk grammar dynamically and execute
 * button/slider commands directly without requiring the client to send a
 * follow-up simulate-button/slider message.
 * Called during block rendering when a block has a speech_trigger setting.
 * @param {HTMLElement} blockWrapper - The block DOM element with settings
 */
function registerBlockSpeechTrigger(blockWrapper) {
    const phrase = blockWrapper.settings.speech_trigger;
    const aliases = blockWrapper.settings.speech_aliases ? blockWrapper.settings.speech_aliases.split(',').map(s => s.trim()).filter(s => s) : [];
    const triggerType = blockWrapper.settings.speech_type || 'button';
    const joystickIndex = parseInt(blockWrapper.settings.joystick) || 0;

    let buttonId = 0;
    let axisId = 0;

    if (triggerType === 'button') {
        const btn = blockWrapper.querySelector('[emulate-button]');
        if (btn) {
            buttonId = parseInt(btn.getAttribute('emulate-button')) || 0;
        }
    } else if (triggerType === 'slider') {
        const slider = blockWrapper.querySelector('[emulate-slider]');
        if (slider) {
            axisId = parseInt(slider.getAttribute('emulate-slider')) || 0;
        }
    }

    if (socket && socket.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({
            type: 'register-speech-trigger',
            data: {
                block_id: blockWrapper.id,
                phrase: phrase,
                aliases: aliases,
                type: triggerType,
                joystick_index: joystickIndex,
                button_id: buttonId,
                axis_id: axisId
            }
        }));
    }
}

/**
 * Initializes a sequence button block by resolving the icon URL and
 * configuring the layout (image-only, text-only, image-text) and text
 * position (top, bottom, left, right).
 *
 * Called during block rendering when a .sequence-button element is detected.
 * The icon URL is normalized to /assets/<theme>/<icon> so the browser can
 * load it from the server's asset directory.
 *
 * @param {HTMLElement} blockWrapper - The loaded-block element with settings.
 * @param {HTMLButtonElement} btn - The button element inside the block.
 */
function initSequenceButton(blockWrapper, btn) {
    const settings = blockWrapper.settings;
    const iconUrl = settings['icon_url'];
    const layout = settings['layout'] || 'image-text';
    const textPosition = settings['text_position'] || 'bottom';
    const iconEl = blockWrapper.querySelector('.btn-icon');
    const labelEl = blockWrapper.querySelector('.btn-label');
    const contentEl = blockWrapper.querySelector('.btn-content');

    if (contentEl) {
        contentEl.className = `btn-content layout-${layout} text-${textPosition}`;
    }

    if (iconEl) {
        if (iconUrl && iconUrl.trim() !== '' && !iconUrl.startsWith('settings-')) {
            const assetsDir = '/assets';
            const normalizedPath = iconUrl.startsWith('/') ? iconUrl : `${assetsDir}/${iconUrl}`;
            iconEl.src = normalizedPath;
            iconEl.style.display = (layout === 'image-text' || layout === 'image-only') ? '' : 'none';
        } else {
            iconEl.style.display = 'none';
        }
    }

    if (labelEl) {
        labelEl.style.display = (layout === 'text-only' || layout === 'image-text') ? '' : 'none';
    }
}

/**
 * Initializes icon and layout for standard button and command blocks.
 * Reuses the same pattern as initSequenceButton for consistency across
 * all button-type blocks.
 *
 * Reads layout ("image-only", "text-only", "image-text") and text_position
 * ("top", "bottom", "left", "right") from block settings. Resolves the
 * icon URL to /assets/<path> and toggles visibility of .btn-icon and
 * .btn-label based on the layout mode.
 *
 * @param {HTMLElement} blockWrapper - The loaded-block element with settings.
 * @param {HTMLButtonElement} btn - The button element inside the block.
 */
function initButtonLayout(blockWrapper, btn) {
    const settings = blockWrapper.settings;
    const iconUrl = settings['icon_url'];
    const layout = settings['layout'] || 'text-only';
    const textPosition = settings['text_position'] || 'bottom';
    const iconEl = blockWrapper.querySelector('.btn-icon');
    const labelEl = blockWrapper.querySelector('.btn-label');
    const contentEl = blockWrapper.querySelector('.btn-content');

    if (contentEl) {
        contentEl.className = `btn-content layout-${layout} text-${textPosition}`;
    }

    if (iconEl) {
        if (iconUrl && iconUrl.trim() !== '' && !iconUrl.startsWith('settings-')) {
            const assetsDir = '/assets';
            const normalizedPath = iconUrl.startsWith('/') ? iconUrl : `${assetsDir}/${iconUrl}`;
            iconEl.src = normalizedPath;
            iconEl.style.display = (layout === 'image-text' || layout === 'image-only') ? '' : 'none';
        } else {
            iconEl.style.display = 'none';
        }
    }

    if (labelEl) {
        labelEl.style.display = (layout === 'text-only' || layout === 'image-text') ? '' : 'none';
    }
}

function initJoystick(blockWrapper) {
    const hitbox = blockWrapper.querySelector('.joy-hitbox');
    const base = blockWrapper.querySelector('.joy-base');
    const thumb = blockWrapper.querySelector('.joy-thumb');

    if (!hitbox || !base || !thumb) {
        console.warn("Joystick elements missing in block:", blockWrapper.id);
        return;
    }

    const maxTravel = parseInt(blockWrapper.settings.travel) || 60;
    const safeZone = parseInt(blockWrapper.settings.safe_zone) || 30;

    let active = false;
    let centerX, centerY;
    let activePointerId = null;

    hitbox.style.touchAction = "none";

    function handlePointerStart(e) {
        e.stopPropagation();
        e.preventDefault();

        const rect = hitbox.getBoundingClientRect();
        active = true;
        activePointerId = e.pointerId;

        centerX = e.clientX - rect.left;
        centerY = e.clientY - rect.top;

        base.style.display = 'block';
        base.style.left = `${centerX}px`;
        base.style.top = `${centerY}px`;
    }

    function handlePointerMove(e) {
        if (!active || e.pointerId !== activePointerId) return;
        e.preventDefault();

        const rect = hitbox.getBoundingClientRect();
        const dx = Math.max(-maxTravel, Math.min(maxTravel, (e.clientX - rect.left) - centerX));
        const dy = Math.max(-maxTravel, Math.min(maxTravel, (e.clientY - rect.top) - centerY));

        thumb.style.transform = `translate(calc(-50% + ${dx}px), calc(-50% + ${dy}px))`;

        blockWrapper.dataset.joyX = Math.round(((dx / maxTravel + 1) / 2) * 255);
        blockWrapper.dataset.joyY = Math.round((((dy / maxTravel) + 1) / 2) * 255);

        const payload = {
            type: 'simulate-joystick',
            data: {
                js: blockWrapper.settings['joystick'],
                id: blockWrapper.settings['slider'],
                value: { x: parseInt(blockWrapper.dataset.joyX), y: parseInt(blockWrapper.dataset.joyY) }
            }
        };
        socket.send(JSON.stringify(payload));
    }

    function handlePointerEnd(e) {
        if (!active) return;
        active = false;
        activePointerId = null;
        base.style.display = 'none';
        blockWrapper.dataset.joyX = 127;
        blockWrapper.dataset.joyY = 127;
        thumb.style.transform = `translate(-50%, -50%)`;

        const payload = {
            type: 'simulate-joystick',
            data: {
                js: blockWrapper.settings['joystick'],
                id: blockWrapper.settings['slider'],
                value: { x: parseInt(blockWrapper.dataset.joyX), y: parseInt(blockWrapper.dataset.joyY) }
            }
        };
        socket.send(JSON.stringify(payload));
    }

    hitbox.addEventListener('pointerdown', handlePointerStart);
    hitbox.addEventListener('pointermove', handlePointerMove);
    hitbox.addEventListener('pointerup', handlePointerEnd);
    hitbox.addEventListener('pointercancel', handlePointerEnd);
}

function initMousepad(blockWrapper) {
    const hitbox = blockWrapper.querySelector('.mousepad-hitbox');
    const cursor = blockWrapper.querySelector('.mousepad-cursor');
    const leftBtn = blockWrapper.querySelector('.mousepad-btn.left');
    const rightBtn = blockWrapper.querySelector('.mousepad-btn.right');

    if (!hitbox) {
        console.warn("Mousepad hitbox missing in block:", blockWrapper.id);
        return;
    }

    console.log("Mousepad initialized for block:", blockWrapper.id);

    const sensitivity = parseInt(blockWrapper.settings.sensitivity) || 5;
    const mousepadIndex = parseInt(blockWrapper.settings.mousepad) || 0;

    console.log("Mousepad settings:", { sensitivity, mousepadIndex });

    let active = false;
    let activePointerId = null;
    let lastX = 0;
    let lastY = 0;

    hitbox.style.touchAction = "none";

    function handlePointerStart(e) {
        e.stopPropagation();
        e.preventDefault();

        console.log("Mousepad pointer down:", e.pointerId);
        active = true;
        activePointerId = e.pointerId;
        lastX = e.clientX;
        lastY = e.clientY;

        if (cursor) cursor.classList.add('active');
    }

    function handlePointerMove(e) {
        if (!active || e.pointerId !== activePointerId) return;
        e.preventDefault();

        const dx = Math.round((e.clientX - lastX) * sensitivity);
        const dy = Math.round((e.clientY - lastY) * sensitivity);

        lastX = e.clientX;
        lastY = e.clientY;

        if (dx !== 0 || dy !== 0) {
            const payload = {
                type: 'simulate-mousepad',
                data: {
                    js: mousepadIndex,
                    value: { x: dx, y: dy }
                }
            };
            console.log("Sending mousepad move:", payload);
            socket.send(JSON.stringify(payload));
        }

        if (cursor) {
            const rect = hitbox.getBoundingClientRect();
            const cursorX = e.clientX - rect.left;
            const cursorY = e.clientY - rect.top;
            cursor.style.left = `${cursorX}px`;
            cursor.style.top = `${cursorY}px`;
        }
    }

    function handlePointerEnd(e) {
        if (!active) return;
        console.log("Mousepad pointer up");
        active = false;
        activePointerId = null;
        if (cursor) cursor.classList.remove('active');
    }

    hitbox.addEventListener('pointerdown', handlePointerStart);
    hitbox.addEventListener('pointermove', handlePointerMove);
    hitbox.addEventListener('pointerup', handlePointerEnd);
    hitbox.addEventListener('pointercancel', handlePointerEnd);

    hitbox.addEventListener('wheel', (e) => {
        e.preventDefault();
        e.stopPropagation();
        const delta = Math.sign(e.deltaY) * -1;
        const payload = {
            type: 'simulate-mousewheel',
            data: {
                js: mousepadIndex,
                delta: delta
            }
        };
        socket.send(JSON.stringify(payload));
    }, { passive: false });

    if (leftBtn) {
        leftBtn.addEventListener('pointerdown', (e) => {
            e.stopPropagation();
            e.preventDefault();
            leftBtn.setPointerCapture(e.pointerId);
            leftBtn.classList.add('active');
            socket.send(JSON.stringify({
                type: 'simulate-mousebtn',
                data: { js: mousepadIndex, btn: 'left', state: 1 }
            }));
        });
        leftBtn.addEventListener('pointerup', () => {
            leftBtn.classList.remove('active');
            socket.send(JSON.stringify({
                type: 'simulate-mousebtn',
                data: { js: mousepadIndex, btn: 'left', state: 0 }
            }));
        });
        leftBtn.addEventListener('pointercancel', () => {
            leftBtn.classList.remove('active');
            socket.send(JSON.stringify({
                type: 'simulate-mousebtn',
                data: { js: mousepadIndex, btn: 'left', state: 0 }
            }));
        });
    }

    if (rightBtn) {
        rightBtn.addEventListener('pointerdown', (e) => {
            e.stopPropagation();
            e.preventDefault();
            rightBtn.setPointerCapture(e.pointerId);
            rightBtn.classList.add('active');
            socket.send(JSON.stringify({
                type: 'simulate-mousebtn',
                data: { js: mousepadIndex, btn: 'right', state: 1 }
            }));
        });
        rightBtn.addEventListener('pointerup', () => {
            rightBtn.classList.remove('active');
            socket.send(JSON.stringify({
                type: 'simulate-mousebtn',
                data: { js: mousepadIndex, btn: 'right', state: 0 }
            }));
        });
        rightBtn.addEventListener('pointercancel', () => {
            rightBtn.classList.remove('active');
            socket.send(JSON.stringify({
                type: 'simulate-mousebtn',
                data: { js: mousepadIndex, btn: 'right', state: 0 }
            }));
        });
    }
}

function initPushToTalk(pttBlock) {
    pttBlock.addEventListener('pointerdown', (e) => {
        e.preventDefault();
        e.stopPropagation();
        if (speechConfig.enabled && !isRecording && !(speechConfig.triggerMode === 'wake-word' && recordingLocation === 'host')) {
            startPushToTalkRecording();
            pttBlock.classList.add('recording');
        }
        pttBlock.setPointerCapture(e.pointerId);
    });

    pttBlock.addEventListener('pointerup', (e) => {
        e.preventDefault();
        e.stopPropagation();
        stopRecording();
        pttBlock.classList.remove('recording');
    });

    pttBlock.addEventListener('pointercancel', () => {
        stopRecording();
        pttBlock.classList.remove('recording');
    });
}

async function applyBackground(blockWrapper, contentArea) {
    const bgFile = blockWrapper.settings['background_image'];
    const target = contentArea.querySelector('.ui-container') || contentArea;

    if (bgFile && bgFile !== 'none') {
        const assetsDir = '/assets';

        const normalizedPath = `${assetsDir}/${bgFile}`.replace(/\\/g, '/');
        const fullPath = `url('${normalizedPath}')`;

        target.style.backgroundImage = fullPath;
        target.style.backgroundSize = 'cover';
        target.style.backgroundPosition = 'center';
    } else {
        target.style.backgroundImage = 'none';
    }
}

function applyConditionalDisplay(blockWrapper, contentArea) {
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

function rebuildPages(blockWrapper, header, container, pagesSetting) {
    header.innerHTML = '';

    const pagesVal = pagesSetting.toString();
    let pageNames = [];
    let count = 0;

    if (pagesVal.includes(',')) {
        pageNames = pagesVal.split(',').map(s => s.trim());
        count = pageNames.length;
    } else if (!isNaN(pagesVal) && pagesVal.trim() !== "") {
        count = parseInt(pagesVal);
        for (let i = 1; i <= count; i++) pageNames.push(`Page ${i}`);
    } else {
        pageNames = [pagesVal];
        count = 1;
    }

    const rescued = [];
    container.querySelectorAll(':scope > .page-wrapper').forEach(page => {
        const idx = parseInt(page.dataset.pageIndex);
        const children = Array.from(page.querySelectorAll(':scope > .loaded-block'));
        children.forEach(child => rescued.push({ pageIndex: idx, element: child }));
    });

    container.innerHTML = '';

    if (!blockWrapper.currentPage) blockWrapper.currentPage = 1;

    for (let i = 1; i <= count; i++) {
        const isActive = (i === blockWrapper.currentPage);

        const btn = document.createElement('button');
        btn.className = `tab-btn ${isActive ? 'active' : ''}`;

        btn.innerText = pageNames[i - 1] || `Page ${i}`;

        const page = document.createElement('div');
        page.className = `page-wrapper ${isActive ? 'active' : ''}`;
        page.dataset.pageIndex = i;

        rescued.forEach(item => {
            if (item.pageIndex === i) {
                page.appendChild(item.element);
            }
        });

        btn.onclick = (e) => {
            if (e) e.stopPropagation();
            blockWrapper.currentPage = i;
            header.querySelectorAll(':scope > .tab-btn').forEach(b => b.classList.remove('active'));
            container.querySelectorAll(':scope > .page-wrapper').forEach(p => p.classList.remove('active'));
            btn.classList.add('active');
            page.classList.add('active');
        };

        header.appendChild(btn);
        container.appendChild(page);
    }
}

function renderCommandParams(blockWrapper) {
    const paramsContainer = blockWrapper.querySelector('.params-container');
    if (!paramsContainer) return;

    const paramKeys = Object.keys(blockWrapper.settings).filter(k => k.startsWith('param_'));
    if (paramKeys.length === 0) {
        paramsContainer.style.display = 'none';
        return;
    }

    paramsContainer.innerHTML = '';
    blockWrapper.paramValues = {};

    paramKeys.forEach(key => {
        const paramName = key.replace('param_', '');
        const value = blockWrapper.settings[key];
        const meta = (blockWrapper.settingsMeta && blockWrapper.settingsMeta[key]) || { type: 'text' };

        const group = document.createElement('div');
        group.className = 'param-group';

        const label = document.createElement('span');
        label.className = 'param-label';
        label.textContent = paramName;
        group.appendChild(label);

        let input;
        if (meta.type === 'number' || meta.type === 'percentage') {
            input = document.createElement('input');
            input.type = 'number';
            input.value = meta.type === 'percentage' ? value.replace('%', '') : value;
            if (meta.min !== undefined) input.min = meta.min;
            if (meta.max !== undefined) input.max = meta.max;
        } else if (meta.type === 'toggle') {
            input = document.createElement('input');
            input.type = 'checkbox';
            input.checked = value === 'true' || value === true;
        } else {
            input = document.createElement('input');
            input.type = 'text';
            input.value = value;
        }

        input.className = 'param-input';
        input.dataset.paramKey = key;

        input.addEventListener('input', () => {
            if (input.type === 'checkbox') {
                blockWrapper.paramValues[key] = input.checked ? 'true' : 'false';
            } else {
                blockWrapper.paramValues[key] = input.value;
            }
        });

        blockWrapper.paramValues[key] = input.type === 'checkbox' ? (input.checked ? 'true' : 'false') : input.value;

        group.appendChild(input);
        paramsContainer.appendChild(group);
    });
}

function handleCommandResult(data) {
    const blockId = data.block_id;
    const success = data.success;
    const output = data.output || '';

    const block = document.getElementById(blockId);
    if (!block) return;

    const btn = block.querySelector('.command-button');
    if (!btn) return;

    btn.classList.remove('success', 'error');

    if (success) {
        btn.classList.add('success');
    } else {
        btn.classList.add('error');
    }

    setTimeout(() => {
        btn.classList.remove('success', 'error');
    }, 1500);

    if (output.trim()) {
        showCommandToast(output.trim(), success);
    }
}

/**
 * Processes data-update WebSocket messages and updates DOM elements.
 * Handles two block types:
 * 1. Generic data-display blocks: updates .data-value and .data-unit elements
 *    for any block with a [data-key] attribute matching a DataBus key.
 * 2. Media player blocks: updates cover art, title, artist, progress bar,
 *    and play/pause icon based on the block's data-control-mode attribute.
 *    - "mpris" mode: reads from mpris_* DataBus keys, rewrites file:// URLs
 *      to /api/mpris/cover?url=... so browsers can load local cover art.
 *    - "keyboard" mode: reads from custom data-media-* attribute keys.
 *
 * Also processes MPRIS player list updates:
 *    - mpris_available_players: JSON array of {name, identity} objects. When
 *      the list changes, renderMediaSourceTabs() is called to show/hide
 *      source selection tabs on the cover art overlay.
 *    - mpris_player_name: tracks the currently selected player for tab highlighting.
 *
 * @param {Object} data - Snapshot of the DataBus from a data-update message.
 */
function handleDataUpdate(data) {
    // Update generic data-display blocks
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

    // Update media player blocks
    const mediaBlocks = document.querySelectorAll('.media-player');
    mediaBlocks.forEach(block => {
        const controlMode = block.getAttribute('data-control-mode');
        const coverKey = block.getAttribute('data-media-cover');
        const titleKey = block.getAttribute('data-media-title');
        const artistKey = block.getAttribute('data-media-artist');
        const progressKey = block.getAttribute('data-media-progress');

        // Parse available players list and render source tabs
        if (data['mpris_available_players']) {
            try {
                const players = JSON.parse(data['mpris_available_players'].value);
                if (Array.isArray(players)) {
                    const changed = players.length !== mprisAvailablePlayers.length ||
                        JSON.stringify(players.map(p => p.name)) !== JSON.stringify(mprisAvailablePlayers.map(p => p.name));
                    if (changed) {
                        mprisAvailablePlayers = players;
                        renderMediaSourceTabs();
                    }
                }
            } catch (e) {
                // Ignore parse errors
            }
        }

        // Track selected player
        if (data['mpris_player_name']) {
            mprisSelectedPlayer = data['mpris_player_name'].value;
        }

        if (controlMode === 'mpris') {
            // MPRIS mode: read from mpris_* DataBus keys published by the backend watcher
            const mprisCoverKey = 'mpris_cover_url';
            const mprisTitleKey = 'mpris_title';
            const mprisArtistKey = 'mpris_artist';
            const mprisProgressKey = 'mpris_progress';
            const mprisStatusKey = 'mpris_playback_status';

            // Cover art: rewrite file:// URLs to HTTP proxy so browsers can load them
            if (data[mprisCoverKey]) {
                const coverImg = block.querySelector('.media-cover-image');
                if (coverImg) {
                    let newCover = data[mprisCoverKey].value;
                    if (newCover && newCover.startsWith('file://')) {
                        newCover = '/api/mpris/cover?url=' + encodeURIComponent(newCover.substring(7));
                    }
                    if (coverImg.src !== newCover && newCover) {
                        coverImg.src = newCover;
                    }
                }
            }

            if (data[mprisTitleKey]) {
                const titleEl = block.querySelector('.media-title');
                if (titleEl) {
                    const newTitle = data[mprisTitleKey].value;
                    if (titleEl.textContent !== newTitle) {
                        titleEl.textContent = newTitle || 'Unknown Title';
                    }
                }
            }

            if (data[mprisArtistKey]) {
                const artistEl = block.querySelector('.media-artist');
                if (artistEl) {
                    const newArtist = data[mprisArtistKey].value;
                    if (artistEl.textContent !== newArtist) {
                        artistEl.textContent = newArtist || 'Unknown Artist';
                    }
                }
            }

            if (data[mprisProgressKey]) {
                const progressFill = block.querySelector('.media-progress-fill');
                if (progressFill) {
                    const newProgress = data[mprisProgressKey].value;
                    const progressPercent = typeof newProgress === 'number' ? newProgress : parseFloat(newProgress);
                    if (!isNaN(progressPercent)) {
                        const currentWidth = progressFill.style.width;
                        const newWidth = `${progressPercent}%`;
                        if (currentWidth !== newWidth) {
                            progressFill.style.width = newWidth;
                        }
                    }
                }
            }

            // Toggle play/pause icons based on playback status
            if (data[mprisStatusKey]) {
                const playIcon = block.querySelector('.play-icon');
                const pauseIcon = block.querySelector('.pause-icon');
                if (playIcon && pauseIcon) {
                    const status = data[mprisStatusKey].value;
                    if (status === 'Playing') {
                        playIcon.style.display = 'none';
                        pauseIcon.style.display = 'block';
                    } else {
                        playIcon.style.display = 'block';
                        pauseIcon.style.display = 'none';
                    }
                }
            }
        } else {
            // Keyboard mode: read from custom data-media-* attribute keys
            if (coverKey && data[coverKey]) {
                const coverImg = block.querySelector('.media-cover-image');
                if (coverImg) {
                    const newCover = data[coverKey].value;
                    if (coverImg.src !== newCover) {
                        coverImg.src = newCover;
                    }
                }
            }

            if (titleKey && data[titleKey]) {
                const titleEl = block.querySelector('.media-title');
                if (titleEl) {
                    const newTitle = data[titleKey].value;
                    if (titleEl.textContent !== newTitle) {
                        titleEl.textContent = newTitle;
                    }
                }
            }

            if (artistKey && data[artistKey]) {
                const artistEl = block.querySelector('.media-artist');
                if (artistEl) {
                    const newArtist = data[artistKey].value;
                    if (artistEl.textContent !== newArtist) {
                        artistEl.textContent = newArtist;
                    }
                }
            }

            if (progressKey && data[progressKey]) {
                const progressFill = block.querySelector('.media-progress-fill');
                if (progressFill) {
                    const newProgress = data[progressKey].value;
                    const progressPercent = typeof newProgress === 'number' ? newProgress : parseFloat(newProgress);
                    if (!isNaN(progressPercent)) {
                        const currentWidth = progressFill.style.width;
                        const newWidth = `${progressPercent}%`;
                        if (currentWidth !== newWidth) {
                            progressFill.style.width = newWidth;
                        }
                    }
                }
            }
        }
    });
}

function showCommandToast(message, success) {
    const existing = document.querySelector('.command-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = `command-toast ${success ? 'success' : 'error'}`;

    const maxLines = 8;
    const lines = message.split('\n').slice(0, maxLines);
    const displayText = lines.join('\n') + (message.split('\n').length > maxLines ? '\n...' : '');

    toast.innerHTML = `
        <div class="toast-header">${success ? 'Success' : 'Error'}</div>
        <pre class="toast-body">${displayText}</pre>
    `;

    document.body.appendChild(toast);

    toast.addEventListener('click', () => {
        toast.remove();
    });

    setTimeout(() => {
        if (toast.parentElement) {
            toast.classList.add('fading');
            setTimeout(() => toast.remove(), 300);
        }
    }, 5000);
}

async function keepScreenAlive() {
    if ('wakeLock' in navigator) {
        try {
            const wakeLock = await navigator.wakeLock.request('screen');
            console.log('Wake Lock is active! Screen will stay on.');

            document.addEventListener('visibilitychange', async () => {
                if (document.visibilityState === 'visible') {
                    await navigator.wakeLock.request('screen');
                }
            });
        } catch (err) {
            console.error(`Wake Lock failed: ${err.name}, ${err.message}`);
        }
    } else {
        console.warn('Wake Lock API not supported in this browser.');
    }
}

function connect() {
    if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
        return;
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    console.log("Attempting to connect...");
    socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

    socket.onopen = () => {
        console.log("Connected to OmniPanel-go Host");
        console.log('Secure context:', window.isSecureContext);
    };

    socket.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        console.log('WS message received:', msg.type);

        if (msg.type === 'force-reload') {
            console.log("Host changed panel. Reloading...");
            location.reload();
        }

        if (msg.type === 'enter-fullscreen') {
            showFullscreenPopup();
        } else if (msg.type === 'exit-fullscreen') {
            if (document.fullscreenElement) {
                document.exitFullscreen();
            }
            const popup = document.getElementById('fullscreen-prompt');
            if (popup) popup.remove();
        }

        if (msg.type === 'command-result') {
            handleCommandResult(msg.data);
        }

        if (msg.type === 'data-update') {
            handleDataUpdate(msg.data);
        }

        if (msg.type === 'rss-update') {
            handleRSSUpdate(msg.data);
        }

        if (msg.type === 'speech-config-init') {
            speechConfig.enabled = msg.data.enabled;
            recordingLocation = msg.data.recordingLocation || 'client';
            speechConfig.recordingLocation = recordingLocation;
            speechConfig.triggerMode = msg.data.triggerMode || 'push-to-talk';
            speechConfig.wakeWord = msg.data.wakeWord || 'omnipanel-go';
            speechConfig.ttsEnabled = msg.data.ttsEnabled;
            speechConfig.wakeWordListenSec = msg.data.wakeWordListenSec || 8;

            console.log('Speech config received:', speechConfig);

            if (speechConfig.enabled && speechConfig.triggerMode === 'wake-word' && recordingLocation === 'client') {
                console.log('Auto-starting wake word mode...');
                startWakeWordMode();
            }

            if (speechConfig.enabled && speechConfig.triggerMode === 'wake-word' && recordingLocation === 'host') {
                console.log('Host wake word mode active - hiding mic button');
                const micBtn = document.getElementById('speech-mic-btn');
                if (micBtn) micBtn.style.display = 'none';
            }
        }

        if (msg.type === 'speech-result') {
            handleSpeechResult(msg.data);

            document.querySelectorAll('.speech-command-block').forEach(block => {
                block.dispatchEvent(new CustomEvent('speech-result-' + block.dataset.blockId, { detail: msg.data }));
            });

            document.querySelectorAll('.loaded-block').forEach(block => {
                block.dispatchEvent(new CustomEvent('speech-result', { detail: msg.data }));
            });
        }

        if (msg.type === 'speech-error') {
            showSpeechToast(msg.data.error, false);

            document.querySelectorAll('.speech-command-block').forEach(block => {
                block.dispatchEvent(new CustomEvent('speech-error-' + block.dataset.blockId, { detail: msg.data }));
            });
        }

        if (msg.type === 'recording-status') {
            if (msg.data.location) {
                recordingLocation = msg.data.location;
            }
            updateMicButtonState(msg.data.state);

            document.querySelectorAll('.speech-command-block').forEach(block => {
                block.dispatchEvent(new CustomEvent('recording-status-' + block.dataset.blockId, { detail: msg.data }));
            });
        }

        if (msg.type === 'speech-button-trigger') {
            simulateButtonFromSpeech(msg.data.block_id);
        }

        if (msg.type === 'speech-slider-trigger') {
            simulateSliderFromSpeech(msg.data.block_id, msg.data.value);
        }
    };
}

function showFullscreenPopup() {
    if (document.getElementById('fullscreen-prompt')) return;

    const overlay = document.createElement('div');
    overlay.id = 'fullscreen-prompt';

    overlay.innerHTML = `
        <div class="prompt-card">
            <h3>Fullscreen Requested</h3>
            <p>The host wants to switch to fullscreen mode.</p>
            <button id="accept-fullscreen">Go Fullscreen</button>
        </div>
    `;

    document.body.appendChild(overlay);

    document.getElementById('accept-fullscreen').addEventListener('click', () => {
        document.documentElement.requestFullscreen()
            .then(() => {
                overlay.remove();
            })
            .catch(err => {
                console.error(`Fullscreen denied: ${err.message}`);
                overlay.remove();
            });
    });
}

function startWatchdog() {
    if (reconnectInterval) clearInterval(reconnectInterval);

    reconnectInterval = setInterval(() => {
        if (!socket || socket.readyState === WebSocket.CLOSED || socket.readyState === WebSocket.CLOSING) {
            console.warn("Watchdog detected closed connection. Reconnecting...");
            connect();
        }
    }, 3000);
}

/**
 * Attaches event listeners to all interactive elements on the panel.
 * Handles buttons (momentary and toggle modes), sliders, command buttons,
 * and keyboard shortcuts. Toggle mode buttons persist their visual state
 * and send state-based signals; momentary mode buttons send a quick
 * press/release pulse (100ms) while also persisting their visual state.
 * Called after loadPanel() renders all blocks.
 */
function enableInputs() {
    
    function getJoystickIndex(element) {
        return parent ? element.getAttribute('virtual-joystick') : "0";
    }

    keyboardShortcutMap = {};
    activeKeyboardKeys.clear();

    const buttons = document.querySelectorAll('[emulate-button]');
    if (buttons != null) {
        buttons.forEach(button => {
            const btnId = button.getAttribute('emulate-button');
            const kbKey = button.getAttribute('keyboard-key');
            const kbIndex = button.getAttribute('keyboard-index') || '0';
            const toggleMode = button.getAttribute('toggle-mode');
            const isToggleMode = toggleMode === 'toggle';
            // Skip empty values and unreplaced template placeholders (e.g., "settings-keyboard_key")
            // from panels created before keyboard support was added.
            const hasKeyboard = kbKey && kbKey.trim() !== '' && !kbKey.startsWith('settings-');
            const normalizedKey = hasKeyboard ? kbKey.trim().toLowerCase() : null;

            if (hasKeyboard) {
                keyboardShortcutMap[normalizedKey] = {
                    type: 'button',
                    js: getJoystickIndex(button),
                    id: btnId,
                    keyboardIndex: kbIndex,
                    element: button
                };
            }

            if (isToggleMode) {
                const savedState = button.dataset.toggledState;
                if (savedState === 'true') {
                    button.classList.add('active');
                }

                button.addEventListener('pointerup', (e) => {
                    e.stopPropagation();
                    const jsIndex = getJoystickIndex(button);
                    const currentState = button.classList.contains('active');
                    const newState = !currentState;

                    if (newState) {
                        button.classList.add('active');
                    } else {
                        button.classList.remove('active');
                    }
                    button.dataset.toggledState = newState ? 'true' : 'false';

                    if (hasKeyboard) {
                        socket.send(JSON.stringify({
                            type: 'simulate-keyboard',
                            data: {
                                js: parseInt(kbIndex),
                                key: normalizedKey,
                                state: newState ? 1 : 0
                            }
                        }));
                    } else {
                        socket.send(JSON.stringify({
                            type: 'simulate-button',
                            data: {
                                js: jsIndex,
                                id: btnId,
                                state: newState ? 1 : 0
                            }
                        }));
                    }
                });
            } else {
                const savedState = button.dataset.toggledState;
                if (savedState === 'true') {
                    button.classList.add('active');
                }

                button.addEventListener('pointerup', (e) => {
                    e.stopPropagation();
                    const jsIndex = getJoystickIndex(button);
                    const currentState = button.classList.contains('active');
                    const newState = !currentState;

                    if (newState) {
                        button.classList.add('active');
                    } else {
                        button.classList.remove('active');
                    }
                    button.dataset.toggledState = newState ? 'true' : 'false';

                    if (hasKeyboard) {
                        socket.send(JSON.stringify({
                            type: 'simulate-keyboard',
                            data: {
                                js: parseInt(kbIndex),
                                key: normalizedKey,
                                state: 1
                            }
                        }));
                        setTimeout(() => {
                            socket.send(JSON.stringify({
                                type: 'simulate-keyboard',
                                data: {
                                    js: parseInt(kbIndex),
                                    key: normalizedKey,
                                    state: 0
                                }
                            }));
                        }, 100);
                    } else {
                        socket.send(JSON.stringify({
                            type: 'simulate-button',
                            data: {
                                js: jsIndex,
                                id: btnId,
                                state: 1
                            }
                        }));
                        setTimeout(() => {
                            socket.send(JSON.stringify({
                                type: 'simulate-button',
                                data: {
                                    js: jsIndex,
                                    id: btnId,
                                    state: 0
                                }
                            }));
                        }, 100);
                    }
                });
            }
        });
    }

    /**
     * Slider initialization — handles both joystick axis emulation and system control.
     *
     * Each slider has two modes:
     * - Joystick mode (default): Sends `simulate-slider` WebSocket messages with the
     *   current value (0 to max_value). The host receives this as a virtual joystick axis.
     * - System control mode: When `system-control` and `system-control-command` attributes
     *   are both set, sends `execute-command` messages instead. The command template uses
     *   `{value}` as a placeholder that gets replaced with the current slider value.
     *   Example: `pactl set-sink-volume @DEFAULT_SINK@ {value}%` for volume control.
     *
     * Keyboard shortcuts: If `keyboard-key` is set, the slider is added to the shortcut
     * map so physical key presses can drive the slider value.
     */
    const sliders = document.querySelectorAll('[emulate-slider]');
    if (sliders != null) {
        sliders.forEach(slider => {
            const axisId = parseInt(slider.getAttribute('emulate-slider'));
            const systemControl = slider.getAttribute('system-control');
            const systemControlCommand = slider.getAttribute('system-control-command');
            // System control mode is active only when both attributes are set
            // and contain actual values (not unresolved template placeholders).
            const useSystemControl = systemControl && systemControl.trim() !== '' && !systemControl.startsWith('settings-') && systemControlCommand && systemControlCommand.trim() !== '' && !systemControlCommand.startsWith('settings-');

            slider.addEventListener('input', (event) => {
                const jsIndex = getJoystickIndex(slider);
                const value = parseInt(event.target.value);

                if (useSystemControl) {
                    // Replace {value} placeholder and send as shell command
                    const command = systemControlCommand.replace('{value}', value.toString());
                    const payload = {
                        type: 'execute-command',
                        data: {
                            block_id: slider.closest('.loaded-block')?.id || '',
                            command_type: 'shell',
                            command: command,
                            http_method: '',
                            http_url: '',
                            http_body: '',
                            params: {}
                        }
                    };
                    socket.send(JSON.stringify(payload));
                } else {
                    // Default: emulate a virtual joystick axis
                    const payload = {
                        type: 'simulate-slider',
                        data: {
                            js: jsIndex,
                            id: axisId,
                            value: value
                        }
                    };
                    socket.send(JSON.stringify(payload));
                }
            });

            const kbKey = slider.getAttribute('keyboard-key');
            const kbIndex = slider.getAttribute('keyboard-index') || '0';
            if (kbKey && kbKey.trim() !== '' && !kbKey.startsWith('settings-')) {
                const normalizedKey = kbKey.trim().toLowerCase();
                keyboardShortcutMap[normalizedKey] = {
                    type: 'slider',
                    js: jsIndex,
                    id: axisId,
                    keyboardIndex: kbIndex,
                    element: slider
                };
            }
        });
    }

    const commandButtons = document.querySelectorAll('.command-button');
    if (commandButtons != null) {
        commandButtons.forEach(btn => {
            const blockWrapper = btn.closest('.loaded-block');
            if (!blockWrapper) return;

            const blockId = blockWrapper.id;

            btn.addEventListener('pointerdown', (e) => {
                e.stopPropagation();
                e.preventDefault();
                btn.setPointerCapture(e.pointerId);

                executeCommand(btn, blockWrapper, blockId);

                const holdRepeat = btn.dataset.holdRepeat === 'true';
                const holdInterval = parseInt(btn.dataset.holdInterval) || 200;

                if (holdRepeat) {
                    if (commandHoldIntervals[blockId]) {
                        clearInterval(commandHoldIntervals[blockId]);
                    }
                    commandHoldIntervals[blockId] = setInterval(() => {
                        executeCommand(btn, blockWrapper, blockId);
                    }, holdInterval);
                }
            });

            btn.addEventListener('pointerup', () => {
                if (commandHoldIntervals[blockId]) {
                    clearInterval(commandHoldIntervals[blockId]);
                    delete commandHoldIntervals[blockId];
                }
            });

            btn.addEventListener('pointercancel', () => {
                if (commandHoldIntervals[blockId]) {
                    clearInterval(commandHoldIntervals[blockId]);
                    delete commandHoldIntervals[blockId];
                }
            });
        });
    }

    // Media player control buttons: dual-mode routing
    // - MPRIS mode: sends HTTP requests to /api/mpris/control for playback commands,
    //   or fetches current volume then sends a volume set command (±5% delta).
    // - Keyboard mode: sends simulate-keyboard WebSocket messages to trigger
    //   media keys (MediaPlayPause, MediaTrackNext, etc.) on the host.
    document.querySelectorAll('.media-control-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            e.preventDefault();
            e.stopPropagation();

            const block = btn.closest('.media-player');
            const controlMode = block?.getAttribute('data-control-mode');
            const action = btn.getAttribute('data-action');

            if (!controlMode || !action) return;

            if (controlMode === 'mpris') {
                // Volume buttons need special handling: fetch current volume first,
                // then send a set-volume command with a relative delta.
                let mprisAction = action;
                if (action === 'volumedown' || action === 'volumeup') {
                    const volumeData = await fetch('/api/mpris/players').then(r => r.json());
                    if (volumeData.players && volumeData.players.length > 0) {
                        const currentVolume = volumeData.players[0].volume || 0.5;
                        const delta = action === 'volumedown' ? -0.05 : 0.05;
                        const newVolume = Math.max(0, Math.min(1, currentVolume + delta));
                        await fetch('/api/mpris/control', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ action: 'volume', volume: newVolume })
                        });
                    }
                    return;
                }

                await fetch('/api/mpris/control', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ action: mprisAction })
                });
            } else {
                // Keyboard mode: simulate media key press/release via WebSocket
                const keyboardIndex = btn.getAttribute('keyboard-index') || '0';
                const keyboardKey = btn.getAttribute('keyboard-key');

                if (keyboardKey) {
                    socket.send(JSON.stringify({
                        type: 'simulate-keyboard',
                        data: {
                            js: parseInt(keyboardIndex),
                            key: keyboardKey,
                            state: 1
                        }
                    }));

                    setTimeout(() => {
                        socket.send(JSON.stringify({
                            type: 'simulate-keyboard',
                            data: {
                                js: parseInt(keyboardIndex),
                                key: keyboardKey,
                                state: 0
                            }
                        }));
                    }, 100);
                }
            }
        });
    });

    document.querySelectorAll('.sequence-button button').forEach(btn => {
        btn.addEventListener('pointerup', (e) => {
            e.stopPropagation();
            e.preventDefault();
            executeSequence(btn);
        });
    });

    // RSS entry click handlers — open URL on host or client based on block setting.
    // When open_url_location is "client", window.open() is used directly.
    // When "host" (default), an open-url WebSocket message is sent to the server.
    // Two handlers are needed: one for entries present at enableInputs() time,
    // and a delegated document-level handler for entries created later via innerHTML.
    document.querySelectorAll('.rss-entry').forEach(entry => {
        entry.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            const url = entry.getAttribute('data-link');
            if (!url || !socket || socket.readyState !== WebSocket.OPEN) return;
            const block = entry.closest('.loaded-block');
            const openLocation = block?.settings?.['open_url_location'] || 'host';
            if (openLocation === 'client') {
                window.open(url, '_blank');
            } else {
                socket.send(JSON.stringify({
                    type: 'open-url',
                    data: { url: url }
                }));
            }
        });
    });

    document.addEventListener('click', (e) => {
        const rssEntry = e.target.closest('.rss-entry');
        if (rssEntry) {
            e.preventDefault();
            e.stopPropagation();
            const url = rssEntry.getAttribute('data-link');
            if (!url || !socket || socket.readyState !== WebSocket.OPEN) return;
            const block = rssEntry.closest('.loaded-block');
            const openLocation = block?.settings?.['open_url_location'] || 'host';
            if (openLocation === 'client') {
                window.open(url, '_blank');
            } else {
                socket.send(JSON.stringify({
                    type: 'open-url',
                    data: { url: url }
                }));
            }
        }
    });
}

/**
 * Executes a key sequence by holding Ctrl and tapping each key in order.
 *
 * The sequence code (e.g., "d,d,w,s,a") is split into individual keys.
 * Ctrl is pressed first, then each key is pressed and released with a
 * configurable delay between them, and finally Ctrl is released. This
 * mimics the behavior of holding Ctrl while tapping WASD keys — common
 * in games that use Ctrl+directional inputs.
 *
 * Each key event is sent as a simulate-keyboard WebSocket message with
 * state=1 (press) followed by state=0 (release) after 50ms. The Ctrl
 * key is held for the entire duration of the sequence.
 *
 * @param {HTMLButtonElement} btn - The button element with data-sequence-code,
 *   data-key-delay, and data-keyboard-index attributes.
 */
function executeSequence(btn) {
    const codeStr = btn.dataset.sequenceCode;
    if (!codeStr || codeStr.trim() === '') return;

    const keyDelay = parseInt(btn.dataset.keyDelay) || 100;
    const keyboardIndex = parseInt(btn.dataset.keyboardIndex) || 0;
    const keys = codeStr.split(',').map(k => k.trim().toLowerCase()).filter(k => k);

    btn.classList.add('executing');

    // Send CTRL press
    socket.send(JSON.stringify({
        type: 'simulate-keyboard',
        data: {
            js: keyboardIndex,
            key: 'ctrl',
            state: 1
        }
    }));

    let index = 0;
    function sendNextKey() {
        if (index >= keys.length) {
            setTimeout(() => {
                // Release CTRL
                socket.send(JSON.stringify({
                    type: 'simulate-keyboard',
                    data: {
                        js: keyboardIndex,
                        key: 'ctrl',
                        state: 0
                    }
                }));
                setTimeout(() => btn.classList.remove('executing'), 200);
            }, 50);
            return;
        }

        const key = keys[index];

        // Send key press
        socket.send(JSON.stringify({
            type: 'simulate-keyboard',
            data: {
                js: keyboardIndex,
                key: key,
                state: 1
            }
        }));

        setTimeout(() => {
            // Send key release
            socket.send(JSON.stringify({
                type: 'simulate-keyboard',
                data: {
                    js: keyboardIndex,
                    key: key,
                    state: 0
                }
            }));

            index++;
            setTimeout(sendNextKey, keyDelay);
        }, 50);
    }

    setTimeout(sendNextKey, 50);
}

function executeCommand(btn, blockWrapper, blockId) {
    const params = {};
    if (blockWrapper.paramValues) {
        Object.keys(blockWrapper.paramValues).forEach(key => {
            const paramName = key.replace('param_', '');
            params[paramName] = blockWrapper.paramValues[key];
        });
    }

    const payload = {
        type: 'execute-command',
        data: {
            block_id: blockId,
            command_type: btn.dataset.commandType || 'shell',
            command: btn.dataset.command || '',
            http_method: btn.dataset.httpMethod || 'GET',
            http_url: btn.dataset.httpUrl || '',
            http_body: btn.dataset.httpBody || '',
            params: params
        }
    };

    socket.send(JSON.stringify(payload));

    btn.classList.add('executing');
    setTimeout(() => {
        btn.classList.remove('executing');
    }, 300);
}

function createMicButton() {
    const micBtn = document.createElement('button');
    micBtn.id = 'speech-mic-btn';
    micBtn.className = 'mic-btn';
    micBtn.innerHTML = '&#x1F3A4;';
    micBtn.title = 'Push-to-talk';

    micBtn.addEventListener('pointerdown', (e) => {
        e.preventDefault();
        e.stopPropagation();
        startRecording();
    });

    micBtn.addEventListener('pointerup', (e) => {
        e.preventDefault();
        e.stopPropagation();
        stopRecording();
    });

    micBtn.addEventListener('pointercancel', () => {
        stopRecording();
    });

    document.body.appendChild(micBtn);
}

/**
 * Starts audio recording based on current speech config.
 * Routes to the appropriate recording mode:
 * - Host wake word: recording is handled entirely by the server
 * - Client wake word: starts Web Speech API wake word detection
 * - Push-to-talk: starts microphone capture (client or host)
 */
function startRecording() {
    if (!speechConfig.enabled || isRecording) return;

    if (speechConfig.triggerMode === 'wake-word' && recordingLocation === 'host') {
        console.log('Host wake word mode - recording handled by host');
        return;
    }

    if (speechConfig.triggerMode === 'wake-word' && recordingLocation === 'client') {
        startWakeWordMode();
        return;
    }

    startPushToTalkRecording();
}

function startPushToTalkRecording() {
    socket.send(JSON.stringify({
        type: 'start-recording',
        data: { mode: 'push-to-talk' }
    }));

    isRecording = true;
    updateMicButtonState('recording');

    if (recordingLocation === 'host') {
        return;
    }

    navigator.mediaDevices.getUserMedia({
        audio: {
            sampleRate: 16000,
            channelCount: 1,
            echoCancellation: true,
            noiseSuppression: true
        }
    }).then(stream => {
        audioStream = stream;
        pcmBuffer = [];

        audioContext = new (window.AudioContext || window.webkitAudioContext)({ sampleRate: 16000 });
        const source = audioContext.createMediaStreamSource(stream);

        const bufferSize = 4096;
        scriptProcessor = audioContext.createScriptProcessor(bufferSize, 1, 1);

        scriptProcessor.onaudioprocess = (e) => {
            const input = e.inputBuffer.getChannelData(0);
            const int16 = new Int16Array(input.length);
            for (let i = 0; i < input.length; i++) {
                const s = Math.max(-1, Math.min(1, input[i]));
                int16[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
            }
            pcmBuffer.push(new Uint8Array(int16.buffer));
        };

        source.connect(scriptProcessor);
        scriptProcessor.connect(audioContext.destination);
    }).catch(err => {
        console.error('Microphone access denied:', err);
        showSpeechToast('Microphone access denied', false);
        isRecording = false;
        updateMicButtonState('idle');
    });
}

function stopRecording() {
    if (!isRecording) return;

    if (recordingLocation === 'host') {
        socket.send(JSON.stringify({ type: 'stop-recording' }));
        isRecording = false;
        updateMicButtonState('processing');
        return;
    }

    if (scriptProcessor) {
        scriptProcessor.disconnect();
        scriptProcessor = null;
    }

    if (audioStream) {
        audioStream.getTracks().forEach(track => track.stop());
        audioStream = null;
    }

    if (audioContext) {
        audioContext.close();
        audioContext = null;
    }

    isRecording = false;
    updateMicButtonState('processing');

    if (pcmBuffer.length > 0) {
        const totalLen = pcmBuffer.reduce((sum, buf) => sum + buf.length, 0);
        const pcm = new Uint8Array(totalLen);
        let offset = 0;
        for (const buf of pcmBuffer) {
            pcm.set(buf, offset);
            offset += buf.length;
        }
        socket.send(pcm);
    }

    socket.send(JSON.stringify({ type: 'stop-recording' }));
}

/**
 * Starts wake word detection using the Web Speech API (Chrome/Edge only).
 * Continuously listens for the configured wake word. When detected,
 * switches to push-to-talk recording for follow-up speech, then
 * returns to wake word listening. Requires a secure context (HTTPS).
 */
function startWakeWordMode() {
    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
        showSpeechToast('Wake word requires Chrome/Edge', false);
        console.warn('SpeechRecognition API not available');
        return;
    }

    if (!window.isSecureContext) {
        showSpeechToast('Wake word requires HTTPS or localhost', false);
        console.warn('SpeechRecognition requires secure context');
        return;
    }

    if (speechRecognition) {
        console.log('Wake word already active, skipping');
        return;
    }

    console.log('Creating SpeechRecognition instance for wake word:', speechConfig.wakeWord);
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    speechRecognition = new SpeechRecognition();
    speechRecognition.continuous = true;
    speechRecognition.interimResults = true;
    speechRecognition.lang = 'en-US';

    speechRecognition.onresult = (event) => {
        let transcript = '';
        for (let i = event.resultIndex; i < event.results.length; i++) {
            transcript += event.results[i][0].transcript;
        }

        console.log('Wake word heard:', transcript);

        const wakeWord = speechConfig.wakeWord.toLowerCase();
        if (transcript.toLowerCase().includes(wakeWord)) {
            wakeWordActive = true;
            updateMicButtonState('wake-detected');
            showSpeechToast('Wake word detected, listening...', true);

            setTimeout(() => {
                speechRecognition.stop();
                startPushToTalkRecording();
                setTimeout(() => {
                    stopRecording();
                    wakeWordActive = false;
                    startWakeWordMode();
                }, 3000);
            }, 500);
        }
    };

    speechRecognition.onerror = (event) => {
        if (event.error !== 'aborted') {
            console.error('Wake word speech recognition error:', event.error, event.message);
        }
    };

    speechRecognition.onend = () => {
        console.log('Wake word recognition ended');
        if (speechConfig.triggerMode === 'wake-word' && recordingLocation === 'client' && speechRecognition && !wakeWordActive) {
            try {
                speechRecognition.start();
                console.log('Wake word recognition restarted');
            } catch (e) {
                console.error('Wake word restart failed:', e);
                speechRecognition = null;
            }
        } else {
            speechRecognition = null;
        }
    };

    try {
        speechRecognition.start();
        console.log('Wake word recognition started');
    } catch (e) {
        console.error('Failed to start wake word recognition:', e);
        speechRecognition = null;
    }
    wakeWordActive = false;
    updateMicButtonState('wake-listening');
}

/**
 * Stops wake word detection and resets the mic button to idle state.
 * Called when speech is disabled or the panel is reloaded.
 */
function stopWakeWordMode() {
    if (speechRecognition) {
        speechRecognition.stop();
        speechRecognition = null;
    }
    wakeWordActive = false;
    updateMicButtonState('idle');
}

/**
 * Updates the floating microphone button appearance based on state.
 * @param {string} state - One of: 'idle', 'recording', 'processing', 'wake-listening', 'wake-detection'
 */
function updateMicButtonState(state) {
    const micBtn = document.getElementById('speech-mic-btn');
    if (!micBtn) return;

    micBtn.classList.remove('recording', 'processing', 'wake-listening', 'wake-detected');

    switch (state) {
        case 'recording':
            micBtn.classList.add('recording');
            micBtn.title = 'Recording...';
            break;
        case 'processing':
            micBtn.classList.add('processing');
            micBtn.title = 'Processing...';
            break;
        case 'wake-listening':
            micBtn.classList.add('wake-listening');
            micBtn.title = 'Listening for wake word';
            break;
        case 'wake-detected':
            micBtn.classList.add('wake-detected');
            micBtn.title = 'Wake word detected!';
            break;
        default:
            micBtn.title = 'Push-to-talk';
            break;
    }
}

function handleSpeechResult(data) {
    const text = data.text || '';
    const matched = data.matched || false;
    const speakText = data.speak || '';

    showSpeechToast(`"${text}"${matched ? ' - Matched!' : ''}`, matched);

    if (speakText && speechConfig.ttsEnabled) {
        speakText(speakText);
    }

    updateMicButtonState('idle');
}

function showSpeechToast(message, success) {
    const existing = document.querySelector('.speech-toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = `speech-toast ${success ? 'success' : 'error'}`;
    toast.textContent = message;

    document.body.appendChild(toast);

    toast.addEventListener('click', () => {
        toast.remove();
    });

    setTimeout(() => {
        if (toast.parentElement) {
            toast.classList.add('fading');
            setTimeout(() => toast.remove(), 300);
        }
    }, 3000);
}

function speakText(text) {
    if (!('speechSynthesis' in window)) return;

    const utterance = new SpeechSynthesisUtterance(text);
    utterance.rate = 1.0;
    utterance.pitch = 1.0;
    utterance.volume = 0.8;
    window.speechSynthesis.speak(utterance);
}

/**
 * Visually simulates a button press triggered by speech recognition.
 * Adds an 'active' class to the button for a brief flash animation and
 * a 'speech-flash' class to the block wrapper for a cyan glow effect.
 * The actual button press is executed server-side via the JoystickManager,
 * so this function only handles visual feedback.
 * @param {string} blockId - The ID of the block to flash
 */
function simulateButtonFromSpeech(blockId) {
    const block = document.getElementById(blockId);
    if (!block) return;

    const btn = block.querySelector('.command-button, [emulate-button]');
    if (btn) {
        btn.classList.add('active');
        setTimeout(() => btn.classList.remove('active'), 200);
    }

    block.classList.add('speech-flash');
    setTimeout(() => block.classList.remove('speech-flash'), 500);
}

/**
 * Visually simulates a slider change triggered by speech recognition.
 * Updates the slider's value, dispatches an input event for UI updates,
 * and adds a 'speech-flash' class to the block wrapper for a cyan glow.
 * The actual slider change is executed server-side via the JoystickManager,
 * so this function only handles visual feedback.
 * @param {string} blockId - The ID of the block containing the slider
 * @param {string} valueStr - The spoken value to set (0-255)
 */
function simulateSliderFromSpeech(blockId, valueStr) {
    const block = document.getElementById(blockId);
    if (!block) return;

    const slider = block.querySelector('[emulate-slider]');
    if (slider) {
        const value = Math.max(0, Math.min(255, parseInt(valueStr) || 128));

        slider.value = value;
        slider.dispatchEvent(new Event('input'));

        block.classList.add('speech-flash');
        setTimeout(() => block.classList.remove('speech-flash'), 500);
    }
}

/**
 * Initializes RSS feed polling for a block by sending an rss-configure
 * WebSocket message to the server with the block's feed URLs, refresh
 * interval, and maximum entry count.
 * Called during block rendering when a .rss-feed element is detected.
 * Feed URLs are parsed from the feed_urls setting (newline-separated,
 * supporting "URL|Label" format for optional display labels).
 * If the WebSocket is not yet open, retries every 100ms until connected
 * to prevent the configuration message from being dropped on first load.
 * @param {HTMLElement} blockWrapper - The block DOM element with settings.
 */
function initRSSFeed(blockWrapper) {
    const feedUrlsSetting = blockWrapper.settings['feed_urls'];
    if (!feedUrlsSetting) return;

    const feedUrls = feedUrlsSetting.split('\n').map(u => u.trim()).filter(u => u);
    if (feedUrls.length === 0) return;

    const refreshInterval = parseInt(blockWrapper.settings['refresh_interval']) || 60;
    const maxEntries = parseInt(blockWrapper.settings['max_entries']) || 20;

    const config = {
        type: 'rss-configure',
        data: {
            block_id: blockWrapper.id,
            feed_urls: feedUrls,
            refresh_interval: refreshInterval,
            max_entries: maxEntries
        }
    };

    function trySend() {
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(JSON.stringify(config));
        } else {
            setTimeout(trySend, 100);
        }
    }

    trySend();
}

/**
 * Processes rss-update WebSocket messages and renders feed entries into the DOM.
 * Determines which entries are "new" by comparing entry GUIDs against the
 * client's seen-entry set (rssSeenEntries). New entries receive the "new-entry"
 * CSS class for highlighting. Entry metadata (feed label, date) and description
 * are rendered based on block settings (show_feed_label, show_date, show_description).
 * Descriptions are stripped of HTML tags and truncated to description_max_length.
 * @param {Object} data - RSS update payload with block_id and entries array.
 * Each entry has: guid, title, link, published, description, feed_label, is_new.
 */
function handleRSSUpdate(data) {
    const blockId = data.block_id;
    const entries = data.entries || [];
    const block = document.getElementById(blockId);
    if (!block) return;

    const entriesList = block.querySelector('.rss-entries-list');
    if (!entriesList) return;

    const settings = block.settings || {};
    const showDate = settings['show_date'] !== 'false';
    const showFeedLabel = settings['show_feed_label'] !== 'false';
    const showDescription = settings['show_description'] !== 'false';
    const descMaxLength = parseInt(settings['description_max_length']) || 150;

    if (!rssSeenEntries[blockId]) {
        rssSeenEntries[blockId] = new Set();
    }

    const currentGUIDs = new Set();
    let html = '';

    if (entries.length === 0) {
        html = '<div class="rss-empty-state">No entries found</div>';
    } else {
        entries.forEach(entry => {
            const isNew = entry.is_new && !rssSeenEntries[blockId].has(entry.guid);
            currentGUIDs.add(entry.guid);

            let metaHtml = '';
            const metaParts = [];
            if (showFeedLabel && entry.feed_label) {
                metaParts.push(`<span class="rss-entry-feed-label">${escapeHtml(entry.feed_label)}</span>`);
            }
            if (showDate && entry.published) {
                const date = new Date(entry.published);
                metaParts.push(`<span class="rss-entry-date">${date.toLocaleDateString()} ${date.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>`);
            }
            if (metaParts.length > 0) {
                metaHtml = `<div class="rss-entry-meta">${metaParts.join('')}</div>`;
            }

            let descHtml = '';
            if (showDescription && entry.description) {
                let desc = entry.description.replace(/<[^>]*>/g, '');
                if (desc.length > descMaxLength) {
                    desc = desc.substring(0, descMaxLength) + '...';
                }
                descHtml = `<div class="rss-entry-description">${escapeHtml(desc)}</div>`;
            }

            html += `
                <div class="rss-entry ${isNew ? 'new-entry' : ''}" data-link="${escapeHtml(entry.link)}">
                    <div class="rss-entry-title">${escapeHtml(entry.title || 'Untitled')}</div>
                    ${metaHtml}
                    ${descHtml}
                </div>
            `;
        });
    }

    entriesList.innerHTML = html;

    rssSeenEntries[blockId] = currentGUIDs;
}

/**
 * Escapes HTML special characters in a string to prevent XSS when
 * injecting user-provided content (feed titles, descriptions, labels)
 * into the DOM via innerHTML.
 * @param {string} text - The raw text to escape.
 * @returns {string} The escaped text safe for HTML injection.
 */
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

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
                document.body.innerHTML = `<h1>Error</h1><p>${err.message}</p><p><a href="/">Go to start page</a></p>`;
            });
    } else {
        document.body.innerHTML = `<h1>No Panel Specified</h1><p>Go to the <a href="/">start page</a> to select a panel.</p>`;
    }

    connect();
    startWatchdog();
    keepScreenAlive();
    createMicButton();
    enableKeyboardShortcuts();
    enableThreeFingerSwipe();
});

/**
 * Registers global keyboard shortcut listeners.
 * When a key or combination matches a block's keyboard-key setting,
 * sends a simulate-keyboard event to the server and triggers the
 * corresponding button press or slider change.
 *
 * Supported modifier keys: ctrl, shift, alt, meta.
 * Example shortcuts: "a", "ctrl+a", "ctrl+shift+escape".
 */
function enableKeyboardShortcuts() {
    /**
     * Converts a KeyboardEvent into a normalized shortcut string.
     * Modifiers are prepended in order: ctrl+shift+alt+meta+key.
     * Special keys are mapped to their names (e.g., " " -> "space").
     * @param {KeyboardEvent} e - The keyboard event
     * @returns {string} Normalized shortcut string (e.g., "ctrl+a")
     */
    function buildKeyString(e) {
        const parts = [];
        if (e.ctrlKey) parts.push('ctrl');
        if (e.shiftKey) parts.push('shift');
        if (e.altKey) parts.push('alt');
        if (e.metaKey) parts.push('meta');
        
        const key = e.key.toLowerCase();
        if (key === 'control' || key === 'shift' || key === 'alt' || key === 'meta') {
            return parts.join('+');
        }
        
        const keyMap = {
            ' ': 'space',
            'escape': 'escape',
            'enter': 'enter',
            'tab': 'tab',
            'backspace': 'backspace',
            'capslock': 'capslock',
            'arrowup': 'up',
            'arrowdown': 'down',
            'arrowleft': 'left',
            'arrowright': 'right',
        };
        
        const mappedKey = keyMap[key] || key;
        parts.push(mappedKey);
        return parts.join('+');
    }

    document.addEventListener('keydown', (e) => {
        if (e.repeat) return;
        
        const keyStr = buildKeyString(e);
        
        if (keyboardShortcutMap[keyStr]) {
            e.preventDefault();
            activeKeyboardKeys.add(keyStr);
            
            const shortcut = keyboardShortcutMap[keyStr];
            
            socket.send(JSON.stringify({
                type: 'simulate-keyboard',
                data: {
                    js: parseInt(shortcut.keyboardIndex),
                    key: keyStr,
                    state: 1
                }
            }));
            
            if (shortcut.type === 'button') {
                socket.send(JSON.stringify({
                    type: 'simulate-button',
                    data: {
                        js: parseInt(shortcut.js),
                        id: parseInt(shortcut.id),
                        state: 1
                    }
                }));
                if (shortcut.element) {
                    shortcut.element.classList.add('active');
                }
            } else if (shortcut.type === 'slider') {
                const currentValue = parseInt(shortcut.element.value) || 0;
                const newValue = Math.min(255, currentValue + 25);
                shortcut.element.value = newValue;
                shortcut.element.dispatchEvent(new Event('input'));
            }
        }
    });

    document.addEventListener('keyup', (e) => {
        const keyStr = buildKeyString(e);
        
        if (keyboardShortcutMap[keyStr] && activeKeyboardKeys.has(keyStr)) {
            e.preventDefault();
            activeKeyboardKeys.delete(keyStr);
            
            const shortcut = keyboardShortcutMap[keyStr];
            
            socket.send(JSON.stringify({
                type: 'simulate-keyboard',
                data: {
                    js: parseInt(shortcut.keyboardIndex),
                    key: keyStr,
                    state: 0
                }
            }));
            
            if (shortcut.type === 'button') {
                socket.send(JSON.stringify({
                    type: 'simulate-button',
                    data: {
                        js: parseInt(shortcut.js),
                        id: parseInt(shortcut.id),
                        state: 0
                    }
                }));
                if (shortcut.element) {
                    shortcut.element.classList.remove('active');
                }
            }
        }
    });
}

/**
 * Global state for 3-finger swipe gesture detection.
 * Tracks active touch pointers, swipe start position, and timing.
 */
let availablePanels = [];
let currentPanelIndex = -1;
let swipeState = {
    activePointers: new Map(),
    startX: 0,
    isTracking: false,
    lastSwitchTime: 0,
};

/**
 * Minimum horizontal distance (in pixels) required to trigger a panel switch.
 * Prevents accidental switches from small movements.
 */
const SWIPE_THRESHOLD = 100;

/**
 * Minimum time (in milliseconds) between consecutive panel switches.
 * Prevents rapid accidental switching.
 */
const SWIPE_COOLDOWN = 500;

/**
 * Fetches the list of available panels from the server and determines
 * the current panel's position in the alphabetical ordering.
 * Called once when the 3-finger swipe gesture system is initialized.
 */
async function loadPanelList() {
    try {
        const res = await fetch('/api/panels');
        const data = await res.json();
        availablePanels = data.allPanels || [];
        
        const urlParams = new URLSearchParams(window.location.search);
        const currentPanel = urlParams.get('name');
        
        if (currentPanel) {
            currentPanelIndex = availablePanels.indexOf(currentPanel);
        }
    } catch (e) {
        console.error('Failed to load panel list:', e);
        availablePanels = [];
    }
}

/**
 * Switches to the next or previous panel based on swipe direction.
 * Implements wrap-around behavior (last panel → first, first → last).
 * Fetches panel content via HTTP and renders it without page reload.
 * Updates the browser URL to reflect the current panel.
 * @param {string} direction - 'left' for previous panel, 'right' for next panel
 */
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
    
    if (targetIndex === currentPanelIndex) return;
    
    const targetPanel = availablePanels[targetIndex];
    swipeState.lastSwitchTime = now;
    
    showSwipeFeedback(direction, targetPanel);
    
    fetch(`/api/panel/content?name=${encodeURIComponent(targetPanel)}`)
        .then(res => {
            if (!res.ok) throw new Error(`Panel "${targetPanel}" not found`);
            return res.json();
        })
        .then(data => {
            loadPanel(data);
            currentPanelIndex = targetIndex;
            window.history.replaceState({}, '', `/panel?name=${encodeURIComponent(targetPanel)}`);
            setTimeout(() => hideSwipeFeedback(), 300);
        })
        .catch(err => {
            console.error('Failed to switch panel:', err);
            hideSwipeFeedback();
            showToast(`Failed to load panel: ${targetPanel}`, 'error');
        });
}

/**
 * Displays a visual overlay during swipe gestures showing the target panel name.
 * The overlay slides in from the swipe direction and fades in proportionally
 * to the swipe distance.
 * @param {string} direction - 'left' or 'right', determines slide animation
 * @param {string} panelName - Name of the target panel to display
 * @param {number} opacity - Opacity level (0-1), defaults to 1
 */
function showSwipeFeedback(direction, panelName, opacity = 1) {
    let overlay = document.getElementById('swipe-overlay');
    if (!overlay) {
        overlay = document.createElement('div');
        overlay.id = 'swipe-overlay';
        overlay.innerHTML = '<div class="swipe-panel-name"></div>';
        document.body.appendChild(overlay);
    }
    
    overlay.className = `swipe-${direction}`;
    overlay.querySelector('.swipe-panel-name').textContent = panelName;
    overlay.style.display = 'flex';
    overlay.style.opacity = opacity;
}

/**
 * Hides the swipe gesture feedback overlay after panel switch completes.
 */
function hideSwipeFeedback() {
    const overlay = document.getElementById('swipe-overlay');
    if (overlay) {
        overlay.style.display = 'none';
    }
}

/**
 * Displays a toast notification for swipe-related errors or status messages.
 * Auto-hides after 1.5 seconds.
 * @param {string} message - Text to display in the toast
 * @param {string} type - 'info', 'error', or 'success' for styling
 */
function showToast(message, type = 'info') {
    let toast = document.getElementById('swipe-toast');
    if (!toast) {
        toast = document.createElement('div');
        toast.id = 'swipe-toast';
        document.body.appendChild(toast);
    }
    
    toast.textContent = message;
    toast.className = `toast-${type} show`;
    
    setTimeout(() => {
        toast.classList.remove('show');
    }, 1500);
}

/**
 * Enables 3-finger swipe gesture detection for panel navigation.
 * 
 * Listens for pointer events in capture phase to intercept gestures before
 * child elements (joysticks, mousepads) consume them. Tracks exactly 3
 * simultaneous touch pointers and calculates cumulative horizontal movement.
 * 
 * Gesture behavior:
 * - 3 fingers swipe left (right-to-left): Switch to next panel
 * - 3 fingers swipe right (left-to-right): Switch to previous panel
 * - Wrap-around: Last panel → first, first → last
 * - Threshold: 100px minimum horizontal movement
 * - Cooldown: 500ms between switches
 * 
 * The gesture system provides real-time visual feedback with an overlay
 * that fades in proportionally to swipe distance, showing the target
 * panel name before the switch completes.
 * 
 * Exclusions: Does not activate on joystick, mousepad, or push-to-talk blocks.
 */
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
    
    document.addEventListener('pointermove', (e) => {
        if (!swipeState.isTracking || swipeState.activePointers.size !== 3) return;
        if (!swipeState.activePointers.has(e.pointerId)) return;
        
        e.preventDefault();
        
        swipeState.activePointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
        
        const pointers = Array.from(swipeState.activePointers.values());
        const currentCenterX = pointers.reduce((sum, p) => sum + p.x, 0) / 3;
        const deltaY = Math.abs(pointers.reduce((sum, p) => sum + p.y, 0) / 3 - 
                                (swipeState.startY || lastCenterX));
        
        cumulativeDeltaX += (currentCenterX - lastCenterX);
        lastCenterX = currentCenterX;
        
        if (Math.abs(cumulativeDeltaX) > 30) {
            const direction = cumulativeDeltaX > 0 ? 'right' : 'left';
            const opacity = Math.min(0.8, Math.abs(cumulativeDeltaX) / SWIPE_THRESHOLD);
            showSwipeFeedback(direction, '', opacity);
        }
    }, { passive: false, capture: true });
    
    document.addEventListener('pointerup', handlePointerEnd, true);
    document.addEventListener('pointercancel', handlePointerEnd, true);
    
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
