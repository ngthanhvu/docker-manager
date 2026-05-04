<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  Container,
  Box,
  HardDrive,
  Network,
  Activity,
  ChevronDown,
} from 'lucide-vue-next';
import VChart from 'vue-echarts';
import { use } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { LineChart } from 'echarts/charts';
import {
  GridComponent,
  TooltipComponent,
  DataZoomComponent,
  LegendComponent,
} from 'echarts/components';
import { appSettings } from '../ui/settings';
import { dockerApi } from '../api';
const { t } = useI18n();

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
type DashboardNetworkMetric = {
  rxBytes: number;
  txBytes: number;
  rxRateBytes: number;
  txRateBytes: number;
};

type DashboardDiskMetric = {
  readBytes: number;
  writeBytes: number;
  readRateBytes: number;
  writeRateBytes: number;
};

type DashboardMetricPoint = {
  timestamp: string;
  cpuPercent: number;
  memoryPercent: number;
  memoryUsedBytes: number;
  memoryTotalBytes: number;
  networks: Record<string, DashboardNetworkMetric>;
  disk: DashboardDiskMetric;
};

const metricPoints = ref<DashboardMetricPoint[]>([]);
const networkInterfaces = ref<string[]>([]);
const monitorMode = ref<MonitorMode>('network');
const networkCard = ref('all');
let interval: number | null = null;

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

const activeMetricPoints = computed(() => metricPoints.value);
const activeNetworkInterfaces = computed(() => networkInterfaces.value);

const labels = computed(() => activeMetricPoints.value.map((point) => new Date(point.timestamp).toLocaleTimeString([], {
  hour12: appSettings.general.timeFormat === '12h',
})));
const cpuData = computed(() => activeMetricPoints.value.map((point) => toNumber(point.cpuPercent.toFixed(2))));
const memData = computed(() => activeMetricPoints.value.map((point) => toNumber(point.memoryPercent.toFixed(2))));
const cpuPercent = computed(() => toNumber((cpuData.value[cpuData.value.length - 1] ?? 0).toFixed(2)));
const memPercent = computed(() => toNumber((memData.value[memData.value.length - 1] ?? 0).toFixed(2)));
const ncpu = computed(() => Math.max(1, toNumber(props.systemInfo?.NCPU || 1)));
const loadAvg = computed(() => toNumber(((cpuPercent.value / 100) * ncpu.value).toFixed(2)));
const loadPercent = computed(() => Math.min(100, toNumber(((loadAvg.value / ncpu.value) * 100).toFixed(2))));

const latestMetricPoint = computed(() => activeMetricPoints.value[activeMetricPoints.value.length - 1] || null);
const memTotalBytes = computed(() => toNumber(latestMetricPoint.value?.memoryTotalBytes ?? props.systemInfo?.MemTotal ?? 0));
const memUsedBytes = computed(() => toNumber(latestMetricPoint.value?.memoryUsedBytes ?? ((memTotalBytes.value * memPercent.value) / 100)));

const volumeCount = computed(() => toNumber(props.resourceCounts?.volumes ?? props.systemInfo?.Volumes ?? 0));
const networkCount = computed(() => toNumber(props.resourceCounts?.networks ?? props.systemInfo?.Networks ?? 0));

const availableNetworkCards = computed(() => ['all', ...activeNetworkInterfaces.value]);

const selectedNetworkSeries = computed(() => activeMetricPoints.value.map((point) => {
  const entries = Object.entries(point.networks || {});
  if (networkCard.value === 'all') {
    return entries.reduce((acc, [, metric]) => ({
      upRateBytes: acc.upRateBytes + toNumber(metric.txRateBytes),
      downRateBytes: acc.downRateBytes + toNumber(metric.rxRateBytes),
      upBytes: acc.upBytes + toNumber(metric.txBytes),
      downBytes: acc.downBytes + toNumber(metric.rxBytes),
    }), {
      upRateBytes: 0,
      downRateBytes: 0,
      upBytes: 0,
      downBytes: 0,
    });
  }

  const metric = point.networks?.[networkCard.value];
  return {
    upRateBytes: toNumber(metric?.txRateBytes ?? 0),
    downRateBytes: toNumber(metric?.rxRateBytes ?? 0),
    upBytes: toNumber(metric?.txBytes ?? 0),
    downBytes: toNumber(metric?.rxBytes ?? 0),
  };
}));

const netUpData = computed(() => selectedNetworkSeries.value.map((point) => toNumber((point.upRateBytes / 1024).toFixed(2))));
const netDownData = computed(() => selectedNetworkSeries.value.map((point) => toNumber((point.downRateBytes / 1024).toFixed(2))));
const diskReadData = computed(() => activeMetricPoints.value.map((point) => toNumber((toNumber(point.disk?.readRateBytes) / 1024).toFixed(2))));
const diskWriteData = computed(() => activeMetricPoints.value.map((point) => toNumber((toNumber(point.disk?.writeRateBytes) / 1024).toFixed(2))));

const gauges = computed(() => ([
  {
    key: 'load',
    label: t('dashboard.load'),
    percent: loadPercent.value,
    value: `${loadAvg.value}`,
    unit: '',
    detail: `(${loadAvg.value} / ${ncpu.value}) cores`,
  },
  {
    key: 'cpu',
    label: t('dashboard.cpu'),
    percent: cpuPercent.value,
    value: `${cpuPercent.value}`,
    unit: '%',
    detail: `${ncpu.value} cores`,
  },
  {
    key: 'memory',
    label: t('dashboard.memory'),
    percent: memPercent.value,
    value: `${memPercent.value}`,
    unit: '%',
    detail: `${formatBytes(memUsedBytes.value)} / ${formatBytes(memTotalBytes.value)}`,
  },
]));

const metricSeries = computed(() => {
  if (monitorMode.value === 'network') {
    return {
      unit: 'KB/s',
      leftLabel: t('dashboard.up'),
      rightLabel: t('dashboard.down'),
      leftColor: '#3ddc84',
      rightColor: '#f4b942',
      leftAreaStart: 'rgba(61, 220, 132, 0.22)',
      leftAreaEnd: 'rgba(61, 220, 132, 0.01)',
      rightAreaStart: 'rgba(244, 185, 66, 0.24)',
      rightAreaEnd: 'rgba(244, 185, 66, 0.03)',
      leftData: netUpData.value,
      rightData: netDownData.value,
    };
  }

  return {
    unit: 'KB/s',
    leftLabel: t('dashboard.read'),
    rightLabel: t('dashboard.write'),
    leftColor: '#58a6ff',
    rightColor: '#ff7a59',
    leftAreaStart: 'rgba(88, 166, 255, 0.22)',
    leftAreaEnd: 'rgba(88, 166, 255, 0.05)',
    rightAreaStart: 'rgba(255, 122, 89, 0.22)',
    rightAreaEnd: 'rgba(255, 122, 89, 0.04)',
    leftData: diskReadData.value,
    rightData: diskWriteData.value,
  };
});

const latestLeftValue = computed(() => metricSeries.value.leftData[metricSeries.value.leftData.length - 1] ?? 0);
const latestRightValue = computed(() => metricSeries.value.rightData[metricSeries.value.rightData.length - 1] ?? 0);
const totalLeftValue = computed(() => {
  if (monitorMode.value === 'network') {
    return toNumber(selectedNetworkSeries.value[selectedNetworkSeries.value.length - 1]?.upBytes ?? 0);
  }
  return toNumber(latestMetricPoint.value?.disk?.readBytes ?? 0);
});
const totalRightValue = computed(() => {
  if (monitorMode.value === 'network') {
    return toNumber(selectedNetworkSeries.value[selectedNetworkSeries.value.length - 1]?.downBytes ?? 0);
  }
  return toNumber(latestMetricPoint.value?.disk?.writeBytes ?? 0);
});
const maxMetricValue = computed(() => {
  const peak = Math.max(40, ...metricSeries.value.leftData, ...metricSeries.value.rightData, 0);
  if (monitorMode.value === 'network') return Math.max(120, Math.ceil(peak / 20) * 20);
  if (monitorMode.value === 'disk') return Math.max(100, Math.ceil(peak / 20) * 20);
  return Math.ceil(peak / 20) * 20;
});

const monitoringPills = computed(() => {
  if (monitorMode.value === 'network') {
    return [
      { label: t('dashboard.up'), value: formatRate(latestLeftValue.value) },
      { label: t('dashboard.down'), value: formatRate(latestRightValue.value) },
      { label: t('dashboard.totalSent'), value: formatBytes(totalLeftValue.value) },
      { label: t('dashboard.totalReceived'), value: formatBytes(totalRightValue.value) },
    ];
  }

  return [
    { label: t('dashboard.read'), value: formatRate(latestLeftValue.value) },
    { label: t('dashboard.write'), value: formatRate(latestRightValue.value) },
    { label: t('dashboard.totalRead'), value: formatBytes(totalLeftValue.value) },
    { label: t('dashboard.totalWrite'), value: formatBytes(totalRightValue.value) },
  ];
});

const monitoringLegend = computed(() => [
  { label: metricSeries.value.leftLabel, color: metricSeries.value.leftColor },
  { label: metricSeries.value.rightLabel, color: metricSeries.value.rightColor },
]);

const chartTheme = computed(() => {
  if (appSettings.ui.theme === 'light') {
    return {
      text: 'rgba(22, 22, 22, 0.74)',
      axisText: 'rgba(22, 22, 22, 0.6)',
      axisName: 'rgba(22, 22, 22, 0.52)',
      axisLine: 'rgba(22, 22, 22, 0.18)',
      splitLine: 'rgba(22, 22, 22, 0.12)',
      tooltipBg: 'rgba(247, 243, 234, 0.98)',
      tooltipBorder: 'rgba(22, 22, 22, 0.14)',
      tooltipText: '#161616',
      pointerLine: 'rgba(22, 22, 22, 0.4)',
      zoomBorder: 'rgba(22, 22, 22, 0.18)',
      zoomBg: 'rgba(22, 22, 22, 0.05)',
      zoomFill: 'rgba(29, 78, 216, 0.14)',
      zoomDataLine: 'rgba(22, 22, 22, 0.34)',
      zoomDataArea: 'rgba(22, 22, 22, 0.06)',
      zoomSelectedLine: 'rgba(22, 22, 22, 0.72)',
      zoomSelectedArea: 'rgba(29, 78, 216, 0.08)',
      zoomHandle: '#1d4ed8',
      zoomHandleMove: 'rgba(29, 78, 216, 0.74)',
      pointBorder: '#f7f3ea',
      shadow: '6px 6px 0 rgba(22, 22, 22, 0.14)',
    };
  }

  return {
    text: 'rgba(244, 244, 240, 0.7)',
    axisText: 'rgba(244, 244, 240, 0.56)',
    axisName: 'rgba(244, 244, 240, 0.38)',
    axisLine: 'rgba(244, 244, 240, 0.12)',
    splitLine: 'rgba(244, 244, 240, 0.08)',
    tooltipBg: 'rgba(20, 20, 20, 0.96)',
    tooltipBorder: 'rgba(244, 244, 240, 0.12)',
    tooltipText: '#f4f4f0',
    pointerLine: 'rgba(244, 244, 240, 0.46)',
    zoomBorder: 'rgba(244, 244, 240, 0.12)',
    zoomBg: 'rgba(255, 255, 255, 0.05)',
    zoomFill: 'rgba(244, 244, 240, 0.14)',
    zoomDataLine: 'rgba(244, 244, 240, 0.4)',
    zoomDataArea: 'rgba(255, 255, 255, 0.08)',
    zoomSelectedLine: 'rgba(255, 255, 255, 0.9)',
    zoomSelectedArea: 'rgba(255, 255, 255, 0.08)',
    zoomHandle: '#f4f4f0',
    zoomHandleMove: 'rgba(255, 255, 255, 0.8)',
    pointBorder: '#f8fafc',
    shadow: '6px 6px 0 rgba(0,0,0,0.28)',
  };
});

const monitoringChartOption = computed(() => ({
  animation: true,
  animationDuration: 280,
  animationDurationUpdate: 280,
  backgroundColor: 'transparent',
  textStyle: {
    color: chartTheme.value.text,
    fontFamily: 'Space Grotesk, sans-serif',
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
    backgroundColor: chartTheme.value.tooltipBg,
    borderColor: chartTheme.value.tooltipBorder,
    borderWidth: 1,
    textStyle: {
      color: chartTheme.value.tooltipText,
    },
    extraCssText: `border-radius:0; box-shadow: ${chartTheme.value.shadow};`,
    axisPointer: {
      type: 'line',
      lineStyle: {
        color: chartTheme.value.pointerLine,
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
        color: chartTheme.value.axisLine,
      },
    },
    axisTick: {
      show: false,
    },
    axisLabel: {
      color: chartTheme.value.axisText,
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
      color: chartTheme.value.axisName,
      padding: [0, 0, 8, -6],
    },
    axisLabel: {
      color: chartTheme.value.axisText,
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
        color: chartTheme.value.splitLine,
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
      borderColor: chartTheme.value.zoomBorder,
      backgroundColor: chartTheme.value.zoomBg,
      fillerColor: chartTheme.value.zoomFill,
      dataBackground: {
        lineStyle: {
          color: chartTheme.value.zoomDataLine,
        },
        areaStyle: {
          color: chartTheme.value.zoomDataArea,
        },
      },
      selectedDataBackground: {
        lineStyle: {
          color: chartTheme.value.zoomSelectedLine,
        },
        areaStyle: {
          color: chartTheme.value.zoomSelectedArea,
        },
      },
      handleSize: 20,
      handleStyle: {
        color: chartTheme.value.zoomHandle,
        borderColor: chartTheme.value.zoomHandle,
        shadowBlur: 0,
      },
      moveHandleStyle: {
        color: chartTheme.value.zoomHandleMove,
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
    },
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
        borderColor: chartTheme.value.pointBorder,
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
        borderColor: chartTheme.value.pointBorder,
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
    },
  ],
}));

const fetchMetrics = async () => {
  try {
    const { data } = await dockerApi.getDashboardMetrics();
    metricPoints.value = Array.isArray(data?.points) ? data.points : [];
    networkInterfaces.value = Array.isArray(data?.interfaces) ? data.interfaces : [];
  } catch (err) {
    console.error('Failed to fetch dashboard metrics:', err);
    metricPoints.value = [];
    networkInterfaces.value = [];
  }
};

const setupInterval = () => {
  if (interval) window.clearInterval(interval);
  const ms = appSettings.general.autoRefreshMs;
  if (ms > 0) {
    interval = window.setInterval(fetchMetrics, ms);
  }
};

onMounted(async () => {
  await fetchMetrics();
  setupInterval();
});

onUnmounted(() => {
  if (interval) window.clearInterval(interval);
});

watch(() => appSettings.general.autoRefreshMs, () => {
  setupInterval();
});

watch(activeNetworkInterfaces, (interfaces) => {
  if (networkCard.value !== 'all' && !interfaces.includes(networkCard.value)) {
    networkCard.value = 'all';
  }
});
</script>

<template>
  <div class="flex flex-col gap-6">
    <div class="grid gap-4 xl:grid-cols-4">
      <div v-for="card in [
        { key: 'containers', label: t('dashboard.containers'), value: `${systemInfo?.ContainersRunning || 0} / ${systemInfo?.Containers || 0}`, detail: t('dashboard.runningTotal'), icon: Container, tone: 'var(--primary)' },
        { key: 'images', label: t('dashboard.images'), value: `${systemInfo?.Images || 0}`, detail: t('dashboard.localArtifacts'), icon: Box, tone: 'var(--success)' },
        { key: 'volumes', label: t('dashboard.volumes'), value: `${volumeCount}`, detail: t('dashboard.dockerVolumes'), icon: HardDrive, tone: 'var(--warning)' },
        { key: 'networks', label: t('dashboard.networks'), value: `${networkCount}`, detail: t('dashboard.dockerNetworks'), icon: Network, tone: '#58a6ff' },
      ]" :key="card.key" class="glass-panel p-5">
        <div class="mb-6 flex items-start justify-between gap-4">
          <div>
            <p class="mb-2 text-[11px] uppercase tracking-[0.22em]" style="color: var(--text-muted);">{{ card.label }}
            </p>
            <div class="text-3xl font-bold tracking-tight">{{ card.value }}</div>
          </div>
          <div class="grid h-11 w-11 place-items-center border"
            :style="{ borderColor: card.tone, color: card.tone, background: 'color-mix(in srgb, ' + card.tone + ' 12%, transparent)' }">
            <component :is="card.icon" :size="18" />
          </div>
        </div>
        <div class="border-t pt-3 text-sm" style="border-color: var(--glass-border); color: var(--text-muted);">
          {{ card.detail }}
        </div>
      </div>
    </div>

    <div class="grid gap-6 2xl:grid-cols-[360px_minmax(0,1fr)]">
      <section class="glass-panel p-5">
        <p class="section-heading">{{ t('dashboard.statusRings') }}</p>
        <div class="grid gap-5 md:grid-cols-3 2xl:grid-cols-1">
          <div v-for="g in gauges" :key="g.key" class="border p-4 text-center"
            style="border-color: var(--glass-border); background: var(--glass);">
            <div class="mx-auto grid h-32 w-32 place-items-center rounded-full border-10"
              :style="{ borderColor: 'rgba(255,255,255,0.08)', borderTopColor: 'var(--primary)', borderRightColor: g.key === 'memory' ? 'var(--warning)' : 'var(--primary)' }">
              <div class="text-center">
                <div class="text-3xl font-bold leading-none">
                  {{ g.value }}<span class="text-base" style="color: var(--text-muted);">{{ g.unit }}</span>
                </div>
                <div class="mt-2 text-sm font-semibold">{{ g.label }}</div>
              </div>
            </div>
            <p class="mt-4 text-sm leading-6" style="color: var(--text-muted);">{{ g.detail }}</p>
          </div>
        </div>
      </section>

      <section class="glass-panel p-5">
        <div class="mb-5 flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div>
            <div class="mb-2 flex items-center gap-2 text-lg font-semibold">
              <Activity :size="18" />
              {{ t('dashboard.monitoring') }}
            </div>
            <p class="text-sm leading-6" style="color: var(--text-muted);">
              {{ t('dashboard.monitoringDesc') }}
            </p>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap">
            <div class="inline-flex border" style="border-color: var(--glass-border);">
              <button class="px-4 py-2 text-sm font-semibold"
                :style="monitorMode === 'network' ? 'background: var(--primary); color: white;' : 'background: var(--glass); color: var(--text-muted);'"
                @click="monitorMode = 'network'">
                {{ t('dashboard.network') }}
              </button>
              <button class="px-4 py-2 text-sm font-semibold"
                :style="monitorMode === 'disk' ? 'background: var(--primary); color: white;' : 'background: var(--glass); color: var(--text-muted);'"
                @click="monitorMode = 'disk'">
                {{ t('dashboard.disk') }}
              </button>
            </div>

            <label v-if="monitorMode === 'network'" class="relative flex min-w-47.5 items-center border pr-10"
              style="border-color: var(--glass-border); background: var(--glass);">
              <span class="px-3 text-xs uppercase tracking-[0.2em]" style="color: var(--text-muted);">{{
                t('dashboard.iface') }}</span>
              <select v-model="networkCard"
                class="app-select border-0 bg-transparent py-2 pr-8 pl-0 shadow-none focus:shadow-none">
                <option v-for="option in availableNetworkCards" :key="option" :value="option">
                  {{ option === 'all' ? t('dashboard.all') : option }}
                </option>
              </select>
              <ChevronDown class="pointer-events-none absolute right-3" :size="16" style="color: var(--text-muted);" />
            </label>
          </div>
        </div>

        <div class="mb-4 flex flex-wrap gap-2">
          <div v-for="pill in monitoringPills" :key="pill.label" class="border px-3 py-2 text-sm"
            style="border-color: var(--glass-border); background: var(--glass);">
            <span style="color: var(--text-muted);">{{ pill.label }}:</span>
            <strong class="ml-2">{{ pill.value }}</strong>
          </div>
        </div>

        <div class="mb-3 flex flex-wrap gap-4 text-sm" style="color: var(--text-muted);">
          <span v-for="item in monitoringLegend" :key="item.label" class="inline-flex items-center gap-2">
            <span class="h-2.5 w-2.5" :style="{ background: item.color }"></span>
            {{ item.label }}
          </span>
        </div>

        <VChart class="h-115 w-full" :option="monitoringChartOption" autoresize />
      </section>
    </div>
  </div>
</template>
