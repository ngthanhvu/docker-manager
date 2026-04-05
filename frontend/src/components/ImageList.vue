<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { Box, Trash2, RefreshCw, BrushCleaning, List, LayoutGrid, Ellipsis } from 'lucide-vue-next';
import { useI18n } from 'vue-i18n';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';
import dayjs from 'dayjs';

const images = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const IMAGE_PAGE_SIZE_KEY = 'dock-manager.images.page-size';
const IMAGE_VIEW_MODE_KEY = 'dock-manager.images.view-mode';
const pageSize = ref(loadStoredNumber(IMAGE_PAGE_SIZE_KEY, 10, 10, 50));
const viewMode = ref<'list' | 'card'>(localStorage.getItem(IMAGE_VIEW_MODE_KEY) === 'card' ? 'card' : 'list');
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);
const pruning = ref(false);
const activeCardMenuId = ref<string | null>(null);
const sortKey = ref<'repoTag' | 'id' | 'size' | 'created'>('created');
const sortDirection = ref<'asc' | 'desc'>('desc');
const { t } = useI18n();

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
        title: t('imagesView.deleteTitle'),
        message: t('imagesView.deleteMessage'),
        confirmText: t('common.delete'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        await dockerApi.removeImage(id);
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
        await fetchImages();
        feedback.success(t('imagesView.deletedSuccess'));
    } catch (err) {
        feedback.error(t('imagesView.deleteFailed', { error: String(err) }));
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    const removeCount = selectedIds.value.length;
    const accepted = await feedback.confirmAction({
        title: t('imagesView.deleteManyTitle'),
        message: t('imagesView.deleteManyMessage', { count: removeCount }),
        confirmText: t('common.delete'),
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
        feedback.success(t('imagesView.deletedManySuccess', { count: removeCount }));
    } catch (err) {
        feedback.error(t('imagesView.bulkDeleteFailed', { error: String(err) }));
    }
};

const pruneImages = async () => {
    if (pruning.value) return;
    const accepted = await feedback.confirmAction({
        title: t('imagesView.pruneTitle'),
        message: t('imagesView.pruneMessage'),
        confirmText: t('common.prune'),
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
        feedback.success(t('imagesView.prunedSuccess', { count: deletedCount, size: reclaimed }));
    } catch (err) {
        feedback.error(t('imagesView.pruneFailed', { error: String(err) }));
    } finally {
        pruning.value = false;
    }
};

const formatSize = (bytes: number) => `${(bytes / 1024 / 1024).toFixed(1)} MB`;

const compareValues = (left: string | number, right: string | number) => {
    if (typeof left === 'number' && typeof right === 'number') return left - right;
    return String(left).localeCompare(String(right), undefined, { numeric: true, sensitivity: 'base' });
};

const getImageSortValue = (image: any) => {
    if (sortKey.value === 'repoTag') return image.RepoTags?.[0] || '<none>:<none>';
    if (sortKey.value === 'id') return image.Id.substring(7, 19);
    if (sortKey.value === 'size') return Number(image.Size || 0);
    return Number(image.Created || 0);
};

const sortedImages = computed(() => {
    const list = [...images.value];
    list.sort((a, b) => {
        const result = compareValues(getImageSortValue(a), getImageSortValue(b));
        return sortDirection.value === 'asc' ? result : -result;
    });
    return list;
});

const totalItems = computed(() => sortedImages.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
const paginatedImages = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    return sortedImages.value.slice(start, start + pageSize.value);
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

const toggleCardMenu = (id: string) => {
    activeCardMenuId.value = activeCardMenuId.value === id ? null : id;
};

const closeCardMenu = () => {
    activeCardMenuId.value = null;
};

const handleDocumentClick = (event: MouseEvent) => {
    const target = event.target as HTMLElement | null;
    if (target?.closest('.card-actions-menu')) return;
    closeCardMenu();
};

const toggleSort = (key: 'repoTag' | 'id' | 'size' | 'created') => {
    if (sortKey.value === key) {
        sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc';
        return;
    }
    sortKey.value = key;
    sortDirection.value = key === 'created' ? 'desc' : 'asc';
};

const getSortIndicator = (key: 'repoTag' | 'id' | 'size' | 'created') => {
    if (sortKey.value !== key) return '↕';
    return sortDirection.value === 'asc' ? '↑' : '↓';
};

watch(pageSize, () => {
    currentPage.value = 1;
    persistStoredValue(IMAGE_PAGE_SIZE_KEY, pageSize.value);
});
watch(viewMode, () => {
    persistStoredValue(IMAGE_VIEW_MODE_KEY, viewMode.value);
});
watch(totalPages, (maxPage) => {
    if (currentPage.value > maxPage) currentPage.value = maxPage;
});
watch(images, (list) => {
    const valid = new Set(list.map((img) => img.Id));
    selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
});

onMounted(() => {
    fetchImages();
    document.addEventListener('click', handleDocumentClick);
});

onUnmounted(() => {
    document.removeEventListener('click', handleDocumentClick);
});
</script>

<template>
    <div class="image-list-view">
        <div class="toolbar glass-panel">
            <div class="title-with-icon">
                <Box :size="20" class="icon-indigo" />
                <h2>{{ t('imagesView.title') }}</h2>
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
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0" @click="bulkDelete">
                    <Trash2 :size="16" />
                    {{ t('common.delete') }} ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost text-warning" :disabled="pruning" @click="pruneImages">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    {{ t('common.prune') }}
                </button>
                <button class="btn btn-ghost" :disabled="pruning" @click="fetchImages">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading || pruning }" />
                    {{ t('common.refresh') }}
                </button>
            </div>
        </div>

        <div v-if="viewMode === 'list'" class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="allPageSelected" @change="toggleSelectAllPage" /></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('repoTag')">{{ t('imagesView.repositoryTag') }}<span class="sort-indicator">{{ getSortIndicator('repoTag') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('id')">ID<span class="sort-indicator">{{ getSortIndicator('id') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('size')">{{ t('imagesView.size') }}<span class="sort-indicator">{{ getSortIndicator('size') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('created')">{{ t('imagesView.created') }}<span class="sort-indicator">{{ getSortIndicator('created') }}</span></button></th>
                        <th class="actions-cell">{{ t('common.actions') }}</th>
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
                            <div class="action-group">
                                <button class="action-btn action-danger" :title="t('common.remove')" @click="removeImage(image.Id)">
                                    <Trash2 :size="16" />
                                </button>
                            </div>
                        </td>
                    </tr>
                    <tr v-if="images.length === 0 && !loading">
                        <td colspan="6" class="empty-state">{{ t('imagesView.noItems') }}</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-else class="card-container">
            <div v-if="images.length === 0 && !loading" class="glass-panel card-empty-state">
                {{ t('imagesView.noItems') }}
            </div>
            <div v-else class="card-grid">
                <article v-for="image in paginatedImages" :key="image.Id" class="glass-panel entity-card">
                    <div class="entity-card-header">
                        <label class="card-check">
                            <input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(image.Id)"
                                @change="toggleSelect(image.Id)" />
                        </label>
                        <div class="card-title-block">
                            <div class="name-cell">{{ image.RepoTags?.[0] || '&lt;none&gt;:&lt;none&gt;' }}</div>
                            <code class="card-id">{{ image.Id.substring(7, 19) }}</code>
                        </div>
                    </div>
                    <div class="card-meta-list">
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('imagesView.size') }}</span>
                            <span>{{ formatSize(image.Size) }}</span>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('imagesView.created') }}</span>
                            <span>{{ dayjs.unix(image.Created).format('YYYY-MM-DD HH:mm') }}</span>
                        </div>
                    </div>
                    <div class="card-actions-menu">
                        <button class="action-btn action-neutral card-menu-trigger" type="button"
                            :title="t('common.actions')" @click.stop="toggleCardMenu(image.Id)">
                            <Ellipsis :size="16" />
                        </button>
                        <div v-if="activeCardMenuId === image.Id" class="card-actions-popover glass-panel" @click.stop>
                            <button class="card-action-item action-danger" type="button" @click="removeImage(image.Id)">
                                <Trash2 :size="16" />
                                {{ t('common.remove') }}
                            </button>
                        </div>
                    </div>
                </article>
            </div>
        </div>

        <div v-if="images.length > 0" class="pagination glass-panel">
            <div class="pager-meta">
                <span>{{ t('common.rows') }}</span>
                <select v-model.number="pageSize">
                    <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }}</option>
                </select>
                <span>{{ pageStart }}-{{ pageEnd }} / {{ totalItems }}</span>
            </div>
            <div class="pager-actions">
                <button class="btn btn-ghost" :disabled="currentPage === 1" @click="currentPage--">{{ t('common.prev') }}</button>
                <span class="pager-page">{{ t('common.page') }} {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost" :disabled="currentPage >= totalPages" @click="currentPage++">{{ t('common.next') }}</button>
            </div>
        </div>
    </div>
</template>

<style scoped>
.image-list-view { display: flex; flex-direction: column; gap: 24px; }
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
.entity-card { position: relative; display: flex; flex-direction: column; gap: 16px; padding: 18px; }
.entity-card-header { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: flex-start; gap: 12px; }
.card-check { display: inline-flex; align-items: center; justify-content: center; padding-top: 2px; }
.card-title-block { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.card-id { color: var(--text-muted); }
.card-meta-list { display: flex; flex-direction: column; gap: 12px; }
.card-meta-item { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.card-meta-label { font-size: 0.76rem; letter-spacing: 0.04em; text-transform: uppercase; color: var(--text-muted); }
.docker-table { width: 100%; border-collapse: collapse; }
.docker-table th { text-align: left; padding: 14px 20px; font-size: 0.86rem; color: var(--text-muted); border-bottom: 1px solid var(--glass-border); }
.docker-table td { padding: 14px 20px; font-size: 0.88rem; border-bottom: 1px solid var(--glass-border); }
.sort-header { display: inline-flex; align-items: center; gap: 6px; padding: 0; border: none; background: transparent; color: inherit; font: inherit; cursor: pointer; }
.sort-indicator { font-size: 0.8em; color: var(--text-muted); }
.check-col { width: 56px; text-align: center !important; padding: 10px !important; }
.bulk-checkbox { width: 22px; height: 22px; cursor: pointer; accent-color: var(--primary); border-radius: 6px; }
.bulk-checkbox:hover { filter: brightness(1.08); }
.bulk-checkbox:focus-visible { outline: 2px solid rgba(36, 150, 237, 0.55); outline-offset: 2px; }
.docker-table tr:last-child td { border-bottom: none; }
.docker-table tr:hover { background: var(--glass); }
.name-cell { font-weight: 600; word-break: break-all; }
.actions-cell { width: 100px; text-align: center; }
.action-group { display: flex; align-items: center; justify-content: center; }
.card-actions-menu { position: absolute; top: 18px; right: 18px; }
.card-menu-trigger { background: rgba(255, 255, 255, 0.05); }
.card-actions-popover { position: absolute; top: calc(100% + 8px); right: 0; display: flex; flex-direction: column; gap: 6px; min-width: 160px; padding: 8px; z-index: 5; }
.card-action-item { display: inline-flex; align-items: center; gap: 10px; width: 100%; min-height: 36px; padding: 0 12px; border-radius: 10px; border: 1px solid var(--glass-border); background: rgba(255, 255, 255, 0.03); color: var(--text-main); cursor: pointer; transition: all 0.18s ease; }
.card-action-item:hover { transform: translateY(-1px); }
.card-action-item:disabled { opacity: 0.45; cursor: not-allowed; transform: none; }
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
@media (max-width: 900px) {
    .toolbar { flex-direction: column; align-items: stretch; }
    .view-toggle { width: 100%; justify-content: space-between; }
    .view-toggle-btn { flex: 1; justify-content: center; }
    .card-grid { grid-template-columns: 1fr; }
}
.animate-spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
