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
}>();

const cpuData = ref<number[]>([]);
const memData = ref<number[]>([]);
const labels = ref<string[]>([]);
const maxDataPoints = 20;

const chartData = computed(() => ({
    labels: [...labels.value],
    datasets: [
        {
            label: 'CPU Usage (%)',
            backgroundColor: 'rgba(99, 102, 241, 0.2)',
            borderColor: '#6366f1',
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
    background: rgba(99, 102, 241, 0.1);
    color: #6366f1;
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
