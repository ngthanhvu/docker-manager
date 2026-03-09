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
    <div class="settings-view">
        <section class="glass-panel section">
            <h3>General</h3>
            <div class="grid">
                <label class="field">
                    <span>Auto refresh interval</span>
                    <select v-model.number="appSettings.general.autoRefreshMs">
                        <option :value="0">Off</option>
                        <option :value="2000">2s</option>
                        <option :value="5000">5s</option>
                        <option :value="10000">10s</option>
                    </select>
                </label>
                <label class="field">
                    <span>Confirm destructive actions</span>
                    <input v-model="appSettings.general.confirmDestructive" type="checkbox" />
                </label>
                <label class="field">
                    <span>Language</span>
                    <select v-model="appSettings.general.language">
                        <option value="vi">Vietnamese</option>
                        <option value="en">English</option>
                    </select>
                </label>
                <label class="field">
                    <span>Time format</span>
                    <select v-model="appSettings.general.timeFormat">
                        <option value="24h">24-hour</option>
                        <option value="12h">12-hour</option>
                    </select>
                </label>
            </div>
        </section>

        <section class="glass-panel section">
            <h3>UI</h3>
            <div class="grid">
                <label class="field">
                    <span>Theme</span>
                    <select v-model="appSettings.ui.theme">
                        <option value="dark">Dark</option>
                        <option value="light">Light</option>
                    </select>
                </label>
                <label class="field">
                    <span>Density</span>
                    <select v-model="appSettings.ui.density">
                        <option value="comfortable">Comfortable</option>
                        <option value="compact">Compact</option>
                    </select>
                </label>
                <label class="field">
                    <span>Font scale ({{ appSettings.ui.fontScale.toFixed(2) }})</span>
                    <input v-model.number="appSettings.ui.fontScale" type="range" min="0.9" max="1.15" step="0.01" />
                </label>
                <label class="field">
                    <span>Show sidebar stats</span>
                    <input v-model="appSettings.ui.showSidebarStats" type="checkbox" />
                </label>
            </div>
            <div class="section-actions">
                <button class="btn btn-ghost" @click="resetUI">Reset UI</button>
            </div>
        </section>

        <section class="glass-panel section">
            <h3>Docker Runtime</h3>
            <div class="grid">
                <label class="field field-full">
                    <span>Docker API endpoint</span>
                    <input v-model.trim="appSettings.runtime.apiBaseUrl" type="text" placeholder="http://localhost:8080" />
                    <small>Current API base: {{ apiHint }}</small>
                </label>
                <label class="field">
                    <span>Default compose log tail</span>
                    <input v-model.number="appSettings.runtime.defaultLogTail" type="number" min="50" max="5000" step="50" />
                </label>
                <label class="field">
                    <span>Terminal shell</span>
                    <select v-model="appSettings.runtime.terminalShell">
                        <option value="/bin/sh">/bin/sh</option>
                        <option value="/bin/bash">/bin/bash</option>
                    </select>
                </label>
                <label class="field">
                    <span>Compose refresh interval</span>
                    <select v-model.number="appSettings.runtime.composeRefreshMs">
                        <option :value="0">Off</option>
                        <option :value="2000">2s</option>
                        <option :value="5000">5s</option>
                        <option :value="10000">10s</option>
                    </select>
                </label>
            </div>
        </section>

        <section class="glass-panel section">
            <h3>Notifications</h3>
            <div class="grid">
                <label class="field">
                    <span>Toast duration (ms)</span>
                    <input v-model.number="appSettings.notifications.toastDurationMs" type="number" min="1000" max="10000" step="100" />
                </label>
                <label class="field">
                    <span>Show success toasts</span>
                    <input v-model="appSettings.notifications.showSuccessToast" type="checkbox" />
                </label>
                <label class="field">
                    <span>Show detailed errors</span>
                    <input v-model="appSettings.notifications.showDetailedErrors" type="checkbox" />
                </label>
            </div>
        </section>

        <section class="glass-panel section">
            <h3>Safety</h3>
            <div class="grid">
                <label class="field">
                    <span>Require typing DELETE for destructive actions</span>
                    <input v-model="appSettings.safety.softDeleteRequireTyping" type="checkbox" />
                </label>
                <label class="field field-full">
                    <span>Protected resources (comma-separated)</span>
                    <input
                        v-model="appSettings.safety.protectedResources"
                        type="text"
                        placeholder="mysql-data, redis-network"
                    />
                </label>
            </div>
        </section>

        <section class="glass-panel section">
            <h3>About</h3>
            <div class="about-grid">
                <div>App version</div>
                <div>v{{ appSettings.about.appVersion }}</div>
                <div>Build date</div>
                <div>{{ appSettings.about.buildDate }}</div>
                <div>Docker engine</div>
                <div>{{ props.systemInfo?.ServerVersion || 'N/A' }}</div>
                <div>Operating system</div>
                <div>{{ props.systemInfo?.OperatingSystem || 'N/A' }}</div>
            </div>
        </section>
    </div>
</template>

<style scoped>
.settings-view { display: flex; flex-direction: column; gap: 16px; }
.section { padding: 16px; }
.section h3 { margin: 0 0 12px; font-size: 1.08rem; }
.grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px 16px;
}
.field { display: flex; flex-direction: column; gap: 6px; font-size: 0.85rem; color: var(--text-muted); }
.field input[type="text"],
.field input[type="number"],
.field select {
    border: 1px solid var(--glass-border);
    background: rgba(15, 23, 42, 0.6);
    color: var(--text-main);
    border-radius: 10px;
    padding: 8px 10px;
    outline: none;
}
.field input[type="checkbox"] { width: 18px; height: 18px; }
.field input[type="range"] { width: 100%; }
.field small { font-size: 0.76rem; color: var(--text-muted); }
.field-full { grid-column: 1 / -1; }
.section-actions { margin-top: 12px; display: flex; justify-content: flex-end; }
.about-grid {
    display: grid;
    grid-template-columns: 180px 1fr;
    gap: 8px 16px;
    font-size: 0.88rem;
}

@media (max-width: 980px) {
    .grid { grid-template-columns: 1fr; }
    .about-grid { grid-template-columns: 1fr; }
}
</style>
