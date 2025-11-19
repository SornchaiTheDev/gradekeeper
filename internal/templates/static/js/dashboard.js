'use strict';

const dashboardSecret = document.body.dataset.dashboardSecret || '';
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsBase = `${wsProtocol}//${window.location.host}/ws`;
let ws;
let previousClientIds = new Set();
let currentConfig = { urls: [] };
let configStatusTimeout = null;
let codeEditor = null;
let currentClientForFiles = null;

// Initialize Lucide icons after DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    if (window.lucide && typeof window.lucide.createIcons === 'function') {
        window.lucide.createIcons();
    }
});

function connect() {
    if (!dashboardSecret) {
        log('Dashboard secret missing, cannot connect');
        return;
    }

    ws = new WebSocket(`${wsBase}?dashboard=${dashboardSecret}`);

    ws.onopen = function() {
        log('Dashboard connected to master server');
        refreshClients();
        loadConfig();
    };

    ws.onmessage = function(event) {
        const data = JSON.parse(event.data);

        switch (data.type) {
            case 'dashboard-welcome':
                log('Dashboard authenticated successfully');
                break;
            case 'client-connected':
                log('Client connected: ' + data.data.clientId + ' (Total: ' + data.data.totalClients + ')');
                setTimeout(() => refreshClients(true), 100);
                break;
            case 'client-disconnected':
                log('Client disconnected: ' + data.data.clientId + ' (Total: ' + data.data.totalClients + ')');
                refreshClients();
                break;
            case 'command-sent':
                log('Command sent: ' + data.data.action + ' to ' + (data.data.target || 'all') + ' (' + data.data.clientCount + ' clients)');
                break;
            case 'client_action_update':
                log('Client ' + data.data.clientId + ': ' + data.data.action + ' -> ' + data.data.status +
                    (data.data.error ? ' (Error: ' + data.data.error + ')' : ''));
                refreshClients();
                break;
            case 'file_data':
                handleFileData(data.data);
                break;
            case 'config_update':
                applyConfigUpdate(data.data);
                const urlCount = (data.data && Array.isArray(data.data.urls)) ? data.data.urls.length : 0;
                log('Configuration updated (' + urlCount + ' URLs)');
                break;
        }
    };

    ws.onclose = function() {
        log('Dashboard disconnected from master server');
        setTimeout(connect, 3000);
    };

    ws.onerror = function(error) {
        log('Dashboard WebSocket error: ' + error);
    };
}

function setupAll() {
    const command = { action: 'setupAll' };
    if (ws && ws.readyState === WebSocket.OPEN) {
        fetch('/api/command', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(command)
        });
        log('Sent command: setupAll to all clients');
    }
}

function clearAll() {
    if (confirm('⚠️ This will clear all environments on all clients. Are you sure?')) {
        const command = { action: 'clear' };
        if (ws && ws.readyState === WebSocket.OPEN) {
            fetch('/api/command', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(command)
            });
            log('Sent command: clear to all clients');
        }
    }
}

function refreshClients(animateNewClient = false) {
    fetch('/api/clients')
        .then(response => response.json())
        .then(clients => {
            const container = document.getElementById('clients');
            const currentClientIds = new Set(clients.map(c => c.id));

            container.innerHTML = clients.map(client => {
                const isConnected = client.status === 'connected';
                const hasFailed = client.actionStatus === 'failed';
                const isNewClient = !previousClientIds.has(client.id) && isConnected;

                const statusColor = hasFailed ? 'border-l-red-500' : (isConnected ? 'border-l-success' : 'border-l-gray-400');
                const statusBadge = hasFailed ? 'bg-red-100 text-red-800' :
                    (isConnected ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-600');
                const statusText = hasFailed ? 'Failed' :
                    (isConnected ? 'Connected' : 'Disconnected');
                const statusIcon = hasFailed ? 'alert-triangle' :
                    (isConnected ? 'wifi' : 'wifi-off');

                const actionStatus = client.actionStatus ?
                    `<div class="mt-3 text-sm ${client.actionStatus === 'failed' ? 'text-red-600' :
                        (client.actionStatus === 'success' ? 'text-green-600' : 'text-blue-600')}">
                        <span class="font-semibold">Action:</span>
                        ${client.action ? client.action : 'Unknown'} - ${client.actionStatus}
                        ${client.actionError ? `<div class="text-red-500 text-xs">Error: ${client.actionError}</div>` : ''}
                    </div>` : '';

                const actionButtons = isConnected ? `
                    <div class="flex gap-2 mt-4">
                        <button onclick="sendCommand('${client.id}', 'setup')" class="bg-primary hover:bg-blue-700 text-white px-3 py-2 rounded-md text-sm flex items-center gap-2">
                            <i data-lucide="tool" class="w-4 h-4"></i>
                            Setup
                        </button>
                        <button onclick="sendCommand('${client.id}', 'clear')" class="bg-danger hover:bg-red-700 text-white px-3 py-2 rounded-md text-sm flex items-center gap-2">
                            <i data-lucide="trash-2" class="w-4 h-4"></i>
                            Clear
                        </button>
                        <button onclick="showFileViewer('${client.id}', '${client.name}')" class="bg-gray-700 hover:bg-gray-800 text-white px-3 py-2 rounded-md text-sm flex items-center gap-2">
                            <i data-lucide="folder-open" class="w-4 h-4"></i>
                            Files
                        </button>
                    </div>` : `
                    <p class="text-sm text-gray-500 mt-4">Client offline. Commands disabled.</p>`;

                const cardClasses = [
                    'bg-white rounded-lg shadow-sm border border-gray-100 p-5',
                    'transition-all duration-300 hover:shadow-md',
                    'border-l-4 ' + statusColor,
                    'relative overflow-hidden'
                ];

                if (animateNewClient && isNewClient) {
                    cardClasses.push('client-card-animated');
                }
                if (isConnected) {
                    cardClasses.push('client-card-connected');
                }

                return `
                    <div class="${cardClasses.join(' ')}">
                        <div class="flex items-center gap-4">
                            <div class="bg-blue-100 text-blue-600 rounded-full p-3">
                                <i data-lucide="monitor" class="w-6 h-6"></i>
                            </div>
                            <div class="flex-1">
                                <div class="flex items-center gap-3">
                                    <h3 class="text-lg font-semibold text-gray-800">${client.name}</h3>
                                    <span class="text-xs ${statusBadge} px-2 py-1 rounded-full flex items-center gap-1">
                                        <i data-lucide="${statusIcon}" class="w-3 h-3"></i>
                                        ${statusText}
                                    </span>
                                </div>
                                <p class="text-sm text-gray-500">${client.id}</p>
                                <p class="text-xs text-gray-400">First seen: ${new Date(client.firstSeen).toLocaleString()}</p>
                            </div>
                        </div>
                        <div class="mt-4 border-t border-gray-100 pt-4">
                            <div class="grid grid-cols-2 gap-4 text-sm">
                                <div>
                                    <p class="text-gray-500 text-xs">Last Seen</p>
                                    <p class="font-medium text-gray-800">${new Date(client.lastSeen).toLocaleString()}</p>
                                </div>
                                <div>
                                    <p class="text-gray-500 text-xs">Last Action</p>
                                    <p class="font-medium text-gray-800">${client.action || 'None'}</p>
                                </div>
                            </div>
                            ${actionStatus}
                            ${actionButtons}
                        </div>
                    </div>
                `;
            }).join('');

            previousClientIds = currentClientIds;
            if (window.lucide && typeof window.lucide.createIcons === 'function') {
                window.lucide.createIcons();
            }
        });
}

function sendCommand(clientId, action) {
    const command = {
        action: action,
        target: clientId
    };

    fetch('/api/command', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(command)
    });

    log('Sent command: ' + action + ' to ' + clientId);
}

function refreshClientsWithAnimation() {
    refreshClients(true);
}

function showFileViewer(clientId, clientName) {
    const modal = document.getElementById('fileViewerModal');
    const title = document.getElementById('modalTitle');
    const fileList = document.getElementById('fileList');
    const fileContent = document.getElementById('fileContent');
    const noFileSelected = document.getElementById('noFileSelected');

    currentClientForFiles = clientId;
    title.textContent = 'Files - ' + clientName;
    fileList.innerHTML = '<p class="text-gray-500 text-sm">Loading files...</p>';
    fileContent.classList.add('hidden');
    noFileSelected.classList.remove('hidden');

    modal.classList.remove('hidden');
    document.body.classList.add('overflow-hidden');

    fetch('/api/files', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ clientId, action: 'list' })
    });
}

function closeFileViewer() {
    const modal = document.getElementById('fileViewerModal');
    if (modal) {
        modal.classList.add('hidden');
        document.body.classList.remove('overflow-hidden');
        currentClientForFiles = null;
    }
}

function handleFileData(data) {
    if (!data || !data.clientId || data.clientId !== currentClientForFiles) {
        return;
    }

    if (data.type === 'list') {
        renderFileList(data.files || []);
    } else if (data.type === 'file') {
        displayFileContent(data.file);
    }
}

function renderFileList(files) {
    const fileList = document.getElementById('fileList');
    if (!fileList) return;

    if (!files || files.length === 0) {
        fileList.innerHTML = '<p class="text-gray-500 text-sm">No files available</p>';
        return;
    }

    fileList.innerHTML = files.map(file => {
        const icon = getFileIcon(file.ext);
        const size = formatFileSize(file.size);

        return `
            <div class="bg-white border border-gray-200 rounded-lg p-3 hover:border-blue-400 cursor-pointer transition-colors"
                onclick="requestFile('${file.path}')">
                <div class="flex items-center gap-3">
                    <div class="text-blue-500 bg-blue-50 rounded-full p-2">
                        <i data-lucide="${icon}" class="w-4 h-4"></i>
                    </div>
                    <div>
                        <p class="text-sm font-medium text-gray-800">${escapeHtml(file.name)}</p>
                        <p class="text-xs text-gray-500">${size}</p>
                    </div>
                </div>
            </div>
        `;
    }).join('');

    if (window.lucide && typeof window.lucide.createIcons === 'function') {
        window.lucide.createIcons();
    }
}

function requestFile(filePath) {
    if (!currentClientForFiles) return;

    fetch('/api/files', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ clientId: currentClientForFiles, action: 'get', filePath: filePath })
    });
}

function displayFileContent(fileData) {
    if (!fileData) {
        log('No file data received');
        return;
    }

    if (fileData.error) {
        log('Error loading file: ' + (fileData.error || 'Unknown error'));
        return;
    }

    document.getElementById('noFileSelected').classList.add('hidden');
    document.getElementById('fileContent').classList.remove('hidden');

    document.getElementById('currentFileName').textContent = fileData.name;
    document.getElementById('currentFileSize').textContent = '(' + formatFileSize(fileData.size) + ')';

    const content = atob(fileData.content);

    if (!codeEditor) {
        codeEditor = CodeMirror.fromTextArea(document.getElementById('codeEditor'), {
            lineNumbers: true,
            readOnly: true,
            theme: 'monokai',
            mode: getCodeMirrorMode(fileData.ext)
        });
    } else {
        codeEditor.setOption('mode', getCodeMirrorMode(fileData.ext));
    }

    codeEditor.getDoc().setValue(content);
    codeEditor.refresh();
}

function getCodeMirrorMode(ext) {
    switch (ext) {
        case '.cpp':
        case '.c':
        case '.cc':
        case '.cxx':
        case '.hpp':
        case '.h':
            return 'text/x-c++src';
        case '.py':
            return 'python';
        default:
            return 'text/plain';
    }
}

function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function getFileIcon(ext) {
    switch (ext) {
        case '.cpp':
        case '.c':
        case '.cc':
        case '.cxx':
            return 'cpu';
        case '.py':
            return 'zap';
        case '.h':
        case '.hpp':
            return 'file-text';
        default:
            return 'file';
    }
}

function loadConfig() {
    fetch('/api/config')
        .then(response => response.json())
        .then(cfg => {
            applyConfigUpdate(cfg, true);
            setConfigStatus('Configuration synced with master', 'success');
        })
        .catch(error => {
            console.error('Failed to load configuration', error);
            setConfigStatus('Failed to load configuration', 'error');
        });
}

function renderUrlInputs() {
    const container = document.getElementById('urlList');
    if (!container) return;

    if (!currentConfig.urls || currentConfig.urls.length === 0) {
        currentConfig.urls = [''];
    }

    container.innerHTML = currentConfig.urls.map((url, index) => {
        const value = escapeHtml(url || '');
        return '<div class="flex gap-3 items-center">' +
            '<div class="flex-1">' +
            '<input type="text" data-url-input data-index="' + index + '" value="' + value + '" ' +
            'placeholder="https://example.com" class="w-full border border-gray-300 rounded-md px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400" ' +
            'oninput="updateUrlValue(' + index + ', this.value)">' +
            '</div>' +
            '<button type="button" onclick="removeUrlField(' + index + ')" class="text-gray-500 hover:text-red-600 p-2 rounded-full border border-transparent hover:border-red-200" title="Remove URL">' +
            '<i data-lucide="trash-2" class="w-4 h-4"></i>' +
            '</button>' +
            '</div>';
    }).join('');

    if (window.lucide && typeof window.lucide.createIcons === 'function') {
        window.lucide.createIcons();
    }
}

function addUrlField() {
    if (!currentConfig.urls) {
        currentConfig.urls = [];
    }
    currentConfig.urls.push('');
    renderUrlInputs();
}

function removeUrlField(index) {
    if (!currentConfig.urls) return;
    currentConfig.urls.splice(index, 1);
    if (currentConfig.urls.length === 0) {
        currentConfig.urls.push('');
    }
    renderUrlInputs();
}

function updateUrlValue(index, value) {
    if (!currentConfig.urls) return;
    currentConfig.urls[index] = value;
}

function saveConfig() {
    const inputs = document.querySelectorAll('[data-url-input]');
    const urls = Array.from(inputs)
        .map(input => input.value.trim())
        .filter(url => url.length > 0);

    if (urls.length === 0) {
        setConfigStatus('Please enter at least one URL before saving', 'error');
        return;
    }

    setConfigStatus('Saving configuration...', 'info');

    fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ urls: urls })
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(text => { throw new Error(text || 'Failed to save configuration'); });
        }
        return response.json();
    })
    .then(() => {
        currentConfig.urls = urls.slice();
        setConfigStatus('Configuration saved', 'success');
    })
    .catch(error => {
        console.error('Failed to save configuration', error);
        setConfigStatus(error.message || 'Failed to save configuration', 'error');
    });
}

function applyConfigUpdate(cfg, silent = false) {
    if (!cfg || !Array.isArray(cfg.urls)) {
        return;
    }

    currentConfig = { urls: cfg.urls.slice() };
    renderUrlInputs();

    if (!silent) {
        setConfigStatus('Configuration updated from master', 'info');
    }
}

function setConfigStatus(message, state = 'info') {
    const statusEl = document.getElementById('configStatus');
    if (!statusEl) return;

    if (configStatusTimeout) {
        clearTimeout(configStatusTimeout);
        configStatusTimeout = null;
    }

    let colorClass = 'text-gray-600';
    if (state === 'success') {
        colorClass = 'text-green-600';
    } else if (state === 'error') {
        colorClass = 'text-red-600';
    }

    statusEl.className = 'text-sm mt-3 ' + colorClass;
    statusEl.textContent = message;

    if (state === 'success') {
        configStatusTimeout = setTimeout(() => {
            statusEl.textContent = '';
        }, 3000);
    }
}

function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

function log(message) {
    const logEl = document.getElementById('log');
    const timestamp = new Date().toLocaleTimeString();
    logEl.innerHTML += '<div><span class="text-cyan-400">[' + timestamp + ']</span> ' + message + '</div>';
    logEl.scrollTop = logEl.scrollHeight;
}

renderUrlInputs();
loadConfig();
connect();
setInterval(refreshClients, 5000);
