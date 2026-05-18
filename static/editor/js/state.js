/**
 * EditorState manages the editor's undo/redo history (100 steps), block selection,
 * and dirty state. Actions are stored as minimal diffs (not full snapshots) using
 * the command pattern. The clearSelection method dispatches a selection-cleared
 * event so other UI components can react without tight coupling.
 */
class EditorState {
    constructor() {
        this.history = [];
        this.historyIndex = -1;
        this.maxHistory = 100;
        this.currentPanel = null;
        this.selectedBlocks = new Set();
        this.isDirty = false;
        this.isUndoRedoInProgress = false;
    }

    /**
     * Pushes an action to the undo/redo history.
     * Trims future states if currently in the middle of history.
     * Does nothing during undo/redo operations to prevent recursive history changes.
     * @param {Object} action - Action object with type and relevant data.
     */
    pushState(action) {
        if (this.isUndoRedoInProgress) return;

        if (this.historyIndex < this.history.length - 1) {
            this.history = this.history.slice(0, this.historyIndex + 1);
        }

        this.history.push({
            action: { ...action },
            timestamp: Date.now()
        });

        if (this.history.length > this.maxHistory) {
            this.history.shift();
        } else {
            this.historyIndex++;
        }

        this.isDirty = true;
        this.updateUndoRedoButtons();
    }

    undo() {
        if (!this.canUndo()) return;
        this.isUndoRedoInProgress = true;
        const state = this.history[this.historyIndex];
        this.applyReverse(state.action);
        this.historyIndex--;
        this.isUndoRedoInProgress = false;
        this.updateUndoRedoButtons();
    }

    redo() {
        if (!this.canRedo()) return;
        this.isUndoRedoInProgress = true;
        this.historyIndex++;
        const state = this.history[this.historyIndex];
        this.applyForward(state.action);
        this.isUndoRedoInProgress = false;
        this.updateUndoRedoButtons();
    }

    applyReverse(action) {
        switch (action.type) {
            case 'block-added':
                this.removeBlockById(action.blockId);
                break;
            case 'block-removed':
                this.restoreBlock(action.blockData, action.parentId);
                break;
            case 'block-moved':
                this.moveBlock(action.blockId, action.oldPos);
                break;
            case 'block-resized':
                this.resizeBlock(action.blockId, action.oldSize);
                break;
            case 'setting-changed':
                this.setBlockSetting(action.blockId, action.key, action.oldValue);
                break;
            case 'block-duplicated':
                this.removeBlockById(action.newId);
                break;
        }
        this.refreshUI();
    }

    applyForward(action) {
        switch (action.type) {
            case 'block-added':
                this.restoreBlock(action.blockData, action.parentId);
                break;
            case 'block-removed':
                this.removeBlockById(action.blockId);
                break;
            case 'block-moved':
                this.moveBlock(action.blockId, action.newPos);
                break;
            case 'block-resized':
                this.resizeBlock(action.blockId, action.newSize);
                break;
            case 'setting-changed':
                this.setBlockSetting(action.blockId, action.key, action.newValue);
                break;
            case 'block-duplicated':
                this.duplicateBlockById(action.originalId, action.newId);
                break;
        }
        this.refreshUI();
    }

    removeBlockById(blockId) {
        const block = document.getElementById(blockId);
        if (block) block.remove();
    }

    restoreBlock(blockData, parentId) {
        if (window.editor && window.editor.blockRenderer) {
            window.editor.blockRenderer.createFromData(blockData, parentId ? document.getElementById(parentId) : document.getElementById('maincontainer'));
        }
    }

    moveBlock(blockId, pos) {
        const block = document.getElementById(blockId);
        if (block) {
            Object.assign(block.style, { left: pos.left, top: pos.top });
        }
    }

    resizeBlock(blockId, size) {
        const block = document.getElementById(blockId);
        if (block) {
            Object.assign(block.style, { width: size.width, height: size.height });
        }
    }

    setBlockSetting(blockId, key, value) {
        const block = document.getElementById(blockId);
        if (block && block.settings) {
            block.settings[key] = value;
            if (window.editor && window.editor.blockRenderer) {
                window.editor.blockRenderer.renderBlock(block);
            }
        }
    }

    duplicateBlockById(originalId, newId) {
        const original = document.getElementById(originalId);
        if (original && window.editor && window.editor.blockRenderer) {
            window.editor.blockRenderer.duplicateBlock(original);
        }
    }

    /**
     * Refreshes all UI components after an undo/redo operation.
     * Updates properties panel, layers, bindings, and speech center.
     */
    refreshUI() {
        if (window.editor) {
            if (window.editor.propertiesPanel) window.editor.propertiesPanel.refresh();
            if (window.editor.layersPanel) window.editor.layersPanel.update();
            if (window.editor.bindingManager) window.editor.bindingManager.refresh();
            if (window.editor.speechCenter) window.editor.speechCenter.refresh();
        }
    }

    canUndo() {
        return this.historyIndex >= 0;
    }

    canRedo() {
        return this.historyIndex < this.history.length - 1;
    }

    updateUndoRedoButtons() {
        const undoBtn = document.getElementById('btn-undo');
        const redoBtn = document.getElementById('btn-redo');
        if (undoBtn) undoBtn.disabled = !this.canUndo();
        if (redoBtn) redoBtn.disabled = !this.canRedo();
    }

    clearHistory() {
        this.history = [];
        this.historyIndex = -1;
        this.isDirty = false;
        this.updateUndoRedoButtons();
    }

    selectBlock(blockId) {
        this.selectedBlocks.clear();
        if (blockId) {
            this.selectedBlocks.add(blockId);
        }
        document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected'));
        if (blockId) {
            const block = document.getElementById(blockId);
            if (block) block.classList.add('selected');
        }
    }

    addSelection(blockId) {
        this.selectedBlocks.add(blockId);
        const block = document.getElementById(blockId);
        if (block) block.classList.add('selected');
    }

    removeSelection(blockId) {
        this.selectedBlocks.delete(blockId);
        const block = document.getElementById(blockId);
        if (block) block.classList.remove('selected');
    }

    /**
     * Clears all selected blocks and dispatches a selection-cleared event.
     * Other UI components (properties panel, binding manager) listen for this event.
     */
    clearSelection() {
        this.selectedBlocks.clear();
        document.querySelectorAll('.loaded-block.selected').forEach(el => el.classList.remove('selected'));
        document.dispatchEvent(new CustomEvent('selection-cleared'));
    }

    getSelectedBlocks() {
        return Array.from(this.selectedBlocks).map(id => document.getElementById(id)).filter(Boolean);
    }
}
