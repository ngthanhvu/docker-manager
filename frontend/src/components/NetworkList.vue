<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue';
import { Network, Trash2, RefreshCw, BrushCleaning, List, LayoutGrid } from 'lucide-vue-next';
import { useI18n } from 'vue-i18n';
import { dockerApi } from '../api';
import { feedback } from '../ui/feedback';
import { appSettings } from '../ui/settings';
import { loadStoredNumber, persistStoredValue } from '../ui/viewState';

const networks = ref<any[]>([]);
const loading = ref(true);
const currentPage = ref(1);
const NETWORK_PAGE_SIZE_KEY = 'dock-manager.networks.page-size';
const NETWORK_VIEW_MODE_KEY = 'dock-manager.networks.view-mode';
const pageSize = ref(loadStoredNumber(NETWORK_PAGE_SIZE_KEY, 10, 10, 50));
const viewMode = ref<'list' | 'card'>(localStorage.getItem(NETWORK_VIEW_MODE_KEY) === 'card' ? 'card' : 'list');
const pageSizeOptions = [10, 20, 50];
const selectedIds = ref<string[]>([]);
const pruning = ref(false);
const { t } = useI18n();

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
        title: t('networksView.deleteTitle'),
        message: t('networksView.deleteMessage'),
        confirmText: t('common.delete'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'DELETE' : undefined,
    });
    if (!accepted) return;
    try {
        await dockerApi.removeNetwork(id);
        selectedIds.value = selectedIds.value.filter((x) => x !== id);
        await fetchNetworks();
        feedback.success(t('networksView.deletedSuccess'));
    } catch (err) {
        feedback.error(t('networksView.deleteFailed', { error: String(err) }));
    }
};

const bulkDelete = async () => {
    if (selectedIds.value.length === 0) return;
    const removeCount = selectedIds.value.length;
    const accepted = await feedback.confirmAction({
        title: t('networksView.deleteManyTitle'),
        message: t('networksView.deleteManyMessage', { count: removeCount }),
        confirmText: t('common.delete'),
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
        feedback.success(t('networksView.deletedManySuccess', { count: removeCount }));
    } catch (err) {
        feedback.error(t('networksView.bulkDeleteFailed', { error: String(err) }));
    }
};

const pruneNetworks = async () => {
    if (pruning.value) return;
    const accepted = await feedback.confirmAction({
        title: t('networksView.pruneTitle'),
        message: t('networksView.pruneMessage'),
        confirmText: t('common.prune'),
        danger: true,
        requireText: appSettings.safety.softDeleteRequireTyping ? 'PRUNE' : undefined,
    });
    if (!accepted) return;
    try {
        pruning.value = true;
        const { data } = await dockerApi.pruneNetworks();
        await fetchNetworks();
        const deletedCount = Array.isArray(data?.NetworksDeleted) ? data.NetworksDeleted.length : 0;
        feedback.success(t('networksView.prunedSuccess', { count: deletedCount }));
    } catch (err) {
        feedback.error(t('networksView.pruneFailed', { error: String(err) }));
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
watch(viewMode, () => {
    persistStoredValue(NETWORK_VIEW_MODE_KEY, viewMode.value);
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
                <h2>{{ t('networksView.title') }}</h2>
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
                <button class="btn btn-ghost text-danger" :disabled="selectedCount === 0 || pruning" @click="bulkDelete">
                    <Trash2 :size="16" />
                    {{ t('common.delete') }} ({{ selectedCount }})
                </button>
                <button class="btn btn-ghost text-warning" :disabled="pruning" @click="pruneNetworks">
                    <RefreshCw v-if="pruning" :size="16" class="animate-spin" />
                    <BrushCleaning v-else :size="16" />
                    {{ t('common.prune') }}
                </button>
                <button class="btn btn-ghost" :disabled="pruning" @click="fetchNetworks">
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
                        <th>{{ t('networksView.name') }}</th>
                        <th>ID</th>
                        <th>{{ t('networksView.driver') }}</th>
                        <th>{{ t('networksView.scope') }}</th>
                        <th>{{ t('networksView.internal') }}</th>
                        <th class="actions-cell">{{ t('common.actions') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="net in paginatedNetworks" :key="net.Id">
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(net.Id)" @change="toggleSelect(net.Id)" /></td>
                        <td class="name-cell">{{ net.Name }}</td>
                        <td><code>{{ net.Id.substring(0, 12) }}</code></td>
                        <td>{{ net.Driver }}</td>
                        <td>{{ net.Scope }}</td>
                        <td>{{ net.Internal ? t('common.yes') : t('common.no') }}</td>
                        <td class="actions-cell">
                            <div class="action-group">
                                <button class="action-btn action-danger" :title="t('common.remove')" @click="removeNetwork(net.Id)">
                                    <Trash2 :size="16" />
                                </button>
                            </div>
                        </td>
                    </tr>
                    <tr v-if="networks.length === 0 && !loading">
                        <td colspan="7" class="empty-state">{{ t('networksView.noItems') }}</td>
                    </tr>
                </tbody>
            </table>
        </div>

        <div v-else class="card-container">
            <div v-if="networks.length === 0 && !loading" class="glass-panel card-empty-state">
                {{ t('networksView.noItems') }}
            </div>
            <div v-else class="card-grid">
                <article v-for="net in paginatedNetworks" :key="net.Id" class="glass-panel entity-card">
                    <div class="entity-card-header">
                        <label class="card-check">
                            <input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(net.Id)"
                                @change="toggleSelect(net.Id)" />
                        </label>
                        <div class="card-title-block">
                            <div class="name-cell">{{ net.Name }}</div>
                            <code class="card-id">{{ net.Id.substring(0, 12) }}</code>
                        </div>
                    </div>
                    <div class="card-meta-list">
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('networksView.driver') }}</span>
                            <span>{{ net.Driver }}</span>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('networksView.scope') }}</span>
                            <span>{{ net.Scope }}</span>
                        </div>
                        <div class="card-meta-item">
                            <span class="card-meta-label">{{ t('networksView.internal') }}</span>
                            <span>{{ net.Internal ? t('common.yes') : t('common.no') }}</span>
                        </div>
                    </div>
                    <div class="action-group card-action-group">
                        <button class="action-btn action-danger" :title="t('common.remove')" @click="removeNetwork(net.Id)">
                            <Trash2 :size="16" />
                        </button>
                    </div>
                </article>
            </div>
        </div>

        <div v-if="networks.length > 0" class="pagination glass-panel">
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
.network-list-view { display: flex; flex-direction: column; gap: 24px; }
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
.card-id { color: var(--text-muted); }
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
.name-cell { font-weight: 600; }
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
