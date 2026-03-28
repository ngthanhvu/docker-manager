<script setup lang="ts">
import { computed } from 'vue';
import { appSettings } from '../ui/settings';

const props = defineProps<{
  systemInfo?: any;
}>();

const apiHint = computed(() => `${appSettings.runtime.apiBaseUrl.replace(/\/+$/, '')}/api`);

const resetUI = () => {
  appSettings.ui.theme = 'dark';
  appSettings.ui.density = 'comfortable';
  appSettings.ui.fontScale = 1;
  appSettings.ui.showSidebarStats = true;
};
</script>

<template>
  <div class="flex flex-col gap-5">
    <section class="glass-panel p-5">
      <p class="section-heading">General</p>
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Auto refresh interval</span>
          <select v-model.number="appSettings.general.autoRefreshMs" class="app-select">
            <option :value="0">Off</option>
            <option :value="2000">2s</option>
            <option :value="5000">5s</option>
            <option :value="10000">10s</option>
          </select>
        </label>
        <label class="flex items-center justify-between border px-4 py-3" style="border-color: var(--glass-border); background: var(--glass);">
          <span class="text-sm font-semibold">Confirm destructive actions</span>
          <input v-model="appSettings.general.confirmDestructive" type="checkbox" class="h-5 w-5 accent-blue-600" />
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Language</span>
          <select v-model="appSettings.general.language" class="app-select">
            <option value="vi">Vietnamese</option>
            <option value="en">English</option>
          </select>
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Time format</span>
          <select v-model="appSettings.general.timeFormat" class="app-select">
            <option value="24h">24-hour</option>
            <option value="12h">12-hour</option>
          </select>
        </label>
      </div>
    </section>

    <section class="glass-panel p-5">
      <div class="mb-4 flex items-center justify-between gap-4">
        <p class="section-heading mb-0">UI</p>
        <button class="btn btn-ghost" @click="resetUI">Reset UI</button>
      </div>
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Theme</span>
          <select v-model="appSettings.ui.theme" class="app-select">
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Density</span>
          <select v-model="appSettings.ui.density" class="app-select">
            <option value="comfortable">Comfortable</option>
            <option value="compact">Compact</option>
          </select>
        </label>
        <label class="block lg:col-span-2">
          <span class="mb-2 block text-sm font-semibold">Font scale ({{ appSettings.ui.fontScale.toFixed(2) }})</span>
          <div class="border px-4 py-4" style="border-color: var(--glass-border); background: var(--glass);">
            <input v-model.number="appSettings.ui.fontScale" type="range" min="0.9" max="1.15" step="0.01" class="w-full accent-blue-600" />
          </div>
        </label>
        <label class="flex items-center justify-between border px-4 py-3 lg:col-span-2" style="border-color: var(--glass-border); background: var(--glass);">
          <span class="text-sm font-semibold">Show sidebar stats</span>
          <input v-model="appSettings.ui.showSidebarStats" type="checkbox" class="h-5 w-5 accent-blue-600" />
        </label>
      </div>
    </section>

    <section class="glass-panel p-5">
      <p class="section-heading">Docker Runtime</p>
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="block lg:col-span-2">
          <span class="mb-2 block text-sm font-semibold">Docker API endpoint</span>
          <input v-model.trim="appSettings.runtime.apiBaseUrl" type="text" placeholder="http://localhost:8080" class="app-input" />
          <small class="mt-2 block text-xs" style="color: var(--text-muted);">Current API base: {{ apiHint }}</small>
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Default log tail</span>
          <input v-model.number="appSettings.runtime.defaultLogTail" type="number" min="50" max="5000" step="50" class="app-input" />
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Terminal shell</span>
          <select v-model="appSettings.runtime.terminalShell" class="app-select">
            <option value="/bin/sh">/bin/sh</option>
            <option value="/bin/bash">/bin/bash</option>
          </select>
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Terminal theme</span>
          <select v-model="appSettings.runtime.terminalTheme" class="app-select">
            <option value="ocean">Ocean Blue</option>
            <option value="matrix">Matrix Green</option>
            <option value="amber">Amber Gold</option>
          </select>
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Terminal font size</span>
          <input v-model.number="appSettings.runtime.terminalFontSize" type="number" min="11" max="20" step="1" class="app-input" />
        </label>
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Compose refresh interval</span>
          <select v-model.number="appSettings.runtime.composeRefreshMs" class="app-select">
            <option :value="0">Off</option>
            <option :value="2000">2s</option>
            <option :value="5000">5s</option>
            <option :value="10000">10s</option>
          </select>
        </label>
      </div>
    </section>

    <section class="glass-panel p-5">
      <p class="section-heading">Notifications</p>
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="block">
          <span class="mb-2 block text-sm font-semibold">Toast duration (ms)</span>
          <input v-model.number="appSettings.notifications.toastDurationMs" type="number" min="1000" max="10000" step="100" class="app-input" />
        </label>
        <label class="flex items-center justify-between border px-4 py-3" style="border-color: var(--glass-border); background: var(--glass);">
          <span class="text-sm font-semibold">Show success toasts</span>
          <input v-model="appSettings.notifications.showSuccessToast" type="checkbox" class="h-5 w-5 accent-blue-600" />
        </label>
        <label class="flex items-center justify-between border px-4 py-3" style="border-color: var(--glass-border); background: var(--glass);">
          <span class="text-sm font-semibold">Show detailed errors</span>
          <input v-model="appSettings.notifications.showDetailedErrors" type="checkbox" class="h-5 w-5 accent-blue-600" />
        </label>
      </div>
    </section>

    <section class="glass-panel p-5">
      <p class="section-heading">Safety</p>
      <div class="grid gap-4 lg:grid-cols-2">
        <label class="flex items-center justify-between border px-4 py-3 lg:col-span-2" style="border-color: var(--glass-border); background: var(--glass);">
          <span class="text-sm font-semibold">Require typing DELETE for destructive actions</span>
          <input v-model="appSettings.safety.softDeleteRequireTyping" type="checkbox" class="h-5 w-5 accent-blue-600" />
        </label>
        <label class="block lg:col-span-2">
          <span class="mb-2 block text-sm font-semibold">Protected resources (comma-separated)</span>
          <input v-model="appSettings.safety.protectedResources" type="text" placeholder="mysql-data, redis-network" class="app-input" />
        </label>
      </div>
    </section>

    <section class="glass-panel p-5">
      <p class="section-heading">About</p>
      <div class="grid gap-px border text-sm sm:grid-cols-[220px_minmax(0,1fr)]" style="border-color: var(--glass-border); background: var(--glass-border);">
        <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">App version</div>
        <div class="px-4 py-3" style="background: var(--bg-card);">v{{ appSettings.about.appVersion }}</div>
        <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">Build date</div>
        <div class="px-4 py-3" style="background: var(--bg-card);">{{ appSettings.about.buildDate }}</div>
        <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">Docker engine</div>
        <div class="px-4 py-3" style="background: var(--bg-card);">{{ props.systemInfo?.ServerVersion || 'N/A' }}</div>
        <div class="px-4 py-3 font-semibold" style="background: var(--table-header-bg);">Operating system</div>
        <div class="px-4 py-3" style="background: var(--bg-card);">{{ props.systemInfo?.OperatingSystem || 'N/A' }}</div>
      </div>
    </section>
  </div>
</template>
