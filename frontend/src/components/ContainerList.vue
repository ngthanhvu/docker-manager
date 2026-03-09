<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue';
import {
    Play,
    Square,
    Trash2,
    Terminal as TerminalIcon,
    FileText,
    Search,
    RefreshCw
} from 'lucide-vue-next';
import { dockerApi, getWsUrl } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const containers = ref<any[]>([]);
const loading = ref(true);
const searchQuery = ref('');
const activeContainer = ref<any | null>(null);
const currentPage = ref(1);
const pageSize = ref(10);
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);

const showLogsModal = ref(false);
const logsOutput = ref('');
const logsEl = ref<HTMLElement | null>(null);
let logsSocket: WebSocket | null = null;

const showTerminalModal = ref(false);
const terminalEl = ref<HTMLDivElement | null>(null);
let terminalSocket: WebSocket | null = null;
let terminalReconnectTimer: number | null = null;
let terminalReconnectAttempts = 0;
let terminalManualClose = false;
let xterm: XTerm | null = null;
let fitAddon: FitAddon | null = null;
let terminalResizeObserver: ResizeObserver | null = null;
let terminalDataDisposable: { dispose: () => void } | null = null;
let terminalContainerName = '';

const fetchContainers = async () => {
    try {
        const { data } = await dockerApi.getContainers();
        containers.value = data || [];
    } catch (err) {
        console.error('Failed to fetch containers:', err);
    } finally {
        loading.value = false;
    }
};

const filteredContainers = computed(() => {
    const query = searchQuery.value.trim().toLowerCase();
    if (!query) return containers.value;

    return containers.value.filter((container) => {
        const name = container.Names?.[0]?.replace('/', '').toLowerCase() || '';
        const image = (container.Image || '').toLowerCase();
        const id = (container.Id || '').toLowerCase();
        return name.includes(query) || image.includes(query) || id.includes(query);
    });
});

const totalItems = computed(() => filteredContainers.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
const paginatedContainers = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    return filteredContainers.value.slice(start, start + pageSize.value);
});
const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));

const pageContainerIds = computed(() => paginatedContainers.value.map((c) => c.Id));
const selectedCount = computed(() => selectedIds.value.length);
const allPageSelected = computed(() => pageContainerIds.value.length > 0 && pageContainerIds.value.every((id) => selectedIds.value.includes(id)));

const toggleSelect = (id: string) => {
    if (selectedIds.value.includes(id)) {
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
    } else {
        selectedIds.value = [...selectedIds.value, id];
    }
};

const toggleSelectAllPage = () => {
    if (allPageSelected.value) {
        selectedIds.value = selectedIds.value.filter((id) => !pageContainerIds.value.includes(id));
    } else {
        selectedIds.value = Array.from(new Set([...selectedIds.value, ...pageContainerIds.value]));
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    const removeCount = selectedIds.value.length;
    const accepted = await feedback.confirmAction({
        title: 'Delete Containers',
        message: `Remove ${removeCount} selected container(s)? This action cannot be undone.`,
        confirmText: 'Delete',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        for (const id of selectedIds.value) {
            await dockerApi.removeContainer(id);
        }
        selectedIds.value = [];
        await fetchContainers();
        feedback.success(`Deleted ${removeCount} container(s) successfully.`);
    } catch (err) {
        feedback.error(`Bulk delete failed: ${err}`);
    }
};

const scrollToBottom = async () => {
    await nextTick();
    const el = logsEl.value;
    if (el) el.scrollTop = el.scrollHeight;
};

const appendLogs = (text: string) => {
    logsOutput.value += text;
    scrollToBottom();
};

const closeLogs = () => {
    showLogsModal.value = false;
    if (logsSocket) {
        logsSocket.close();
        logsSocket = null;
    }
};

const closeTerminal = () => {
    showTerminalModal.value = false;
    terminalManualClose = true;
    if (terminalReconnectTimer) {
        window.clearTimeout(terminalReconnectTimer);
        terminalReconnectTimer = null;
    }
    if (terminalSocket) {
        terminalSocket.close();
        terminalSocket = null;
    }
    if (terminalResizeObserver && terminalEl.value) {
        terminalResizeObserver.unobserve(terminalEl.value);
    }
    terminalResizeObserver = null;
    if (xterm) {
        if (terminalDataDisposable) {
            terminalDataDisposable.dispose();
            terminalDataDisposable = null;
        }
        xterm.dispose();
        xterm = null;
    }
    fitAddon = null;
};

const openLogs = (container: any) => {
    closeLogs();
    activeContainer.value = container;
    logsOutput.value = '';
    showLogsModal.value = true;

    logsSocket = new WebSocket(getWsUrl(`/logs/${container.Id}`));
    logsSocket.onopen = () => appendLogs(`[connected] Streaming logs for ${container.Names?.[0]?.replace('/', '')}\n`);
    logsSocket.onmessage = (event) => appendLogs(String(event.data));
    logsSocket.onerror = () => appendLogs('\n[error] Failed to read logs stream.\n');
    logsSocket.onclose = () => appendLogs('\n[closed] Log stream closed.\n');
};

const initTerminalUi = async () => {
    await nextTick();
    if (!terminalEl.value) return;
    const css = getComputedStyle(document.documentElement);
    const fg = css.getPropertyValue('--code-text').trim() || '#d1d5db';
    const bg = css.getPropertyValue('--code-bg').trim() || '#0b1220';
    const cursor = css.getPropertyValue('--primary').trim() || '#2496ed';
    xterm = new XTerm({
        cursorBlink: true,
        fontFamily: 'JetBrains Mono, Fira Code, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace',
        fontSize: 13,
        convertEol: true,
        theme: {
            foreground: fg,
            background: bg,
            cursor,
            cursorAccent: bg,
        },
    });
    fitAddon = new FitAddon();
    xterm.loadAddon(fitAddon);
    xterm.open(terminalEl.value);
    fitAddon.fit();
    xterm.focus();
    terminalDataDisposable = xterm.onData((data) => {
        if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) return;
        terminalSocket.send(data);
    });
    terminalResizeObserver = new ResizeObserver(() => fitAddon?.fit());
    terminalResizeObserver.observe(terminalEl.value);
};

const writeTerminal = (text: string) => {
    xterm?.write(text);
};

const openTerminal = async (container: any) => {
    closeTerminal();
    activeContainer.value = container;
    showTerminalModal.value = true;
    terminalManualClose = false;
    terminalReconnectAttempts = 0;
    terminalContainerName = container.Names?.[0]?.replace('/', '') || container.Id.substring(0, 12);
    await initTerminalUi();

    const connectTerminal = (silent = false) => {
        const shell = encodeURIComponent(appSettings.runtime.terminalShell);
        terminalSocket = new WebSocket(getWsUrl(`/terminal/${container.Id}?shell=${shell}`));
        terminalSocket.onopen = () => {
            terminalReconnectAttempts = 0;
            if (!silent) {
                writeTerminal(`\r\n[connected] Terminal attached to ${terminalContainerName}\r\n`);
            }
            xterm?.focus();
        };
        terminalSocket.onmessage = (event) => writeTerminal(String(event.data));
        terminalSocket.onerror = () => writeTerminal('\r\n[error] Terminal connection failed.\r\n');
        terminalSocket.onclose = () => {
            terminalSocket = null;
            if (terminalManualClose || !showTerminalModal.value) return;
            terminalReconnectAttempts += 1;
            if (terminalReconnectAttempts <= 3) {
                writeTerminal(`\r\n[reconnect] Terminal disconnected. Reconnecting (${terminalReconnectAttempts}/3)...\r\n`);
                terminalReconnectTimer = window.setTimeout(() => connectTerminal(true), 900);
                return;
            }
            writeTerminal('\r\n[closed] Terminal disconnected.\r\n');
        };
    };

    connectTerminal();
};

const handleAction = async (action: string, id: string) => {
    try {
        if (action === 'start') await dockerApi.startContainer(id);
        else if (action === 'stop') await dockerApi.stopContainer(id);
        else if (action === 'remove') {
            const accepted = await feedback.confirmAction({
                title: 'Delete Container',
                message: 'Are you sure you want to remove this container?',
                confirmText: 'Delete',
                danger: true,
                requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
            });
            if (!accepted) return;
            await dockerApi.removeContainer(id);
            selectedIds.value = selectedIds.value.filter((x) => x !== id);
        }
        await fetchContainers();
        if (action === 'start') feedback.success('Container started successfully.');
        else if (action === 'stop') feedback.success('Container stopped successfully.');
        else if (action === 'remove') feedback.success('Container removed successfully.');
    } catch (err) {
        feedback.error(`Action failed: ${err}`);
    }
};

const getStatusColor = (status: string) => {
    if (status.includes('Up')) return 'var(--success)';
    if (status.includes('Exited')) return 'var(--danger)';
    return 'var(--warning)';
};

let interval: any;
const setupInterval = () => {
    if (interval) clearInterval(interval);
    const ms = appSettings.general.autoRefreshMs;
    if (ms > 0) {
        interval = setInterval(fetchContainers, ms);
    }
};
onMounted(() => {
    fetchContainers();
    setupInterval();
});

onUnmounted(() => {
    clearInterval(interval);
    closeLogs();
    closeTerminal();
});

watch(searchQuery, () => {
    currentPage.value = 1;
});

watch(pageSize, () => {
    currentPage.value = 1;
});

watch(totalPages, (maxPage) => {
    if (currentPage.value > maxPage) currentPage.value = maxPage;
});

watch(filteredContainers, (list) => {
    const valid = new Set(list.map((c) => c.Id));
    selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
});

watch(() => appSettings.general.autoRefreshMs, () => {
    setupInterval();
});
</script>

<template>
    <div class="container-list-view">
        <div class="toolbar glass-panel">
            <div class="search-box">
                <Search :size="18" />
                <input v-model="searchQuery" type="text" placeholder="Search containers..." />
            </div>
            <div class="toolbar-actions">
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0" @click="bulkDelete">
                    <Trash2 :size="16" />
                    Bulk Delete ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost" @click="fetchContainers">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading }" />
                    Refresh
                </button>
            </div>
        </div>

        <div v-if="selectedCount > 0" class="selection-bar glass-panel">
            {{ selectedCount }} selected
            <button class="btn btn-ghost" @click="selectedIds = []">Clear</button>
        </div>

        <div class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col">
                            <input type="checkbox" :checked="allPageSelected" @change="toggleSelectAllPage" />
                        </th>
                        <th>Name</th>
                        <th>Image</th>
                        <th>Status</th>
                        <th>Ports</th>
                        <th>Created</th>
                        <th class="actions-cell">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="container in paginatedContainers" :key="container.Id">
                        <td class="check-col">
                            <input type="checkbox" :checked="selectedIds.includes(container.Id)" @change="toggleSelect(container.Id)" />
                        </td>
                        <td class="name-cell">
                            <div class="container-name">
                                {{ container.Names[0].replace('/', '') }}
                                <span class="id-short">{{ container.Id.substring(0, 12) }}</span>
                            </div>
                        </td>
                        <td><code class="image-name">{{ container.Image }}</code></td>
                        <td>
                            <div class="status-pill" :style="{ '--color': getStatusColor(container.Status) }">
                                <span class="dot"></span>
                                {{ container.Status }}
                            </div>
                        </td>
                        <td>
                            <div class="ports">
                                <span v-for="port in container.Ports" :key="port.PublicPort" class="port-tag">
                                    {{ port.PublicPort }}:{{ port.PrivatePort }}
                                </span>
                            </div>
                        </td>
                        <td class="time-cell">{{ dayjs.unix(container.Created).fromNow() }}</td>
                        <td class="actions-cell">
                            <div class="action-group">
                                <button
                                    v-if="!container.Status.includes('Up')"
                                    class="action-btn action-start"
                                    title="Start"
                                    @click="handleAction('start', container.Id)">
                                    <Play :size="16" />
                                </button>
                                <button
                                    v-else
                                    class="action-btn action-stop"
                                    title="Stop"
                                    @click="handleAction('stop', container.Id)">
                                    <Square :size="16" />
                                </button>
                                <button class="action-btn action-neutral" title="Logs" @click="openLogs(container)">
                                    <FileText :size="16" />
                                </button>
                                <button class="action-btn action-neutral" title="Terminal" @click="openTerminal(container)">
                                    <TerminalIcon :size="16" />
                                </button>
                                <button
                                    class="action-btn action-danger"
                                    title="Remove"
                                    @click="handleAction('remove', container.Id)">
                                    <Trash2 :size="16" />
                                </button>
                            </div>
                        </td>
                    </tr>
                    <tr v-if="filteredContainers.length === 0 && !loading">
                        <td colspan="7" class="empty-state">No containers found</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-if="filteredContainers.length > 0" class="pagination glass-panel">
            <div class="pager-meta">
                <span>Rows</span>
                <select v-model.number="pageSize">
                    <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select>
                <span>{{ pageStart }}-{{ pageEnd }} / {{ totalItems }}</span>
            </div>
            <div class="pager-actions">
                <button class="btn btn-ghost" :disabled="currentPage === 1" @click="currentPage--">Prev</button>
                <span class="pager-page">Page {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost" :disabled="currentPage >= totalPages" @click="currentPage++">Next</button>
            </div>
        </div>

        <div v-if="showLogsModal" class="modal-backdrop" @click.self="closeLogs">
            <div class="modal-panel glass-panel">
                <div class="modal-header">
                    <h3>Logs: {{ activeContainer?.Names?.[0]?.replace('/', '') }}</h3>
                    <button class="btn btn-ghost" @click="closeLogs">Close</button>
                </div>
                <pre ref="logsEl" class="terminal-output log-output">{{ logsOutput }}</pre>
            </div>
        </div>

        <div v-if="showTerminalModal" class="modal-backdrop" @click.self="closeTerminal">
            <div class="modal-panel glass-panel">
                <div class="modal-header">
                    <h3>Terminal: {{ activeContainer?.Names?.[0]?.replace('/', '') }}</h3>
                    <button class="btn btn-ghost" @click="closeTerminal">Close</button>
                </div>
                <div ref="terminalEl" class="terminal-output terminal-xterm"></div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.container-list-view {
    display: flex;
    flex-direction: column;
    gap: 24px;
}

.toolbar {
    padding: 12px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.toolbar-actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

.selection-bar {
    padding: 8px 14px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    color: var(--text-muted);
}

.search-box {
    display: flex;
    align-items: center;
    gap: 12px;
    background: var(--glass);
    padding: 8px 16px;
    border-radius: 10px;
    border: 1px solid var(--glass-border);
    width: 300px;
}

.search-box input {
    background: transparent;
    border: none;
    color: var(--text-main);
    outline: none;
    font-size: 0.9rem;
    width: 100%;
}

.table-container {
    overflow: hidden;
}

.docker-table {
    width: 100%;
    border-collapse: collapse;
    text-align: left;
}

.docker-table th {
    padding: 16px 24px;
    font-family: 'Outfit', sans-serif;
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-muted);
    border-bottom: 1px solid var(--glass-border);
}

.docker-table td {
    padding: 16px 24px;
    font-size: 0.9rem;
    border-bottom: 1px solid var(--glass-border);
}

.check-col {
    width: 40px;
    text-align: center !important;
    padding: 12px !important;
}

.docker-table tr:last-child td {
    border-bottom: none;
}

.docker-table tr:hover {
    background: var(--glass);
}

.container-name {
    font-weight: 600;
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.id-short {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-weight: 400;
}

.image-name {
    background: var(--glass);
    padding: 4px 8px;
    border-radius: 6px;
    color: var(--primary);
}

.status-pill {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 4px 12px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid var(--glass-border);
    border-radius: 20px;
    font-size: 0.8rem;
    color: var(--color);
}

.dot {
    width: 6px;
    height: 6px;
    background: var(--color);
    border-radius: 50%;
}

.ports {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
}

.port-tag {
    background: var(--glass);
    color: var(--text-muted);
    font-size: 0.75rem;
    padding: 2px 6px;
    border-radius: 4px;
}

.action-group {
    display: flex;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
}

.action-btn {
    width: 34px;
    height: 34px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    border: 1px solid var(--glass-border);
    background: rgba(255, 255, 255, 0.03);
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.18s ease;
}

.action-btn:hover {
    transform: translateY(-1px);
    color: var(--text-main);
}

.action-neutral:hover {
    background: rgba(36, 150, 237, 0.12);
    border-color: rgba(36, 150, 237, 0.45);
}

.action-start {
    color: #6ee7b7;
    border-color: rgba(16, 185, 129, 0.32);
    background: rgba(16, 185, 129, 0.08);
}

.action-start:hover {
    background: rgba(16, 185, 129, 0.16);
    border-color: rgba(16, 185, 129, 0.55);
}

.action-stop {
    color: #fcd34d;
    border-color: rgba(245, 158, 11, 0.32);
    background: rgba(245, 158, 11, 0.08);
}

.action-stop:hover {
    background: rgba(245, 158, 11, 0.16);
    border-color: rgba(245, 158, 11, 0.55);
}

.action-danger {
    color: #fda4af;
    border-color: rgba(239, 68, 68, 0.32);
    background: rgba(239, 68, 68, 0.08);
}

.action-danger:hover {
    background: rgba(239, 68, 68, 0.16);
    border-color: rgba(239, 68, 68, 0.55);
}

.actions-cell {
    text-align: right;
    width: 200px;
}

.animate-spin {
    animation: spin 1s linear infinite;
}

@keyframes spin {
    from {
        transform: rotate(0deg);
    }

    to {
        transform: rotate(360deg);
    }
}

.empty-state {
    padding: 80px 0;
    text-align: center;
    color: var(--text-muted);
}

.pagination {
    padding: 10px 14px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
}

.pager-meta,
.pager-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text-muted);
    font-size: 0.82rem;
}

.pager-meta select {
    background: var(--glass);
    border: 1px solid var(--glass-border);
    color: var(--text-main);
    border-radius: 6px;
    padding: 4px 6px;
}

.pager-page {
    min-width: 92px;
    text-align: center;
}

.modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
}

.modal-panel {
    width: min(980px, 95vw);
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 18px;
}

.modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.modal-header h3 {
    margin: 0;
    font-size: 1rem;
}

.terminal-output {
    height: 60vh;
    margin: 0;
    border-radius: 8px;
    border: 1px solid var(--glass-border);
    background: var(--code-bg);
}

.log-output {
    padding: 12px;
    overflow: auto;
    color: var(--code-text);
    font-family: var(--font-mono);
    font-size: 0.85rem;
    line-height: 1.4;
    white-space: pre-wrap;
    word-break: break-word;
}

.terminal-xterm {
    overflow: hidden;
}

.terminal-xterm :deep(.xterm) {
    height: 100%;
    padding: 10px 12px;
}

.terminal-xterm :deep(.xterm-viewport) {
    overflow-y: auto !important;
    background: transparent !important;
}
</style>
