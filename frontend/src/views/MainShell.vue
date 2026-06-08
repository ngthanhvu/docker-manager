<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from '../i18n';
import {
  LayoutDashboard,
  Container,
  Box,
  HardDrive,
  Network,
  Layers,
  Cpu,
  Database,
  Settings,
  Moon,
  Sun,
  Menu,
  X,
} from 'lucide-vue-next';
import { dockerApi } from '../api';
import { appSettings } from '../ui/settings';
import { persistStoredValue } from '../ui/viewState';
import Dashboard from '../components/Dashboard.vue';
import ContainerList from '../components/ContainerList.vue';
import ImageList from '../components/ImageList.vue';
import VolumeList from '../components/VolumeList.vue';
import NetworkList from '../components/NetworkList.vue';
import ComposeList from '../components/ComposeList.vue';
import SettingsPanel from '../components/SettingsPanel.vue';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const systemInfo = ref<any>(null);
const resourceCounts = ref<{ volumes: number; networks: number }>({ volumes: 0, networks: 0 });
const sidebarOpen = ref(false);
let statsTimer: number | null = null;

const tabs = [
  { id: 'dashboard', nameKey: 'nav.dashboard', icon: LayoutDashboard },
  { id: 'containers', nameKey: 'nav.containers', icon: Container },
  { id: 'images', nameKey: 'nav.images', icon: Box },
  { id: 'volumes', nameKey: 'nav.volumes', icon: HardDrive },
  { id: 'networks', nameKey: 'nav.networks', icon: Network },
  { id: 'compose', nameKey: 'nav.compose', icon: Layers },
  { id: 'settings', nameKey: 'nav.settings', icon: Settings },
];

const validTabIds = new Set(tabs.map((tab) => tab.id));

const activeTab = computed(() => {
  const tab = typeof route.params.tab === 'string' ? route.params.tab : 'dashboard';
  return validTabIds.has(tab) ? tab : 'dashboard';
});

const activeTabMeta = computed(() => tabs.find((tab) => tab.id === activeTab.value) ?? tabs[0]);

const setActiveTab = async (tabId: string) => {
  if (!validTabIds.has(tabId)) return;
  persistStoredValue('dock-manager.active-tab', tabId);
  await router.replace({ name: 'app', params: { tab: tabId } });
};

const handleTabClick = async (tabId: string) => {
  await setActiveTab(tabId);
  if (window.matchMedia('(max-width: 1023px)').matches) {
    sidebarOpen.value = false;
  }
};

const toggleSidebar = () => {
  sidebarOpen.value = !sidebarOpen.value;
};

const handleGlobalShortcut = (event: KeyboardEvent) => {
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

onMounted(async () => {
  sidebarOpen.value = window.matchMedia('(min-width: 1024px)').matches;
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
  <div class="flex min-h-screen bg-transparent">
    <div v-if="sidebarOpen" class="fixed inset-0 z-30 bg-black/55 lg:hidden" aria-hidden="true" @click="sidebarOpen = false">
    </div>

    <aside class="fixed inset-y-0 left-0 z-40 w-[min(18rem,calc(100vw-2rem))] overflow-hidden p-3 transition-all duration-200 ease-out sm:p-4 lg:sticky lg:top-0 lg:z-10 lg:h-screen lg:shrink-0"
      :class="sidebarOpen ? 'translate-x-0 lg:w-73 lg:p-3 xl:p-4' : '-translate-x-full lg:w-24 lg:translate-x-0 lg:p-3 xl:w-[6.5rem] xl:p-4'">
      <div class="glass-panel flex h-full flex-col overflow-hidden lg:h-[calc(100vh-2rem)]">
        <div class="flex border-b px-4 py-4 sm:px-5 lg:h-16 lg:items-center lg:py-0 xl:h-[77px]"
          :class="sidebarOpen ? '' : 'lg:justify-center lg:px-0'" style="border-color: var(--glass-border);">
          <div class="flex items-center gap-3" :class="sidebarOpen ? '' : 'lg:w-full lg:justify-center'">
            <div class="grid h-10 w-10 shrink-0 place-items-center rounded-lg border text-xl"
              style="border-color: var(--primary); background: rgba(29, 78, 216, 0.12); color: var(--primary);">
              <i class="fa-brands fa-docker" aria-hidden="true"></i>
            </div>
            <div class="min-w-0 flex-1" :class="sidebarOpen ? '' : 'lg:hidden'">
              <p class="text-[10px] font-medium uppercase tracking-[0.08em]" style="color: var(--text-muted);">{{ t('nav.opsPanel')
              }}</p>
              <div class="truncate text-lg font-semibold tracking-tight">Dock Manager</div>
            </div>
            <button class="btn btn-icon h-9 w-9 shrink-0 lg:hidden" type="button" :aria-label="t('common.close')" @click="sidebarOpen = false">
              <X :size="17" />
            </button>
          </div>
        </div>

        <nav class="flex flex-1 flex-col gap-2 p-3 sm:p-4"
          :class="sidebarOpen ? '' : 'lg:items-center lg:p-2'">
          <button v-for="(tab, index) in tabs" :key="tab.id"
            class="cursor-pointer flex items-center justify-between rounded-md border px-3 py-2.5 text-left text-sm font-medium transition"
            :class="sidebarOpen ? '' : 'lg:h-11 lg:w-11 lg:justify-center lg:px-0 lg:py-0'"
            :title="sidebarOpen ? undefined : t(tab.nameKey)"
            :style="activeTab === tab.id
              ? 'border-color: color-mix(in srgb, var(--primary) 22%, var(--glass-border)); background: color-mix(in srgb, var(--primary) 16%, var(--glass)); color: var(--text-main);'
              : 'border-color: var(--glass-border); background: var(--glass); color: var(--text-muted);'"
            @click="handleTabClick(tab.id)">
            <span class="flex items-center gap-3" :class="sidebarOpen ? '' : 'lg:justify-center'">
              <component :is="tab.icon" :size="18" />
              <span :class="sidebarOpen ? '' : 'lg:hidden'">{{ t(tab.nameKey) }}</span>
            </span>
            <span class="font-mono text-[11px]" :class="sidebarOpen ? '' : 'lg:hidden'"
              style="color: var(--text-muted);">ALT+{{ index + 1 }}</span>
          </button>
        </nav>

        <div class="mt-auto border-t px-4 py-4 text-sm sm:px-5" :class="sidebarOpen ? '' : 'lg:hidden'"
          style="border-color: var(--glass-border);">
          <div v-if="systemInfo && appSettings.ui.showSidebarStats" class="mb-4 grid gap-2">
            <div class="flex items-center justify-between rounded-md border px-3 py-2" style="border-color: var(--glass-border);">
              <span class="flex items-center gap-2" style="color: var(--text-muted);">
                <Cpu :size="15" />
                {{ t('nav.cpu') }}
              </span>
              <strong>{{ systemInfo.NCPU }}</strong>
            </div>
            <div class="flex items-center justify-between rounded-md border px-3 py-2" style="border-color: var(--glass-border);">
              <span class="flex items-center gap-2" style="color: var(--text-muted);">
                <Database :size="15" />
                {{ t('nav.memory') }}
              </span>
              <strong>{{ (systemInfo.MemTotal / 1024 / 1024 / 1024).toFixed(1) }} GB</strong>
            </div>
          </div>
          <div class="font-mono text-xs uppercase tracking-[0.16em]" style="color: var(--text-muted);">
            {{ t('nav.build') }} {{ appSettings.about.appVersion }}
          </div>
        </div>
      </div>
    </aside>

    <main class="min-w-0 flex-1 p-2 sm:p-3 lg:pl-0 xl:p-4 xl:pl-0">
      <div class="glass-panel flex h-[calc(100dvh-1rem)] min-w-0 flex-col overflow-hidden sm:h-[calc(100dvh-2rem)]">
        <header class="flex border-b px-4 py-3 sm:px-5 sm:py-3 lg:min-h-16 lg:items-center lg:py-2 xl:min-h-[77px] xl:px-6"
          style="border-color: var(--glass-border); background: var(--bg-card);">
          <div class="flex w-full min-w-0 flex-col gap-2 md:flex-row md:items-center md:justify-between">
            <div class="flex min-w-0 items-center gap-3">
              <button class="btn btn-icon shrink-0" type="button" :aria-expanded="sidebarOpen" :aria-label="t('nav.toggleSidebar')"
                @click="toggleSidebar">
                <Menu :size="18" />
              </button>
              <div class="min-w-0">
                <h1 class="truncate text-lg font-semibold tracking-tight sm:text-xl xl:text-2xl">{{ activeTabMeta ? t(activeTabMeta.nameKey) : ''
                }}</h1>
              </div>
            </div>

            <div class="flex min-w-0 shrink-0 flex-wrap items-center gap-2 sm:justify-end sm:gap-3">
              <div class="inline-flex items-center border p-1"
                style="border-color: var(--glass-border); background: var(--glass); border-radius: var(--radius-control);">
                <button class="grid h-7 w-9 place-items-center transition cursor-pointer" type="button"
                  :title="t('settings.dark')" :aria-label="t('settings.dark')" :style="appSettings.ui.theme === 'dark'
                    ? 'background: var(--primary); color: white; border-radius: 4px;'
                    : 'background: transparent; color: var(--text-main);'" @click="appSettings.ui.theme = 'dark'">
                  <Moon :size="17" />
                </button>
                <button class="grid h-7 w-9 place-items-center transition cursor-pointer" type="button"
                  :title="t('settings.light')" :aria-label="t('settings.light')" :style="appSettings.ui.theme === 'light'
                    ? 'background: var(--primary); color: white; border-radius: 4px;'
                    : 'background: transparent; color: var(--text-main);'" @click="appSettings.ui.theme = 'light'">
                  <Sun :size="17" />
                </button>
              </div>

              <div v-if="systemInfo" class="hidden min-w-0 items-center gap-2 rounded-md border px-3 py-1.5 text-sm min-[1180px]:flex sm:px-4"
                style="border-color: var(--glass-border); background: var(--glass);">
                <span class="h-2.5 w-2.5 animate-pulse rounded-full" style="background: var(--success);"></span>
                <span class="truncate">Docker {{ systemInfo.ServerVersion }}</span>
              </div>
            </div>
          </div>
        </header>

        <section class="animate-fade-in min-h-0 flex-1 overflow-auto px-4 py-4 sm:px-5 sm:py-5 xl:px-8 xl:py-6">
          <Dashboard v-if="activeTab === 'dashboard'" :system-info="systemInfo" :resource-counts="resourceCounts" />
          <ContainerList v-else-if="activeTab === 'containers'" />
          <ImageList v-else-if="activeTab === 'images'" />
          <VolumeList v-else-if="activeTab === 'volumes'" />
          <NetworkList v-else-if="activeTab === 'networks'" />
          <ComposeList v-else-if="activeTab === 'compose'" />
          <SettingsPanel v-else-if="activeTab === 'settings'" :system-info="systemInfo" />
        </section>
      </div>
    </main>
  </div>
</template>
