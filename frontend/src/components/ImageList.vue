<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { Box, Trash2, RefreshCw } from 'lucide-vue-next';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import dayjs from 'dayjs';

const images = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const pageSize = ref(10);
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);
const pruning = ref(false);

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

const fetchImages = async () => {
    try {
        loading.value = true;
        const { data } = await dockerApi.getImages();
        images.value = data || [];
    } catch (err) {
        console.error('Failed to fetch images:', err);
    } finally {
        loading.value = false;
    }
};

const removeImage = async (id: string) => {
    const accepted = await feedback.confirmAction({
        title: 'Delete Image',
        message: 'Are you sure you want to remove this image?',
        confirmText: 'Delete',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        await dockerApi.removeImage(id);
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
        await fetchImages();
        feedback.success('Image removed successfully.');
    } catch (err) {
        feedback.error(`Failed to remove image: ${err}`);
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    const removeCount = selectedIds.value.length;
    const accepted = await feedback.confirmAction({
        title: 'Delete Images',
        message: `Remove ${removeCount} selected image(s)? This action cannot be undone.`,
        confirmText: 'Delete',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        for (const id of selectedIds.value) {
            await dockerApi.removeImage(id);
        }
        selectedIds.value = [];
        await fetchImages();
        feedback.success(`Deleted ${removeCount} image(s) successfully.`);
    } catch (err) {
        feedback.error(`Bulk delete failed: ${err}`);
    }
};

const pruneImages = async () => {
    if (pruning.value) return;
    const accepted = await feedback.confirmAction({
        title: 'Prune Images',
        message: 'Remove all unused images? This also clears dangling layers.',
        confirmText: 'Prune',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;
    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneImages();
        await fetchImages();
        const deletedCount = Array.isArray(data?.ImagesDeleted) ? data.ImagesDeleted.length : 0;
        const reclaimed = formatBytes(Number(data?.SpaceReclaimed || 0));
        feedback.success(`Pruned ${deletedCount} unused image(s), reclaimed ${reclaimed}.`);
    } catch (err) {
        feedback.error(`Image prune failed: ${err}`);
    } finally {
        pruning.value = false;
    }
};

const formatSize = (bytes: number) => `${(bytes / 1024 / 1024).toFixed(1)} MB`;

const totalItems = computed(() => images.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
const paginatedImages = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    return images.value.slice(start, start + pageSize.value);
});
const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));

const pageImageIds = computed(() => paginatedImages.value.map((img) => img.Id));
const selectedCount = computed(() => selectedIds.value.length);
const allPageSelected = computed(() => pageImageIds.value.length > 0 && pageImageIds.value.every((id) => selectedIds.value.includes(id)));

const toggleSelect = (id: string) => {
    if (selectedIds.value.includes(id)) selectedIds.value = selectedIds.value.filter((x) => x !== id);
    else selectedIds.value = [...selectedIds.value, id];
};

const toggleSelectAllPage = () => {
    if (allPageSelected.value) selectedIds.value = selectedIds.value.filter((id) => !pageImageIds.value.includes(id));
    else selectedIds.value = Array.from(new Set([...selectedIds.value, ...pageImageIds.value]));
};

watch(pageSize, () => {
    currentPage.value = 1;
});
watch(totalPages, (maxPage) => {
    if (currentPage.value > maxPage) currentPage.value = maxPage;
});
watch(images, (list) => {
    const valid = new Set(list.map((img) => img.Id));
    selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
});

onMounted(fetchImages);
</script>

<template>
    <div class="image-list-view">
        <div class="toolbar glass-panel">
            <div class="title-with-icon">
                <Box :size="20" class="icon-indigo" />
                <h2>Images</h2>
            </div>
            <div class="toolbar-actions">
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0" @click="bulkDelete">
                    <Trash2 :size="16" />
                    Delete ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost text-danger" :disabled="pruning" @click="pruneImages">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <Trash2 v-else :size="16" />
                    Prune Unused
                </button>
                <button class="btn btn-ghost" :disabled="pruning" @click="fetchImages">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                    Refresh
                </button>
            </div>
        </div>

        <div class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="allPageSelected" @change="toggleSelectAllPage" /></th>
                        <th>Repository:Tag</th>
                        <th>ID</th>
                        <th>Size</th>
                        <th>Created</th>
                        <th class="actions-cell">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="image in paginatedImages" :key="image.Id">
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(image.Id)" @change="toggleSelect(image.Id)" /></td>
                        <td class="name-cell">{{ image.RepoTags?.[0] || '&lt;none&gt;:&lt;none&gt;' }}</td>
                        <td><code>{{ image.Id.substring(7, 19) }}</code></td>
                        <td>{{ formatSize(image.Size) }}</td>
                        <td>{{ dayjs.unix(image.Created).format('YYYY-MM-DD HH:mm') }}</td>
                        <td class="actions-cell">
                            <button class="btn-icon btn-ghost text-danger" title="Remove" @click="removeImage(image.Id)">
                                <Trash2 :size="16" />
                            </button>
                        </td>
                    </tr>
                    <tr v-if="images.length === 0 && !loading">
                        <td colspan="6" class="empty-state">No images found</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-if="images.length > 0" class="pagination glass-panel">
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
    </div>
</template>

<style scoped>
.image-list-view { display: flex; flex-direction: column; gap: 24px; }
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
