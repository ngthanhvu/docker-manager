<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import {
    Container,
    Box,
    HardDrive,
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
    diskUsage?: { totalBytes?: number; usedBytes?: number } | null;
}>();

const cpuData = ref<number[]>([]);
const memData = ref<number[]>([]);
const labels = ref<string[]>([]);
const maxDataPoints = 20;

const toNumber = (value: unknown) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
};

const parseBytes = (raw: string | undefined) => {
    if (!raw) return 0;
    const normalized = raw.trim().replace(/,/g, '');
    const match = normalized.match(/^([\d.]+)\s*([kmgtp]?i?b?)?$/i);
    if (!match) return 0;

    const value = toNumber(match[1]);
    const unit = (match[2] || 'b').toLowerCase();
    const multipliers: Record<string, number> = {
        b: 1,
        kb: 1000,
        mb: 1000 ** 2,
        gb: 1000 ** 3,
        tb: 1000 ** 4,
        pb: 1000 ** 5,
        kib: 1024,
        mib: 1024 ** 2,
        gib: 1024 ** 3,
        tib: 1024 ** 4,
        pib: 1024 ** 5
    };

    return value * (multipliers[unit] || 1);
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

const getDriverStatusValue = (key: string) => {
    const status = props.systemInfo?.DriverStatus;
    if (!Array.isArray(status)) return '';
    const pair = status.find((entry: unknown) => Array.isArray(entry) && String(entry[0]) === key) as string[] | undefined;
    return pair?.[1] || '';
};

const cpuPercent = computed(() => toNumber((cpuData.value[cpuData.value.length - 1] ?? 0).toFixed(2)));
const memPercent = computed(() => toNumber((memData.value[memData.value.length - 1] ?? 0).toFixed(2)));
const ncpu = computed(() => Math.max(1, toNumber(props.systemInfo?.NCPU || 1)));
const loadAvg = computed(() => toNumber(((cpuPercent.value / 100) * ncpu.value).toFixed(2)));
const loadPercent = computed(() => Math.min(100, toNumber(((loadAvg.value / ncpu.value) * 100).toFixed(2))));

const memTotalBytes = computed(() => toNumber(props.systemInfo?.MemTotal || 0));
const memUsedBytes = computed(() => toNumber((memTotalBytes.value * (memPercent.value / 100)).toFixed(0)));

const diskUsedBytes = computed(() => {
    const fromApi = toNumber(props.diskUsage?.usedBytes || 0);
    if (fromApi > 0) return fromApi;
    return parseBytes(getDriverStatusValue('Data Space Used'));
});
const diskTotalBytes = computed(() => {
    const fromApi = toNumber(props.diskUsage?.totalBytes || 0);
    if (fromApi > 0) return fromApi;
    return parseBytes(getDriverStatusValue('Data Space Total'));
});
const diskPercent = computed(() => {
    if (!diskTotalBytes.value) return 0;
    return Math.min(100, toNumber(((diskUsedBytes.value / diskTotalBytes.value) * 100).toFixed(2)));
});

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
    },
    {
        key: 'disk',
        label: 'Disk',
        percent: diskPercent.value,
        value: diskTotalBytes.value > 0 ? `${diskPercent.value.toFixed(2)}` : 'N/A',
        unit: diskTotalBytes.value > 0 ? '%' : '',
        detail: diskTotalBytes.value > 0
            ? `${formatBytes(diskUsedBytes.value)} / ${formatBytes(diskTotalBytes.value)}`
            : `Used: ${formatBytes(diskUsedBytes.value)}`,
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
const updateCharts = () => {
    const now = new Date().toLocaleTimeString();
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
    interval = setInterval(updateCharts, 2000);
});

onUnmounted(() => {
    clearInterval(interval);
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
                        <h3>Resources</h3>
                        <div class="value">{{ systemInfo?.Volumes || 0 }} Vol / {{ systemInfo?.Networks || 0 }} Net
                        </div>
                    </div>
                </div>
                <div class="card-footer">
                    <span class="label">Storage & Networking</span>
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
    background: #343b50;
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
    color: #e2e8f0;
    line-height: 1;
}

.gauge-value .unit {
    font-size: 0.95rem;
    margin-left: 2px;
    color: #94a3b8;
}

.gauge-label {
    font-size: 0.95rem;
    color: #cbd5e1;
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
