<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { HardDrive, Trash2, RefreshCw, BrushCleaning, List, LayoutGrid } from 'lucide-vue-next';
import { useI18n } from 'vue-i18n';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';
import dayjs from 'dayjs';

const volumes = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const VOLUME_PAGE_SIZE_KEY = 'dock-manager.volumes.page-size';
const VOLUME_VIEW_MODE_KEY = 'dock-manager.volumes.view-mode';
const pageSize = ref(loadStoredNumber(VOLUME_PAGE_SIZE_KEY, 10, 10, 50));
const viewMode = ref<'list' | 'card'>(localStorage.getItem(VOLUME_VIEW_MODE_KEY) === 'card' ? 'card' : 'list');
const pageSizeOptions = [10, 20, 50];
const selectedNames = ref<string[]>([]);
const deletingName = ref<string | null>(null);
const bulkDeleting = ref(false);
const pruning = ref(false);
const { t } = useI18n();

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

const fetchVolumes = async () => {
    try {
        loading.value = true;
        const { data } = await dockerApi.getVolumes();
        volumes.value = data || [];
    } catch (err) {
        console.error('Failed to fetch volumes:', err);
    } finally {
        loading.value = false;
    }
};

const removeVolume = async (name: string) => {
    if (deletingName.value || bulkDeleting.value) return;
    const accepted = await feedback.confirmAction({
        title: t('volumesView.deleteTitle'),
        message: t('volumesView.deleteMessage'),
        confirmText: t('common.delete'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        deletingName.value = name;
        await dockerApi.removeVolume(name);
        selectedNames.value = selectedNames.value.filter((x) => x !== name);
        await fetchVolumes();
        feedback.success(t('volumesView.deletedSuccess'));
    } catch (err) {
        feedback.error(t('volumesView.deleteFailed', { error: String(err) }));
    } finally {
        deletingName.value = null;
    }
};

const bulkDelete = async () => {
    if (selectedNames.value.length === 0 || bulkDeleting.value || deletingName.value) return;
    const namesToDelete = [...selectedNames.value];
    const removeCount = namesToDelete.length;
    const accepted = await feedback.confirmAction({
        title: t('volumesView.deleteManyTitle'),
        message: t('volumesView.deleteManyMessage', { count: removeCount }),
        confirmText: t('common.delete'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;

    const deleted: string[] = [];
    const failed: Array<{ name: string; reason: string }> = [];

    try {
        bulkDeleting.value = true;
        for (const name of namesToDelete) {
            try {
                await dockerApi.removeVolume(name);
                deleted.push(name);
            } catch (err) {
                failed.push({ name, reason: getErrorMessage(err) });
            }
        }

        selectedNames.value = failed.map((item) => item.name);
        await fetchVolumes();

        if (deleted.length > 0) {
            feedback.success(t('volumesView.deletedManySuccess', { deleted: deleted.length, total: removeCount }));
        }

        if (failed.length > 0) {
            const failedNames = failed.map((item) => item.name).join(', ');
            const usedVolumes = failed.filter((item) => item.reason.toLowerCase().includes('in use'));
            const message = usedVolumes.length === failed.length
                ? t('volumesView.deleteInUseFailed', { count: failed.length, names: failedNames })
                : t('volumesView.deleteSomeFailed', { count: failed.length, names: failedNames });
            feedback.warning(message);
        }
    } finally {
        bulkDeleting.value = false;
    }
};

const pruneVolumes = async () => {
    if (pruning.value || bulkDeleting.value || deletingName.value) return;
    const accepted = await feedback.confirmAction({
        title: t('volumesView.pruneTitle'),
        message: t('volumesView.pruneMessage'),
        confirmText: t('common.prune'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;

    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneVolumes();
        await fetchVolumes();
        const deletedCount = Array.isArray(data?.VolumesDeleted) ? data.VolumesDeleted.length : 0;
        feedback.success(t('volumesView.prunedSuccess', { count: deletedCount }));
    } catch (err) {
        feedback.error(t('volumesView.pruneFailed', { error: getErrorMessage(err) }));
    } finally {
        pruning.value = false;
    }
};

const totalItems = computed(() => volumes.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
const paginatedVolumes = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    return volumes.value.slice(start, start + pageSize.value);
});
const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));

const pageVolumeNames = computed(() => paginatedVolumes.value.map((v) => v.Name));
const selectedCount = computed(() => selectedNames.value.length);
const allPageSelected = computed(() => pageVolumeNames.value.length > 0 && pageVolumeNames.value.every((n) => selectedNames.value.includes(n)));

const toggleSelect = (name: string) => {
    if (selectedNames.value.includes(name)) selectedNames.value = selectedNames.value.filter((x) => x !== name);
    else selectedNames.value = [...selectedNames.value, name];
};

const toggleSelectAllPage = () => {
    if (allPageSelected.value) selectedNames.value = selectedNames.value.filter((n) => !pageVolumeNames.value.includes(n));
    else selectedNames.value = Array.from(new Set([...selectedNames.value, ...pageVolumeNames.value]));
};

watch(pageSize, () => {
    currentPage.value = 1;
    persistStoredValue(VOLUME_PAGE_SIZE_KEY, pageSize.value);
});
watch(viewMode, () => {
    persistStoredValue(VOLUME_VIEW_MODE_KEY, viewMode.value);
});
watch(totalPages, (maxPage) => { if (currentPage.value > maxPage) currentPage.value = maxPage; });
watch(volumes, (list) => {
    const valid = new Set(list.map((v) => v.Name));
    selectedNames.value = selectedNames.value.filter((n) => valid.has(n));
});

onMounted(fetchVolumes);
</script>

<template>
    <div class="volume-list-view">
        <div class="toolbar glass-panel">
            <div class="title-with-icon">
                <HardDrive :size="20" class="icon-indigo" />
                <h2>{{ t('volumesView.title') }}</h2>
            </div>
            <div class="toolbar-actions">
                <div class="view-toggle" role="group" :aria-label="t('common.viewMode')">
                    <button class="view-toggle-btn" :class="{ 'is-active': viewMode === 'list' }" type="button"
                        :title="t('common.listView')" @click="viewMode = 'list'">
                        <List :size="16" />
                        {{ t('common.listView') }}
                    </button>
                    <button class="view-toggle-btn" :class="{ 'is-active': viewMode === 'card' }" type="button"
                        :title="t('common.cardView')" @click="viewMode = 'card'">
                        <LayoutGrid :size="16" />
                        {{ t('common.cardView') }}
                    </button>
                </div>
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0 || bulkDeleting || !!deletingName || pruning" @click="bulkDelete">
                    <RefreshCw v-if="bulkDeleting" :size="16" class="animate-spin" />
                    <Trash2 v-else :size="16" />
                    {{ bulkDeleting ? t('volumesView.deleting', { count: selectedCount }) : t('volumesView.delete', { count: selectedCount }) }}
                </button>
                <button class="btn btn-ghost text-warning" :disabled="bulkDeleting || !!deletingName || pruning" @click="pruneVolumes">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    {{ t('common.prune') }}
                </button>
                <button class="btn btn-ghost" :disabled="bulkDeleting || !!deletingName || pruning" @click="fetchVolumes">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                    {{ t('common.refresh') }}
                </button>
            </div>
        </div>

        <div v-if="viewMode === 'list'" class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="allPageSelected" :disabled="bulkDeleting || !!deletingName || pruning" @change="toggleSelectAllPage" /></th>
                        <th>{{ t('volumesView.name') }}</th>
                        <th>{{ t('volumesView.driver') }}</th>
                        <th>{{ t('volumesView.mountpoint') }}</th>
                        <th>{{ t('volumesView.created') }}</th>
                        <th class="actions-cell">{{ t('common.actions') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="vol in paginatedVolumes" :key="vol.Name">
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="selectedNames.includes(vol.Name)" :disabled="bulkDeleting || !!deletingName || pruning" @change="toggleSelect(vol.Name)" /></td>
                        <td class="name-cell">{{ vol.Name }}</td>
                        <td>{{ vol.Driver }}</td>
                        <td class="mountpoint-cell">
                            <div class="mountpoint-trigger">
                                <code class="mountpoint-value">{{ vol.Mountpoint || '-' }}</code>
                                <div v-if="vol.Mountpoint" class="mountpoint-tooltip" role="tooltip">
                                    {{ vol.Mountpoint }}
                                </div>
                            </div>
                        </td>
                        <td>{{ vol.CreatedAt ? dayjs(vol.CreatedAt).format('YYYY-MM-DD HH:mm') : t('common.notAvailable') }}</td>
                        <td class="actions-cell">
                            <div class="action-group">
                                <button class="action-btn action-danger" :disabled="bulkDeleting || !!deletingName || pruning" :title="t('common.remove')" @click="removeVolume(vol.Name)">
                                    <RefreshCw v-if="deletingName === vol.Name" :size="16" class="animate-spin" />
                                    <Trash2 v-else :size="16" />
                                </button>
                            </div>
                        </td>
                    </tr>
                    <tr v-if="volumes.length === 0 && !loading">
                        <td colspan="6" class="empty-state">{{ t('volumesView.noItems') }}</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-else class="card-container">
            <div v-if="volumes.length === 0 && !loading" class="glass-panel card-empty-state">
                {{ t('volumesView.noItems') }}
            </div>
            <div v-else class="card-grid">
                <article v-for="vol in paginatedVolumes" :key="vol.Name" class="glass-panel entity-card">
                    <div class="entity-card-header">
                        <label class="card-check">
                            <input class="bulk-checkbox" type="checkbox" :checked="selectedNames.includes(vol.Name)"
                                :disabled="bulkDeleting || !!deletingName || pruning" @change="toggleSelect(vol.Name)" />
                        </label>
                        <div class="card-title-block">
                            <div class="name-cell">{{ vol.Name }}</div>
                        </div>
                    </div>
                    <div class="card-meta-list">
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('volumesView.driver') }}</span>
                            <span>{{ vol.Driver }}</span>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('volumesView.mountpoint') }}</span>
                            <div class="mountpoint-trigger">
                                <code class="mountpoint-value">{{ vol.Mountpoint || '-' }}</code>
                                <div v-if="vol.Mountpoint" class="mountpoint-tooltip" role="tooltip">
                                    {{ vol.Mountpoint }}
                                </div>
                            </div>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('volumesView.created') }}</span>
                            <span>{{ vol.CreatedAt ? dayjs(vol.CreatedAt).format('YYYY-MM-DD HH:mm') : t('common.notAvailable') }}</span>
                        </div>
                    </div>
                    <div class="action-group card-action-group">
                        <button class="action-btn action-danger" :disabled="bulkDeleting || !!deletingName || pruning"
                            :title="t('common.remove')" @click="removeVolume(vol.Name)">
                            <RefreshCw v-if="deletingName === vol.Name" :size="16" class="animate-spin" />
                            <Trash2 v-else :size="16" />
                        </button>
                    </div>
                </article>
            </div>
        </div>

        <div v-if="volumes.length > 0" class="pagination glass-panel">
            <div class="pager-meta">
                <span>{{ t('common.rows') }}</span>
                <select v-model.number="pageSize" :disabled="bulkDeleting || !!deletingName || pruning">
                    <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select>
                <span>{{ pageStart }}-{{ pageEnd }} / {{ totalItems }}</span>
            </div>
            <div class="pager-actions">
                <button class="btn btn-ghost" :disabled="currentPage === 1 || bulkDeleting || !!deletingName || pruning" @click="currentPage--">{{ t('common.prev') }}</button>
                <span class="pager-page">{{ t('common.page') }} {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost" :disabled="currentPage >= totalPages || bulkDeleting || !!deletingName || pruning" @click="currentPage++">{{ t('common.next') }}</button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.volume-list-view { display: flex; flex-direction: column; gap: 24px; }
.toolbar { padding: 12px 24px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.toolbar-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.title-with-icon { display: flex; align-items: center; gap: 12px; }
.title-with-icon h2 { font-size: 1.2rem; margin: 0; }
.icon-indigo { color: var(--primary); }
.view-toggle { display: inline-flex; align-items: center; gap: 4px; padding: 4px; border-radius: 12px; border: 1px solid var(--glass-border); background: var(--glass); }
.view-toggle-btn { display: inline-flex; align-items: center; gap: 8px; min-height: 36px; padding: 0 12px; border: none; border-radius: 9px; background: transparent; color: var(--text-muted); cursor: pointer; transition: all 0.18s ease; }
.view-toggle-btn:hover { color: var(--text-main); background: rgba(255, 255, 255, 0.04); }
.view-toggle-btn.is-active { background: color-mix(in srgb, var(--primary) 18%, var(--glass)); color: var(--primary); }
.table-container { overflow: hidden; }
.card-container { min-width: 0; }
.card-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 18px; }
.entity-card { display: flex; flex-direction: column; gap: 16px; padding: 18px; }
.entity-card-header { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: flex-start; gap: 12px; }
.card-check { display: inline-flex; align-items: center; justify-content: center; padding-top: 2px; }
.card-title-block { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.card-meta-list { display: flex; flex-direction: column; gap: 12px; }
.card-meta-item { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.card-meta-label { font-size: 0.76rem; letter-spacing: 0.04em; text-transform: uppercase; color: var(--text-muted); }
.docker-table { width: 100%; border-collapse: collapse; }
.docker-table th { text-align: left; padding: 14px 20px; font-size: 0.86rem; color: var(--text-muted); border-bottom: 1px solid var(--glass-border); }
.docker-table td { padding: 14px 20px; font-size: 0.88rem; border-bottom: 1px solid var(--glass-border); }
.check-col { width: 56px; text-align: center !important; padding: 10px !important; }
.bulk-checkbox { width: 22px; height: 22px; cursor: pointer; accent-color: var(--primary); border-radius: 6px; }
.bulk-checkbox:hover { filter: brightness(1.08); }
.bulk-checkbox:focus-visible { outline: 2px solid rgba(36, 150, 237, 0.55); outline-offset: 2px; }
.docker-table tr:last-child td { border-bottom: none; }
.docker-table tr:hover { background: var(--glass); }
.name-cell { font-weight: 600; word-break: break-all; }
.mountpoint-cell { width: 300px; max-width: 300px; }
.mountpoint-trigger {
    position: relative;
    width: 100%;
}
.mountpoint-value {
    display: block;
    width: 100%;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    box-sizing: border-box;
}
.mountpoint-tooltip {
    position: absolute;
    left: 0;
    top: calc(100% + 8px);
    z-index: 20;
    min-width: 100%;
    max-width: min(520px, 60vw);
    padding: 10px 12px;
    border: 1px solid var(--glass-border);
    background: rgba(15, 23, 42, 0.96);
    color: #f8fafc;
    font-size: 0.78rem;
    line-height: 1.45;
    white-space: normal;
    word-break: break-all;
    box-shadow: 0 14px 32px rgba(15, 23, 42, 0.3);
    opacity: 0;
    pointer-events: none;
    transform: translateY(4px);
    transition: opacity 0.16s ease, transform 0.16s ease;
}
.mountpoint-trigger:hover .mountpoint-tooltip {
    opacity: 1;
    transform: translateY(0);
}
.actions-cell { width: 100px; text-align: center; }
.action-group { display: flex; align-items: center; justify-content: center; }
.card-action-group { justify-content: flex-start; }
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
.action-danger {
    color: #fda4af;
    border-color: rgba(239, 68, 68, 0.32);
    background: rgba(239, 68, 68, 0.08);
}
.action-danger:hover {
    background: rgba(239, 68, 68, 0.16);
    border-color: rgba(239, 68, 68, 0.55);
}
.empty-state { text-align: center; color: var(--text-muted); padding: 56px 0; }
.card-empty-state { text-align: center; color: var(--text-muted); padding: 56px 24px; }
.pagination { padding: 10px 14px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.pager-meta, .pager-actions { display: flex; align-items: center; gap: 8px; color: var(--text-muted); font-size: 0.82rem; }
.pager-meta select { background: var(--glass); border: 1px solid var(--glass-border); color: var(--text-main); border-radius: 6px; padding: 4px 6px; }
.pager-page { min-width: 92px; text-align: center; }
.animate-spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
@media (max-width: 900px) {
    .toolbar { flex-direction: column; align-items: stretch; }
    .view-toggle { width: 100%; justify-content: space-between; }
    .view-toggle-btn { flex: 1; justify-content: center; }
    .card-grid { grid-template-columns: 1fr; }
}
</style>
