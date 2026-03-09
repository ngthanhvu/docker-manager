<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { Box, Trash2, RefreshCw } from 'lucide-vue-next';
import { dockerApi } from '../api';
import dayjs from 'dayjs';

const images = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const pageSize = ref(10);
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);

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
    if (!confirm('Are you sure you want to remove this image?')) return;
    try {
        await dockerApi.removeImage(id);
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
        await fetchImages();
    } catch (err) {
        alert(`Failed to remove image: ${err}`);
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    if (!confirm(`Remove ${selectedIds.value.length} selected image(s)?`)) return;
    try {
        for (const id of selectedIds.value) {
            await dockerApi.removeImage(id);
        }
        selectedIds.value = [];
        await fetchImages();
    } catch (err) {
        alert(`Bulk delete failed: ${err}`);
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
                    Bulk Delete ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost" @click="fetchImages">
                    <RefreshCw :size="18" :class="{ 'animate-spin': loading }" />
                    Refresh
                </button>
            </div>
        </div>

        <div class="table-container glass-panel">
            <table class="docker-table">
                <thead>
                    <tr>
                        <th class="check-col"><input type="checkbox" :checked="allPageSelected" @change="toggleSelectAllPage" /></th>
                        <th>Repository:Tag</th>
                        <th>ID</th>
                        <th>Size</th>
                        <th>Created</th>
                        <th class="actions-cell">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="image in paginatedImages" :key="image.Id">
                        <td class="check-col"><input type="checkbox" :checked="selectedIds.includes(image.Id)" @change="toggleSelect(image.Id)" /></td>
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
.check-col { width: 40px; text-align: center !important; padding: 10px !important; }
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
