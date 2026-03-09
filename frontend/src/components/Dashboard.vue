<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import {
    Container,
    Box,
    HardDrive,
    Network,
    TrendingUp
} from 'lucide-vue-next';
import {
    Chart as ChartJS,
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    Filler
} from 'chart.js';
import { Line } from 'vue-chartjs';
import { appSettings } from '../ui/settings';

ChartJS.register(
    CategoryScale,
    LinearScale,
    PointElement,
    LineElement,
    Title,
    Tooltip,
    Legend,
    Filler
);

const props = defineProps<{
    systemInfo: any;
    resourceCounts?: {
        volumes?: number;
        networks?: number;
    } | null;
}>();

const cpuData = ref<number[]>([]);
const memData = ref<number[]>([]);
const labels = ref<string[]>([]);
const maxDataPoints = 20;

const toNumber = (value: unknown) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
};

const formatBytes = (bytes: number) => {
    if (!bytes || bytes < 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    let value = bytes;
    let idx = 0;
    while (value >= 1024 && idx < units.length - 1) {
        value /= 1024;
        idx++;
    }
    return `${value.toFixed(value >= 100 ? 0 : value >= 10 ? 1 : 2)} ${units[idx]}`;
};

const cpuPercent = computed(() => toNumber((cpuData.value[cpuData.value.length - 1] ?? 0).toFixed(2)));
const memPercent = computed(() => toNumber((memData.value[memData.value.length - 1] ?? 0).toFixed(2)));
const ncpu = computed(() => Math.max(1, toNumber(props.systemInfo?.NCPU || 1)));
const loadAvg = computed(() => toNumber(((cpuPercent.value / 100) * ncpu.value).toFixed(2)));
const loadPercent = computed(() => Math.min(100, toNumber(((loadAvg.value / ncpu.value) * 100).toFixed(2))));

const memTotalBytes = computed(() => toNumber(props.systemInfo?.MemTotal || 0));
const memUsedBytes = computed(() => toNumber((memTotalBytes.value * (memPercent.value / 100)).toFixed(0)));

const volumeCount = computed(() => toNumber(props.resourceCounts?.volumes ?? props.systemInfo?.Volumes ?? 0));
const networkCount = computed(() => toNumber(props.resourceCounts?.networks ?? props.systemInfo?.Networks ?? 0));

const gauges = computed(() => ([
    {
        key: 'load',
        label: 'Load',
        percent: loadPercent.value,
        value: `${loadAvg.value}`,
        unit: '',
        detail: `(${loadAvg.value} / ${ncpu.value}) cores`,
    },
    {
        key: 'cpu',
        label: 'CPU',
        percent: cpuPercent.value,
        value: `${cpuPercent.value}`,
        unit: '%',
        detail: `${ncpu.value} cores`,
    },
    {
        key: 'memory',
        label: 'Memory',
        percent: memPercent.value,
        value: `${memPercent.value}`,
        unit: '%',
        detail: `${formatBytes(memUsedBytes.value)} / ${formatBytes(memTotalBytes.value)}`,
    }
]));

const chartData = computed(() => ({
    labels: [...labels.value],
    datasets: [
        {
            label: 'CPU Usage (%)',
            backgroundColor: 'rgba(36, 150, 237, 0.2)',
            borderColor: '#2496ed',
            data: [...cpuData.value],
            fill: true,
            tension: 0.4,
            pointRadius: 0
        },
        {
            label: 'RAM Usage (%)',
            backgroundColor: 'rgba(16, 185, 129, 0.2)',
            borderColor: '#10b981',
            data: [...memData.value],
            fill: true,
            tension: 0.4,
            pointRadius: 0
        }
    ]
}));

const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    scales: {
        y: {
            beginAtZero: true,
            max: 100,
            grid: {
                color: 'rgba(255, 255, 255, 0.05)'
            },
            ticks: {
                color: '#94a3b8',
                font: { size: 10 }
            }
        },
        x: {
            display: false
        }
    },
    plugins: {
        legend: {
            display: true,
            labels: {
                color: '#94a3b8',
                usePointStyle: true,
                boxWidth: 6
            }
        },
        tooltip: {
            mode: 'index' as const,
            intersect: false,
        }
    }
};

// Simulate real-time data since the backend /info doesn't provide dynamic CPU/RAM over time yet
let interval: any;
const setupInterval = () => {
    clearInterval(interval);
    const ms = appSettings.general.autoRefreshMs;
    if (ms > 0) {
        interval = setInterval(updateCharts, ms);
    }
};
const updateCharts = () => {
    const now = new Date().toLocaleTimeString([], {
        hour12: appSettings.general.timeFormat === '12h',
    });
    labels.value.push(now);

    // Simulated dynamic data
    cpuData.value.push(Math.random() * 30 + 10);
    memData.value.push(Math.random() * 20 + 40);

    if (labels.value.length > maxDataPoints) {
        labels.value.shift();
        cpuData.value.shift();
        memData.value.shift();
    }
};

onMounted(() => {
    updateCharts();
    setupInterval();
});

onUnmounted(() => {
    clearInterval(interval);
});

watch(() => appSettings.general.autoRefreshMs, () => {
    setupInterval();
});
</script>

<template>
    <div class="dashboard-view">
        <div class="stats-overview">
            <div class="stat-card glass-panel">
                <div class="card-header">
                    <div class="icon-box indigo">
                        <Container :size="20" />
                    </div>
                    <div class="header-text">
                        <h3>Containers</h3>
                        <div class="value">{{ systemInfo?.ContainersRunning || 0 }} / {{ systemInfo?.Containers || 0 }}
                        </div>
                    </div>
                </div>
                <div class="card-footer">
                    <span class="label">Running / Total</span>
                </div>
            </div>

            <div class="stat-card glass-panel">
                <div class="card-header">
                    <div class="icon-box emerald">
                        <Box :size="20" />
                    </div>
                    <div class="header-text">
                        <h3>Images</h3>
                        <div class="value">{{ systemInfo?.Images || 0 }}</div>
                    </div>
                </div>
                <div class="card-footer">
                    <span class="label">Local artifacts</span>
                </div>
            </div>

            <div class="stat-card glass-panel">
                <div class="card-header">
                    <div class="icon-box amber">
                        <HardDrive :size="20" />
                    </div>
                    <div class="header-text">
                        <h3>Volumes</h3>
                        <div class="value">{{ volumeCount }}</div>
                    </div>
                </div>
                <div class="card-footer">
                    <span class="label">Docker volumes</span>
                </div>
            </div>

            <div class="stat-card glass-panel">
                <div class="card-header">
                    <div class="icon-box cyan">
                        <Network :size="20" />
                    </div>
                    <div class="header-text">
                        <h3>Networks</h3>
                        <div class="value">{{ networkCount }}</div>
                    </div>
                </div>
                <div class="card-footer">
                    <span class="label">Docker networks</span>
                </div>
            </div>
        </div>

        <div class="charts-section">
            <div class="status-card glass-panel">
                <div class="status-title">Status</div>
                <div class="gauge-grid">
                    <div v-for="g in gauges" :key="g.key" class="gauge-item">
                        <div class="gauge-ring" :style="{ '--p': `${g.percent}` }">
                            <div class="gauge-inner">
                                <div class="gauge-value">
                                    {{ g.value }}<span class="unit">{{ g.unit }}</span>
                                </div>
                                <div class="gauge-label">{{ g.label }}</div>
                            </div>
                        </div>
                        <div class="gauge-detail">{{ g.detail }}</div>
                    </div>
                </div>
            </div>

            <div class="chart-card glass-panel">
                <div class="chart-header">
                    <div class="title">
                        <TrendingUp :size="18" class="text-primary" />
                        <span>Real-time Performance</span>
                    </div>
                    <div class="live-indicator">
                        <span class="pulse"></span>
                        LIVE
                    </div>
                </div>
                <div class="chart-container">
                    <Line :data="chartData" :options="chartOptions" />
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
.dashboard-view {
    display: flex;
    flex-direction: column;
    gap: 24px;
}

.stats-overview {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 24px;
}

.stat-card {
    padding: 24px;
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.card-header {
    display: flex;
    align-items: center;
    gap: 16px;
}

.icon-box {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
}

.icon-box.indigo {
    background: rgba(36, 150, 237, 0.12);
    color: #2496ed;
}

.icon-box.emerald {
    background: rgba(16, 185, 129, 0.1);
    color: #10b981;
}

.icon-box.amber {
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
}

.icon-box.cyan {
    background: rgba(14, 165, 233, 0.12);
    color: #0ea5e9;
}

.header-text h3 {
    font-size: 0.9rem;
    color: var(--text-muted);
    font-weight: 500;
    margin: 0;
}

.header-text .value {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-main);
}

.card-footer .label {
    font-size: 0.8rem;
    color: var(--text-muted);
}

.charts-section {
    display: grid;
    grid-template-columns: 1fr;
    gap: 24px;
}

.status-card {
    padding: 20px 24px;
}

.status-title {
    font-family: 'Outfit', sans-serif;
    font-size: 1.05rem;
    margin-bottom: 16px;
    color: var(--text-main);
}

.gauge-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 18px;
}

.gauge-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 10px;
}

.gauge-ring {
    --p: 0;
    width: 132px;
    height: 132px;
    border-radius: 50%;
    background:
        radial-gradient(circle at center, rgba(36, 43, 62, 1) 58%, transparent 60%),
        conic-gradient(#4a96ff calc(var(--p) * 1%), rgba(148, 163, 184, 0.18) 0);
    padding: 9px;
    display: flex;
    align-items: center;
    justify-content: center;
}

.gauge-inner {
    width: 100%;
    height: 100%;
    border-radius: 50%;
    background: var(--surface-elevated);
    border: 1px solid rgba(148, 163, 184, 0.22);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
}

.gauge-value {
    font-family: 'Outfit', sans-serif;
    font-size: 1.8rem;
    font-weight: 700;
    color: var(--text-main);
    line-height: 1;
}

.gauge-value .unit {
    font-size: 0.95rem;
    margin-left: 2px;
    color: var(--text-muted);
}

.gauge-label {
    font-size: 0.95rem;
    color: var(--text-main);
}

.gauge-detail {
    color: #93c5fd;
    font-size: 0.85rem;
    text-align: center;
}

.chart-card {
    padding: 24px;
    height: 400px;
    display: flex;
    flex-direction: column;
}

.chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
}

.chart-header .title {
    display: flex;
    align-items: center;
    gap: 10px;
    font-family: 'Outfit', sans-serif;
    font-weight: 600;
    font-size: 1.1rem;
}

.live-indicator {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--text-muted);
    letter-spacing: 1px;
}

.pulse {
    width: 6px;
    height: 6px;
    background: var(--danger);
    border-radius: 50%;
    animation: pulse-red 2s infinite;
}

@keyframes pulse-red {
    0% {
        box-shadow: 0 0 0 0 rgba(239, 68, 68, 0.7);
    }

    70% {
        box-shadow: 0 0 0 6px rgba(239, 68, 68, 0);
    }

    100% {
        box-shadow: 0 0 0 0 rgba(239, 68, 68, 0);
    }
}

.chart-container {
    flex-grow: 1;
    position: relative;
}

.text-primary {
    color: var(--primary);
}
</style>
