class SpeechCenter {
    constructor(state) {
        this.state = state;
    }

    init() {
        document.addEventListener('blocks-changed', () => this.refresh());
    }

    refresh() {
        const panel = document.getElementById('speech-panel');
        const blocks = document.querySelectorAll('.loaded-block');

        const speechBlocks = [];
        blocks.forEach(block => {
            if (block.settings.speech_trigger) {
                speechBlocks.push({
                    id: block.id,
                    label: block.settings.label || block.id,
                    trigger: block.settings.speech_trigger,
                    aliases: block.settings.speech_aliases || '',
                    type: block.settings.speech_type || 'button',
                    joystick: block.settings.joystick,
                    button: block.settings.button,
                    slider: block.settings.slider
                });
            }
        });

        panel.innerHTML = `
            <div class="speech-header">
                <h4>Speech Commands</h4>
                <span class="speech-count">${speechBlocks.length} configured</span>
            </div>
            ${speechBlocks.length > 0 ? `
                <div class="speech-list">
                    ${speechBlocks.map(s => this.renderSpeechItem(s)).join('')}
                </div>
            ` : '<div class="empty-state">No speech triggers configured. Add a speech_trigger setting to any block.</div>'}
            <div id="speech-conflicts"></div>
        `;

        this.checkConflicts(speechBlocks);
    }

    renderSpeechItem(speech) {
        let actionText = '';
        if (speech.type === 'button') {
            actionText = `Joystick ${speech.joystick}, Button ${speech.button}`;
        } else if (speech.type === 'slider') {
            actionText = `Joystick ${speech.joystick}, Slider ${speech.slider}`;
        } else {
            actionText = speech.type;
        }

        return `
            <div class="speech-command-item" data-block-id="${escapeHtml(speech.id)}">
                <div class="speech-command-header">
                    <span class="speech-command-label">${escapeHtml(speech.label)}</span>
                    <span class="speech-command-type">${escapeHtml(speech.type)}</span>
                </div>
                <div class="speech-command-details">
                    <div class="speech-trigger-row">
                        <label>Trigger:</label>
                        <code>"${escapeHtml(speech.trigger)}"</code>
                    </div>
                    ${speech.aliases ? `
                        <div class="speech-aliases-row">
                            <label>Aliases:</label>
                            <span>${escapeHtml(speech.aliases)}</span>
                        </div>
                    ` : ''}
                    <div class="speech-action-row">
                        <label>Action:</label>
                        <span>${escapeHtml(actionText)}</span>
                    </div>
                </div>
            </div>
        `;
    }

    checkConflicts(speechBlocks) {
        const conflicts = [];
        for (let i = 0; i < speechBlocks.length; i++) {
            for (let j = i + 1; j < speechBlocks.length; j++) {
                const a = speechBlocks[i];
                const b = speechBlocks[j];
                if (isSimilar(a.trigger, b.trigger)) {
                    conflicts.push({
                        a: a.trigger,
                        b: b.trigger,
                        blocks: [a.label, b.label]
                    });
                }
                if (a.aliases && b.aliases) {
                    const aAliases = a.aliases.split(',').map(s => s.trim());
                    const bAliases = b.aliases.split(',').map(s => s.trim());
                    for (const aAlias of aAliases) {
                        for (const bAlias of bAliases) {
                            if (isSimilar(aAlias, bAlias)) {
                                conflicts.push({
                                    a: `"${a.trigger}" alias: ${aAlias}`,
                                    b: `"${b.trigger}" alias: ${bAlias}`,
                                    blocks: [a.label, b.label]
                                });
                            }
                        }
                    }
                }
            }
        }

        const container = document.getElementById('speech-conflicts');
        if (conflicts.length > 0) {
            container.innerHTML = `
                <div class="conflict-warning">
                    <h5>\u26A0 Potential Conflicts</h5>
                    <ul>
                        ${conflicts.map(c => `<li>${escapeHtml(c.a)} vs ${escapeHtml(c.b)} (${c.blocks.map(escapeHtml).join(' vs ')})</li>`).join('')}
                    </ul>
                </div>
            `;
        } else {
            container.innerHTML = '';
        }
    }
}
