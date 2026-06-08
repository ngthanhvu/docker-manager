<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { Eye, Play, RefreshCw, RotateCw, Save, Search, Square, Trash2, X } from 'lucide-vue-next';
import { useI18n } from '../i18n';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, loadStoredString, persistStoredValue } from '../ui/viewState';

type ComposeService = {
    id: string;
    name: string;
    image: string;
    state: string;
    status: string;
    created: number;
};

type ComposeProject = {
    name: string;
    status: string;
    running: number;
    total: number;
    services: ComposeService[];
    workingDir?: string;
};

type ComposeProjectFile = {
    path: string;
    kind?: 'compose' | 'env';
    content?: string;
    error?: string;
};

type ComposeWorkspaceView = 'compose' | 'logs';

const projects = ref<ComposeProject[]>([]);
const loadingProjects = ref(true);
const COMPOSE_SEARCH_KEY = 'dock-manager.compose.search';
const COMPOSE_SPLIT_RATIO_KEY = 'dock-manager.compose.split-ratio';
const searchQuery = ref(loadStoredString(COMPOSE_SEARCH_KEY, ''));

const selectedProjectName = ref('');
const selectedProject = computed(() => projects.value.find((p) => p.name === selectedProjectName.value) || null);
const filteredProjects = computed(() => {
    const query = searchQuery.value.trim().toLowerCase();
    if (!query) return projects.value;
    return projects.value.filter((project) => {
        if (project.name.toLowerCase().includes(query)) return true;
        if ((project.workingDir || '').toLowerCase().includes(query)) return true;
        return project.services.some((service) => service.name.toLowerCase().includes(query));
    });
});

const files = ref<ComposeProjectFile[]>([]);
const loadingFiles = ref(false);
const selectedFilePath = ref('');
const fileDraft = ref('');
const savingFile = ref(false);
const validatingCompose = ref(false);
const composeValidationError = ref('');
const editorInput = ref<HTMLTextAreaElement | null>(null);
const editorHighlight = ref<HTMLElement | null>(null);
const showingDiffPreview = ref(false);
const restartingAfterSave = ref(false);
const splitRatio = ref(loadStoredNumber(COMPOSE_SPLIT_RATIO_KEY, 0.64, 0.4, 0.78));

const logsRawOutput = ref('');
const logsTail = ref(appSettings.runtime.defaultLogTail);
const loadingLogs = ref(false);
const logsPanel = ref<HTMLElement | null>(null);
const logsSearchQuery = ref('');
const selectedLogService = ref('all');
const logsFollow = ref(true);
const activeWorkspaceView = ref<ComposeWorkspaceView>('compose');
const serviceActionLoadingId = ref('');
const splitRoot = ref<HTMLElement | null>(null);
let splitDragging = false;
let composeValidationTimer: number | null = null;
let composeValidationRequestId = 0;
const { t } = useI18n();

const fetchProjects = async () => {
    try {
        const { data } = await dockerApi.getComposeProjects();
        projects.value = data || [];
        if (!selectedProjectName.value && projects.value.length > 0) {
            selectedProjectName.value = projects.value[0]?.name || '';
        }
        if (selectedProjectName.value && !projects.value.some((p) => p.name === selectedProjectName.value)) {
            selectedProjectName.value = projects.value[0]?.name || '';
        }
    } catch (err) {
        console.error('Failed to fetch compose projects:', err);
    } finally {
        loadingProjects.value = false;
    }
};

const fetchFiles = async (projectName: string) => {
    loadingFiles.value = true;
    try {
        const { data } = await dockerApi.getComposeProjectFiles(projectName);
        files.value = data || [];
        const hasCurrent = files.value.some((f) => f.path === selectedFilePath.value);
        if (!hasCurrent) {
            selectedFilePath.value = files.value.find((f) => typeof f.content === 'string')?.path || files.value[0]?.path || '';
        }
        const selected = files.value.find((f) => f.path === selectedFilePath.value);
        fileDraft.value = selected?.content || '';
    } catch (err) {
        files.value = [{ path: t('common.notAvailable'), error: getErrorMessage(err) }];
        selectedFilePath.value = '';
        fileDraft.value = '';
    } finally {
        loadingFiles.value = false;
    }
};

const fetchLogs = async (projectName: string) => {
    loadingLogs.value = true;
    try {
        const shouldStickToBottom = logsFollow.value && isLogsNearBottom();
        const { data } = await dockerApi.getComposeProjectLogs(projectName, logsTail.value);
        logsRawOutput.value = data || '';
        await nextTick();
        if (logsPanel.value && shouldStickToBottom) logsPanel.value.scrollTop = logsPanel.value.scrollHeight;
    } catch (err) {
        logsRawOutput.value = t('compose.logsLoadFailed', { error: getErrorMessage(err) });
    } finally {
        loadingLogs.value = false;
    }
};

const loadDetails = async (projectName: string) => {
    await Promise.all([fetchFiles(projectName), fetchLogs(projectName)]);
};

const reloadDetailsWithGuard = async (projectName: string, message: string) => {
    const accepted = await confirmDiscardChanges(message);
    if (!accepted) return false;
    await loadDetails(projectName);
    return true;
};

const selectedFile = computed(() => files.value.find((f) => f.path === selectedFilePath.value) || null);
const selectedFileKind = computed(() => selectedFile.value?.kind || 'compose');
const selectedFileIsEnv = computed(() => selectedFileKind.value === 'env');
const selectedFileIsEditable = computed(() => !!selectedFile.value && !selectedFile.value.error && typeof selectedFile.value.content === 'string');
const hasMultipleComposeFiles = computed(() => files.value.length > 1);
const isDraftChanged = computed(() => {
    if (!selectedFileIsEditable.value) return false;
    return fileDraft.value !== (selectedFile.value?.content || '');
});
const originalFileContent = computed(() => selectedFile.value?.content || '');
const diffPreview = computed(() => buildDiffPreview(originalFileContent.value, fileDraft.value));
const canSaveCompose = computed(() =>
    isDraftChanged.value
    && selectedFileIsEditable.value
    && !savingFile.value
    && !validatingCompose.value
    && !composeValidationError.value
);
const logServiceOptions = computed(() => [
    { label: t('compose.allServices'), value: 'all' },
    ...((selectedProject.value?.services || []).map((service) => ({
        label: service.name,
        value: service.name,
    }))),
]);

const getFileName = (path: string) => path.split('/').filter(Boolean).pop() || path;
const getFileKindLabel = (file?: ComposeProjectFile | null) => (file?.kind === 'env' ? 'ENV' : 'YAML');
const getEditorTitle = (file?: ComposeProjectFile | null) => {
    if (!file) return t('compose.editFile');
    if (hasMultipleComposeFiles.value) return getFileName(file.path);
    return file.kind === 'env' ? t('compose.editEnvFile') : t('compose.editComposeFile');
};
const getProjectStatusLabel = (status: string) => {
    if (status === 'running') return t('compose.running');
    if (status === 'stopped') return t('compose.stopped');
    if (status === 'partial') return t('compose.partial');
    return status || t('compose.unknown');
};
const getServiceStateLabel = (state: string) => {
    if (state === 'running') return t('compose.running');
    if (state === 'stopped') return t('compose.stopped');
    if (state === 'exited') return t('compose.exited');
    if (state === 'dead') return t('compose.dead');
    if (state === 'partial') return t('compose.partial');
    return state || t('compose.unknown');
};

const escapeHtml = (value: string) =>
    value
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;');

const highlightCompose = (value: string) => {
    const escaped = escapeHtml(value);
    const lines = escaped.split('\n').map((line) => {
        if (!line.trim()) return '&nbsp;';

        let output = line;
        output = output.replace(/(&quot;.*?&quot;|".*?"|'.*?')/g, '<span class="tok-string">$1</span>');
        output = output.replace(/(#.*)$/g, '<span class="tok-comment">$1</span>');
        output = output.replace(/^(\s*-\s+)/g, '<span class="tok-punc">$1</span>');
        output = output.replace(/^(\s*)([A-Za-z0-9_.-]+)(\s*:)/g, '$1<span class="tok-key">$2</span><span class="tok-punc">$3</span>');
        output = output.replace(/\b(true|false|yes|no|on|off|null)\b/gi, '<span class="tok-bool">$1</span>');
        output = output.replace(/([:\s-])(\d+(?:\.\d+)?)(?=\s|$)/g, '$1<span class="tok-number">$2</span>');
        return output || '&nbsp;';
    });
    return lines.join('\n');
};

const highlightEnv = (value: string) => {
    const escaped = escapeHtml(value);
    const lines = escaped.split('\n').map((line) => {
        if (!line.trim()) return '&nbsp;';
        if (/^\s*#/.test(line)) return `<span class="tok-comment">${line}</span>`;
        const match = line.match(/^(\s*[A-Za-z_][A-Za-z0-9_.-]*)(=)(.*)$/);
        if (!match) return line;
        return `<span class="tok-key">${match[1]}</span><span class="tok-punc">${match[2]}</span><span class="tok-string">${match[3]}</span>`;
    });
    return lines.join('\n');
};

const highlightedDraft = computed(() => (selectedFileIsEnv.value ? highlightEnv(fileDraft.value) : highlightCompose(fileDraft.value)));

const ansiPalette: Record<string, string> = {
    '30': '#0f172a',
    '31': '#f87171',
    '32': '#4ade80',
    '33': '#fbbf24',
    '34': '#60a5fa',
    '35': '#c084fc',
    '36': '#22d3ee',
    '37': '#e2e8f0',
    '90': '#64748b',
    '91': '#fca5a5',
    '92': '#86efac',
    '93': '#fde68a',
    '94': '#93c5fd',
    '95': '#d8b4fe',
    '96': '#67e8f9',
    '97': '#f8fafc',
};

const ansiToHtml = (value: string) => {
    let html = '';
    const stack: string[] = [];
    const regex = /\x1b\[([0-9;]*)m/g;
    let lastIndex = 0;

    for (const match of value.matchAll(regex)) {
        const index = match.index ?? 0;
        html += escapeHtml(value.slice(lastIndex, index));
        lastIndex = index + match[0].length;
        const codes = (match[1] || '0').split(';').filter(Boolean);

        if (codes.length === 0 || codes.includes('0')) {
            while (stack.length) html += stack.pop();
            continue;
        }

        const styles: string[] = [];
        for (const code of codes) {
            if (code === '1') styles.push('font-weight:700');
            else if (code === '3') styles.push('font-style:italic');
            else if (code === '4') styles.push('text-decoration:underline');
            else if (ansiPalette[code]) styles.push(`color:${ansiPalette[code]}`);
        }

        if (styles.length > 0) {
            html += `<span style="${styles.join(';')}">`;
            stack.push('</span>');
        }
    }

    html += escapeHtml(value.slice(lastIndex));
    while (stack.length) html += stack.pop();
    return html;
};

type ParsedLogBlock = {
    service: string;
    header: string;
    body: string;
};

const parseComposeLogs = (raw: string): ParsedLogBlock[] => {
    if (!raw.trim()) return [];
    const blocks = raw.split(/^===== /m).filter(Boolean);
    return blocks.map((block) => {
        const trimmed = block.trimStart();
        const firstLineEnd = trimmed.indexOf('\n');
        const headerLine = firstLineEnd >= 0 ? trimmed.slice(0, firstLineEnd) : trimmed;
        const body = firstLineEnd >= 0 ? trimmed.slice(firstLineEnd + 1) : '';
        const service = headerLine.split(' (')[0]?.trim() || 'unknown';
        return {
            service,
            header: `===== ${headerLine}`,
            body,
        };
    });
};

const filteredLogBlocks = computed(() => {
    const query = logsSearchQuery.value.trim().toLowerCase();
    return parseComposeLogs(logsRawOutput.value).filter((block) => {
        if (selectedLogService.value !== 'all' && block.service !== selectedLogService.value) return false;
        if (!query) return true;
        return `${block.header}\n${block.body}`.toLowerCase().includes(query);
    });
});

const renderedLogsHtml = computed(() => {
    if (filteredLogBlocks.value.length === 0) {
        return escapeHtml(logsRawOutput.value || t('compose.noLogsYet'));
    }

    return filteredLogBlocks.value
        .map((block) => {
            const header = `<span class="log-block-header">${escapeHtml(block.header)}</span>`;
            const body = ansiToHtml(block.body);
            return `${header}\n${body}`;
        })
        .join('\n\n');
});

const highlightDiffLine = (line: string) => {
    const escaped = escapeHtml(line);
    if (line.startsWith('+++') || line.startsWith('---') || line.startsWith('@@')) {
        return `<span class="diff-meta">${escaped}</span>`;
    }
    if (line.startsWith('+')) {
        return `<span class="diff-add">${escaped}</span>`;
    }
    if (line.startsWith('-')) {
        return `<span class="diff-remove">${escaped}</span>`;
    }
    return escaped;
};

const buildDiffPreview = (original: string, edited: string) => {
    if (original === edited) return [t('compose.noChangesYet')];

    const oldLines = original.split('\n');
    const newLines = edited.split('\n');
    const max = Math.max(oldLines.length, newLines.length);
    const lines: string[] = [`--- ${t('compose.currentLabel')}`, `+++ ${t('compose.draftLabel')}`];

    for (let i = 0; i < max; i += 1) {
        const before = oldLines[i];
        const after = newLines[i];
        if (before === after) {
            if (before !== undefined) {
                lines.push(`  ${before}`);
            }
            continue;
        }
        if (before !== undefined) {
            lines.push(`- ${before}`);
        }
        if (after !== undefined) {
            lines.push(`+ ${after}`);
        }
    }

    return lines;
};

const syncEditorScroll = () => {
    if (!editorInput.value || !editorHighlight.value) return;
    editorHighlight.value.scrollTop = editorInput.value.scrollTop;
    editorHighlight.value.scrollLeft = editorInput.value.scrollLeft;
};

const isLogsNearBottom = () => {
    const el = logsPanel.value;
    if (!el) return true;
    return el.scrollHeight - el.scrollTop - el.clientHeight < 48;
};

const handleLogsScroll = () => {
    logsFollow.value = isLogsNearBottom();
};

const jumpToLatestLogs = async () => {
    logsFollow.value = true;
    await nextTick();
    if (logsPanel.value) logsPanel.value.scrollTop = logsPanel.value.scrollHeight;
};

const stopSplitDrag = () => {
    splitDragging = false;
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
    persistStoredValue(COMPOSE_SPLIT_RATIO_KEY, splitRatio.value);
};

const handleSplitDrag = (event: MouseEvent) => {
    if (!splitDragging || !splitRoot.value) return;
    const rect = splitRoot.value.getBoundingClientRect();
    if (!rect.width) return;
    const nextRatio = (event.clientX - rect.left) / rect.width;
    splitRatio.value = Math.min(0.78, Math.max(0.4, nextRatio));
};

const confirmDiscardChanges = async (message: string) => {
    if (!isDraftChanged.value) return true;
    return feedback.confirmAction({
        title: t('compose.discardTitle'),
        message,
        confirmText: t('compose.discard'),
        cancelText: t('compose.stay'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DISCARD' : undefined,
    });
};

const selectFile = async (path: string) => {
    if (path === selectedFilePath.value) return;
    if (isDraftChanged.value) {
        const accepted = await confirmDiscardChanges(t('compose.discardSwitchFile'));
        if (!accepted) return;
    }
    selectedFilePath.value = path;
    const file = files.value.find((f) => f.path === path);
    fileDraft.value = file?.content || '';
};

const resetDraft = () => {
    if (!selectedFile.value) return;
    fileDraft.value = selectedFile.value.content || '';
    composeValidationError.value = '';
    syncEditorScroll();
};

const getErrorMessage = (err: unknown) => {
    if (typeof err === 'string') return err;
    if (err && typeof err === 'object') {
        const maybeAxiosError = err as {
            response?: { data?: unknown };
            message?: string;
        };
        if (typeof maybeAxiosError.response?.data === 'string') return maybeAxiosError.response.data;
        if (typeof maybeAxiosError.message === 'string') return maybeAxiosError.message;
    }
    return t('common.unknownError');
};

const validateDraftNow = async () => {
    if (!selectedProjectName.value || !selectedFile.value || !selectedFileIsEditable.value || selectedFileIsEnv.value) {
        composeValidationError.value = '';
        validatingCompose.value = false;
        return;
    }

    const requestId = ++composeValidationRequestId;
    validatingCompose.value = true;
    try {
        const { data } = await dockerApi.validateComposeProjectFile(selectedProjectName.value, {
            path: selectedFile.value.path,
            content: fileDraft.value,
        });
        if (requestId !== composeValidationRequestId) return;
        composeValidationError.value = data?.valid ? '' : String(data?.error || t('compose.invalidDefault'));
    } catch (err) {
        if (requestId !== composeValidationRequestId) return;
        composeValidationError.value = getErrorMessage(err);
    } finally {
        if (requestId !== composeValidationRequestId) return;
        validatingCompose.value = false;
    }
};

const scheduleComposeValidation = () => {
    if (composeValidationTimer) window.clearTimeout(composeValidationTimer);
    if (!selectedFileIsEditable.value || !selectedFile.value || selectedFileIsEnv.value) {
        composeValidationError.value = '';
        validatingCompose.value = false;
        return;
    }
    composeValidationTimer = window.setTimeout(() => {
        validateDraftNow();
    }, 350);
};

const saveSelectedFile = async (restartAfter = false) => {
    if (!selectedProjectName.value || !selectedFile.value || !selectedFileIsEditable.value || savingFile.value) return;
    if (validatingCompose.value) {
        feedback.warning(t('compose.validationInProgress'));
        return;
    }
    if (composeValidationError.value) {
        feedback.error(t('compose.validationFailed', { error: composeValidationError.value }));
        return;
    }
    try {
        savingFile.value = true;
        restartingAfterSave.value = restartAfter;
        await dockerApi.updateComposeProjectFile(selectedProjectName.value, {
            path: selectedFile.value.path,
            content: fileDraft.value,
        });
        feedback.success(selectedFileIsEnv.value ? t('compose.savedEnv') : t('compose.savedCompose'));
        if (restartAfter) {
            await dockerApi.restartComposeProject(selectedProjectName.value);
            feedback.success(t('compose.projectRestarted', { name: selectedProjectName.value }));
        }
        await fetchFiles(selectedProjectName.value);
        composeValidationError.value = '';
        syncEditorScroll();
        showingDiffPreview.value = false;
    } catch (err) {
        feedback.error(t('compose.saveFailed', { error: String(err) }));
    } finally {
        savingFile.value = false;
        restartingAfterSave.value = false;
    }
};

const selectProject = async (projectName: string) => {
    if (!projectName) return;
    if (projectName !== selectedProjectName.value) {
        const accepted = await confirmDiscardChanges(t('compose.discardSwitchProject'));
        if (!accepted) return;
    }
    selectedProjectName.value = projectName;
    await loadDetails(projectName);
};

const runAction = async (action: 'start' | 'stop' | 'restart' | 'down', projectName: string) => {
    try {
        if (selectedProjectName.value === projectName) {
            const accepted = await confirmDiscardChanges(t('compose.discardProjectAction', { action }));
            if (!accepted) return;
        }
        if (action === 'down') {
            const accepted = await feedback.confirmAction({
                title: t('compose.downTitle'),
                message: t('compose.downMessage', { name: projectName }),
                confirmText: t('compose.down'),
                danger: true,
                requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
            });
            if (!accepted) return;
            await dockerApi.downComposeProject(projectName);
        } else if (action === 'start') {
            await dockerApi.startComposeProject(projectName);
        } else if (action === 'stop') {
            await dockerApi.stopComposeProject(projectName);
        } else {
            await dockerApi.restartComposeProject(projectName);
        }
        await fetchProjects();
        if (selectedProjectName.value) {
            await loadDetails(selectedProjectName.value);
        }
        if (action === 'start') feedback.success(t('compose.projectStarted', { name: projectName }));
        else if (action === 'stop') feedback.success(t('compose.projectStopped', { name: projectName }));
        else if (action === 'restart') feedback.success(t('compose.projectRestarted', { name: projectName }));
        else feedback.success(t('compose.projectDown', { name: projectName }));
    } catch (err) {
        feedback.error(t('compose.actionFailed', { error: String(err) }));
    }
};

const runServiceAction = async (action: 'start' | 'stop' | 'restart', service: ComposeService) => {
    if (!service?.id) return;
    try {
        const accepted = await confirmDiscardChanges(t('compose.discardServiceAction', { action, name: service.name }));
        if (!accepted) return;
        serviceActionLoadingId.value = service.id;
        if (action === 'start') {
            await dockerApi.startContainer(service.id);
        } else if (action === 'stop') {
            await dockerApi.stopContainer(service.id);
        } else {
            if (service.state === 'running') {
                await dockerApi.stopContainer(service.id);
            }
            await dockerApi.startContainer(service.id);
        }

        await fetchProjects();
        if (selectedProjectName.value) {
            await loadDetails(selectedProjectName.value);
        }
        if (action === 'start') feedback.success(t('compose.serviceStarted', { name: service.name }));
        else if (action === 'stop') feedback.success(t('compose.serviceStopped', { name: service.name }));
        else feedback.success(t('compose.serviceRestarted', { name: service.name }));
    } catch (err) {
        feedback.error(t('compose.serviceActionFailed', { error: String(err) }));
    } finally {
        serviceActionLoadingId.value = '';
    }
};

const getProjectStatusClass = (status: string) => {
    if (status === 'running') return 'ok';
    if (status === 'stopped') return 'bad';
    return 'warn';
};

const getServiceClass = (state: string) => {
    if (state === 'running') return 'ok';
    if (state === 'exited' || state === 'dead') return 'bad';
    return 'warn';
};

let projectsInterval: any;
let logsInterval: any;
const handleBeforeUnload = (event: BeforeUnloadEvent) => {
    if (!isDraftChanged.value) return;
    event.preventDefault();
    event.returnValue = '';
};

const handleComposeShortcut = (event: KeyboardEvent) => {
    if (!selectedFileIsEditable.value || !selectedProjectName.value) return;
    if (!(event.ctrlKey || event.metaKey) || event.key.toLowerCase() !== 's') return;
    event.preventDefault();
    if (event.shiftKey) {
        saveSelectedFile(true);
        return;
    }
    saveSelectedFile(false);
};

const setupIntervals = () => {
    clearInterval(projectsInterval);
    clearInterval(logsInterval);
    const ms = appSettings.runtime.composeRefreshMs;
    if (ms <= 0) return;
    projectsInterval = setInterval(fetchProjects, ms);
    logsInterval = setInterval(() => {
        if (selectedProjectName.value) fetchLogs(selectedProjectName.value);
    }, ms);
};

onMounted(async () => {
    await fetchProjects();
    if (selectedProjectName.value) {
        await loadDetails(selectedProjectName.value);
    }
    setupIntervals();
    window.addEventListener('beforeunload', handleBeforeUnload);
    window.addEventListener('mousemove', handleSplitDrag);
    window.addEventListener('mouseup', stopSplitDrag);
    window.addEventListener('keydown', handleComposeShortcut);
});

onUnmounted(() => {
    clearInterval(projectsInterval);
    clearInterval(logsInterval);
    if (composeValidationTimer) window.clearTimeout(composeValidationTimer);
    window.removeEventListener('beforeunload', handleBeforeUnload);
    window.removeEventListener('mousemove', handleSplitDrag);
    window.removeEventListener('mouseup', stopSplitDrag);
    window.removeEventListener('keydown', handleComposeShortcut);
    stopSplitDrag();
});

watch(() => appSettings.runtime.composeRefreshMs, () => {
    setupIntervals();
});

watch(() => appSettings.runtime.defaultLogTail, (next) => {
    logsTail.value = next;
});

watch(selectedProjectName, () => {
    selectedFilePath.value = '';
    fileDraft.value = '';
    composeValidationError.value = '';
    validatingCompose.value = false;
    selectedLogService.value = 'all';
    logsSearchQuery.value = '';
    logsFollow.value = true;
});

watch(fileDraft, async () => {
    await nextTick();
    syncEditorScroll();
    scheduleComposeValidation();
});

watch(searchQuery, (next) => {
    persistStoredValue(COMPOSE_SEARCH_KEY, next);
});

watch(selectedFilePath, () => {
    composeValidationError.value = '';
    validatingCompose.value = false;
    scheduleComposeValidation();
});
</script>

<template>
    <div class="compose-layout">
        <aside class="left-col glass-panel">
            <div class="left-header">
                <h3>{{ t('compose.title') }}</h3>
                <button class="btn btn-ghost compact-btn" @click="fetchProjects">
                    <RefreshCw :size="16" :class="{ 'animate-spin': loadingProjects }" />
                    {{ t('compose.refresh') }}
                </button>
            </div>

            <div class="search-box">
                <Search :size="16" />
                <input v-model="searchQuery" type="text" :placeholder="t('compose.searchPlaceholder')" />
            </div>

            <div class="project-list">
                <button v-for="project in filteredProjects" :key="project.name" class="project-item"
                    :class="{ active: selectedProjectName === project.name }" @click="selectProject(project.name)">
                    <div class="row-1">
                        <span class="name">{{ project.name }}</span>
                        <span class="status" :class="getProjectStatusClass(project.status)">{{ getProjectStatusLabel(project.status) }}</span>
                    </div>
                    <div class="row-2">{{ project.running }} / {{ project.total }}</div>
                </button>
                <div v-if="filteredProjects.length === 0 && !loadingProjects" class="empty">{{ t('compose.noProjects') }}</div>
            </div>
        </aside>

        <section class="right-col glass-panel">
            <div v-if="selectedProject" class="detail-wrap">
                <div class="detail-header">
                    <div>
                        <h2>{{ selectedProject.name }}</h2>
                        <p class="path">{{ selectedProject.workingDir || t('common.notAvailable') }}</p>
                    </div>
                    <div class="actions">
                        <div class="action-cluster">
                            <button class="btn btn-ghost action-btn" :title="t('compose.start')"
                                @click="runAction('start', selectedProject.name)">
                                <Play :size="16" />
                                <span>{{ t('compose.start') }}</span>
                            </button>
                            <button class="btn btn-ghost action-btn" :title="t('compose.stop')"
                                @click="runAction('stop', selectedProject.name)">
                                <Square :size="16" />
                                <span>{{ t('compose.stop') }}</span>
                            </button>
                            <button class="btn btn-ghost action-btn" :title="t('compose.restart')"
                                @click="runAction('restart', selectedProject.name)">
                                <RotateCw :size="16" />
                                <span>{{ t('compose.restart') }}</span>
                            </button>
                            <button class="btn btn-ghost action-btn"
                                @click="reloadDetailsWithGuard(selectedProject.name, t('compose.discardReload'))">
                                <RefreshCw :size="16" />
                                <span>{{ t('compose.reload') }}</span>
                            </button>
                        </div>
                        <button class="btn btn-danger-soft action-btn danger-btn" :title="t('compose.down')"
                            @click="runAction('down', selectedProject.name)">
                            <Trash2 :size="16" />
                            <span>{{ t('compose.down') }}</span>
                        </button>
                    </div>
                </div>

                <div class="services-panel">
                    <div class="panel-head services-head">
                        <h4>{{ t('compose.services') }}</h4>
                        <span class="hint">{{ t('compose.containersCount', { count: selectedProject.services.length }) }}</span>
                    </div>
                    <div class="services-table-wrap">
                        <table class="services-table">
                            <thead>
                                <tr>
                                    <th>{{ t('compose.service') }}</th>
                                    <th>{{ t('compose.state') }}</th>
                                    <th>{{ t('containersView.image') }}</th>
                                    <th class="service-actions-col">{{ t('common.actions') }}</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="service in selectedProject.services" :key="service.id">
                                    <td>{{ service.name }}</td>
                                    <td>
                                        <span class="service-state" :class="getServiceClass(service.state)">
                                            {{ getServiceStateLabel(service.state) }}
                                        </span>
                                    </td>
                                    <td><code>{{ service.image }}</code></td>
                                    <td class="service-actions-col">
                                        <div class="service-actions">
                                            <button class="btn btn-ghost compact-btn"
                                                :disabled="serviceActionLoadingId === service.id || service.state === 'running'"
                                                @click="runServiceAction('start', service)">
                                                <Play :size="14" />
                                                {{ t('compose.start') }}
                                            </button>
                                            <button class="btn btn-ghost compact-btn"
                                                :disabled="serviceActionLoadingId === service.id || service.state !== 'running'"
                                                @click="runServiceAction('stop', service)">
                                                <Square :size="14" />
                                                {{ t('compose.stop') }}
                                            </button>
                                            <button class="btn btn-ghost compact-btn"
                                                :disabled="serviceActionLoadingId === service.id"
                                                @click="runServiceAction('restart', service)">
                                                <RotateCw :size="14"
                                                    :class="{ 'animate-spin': serviceActionLoadingId === service.id }" />
                                                {{ t('compose.restart') }}
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <div class="workspace-toggle" role="tablist" :aria-label="t('compose.workspaceAria')">
                    <button type="button" class="workspace-toggle-btn"
                        :class="{ active: activeWorkspaceView === 'compose' }" @click="activeWorkspaceView = 'compose'">
                        {{ t('compose.title') }}
                    </button>
                    <button type="button" class="workspace-toggle-btn"
                        :class="{ active: activeWorkspaceView === 'logs' }" @click="activeWorkspaceView = 'logs'">
                        {{ t('compose.viewLogs') }}
                    </button>
                </div>

                <div ref="splitRoot" class="split split-single">
                    <div v-if="activeWorkspaceView === 'compose'" class="panel">
                        <div class="panel-head">
                            <div class="compose-files-head">
                                <h4>{{ t('compose.composeFiles') }}</h4>
                                <span v-if="isDraftChanged" class="dirty-badge">{{ t('compose.unsavedChanges') }}</span>
                            </div>
                            <span class="hint">
                                {{ loadingFiles ? t('compose.loading') : selectedFile ? getFileName(selectedFile.path) :
                                    t('compose.filesCount', { count: files.length }) }}
                            </span>
                        </div>
                        <div class="panel-body file-body file-editor-layout"
                            :class="{ 'has-file-list': hasMultipleComposeFiles }">
                            <div v-if="files.length === 0 && !loadingFiles" class="empty">{{ t('compose.noProjectFiles') }}</div>

                            <div v-if="hasMultipleComposeFiles" class="file-list">
                                <button v-for="file in files" :key="file.path" class="file-item-btn"
                                    :class="{ active: selectedFilePath === file.path }" @click="selectFile(file.path)">
                                    <span class="file-kind" :class="{ 'file-kind-env': file.kind === 'env' }">{{ getFileKindLabel(file) }}</span>
                                    <span class="file-path-short">{{ getFileName(file.path) }}</span>
                                </button>
                            </div>

                            <div v-if="selectedFile" class="file-editor-box">
                                <div class="file-path">
                                    <div class="file-path-meta">
                                        <span class="file-chip" :class="{ 'file-chip-env': selectedFile.kind === 'env' }">{{ getFileKindLabel(selectedFile) }}</span>
                                        <span>{{ getEditorTitle(selectedFile) }}</span>
                                    </div>
                                    <div v-if="selectedFile" class="editor-actions editor-actions-inline">
                                        <button class="btn btn-ghost compact-btn"
                                            :disabled="!isDraftChanged || !selectedFileIsEditable || !!composeValidationError || validatingCompose"
                                            @click="showingDiffPreview = true">
                                            <Eye :size="14" />
                                            {{ t('compose.previewDiff') }}
                                        </button>
                                        <button class="btn btn-ghost compact-btn"
                                            :disabled="!isDraftChanged || savingFile || !selectedFileIsEditable"
                                            @click="resetDraft">
                                            <X :size="14" />
                                            {{ t('common.reset') }}
                                        </button>
                                        <button class="btn btn-primary compact-btn" :disabled="!canSaveCompose"
                                            @click="() => saveSelectedFile()">
                                            <Save :size="14" :class="{ 'animate-spin': savingFile }" />
                                            {{ t('common.save') }}
                                        </button>
                                        <button class="btn btn-ghost compact-btn save-restart-btn"
                                            :disabled="!canSaveCompose" @click="saveSelectedFile(true)">
                                            <RotateCw :size="14" :class="{ 'animate-spin': restartingAfterSave }" />
                                            {{ t('compose.saveAndRestart') }}
                                        </button>
                                    </div>
                                </div>

                                <div v-if="selectedFileIsEditable && !selectedFileIsEnv && (validatingCompose || composeValidationError || isDraftChanged)"
                                    class="validation-banner"
                                    :class="{ invalid: !!composeValidationError, valid: !composeValidationError && isDraftChanged }">
                                    <span v-if="validatingCompose">{{ t('compose.validating') }}</span>
                                    <span v-else-if="composeValidationError">{{ composeValidationError }}</span>
                                    <span v-else>{{ t('compose.valid') }}</span>
                                </div>

                                <div v-if="selectedFileIsEditable" class="editor-shell">
                                    <pre ref="editorHighlight" class="code editor-highlight"
                                        v-html="highlightedDraft"></pre>
                                    <textarea ref="editorInput" v-model="fileDraft" class="code editor"
                                        spellcheck="false" @scroll="syncEditorScroll" />
                                </div>
                                <pre v-else class="code error">{{ t('compose.cannotReadFile', { error: selectedFile.error }) }}</pre>
                            </div>
                        </div>
                    </div>

                    <div v-else class="panel">
                        <div class="panel-head">
                            <h4>{{ t('compose.logs') }}</h4>
                            <div class="log-controls">
                                <select v-model="selectedLogService" class="log-control log-service-select">
                                    <option v-for="option in logServiceOptions" :key="option.value"
                                        :value="option.value">
                                        {{ option.label }}
                                    </option>
                                </select>
                                <input v-model="logsSearchQuery" type="text" :placeholder="t('compose.searchLogs')"
                                    class="log-control log-search" />
                                <label>{{ t('compose.tail') }}</label>
                                <input v-model.number="logsTail" type="number" min="50" max="2000" step="50"
                                    class="log-control log-tail-input" />
                                <button class="btn btn-ghost compact-btn" :class="{ 'is-active': logsFollow }"
                                    @click="jumpToLatestLogs">
                                    {{ logsFollow ? t('compose.following') : t('compose.follow') }}
                                </button>
                                <button class="btn btn-ghost compact-btn" @click="fetchLogs(selectedProject.name)">
                                    <RefreshCw :size="14" :class="{ 'animate-spin': loadingLogs }" />
                                    {{ t('compose.refresh') }}
                                </button>
                            </div>
                        </div>
                        <pre ref="logsPanel" class="panel-body logs compose-logs"
                            @scroll="handleLogsScroll"><code v-html="renderedLogsHtml"></code></pre>
                    </div>
                </div>
            </div>

            <div v-else class="empty-state">{{ t('compose.selectProject') }}</div>
        </section>

        <div v-if="showingDiffPreview" class="diff-modal-backdrop" @click.self="showingDiffPreview = false">
            <div class="diff-modal glass-panel">
                <div class="diff-modal-head">
                    <div>
                        <h3>{{ t('compose.diffTitle') }}</h3>
                        <p>{{ selectedFile ? getFileName(selectedFile.path) : t('compose.currentFile') }}</p>
                    </div>
                    <div class="diff-modal-actions">
                        <button class="btn btn-ghost compact-btn" @click="showingDiffPreview = false">
                            {{ t('common.close') }}
                        </button>
                        <button class="btn btn-primary compact-btn" :disabled="!canSaveCompose"
                            @click="() => saveSelectedFile()">
                            <Save :size="14" :class="{ 'animate-spin': savingFile && !restartingAfterSave }" />
                            {{ t('common.save') }}
                        </button>
                    </div>
                </div>
                <pre
                    class="diff-view"><code v-for="(line, idx) in diffPreview" :key="idx" class="diff-line" v-html="highlightDiffLine(line)"></code></pre>
            </div>
        </div>
    </div>
</template>

<style scoped>
.compose-layout {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 16px;
    height: calc(100vh - 135px);
    min-height: 0;
}

.left-col,
.right-col {
    padding: 14px;
    min-height: 0;
    overflow: hidden;
}

.left-col {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.left-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
}

.left-header h3 {
    margin: 0;
}

.compact-btn {
    padding: 7px 10px;
    font-size: 0.82rem;
}

.search-box {
    display: flex;
    align-items: center;
    gap: 8px;
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    background: var(--glass);
    padding: 8px 10px;
}

.search-box input {
    width: 100%;
    background: transparent;
    border: none;
    outline: none;
    color: var(--text-main);
}

.project-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    flex: 1;
    min-height: 0;
    overflow: auto;
}

.project-item {
    width: 100%;
    text-align: left;
    border: 1px solid var(--glass-border);
    background: var(--glass);
    color: var(--text-main);
    border-radius: 10px;
    padding: 10px;
    cursor: pointer;
}

.project-item.active {
    border-color: var(--primary);
    box-shadow: inset 0 0 0 1px var(--primary);
}

.row-1 {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 10px;
}

.row-2 {
    margin-top: 6px;
    color: var(--text-muted);
    font-size: 0.8rem;
}

.name {
    font-weight: 600;
}

.status {
    text-transform: uppercase;
    font-size: 0.7rem;
    font-weight: 700;
}

.status.ok {
    color: var(--success);
}

.status.warn {
    color: var(--warning);
}

.status.bad {
    color: var(--danger);
}

.detail-wrap {
    display: flex;
    flex-direction: column;
    gap: 12px;
    height: 100%;
    min-height: 0;
}

.detail-header {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: flex-start;
    min-width: 0;
    flex-wrap: wrap;
}

.detail-header h2 {
    margin: 0;
    font-size: 1.3rem;
}

.path {
    margin: 4px 0 0;
    color: var(--text-muted);
    font-size: 0.82rem;
    word-break: break-all;
}

.actions {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-wrap: nowrap;
    overflow-x: auto;
    padding-bottom: 2px;
    min-width: 0;
}

.action-cluster {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: nowrap;
}

.action-btn {
    padding: 7px 11px;
    font-size: 0.82rem;
    border: 1px solid var(--glass-border);
}

.danger-btn {
    border-color: rgba(239, 68, 68, 0.35);
}

.btn-danger-soft {
    background: rgba(239, 68, 68, 0.1);
    color: #fca5a5;
}

.btn-danger-soft:hover {
    background: rgba(239, 68, 68, 0.2);
    color: #fecaca;
}

.services-panel {
    border: 1px solid var(--glass-border);
    border-radius: 10px;
    overflow: hidden;
}

.services-head {
    border-bottom: 1px solid var(--glass-border);
}

.services-table-wrap {
    overflow: auto;
    max-height: 128px;
}

.services-table {
    width: 100%;
    border-collapse: collapse;
}

.services-table th,
.services-table td {
    padding: 10px 12px;
    border-bottom: 1px solid var(--glass-border);
    font-size: 0.84rem;
    text-align: left;
}

.services-table th {
    color: var(--text-muted);
    font-weight: 600;
    position: sticky;
    top: 0;
    z-index: 1;
    background: var(--table-header-bg);
}

.services-table tr:last-child td {
    border-bottom: none;
}

.service-actions-col {
    width: 260px;
    text-align: right !important;
}

.service-actions {
    display: flex;
    justify-content: flex-end;
    gap: 6px;
}

.service-state {
    text-transform: uppercase;
    font-size: 0.68rem;
    font-weight: 700;
}

.service-state.ok {
    color: var(--success);
}

.service-state.warn {
    color: var(--warning);
}

.service-state.bad {
    color: var(--danger);
}

.split {
    display: grid;
    gap: 12px;
    min-height: 0;
    flex: 1;
    align-items: stretch;
    min-width: 0;
}

.split.split-single {
    display: flex;
    flex-direction: column;
}

.workspace-toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    width: fit-content;
    padding: 4px;
    border: 1px solid var(--glass-border);
    border-radius: 10px;
    background: color-mix(in srgb, var(--glass) 88%, transparent);
}

.workspace-toggle-btn {
    border: 0;
    background: transparent;
    color: var(--text-muted);
    padding: 8px 14px;
    border-radius: 7px;
    font-size: 0.88rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;
}

.workspace-toggle-btn.active {
    background: var(--primary);
    color: #fff;
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--primary-hover) 60%, transparent);
}

.workspace-toggle-btn:hover:not(.active) {
    background: color-mix(in srgb, var(--primary) 12%, var(--glass));
    color: var(--text-main);
}

.panel {
    border: 1px solid var(--glass-border);
    border-radius: 10px;
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
    overflow: hidden;
}

.splitter {
    position: relative;
    min-width: 10px;
    cursor: col-resize;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 999px;
}

.splitter::before {
    content: '';
    position: absolute;
    top: 6px;
    bottom: 6px;
    left: 50%;
    width: 1px;
    transform: translateX(-50%);
    background: rgba(148, 163, 184, 0.18);
}

.splitter-grip {
    width: 6px;
    height: 44px;
    border-radius: 999px;
    background: rgba(96, 165, 250, 0.18);
    border: 1px solid rgba(96, 165, 250, 0.2);
    box-shadow: 0 0 0 1px rgba(15, 23, 42, 0.08);
}

.splitter:hover .splitter-grip {
    background: rgba(96, 165, 250, 0.26);
    border-color: rgba(96, 165, 250, 0.3);
}

.panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--glass-border);
    min-width: 0;
    flex-wrap: wrap;
}

.panel-head h4 {
    margin: 0;
}

.compose-files-head {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
}

.dirty-badge {
    display: inline-flex;
    align-items: center;
    padding: 3px 8px;
    border-radius: 999px;
    font-size: 0.72rem;
    font-weight: 700;
    color: #fcd34d;
    background: rgba(245, 158, 11, 0.14);
    border: 1px solid rgba(245, 158, 11, 0.28);
}

.hint {
    color: var(--text-muted);
    font-size: 0.8rem;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.panel-body {
    margin: 0;
    overflow: auto;
    padding: 10px;
    min-height: 0;
    flex: 1 1 0%;
}

.file-body {
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow: hidden;
    flex: 1 1 0%;
}

.file-editor-layout {
    display: grid;
    grid-template-columns: minmax(180px, 220px) minmax(0, 1fr);
    gap: 10px;
    min-height: 0;
    align-items: stretch;
    align-content: start;
}

.file-editor-layout:not(.has-file-list)>.file-editor-box {
    grid-column: 1 / -1;
}

.file-editor-layout.has-file-list>.file-editor-box {
    grid-column: 2 / 3;
}

.file-list {
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    overflow: auto;
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
    min-height: 0;
    max-width: 100%;
}

.file-item-btn {
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    background: var(--glass);
    color: var(--text-main);
    text-align: left;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    cursor: pointer;
}

.file-item-btn.active {
    border-color: var(--primary);
    box-shadow: inset 0 0 0 1px var(--primary);
}

.file-kind {
    display: inline-flex;
    width: fit-content;
    padding: 2px 6px;
    border-radius: 999px;
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    color: #93c5fd;
    background: rgba(59, 130, 246, 0.14);
    border: 1px solid rgba(59, 130, 246, 0.3);
}

.file-kind-env {
    color: #86efac;
    background: rgba(34, 197, 94, 0.14);
    border-color: rgba(34, 197, 94, 0.3);
}

.file-path-short {
    font-size: 0.74rem;
    color: var(--text-muted);
    word-break: break-all;
}

.file-editor-box {
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
}

.file-box {
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    overflow: hidden;
}

.file-path {
    font-size: 0.78rem;
    color: var(--text-muted);
    padding: 10px 12px;
    border-bottom: 1px solid var(--glass-border);
    background: var(--glass);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    min-width: 0;
    flex-wrap: nowrap;
}

.file-path-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    flex: 1 1 auto;
    overflow: hidden;
}

.file-chip {
    flex-shrink: 0;
    font-size: 0.66rem;
    font-weight: 700;
    color: #60a5fa;
    border: 1px solid rgba(96, 165, 250, 0.35);
    background: rgba(96, 165, 250, 0.14);
    border-radius: 999px;
    padding: 2px 6px;
}

.file-chip-env {
    color: #86efac;
    border-color: rgba(34, 197, 94, 0.35);
    background: rgba(34, 197, 94, 0.14);
}

.file-path-meta span:last-child {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.validation-banner {
    padding: 8px 10px;
    border-bottom: 1px solid var(--glass-border);
    background: rgba(22, 101, 52, 0.12);
    color: #bbf7d0;
    font-size: 0.8rem;
}

.validation-banner.invalid {
    background: rgba(127, 29, 29, 0.18);
    color: #fecaca;
}

.validation-banner.valid {
    background: rgba(22, 101, 52, 0.12);
    color: #bbf7d0;
}

.code {
    margin: 0;
    background: var(--code-bg);
    color: var(--code-text);
    font-size: 0.8rem;
    line-height: 1.35;
    padding: 10px;
    overflow: visible;
    max-height: none;
    white-space: pre;
    min-width: 0;
}

.code.error {
    color: #fda4af;
}

.editor-shell {
    position: relative;
    flex: 1;
    min-height: 540px;
    background:
        linear-gradient(180deg, rgba(37, 99, 235, 0.06), transparent 18%),
        var(--code-bg);
}

.editor-highlight,
.editor {
    position: absolute;
    inset: 0;
    margin: 0;
    padding: 14px 14px 16px;
    overflow: auto;
    font-family: var(--font-mono);
    font-size: 0.82rem;
    line-height: 1.5;
    white-space: pre;
    tab-size: 2;
}

.editor-highlight {
    pointer-events: none;
    color: #d8dee9;
}

.editor {
    border: none;
    outline: none;
    resize: none;
    min-height: 540px;
    width: 100%;
    color: transparent;
    background: transparent;
    caret-color: #f8fafc;
    -webkit-text-fill-color: transparent;
}

.editor::selection {
    background: rgba(96, 165, 250, 0.22);
}

.editor-highlight :deep(.tok-key) {
    color: #7dd3fc;
}

.editor-highlight :deep(.tok-punc) {
    color: #94a3b8;
}

.editor-highlight :deep(.tok-string) {
    color: #86efac;
}

.editor-highlight :deep(.tok-comment) {
    color: #64748b;
}

.editor-highlight :deep(.tok-number) {
    color: #f9a8d4;
}

.editor-highlight :deep(.tok-bool) {
    color: #fbbf24;
}

.editor-actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    flex-wrap: nowrap;
    gap: 8px;
}

.editor-actions-inline {
    flex: 0 1 auto;
    margin-left: auto;
    flex-shrink: 0;
}

.editor-actions .btn {
    white-space: nowrap;
    width: auto;
    min-width: fit-content;
    justify-content: center;
}

.editor-actions-inline .btn {
    padding: 5px 8px;
    min-height: 30px;
    font-size: 0.74rem;
    gap: 5px;
}

.editor-actions-inline :deep(svg) {
    width: 13px;
    height: 13px;
}

.save-restart-btn {
    border-color: rgba(59, 130, 246, 0.34);
}

.diff-modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 60;
    background: rgba(2, 6, 23, 0.62);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
}

.diff-modal {
    width: min(1080px, 94vw);
    max-height: 88vh;
    display: flex;
    flex-direction: column;
    min-height: 0;
    padding: 16px;
    gap: 12px;
}

.diff-modal-head {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
}

.diff-modal-head h3 {
    margin: 0;
    font-size: 1.05rem;
}

.diff-modal-head p {
    margin: 4px 0 0;
    color: var(--text-muted);
    font-size: 0.82rem;
}

.diff-modal-actions {
    display: flex;
    align-items: center;
    gap: 8px;
}

.diff-view {
    margin: 0;
    min-height: 0;
    flex: 1;
    overflow: auto;
    padding: 14px;
    border-radius: 10px;
    border: 1px solid var(--glass-border);
    background: linear-gradient(180deg, rgba(15, 23, 42, 0.92), rgba(2, 6, 23, 0.96));
    font-family: var(--font-mono);
    font-size: 0.8rem;
    line-height: 1.45;
}

.diff-line {
    display: block;
    white-space: pre-wrap;
    word-break: break-word;
}

.diff-view :deep(.diff-meta) {
    color: #93c5fd;
}

.diff-view :deep(.diff-add) {
    color: #86efac;
}

.diff-view :deep(.diff-remove) {
    color: #fca5a5;
}

.logs {
    background: var(--code-bg);
    color: var(--code-text);
    font-size: 0.8rem;
    line-height: 1.35;
    min-width: 0;
    min-height: 0;
    height: 100%;
}

.compose-logs {
    white-space: pre-wrap;
    word-break: break-word;
}

.compose-logs :deep(code) {
    display: block;
    font-family: var(--font-mono);
}

.compose-logs :deep(.log-block-header) {
    color: #93c5fd;
    font-weight: 700;
}

.log-controls {
    display: flex;
    align-items: center;
    gap: 6px;
    justify-content: flex-end;
    min-width: 0;
}

.log-controls label {
    color: var(--text-muted);
    font-size: 0.74rem;
    flex-shrink: 0;
}

.log-controls input,
.log-controls select {
    border: 1px solid var(--glass-border);
    border-radius: 6px;
    padding: 3px 6px;
    background: var(--glass);
    color: var(--text-main);
    min-width: 0;
    min-height: 30px;
    font-size: 0.78rem;
}

.log-controls select {
    appearance: none;
    -webkit-appearance: none;
    -moz-appearance: none;
    background-color: var(--input-bg);
    color-scheme: inherit;
}

.log-controls select option {
    background: var(--select-option-bg);
    color: var(--select-option-text);
}

.log-control {
    flex: 0 1 auto;
    min-width: 0;
}

.log-service-select {
    width: 140px;
}

.log-search {
    width: min(180px, 100%);
}

.log-tail-input {
    width: 60px;
}

.log-controls .is-active {
    border-color: rgba(36, 150, 237, 0.45);
    color: var(--primary);
}

.log-controls .compact-btn {
    padding: 5px 8px;
    min-height: 30px;
    font-size: 0.74rem;
    gap: 5px;
}

.log-controls :deep(svg) {
    width: 13px;
    height: 13px;
}

.text-danger {
    color: var(--danger) !important;
}

.empty,
.empty-state {
    color: var(--text-muted);
    text-align: center;
    padding: 20px;
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

@media (max-width: 1100px) {
    .compose-layout {
        grid-template-columns: 1fr;
        height: auto;
    }

    .split {
        grid-template-columns: 1fr;
    }

    .splitter {
        display: none;
    }

    .file-editor-layout {
        grid-template-columns: 1fr;
    }

    .file-editor-layout.has-file-list>.file-editor-box {
        grid-column: 1 / -1;
    }

    .file-list {
        max-height: 170px;
    }

    .actions {
        width: 100%;
        flex-wrap: wrap;
        overflow-x: visible;
    }

    .editor-actions {
        justify-content: flex-start;
        flex-wrap: wrap;
    }

    .file-path {
        align-items: stretch;
        flex-wrap: wrap;
    }

    .file-path-meta {
        width: 100%;
    }

    .editor-actions-inline {
        width: 100%;
        flex: 1 1 100%;
        min-width: 0;
        margin-left: 0;
    }

    .editor-shell,
    .editor {
        min-height: 460px;
    }

    .service-actions-col {
        width: auto;
    }

    .service-actions {
        justify-content: flex-start;
        flex-wrap: wrap;
    }

    .log-controls {
        justify-content: flex-start;
    }
}

@media (max-width: 1320px) {
    .detail-header {
        align-items: stretch;
    }

    .actions {
        width: 100%;
    }

    .split {
        grid-template-columns: 1fr;
    }

    .splitter {
        display: none;
    }
}
</style>
