class Editor {
    constructor() {
        this.state = new EditorState();
        this.workspace = new Workspace(this.state);
        this.blockLibrary = new BlockLibrary();
        this.blockRenderer = new BlockRenderer(this.state, this.workspace);
        this.dragDrop = new DragDropManager(this.state, this.workspace, this.blockRenderer);
        this.selection = new SelectionManager(this.state, this.workspace);
        this.propertiesPanel = new PropertiesPanel(this.state);
        this.panelManager = new PanelManager(this.state);
        this.bindingManager = new BindingManager(this.state);
        this.speechCenter = new SpeechCenter(this.state);
        this.layersPanel = new LayersPanel(this.state);
    }

    async init() {
        console.log('OmniPanel-go Editor initializing...');

        this.workspace.init();
        this.dragDrop.init();
        this.selection.init();
        this.panelManager.init();
        this.bindingManager.init();
        this.speechCenter.init();
        this.layersPanel.init();

        await this.blockLibrary.init();
        await this.propertiesPanel.init();

        this.initTabs();
        this.initUndoRedo();
        this.initDeleteModal();

        const urlParams = new URLSearchParams(window.location.search);
        const isNewPanel = urlParams.get('new') === 'true' || urlParams.get('isNew') === 'true';

        if (!isNewPanel) {
            await this.loadLastPanel();
        }

        console.log('Editor initialized');
    }

    initTabs() {
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                btn.classList.add('active');
                document.getElementById(`tab-${btn.dataset.tab}`).classList.add('active');
            });
        });
    }

    initUndoRedo() {
        document.getElementById('btn-undo').onclick = () => this.state.undo();
        document.getElementById('btn-redo').onclick = () => this.state.redo();

        document.addEventListener('keydown', (e) => {
            if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
                e.preventDefault();
                this.state.undo();
            }
            if ((e.ctrlKey || e.metaKey) && e.key === 'z' && e.shiftKey) {
                e.preventDefault();
                this.state.redo();
            }
            if ((e.ctrlKey || e.metaKey) && e.key === 'y') {
                e.preventDefault();
                this.state.redo();
            }
            if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                this.panelManager.savePanel();
            }
            if (e.key === 'Delete' && this.state.selectedBlocks.size > 0) {
                const blocks = this.state.getSelectedBlocks();
                blocks.forEach(block => {
                    this.blockRenderer.deleteBlock(block);
                });
            }
        });
    }

    initDeleteModal() {
        const modal = document.getElementById('delete-modal');
        document.getElementById('modal-cancel').onclick = () => {
            modal.classList.add('hidden');
        };
        document.getElementById('modal-confirm').onclick = () => {
            modal.classList.add('hidden');
        };
    }

    async loadLastPanel() {
        try {
            const panelsRes = await fetch('/api/panels');
            const panels = await panelsRes.json();
            if (panels.allPanels && panels.allPanels.length > 0) {
                await this.panelManager.loadPanel(panels.allPanels[0]);
            }
        } catch (e) {
            console.error('Failed to load last panel:', e);
        }
    }
}

class LayersPanel {
    constructor(state) {
        this.state = state;
    }

    init() {
        document.addEventListener('blocks-changed', () => this.update());
        document.addEventListener('block-selected', (e) => this.highlightLayer(e.detail.block.id));
    }

    update() {
        const tree = document.getElementById('layers-tree');
        const mainContainer = document.getElementById('maincontainer');
        tree.innerHTML = '';

        const rootList = document.createElement('ul');
        rootList.className = 'block-list';

        this.buildTree(mainContainer, rootList);
        tree.appendChild(rootList);
    }

    buildTree(parentDOMNode, parentUL) {
        const children = this.getDirectBlockChildren(parentDOMNode);

        children.forEach(block => {
            const li = document.createElement('li');
            const label = document.createElement('span');
            label.className = 'tree-label';

            if (!block.id) block.id = generateId();
            label.dataset.blockTarget = block.id;

            const settings = block.settings || {};
            label.textContent = settings.label || block.id || 'Unnamed';

            label.addEventListener('click', (e) => {
                e.stopPropagation();
                this.state.selectBlock(block.id);
                document.dispatchEvent(new CustomEvent('block-selected', { detail: { block } }));
                if (window.editor && window.editor.propertiesPanel) {
                    window.editor.propertiesPanel.showProperties(block);
                }
                let parentFolder = label.closest('.block-folder');
                while (parentFolder) {
                    parentFolder.classList.remove('collapsed');
                    parentFolder = parentFolder.parentElement.closest('.block-folder');
                }
            });

            const dupBtn = document.createElement('button');
            dupBtn.className = 'layer-dup-btn';
            dupBtn.innerHTML = '\u29C9';
            dupBtn.title = 'Duplicate';
            dupBtn.addEventListener('click', (e) => {
                e.stopPropagation();
                if (window.editor && window.editor.blockRenderer) {
                    window.editor.blockRenderer.duplicateBlock(block);
                }
            });

            const subChildren = this.getDirectBlockChildren(block);
            if (subChildren.length > 0) {
                li.className = 'block-folder';
                li.appendChild(label);
                const subUl = document.createElement('ul');
                subUl.className = 'block-list';
                this.buildTree(block, subUl);
                li.appendChild(subUl);
                li.appendChild(dupBtn);
            } else {
                li.className = 'block-file';
                li.appendChild(label);
                li.appendChild(dupBtn);
            }

            parentUL.appendChild(li);
        });
    }

    getDirectBlockChildren(parentElement) {
        return Array.from(parentElement.querySelectorAll('.loaded-block')).filter(block => {
            let parentBlock = block.parentElement.closest('.loaded-block');
            return parentElement.id === 'maincontainer' ? !parentBlock : parentBlock === parentElement;
        });
    }

    highlightLayer(blockId) {
        document.querySelectorAll('.tree-label.active-layer').forEach(l => l.classList.remove('active-layer'));
        const label = document.querySelector(`.tree-label[data-block-target="${blockId}"]`);
        if (label) {
            label.classList.add('active-layer');
            let parentFolder = label.closest('.block-folder');
            while (parentFolder) {
                parentFolder.classList.remove('collapsed');
                parentFolder = parentFolder.parentElement.closest('.block-folder');
            }
        }
    }
}

let editor;
document.addEventListener('DOMContentLoaded', () => {
    editor = new Editor();
    editor.init();
    window.editor = editor;
});
