<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { HardDrive, Trash2, RefreshCw, BrushCleaning } from 'lucide-vue-next';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';
import dayjs from 'dayjs';

const volumes = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const VOLUME_PAGE_SIZE_KEY = 'dock-manager.volumes.page-size';
const pageSize = ref(loadStoredNumber(VOLUME_PAGE_SIZE_KEY, 10, 10, 50));
const pageSizeOptions = [10, 20, 50];
const selectedNames = ref<string[]>([]);
const deletingName = ref<string | null>(null);
const bulkDeleting = ref(false);
const pruning = ref(false);

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
    return 'Unknown error';
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
        title: 'Delete Volume',
        message: 'Are you sure you want to remove this volume?',
        confirmText: 'Delete',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        deletingName.value = name;
        await dockerApi.removeVolume(name);
        selectedNames.value = selectedNames.value.filter((x) => x !== name);
        await fetchVolumes();
        feedback.success('Volume removed successfully.');
    } catch (err) {
        feedback.error(`Failed to remove volume: ${err}`);
    } finally {
        deletingName.value = null;
    }
};

const bulkDelete = async () => {
    if (selectedNames.value.length === 0 || bulkDeleting.value || deletingName.value) return;
    const namesToDelete = [...selectedNames.value];
    const removeCount = namesToDelete.length;
    const accepted = await feedback.confirmAction({
        title: 'Delete Volumes',
        message: `Remove ${removeCount} selected volume(s)? This action cannot be undone.`,
        confirmText: 'Delete',
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
            feedback.success(`Deleted ${deleted.length}/${removeCount} volume(s).`);
        }

        if (failed.length > 0) {
            const failedNames = failed.map((item) => item.name).join(', ');
            const usedVolumes = failed.filter((item) => item.reason.toLowerCase().includes('in use'));
            const message = usedVolumes.length === failed.length
                ? `Could not delete ${failed.length} in-use volume(s): ${failedNames}`
                : `Could not delete ${failed.length} volume(s): ${failedNames}`;
            feedback.warning(message);
        }
    } finally {
        bulkDeleting.value = false;
    }
};

const pruneVolumes = async () => {
    if (pruning.value || bulkDeleting.value || deletingName.value) return;
    const accepted = await feedback.confirmAction({
        title: 'Prune Volumes',
        message: 'Remove all unused volumes? Volumes still attached to containers will be kept.',
        confirmText: 'Prune',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;

    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneVolumes();
        await fetchVolumes();
        const deletedCount = Array.isArray(data?.VolumesDeleted) ? data.VolumesDeleted.length : 0;
        feedback.success(`Pruned ${deletedCount} unused volume(s).`);
    } catch (err) {
        feedback.error(`Volume prune failed: ${getErrorMessage(err)}`);
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
                <h2>Volumes</h2>
            </div>
            <div class="toolbar-actions">
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0 || bulkDeleting || !!deletingName || pruning" @click="bulkDelete">
                    <RefreshCw v-if="bulkDeleting" :size="16" class="animate-spin" />
                    <Trash2 v-else :size="16" />
                    {{ bulkDeleting ? `Deleting (${selectedCount})...` : `Delete (${selectedCount})` }}
                </button>
                <button class="btn btn-ghost text-warning" :disabled="bulkDeleting || !!deletingName || pruning" @click="pruneVolumes">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    Prune
                </button>
                <button class="btn btn-ghost" :disabled="bulkDeleting || !!deletingName || pruning" @click="fetchVolumes">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                    Refresh
                </button>
            </div>
        </div>

        <div class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="allPageSelected" :disabled="bulkDeleting || !!deletingName || pruning" @change="toggleSelectAllPage" /></th>
                        <th>Name</th>
                        <th>Driver</th>
                        <th>Mountpoint</th>
                        <th>Created</th>
                        <th class="actions-cell">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="vol in paginatedVolumes" :key="vol.Name">
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="selectedNames.includes(vol.Name)" :disabled="bulkDeleting || !!deletingName || pruning" @change="toggleSelect(vol.Name)" /></td>
                        <td class="name-cell">{{ vol.Name }}</td>
                        <td>{{ vol.Driver }}</td>
                        <td><code>{{ vol.Mountpoint || '-' }}</code></td>
                        <td>{{ vol.CreatedAt ? dayjs(vol.CreatedAt).format('YYYY-MM-DD HH:mm') : '-' }}</td>
                        <td class="actions-cell">
                            <button class="btn-icon btn-ghost text-danger" :disabled="bulkDeleting || !!deletingName || pruning" title="Remove" @click="removeVolume(vol.Name)">
                                <RefreshCw v-if="deletingName === vol.Name" :size="16" class="animate-spin" />
                                <Trash2 v-else :size="16" />
                            </button>
                        </td>
                    </tr>
                    <tr v-if="volumes.length === 0 && !loading">
                        <td colspan="6" class="empty-state">No volumes found</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-if="volumes.length > 0" class="pagination glass-panel">
            <div class="pager-meta">
                <span>Rows</span>
                <select v-model.number="pageSize" :disabled="bulkDeleting || !!deletingName || pruning">
                    <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select>
                <span>{{ pageStart }}-{{ pageEnd }} / {{ totalItems }}</span>
            </div>
            <div class="pager-actions">
                <button class="btn btn-ghost" :disabled="currentPage === 1 || bulkDeleting || !!deletingName || pruning" @click="currentPage--">Prev</button>
                <span class="pager-page">Page {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost" :disabled="currentPage >= totalPages || bulkDeleting || !!deletingName || pruning" @click="currentPage++">Next</button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.volume-list-view { display: flex; flex-direction: column; gap: 24px; }
.toolbar { padding: 12px 24px; display: flex; justify-content: space-between; align-items: center; }
.toolbar-actions { display: flex; align-items: center; gap: 8px; }
.title-with-icon { display: flex; align-items: center; gap: 12px; }
.title-with-icon h2 { font-size: 1.2rem; margin: 0; }
.icon-indigo { color: var(--primary); }
.table-container { overflow: hidden; }
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
.actions-cell { text-align: right; width: 100px; }
.empty-state { text-align: center; color: var(--text-muted); padding: 56px 0; }
.pagination { padding: 10px 14px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.pager-meta, .pager-actions { display: flex; align-items: center; gap: 8px; color: var(--text-muted); font-size: 0.82rem; }
.pager-meta select { background: var(--glass); border: 1px solid var(--glass-border); color: var(--text-main); border-radius: 6px; padding: 4px 6px; }
.pager-page { min-width: 92px; text-align: center; }
.animate-spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
