<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useI18n } from 'vue-i18n';
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
  ChevronDown,
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
const languageMenuOpen = ref(false);
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

const languages = [
  { value: 'vi', flagClass: 'fi fi-vn' },
  { value: 'en', flagClass: 'fi fi-sh' },
  { value: 'zh', flagClass: 'fi fi-cn' },
] as const;

const activeLanguage = computed(() =>
  languages.find((language) => language.value === appSettings.general.language) ?? languages[1]
);

const languageLabelKeys: Record<typeof languages[number]['value'], string> = {
  vi: 'settings.vietnamese',
  en: 'settings.english',
  zh: 'settings.chinese',
};

const setLanguage = (language: typeof languages[number]['value']) => {
  appSettings.general.language = language;
  languageMenuOpen.value = false;
};

const setActiveTab = async (tabId: string) => {
  if (!validTabIds.has(tabId)) return;
  persistStoredValue('dock-manager.active-tab', tabId);
  await router.replace({ name: 'app', params: { tab: tabId } });
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
    <aside class="hidden w-73 shrink-0 p-4 lg:block">
      <div class="glass-panel flex h-[calc(100vh-2rem)] flex-col overflow-hidden">
        <div class="border-b px-6 py-6" style="border-color: var(--glass-border);">
          <div class="mb-2 flex items-center gap-3">
            <div class="grid h-12 w-12 place-items-center border text-2xl"
              style="border-color: var(--primary); background: rgba(29, 78, 216, 0.12); color: var(--primary);">
              <i class="fa-brands fa-docker" aria-hidden="true"></i>
            </div>
            <div>
              <p class="text-[11px] uppercase tracking-[0.24em]" style="color: var(--text-muted);">{{ t('nav.opsPanel')
              }}</p>
              <div class="text-xl font-bold tracking-tight">Dock Manager</div>
            </div>
          </div>
        </div>

        <nav class="flex flex-1 flex-col gap-2 p-4">
          <button v-for="(tab, index) in tabs" :key="tab.id"
            class="cursor-pointer flex items-center justify-between border px-4 py-3 text-left text-sm font-semibold transition"
            :class="activeTab === tab.id ? 'shadow-[4px_4px_0_rgba(0,0,0,0.28)]' : ''" :style="activeTab === tab.id
              ? 'border-color: var(--primary); background: var(--primary); color: white;'
              : 'border-color: var(--glass-border); background: var(--glass); color: var(--text-muted);'"
            @click="setActiveTab(tab.id)">
            <span class="flex items-center gap-3">
              <component :is="tab.icon" :size="18" />
              {{ t(tab.nameKey) }}
            </span>
            <span class="font-mono text-[11px]">ALT+{{ index + 1 }}</span>
          </button>
        </nav>

        <div class="mt-auto border-t px-6 py-5 text-sm" style="border-color: var(--glass-border);">
          <div v-if="systemInfo && appSettings.ui.showSidebarStats" class="mb-4 grid gap-2">
            <div class="flex items-center justify-between border px-3 py-2" style="border-color: var(--glass-border);">
              <span class="flex items-center gap-2" style="color: var(--text-muted);">
                <Cpu :size="15" />
                {{ t('nav.cpu') }}
              </span>
              <strong>{{ systemInfo.NCPU }}</strong>
            </div>
            <div class="flex items-center justify-between border px-3 py-2" style="border-color: var(--glass-border);">
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

    <main class="min-w-0 flex-1 p-4 pl-4 lg:pl-0">
      <div class="glass-panel flex h-[calc(100dvh-2rem)] min-w-0 flex-col overflow-hidden">
        <header class="border-b px-5 py-5 sm:px-8"
          style="border-color: var(--glass-border); background: linear-gradient(180deg, rgba(255,255,255,0.02), transparent);">
          <div class="flex flex-col gap-5 xl:flex-row xl:items-start xl:justify-between">
            <div>
              <p class="mb-2 text-[11px] uppercase tracking-[0.24em]" style="color: var(--text-muted);">
                {{ activeTabMeta ? t(activeTabMeta.nameKey) : '' }}
              </p>
              <h1 class="text-3xl font-bold tracking-tight sm:text-4xl">{{ activeTabMeta ? t(activeTabMeta.nameKey) : ''
              }}</h1>
            </div>

            <div class="flex flex-wrap items-center justify-end gap-3">
              <div class="relative">
                <button
                  class="inline-flex h-10 items-center justify-between gap-3 border px-3 text-sm font-semibold transition cursor-pointer"
                  type="button" :aria-label="t(languageLabelKeys[activeLanguage.value])"
                  :aria-expanded="languageMenuOpen"
                  style="border-color: var(--glass-border); background: var(--glass); color: var(--text-main);"
                  @click="languageMenuOpen = !languageMenuOpen">
                  <span class="inline-flex items-center gap-2">
                    <span :class="activeLanguage.flagClass"></span>
                  </span>
                  <ChevronDown :size="16" style="color: var(--text-muted);" />
                </button>

                <div v-if="languageMenuOpen"
                  class="absolute right-0 z-30 mt-2 grid border p-1 shadow-[4px_4px_0_rgba(0,0,0,0.28)]"
                  style="border-color: var(--glass-border); background: var(--bg-card);">
                  <button v-for="language in languages" :key="language.value"
                    class="inline-flex h-9 w-11 items-center justify-center text-sm font-semibold transition cursor-pointer"
                    :aria-label="t(languageLabelKeys[language.value])" type="button" :style="appSettings.general.language === language.value
                      ? 'background: var(--primary); color: white;'
                      : 'background: transparent; color: var(--text-main);'" @click="setLanguage(language.value)">
                    <span :class="language.flagClass"></span>
                  </button>
                </div>
              </div>

              <div class="inline-flex items-center border p-1"
                style="border-color: var(--glass-border); background: var(--glass);">
                <button class="grid h-8 w-10 place-items-center transition cursor-pointer" type="button"
                  :title="t('settings.dark')" :aria-label="t('settings.dark')" :style="appSettings.ui.theme === 'dark'
                    ? 'background: var(--primary); color: white;'
                    : 'background: transparent; color: var(--text-main);'" @click="appSettings.ui.theme = 'dark'">
                  <Moon :size="17" />
                </button>
                <button class="grid h-8 w-10 place-items-center transition cursor-pointer" type="button"
                  :title="t('settings.light')" :aria-label="t('settings.light')" :style="appSettings.ui.theme === 'light'
                    ? 'background: var(--primary); color: white;'
                    : 'background: transparent; color: var(--text-main);'" @click="appSettings.ui.theme = 'light'">
                  <Sun :size="17" />
                </button>
              </div>

              <div v-if="systemInfo" class="flex items-center gap-3 border px-4 py-2 text-sm"
                style="border-color: var(--glass-border); background: var(--glass);">
                <span class="h-2.5 w-2.5 animate-pulse" style="background: var(--success);"></span>
                Docker {{ systemInfo.ServerVersion }}
              </div>
            </div>
          </div>
        </header>

        <div class="border-b px-5 py-3 lg:hidden" style="border-color: var(--glass-border);">
          <div class="flex gap-2 overflow-x-auto">
            <button v-for="tab in tabs" :key="tab.id" class="shrink-0 border px-3 py-2 text-sm font-semibold" :style="activeTab === tab.id
              ? 'border-color: var(--primary); background: var(--primary); color: white;'
              : 'border-color: var(--glass-border); background: var(--glass); color: var(--text-muted);'"
              @click="setActiveTab(tab.id)">
              {{ t(tab.nameKey) }}
            </button>
          </div>
        </div>

        <section class="animate-fade-in min-h-0 flex-1 overflow-auto px-5 py-5 sm:px-8 sm:py-6">
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
