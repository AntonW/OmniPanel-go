class DragDropManager {
    constructor(state, workspace, blockRenderer) {
        this.state = state;
        this.workspace = workspace;
        this.blockRenderer = blockRenderer;
        this.guideLines = [];
        this.ghostBlock = null;
        this.snapThreshold = 3;
    }

    init() {
        const mainContainer = document.getElementById('maincontainer');

        mainContainer.addEventListener('dragover', (e) => this.handleDragOver(e));
        mainContainer.addEventListener('dragleave', (e) => this.handleDragLeave(e));
        mainContainer.addEventListener('drop', (e) => this.handleDrop(e));
    }

    handleDragOver(e) {
        e.preventDefault();

        const draggedBlock = window.draggedElement;
        if (!draggedBlock) return;

        const targetDropZone = e.target.closest('.nested-dropzone') || document.getElementById('maincontainer');
        const action = e.dataTransfer.getData('action');

        if (action === 'move') {
            this.handleMoveDragOver(e, draggedBlock, targetDropZone);
        } else {
            this.handleNewBlockDragOver(e, targetDropZone);
        }
    }

    handleMoveDragOver(e, draggedBlock, targetDropZone) {
        if (draggedBlock.contains(targetDropZone)) return;

        const rect = targetDropZone.getBoundingClientRect();
        const offset = JSON.parse(e.dataTransfer.getData('offset') || '{"x":0,"y":0}');

        const gX = parseInt(targetDropZone.dataset.gridx) || this.workspace.getGridSize();
        const gY = parseInt(targetDropZone.dataset.gridy) || this.workspace.getGridSize();
        const sX = 100 / gX;
        const sY = 100 / gY;

        const physWidth = draggedBlock.offsetWidth;
        const physHeight = draggedBlock.offsetHeight;
        let wPct = (physWidth / rect.width) * 100;
        let hPct = (physHeight / rect.height) * 100;
        let snappedW = clamp(Math.round(wPct / sX) * sX, sX, 100);
        let snappedH = clamp(Math.round(hPct / sY) * sY, sY, 100);

        let lPct = ((e.clientX - rect.left - offset.x) / rect.width) * 100;
        let tPct = ((e.clientY - rect.top - offset.y) / rect.height) * 100;
        let snappedL = Math.floor(lPct / sX) * sX;
        let snappedT = Math.floor(tPct / sY) * sY;
        snappedL = clamp(snappedL, 0, 100 - snappedW);
        snappedT = clamp(snappedT, 0, 100 - snappedH);

        if (!this.ghostBlock) {
            this.ghostBlock = document.createElement('div');
            this.ghostBlock.className = 'block-placeholder';
        }
        if (this.ghostBlock.parentElement !== targetDropZone) {
            targetDropZone.appendChild(this.ghostBlock);
        }

        this.ghostBlock.style.width = `${snappedW}%`;
        this.ghostBlock.style.height = `${snappedH}%`;
        this.ghostBlock.style.left = `${snappedL}%`;
        this.ghostBlock.style.top = `${snappedT}%`;

        if (this.workspace.areGuidesEnabled()) {
            this.updateAlignmentGuides(draggedBlock);
        }

        targetDropZone.classList.add('drag-over');
    }

    handleNewBlockDragOver(e, targetDropZone) {
        const rect = targetDrop.getBoundingClientRect();
        const draggedCard = document.querySelector('.block-card:hover');
        if (!draggedCard) return;

        const gX = parseInt(targetDropZone.dataset.gridx) || this.workspace.getGridSize();
        const gY = parseInt(targetDropZone.dataset.gridy) || this.workspace.getGridSize();
        const sX = 100 / gX;
        const sY = 100 / gY;

        const cardRect = draggedCard.getBoundingClientRect();
        let wPct = (cardRect.width / rect.width) * 100;
        let hPct = (cardRect.height / rect.height) * 100;
        let snappedW = Math.max(sX, Math.round(wPct / sX) * sX);
        let snappedH = Math.max(sY, Math.round(hPct / sY) * sY);

        let lPct = ((e.clientX - rect.left) / rect.width) * 100 - sX;
        let tPct = ((e.clientY - rect.top) / rect.height) * 100 - sY;
        let snappedL = Math.floor(lPct / sX) * sX;
        let snappedT = Math.floor(tPct / sY) * sY;
        snappedL = clamp(snappedL, 0, 100 - snappedW);
        snappedT = clamp(snappedT, 0, 100 - snappedH);

        if (!this.ghostBlock) {
            this.ghostBlock = document.createElement('div');
            this.ghostBlock.className = 'block-placeholder';
        }
        if (this.ghostBlock.parentElement !== targetDropZone) {
            targetDropZone.appendChild(this.ghostBlock);
        }

        this.ghostBlock.style.width = `${snappedW}%`;
        this.ghostBlock.style.height = `${snappedH}%`;
        this.ghostBlock.style.left = `${snappedL}%`;
        this.ghostBlock.style.top = `${snappedT}%`;

        targetDropZone.classList.add('drag-over');
    }

    handleDragLeave(e) {
        const targetDropZone = e.target.closest('.nested-dropzone');
        if (targetDropZone) {
            targetDropZone.classList.remove('drag-over');
        }
    }

    async handleDrop(e) {
        e.preventDefault();
        this.clearGuides();
        if (this.ghostBlock) {
            this.ghostBlock.remove();
            this.ghostBlock = null;
        }

        const targetDropZone = e.target.closest('.nested-dropzone') || document.getElementById('maincontainer');
        targetDropZone.classList.remove('drag-over');

        const action = e.dataTransfer.getData('action');

        if (action === 'move' && window.draggedElement) {
            const block = window.draggedElement;
            let finalTarget = targetDropZone;
            if (block.contains(finalTarget)) {
                finalTarget = block.parentElement.closest('.nested-dropzone') || document.getElementById('maincontainer');
            }

            const pRect = finalTarget.getBoundingClientRect();
            const offset = JSON.parse(e.dataTransfer.getData('offset'));
            const gX = parseInt(finalTarget.dataset.gridx) || this.workspace.getGridSize();
            const gY = parseInt(finalTarget.dataset.gridy) || this.workspace.getGridSize();
            const sX = 100 / gX;
            const sY = 100 / gY;

            const physWidth = block.offsetWidth;
            const physHeight = block.offsetHeight;
            let wPct = (physWidth / pRect.width) * 100;
            let hPct = (physHeight / pRect.height) * 100;
            let snappedW = clamp(Math.round(wPct / sX) * sX, sX, 100);
            let snappedH = clamp(Math.round(hPct / sY) * sY, sY, 100);

            let lPct = ((e.clientX - pRect.left - offset.x) / pRect.width) * 100;
            let tPct = ((e.clientY - pRect.top - offset.y) / pRect.height) * 100;
            let snappedL = Math.floor(lPct / sX) * sX;
            let snappedT = Math.floor(tPct / sY) * sY;
            snappedL = clamp(snappedL, 0, 100 - snappedW);
            snappedT = clamp(snappedT, 0, 100 - snappedH);

            const oldPos = { left: block.style.left, top: block.style.top };
            const newPos = { left: `${snappedL}%`, top: `${snappedT}%` };

            Object.assign(block.style, { width: `${snappedW}%`, height: `${snappedH}%`, ...newPos });

            if (block.parentElement !== finalTarget) {
                finalTarget.appendChild(block);
            }

            this.state.pushState({
                type: 'block-moved',
                blockId: block.id,
                oldPos,
                newPos
            });

            this.blockRenderer.notifyBlocksChanged();
        } else {
            const filePath = e.dataTransfer.getData('text/plain');
            if (filePath && filePath.endsWith('.html')) {
                await this.blockRenderer.createFromDrop(filePath, e, targetDropZone);
            }
        }
    }

    updateAlignmentGuides(block) {
        this.clearGuides();

        const blockRect = block.getBoundingClientRect();
        const allBlocks = Array.from(document.querySelectorAll('.loaded-block')).filter(b => b !== block);
        const threshold = this.snapThreshold;

        for (const other of allBlocks) {
            const otherRect = other.getBoundingClientRect();

            if (Math.abs(blockRect.left - otherRect.left) < threshold) {
                this.addGuide('vertical', otherRect.left);
            }
            if (Math.abs(blockRect.right - otherRect.right) < threshold) {
                this.addGuide('vertical', otherRect.right);
            }
            if (Math.abs(blockRect.top - otherRect.top) < threshold) {
                this.addGuide('horizontal', otherRect.top);
            }
            if (Math.abs(blockRect.bottom - otherRect.bottom) < threshold) {
                this.addGuide('horizontal', otherRect.bottom);
            }

            const blockCenterX = blockRect.left + blockRect.width / 2;
            const otherCenterX = otherRect.left + otherRect.width / 2;
            if (Math.abs(blockCenterX - otherCenterX) < threshold) {
                this.addGuide('vertical', otherCenterX);
            }

            const blockCenterY = blockRect.top + blockRect.height / 2;
            const otherCenterY = otherRect.top + otherRect.height / 2;
            if (Math.abs(blockCenterY - otherCenterY) < threshold) {
                this.addGuide('horizontal', otherCenterY);
            }
        }
    }

    addGuide(type, position) {
        const viewport = document.getElementById('workspace-viewport');
        const line = document.createElement('div');
        line.className = `alignment-guide ${type}`;
        if (type === 'vertical') {
            line.style.left = `${position}px`;
        } else {
            line.style.top = `${position}px`;
        }
        viewport.appendChild(line);
        this.guideLines.push(line);
    }

    clearGuides() {
        this.guideLines.forEach(line => line.remove());
        this.guideLines = [];
    }
}
