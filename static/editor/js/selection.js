class SelectionManager {
    constructor(state, workspace) {
        this.state = state;
        this.workspace = workspace;
        this.isSelecting = false;
        this.startPoint = null;
        this.selectionRect = null;
    }

    init() {
        const mainContainer = document.getElementById('maincontainer');
        const viewport = document.getElementById('workspace-viewport');

        viewport.addEventListener('mousedown', (e) => {
            if (e.target === viewport || e.target.id === 'workspace-canvas' || e.target.id === 'maincontainer') {
                if (e.button === 0 && !e.shiftKey) {
                    this.startSelectionRectangle(e);
                }
            }
        });

        window.addEventListener('mousemove', (e) => {
            if (this.isSelecting) {
                this.updateSelectionRectangle(e);
            }
        });

        window.addEventListener('mouseup', () => {
            if (this.isSelecting) {
                this.endSelectionRectangle();
            }
        });

        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                this.state.clearSelection();
                document.dispatchEvent(new CustomEvent('selection-cleared'));
            }
        });
    }

    startSelectionRectangle(e) {
        this.isSelecting = true;
        this.startPoint = { x: e.clientX, y: e.clientY };

        this.selectionRect = document.createElement('div');
        this.selectionRect.className = 'selection-rectangle';
        document.getElementById('workspace-viewport').appendChild(this.selectionRect);
    }

    updateSelectionRectangle(e) {
        if (!this.selectionRect || !this.startPoint) return;

        const x = Math.min(this.startPoint.x, e.clientX);
        const y = Math.min(this.startPoint.y, e.clientY);
        const width = Math.abs(e.clientX - this.startPoint.x);
        const height = Math.abs(e.clientY - this.startPoint.y);

        this.selectionRect.style.left = `${x}px`;
        this.selectionRect.style.top = `${y}px`;
        this.selectionRect.style.width = `${width}px`;
        this.selectionRect.style.height = `${height}px`;
    }

    endSelectionRectangle() {
        if (!this.selectionRect || !this.startPoint) {
            this.isSelecting = false;
            return;
        }

        const rect = this.selectionRect.getBoundingClientRect();
        const blocks = document.querySelectorAll('.loaded-block');

        if (rect.width > 5 || rect.height > 5) {
            this.state.clearSelection();

            blocks.forEach(block => {
                const blockRect = block.getBoundingClientRect();
                if (this.intersects(rect, blockRect)) {
                    this.state.addSelection(block.id);
                }
            });
        }

        this.selectionRect.remove();
        this.selectionRect = null;
        this.isSelecting = false;
    }

    intersects(a, b) {
        return !(a.right < b.left || a.left > b.right || a.bottom < b.top || a.top > b.bottom);
    }
}
