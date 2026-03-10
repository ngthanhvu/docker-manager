<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import {
    Container,
    Box,
    HardDrive,
    Network,
    Activity,
    ChevronDown
} from 'lucide-vue-next';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
    GridComponent,
    TooltipComponent,
    DataZoomComponent,
    LegendComponent
} from 'echarts/components';
import { appSettings } from '../ui/settings';

use([
    CanvasRenderer,
    LineChart,
    GridComponent,
    TooltipComponent,
    DataZoomComponent,
    LegendComponent,
]);

const props = defineProps<{
    systemInfo: any;
    resourceCounts?: {
        volumes?: number;
        networks?: number;
    } | null;
}>();

type MonitorMode = 'network' | 'disk';

const labels = ref<string[]>([]);
const cpuData = ref<number[]>([]);
const memData = ref<number[]>([]);
const netUpData = ref<number[]>([]);
const netDownData = ref<number[]>([]);
const diskReadData = ref<number[]>([]);
const diskWriteData = ref<number[]>([]);
const maxDataPoints = 36;
const monitorMode = ref<MonitorMode>('network');
const networkCard = ref('all');

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

const formatRate = (value: number) =>
    value >= 1024 ? `${(value / 1024).toFixed(2)} MB/s` : `${value.toFixed(value >= 10 ? 1 : 2)} KB/s`;

const sum = (values: number[]) => values.reduce((total, value) => total + value, 0);

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

const metricSeries = computed(() => {
    if (monitorMode.value === 'network') {
        return {
            unit: 'KB/s',
            leftLabel: 'Up',
            rightLabel: 'Down',
            leftColor: '#31d889',
            rightColor: '#d7ab46',
            leftAreaStart: 'rgba(49, 216, 137, 0.12)',
            leftAreaEnd: 'rgba(49, 216, 137, 0.01)',
            rightAreaStart: 'rgba(61, 125, 255, 0.55)',
            rightAreaEnd: 'rgba(61, 125, 255, 0.08)',
            leftData: netUpData.value,
            rightData: netDownData.value,
        };
    }

    return {
        unit: 'KB/s',
        leftLabel: 'Read',
        rightLabel: 'Write',
        leftColor: '#58c5ff',
        rightColor: '#a77bff',
        leftAreaStart: 'rgba(88, 197, 255, 0.22)',
        leftAreaEnd: 'rgba(88, 197, 255, 0.05)',
        rightAreaStart: 'rgba(167, 123, 255, 0.3)',
        rightAreaEnd: 'rgba(167, 123, 255, 0.07)',
        leftData: diskReadData.value,
        rightData: diskWriteData.value,
    };
});

const latestLeftValue = computed(() => metricSeries.value.leftData[metricSeries.value.leftData.length - 1] ?? 0);
const latestRightValue = computed(() => metricSeries.value.rightData[metricSeries.value.rightData.length - 1] ?? 0);
const totalLeftValue = computed(() => sum(metricSeries.value.leftData));
const totalRightValue = computed(() => sum(metricSeries.value.rightData));
const maxMetricValue = computed(() => {
    if (monitorMode.value === 'network') return 120;
    if (monitorMode.value === 'disk') return 100;
    const peak = Math.max(40, ...metricSeries.value.leftData, ...metricSeries.value.rightData, 0);
    return Math.ceil(peak / 20) * 20;
});

const monitoringPills = computed(() => {
    if (monitorMode.value === 'network') {
        return [
            { label: 'Up', value: formatRate(latestLeftValue.value) },
            { label: 'Down', value: formatRate(latestRightValue.value) },
            { label: 'Total sent', value: formatBytes(totalLeftValue.value * 1024) },
            { label: 'Total received', value: formatBytes(totalRightValue.value * 1024) },
        ];
    }

    return [
        { label: 'Read', value: formatRate(latestLeftValue.value) },
        { label: 'Write', value: formatRate(latestRightValue.value) },
        { label: 'Total read', value: formatBytes(totalLeftValue.value * 1024) },
        { label: 'Total write', value: formatBytes(totalRightValue.value * 1024) },
    ];
});

const monitoringLegend = computed(() => [
    { label: metricSeries.value.leftLabel, color: metricSeries.value.leftColor },
    { label: metricSeries.value.rightLabel, color: metricSeries.value.rightColor },
]);

const monitoringChartOption = computed(() => ({
    animation: true,
    animationDuration: 280,
    animationDurationUpdate: 280,
    backgroundColor: 'transparent',
    textStyle: {
        color: 'rgba(226, 232, 240, 0.68)',
        fontFamily: 'Inter, sans-serif',
    },
    grid: {
        left: 58,
        right: 26,
        top: 28,
        bottom: 92,
        containLabel: false,
    },
    tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(74, 80, 98, 0.82)',
        borderColor: 'rgba(255, 255, 255, 0.08)',
        borderWidth: 1,
        textStyle: {
            color: '#edf2f7',
        },
        extraCssText: 'backdrop-filter: blur(8px); border-radius: 10px; box-shadow: 0 10px 30px rgba(15,23,42,0.28);',
        axisPointer: {
            type: 'line',
            lineStyle: {
                color: 'rgba(240, 245, 255, 0.72)',
                type: 'dashed',
            },
        },
        valueFormatter: (value: number) => formatRate(Number(value || 0)),
    },
    xAxis: {
        type: 'category',
        boundaryGap: false,
        data: labels.value,
        axisLine: {
            lineStyle: {
                color: 'rgba(255, 255, 255, 0.06)',
            },
        },
        axisTick: {
            show: false,
        },
        axisLabel: {
            color: 'rgba(203, 213, 225, 0.46)',
            margin: 12,
            interval: Math.max(0, Math.floor(labels.value.length / 8) - 1),
        },
        splitLine: {
            show: false,
        },
    },
    yAxis: {
        type: 'value',
        min: 0,
        max: maxMetricValue.value,
        name: `(${metricSeries.value.unit})`,
        nameTextStyle: {
            color: 'rgba(203, 213, 225, 0.42)',
            padding: [0, 0, 8, -6],
        },
        axisLabel: {
            color: 'rgba(203, 213, 225, 0.58)',
            margin: 10,
        },
        axisLine: {
            show: false,
        },
        axisTick: {
            show: false,
        },
        splitLine: {
            lineStyle: {
                color: 'rgba(255, 255, 255, 0.08)',
                type: 'dashed',
            },
        },
    },
    dataZoom: [
        {
            type: 'slider',
            height: 34,
            bottom: 18,
            left: 34,
            right: 34,
            borderColor: 'rgba(255, 255, 255, 0.12)',
            backgroundColor: 'rgba(160, 174, 255, 0.08)',
            fillerColor: 'rgba(228, 232, 245, 0.18)',
            dataBackground: {
                lineStyle: {
                    color: 'rgba(146, 179, 255, 0.56)',
                },
                areaStyle: {
                    color: 'rgba(109, 132, 199, 0.18)',
                },
            },
            selectedDataBackground: {
                lineStyle: {
                    color: 'rgba(244, 247, 255, 0.92)',
                },
                areaStyle: {
                    color: 'rgba(163, 177, 255, 0.14)',
                },
            },
            handleSize: 20,
            handleStyle: {
                color: '#eef2ff',
                borderColor: '#eef2ff',
                shadowBlur: 0,
            },
            moveHandleStyle: {
                color: 'rgba(255, 255, 255, 0.8)',
            },
            textStyle: {
                color: 'transparent',
            },
            brushSelect: false,
            startValue: Math.max(0, labels.value.length - 12),
            endValue: Math.max(0, labels.value.length - 1),
        },
        {
            type: 'inside',
            startValue: Math.max(0, labels.value.length - 12),
            endValue: Math.max(0, labels.value.length - 1),
        }
    ],
    series: [
        {
            name: metricSeries.value.leftLabel,
            type: 'line',
            smooth: false,
            showSymbol: false,
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: {
                width: 2.2,
                color: metricSeries.value.leftColor,
            },
            itemStyle: {
                color: metricSeries.value.leftColor,
                borderColor: '#f8fafc',
                borderWidth: 2,
            },
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0,
                    y: 0,
                    x2: 0,
                    y2: 1,
                    colorStops: [
                        { offset: 0, color: metricSeries.value.leftAreaStart },
                        { offset: 1, color: metricSeries.value.leftAreaEnd },
                    ],
                },
            },
            emphasis: {
                focus: 'series',
            },
            data: metricSeries.value.leftData,
            z: 3,
        },
        {
            name: metricSeries.value.rightLabel,
            type: 'line',
            smooth: false,
            showSymbol: false,
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: {
                width: 2.2,
                color: metricSeries.value.rightColor,
            },
            itemStyle: {
                color: metricSeries.value.rightColor,
                borderColor: '#f8fafc',
                borderWidth: 2,
            },
            areaStyle: {
                color: {
                    type: 'linear',
                    x: 0,
                    y: 0,
                    x2: 0,
                    y2: 1,
                    colorStops: [
                        { offset: 0, color: metricSeries.value.rightAreaStart },
                        { offset: 1, color: metricSeries.value.rightAreaEnd },
                    ],
                },
            },
            emphasis: {
                focus: 'series',
            },
            data: metricSeries.value.rightData,
            z: 2,
        }
    ],
}));

const sharpSeriesPoint = (idleMax: number, spikeMin: number, spikeMax: number, spikeChance: number) => {
    if (Math.random() < spikeChance) {
        return spikeMin + Math.random() * (spikeMax - spikeMin);
    }
    return Math.random() * idleMax;
};

const networkTrafficPoint = () => {
    const majorSpike = Math.random() < 0.16;
    const down = majorSpike
        ? 18 + Math.random() * 98
        : sharpSeriesPoint(1.6, 8, 28, 0.2);
    const up = majorSpike
        ? down * (0.58 + Math.random() * 0.24)
        : sharpSeriesPoint(1.2, 6, 18, 0.18);
    return {
        up: Number(up.toFixed(2)),
        down: Number(down.toFixed(2)),
    };
};

const diskTrafficPoint = () => {
    const majorSpike = Math.random() < 0.14;
    const read = majorSpike
        ? 16 + Math.random() * 72
        : sharpSeriesPoint(2.2, 7, 22, 0.18);
    const write = majorSpike
        ? read * (0.42 + Math.random() * 0.35)
        : sharpSeriesPoint(1.8, 5, 18, 0.16);
    return {
        read: Number(read.toFixed(2)),
        write: Number(write.toFixed(2)),
    };
};

let interval: number | null = null;
const setupInterval = () => {
    if (interval) window.clearInterval(interval);
    const ms = appSettings.general.autoRefreshMs;
    if (ms > 0) {
        interval = window.setInterval(updateCharts, ms);
    }
};

const updateCharts = () => {
    const now = new Date().toLocaleTimeString([], {
        hour12: appSettings.general.timeFormat === '12h',
    });

    labels.value.push(now);
    cpuData.value.push(Math.random() * 30 + 10);
    memData.value.push(Math.random() * 20 + 40);
    const networkPoint = networkTrafficPoint();
    const diskPoint = diskTrafficPoint();
    netUpData.value.push(networkPoint.up);
    netDownData.value.push(networkPoint.down);
    diskReadData.value.push(diskPoint.read);
    diskWriteData.value.push(diskPoint.write);

    if (labels.value.length > maxDataPoints) {
        labels.value.shift();
        cpuData.value.shift();
        memData.value.shift();
        netUpData.value.shift();
        netDownData.value.shift();
        diskReadData.value.shift();
        diskWriteData.value.shift();
    }
};

onMounted(() => {
    updateCharts();
    setupInterval();
});

onUnmounted(() => {
    if (interval) window.clearInterval(interval);
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
                        <div class="value">{{ systemInfo?.ContainersRunning || 0 }} / {{ systemInfo?.Containers || 0 }}</div>
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

            <div class="monitoring-card">
                <div class="monitoring-topbar">
                    <div class="monitoring-title">
                        <Activity :size="17" />
                        <span>Monitoring</span>
                    </div>

                    <div class="monitoring-controls">
                        <div class="monitor-tabs">
                            <button class="monitor-tab" :class="{ active: monitorMode === 'network' }" @click="monitorMode = 'network'">
                                Network
                            </button>
                            <button class="monitor-tab" :class="{ active: monitorMode === 'disk' }" @click="monitorMode = 'disk'">
                                Disk I/O
                            </button>
                        </div>

                        <label class="monitor-select">
                            <span>{{ monitorMode === 'network' ? 'Network card' : 'Disk group' }}</span>
                            <select v-model="networkCard">
                                <option value="all">All</option>
                            </select>
                            <ChevronDown :size="16" class="select-icon" />
                        </label>
                    </div>
                </div>

                <div class="monitoring-pills">
                    <div v-for="pill in monitoringPills" :key="pill.label" class="metric-pill">
                        <span class="metric-label">{{ pill.label }}:</span>
                        <strong>{{ pill.value }}</strong>
                    </div>
                </div>

                <div class="legend-strip">
                    <span v-for="item in monitoringLegend" :key="item.label" class="legend-item">
                        <span class="legend-dot" :style="{ background: item.color }"></span>
                        {{ item.label }}
                    </span>
                </div>

                <VChart class="monitoring-chart" :option="monitoringChartOption" autoresize />
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

.monitoring-card {
    overflow: hidden;
    border-radius: 26px;
    border: 1px solid rgba(148, 163, 184, 0.14);
    background:
        radial-gradient(circle at top left, rgba(59, 130, 246, 0.14), transparent 28%),
        linear-gradient(180deg, #1f2438 0%, #1b2134 100%);
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.03), 0 18px 40px rgba(3, 7, 18, 0.28);
    padding: 22px 22px 12px;
}

.monitoring-topbar {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 16px;
    margin-bottom: 18px;
}

.monitoring-title {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    color: #d8e1f5;
    font-family: 'Outfit', sans-serif;
    font-size: 1.1rem;
}

.monitoring-title svg {
    color: #5ea0ff;
}

.monitoring-controls {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    justify-content: flex-end;
}

.monitor-tabs {
    display: inline-flex;
    border: 1px solid rgba(148, 163, 184, 0.18);
    border-radius: 10px;
    overflow: hidden;
    background: rgba(255, 255, 255, 0.02);
}

.monitor-tab {
    border: none;
    background: transparent;
    color: rgba(226, 232, 240, 0.72);
    padding: 10px 16px;
    font-size: 0.9rem;
    cursor: pointer;
    transition: 0.18s ease;
}

.monitor-tab:hover {
    background: rgba(94, 160, 255, 0.08);
    color: #f8fafc;
}

.monitor-tab.active {
    background: linear-gradient(180deg, #4b94ff 0%, #2f79f2 100%);
    color: #eff6ff;
}

.monitor-select {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 0 38px 0 12px;
    min-height: 42px;
    border-radius: 10px;
    border: 1px solid rgba(148, 163, 184, 0.18);
    background: rgba(255, 255, 255, 0.03);
    color: rgba(226, 232, 240, 0.78);
    font-size: 0.88rem;
}

.monitor-select span {
    color: rgba(203, 213, 225, 0.56);
}

.monitor-select select {
    appearance: none;
    border: none;
    outline: none;
    background: transparent;
    color: #f8fafc;
    font-size: 0.9rem;
    padding-right: 10px;
}

.select-icon {
    position: absolute;
    right: 12px;
    color: rgba(203, 213, 225, 0.56);
    pointer-events: none;
}

.monitoring-pills {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    margin-bottom: 14px;
}

.metric-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 7px 12px;
    border-radius: 8px;
    border: 1px solid rgba(125, 161, 224, 0.28);
    background: rgba(30, 39, 62, 0.74);
    color: #76a8ff;
    font-size: 0.85rem;
}

.metric-pill strong {
    color: #cde0ff;
    font-weight: 600;
}

.metric-label {
    color: #6ea3ff;
}

.legend-strip {
    display: flex;
    justify-content: flex-end;
    align-items: center;
    gap: 16px;
    color: rgba(226, 232, 240, 0.46);
    font-size: 0.86rem;
    margin-bottom: 8px;
}

.legend-item {
    display: inline-flex;
    align-items: center;
    gap: 7px;
}

.legend-dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    box-shadow: 0 0 12px currentColor;
}

.monitoring-chart {
    height: 480px;
    width: 100%;
}

@media (max-width: 980px) {
    .monitoring-topbar {
        flex-direction: column;
        align-items: stretch;
    }

    .monitoring-controls {
        justify-content: flex-start;
    }

    .legend-strip {
        justify-content: flex-start;
    }
}

@media (max-width: 720px) {
    .stats-overview {
        grid-template-columns: 1fr;
    }

    .gauge-grid {
        grid-template-columns: 1fr;
    }

    .monitor-tabs {
        width: 100%;
    }

    .monitor-tab {
        flex: 1;
    }

    .monitor-select {
        width: 100%;
        justify-content: space-between;
    }

    .monitoring-card {
        padding: 18px 16px 10px;
    }

    .monitoring-chart {
        height: 420px;
    }
}
</style>
