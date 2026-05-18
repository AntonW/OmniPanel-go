/**
 * Workspace manages the editor canvas: zoom, pan, grid, aspect ratio, and background.
 * It applies CSS transforms to center and scale the panel within the viewport.
 */
class Workspace {
    constructor(state) {
        this.state = state;
        this.zoom = 1;
        this.panX = 0;
        this.panY = 0;
        this.isPanning = false;
        this.lastPanPoint = null;
        this.gridSize = 12;
        this.gridEnabled = true;
        this.snapEnabled = true;
        this.guidesEnabled = true;
        this.aspectRatio = 16 / 9;
    }

    /** Initializes zoom, grid, aspect ratio, and background controls. */
    init() {
        this.initZoom();
        this.initGrid();
        this.initAspect();
        this.initWorkspaceBg();
    }

    /** Sets up Ctrl+wheel zoom, middle-click pan, and zoom button handlers. */
    initZoom() {
        const viewport = document.getElementById('workspace-viewport');

        viewport.addEventListener('wheel', (e) => {
            if (e.ctrlKey) {
                e.preventDefault();
                const delta = e.deltaY > 0 ? -0.05 : 0.05;
                this.setZoom(this.zoom + delta);
            }
        }, { passive: false });

        viewport.addEventListener('mousedown', (e) => {
            if (e.button === 1) {
                e.preventDefault();
                this.isPanning = true;
                this.lastPanPoint = { x: e.clientX, y: e.clientY };
                viewport.style.cursor = 'grabbing';
            }
        });

        window.addEventListener('mousemove', (e) => {
            if (!this.isPanning) return;
            const dx = e.clientX - this.lastPanPoint.x;
            const dy = e.clientY - this.lastPanPoint.y;
            this.panX += dx;
            this.panY += dy;
            this.lastPanPoint = { x: e.clientX, y: e.clientY };
            this.applyTransform();
        });

        window.addEventListener('mouseup', () => {
            this.isPanning = false;
            viewport.style.cursor = '';
        });

        document.getElementById('btn-zoom-in').onclick = () => this.setZoom(this.zoom + 0.1);
        document.getElementById('btn-zoom-out').onclick = () => this.setZoom(this.zoom - 0.1);
        document.getElementById('btn-zoom-fit').onclick = () => this.fitToScreen();
    }

    /**
     * Sets the zoom level, clamped between 25% and 300%.
     * @param {number} level - Target zoom multiplier (e.g., 1.0 = 100%).
     */
    setZoom(level) {
        this.zoom = clamp(level, 0.25, 3);
        document.getElementById('zoom-level').textContent = `${Math.round(this.zoom * 100)}%`;
        this.applyTransform();
    }

    /** Applies the current pan and zoom as a CSS transform to the workspace canvas. */
    applyTransform() {
        const canvas = document.getElementById('workspace-canvas');
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

    initGrid() {
        const gridToggle = document.getElementById('grid-toggle');
        const gridSizeSelect = document.getElementById('grid-size');
        const snapToggle = document.getElementById('snap-toggle');
        const guidesToggle = document.getElementById('align-guides');

        gridToggle.addEventListener('change', () => {
            this.gridEnabled = gridToggle.checked;
            document.getElementById('maincontainer').classList.toggle('grid-hidden', !this.gridEnabled);
        });

        gridSizeSelect.addEventListener('change', () => {
            this.gridSize = parseInt(gridSizeSelect.value);
            this.updateGrid();
        });

        snapToggle.addEventListener('change', () => {
            this.snapEnabled = snapToggle.checked;
        });

        guidesToggle.addEventListener('change', () => {
            this.guidesEnabled = guidesToggle.checked;
        });

        this.updateGrid();
    }

    updateGrid() {
        const mainContainer = document.getElementById('maincontainer');
        const cellWidth = 100 / this.gridSize;
        const cellHeight = 100 / this.gridSize;
        mainContainer.style.setProperty('--grid-w', `${cellWidth}%`);
        mainContainer.style.setProperty('--grid-h', `${cellHeight}%`);
    }

    snapToGrid(value, gridSize) {
        if (!this.snapEnabled) return value;
        const step = 100 / (gridSize || this.gridSize);
        return Math.round(value / step) * step;
    }

    /**
     * Initializes aspect ratio dropdown, ResizeObserver, and auto-fits the panel.
     * Uses requestAnimationFrame to ensure DOM layout is complete before fitting.
     */
    initAspect() {
        const dropdown = document.getElementById('aspect-preset');
        const mainContainer = document.getElementById('maincontainer');

        const calculateAndApplySize = () => {
            const workspace = document.getElementById('workspace-container');
            const availableW = workspace.clientWidth * 0.9;
            const availableH = workspace.clientHeight * 0.9;
            let targetW = availableW;
            let targetH = targetW / this.aspectRatio;
            if (targetH > availableH) {
                targetH = availableH;
                targetW = targetH * this.aspectRatio;
            }
            mainContainer.style.width = `${targetW}px`;
            mainContainer.style.height = `${targetH}px`;
        };

        dropdown.addEventListener('change', (e) => {
            mainContainer.classList.remove('is-resizing');
            const [w, h] = e.target.value.split('/').map(Number);
            this.aspectRatio = w / h;
            calculateAndApplySize();
        });

        const observer = new ResizeObserver(() => {
            mainContainer.classList.add('is-resizing');
            calculateAndApplySize();
            setTimeout(() => mainContainer.classList.remove('is-resizing'), 150);
        });
        observer.observe(document.getElementById('workspace-container'));

        const [startW, startH] = dropdown.value.split('/').map(Number);
        this.aspectRatio = startW / startH;
        calculateAndApplySize();
        
        requestAnimationFrame(() => this.fitToScreen());
    }

    initWorkspaceBg() {
        const colorInput = document.getElementById('workspace-bg-color');
        const textInput = document.getElementById('workspace-bg-text');
        const imageSelect = document.getElementById('workspace-bg-image');
        const fitSelect = document.getElementById('workspace-bg-fit');
        const mainContainer = document.getElementById('maincontainer');

        colorInput.addEventListener('input', () => {
            mainContainer.style.setProperty('--workspace-bg', colorInput.value);
            textInput.value = colorInput.value.toUpperCase();
        });

        textInput.addEventListener('input', () => {
            let val = textInput.value.trim();
            if (!val.startsWith('#')) val = '#' + val;
            if (/^#([A-Fa-f0-9]{3}|[A-Fa-f0-9]{6}|[A-Fa-f0-9]{8})$/.test(val)) {
                mainContainer.style.setProperty('--workspace-bg', val);
                colorInput.value = val.length >= 7 ? val.substring(0, 7) : val;
            }
        });

        imageSelect.addEventListener('change', () => {
            this.applyBackground(mainContainer, imageSelect.value, fitSelect.value);
        });

        fitSelect.addEventListener('change', () => {
            this.applyBackground(mainContainer, imageSelect.value, fitSelect.value);
        });

        this.loadAssets(imageSelect);
    }

    async loadAssets(imageSelect) {
        try {
            const res = await fetch('/assets/');
            if (!res.ok) return;
            const text = await res.text();
            const parser = new DOMParser();
            const doc = parser.parseFromString(text, 'text/html');
            const links = doc.querySelectorAll('a[href]');
            const assets = Array.from(links)
                .map(a => a.getAttribute('href'))
                .filter(h => /\.(png|jpg|jpeg|svg|webp|gif)$/i.test(h))
                .map(h => h.split('/').pop());

            assets.forEach(asset => {
                const opt = document.createElement('option');
                opt.value = asset;
                opt.textContent = asset;
                imageSelect.appendChild(opt);
            });
        } catch (err) {
            console.error('Could not read assets', err);
        }
    }

    applyBackground(mainContainer, imageFile, fit) {
        if (imageFile && imageFile !== 'none') {
            let bgSize, bgRepeat;
            switch (fit) {
                case 'cover':
                    bgSize = 'cover';
                    bgRepeat = 'no-repeat';
                    break;
                case 'contain':
                    bgSize = 'contain';
                    bgRepeat = 'no-repeat';
                    break;
                case 'stretch':
                    bgSize = '100% 100%';
                    bgRepeat = 'no-repeat';
                    break;
                case 'tile':
                    bgSize = 'auto';
                    bgRepeat = 'repeat';
                    break;
                default:
                    bgSize = 'cover';
                    bgRepeat = 'no-repeat';
            }
            mainContainer.style.backgroundImage = `url('/assets/${imageFile}')`;
            mainContainer.style.backgroundSize = bgSize;
            mainContainer.style.backgroundRepeat = bgRepeat;
            mainContainer.style.backgroundPosition = 'center';
        } else {
            mainContainer.style.backgroundImage = 'none';
        }
    }

    getGridSize() {
        return this.gridSize;
    }

    isSnapEnabled() {
        return this.snapEnabled;
    }

    areGuidesEnabled() {
        return this.guidesEnabled;
    }
}
