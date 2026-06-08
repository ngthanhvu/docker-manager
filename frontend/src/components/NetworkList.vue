<script setup lang="ts">
import { Network, Trash2, RefreshCw, BrushCleaning, List, LayoutGrid, Ellipsis, ChevronLeft, ChevronRight } from 'lucide-vue-next';
import { useNetworkList } from '../composables/useNetworkList';

const {
    t,
    networks,
    loading,
    currentPage,
    pageSize,
    pageSizeOptions,
    viewMode,
    selectedIds,
    pruning,
    activeCardMenuId,
    selectedCount,
    totalItems,
    totalPages,
    paginatedNetworks,
    pageStart,
    pageEnd,
    allPageSelected,
    fetchNetworks,
    removeNetwork,
    bulkDelete,
    pruneNetworks,
    toggleSelect,
    toggleSelectAllPage,
    toggleCardMenu,
    toggleSort,
    getSortIndicator,
} = useNetworkList();
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
                        <th class="name-cell"><button class="sort-header" type="button" @click="toggleSort('name')">{{ t('networksView.name') }}<span class="sort-indicator">{{ getSortIndicator('name') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('id')">ID<span class="sort-indicator">{{ getSortIndicator('id') }}</span></button></th>
                        <th><button class="sort-header" type="button" @click="toggleSort('driver')">{{ t('networksView.driver') }}<span class="sort-indicator">{{ getSortIndicator('driver') }}</span></button></th>
                        <th class="scope-cell"><button class="sort-header" type="button" @click="toggleSort('scope')">{{ t('networksView.scope') }}<span class="sort-indicator">{{ getSortIndicator('scope') }}</span></button></th>
                        <th class="internal-cell"><button class="sort-header" type="button" @click="toggleSort('internal')">{{ t('networksView.internal') }}<span class="sort-indicator">{{ getSortIndicator('internal') }}</span></button></th>
                        <th class="actions-cell">{{ t('common.actions') }}</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-for="net in paginatedNetworks" :key="net.Id">
                        <td class="check-col"><input class="bulk-checkbox" type="checkbox" :checked="selectedIds.includes(net.Id)" @change="toggleSelect(net.Id)" /></td>
                        <td class="name-cell">{{ net.Name }}</td>
                        <td><code>{{ net.Id.substring(0, 12) }}</code></td>
                        <td>{{ net.Driver }}</td>
                        <td class="scope-cell">{{ net.Scope }}</td>
                        <td class="internal-cell">{{ net.Internal ? t('common.yes') : t('common.no') }}</td>
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
                    <div class="card-actions-menu">
                        <button class="action-btn action-neutral card-menu-trigger" type="button"
                            :title="t('common.actions')" @click.stop="toggleCardMenu(net.Id)">
                            <Ellipsis :size="16" />
                        </button>
                        <div v-if="activeCardMenuId === net.Id" class="card-actions-popover glass-panel" @click.stop>
                            <button class="card-action-item action-danger" type="button" @click="removeNetwork(net.Id)">
                                <Trash2 :size="16" />
                                {{ t('common.remove') }}
                            </button>
                        </div>
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
                <button class="btn btn-ghost btn-icon" :disabled="currentPage === 1" :aria-label="t('common.prev')" :title="t('common.prev')" @click="currentPage--">
                    <ChevronLeft :size="16" />
                </button>
                <span class="pager-page">{{ t('common.page') }} {{ currentPage }} / {{ totalPages }}</span>
                <button class="btn btn-ghost btn-icon" :disabled="currentPage >= totalPages" :aria-label="t('common.next')" :title="t('common.next')" @click="currentPage++">
                    <ChevronRight :size="16" />
                </button>
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
.entity-card { position: relative; display: flex; flex-direction: column; gap: 16px; padding: 18px; }
.entity-card-header { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: flex-start; gap: 12px; }
.card-check { display: inline-flex; align-items: center; justify-content: center; padding-top: 2px; }
.card-title-block { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.card-id { color: var(--text-muted); }
.card-meta-list { display: flex; flex-direction: column; gap: 12px; }
.card-meta-item { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.card-meta-label { font-size: 0.76rem; letter-spacing: 0.04em; text-transform: uppercase; color: var(--text-muted); }
.docker-table { width: 100%; border-collapse: collapse; }
.docker-table th { text-align: left; padding: 14px 20px; font-size: 0.86rem; color: var(--text-muted); border-bottom: 1px solid var(--glass-border); vertical-align: middle; }
.docker-table td { padding: 14px 20px; font-size: 0.88rem; border-bottom: 1px solid var(--glass-border); vertical-align: middle; }
.sort-header { display: inline-flex; align-items: center; gap: 6px; padding: 0; border: none; background: transparent; color: inherit; font: inherit; cursor: pointer; }
.sort-indicator { font-size: 0.8em; color: var(--text-muted); }
.check-col { width: 56px; text-align: center !important; padding: 10px !important; }
.bulk-checkbox { width: 22px; height: 22px; cursor: pointer; accent-color: var(--primary); border-radius: 6px; }
.bulk-checkbox:hover { filter: brightness(1.08); }
.bulk-checkbox:focus-visible { outline: 2px solid rgba(36, 150, 237, 0.55); outline-offset: 2px; }
.docker-table tr:last-child td { border-bottom: none; }
.docker-table tr:hover { background: var(--glass); }
.name-cell { font-weight: 600; }
.scope-cell,
.internal-cell { text-align: center; }
th.scope-cell .sort-header,
th.internal-cell .sort-header { justify-content: center; width: 100%; }
.actions-cell { width: 100px; text-align: center; }
.action-group { display: flex; align-items: center; justify-content: center; }
.card-actions-menu { position: absolute; top: 18px; right: 18px; }
.card-menu-trigger { background: rgba(255, 255, 255, 0.05); }
.card-actions-popover { position: absolute; top: calc(100% + 8px); right: 0; display: flex; flex-direction: column; gap: 6px; min-width: 160px; padding: 8px; z-index: 5; }
.card-action-item { display: inline-flex; align-items: center; gap: 10px; width: 100%; min-height: 36px; padding: 0 12px; border-radius: 10px; border: 1px solid var(--glass-border); background: rgba(255, 255, 255, 0.03); color: var(--text-main); cursor: pointer; transition: all 0.18s ease; }
.card-action-item:hover { border-color: color-mix(in srgb, var(--primary) 30%, var(--glass-border)); background: color-mix(in srgb, var(--primary) 8%, var(--glass)); }
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
    border-color: color-mix(in srgb, var(--primary) 30%, var(--glass-border));
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
