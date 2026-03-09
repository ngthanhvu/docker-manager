<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { Network, Trash2, RefreshCw } from 'lucide-vue-next';
import { dockerApi } from '../api';

const networks = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const pageSize = ref(10);
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);

const fetchNetworks = async () => {
    try {
        loading.value = true;
        const { data } = await dockerApi.getNetworks();
        networks.value = data || [];
    } catch (err) {
        console.error('Failed to fetch networks:', err);
    } finally {
        loading.value = false;
    }
};

const removeNetwork = async (id: string) => {
    if (!confirm('Are you sure you want to remove this network?')) return;
    try {
        await dockerApi.removeNetwork(id);
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
        await fetchNetworks();
    } catch (err) {
        alert(`Failed to remove network: ${err}`);
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    if (!confirm(`Remove ${selectedIds.value.length} selected network(s)?`)) return;
    try {
        for (const id of selectedIds.value) {
            await dockerApi.removeNetwork(id);
        }
        selectedIds.value = [];
        await fetchNetworks();
    } catch (err) {
        alert(`Bulk delete failed: ${err}`);
    }
};

const totalItems = computed(() => networks.value.length);
const totalPages = computed(() => Math.max(1, Math.ceil(totalItems.value / pageSize.value)));
const paginatedNetworks = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    return networks.value.slice(start, start + pageSize.value);
});
const pageStart = computed(() => (totalItems.value === 0 ? 0 : (currentPage.value - 1) * pageSize.value + 1));
const pageEnd = computed(() => Math.min(currentPage.value * pageSize.value, totalItems.value));

const pageNetworkIds = computed(() => paginatedNetworks.value.map((n) => n.Id));
const selectedCount = computed(() => selectedIds.value.length);
const allPageSelected = computed(() => pageNetworkIds.value.length > 0 && pageNetworkIds.value.every((id) => selectedIds.value.includes(id)));

const toggleSelect = (id: string) => {
    if (selectedIds.value.includes(id)) selectedIds.value = selectedIds.value.filter((x) => x !== id);
    else selectedIds.value = [...selectedIds.value, id];
};

const toggleSelectAllPage = () => {
    if (allPageSelected.value) selectedIds.value = selectedIds.value.filter((id) => !pageNetworkIds.value.includes(id));
    else selectedIds.value = Array.from(new Set([...selectedIds.value, ...pageNetworkIds.value]));
};

watch(pageSize, () => {
    currentPage.value = 1;
});
watch(totalPages, (maxPage) => {
    if (currentPage.value > maxPage) currentPage.value = maxPage;
});
watch(networks, (list) => {
    const valid = new Set(list.map((n) => n.Id));
    selectedIds.value = selectedIds.value.filter((id) => valid.has(id));
});

onMounted(fetchNetworks);
</script>

<template>
    <div class="network-list-view">
        <div class="toolbar glass-panel">
            <div class="title-with-icon">
                <Network :size="20" class="icon-indigo" />
                <h2>Networks</h2>
            </div>
            <div class="toolbar-actions">
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0" @click="bulkDelete">
                    <Trash2 :size="16" />
                    Bulk Delete ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost" @click="fetchNetworks">
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
                        <th>ID</th>
                        <th>Driver</th>
                        <th>Scope</th>
                        <th>Internal</th>
                        <th class="actions-cell">Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="net in paginatedNetworks" :key="net.Id">
                        <td class="check-col"><input type="checkbox" :checked="selectedIds.includes(net.Id)" @change="toggleSelect(net.Id)" /></td>
                        <td class="name-cell">{{ net.Name }}</td>
                        <td><code>{{ net.Id.substring(0, 12) }}</code></td>
                        <td>{{ net.Driver }}</td>
                        <td>{{ net.Scope }}</td>
                        <td>{{ net.Internal ? 'Yes' : 'No' }}</td>
                        <td class="actions-cell">
                            <button class="btn-icon btn-ghost text-danger" title="Remove" @click="removeNetwork(net.Id)">
                                <Trash2 :size="16" />
                            </button>
                        </td>
                    </tr>
                    <tr v-if="networks.length === 0 && !loading">
                        <td colspan="7" class="empty-state">No networks found</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-if="networks.length > 0" class="pagination glass-panel">
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
.network-list-view { display: flex; flex-direction: column; gap: 24px; }
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
.name-cell { font-weight: 600; }
.actions-cell { text-align: right; width: 100px; }
.empty-state { text-align: center; color: var(--text-muted); padding: 56px 0; }
.pagination { padding: 10px 14px; display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.pager-meta, .pager-actions { display: flex; align-items: center; gap: 8px; color: var(--text-muted); font-size: 0.82rem; }
.pager-meta select { background: var(--glass); border: 1px solid var(--glass-border); color: var(--text-main); border-radius: 6px; padding: 4px 6px; }
.pager-page { min-width: 92px; text-align: center; }
.animate-spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
