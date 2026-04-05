<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
    Play,
    Square,
    Trash2,
    X,
    Minus,
    Plus,
    ArrowDownToLine,
    Terminal as TerminalIcon,
    FileText,
    Search,
    RefreshCw,
    RotateCw,
    BrushCleaning,
    Copy,
    ClipboardPaste,
    Maximize2,
    Minimize2,
    List,
    LayoutGrid,
    Ellipsis
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
const { t } = useI18n();

const containers = ref<any[]>([]);
const loading = ref(true);
const CONTAINER_SEARCH_KEY = 'dock-manager.containers.search';
const CONTAINER_PAGE_SIZE_KEY = 'dock-manager.containers.page-size';
const CONTAINER_VIEW_MODE_KEY = 'dock-manager.containers.view-mode';
const searchQuery = ref(loadStoredString(CONTAINER_SEARCH_KEY, ''));
const viewMode = ref<'list' | 'card'>(loadStoredString(CONTAINER_VIEW_MODE_KEY, 'list') === 'card' ? 'card' : 'list');
const activeContainer = ref<any | null>(null);
const currentPage = ref(1);
const pageSize = ref(loadStoredNumber(CONTAINER_PAGE_SIZE_KEY, 10, 10, 50));
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);
const pruning = ref(false);
const searchInput = ref<HTMLInputElement | null>(null);
const containerStats = ref<Record<string, any>>({});
const activeCardMenuId = ref<string | null>(null);
const sortKey = ref<'name' | 'image' | 'status' | 'ports' | 'created'>('created');
const sortDirection = ref<'asc' | 'desc'>('desc');

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
    { value: 'ocean', label: t('settings.themeOcean') },
    { value: 'matrix', label: t('settings.themeMatrix') },
    { value: 'amber', label: t('settings.themeAmber') },
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

const compareValues = (left: string | number, right: string | number) => {
    if (typeof left === 'number' && typeof right === 'number') return left - right;
    return String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: 'base' });
};

const getContainerSortValue = (container: any) => {
    if (sortKey.value === 'name') return getContainerName(container);
    if (sortKey.value === 'image') return container.Image || '';
    if (sortKey.value === 'status') return getContainerStateLabel(container);
    if (sortKey.value === 'ports') return Array.isArray(container.Ports) ? container.Ports.length : 0;
    return Number(container.Created || 0);
};

const sortedFilteredContainers = computed(() => {
    const list = [...filteredContainers.value];
    list.sort((a, b) => {
        const result = compareValues(getContainerSortValue(a), getContainerSortValue(b));
        return sortDirection.value === 'asc' ? result : -result;
    });
    return list;
});

const totalItems = computed(() => sortedFilteredContainers.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
const paginatedContainers = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    return sortedFilteredContainers.value.slice(start, start + pageSize.value);
});
const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));

const pageContainerIds = computed(() => paginatedContainers.value.map((c) => c.Id));
const selectedCount = computed(() => selectedIds.value.length);
const allPageSelected = computed(() => pageContainerIds.value.length > 0 && pageContainerIds.value.every((id) => selectedIds.value.includes(id)));
const getContainerName = (container: any) => container?.Names?.[0]?.replace('/', '') || container?.Id?.substring(0, 12) || '';
const getContainerStateLabel = (container: any) => {
    const rawState = String(container?.State || '').trim();
    if (rawState) return rawState.charAt(0).toUpperCase() + rawState.slice(1);
    if (String(container?.Status || '').includes('Up')) return 'Running';
    if (String(container?.Status || '').includes('Exited')) return 'Exited';
    return String(container?.Status || 'Unknown');
};

const formatBytes = (bytes?: number) => {
    if (!bytes || bytes <= 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let value = bytes;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex += 1;
    }
    return `${value.toFixed(value >= 10 || unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
};

const fetchContainerStats = async () => {
    const runningIds = paginatedContainers.value
        .filter((container) => container.Status?.includes('Up'))
        .map((container) => container.Id);

    if (runningIds.length === 0) {
        containerStats.value = {};
        return;
    }

    try {
        const { data } = await dockerApi.getContainerStats(runningIds);
        containerStats.value = data && typeof data === 'object' ? data : {};
    } catch (err) {
        console.error('Failed to fetch container stats:', err);
        containerStats.value = {};
    }
};

const getContainerStats = (id: string) => containerStats.value[id];
const getContainerMemoryLabel = (id: string) => {
    const stats = getContainerStats(id);
    if (!stats) return '--';
    if (stats.memoryLimitBytes > 0) {
        return `${formatBytes(stats.memoryUsedBytes)} / ${formatBytes(stats.memoryLimitBytes)}`;
    }
    return formatBytes(stats.memoryUsedBytes);
};

const getContainerNetworkLabel = (id: string) => {
    const stats = getContainerStats(id);
    if (!stats) return '--';
    return `${t('containersView.rx')}: ${formatBytes(stats.networkRxBytes)} · ${t('containersView.tx')}: ${formatBytes(stats.networkTxBytes)}`;
};

const getContainerCpuPercent = (id: string) => {
    const stats = getContainerStats(id);
    if (!stats) return 0;
    return Math.max(0, Math.min(100, Number(stats.cpuPercent || 0)));
};

const getContainerMemoryPercent = (id: string) => {
    const stats = getContainerStats(id);
    if (!stats) return 0;
    return Math.max(0, Math.min(100, Number(stats.memoryPercent || 0)));
};

const toggleCardMenu = (id: string) => {
    activeCardMenuId.value = activeCardMenuId.value === id ? null : id;
};

const closeCardMenu = () => {
    activeCardMenuId.value = null;
};

const handleCardAction = async (action: string, container: any) => {
    closeCardMenu();
    if (action === 'logs') {
        openLogs(container);
        return;
    }
    if (action === 'terminal') {
        openTerminal(container);
        return;
    }
    await handleAction(action, container.Id);
};

const toggleSort = (key: 'name' | 'image' | 'status' | 'ports' | 'created') => {
    if (sortKey.value === key) {
        sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
        return;
    }
    sortKey.value = key;
    sortDirection.value = key === 'created' ? 'desc' : 'asc';
};

const getSortIndicator = (key: 'name' | 'image' | 'status' | 'ports' | 'created') => {
    if (sortKey.value !== key) return '↕';
    return sortDirection.value === 'asc' ? '↑' : '↓';
};

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
        title: t('containersView.deleteManyTitle'),
        message: t('containersView.deleteManyMessage', { count: removeCount }),
        confirmText: t('common.delete'),
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
        feedback.success(t('containersView.deletedManySuccess', { count: removeCount }));
    } catch (err) {
        feedback.error(`Bulk delete failed: ${err}`);
    }
};

const pruneContainers = async () => {
    if (pruning.value) return;
    const accepted = await feedback.confirmAction({
        title: t('containersView.pruneTitle'),
        message: t('containersView.pruneMessage'),
        confirmText: t('common.prune'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;

    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneContainers();
        await fetchContainers();
        const deletedCount = Array.isArray(data?.ContainersDeleted) ? data.ContainersDeleted.length : 0;
        feedback.success(t('containersView.prunedSuccess', { count: deletedCount }));
    } catch (err) {
        feedback.error(t('containersView.pruneFailed', { error: String(err) }));
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
        feedback.success(t('containersView.startedManySuccess', { count: total }));
    } else if (failed === total) {
        feedback.error(t('containersView.startedAllFailed'));
    } else {
        feedback.warning(t('containersView.startedPartial', { success: total - failed, total }));
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
        feedback.success(t('containersView.restartedManySuccess', { count: total }));
    } else if (failed === total) {
        feedback.error(t('containersView.restartedAllFailed'));
    } else {
        feedback.warning(t('containersView.restartedPartial', { success: total - failed, total }));
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
        document.exitFullscreen().catch(() => { });
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
    logsSocket.onopen = () => appendLogs(`${t('containersView.logsConnected', { name: getContainerName(container) })}\n`);
    logsSocket.onmessage = (event) => appendLogs(String(event.data));
    logsSocket.onerror = () => appendLogs(`\n${t('containersView.logsError')}\n`);
    logsSocket.onclose = () => appendLogs(`\n${t('containersView.logsClosed')}\n`);
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
        feedback.warning(t('containersView.selectTerminalText'));
        return;
    }
    try {
        await navigator.clipboard.writeText(selectedText);
        feedback.success(t('containersView.selectionCopied'));
    } catch (err) {
        feedback.error(t('containersView.copyFailed', { error: String(err) }));
    }
};

const pasteIntoTerminal = async () => {
    if (!terminalSocket || terminalSocket.readyState !== WebSocket.OPEN) {
        feedback.warning(t('containersView.terminalNotConnected'));
        return;
    }
    try {
        const text = await navigator.clipboard.readText();
        if (!text) {
            feedback.warning(t('containersView.clipboardEmpty'));
            return;
        }
        terminalSocket.send(text);
        xterm?.focus();
    } catch (err) {
        feedback.error(t('containersView.pasteFailed', { error: String(err) }));
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
        feedback.error(t('containersView.fullscreenFailed', { error: String(err) }));
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
                writeTerminal(`\r\n${t('containersView.terminalConnected', { name: terminalContainerName })}\r\n`);
            }
            xterm?.focus();
        };
        terminalSocket.onmessage = (event) => writeTerminal(String(event.data));
        terminalSocket.onerror = () => writeTerminal(`\r\n${t('containersView.terminalError')}\r\n`);
        terminalSocket.onclose = () => {
            terminalSocket = null;
            if (terminalManualClose || !showTerminalModal.value) return;
            terminalReconnectAttempts += 1;
            if (terminalReconnectAttempts <= 3) {
                writeTerminal(`\r\n${t('containersView.terminalReconnect', { attempt: terminalReconnectAttempts })}\r\n`);
                terminalReconnectTimer = window.setTimeout(() => connectTerminal(true), 900);
                return;
            }
            writeTerminal(`\r\n${t('containersView.terminalClosed')}\r\n`);
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
                title: t('containersView.deleteTitle'),
                message: t('containersView.deleteMessage'),
                confirmText: t('common.delete'),
                danger: true,
                requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
            });
            if (!accepted) return;
            await dockerApi.removeContainer(id);
            selectedIds.value = selectedIds.value.filter((x) => x !== id);
        }
        await fetchContainers();
        if (action === 'start') feedback.success(t('containersView.startSuccess'));
        else if (action === 'stop') feedback.success(t('containersView.stopSuccess'));
        else if (action === 'restart') feedback.success(t('containersView.restartSuccess'));
        else if (action === 'remove') feedback.success(t('containersView.removeSuccess'));
    } catch (err) {
        feedback.error(t('containersView.actionFailed', { error: String(err) }));
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

const handleDocumentClick = (event: MouseEvent) => {
    const target = event.target as HTMLElement | null;
    if (target?.closest('.card-actions-menu')) return;
    closeCardMenu();
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
    document.addEventListener('click', handleDocumentClick);
});

onUnmounted(() => {
    clearInterval(interval);
    document.removeEventListener('visibilitychange', handleVisibilityChange);
    document.removeEventListener('fullscreenchange', handleFullscreenChange);
    window.removeEventListener('keydown', handleListShortcut);
    document.removeEventListener('click', handleDocumentClick);
    closeLogs();
    closeTerminal();
    document.body.style.overflow = '';
});

watch(searchQuery, () => {
    currentPage.value = 1;
    persistStoredValue(CONTAINER_SEARCH_KEY, searchQuery.value);
});

watch(pageSize, () => {
    currentPage.value = 1;
    persistStoredValue(CONTAINER_PAGE_SIZE_KEY, pageSize.value);
});

watch(viewMode, () => {
    persistStoredValue(CONTAINER_VIEW_MODE_KEY, viewMode.value);
});

watch(totalPages, (maxPage) => {
    if (currentPage.value > maxPage) currentPage.value = maxPage;
});

watch(filteredContainers, (list) => {
    const valid = new Set(list.map((c) => c.Id));
    selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
});

watch(pageContainerIds, () => {
    fetchContainerStats();
}, { immediate: true });

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

watch(
    [showLogsModal, showTerminalModal],
    ([logsOpen, terminalOpen]) => {
        document.body.style.overflow = logsOpen || terminalOpen ? 'hidden' : '';
    }
);
</script>

<template>
    <div class="container-list-view">
        <div class="toolbar glass-panel">
            <div class="search-box">
                <Search :size="18" />
                <input ref="searchInput" v-model="searchQuery" type="text"
                    :placeholder="t('containersView.searchPlaceholder')" />
            </div>
            <div class="toolbar-actions">
                <div class="view-toggle" role="group" :aria-label="t('containersView.viewMode')">
                    <button class="view-toggle-btn" :class="{ 'is-active': viewMode === 'list' }" type="button"
                        :title="t('containersView.listView')" @click="viewMode = 'list'">
                        <List :size="16" />
                        {{ t('containersView.listView') }}
                    </button>
                    <button class="view-toggle-btn" :class="{ 'is-active': viewMode === 'card' }" type="button"
                        :title="t('containersView.cardView')" @click="viewMode = 'card'">
                        <LayoutGrid :size="16" />
                        {{ t('containersView.cardView') }}
                    </button>
                </div>
                <button class="btn btn-ghost" :disabled="selectedCount === 0 || pruning" @click="bulkStart">
                    <Play :size="16" />
                    {{ t('compose.start') }}
                </button>
                <button class="btn btn-ghost" :disabled="selectedCount === 0 || pruning" @click="bulkRestart">
                    <RefreshCw :size="16" />
                    {{ t('compose.restart') }}
                </button>
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0 || pruning"
                    @click="bulkDelete">
                    <Trash2 :size="16" />
                    {{ t('common.delete') }}
                </button>
                <button class="btn btn-ghost text-warning" :disabled="pruning" @click="pruneContainers">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    {{ t('common.prune') }}
                </button>
                <button class="btn btn-ghost" :disabled="pruning" @click="fetchContainers">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                    {{ t('common.refresh') }}
                </button>
            </div>
        </div>

        <div v-if="viewMode === 'list'" class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col">
                            <input class="bulk-checkbox" type="checkbox" :checked="allPageSelected"
                                @change="toggleSelectAllPage" />
                        </th>
                        <th><button class="sort-header" type="button" @click="toggleSort('name')">{{ t('containersView.name') }}<span class="sort-indicator">{{ getSortIndicator('name') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('image')">{{ t('containersView.image') }}<span class="sort-indicator">{{ getSortIndicator('image') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('status')">{{ t('containersView.status') }}<span class="sort-indicator">{{ getSortIndicator('status') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('ports')">{{ t('containersView.ports') }}<span class="sort-indicator">{{ getSortIndicator('ports') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('created')">{{ t('containersView.created') }}<span class="sort-indicator">{{ getSortIndicator('created') }}</span></button></th>
                        <th class="actions-cell">{{ t('common.actions') }}</th>
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
                        <td class="image-cell">
                            <code class="image-name" :title="container.Image">{{ container.Image }}</code>
                        </td>
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
                                    :title="t('compose.start')" @click="handleAction('start', container.Id)">
                                    <Play :size="16" />
                                </button>
                                <button v-else class="action-btn action-stop" :title="t('compose.stop')"
                                    @click="handleAction('stop', container.Id)">
                                    <Square :size="16" />
                                </button>
                                <button class="action-btn action-neutral" :disabled="!container.Status.includes('Up')"
                                    :title="t('compose.restart')" @click="handleAction('restart', container.Id)">
                                    <RotateCw :size="16" />
                                </button>
                                <button class="action-btn action-neutral" :title="t('compose.logs')"
                                    @click="openLogs(container)">
                                    <FileText :size="16" />
                                </button>
                                <button class="action-btn action-neutral"
                                    :title="t('containersView.terminalTitle', { name: getContainerName(container) })"
                                    @click="openTerminal(container)">
                                    <TerminalIcon :size="16" />
                                </button>
                                <button class="action-btn action-danger" :title="t('common.remove')"
                                    @click="handleAction('remove', container.Id)">
                                    <Trash2 :size="16" />
                                </button>
                            </div>
                        </td>
                    </tr>
                    <tr v-if="filteredContainers.length === 0 && !loading">
                        <td colspan="7" class="empty-state">{{ t('containersView.noItems') }}</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-else class="card-container">
            <div v-if="filteredContainers.length === 0 && !loading" class="glass-panel card-empty-state">
                {{ t('containersView.noItems') }}
            </div>

            <div v-else class="container-card-grid">
                <article v-for="container in paginatedContainers" :key="container.Id" class="glass-panel container-card">
                    <div class="container-card-header">
                        <label class="card-check">
                            <input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(container.Id)"
                                @change="toggleSelect(container.Id)" />
                        </label>
                        <div class="card-title-area">
                            <div class="card-title-row">
                                <div class="container-name">
                                    {{ container.Names[0].replace('/', '') }}
                                    <span class="id-short">{{ container.Id.substring(0, 12) }}</span>
                                </div>
                                <div class="status-pill" :style="{ '--color': getStatusColor(container.Status) }">
                                    <span class="dot"></span>
                                    {{ getContainerStateLabel(container) }}
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="card-meta-list">
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('containersView.resources') }}</span>
                            <div v-if="container.Status.includes('Up') && getContainerStats(container.Id)" class="resource-meters">
                                <div class="resource-meter">
                                    <div class="resource-meter-head">
                                        <span>{{ t('nav.cpu') }}</span>
                                        <strong>{{ getContainerCpuPercent(container.Id).toFixed(1) }}%</strong>
                                    </div>
                                    <div class="resource-progress">
                                        <div class="resource-progress-fill cpu-fill"
                                            :style="{ width: `${getContainerCpuPercent(container.Id)}%` }"></div>
                                    </div>
                                </div>
                                <div class="resource-meter">
                                    <div class="resource-meter-head">
                                        <span>{{ t('nav.memory') }}</span>
                                        <strong>{{ getContainerMemoryLabel(container.Id) }}</strong>
                                    </div>
                                    <div class="resource-progress">
                                        <div class="resource-progress-fill memory-fill"
                                            :style="{ width: `${getContainerMemoryPercent(container.Id)}%` }"></div>
                                    </div>
                                </div>
                                <div class="resource-network-line">
                                    {{ getContainerNetworkLabel(container.Id) }}
                                </div>
                            </div>
                            <span v-else class="card-meta-value muted">{{ t('containersView.noStats') }}</span>
                        </div>
                    </div>

                    <div class="card-actions-menu">
                        <button class="action-btn action-neutral card-menu-trigger" type="button"
                            :title="t('common.actions')" @click.stop="toggleCardMenu(container.Id)">
                            <Ellipsis :size="16" />
                        </button>
                        <div v-if="activeCardMenuId === container.Id" class="card-actions-popover glass-panel" @click.stop>
                            <button v-if="!container.Status.includes('Up')" class="card-action-item action-start" type="button"
                                @click="handleCardAction('start', container)">
                                <Play :size="16" />
                                {{ t('compose.start') }}
                            </button>
                            <button v-else class="card-action-item action-stop" type="button"
                                @click="handleCardAction('stop', container)">
                                <Square :size="16" />
                                {{ t('compose.stop') }}
                            </button>
                            <button class="card-action-item action-neutral" type="button" :disabled="!container.Status.includes('Up')"
                                @click="handleCardAction('restart', container)">
                                <RotateCw :size="16" />
                                {{ t('compose.restart') }}
                            </button>
                            <button class="card-action-item action-neutral" type="button"
                                @click="handleCardAction('logs', container)">
                                <FileText :size="16" />
                                {{ t('compose.logs') }}
                            </button>
                            <button class="card-action-item action-neutral" type="button"
                                @click="handleCardAction('terminal', container)">
                                <TerminalIcon :size="16" />
                                {{ t('containersView.terminalAction') }}
                            </button>
                            <button class="card-action-item action-danger" type="button"
                                @click="handleCardAction('remove', container)">
                                <Trash2 :size="16" />
                                {{ t('common.remove') }}
                            </button>
                        </div>
                    </div>
                </article>
            </div>
        </div>

        <div v-if="filteredContainers.length > 0" class="pagination glass-panel">
            <div class="pager-meta">
                <span>{{ t('common.rows') }}</span>
                <select v-model.number="pageSize">
                    <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select>
                <span>{{ pageStart }}-{{ pageEnd }} / {{ totalItems }}</span>
            </div>
            <div class="pager-actions">
                <button class="btn btn-ghost" :disabled="currentPage === 1" @click="currentPage--">{{ t('common.prev')
                    }}</button>
                <span class="pager-page">{{ t('common.page') }} {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost" :disabled="currentPage >= totalPages" @click="currentPage++">{{
                    t('common.next') }}</button>
            </div>
        </div>

        <Teleport to="body">
            <div v-if="showLogsModal" class="modal-backdrop" @click.self="closeLogs">
                <div class="modal-panel glass-panel logs-modal-panel" :class="{ 'is-expanded': logsModalExpanded }">
                    <div class="modal-header">
                        <div class="modal-title-wrap">
                            <div class="window-controls">
                                <button class="window-control is-close" type="button" :title="t('common.close')"
                                    :aria-label="t('containersView.closeLogs')" @click="closeLogs">
                                    <X :size="10" />
                                </button>
                                <button class="window-control is-minimize" type="button"
                                    :title="logsModalExpanded ? t('containersView.normalSize') : t('containersView.expand')"
                                    :aria-label="logsModalExpanded ? t('containersView.normalSize') : t('containersView.expandLogs')"
                                    @click="toggleLogsSize">
                                    <Minimize2 v-if="logsModalExpanded" :size="10" />
                                    <Maximize2 v-else :size="10" />
                                </button>
                                <button class="window-control is-zoom" type="button"
                                    :title="logsFollow ? t('compose.following') : t('containersView.jumpLatest')"
                                    :aria-label="logsFollow ? t('containersView.followingLogs') : t('containersView.jumpLatest')"
                                    @click="jumpToLatestLogs">
                                    <RefreshCw :size="10" />
                                </button>
                            </div>
                            <h3>{{ t('containersView.logsTitle', {
                                name: activeContainer?.Names?.[0]?.replace('/', '')
                                || '' }) }}</h3>
                        </div>
                        <div class="modal-actions">
                            <button class="btn btn-ghost btn-icon modal-tool-btn" type="button"
                                :title="t('containersView.decreaseFont')" :aria-label="t('containersView.decreaseFont')"
                                @click="adjustLogsFontSize(-1)">
                                <Minus :size="14" />
                            </button>
                            <button class="btn btn-ghost btn-icon modal-tool-btn" type="button"
                                :title="t('containersView.increaseFont')" :aria-label="t('containersView.increaseFont')"
                                @click="adjustLogsFontSize(1)">
                                <Plus :size="14" />
                            </button>
                            <button class="btn btn-ghost btn-icon modal-tool-btn" type="button"
                                :class="{ 'is-active': logsFollow }"
                                :title="logsFollow ? t('compose.following') : t('containersView.jumpLatest')"
                                :aria-label="logsFollow ? t('containersView.followingLogs') : t('containersView.jumpLatest')"
                                @click="jumpToLatestLogs">
                                <ArrowDownToLine :size="14" />
                            </button>
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
                            <div class="window-controls">
                                <button class="window-control is-close" type="button" :title="t('common.close')"
                                    :aria-label="t('containersView.closeTerminal')" @click="closeTerminal">
                                    <X :size="10" />
                                </button>
                                <button class="window-control is-minimize" type="button"
                                    :title="terminalModalExpanded ? t('containersView.normalSize') : t('containersView.expand')"
                                    :aria-label="terminalModalExpanded ? t('containersView.normalSize') : t('containersView.expandTerminal')"
                                    @click="toggleTerminalSize">
                                    <Minimize2 v-if="terminalModalExpanded" :size="10" />
                                    <Maximize2 v-else :size="10" />
                                </button>
                                <button class="window-control is-zoom" type="button"
                                    :title="terminalIsFullscreen ? t('containersView.exitFullscreen') : t('containersView.fullscreen')"
                                    :aria-label="terminalIsFullscreen ? t('containersView.exitFullscreen') : t('containersView.fullscreen')"
                                    @click="toggleTerminalFullscreen">
                                    <Minimize2 v-if="terminalIsFullscreen" :size="10" />
                                    <Maximize2 v-else :size="10" />
                                </button>
                            </div>
                            <h3>{{ t('containersView.terminalTitle', {
                                name: activeContainer?.Names?.[0]?.replace('/',
                                '') || '' }) }}</h3>
                            <span class="terminal-shell-pill">{{ appSettings.runtime.terminalShell }}</span>
                        </div>
                        <div class="modal-actions">
                            <select v-model="appSettings.runtime.terminalTheme" class="terminal-theme-select">
                                <option v-for="theme in terminalThemeOptions" :key="theme.value" :value="theme.value">
                                    {{ theme.label }}
                                </option>
                            </select>
                            <button class="btn btn-ghost btn-icon modal-tool-btn" type="button"
                                :title="t('containersView.decreaseFont')" :aria-label="t('containersView.decreaseFont')"
                                @click="adjustTerminalFontSize(-1)">
                                <Minus :size="14" />
                            </button>
                            <button class="btn btn-ghost btn-icon modal-tool-btn" type="button"
                                :title="t('containersView.increaseFont')" :aria-label="t('containersView.increaseFont')"
                                @click="adjustTerminalFontSize(1)">
                                <Plus :size="14" />
                            </button>
                            <button class="btn btn-ghost" @click="copyTerminalSelection">
                                <Copy :size="14" />
                                {{ t('containersView.copy') }}
                            </button>
                            <button class="btn btn-ghost" @click="pasteIntoTerminal">
                                <ClipboardPaste :size="14" />
                                {{ t('containersView.paste') }}
                            </button>
                        </div>
                    </div>
                    <div ref="terminalEl" class="terminal-output terminal-xterm"></div>
                </div>
            </div>
        </Teleport>
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
    flex-wrap: wrap;
}

.view-toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px;
    border-radius: 12px;
    border: 1px solid var(--glass-border);
    background: var(--glass);
}

.view-toggle-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 36px;
    padding: 0 12px;
    border: none;
    border-radius: 9px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    transition: all 0.18s ease;
}

.view-toggle-btn:hover {
    color: var(--text-main);
    background: rgba(255, 255, 255, 0.04);
}

.view-toggle-btn.is-active {
    background: color-mix(in srgb, var(--primary) 18%, var(--glass));
    color: var(--primary);
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

.card-container {
    min-width: 0;
}

.container-card-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 18px;
}

.container-card {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 18px;
}

.container-card-header {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: flex-start;
    gap: 12px;
    padding-right: 52px;
}

.card-title-area {
    min-width: 0;
}

.card-title-row {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
}

.card-check {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding-top: 2px;
}

.card-meta-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.card-meta-item {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 0;
}

.card-meta-label {
    font-size: 0.76rem;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    color: var(--text-muted);
}

.card-meta-value {
    color: var(--text-main);
}

.muted {
    color: var(--text-muted);
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

.sort-header {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 0;
    border: none;
    background: transparent;
    color: inherit;
    font: inherit;
    cursor: pointer;
}

.sort-indicator {
    font-size: 0.8em;
    color: var(--text-muted);
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
    width: 260px;
    min-width: 220px;
    max-width: 260px;
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
    display: block;
    width: 100%;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    box-sizing: border-box;
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

.ports-wrap {
    flex-wrap: wrap;
    overflow: visible;
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

.card-actions-menu {
    position: absolute;
    top: 18px;
    right: 18px;
}

.card-menu-trigger {
    background: rgba(255, 255, 255, 0.05);
}

.card-actions-popover {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 180px;
    padding: 8px;
    z-index: 5;
}

.card-action-item {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-height: 36px;
    padding: 0 12px;
    border-radius: 10px;
    border: 1px solid var(--glass-border);
    background: rgba(255, 255, 255, 0.03);
    color: var(--text-main);
    cursor: pointer;
    transition: all 0.18s ease;
}

.card-action-item:hover {
    transform: translateY(-1px);
}

.card-action-item:disabled {
    opacity: 0.45;
    cursor: not-allowed;
    transform: none;
}

.resource-meters {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.resource-meter {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.resource-meter-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    font-size: 0.82rem;
    color: var(--text-muted);
}

.resource-meter-head strong {
    color: var(--text-main);
    font-weight: 600;
    text-align: right;
}

.resource-progress {
    width: 100%;
    height: 9px;
    border-radius: 999px;
    overflow: hidden;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.04);
}

.resource-progress-fill {
    height: 100%;
    border-radius: inherit;
    transition: width 0.2s ease;
}

.cpu-fill {
    background: linear-gradient(90deg, #38bdf8, #2563eb);
}

.memory-fill {
    background: linear-gradient(90deg, #fbbf24, #f97316);
}

.resource-network-line {
    font-size: 0.8rem;
    color: var(--text-muted);
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

.card-empty-state {
    padding: 80px 24px;
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
    padding: 2px 18px;
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
    position: sticky;
    top: 0;
    z-index: 2;
    margin: -18px -18px 0;
    padding: 9px 18px 10px;
    background: color-mix(in srgb, var(--bg-card) 96%, transparent);
    border-bottom: 1px solid var(--glass-border);
}

.modal-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    justify-content: flex-end;
}

.modal-header h3 {
    margin: 0;
    font-size: 1rem;
}

.modal-title-wrap,
.terminal-title-wrap {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
}

.window-controls {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
    margin-top: -4px;
}

.window-control {
    width: 12px;
    height: 12px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    border: none;
    border-radius: 999px;
    cursor: pointer;
    color: rgba(20, 20, 20, 0.68);
    transition: transform 0.16s ease, filter 0.16s ease;
}

.window-control :deep(svg) {
    opacity: 0;
    transition: opacity 0.16s ease;
}

.window-controls:hover .window-control :deep(svg),
.window-control:focus-visible :deep(svg) {
    opacity: 1;
}

.window-control:hover {
    transform: scale(1.08);
    filter: brightness(0.98);
}

.window-control:focus-visible {
    outline: 2px solid color-mix(in srgb, var(--primary) 45%, transparent);
    outline-offset: 2px;
}

.window-control.is-close {
    background: #ff5f57;
}

.window-control.is-minimize {
    background: #febc2e;
}

.window-control.is-zoom {
    background: #28c840;
}

.modal-tool-btn {
    min-height: 32px;
    width: 32px;
    padding: 0;
    color: var(--text-muted);
}

.modal-tool-btn.is-active {
    border-color: rgba(36, 150, 237, 0.48);
    background: color-mix(in srgb, var(--primary) 16%, var(--glass));
    color: var(--primary);
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
    .toolbar {
        flex-direction: column;
        align-items: stretch;
    }

    .search-box {
        width: 100%;
    }

    .view-toggle {
        width: 100%;
        justify-content: space-between;
    }

    .view-toggle-btn {
        flex: 1;
        justify-content: center;
    }

    .container-card-grid {
        grid-template-columns: 1fr;
    }

    .card-title-row {
        flex-direction: column;
    }

    .container-card-header .status-pill {
        justify-self: flex-start;
    }

    .terminal-modal-panel,
    .logs-modal-panel {
        min-width: 0;
        width: min(100%, 96vw);
    }

    .modal-header {
        align-items: flex-start;
        flex-direction: column;
        position: static;
        margin: -18px -18px 0;
    }

    .modal-title-wrap,
    .terminal-title-wrap {
        flex-wrap: wrap;
    }
}
</style>
