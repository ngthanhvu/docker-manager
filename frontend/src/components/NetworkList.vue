<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { Network, Trash2, RefreshCw, BrushCleaning } from 'lucide-vue-next';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';

const networks = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const NETWORK_PAGE_SIZE_KEY = 'dock-manager.networks.page-size';
const pageSize = ref(loadStoredNumber(NETWORK_PAGE_SIZE_KEY, 10, 10, 50));
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);
const pruning = ref(false);

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
    const accepted = await feedback.confirmAction({
        title: 'Delete Network',
        message: 'Are you sure you want to remove this network?',
        confirmText: 'Delete',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        await dockerApi.removeNetwork(id);
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
        await fetchNetworks();
        feedback.success('Network removed successfully.');
    } catch (err) {
        feedback.error(`Failed to remove network: ${err}`);
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    const removeCount = selectedIds.value.length;
    const accepted = await feedback.confirmAction({
        title: 'Delete Networks',
        message: `Remove ${removeCount} selected network(s)? This action cannot be undone.`,
        confirmText: 'Delete',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        for (const id of selectedIds.value) {
            await dockerApi.removeNetwork(id);
        }
        selectedIds.value = [];
        await fetchNetworks();
        feedback.success(`Deleted ${removeCount} network(s) successfully.`);
    } catch (err) {
        feedback.error(`Bulk delete failed: ${err}`);
    }
};

const pruneNetworks = async () => {
    if (pruning.value) return;
    const accepted = await feedback.confirmAction({
        title: 'Prune Networks',
        message: 'Remove all unused custom networks? Active and default networks will be kept.',
        confirmText: 'Prune',
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;
    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneNetworks();
        await fetchNetworks();
        const deletedCount = Array.isArray(data?.NetworksDeleted) ? data.NetworksDeleted.length : 0;
        feedback.success(`Pruned ${deletedCount} unused network(s).`);
    } catch (err) {
        feedback.error(`Network prune failed: ${err}`);
    } finally {
        pruning.value = false;
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
    persistStoredValue(NETWORK_PAGE_SIZE_KEY, pageSize.value);
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
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0 || pruning" @click="bulkDelete">
                    <Trash2 :size="16" />
                    Delete ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost text-warning" :disabled="pruning" @click="pruneNetworks">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    Prune
                </button>
                <button class="btn btn-ghost" :disabled="pruning" @click="fetchNetworks">
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
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(net.Id)" @change="toggleSelect(net.Id)" /></td>
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
.check-col { width: 56px; text-align: center !important; padding: 10px !important; }
.bulk-checkbox { width: 22px; height: 22px; cursor: pointer; accent-color: var(--primary); border-radius: 6px; }
.bulk-checkbox:hover { filter: brightness(1.08); }
.bulk-checkbox:focus-visible { outline: 2px solid rgba(36, 150, 237, 0.55); outline-offset: 2px; }
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
