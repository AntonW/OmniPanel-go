/**
 * BlockLibrary loads the block template list from /api/blocks and renders them
 * in the editor's left sidebar, grouped by category. Blocks are now consolidated
 * into a flat directory (user/blocks/*.html) without theme subdirectories; each
 * block type appears once regardless of how many themes exist.
 */

/**
 * Retrieves the auth token from URL params, localStorage, or sessionStorage.
 * @returns {string|null} The auth token or null.
 */
function getBlockLibAuthToken() {
    const urlParams = new URLSearchParams(window.location.search);
    const urlToken = urlParams.get('token');
    if (urlToken) return urlToken;
    return localStorage.getItem('auth_token') || sessionStorage.getItem('auth_token');
}

/**
 * Adds Authorization: Bearer header to fetch headers for block API calls.
 * @param {Object} headers - Existing headers
 * @returns {Object} Headers with auth added (unchanged if no token)
 */
function addBlockLibAuthHeaders(headers = {}) {
    const token = getBlockLibAuthToken();
    if (!token) return headers;
    return { ...headers, 'Authorization': `Bearer ${token}` };
}

class BlockLibrary {
    constructor() {
        this.blocks = [];
        this.categories = {
            'Input Controls': ['button', 'toggle', 'emergency_button', 'sequence_button'],
            'Sliders': ['slider', 'slider_vertical', 'power_slider'],
            'Pointing': ['mousepad', 'touch_pad'],
            'Commands': ['command_block'],
            'Speech': ['speech_command', 'push_to_talk'],
            'Data Display': ['data_display'],
            // Media category: media_player block with MPRIS (Linux D-Bus) or keyboard mode
            'Media': ['media_player'],
            'Containers': ['paged_container', 'section_group'],
        };
    }

    async init() {
        try {
            const res = await fetch('/api/blocks', { headers: addBlockLibAuthHeaders() });
            const blocksRaw = await res.json();
            this.blocks = this.flattenBlocks(blocksRaw);
            this.render();
            this.initSearch();
        } catch (e) {
            console.error('Failed to load blocks:', e);
        }
    }

    flattenBlocks(tree) {
        let result = [];
        for (const item of tree) {
            if (item.type === 'folder') {
                // Skip folders - blocks are now flat
                result = result.concat(this.flattenBlocks(item.children));
            } else if (item.name && item.name.endsWith('.html')) {
                result.push({
                    ...item,
                    displayName: item.name.replace('.html', '').replace(/_/g, ' '),
                    category: this.categorize(item.name)
                });
            }
        }
        // Deduplicate by name (should already be flat, but just in case)
        const seen = new Set();
        return result.filter(block => {
            if (seen.has(block.name)) return false;
            seen.add(block.name);
            return true;
        });
    }

    categorize(filename) {
        const name = filename.replace('.html', '');
        for (const [category, patterns] of Object.entries(this.categories)) {
            if (patterns.some(p => name.includes(p))) return category;
        }
        return 'Other';
    }

    getIconForBlock(filename) {
        const icons = {
            'button': '\u{1F518}',
            'toggle': '\u{1F532}',
            'emergency_button': '\u{1F6A8}',
            'sequence_button': '\u{1F3AF}',
            'slider': '\u{1F39A}\u{FE0F}',
            'slider_vertical': '\u{1F39A}\u{FE0F}',
            'power_slider': '\u26A1',
            'mousepad': '\u{1F5B1}\u{FE0F}',
            'touch_pad': '\u{1F446}',
            'command_block': '\u2699\u{FE0F}',
            'speech_command': '\u{1F3A4}',
            'push_to_talk': '\u{1F399}\u{FE0F}',
            'data_display': '\u{1F4CA}',
            'media_player': '\u{1F3B5}',
            'paged_container': '\u{1F4D1}',
            'section_group': '\u{1F4E6}'
        };
        const name = filename.replace('.html', '');
        return icons[name] || '\u{1F4C4}';
    }

    render(filter = '') {
        const container = document.getElementById('block-library');
        container.innerHTML = '';

        const grouped = {};
        const filterLower = filter.toLowerCase();

        const filteredBlocks = filter
            ? this.blocks.filter(b =>
                b.displayName.toLowerCase().includes(filterLower) ||
                b.category.toLowerCase().includes(filterLower)
            )
            : this.blocks;

        for (const block of filteredBlocks) {
            if (!grouped[block.category]) grouped[block.category] = [];
            grouped[block.category].push(block);
        }

        const allCategories = [...new Set([...Object.keys(this.categories), 'Other'])];

        for (const category of allCategories) {
            const catBlocks = grouped[category];
            if (!catBlocks || catBlocks.length === 0) continue;

            const section = document.createElement('div');
            section.className = 'block-category';
            section.innerHTML = `
                <div class="category-header">
                    <span class="category-toggle">\u25BC</span>
                    <span class="category-name">${escapeHtml(category)}</span>
                    <span class="category-count">${catBlocks.length}</span>
                </div>
                <div class="category-blocks">
                    ${catBlocks.map(b => this.renderBlockCard(b)).join('')}
                </div>
            `;
            container.appendChild(section);
        }

        if (container.children.length === 0) {
            container.innerHTML = '<div class="empty-state">No blocks found</div>';
        }

        this.initDragDrop();
        this.initCategoryCollapse();
    }

    renderBlockCard(block) {
        return `
            <div class="block-card"
                 data-path="${escapeHtml(block.path)}"
                 data-name="${escapeHtml(block.displayName)}"
                 draggable="true">
                <div class="block-preview">
                    <span class="block-icon">${this.getIconForBlock(block.name)}</span>
                </div>
                <div class="block-info">
                    <span class="block-name">${escapeHtml(block.displayName)}</span>
                </div>
            </div>
        `;
    }

    initDragDrop() {
        document.querySelectorAll('.block-card').forEach(card => {
            card.addEventListener('dragstart', (event) => {
                event.dataTransfer.setData('text/plain', card.dataset.path);
                event.dataTransfer.setData('block-name', card.dataset.name);
                event.dataTransfer.effectAllowed = 'copy';

                const rect = card.getBoundingClientRect();
                const offsetX = event.clientX - rect.left;
                const offsetY = event.clientY - rect.top;
                event.dataTransfer.setData('offset', JSON.stringify({ x: offsetX, y: offsetY }));
            });
        });
    }

    initCategoryCollapse() {
        document.querySelectorAll('.category-header').forEach(header => {
            header.addEventListener('click', () => {
                header.classList.toggle('collapsed');
            });
        });
    }

    initSearch() {
        const searchInput = document.getElementById('block-search');
        const debouncedSearch = debounce((value) => {
            this.render(value);
        }, 200);

        searchInput.addEventListener('input', (e) => {
            debouncedSearch(e.target.value);
        });
    }
}
