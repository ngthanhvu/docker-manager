<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { HardDrive, Trash2, RefreshCw } from 'lucide-vue-next';
import { dockerApi } from '../api';
import dayjs from 'dayjs';

const volumes = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const pageSize = ref(10);
const pageSizeOptions = [10, 20, 50];
const selectedNames = ref<string[]>([]);

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
    if (!confirm('Are you sure you want to remove this volume?')) return;
    try {
        await dockerApi.removeVolume(name);
        selectedNames.value = selectedNames.value.filter((x) => x !== name);
        await fetchVolumes();
    } catch (err) {
        alert(`Failed to remove volume: ${err}`);
    }
};

const bulkDelete = async () => {
    if (selectedNames.value.length === 0) return;
    if (!confirm(`Remove ${selectedNames.value.length} selected volume(s)?`)) return;
    try {
        for (const name of selectedNames.value) {
            await dockerApi.removeVolume(name);
        }
        selectedNames.value = [];
        await fetchVolumes();
    } catch (err) {
        alert(`Bulk delete failed: ${err}`);
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

watch(pageSize, () => { currentPage.value = 1; });
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
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0" @click="bulkDelete">
                    <Trash2 :size="16" />
                    Bulk Delete ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost" @click="fetchVolumes">
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
                        <th>Name</th>
                        <th>Driver</th>
                        <th>Mountpoint</th>
                        <th>Created</th>
                        <th class="actions-cell">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="vol in paginatedVolumes" :key="vol.Name">
                        <td class="check-col"><input type="checkbox" :checked="selectedNames.includes(vol.Name)" @change="toggleSelect(vol.Name)" /></td>
                        <td class="name-cell">{{ vol.Name }}</td>
                        <td>{{ vol.Driver }}</td>
                        <td><code>{{ vol.Mountpoint || '-' }}</code></td>
                        <td>{{ vol.CreatedAt ? dayjs(vol.CreatedAt).format('YYYY-MM-DD HH:mm') : '-' }}</td>
                        <td class="actions-cell">
                            <button class="btn-icon btn-ghost text-danger" title="Remove" @click="removeVolume(vol.Name)">
                                <Trash2 :size="16" />
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
