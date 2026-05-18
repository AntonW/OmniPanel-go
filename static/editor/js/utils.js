function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function generateId() {
    return `block_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
}

function pathToBlockUrl(filePath) {
    const idx = filePath.indexOf('blocks/');
    if (idx === -1) return filePath;
    return '/blocks/' + filePath.substring(idx + 7);
}

function debounce(fn, delay) {
    let timer;
    return function (...args) {
        clearTimeout(timer);
        timer = setTimeout(() => fn.apply(this, args), delay);
    };
}

function clamp(value, min, max) {
    return Math.max(min, Math.min(max, value));
}

function showToast(message, type = 'info') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);
    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(10px)';
        toast.style.transition = 'all 0.2s';
        setTimeout(() => toast.remove(), 200);
    }, 3000);
}

function createModal(options) {
    const overlay = document.getElementById('modal-overlay');
    const container = document.getElementById('modal-container');

    let actionsHtml = '';
    if (options.actions) {
        actionsHtml = `<div class="modal-actions">
            ${options.actions.map(a => `<button class="${a.class || ''}" data-action="${a.id || ''}">${a.label}</button>`).join('')}
        </div>`;
    }

    container.innerHTML = `
        <div class="modal-content">
            ${options.title ? `<h3>${escapeHtml(options.title)}</h3>` : ''}
            ${options.content || ''}
            ${actionsHtml}
        </div>
    `;

    overlay.classList.remove('hidden');

    const close = () => overlay.classList.add('hidden');

    container.querySelectorAll('.modal-actions button').forEach(btn => {
        btn.onclick = () => {
            if (btn.dataset.action === 'cancel' || btn.classList.contains('secondary')) {
                close();
            }
            if (options.onAction) {
                options.onAction(btn.dataset.action);
            }
        };
    });

    overlay.onclick = (e) => {
        if (e.target === overlay) close();
    };

    return { close, element: overlay };
}

function parsePercentage(value) {
    if (typeof value === 'string') {
        return parseFloat(value.replace('%', '')) || 0;
    }
    return parseFloat(value) || 0;
}

function formatPercentage(value) {
    return `${value}%`;
}

function levenshteinDistance(a, b) {
    const matrix = [];
    for (let i = 0; i <= b.length; i++) matrix[i] = [i];
    for (let j = 0; j <= a.length; j++) matrix[0][j] = j;
    for (let i = 1; i <= b.length; i++) {
        for (let j = 1; j <= a.length; j++) {
            if (b.charAt(i - 1) === a.charAt(j - 1)) {
                matrix[i][j] = matrix[i - 1][j - 1];
            } else {
                matrix[i][j] = Math.min(
                    matrix[i - 1][j - 1] + 1,
                    matrix[i][j - 1] + 1,
                    matrix[i - 1][j] + 1
                );
            }
        }
    }
    return matrix[b.length][a.length];
}

function isSimilar(a, b) {
    const normalize = s => s.toLowerCase().replace(/[^a-z0-9]/g, '');
    const na = normalize(a);
    const nb = normalize(b);
    if (na === nb) return true;
    const maxLen = Math.max(na.length, nb.length);
    if (maxLen === 0) return true;
    const distance = levenshteinDistance(na, nb);
    return distance / maxLen < 0.2;
}
