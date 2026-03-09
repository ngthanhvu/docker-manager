<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { Play, RefreshCw, RotateCw, Search, Square, Trash2 } from 'lucide-vue-next';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';

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
    content?: string;
    error?: string;
};

const projects = ref<ComposeProject[]>([]);
const loadingProjects = ref(true);
const searchQuery = ref('');

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

const logsOutput = ref('');
const logsTail = ref(appSettings.runtime.defaultLogTail);
const loadingLogs = ref(false);
const logsPanel = ref<HTMLElement | null>(null);
const serviceActionLoadingId = ref('');

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
    } catch (err) {
        files.value = [{ path: 'N/A', error: String(err) }];
    } finally {
        loadingFiles.value = false;
    }
};

const fetchLogs = async (projectName: string) => {
    loadingLogs.value = true;
    try {
        const { data } = await dockerApi.getComposeProjectLogs(projectName, logsTail.value);
        logsOutput.value = data || '';
        await nextTick();
        if (logsPanel.value) logsPanel.value.scrollTop = logsPanel.value.scrollHeight;
    } catch (err) {
        logsOutput.value = `Failed to fetch logs: ${err}`;
    } finally {
        loadingLogs.value = false;
    }
};

const loadDetails = async (projectName: string) => {
    await Promise.all([fetchFiles(projectName), fetchLogs(projectName)]);
};

const selectProject = async (projectName: string) => {
    if (!projectName) return;
    selectedProjectName.value = projectName;
    await loadDetails(projectName);
};

const runAction = async (action: 'start' | 'stop' | 'restart' | 'down', projectName: string) => {
    try {
        if (action === 'down') {
            const accepted = await feedback.confirmAction({
                title: 'Bring Down Compose',
                message: `Bring down compose project "${projectName}"? This will remove project containers.`,
                confirmText: 'Down',
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
        if (action === 'start') feedback.success(`Compose "${projectName}" started successfully.`);
        else if (action === 'stop') feedback.success(`Compose "${projectName}" stopped successfully.`);
        else if (action === 'restart') feedback.success(`Compose "${projectName}" restarted successfully.`);
        else feedback.success(`Compose "${projectName}" down successfully.`);
    } catch (err) {
        feedback.error(`Compose action failed: ${err}`);
    }
};

const runServiceAction = async (action: 'start' | 'stop' | 'restart', service: ComposeService) => {
    if (!service?.id) return;
    try {
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
        const actionLabel = action === 'start' ? 'started' : action === 'stop' ? 'stopped' : 'restarted';
        feedback.success(`Service "${service.name}" ${actionLabel} successfully.`);
    } catch (err) {
        feedback.error(`Service action failed: ${err}`);
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
});

onUnmounted(() => {
    clearInterval(projectsInterval);
    clearInterval(logsInterval);
});

watch(() => appSettings.runtime.composeRefreshMs, () => {
    setupIntervals();
});

watch(() => appSettings.runtime.defaultLogTail, (next) => {
    logsTail.value = next;
});
</script>

<template>
    <div class="compose-layout">
        <aside class="left-col glass-panel">
            <div class="left-header">
                <h3>Compose</h3>
                <button class="btn btn-ghost compact-btn" @click="fetchProjects">
                    <RefreshCw :size="16" :class="{ 'animate-spin': loadingProjects }" />
                    Refresh
                </button>
            </div>

            <div class="search-box">
                <Search :size="16" />
                <input v-model="searchQuery" type="text" placeholder="Search compose..." />
            </div>

            <div class="project-list">
                <button
                    v-for="project in filteredProjects"
                    :key="project.name"
                    class="project-item"
                    :class="{ active: selectedProjectName === project.name }"
                    @click="selectProject(project.name)"
                >
                    <div class="row-1">
                        <span class="name">{{ project.name }}</span>
                        <span class="status" :class="getProjectStatusClass(project.status)">{{ project.status }}</span>
                    </div>
                    <div class="row-2">{{ project.running }} / {{ project.total }} running</div>
                </button>
                <div v-if="filteredProjects.length === 0 && !loadingProjects" class="empty">No projects found</div>
            </div>
        </aside>

        <section class="right-col glass-panel">
            <div v-if="selectedProject" class="detail-wrap">
                <div class="detail-header">
                    <div>
                        <h2>{{ selectedProject.name }}</h2>
                        <p class="path">{{ selectedProject.workingDir || 'No working directory label' }}</p>
                    </div>
                    <div class="actions">
                        <div class="action-cluster">
                            <button class="btn btn-ghost action-btn" title="Start" @click="runAction('start', selectedProject.name)">
                            <Play :size="16" />
                            <span>Start</span>
                        </button>
                        <button class="btn btn-ghost action-btn" title="Stop" @click="runAction('stop', selectedProject.name)">
                            <Square :size="16" />
                            <span>Stop</span>
                        </button>
                        <button class="btn btn-ghost action-btn" title="Restart" @click="runAction('restart', selectedProject.name)">
                            <RotateCw :size="16" />
                            <span>Restart</span>
                        </button>
                        <button class="btn btn-ghost action-btn" @click="loadDetails(selectedProject.name)">
                            <RefreshCw :size="16" />
                            <span>Reload</span>
                        </button>
                        </div>
                        <button class="btn btn-danger-soft action-btn danger-btn" title="Down" @click="runAction('down', selectedProject.name)">
                            <Trash2 :size="16" />
                            <span>Down</span>
                        </button>
                    </div>
                </div>

                <div class="services-panel">
                    <div class="panel-head services-head">
                        <h4>Services</h4>
                        <span class="hint">{{ selectedProject.services.length }} container(s)</span>
                    </div>
                    <div class="services-table-wrap">
                        <table class="services-table">
                            <thead>
                                <tr>
                                    <th>Service</th>
                                    <th>State</th>
                                    <th>Image</th>
                                    <th class="service-actions-col">Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                <tr v-for="service in selectedProject.services" :key="service.id">
                                    <td>{{ service.name }}</td>
                                    <td>
                                        <span class="service-state" :class="getServiceClass(service.state)">
                                            {{ service.state }}
                                        </span>
                                    </td>
                                    <td><code>{{ service.image }}</code></td>
                                    <td class="service-actions-col">
                                        <div class="service-actions">
                                            <button
                                                class="btn btn-ghost compact-btn"
                                                :disabled="serviceActionLoadingId === service.id || service.state === 'running'"
                                                @click="runServiceAction('start', service)">
                                                <Play :size="14" />
                                                Start
                                            </button>
                                            <button
                                                class="btn btn-ghost compact-btn"
                                                :disabled="serviceActionLoadingId === service.id || service.state !== 'running'"
                                                @click="runServiceAction('stop', service)">
                                                <Square :size="14" />
                                                Stop
                                            </button>
                                            <button
                                                class="btn btn-ghost compact-btn"
                                                :disabled="serviceActionLoadingId === service.id"
                                                @click="runServiceAction('restart', service)">
                                                <RotateCw :size="14" :class="{ 'animate-spin': serviceActionLoadingId === service.id }" />
                                                Restart
                                            </button>
                                        </div>
                                    </td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>

                <div class="split">
                    <div class="panel">
                        <div class="panel-head">
                            <h4>Compose Files</h4>
                            <span class="hint">{{ loadingFiles ? 'Loading...' : `${files.length} file(s)` }}</span>
                        </div>
                        <div class="panel-body file-body">
                            <div v-if="files.length === 0 && !loadingFiles" class="empty">No compose files</div>
                            <div v-for="file in files" :key="file.path" class="file-box">
                                <div class="file-path">{{ file.path }}</div>
                                <pre v-if="file.content" class="code">{{ file.content }}</pre>
                                <pre v-else class="code error">Cannot read file: {{ file.error }}</pre>
                            </div>
                        </div>
                    </div>

                    <div class="panel">
                        <div class="panel-head">
                            <h4>Logs (Realtime)</h4>
                            <div class="log-controls">
                                <label>Tail</label>
                                <input v-model.number="logsTail" type="number" min="50" max="2000" step="50" />
                                <button class="btn btn-ghost compact-btn" @click="fetchLogs(selectedProject.name)">
                                    <RefreshCw :size="14" :class="{ 'animate-spin': loadingLogs }" />
                                    Refresh
                                </button>
                            </div>
                        </div>
                        <pre ref="logsPanel" class="panel-body logs">{{ logsOutput || 'No logs yet.' }}</pre>
                    </div>
                </div>
            </div>

            <div v-else class="empty-state">Select a compose project from the left list.</div>
        </section>
    </div>
</template>

<style scoped>
.compose-layout {
    display: grid;
    grid-template-columns: 320px 1fr;
    gap: 16px;
    height: calc(100vh - 210px);
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
    max-height: 150px;
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
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    min-height: 0;
    flex: 1;
}

.panel {
    border: 1px solid var(--glass-border);
    border-radius: 10px;
    display: flex;
    flex-direction: column;
    min-height: 0;
    overflow: hidden;
}

.panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 10px;
    padding: 10px 12px;
    border-bottom: 1px solid var(--glass-border);
}

.panel-head h4 {
    margin: 0;
}

.hint {
    color: var(--text-muted);
    font-size: 0.8rem;
}

.panel-body {
    margin: 0;
    overflow: auto;
    padding: 10px;
    min-height: 0;
}

.file-body {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.file-box {
    border: 1px solid var(--glass-border);
    border-radius: 8px;
    overflow: hidden;
}

.file-path {
    font-size: 0.78rem;
    color: var(--text-muted);
    padding: 6px 8px;
    border-bottom: 1px solid var(--glass-border);
    background: var(--glass);
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
}

.code.error {
    color: #fda4af;
}

.logs {
    background: var(--code-bg);
    color: var(--code-text);
    font-size: 0.8rem;
    line-height: 1.35;
}

.log-controls {
    display: flex;
    align-items: center;
    gap: 6px;
}

.log-controls label {
    color: var(--text-muted);
    font-size: 0.8rem;
}

.log-controls input {
    width: 70px;
    border: 1px solid var(--glass-border);
    border-radius: 6px;
    padding: 4px 6px;
    background: var(--glass);
    color: var(--text-main);
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

    .actions {
        width: 100%;
        flex-wrap: wrap;
        overflow-x: visible;
    }

    .service-actions-col {
        width: auto;
    }

    .service-actions {
        justify-content: flex-start;
        flex-wrap: wrap;
    }
}
</style>
