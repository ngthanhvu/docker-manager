<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue';
import {
    Play,
    Square,
    Trash2,
    Terminal as TerminalIcon,
    FileText,
    Search,
    RefreshCw,
    RotateCw,
    BrushCleaning,
    Copy,
    ClipboardPaste,
    Maximize2,
    Minimize2
} from 'lucide-vue-next';
import { dockerApi, getWsUrl } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, loadStoredString, persistStoredValue } from '../ui/viewState';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import '@xterm/xterm/css/xterm.css';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

const containers = ref<any[]>([]);
const loading = ref(true);
const CONTAINER_SEARCH_KEY = 'dock-manager.containers.search';
const CONTAINER_PAGE_SIZE_KEY = 'dock-manager.containers.page-size';
const searchQuery = ref(loadStoredString(CONTAINER_SEARCH_KEY, ''));
const activeContainer = ref<any | null>(null);
const currentPage = ref(1);
const pageSize = ref(loadStoredNumber(CONTAINER_PAGE_SIZE_KEY, 10, 10, 50));
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);
const pruning = ref(false);
const searchInput = ref<HTMLInputElement | null>(null);

const showLogsModal = ref(false);
const logsOutput = ref('');
const logsEl = ref<HTMLElement | null>(null);
const logsFollow = ref(true);
const logsFontSize = ref(13);
const logsModalExpanded = ref(false);
let logsSocket: WebSocket | null = null;

const showTerminalModal = ref(false);
const terminalEl = ref<HTMLDivElement | null>(null);
const terminalModalExpanded = ref(false);
const terminalModalPanel = ref<HTMLElement | null>(null);
const terminalIsFullscreen = ref(false);
let terminalSocket: WebSocket | null = null;
let terminalReconnectTimer: number | null = null;
let terminalReconnectAttempts = 0;
let terminalManualClose = false;
let xterm: XTerm | null = null;
let fitAddon: FitAddon | null = null;
let terminalResizeObserver: ResizeObserver | null = null;
let terminalDataDisposable: { dispose: () => void } | null = null;
let terminalContainerName = '';

const terminalThemeOptions = [
    { value: 'ocean', label: 'Ocean Blue' },
    { value: 'matrix', label: 'Matrix Green' },
    { value: 'amber', label: 'Amber Gold' },
] as const;

const getTerminalTheme = (themeName: 'ocean' | 'matrix' | 'amber') => {
    if (themeName === 'matrix') {
        return {
            foreground: '#d1fae5',
            background: '#03140c',
            cursor: '#22c55e',
            cursorAccent: '#03140c',
            selectionBackground: 'rgba(34, 197, 94, 0.22)',
            black: '#04130a',
            red: '#f87171',
            green: '#22c55e',
            yellow: '#84cc16',
            blue: '#34d399',
            magenta: '#10b981',
            cyan: '#2dd4bf',
            white: '#d1fae5',
            brightBlack: '#166534',
            brightRed: '#fca5a5',
            brightGreen: '#86efac',
            brightYellow: '#bef264',
            brightBlue: '#6ee7b7',
            brightMagenta: '#34d399',
            brightCyan: '#5eead4',
            brightWhite: '#ecfdf5',
        };
    }

    if (themeName === 'amber') {
        return {
            foreground: '#fef3c7',
            background: '#1a1206',
            cursor: '#f59e0b',
            cursorAccent: '#1a1206',
            selectionBackground: 'rgba(245, 158, 11, 0.24)',
            black: '#120d05',
            red: '#fb7185',
            green: '#fbbf24',
            yellow: '#f59e0b',
            blue: '#fcd34d',
            magenta: '#f97316',
            cyan: '#fdba74',
            white: '#fffbeb',
            brightBlack: '#78350f',
            brightRed: '#fda4af',
            brightGreen: '#fde68a',
            brightYellow: '#fcd34d',
            brightBlue: '#fef08a',
            brightMagenta: '#fdba74',
            brightCyan: '#fed7aa',
            brightWhite: '#fff7ed',
        };
    }

    return {
        foreground: '#dbeafe',
        background: '#081121',
        cursor: '#60a5fa',
        cursorAccent: '#081121',
        selectionBackground: 'rgba(96, 165, 250, 0.24)',
        black: '#0f172a',
        red: '#f87171',
        green: '#34d399',
        yellow: '#fbbf24',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#22d3ee',
        white: '#e2e8f0',
        brightBlack: '#475569',
        brightRed: '#fca5a5',
        brightGreen: '#86efac',
        brightYellow: '#fde68a',
        brightBlue: '#93c5fd',
        brightMagenta: '#d8b4fe',
        brightCyan: '#67e8f9',
        brightWhite: '#f8fafc',
    };
};

const getPortKey = (port: any) => [
    port?.IP || '',
    port?.Type || '',
    port?.PublicPort ?? '',
    port?.PrivatePort ?? '',
].join(':');

const getPortLabel = (port: any) => {
    const privatePort = port?.PrivatePort ?? '?';
    const publicPort = port?.PublicPort;
    return publicPort ? `${publicPort}:${privatePort}` : String(privatePort);
};

const normalizeContainer = (container: any) => {
    const ports = Array.isArray(container?.Ports) ? container.Ports : [];
    const seen = new Set<string>();
    const uniquePorts = ports.filter((port: any) => {
        const key = getPortKey(port);
        if (seen.has(key)) return false;
        seen.add(key);
        return true;
    });

    return {
        ...container,
        Ports: uniquePorts,
    };
};

const fetchContainers = async () => {
    loading.value = true;
    try {
        const { data } = await dockerApi.getContainers();
        containers.value = Array.isArray(data) ? data.map(normalizeContainer) : [];
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

const pruneContainers = async () => {
    if (pruning.value) return;
    const accepted = await feedback.confirmAction({
        title: 'Prune Containers',
        message: 'Remove all stopped containers?',
        confirmText: 'Prune',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;

    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneContainers();
        await fetchContainers();
        const deletedCount = Array.isArray(data?.ContainersDeleted) ? data.ContainersDeleted.length : 0;
        feedback.success(`Pruned ${deletedCount} stopped container(s).`);
    } catch (err) {
        feedback.error(`Container prune failed: ${err}`);
    } finally {
        pruning.value = false;
    }
};

const bulkStart = async () => {
    if (selectedIds.value.length === 0) return;
    const total = selectedIds.value.length;
    let failed = 0;

    for (const id of selectedIds.value) {
        try {
            await dockerApi.startContainer(id);
        } catch (err) {
            console.error(`Bulk start failed for ${id}:`, err);
            failed += 1;
        }
    }

    await fetchContainers();
    if (failed === 0) {
        feedback.success(`Started ${total} container(s) successfully.`);
    } else if (failed === total) {
        feedback.error('Bulk start failed for all selected containers.');
    } else {
        feedback.warning(`Started ${total - failed}/${total} container(s).`);
    }
};

const bulkRestart = async () => {
    if (selectedIds.value.length === 0) return;
    const total = selectedIds.value.length;
    let failed = 0;

    for (const id of selectedIds.value) {
        try {
            await dockerApi.restartContainer(id);
        } catch (err) {
            console.error(`Bulk restart failed for ${id}:`, err);
            failed += 1;
        }
    }

    await fetchContainers();
    if (failed === 0) {
        feedback.success(`Restarted ${total} container(s) successfully.`);
    } else if (failed === total) {
        feedback.error('Bulk restart failed for all selected containers.');
    } else {
        feedback.warning(`Restarted ${total - failed}/${total} container(s).`);
    }
};

const scrollToBottom = async () => {
    await nextTick();
    const el = logsEl.value;
    if (el) el.scrollTop = el.scrollHeight;
};

const stripAnsi = (text: string) => text.replace(/\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])/g, '');

const isNearBottom = () => {
    const el = logsEl.value;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight < 48;
};

const appendLogs = (text: string) => {
    const shouldStickToBottom = logsFollow.value && isNearBottom();
    logsOutput.value += stripAnsi(text);
    if (shouldStickToBottom) {
        scrollToBottom();
    }
};

const closeLogs = () => {
    showLogsModal.value = false;
    logsModalExpanded.value = false;
    if (logsSocket) {
        logsSocket.close();
        logsSocket = null;
    }
};

const closeTerminal = () => {
    showTerminalModal.value = false;
    terminalModalExpanded.value = false;
    if (document.fullscreenElement === terminalModalPanel.value) {
        document.exitFullscreen().catch(() => {});
    }
    terminalIsFullscreen.value = false;
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
    logsFollow.value = true;
    showLogsModal.value = true;

    const tail = Math.max(50, Number(appSettings.runtime.defaultLogTail) || 300);
    logsSocket = new WebSocket(getWsUrl(`/logs/${container.Id}?tail=${tail}`));
    logsSocket.onopen = () => appendLogs(`[connected] Streaming logs for ${container.Names?.[0]?.replace('/', '')}\n`);
    logsSocket.onmessage = (event) => appendLogs(String(event.data));
    logsSocket.onerror = () => appendLogs('\n[error] Failed to read logs stream.\n');
    logsSocket.onclose = () => appendLogs('\n[closed] Log stream closed.\n');
};

const handleLogsScroll = () => {
    logsFollow.value = isNearBottom();
};

const jumpToLatestLogs = () => {
    logsFollow.value = true;
    scrollToBottom();
};

const adjustLogsFontSize = (delta: number) => {
    logsFontSize.value = Math.min(20, Math.max(11, logsFontSize.value + delta));
};

const toggleLogsSize = () => {
    logsModalExpanded.value = !logsModalExpanded.value;
};

const initTerminalUi = async () => {
    await nextTick();
    if (!terminalEl.value) return;
    const terminalTheme = getTerminalTheme(appSettings.runtime.terminalTheme);
    xterm = new XTerm({
        cursorBlink: true,
        fontFamily: 'JetBrains Mono, Fira Code, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, monospace',
        fontSize: appSettings.runtime.terminalFontSize,
        convertEol: true,
        theme: terminalTheme,
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

const adjustTerminalFontSize = (delta: number) => {
    appSettings.runtime.terminalFontSize = Math.min(20, Math.max(11, appSettings.runtime.terminalFontSize + delta));
    if (xterm) {
        xterm.options.fontSize = appSettings.runtime.terminalFontSize;
        fitAddon?.fit();
        xterm.focus();
    }
};

const toggleTerminalSize = async () => {
    terminalModalExpanded.value = !terminalModalExpanded.value;
    await nextTick();
    fitAddon?.fit();
    xterm?.focus();
};

const copyTerminalSelection = async () => {
    const selectedText = xterm?.getSelection()?.trim() || '';
    if (!selectedText) {
        feedback.warning('Select terminal text first.');
        return;
    }
    try {
        await navigator.clipboard.writeText(selectedText);
        feedback.success('Terminal selection copied.');
    } catch (err) {
        feedback.error(`Failed to copy terminal text: ${err}`);
    }
};

const pasteIntoTerminal = async () => {
    if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) {
        feedback.warning('Terminal is not connected.');
        return;
    }
    try {
        const text = await navigator.clipboard.readText();
        if (!text) {
            feedback.warning('Clipboard is empty.');
            return;
        }
        terminalSocket.send(text);
        xterm?.focus();
    } catch (err) {
        feedback.error(`Failed to paste into terminal: ${err}`);
    }
};

const toggleTerminalFullscreen = async () => {
    const panel = terminalModalPanel.value;
    if (!panel) return;
    try {
        if (document.fullscreenElement === panel) {
            await document.exitFullscreen();
            terminalIsFullscreen.value = false;
        } else {
            await panel.requestFullscreen();
            terminalIsFullscreen.value = true;
        }
        await nextTick();
        fitAddon?.fit();
        xterm?.focus();
    } catch (err) {
        feedback.error(`Failed to toggle terminal fullscreen: ${err}`);
    }
};

const handleFullscreenChange = async () => {
    terminalIsFullscreen.value = document.fullscreenElement === terminalModalPanel.value;
    await nextTick();
    fitAddon?.fit();
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
        else if (action === 'restart') await dockerApi.restartContainer(id);
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
        else if (action === 'restart') feedback.success('Container restarted successfully.');
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
const handleVisibilityChange = () => {
    if (!document.hidden) {
        fetchContainers();
    }
};

const handleListShortcut = (event: KeyboardEvent) => {
    if (event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey) return;
    if (event.key !== '/') return;
    const target = event.target as HTMLElement | null;
    const tag = target?.tagName?.toLowerCase();
    if (tag === 'input' || tag === 'textarea' || target?.isContentEditable) return;
    event.preventDefault();
    searchInput.value?.focus();
    searchInput.value?.select();
};

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
    document.addEventListener('visibilitychange', handleVisibilityChange);
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    window.addEventListener('keydown', handleListShortcut);
});

onUnmounted(() => {
    clearInterval(interval);
    document.removeEventListener('visibilitychange', handleVisibilityChange);
    document.removeEventListener('fullscreenchange', handleFullscreenChange);
    window.removeEventListener('keydown', handleListShortcut);
    closeLogs();
    closeTerminal();
});

watch(searchQuery, () => {
    currentPage.value = 1;
    persistStoredValue(CONTAINER_SEARCH_KEY, searchQuery.value);
});

watch(pageSize, () => {
    currentPage.value = 1;
    persistStoredValue(CONTAINER_PAGE_SIZE_KEY, pageSize.value);
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

watch(() => appSettings.runtime.terminalTheme, (themeName) => {
    if (!xterm) return;
    xterm.options.theme = getTerminalTheme(themeName);
    fitAddon?.fit();
});

watch(() => appSettings.runtime.terminalFontSize, (fontSize) => {
    if (!xterm) return;
    xterm.options.fontSize = fontSize;
    fitAddon?.fit();
});
</script>

<template>
    <div class="container-list-view">
        <div class="toolbar glass-panel">
            <div class="search-box">
                <Search :size="18" />
                <input ref="searchInput" v-model="searchQuery" type="text" placeholder="Search containers..." />
            </div>
            <div class="toolbar-actions">
                <button class="btn btn-ghost" :disabled="selectedCount === 0 || pruning" @click="bulkStart">
                    <Play :size="16" />
                    Start
                </button>
                <button class="btn btn-ghost" :disabled="selectedCount === 0 || pruning" @click="bulkRestart">
                    <RefreshCw :size="16" />
                    Restart
                </button>
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0 || pruning"
                    @click="bulkDelete">
                    <Trash2 :size="16" />
                    Delete
                </button>
                <button class="btn btn-ghost text-warning" :disabled="pruning" @click="pruneContainers">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    Prune
                </button>
                <button class="btn btn-ghost" :disabled="pruning" @click="fetchContainers">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                    Refresh
                </button>
            </div>
        </div>

        <div class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col">
                            <input class="bulk-checkbox" type="checkbox" :checked="allPageSelected"
                                @change="toggleSelectAllPage" />
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
                            <input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(container.Id)"
                                @change="toggleSelect(container.Id)" />
                        </td>
                        <td class="name-cell">
                            <div class="container-name">
                                {{ container.Names[0].replace('/', '') }}
                                <span class="id-short">{{ container.Id.substring(0, 12) }}</span>
                            </div>
                        </td>
                        <td class="image-cell"><code class="image-name">{{ container.Image }}</code></td>
                        <td class="status-cell">
                            <div class="status-pill" :style="{ '--color': getStatusColor(container.Status) }">
                                <span class="dot"></span>
                                {{ container.Status }}
                            </div>
                        </td>
                        <td class="ports-cell">
                            <div class="ports">
                                <span v-for="port in container.Ports" :key="`${container.Id}-${getPortKey(port)}`"
                                    class="port-tag">
                                    {{ getPortLabel(port) }}
                                </span>
                            </div>
                        </td>
                        <td class="time-cell">{{ dayjs.unix(container.Created).fromNow() }}</td>
                        <td class="actions-cell">
                            <div class="action-group">
                                <button v-if="!container.Status.includes('Up')" class="action-btn action-start"
                                    title="Start" @click="handleAction('start', container.Id)">
                                    <Play :size="16" />
                                </button>
                                <button v-else class="action-btn action-stop" title="Stop"
                                    @click="handleAction('stop', container.Id)">
                                    <Square :size="16" />
                                </button>
                                <button class="action-btn action-neutral" :disabled="!container.Status.includes('Up')"
                                    title="Restart" @click="handleAction('restart', container.Id)">
                                    <RotateCw :size="16" />
                                </button>
                                <button class="action-btn action-neutral" title="Logs" @click="openLogs(container)">
                                    <FileText :size="16" />
                                </button>
                                <button class="action-btn action-neutral" title="Terminal"
                                    @click="openTerminal(container)">
                                    <TerminalIcon :size="16" />
                                </button>
                                <button class="action-btn action-danger" title="Remove"
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
            <div class="modal-panel glass-panel logs-modal-panel" :class="{ 'is-expanded': logsModalExpanded }">
                <div class="modal-header">
                    <h3>Logs: {{ activeContainer?.Names?.[0]?.replace('/', '') }}</h3>
                    <div class="modal-actions">
                        <button class="btn btn-ghost" @click="adjustLogsFontSize(-1)">A-</button>
                        <button class="btn btn-ghost" @click="adjustLogsFontSize(1)">A+</button>
                        <button class="btn btn-ghost" @click="toggleLogsSize">
                            {{ logsModalExpanded ? 'Normal Size' : 'Expand' }}
                        </button>
                        <button class="btn btn-ghost" :class="{ 'is-active': logsFollow }" @click="jumpToLatestLogs">
                            {{ logsFollow ? 'Following' : 'Jump To Latest' }}
                        </button>
                        <button class="btn btn-ghost" @click="closeLogs">Close</button>
                    </div>
                </div>
                <pre ref="logsEl" class="terminal-output log-output" :style="{ fontSize: `${logsFontSize}px` }"
                    @scroll="handleLogsScroll">{{ logsOutput }}</pre>
            </div>
        </div>

        <div v-if="showTerminalModal" class="modal-backdrop" @click.self="closeTerminal">
            <div ref="terminalModalPanel" class="modal-panel glass-panel terminal-modal-panel"
                :class="{ 'is-expanded': terminalModalExpanded, 'is-fullscreen': terminalIsFullscreen }">
                <div class="modal-header">
                    <div class="terminal-title-wrap">
                        <h3>Terminal: {{ activeContainer?.Names?.[0]?.replace('/', '') }}</h3>
                        <span class="terminal-shell-pill">{{ appSettings.runtime.terminalShell }}</span>
                    </div>
                    <div class="modal-actions">
                        <select v-model="appSettings.runtime.terminalTheme" class="terminal-theme-select">
                            <option v-for="theme in terminalThemeOptions" :key="theme.value" :value="theme.value">
                                {{ theme.label }}
                            </option>
                        </select>
                        <button class="btn btn-ghost" @click="adjustTerminalFontSize(-1)">A-</button>
                        <button class="btn btn-ghost" @click="adjustTerminalFontSize(1)">A+</button>
                        <button class="btn btn-ghost" @click="copyTerminalSelection">
                            <Copy :size="14" />
                            Copy
                        </button>
                        <button class="btn btn-ghost" @click="pasteIntoTerminal">
                            <ClipboardPaste :size="14" />
                            Paste
                        </button>
                        <button class="btn btn-ghost" @click="toggleTerminalSize">
                            {{ terminalModalExpanded ? 'Normal Size' : 'Expand' }}
                        </button>
                        <button class="btn btn-ghost" @click="toggleTerminalFullscreen">
                            <Minimize2 v-if="terminalIsFullscreen" :size="14" />
                            <Maximize2 v-else :size="14" />
                            {{ terminalIsFullscreen ? 'Exit Fullscreen' : 'Fullscreen' }}
                        </button>
                        <button class="btn btn-ghost" @click="closeTerminal">Close</button>
                    </div>
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
    min-width: 0;
}

.toolbar {
    padding: 12px 24px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    min-width: 0;
}

.toolbar-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
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
    max-width: 100%;
    min-width: 0;
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
    overflow: auto;
    width: 100%;
    min-width: 0;
}

.docker-table {
    width: 100%;
    min-width: 1320px;
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
    width: 56px;
    text-align: center !important;
    padding: 10px !important;
}

.name-cell {
    min-width: 240px;
}

.image-cell {
    min-width: 220px;
}

.status-cell {
    min-width: 190px;
}

.ports-cell {
    min-width: 240px;
}

.bulk-checkbox {
    width: 22px;
    height: 22px;
    cursor: pointer;
    accent-color: var(--primary);
    border-radius: 6px;
}

.bulk-checkbox:hover {
    filter: brightness(1.08);
}

.bulk-checkbox:focus-visible {
    outline: 2px solid rgba(36, 150, 237, 0.55);
    outline-offset: 2px;
}

.docker-table tr:last-child td {
    border-bottom: none;
}

.docker-table tr:hover {
    background: var(--glass);
}

.container-name {
    font-weight: 600;
    font-size: 1rem;
    line-height: 1.25;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.id-short {
    font-size: 0.82rem;
    color: var(--text-muted);
    font-weight: 400;
}

.image-name {
    background: var(--glass);
    padding: 4px 8px;
    border-radius: 6px;
    color: var(--primary);
    white-space: nowrap;
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
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
    white-space: nowrap;
}

.dot {
    width: 6px;
    height: 6px;
    background: var(--color);
    border-radius: 50%;
}

.ports {
    display: flex;
    flex-wrap: nowrap;
    gap: 4px;
    overflow-x: auto;
    scrollbar-width: thin;
}

.port-tag {
    background: var(--glass);
    color: var(--text-muted);
    font-size: 0.75rem;
    padding: 2px 6px;
    border-radius: 4px;
    white-space: nowrap;
    flex: 0 0 auto;
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

.action-btn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
    transform: none;
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
    width: 240px;
    min-width: 240px;
    white-space: nowrap;
}

.time-cell {
    min-width: 140px;
    white-space: nowrap;
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

.logs-modal-panel {
    width: min(1240px, 96vw);
    min-width: 720px;
    min-height: 420px;
    resize: both;
    overflow: hidden;
}

.logs-modal-panel.is-expanded {
    width: min(1520px, 98vw);
    max-height: 94vh;
}

.terminal-modal-panel {
    width: min(1240px, 96vw);
    min-width: 760px;
    min-height: 420px;
    resize: both;
    overflow: hidden;
}

.terminal-modal-panel.is-expanded {
    width: min(1520px, 98vw);
    max-height: 94vh;
}

.terminal-modal-panel.is-fullscreen {
    width: 100%;
    height: 100%;
    max-height: none;
    border-radius: 0;
    padding: 20px;
    resize: none;
}

.modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.modal-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: flex-end;
}

.modal-header h3 {
    margin: 0;
    font-size: 1rem;
}

.terminal-title-wrap {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
}

.terminal-shell-pill {
    display: inline-flex;
    align-items: center;
    padding: 3px 8px;
    border-radius: 999px;
    border: 1px solid rgba(96, 165, 250, 0.28);
    background: rgba(59, 130, 246, 0.14);
    color: #93c5fd;
    font-size: 0.74rem;
    font-family: var(--font-mono);
}

.terminal-theme-select {
    border: 1px solid var(--glass-border);
    border-radius: 10px;
    padding: 8px 10px;
    background: var(--glass);
    color: var(--text-main);
}

.modal-actions .is-active {
    border-color: rgba(36, 150, 237, 0.45);
    color: var(--primary);
}

.terminal-output {
    height: 60vh;
    min-height: 360px;
    margin: 0;
    border-radius: 8px;
    border: 1px solid var(--glass-border);
    background:
        radial-gradient(circle at top right, rgba(37, 99, 235, 0.08), transparent 22%),
        linear-gradient(180deg, rgba(15, 23, 42, 0.96), rgba(2, 6, 23, 0.98));
}

.log-output {
    padding: 12px;
    overflow: auto;
    color: var(--code-text);
    font-family: var(--font-mono);
    font-size: 13px;
    line-height: 1.4;
    white-space: pre-wrap;
    word-break: break-word;
}

.terminal-xterm {
    overflow: hidden;
}

.terminal-modal-panel.is-fullscreen .terminal-output {
    height: calc(100vh - 120px);
}

.terminal-xterm :deep(.xterm) {
    height: 100%;
    padding: 10px 12px;
}

.terminal-xterm :deep(.xterm-screen) {
    background: transparent;
}

.terminal-xterm :deep(.xterm-viewport) {
    overflow-y: auto !important;
    background: transparent !important;
}

.terminal-xterm :deep(.xterm-selection div) {
    background: rgba(96, 165, 250, 0.2) !important;
}

@media (max-width: 900px) {
    .terminal-modal-panel,
    .logs-modal-panel {
        min-width: 0;
        width: min(100%, 96vw);
    }

    .modal-header {
        align-items: flex-start;
        flex-direction: column;
    }

    .terminal-title-wrap {
        flex-wrap: wrap;
    }
}
</style>
