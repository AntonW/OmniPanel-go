class BindingManager {
    constructor(state) {
        this.state = state;
    }

    init() {
        document.addEventListener('blocks-changed', () => this.refresh());
    }

    refresh() {
        const panel = document.getElementById('bindings-panel');
        const blocks = document.querySelectorAll('.loaded-block');

        const bindings = {
            joysticks: {},
            keyboards: {},
            mousepads: {},
            commands: [],
            speech: []
        };

        blocks.forEach(block => {
            const settings = block.settings || {};
            const label = settings.label || block.id;

            if (settings.joystick !== undefined && settings.button !== undefined) {
                const js = settings.joystick;
                if (!bindings.joysticks[js]) bindings.joysticks[js] = { buttons: {}, sliders: {} };
                bindings.joysticks[js].buttons[settings.button] = label;
            }
            if (settings.joystick !== undefined && settings.slider !== undefined) {
                const js = settings.joystick;
                if (!bindings.joysticks[js]) bindings.joysticks[js] = { buttons: {}, sliders: {} };
                bindings.joysticks[js].sliders[settings.slider] = label;
            }

            if (settings.keyboard_key) {
                const kbIdx = settings.keyboard_index || 0;
                if (!bindings.keyboards[kbIdx]) bindings.keyboards[kbIdx] = [];
                bindings.keyboards[kbIdx].push({ key: settings.keyboard_key, label });
            }

            if (settings.mousepad !== undefined) {
                bindings.mousepads[settings.mousepad] = label;
            }

            if (settings.command_type && settings.command) {
                bindings.commands.push({
                    label,
                    type: settings.command_type,
                    command: settings.command,
                    httpMethod: settings.http_method,
                    httpUrl: settings.http_url
                });
            }

            if (settings.speech_trigger) {
                bindings.speech.push({
                    label,
                    trigger: settings.speech_trigger,
                    aliases: settings.speech_aliases,
                    type: settings.speech_type || 'button'
                });
            }
        });

        panel.innerHTML = this.renderBindings(bindings);
    }

    renderBindings(bindings) {
        let html = '';

        if (Object.keys(bindings.joysticks).length > 0) {
            html += '<div class="binding-section"><h4>Virtual Joysticks</h4>';
            for (const [jsIndex, js] of Object.entries(bindings.joysticks)) {
                html += `<h5>Joystick ${jsIndex} - Buttons</h5>`;
                html += '<div class="binding-grid">';
                for (let i = 0; i < 16; i++) {
                    const used = js.buttons[i];
                    html += `<div class="binding-slot ${used ? 'used' : ''}">
                        <span class="slot-label">Btn ${i}</span>
                        <span class="slot-value">${used ? escapeHtml(used) : '\u2014'}</span>
                    </div>`;
                }
                html += '</div>';

                html += `<h5>Joystick ${jsIndex} - Sliders/Axes</h5>`;
                html += '<div class="binding-grid">';
                for (let i = 0; i < 8; i++) {
                    const used = js.sliders[i];
                    html += `<div class="binding-slot ${used ? 'used' : ''}">
                        <span class="slot-label">Axis ${i}</span>
                        <span class="slot-value">${used ? escapeHtml(used) : '\u2014'}</span>
                    </div>`;
                }
                html += '</div>';
            }
            html += '</div>';
        }

        if (Object.keys(bindings.keyboards).length > 0) {
            html += '<div class="binding-section"><h4>Virtual Keyboards</h4>';
            for (const [kbIndex, keys] of Object.entries(bindings.keyboards)) {
                html += `<div class="command-list"><h5>Keyboard ${kbIndex}</h5>`;
                keys.forEach(k => {
                    html += `<div class="command-item">
                        <span class="command-label"><kbd>${escapeHtml(k.key)}</kbd></span>
                        <span class="command-type">\u2192 ${escapeHtml(k.label)}</span>
                    </div>`;
                });
                html += '</div>';
            }
            html += '</div>';
        }

        if (Object.keys(bindings.mousepads).length > 0) {
            html += '<div class="binding-section"><h4>Mousepads</h4>';
            for (const [mpIndex, label] of Object.entries(bindings.mousepads)) {
                html += `<div class="command-item">
                    <span class="command-label">Mousepad ${mpIndex}</span>
                    <span class="command-type">${escapeHtml(label)}</span>
                </div>`;
            }
            html += '</div>';
        }

        if (bindings.commands.length > 0) {
            html += '<div class="binding-section"><h4>Commands</h4><div class="command-list">';
            bindings.commands.forEach(cmd => {
                html += `<div class="command-item">
                    <span class="command-label">${escapeHtml(cmd.label)}</span>
                    <span class="command-type">${cmd.type}</span>
                    <code class="command-code">${escapeHtml(cmd.command || cmd.httpUrl)}</code>
                </div>`;
            });
            html += '</div></div>';
        }

        return html || '<div class="empty-state">No bindings configured</div>';
    }
}
