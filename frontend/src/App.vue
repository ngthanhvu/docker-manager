<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue';
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
import { dockerApi } from './api';
import { appSettings } from './ui/settings';
import Dashboard from './components/Dashboard.vue';
import ContainerList from './components/ContainerList.vue';
import ImageList from './components/ImageList.vue';
import VolumeList from './components/VolumeList.vue';
import NetworkList from './components/NetworkList.vue';
import ComposeList from './components/ComposeList.vue';
import UiFeedback from './components/UiFeedback.vue';
import SettingsPanel from './components/SettingsPanel.vue';

const activeTab = ref('dashboard');
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

onMounted(() => {
  fetchStats();
  setupStatsInterval();
});

onUnmounted(() => {
  if (statsTimer) window.clearInterval(statsTimer);
});

watch(() => appSettings.general.autoRefreshMs, () => {
  setupStatsInterval();
});
</script>

<template>
  <div class="app-container">
    <!-- Sidebar -->
    <aside class="sidebar glass-panel">
      <div class="logo">
        <i class="fa-brands fa-docker logo-whale" aria-hidden="true"></i>
        <span>Docker Hub</span>
      </div>

      <nav class="nav-links">
        <button v-for="tab in tabs" :key="tab.id" class="nav-item" :class="{ active: activeTab === tab.id }"
          @click="activeTab = tab.id">
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

    <!-- Main Content -->
    <main class="main-content">
      <header class="content-header">
        <div class="title-group">
          <h1>{{tabs.find(t => t.id === activeTab)?.name}}</h1>
          <p class="subtitle" v-if="activeTab === 'dashboard'">Real-time system health and resource metrics</p>
        </div>
        <div class="header-actions">
          <div class="status-badge" v-if="systemInfo">
            <span class="pulse"></span>
            Docker {{ systemInfo.ServerVersion }}
          </div>
        </div>
      </header>

      <section class="content-area animate-fade-in">
        <!-- Dashboard Component -->
        <Dashboard v-if="activeTab === 'dashboard'" :system-info="systemInfo" :resource-counts="resourceCounts" />

        <!-- Resource Components -->
        <ContainerList v-else-if="activeTab === 'containers'" />
        <ImageList v-else-if="activeTab === 'images'" />
        <VolumeList v-else-if="activeTab === 'volumes'" />
        <NetworkList v-else-if="activeTab === 'networks'" />
        <ComposeList v-else-if="activeTab === 'compose'" />
        <SettingsPanel v-else-if="activeTab === 'settings'" :system-info="systemInfo" />
      </section>
    </main>
  </div>
  <UiFeedback />
</template>

<style scoped>
.app-container {
  display: flex;
  height: 100vh;
  width: 100vw;
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
  font-size: 1.5rem;
  font-weight: 700;
  margin-bottom: 40px;
  padding: 0 8px;
  color: white;
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
  padding: 40px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
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
}
</style>
