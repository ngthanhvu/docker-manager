<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import {
  LayoutDashboard,
  Container,
  Box,
  HardDrive,
  Network,
  Layers,
  Cpu,
  Database,
  Settings
} from 'lucide-vue-next';
import { dockerApi } from '../api';
import { appSettings } from '../ui/settings';
import { persistStoredValue } from '../ui/viewState';
import { authState, signOut } from '../ui/auth';
import Dashboard from '../components/Dashboard.vue';
import ContainerList from '../components/ContainerList.vue';
import ImageList from '../components/ImageList.vue';
import VolumeList from '../components/VolumeList.vue';
import NetworkList from '../components/NetworkList.vue';
import ComposeList from '../components/ComposeList.vue';
import SettingsPanel from '../components/SettingsPanel.vue';

const route = useRoute();
const router = useRouter();
const systemInfo = ref<any>(null);
const resourceCounts = ref<{ volumes: number; networks: number }>({ volumes: 0, networks: 0 });
let statsTimer: number | null = null;

const tabs = [
  { id: 'dashboard', name: 'Dashboard', icon: LayoutDashboard },
  { id: 'containers', name: 'Containers', icon: Container },
  { id: 'images', name: 'Images', icon: Box },
  { id: 'volumes', name: 'Volumes', icon: HardDrive },
  { id: 'networks', name: 'Networks', icon: Network },
  { id: 'compose', name: 'Compose', icon: Layers },
  { id: 'settings', name: 'Settings', icon: Settings },
];

const validTabIds = new Set(tabs.map((tab) => tab.id));

const activeTab = computed(() => {
  const tab = typeof route.params.tab === 'string' ? route.params.tab : 'dashboard';
  return validTabIds.has(tab) ? tab : 'dashboard';
});

const setActiveTab = async (tabId: string) => {
  if (!validTabIds.has(tabId)) return;
  persistStoredValue('dock-manager.active-tab', tabId);
  await router.replace({ name: 'app', params: { tab: tabId } });
};

const handleGlobalShortcut = (event: KeyboardEvent) => {
  if (!authState.user) return;
  if (!event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return;
  const target = event.target as HTMLElement | null;
  const tag = target?.tagName?.toLowerCase();
  if (tag === 'input' || tag === 'textarea' || target?.isContentEditable) return;
  const index = Number(event.key) - 1;
  if (!Number.isInteger(index) || index < 0 || index >= tabs.length) return;
  event.preventDefault();
  void setActiveTab(tabs[index]?.id || 'dashboard');
};

const fetchStats = async () => {
  const [infoRes, volumesRes, networksRes] = await Promise.allSettled([
    dockerApi.getSystemInfo(),
    dockerApi.getVolumes(),
    dockerApi.getNetworks(),
  ]);

  if (infoRes.status === 'fulfilled') {
    systemInfo.value = infoRes.value.data;
  } else {
    console.error('Failed to fetch system info:', infoRes.reason);
  }

  const volumeCount =
    volumesRes.status === 'fulfilled'
      ? (Array.isArray(volumesRes.value.data?.Volumes)
        ? volumesRes.value.data.Volumes.length
        : Array.isArray(volumesRes.value.data)
          ? volumesRes.value.data.length
          : resourceCounts.value.volumes)
      : resourceCounts.value.volumes;
  if (volumesRes.status !== 'fulfilled') {
    console.error('Failed to fetch volumes:', volumesRes.reason);
  }

  const networkCount =
    networksRes.status === 'fulfilled' && Array.isArray(networksRes.value.data)
      ? networksRes.value.data.length
      : resourceCounts.value.networks;
  if (networksRes.status !== 'fulfilled') {
    console.error('Failed to fetch networks:', networksRes.reason);
  }

  resourceCounts.value = {
    volumes: volumeCount,
    networks: networkCount,
  };
};

const setupStatsInterval = () => {
  if (statsTimer) window.clearInterval(statsTimer);
  const ms = appSettings.general.autoRefreshMs;
  if (ms > 0) {
    statsTimer = window.setInterval(fetchStats, ms);
  }
};

const logout = async () => {
  await signOut();
  await router.replace({ name: 'auth' });
};

onMounted(async () => {
  if (!validTabIds.has(activeTab.value)) {
    await setActiveTab('dashboard');
    return;
  }
  await fetchStats();
  setupStatsInterval();
  window.addEventListener('keydown', handleGlobalShortcut);
});

onUnmounted(() => {
  if (statsTimer) window.clearInterval(statsTimer);
  window.removeEventListener('keydown', handleGlobalShortcut);
});

watch(() => appSettings.general.autoRefreshMs, () => {
  setupStatsInterval();
});
</script>

<template>
  <div class="app-container">
    <aside class="sidebar glass-panel">
      <div class="logo">
        <i class="fa-brands fa-docker logo-whale" aria-hidden="true"></i>
        <span>Dock Manager</span>
      </div>

      <nav class="nav-links">
        <button v-for="tab in tabs" :key="tab.id" class="nav-item" :class="{ active: activeTab === tab.id }"
          @click="setActiveTab(tab.id)">
          <component :is="tab.icon" :size="20" />
          {{ tab.name }}
        </button>
      </nav>

      <div class="sidebar-footer">
        <div class="system-stats" v-if="systemInfo && appSettings.ui.showSidebarStats">
          <div class="stat-item">
            <Cpu :size="16" />
            <span>{{ systemInfo.NCPU }} CPUs</span>
          </div>
          <div class="stat-item">
            <Database :size="16" />
            <span>{{ (systemInfo.MemTotal / 1024 / 1024 / 1024).toFixed(1) }} GB</span>
          </div>
        </div>
        <div class="app-version">v{{ appSettings.about.appVersion }}</div>
      </div>
    </aside>

    <main class="main-content">
      <header class="content-header">
        <div class="title-group">
          <h1>{{tabs.find(t => t.id === activeTab)?.name}}</h1>
          <p class="subtitle" v-if="activeTab === 'dashboard'">Real-time system health and resource metrics</p>
        </div>
        <div class="header-actions">
          <div class="user-pill">
            <span class="user-pill-label">Signed in as</span>
            <strong>{{ authState.user?.username }}</strong>
          </div>
          <div class="status-badge" v-if="systemInfo">
            <span class="pulse"></span>
            Docker {{ systemInfo.ServerVersion }}
          </div>
          <button class="logout-btn" type="button" @click="logout">
            Logout
          </button>
        </div>
      </header>

      <section class="content-area animate-fade-in">
        <Dashboard v-if="activeTab === 'dashboard'" :system-info="systemInfo" :resource-counts="resourceCounts" />
        <ContainerList v-else-if="activeTab === 'containers'" />
        <ImageList v-else-if="activeTab === 'images'" />
        <VolumeList v-else-if="activeTab === 'volumes'" />
        <NetworkList v-else-if="activeTab === 'networks'" />
        <ComposeList v-else-if="activeTab === 'compose'" />
        <SettingsPanel v-else-if="activeTab === 'settings'" :system-info="systemInfo" />
      </section>
    </main>
  </div>
</template>

<style scoped>
.app-container {
  display: flex;
  min-height: 100vh;
  min-height: 100dvh;
  height: 100%;
  width: 100%;
  max-width: 100%;
  overflow: hidden;
  background: radial-gradient(circle at top right, rgba(36, 150, 237, 0.08), transparent),
    radial-gradient(circle at bottom left, rgba(29, 99, 237, 0.07), transparent);
}

.sidebar {
  width: 260px;
  height: calc(100vh - 32px);
  margin: 16px;
  display: flex;
  flex-direction: column;
  padding: 24px 16px;
  flex-shrink: 0;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 1.3rem;
  font-weight: 700;
  margin-bottom: 40px;
  padding: 0 8px;
  color: var(--text-main);
}

.logo-whale {
  font-size: 30px;
  color: #2496ed;
  line-height: 1;
  filter: drop-shadow(0 0 8px rgba(36, 150, 237, 0.5));
}

.nav-links {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex-grow: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 0.95rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.nav-item:hover {
  background: var(--glass);
  color: var(--text-main);
}

.nav-item.active {
  background: var(--primary);
  color: white;
  box-shadow: 0 4px 12px rgba(36, 150, 237, 0.32);
}

.sidebar-footer {
  margin-top: auto;
  padding-top: 24px;
  border-top: 1px solid var(--glass-border);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.system-stats {
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8rem;
  color: var(--text-muted);
}

.app-version {
  padding: 0 16px;
  font-size: 0.78rem;
  color: var(--text-muted);
  text-align: center;
  letter-spacing: 0.02em;
}

.main-content {
  flex-grow: 1;
  min-width: 0;
  min-height: 0;
  height: calc(100vh - 32px);
  height: calc(100dvh - 32px);
  margin: 16px 16px 16px 0;
  padding: 28px 32px 48px;
  overflow-y: auto;
  overflow-x: hidden;
  display: flex;
  flex-direction: column;
  border-radius: 28px;
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.2), rgba(15, 23, 42, 0.08));
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 32px;
}

.header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}

.title-group h1 {
  font-size: 2rem;
  font-weight: 700;
  margin: 0;
}

.subtitle {
  color: var(--text-muted);
  font-size: 0.9rem;
  margin-top: 4px;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  background: var(--glass);
  border: 1px solid var(--glass-border);
  border-radius: 20px;
  font-size: 0.85rem;
  color: var(--text-muted);
}

.user-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.28);
  color: rgba(226, 232, 240, 0.86);
}

.user-pill-label {
  font-size: 0.82rem;
  color: var(--text-muted);
}

.logout-btn {
  height: 30px;
  padding: 0 16px;
  border: 1px solid rgba(248, 113, 113, 0.22);
  border-radius: 14px;
  background: rgba(127, 29, 29, 0.16);
  color: #fecaca;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.18s ease, background 0.18s ease;
}

.logout-btn:hover {
  transform: translateY(-1px);
  background: rgba(127, 29, 29, 0.24);
}

.pulse {
  width: 8px;
  height: 8px;
  background: var(--success);
  border-radius: 50%;
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4);
  animation: pulse-green 2s infinite;
}

@keyframes pulse-green {
  0% {
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
  }

  70% {
    box-shadow: 0 0 0 10px rgba(16, 185, 129, 0);
  }

  100% {
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0);
  }
}

.content-area {
  flex-grow: 1;
  min-width: 0;
  min-height: 0;
  width: 100%;
}

@media (max-width: 960px) {
  .content-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-actions {
    justify-content: flex-start;
  }
}
</style>
