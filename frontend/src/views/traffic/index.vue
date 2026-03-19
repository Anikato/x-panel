<template>
  <div class="traffic-page">
    <div class="page-header">
      <h3>{{ $t('traffic.title') }}</h3>
      <div class="header-actions">
        <el-button size="small" type="primary" :icon="Plus" @click="openConfigDialog()">
          {{ $t('traffic.addConfig') }}
        </el-button>
        <el-button size="small" :icon="Refresh" @click="loadAll" :loading="loading">
          {{ $t('commons.refresh') }}
        </el-button>
      </div>
    </div>

    <!-- Summary Cards with realtime speed -->
    <div class="summary-cards" v-if="summary.length > 0">
      <div
        v-for="item in summary"
        :key="item.interfaceName"
        class="summary-card"
        :class="{ 'is-disabled': !item.enabled }"
      >
        <div class="card-glow"></div>
        <div class="card-content">
          <div class="card-top">
            <div class="iface-badge">
              <el-icon :size="16"><Connection /></el-icon>
              <span>{{ item.interfaceName }}</span>
            </div>
            <div class="card-actions">
              <el-button text size="small" @click="openConfigDialog(item)">
                <el-icon><Setting /></el-icon>
              </el-button>
              <el-button text size="small" type="danger" @click="handleDelete(item.interfaceName)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
          </div>

          <!-- Realtime Speed -->
          <div class="realtime-speed">
            <div class="speed-item up">
              <span class="speed-arrow">↑</span>
              <span class="speed-value">{{ formatSpeed(getRealtimeSpeed(item.interfaceName, 'up')) }}</span>
            </div>
            <div class="speed-item down">
              <span class="speed-arrow">↓</span>
              <span class="speed-value">{{ formatSpeed(getRealtimeSpeed(item.interfaceName, 'down')) }}</span>
            </div>
          </div>

          <!-- Progress Ring -->
          <div class="progress-section">
            <div class="progress-ring-wrapper">
              <svg viewBox="0 0 120 120" class="progress-ring">
                <circle cx="60" cy="60" r="52" class="ring-bg" />
                <circle cx="60" cy="60" r="52"
                  class="ring-fill"
                  :style="ringStyle(item)"
                  :stroke="ringColor(item)"
                />
              </svg>
              <div class="ring-center">
                <span class="ring-pct" v-if="item.monthlyLimit > 0">{{ Math.round(item.usedPercent) }}%</span>
                <span class="ring-pct no-limit" v-else>∞</span>
                <span class="ring-label">{{ $t('traffic.used') }}</span>
              </div>
            </div>

            <div class="quota-info">
              <div class="quota-row">
                <span class="quota-label">{{ $t('traffic.monthlyQuota') }}</span>
                <span class="quota-value">{{ item.monthlyLimit > 0 ? formatBytes(item.monthlyLimit) : $t('traffic.unlimited') }}</span>
              </div>
              <div class="quota-row">
                <span class="quota-label">{{ $t('traffic.used') }}</span>
                <span class="quota-value highlight">{{ formatBytes(item.totalUsed) }}</span>
              </div>
              <div class="quota-row">
                <span class="quota-label">{{ $t('traffic.upload') }}</span>
                <span class="quota-value up-color">↑ {{ formatBytes(item.totalSent) }}</span>
              </div>
              <div class="quota-row">
                <span class="quota-label">{{ $t('traffic.download') }}</span>
                <span class="quota-value down-color">↓ {{ formatBytes(item.totalRecv) }}</span>
              </div>
              <div class="quota-row">
                <span class="quota-label">{{ $t('traffic.billingPeriod') }}</span>
                <span class="quota-value period">{{ formatDate(item.periodStart) }} ~ {{ formatDate(item.periodEnd) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-empty v-else-if="!loading" :description="$t('traffic.noConfig')">
      <el-button type="primary" @click="openConfigDialog()">{{ $t('traffic.addConfig') }}</el-button>
    </el-empty>

    <!-- Chart Section -->
    <div class="chart-card" v-if="summary.length > 0">
      <div class="chart-toolbar">
        <el-select v-model="selectedInterface" style="width: 150px" @change="loadStats" size="small">
          <el-option
            v-for="item in summary"
            :key="item.interfaceName"
            :label="item.interfaceName"
            :value="item.interfaceName"
          />
        </el-select>
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          :start-placeholder="$t('traffic.startDate')"
          :end-placeholder="$t('traffic.endDate')"
          value-format="YYYY-MM-DD"
          @change="loadStats"
          size="small"
          style="width: 260px"
        />
        <el-radio-group v-model="groupBy" @change="loadStats" size="small">
          <el-radio-button value="day">{{ $t('traffic.byDay') }}</el-radio-button>
          <el-radio-button value="hour">{{ $t('traffic.byHour') }}</el-radio-button>
        </el-radio-group>
      </div>

      <div ref="chartRef" class="chart-container"></div>

      <div class="stats-footer" v-if="statsTotalSent > 0 || statsTotalRecv > 0">
        <span>{{ $t('traffic.periodTotal') }}:</span>
        <span class="up-color">↑ {{ formatBytes(statsTotalSent) }}</span>
        <span class="down-color">↓ {{ formatBytes(statsTotalRecv) }}</span>
        <span class="total-badge">{{ formatBytes(statsTotalSent + statsTotalRecv) }}</span>
      </div>

      <!-- Data Table -->
      <el-table :data="statsItems" size="small" class="stats-table" max-height="320" stripe>
        <el-table-column prop="timestamp" :label="$t('traffic.time')" min-width="140" />
        <el-table-column :label="$t('traffic.upload')" min-width="120" align="right">
          <template #default="{ row }">
            <span class="up-color">{{ formatBytes(row.bytesSent) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('traffic.download')" min-width="120" align="right">
          <template #default="{ row }">
            <span class="down-color">{{ formatBytes(row.bytesRecv) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('traffic.total')" min-width="120" align="right">
          <template #default="{ row }">
            {{ formatBytes(row.bytesSent + row.bytesRecv) }}
          </template>
        </el-table-column>
      </el-table>
    </div>

    <ConfigDialog ref="configDialogRef" @refresh="loadAll" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, Setting, Delete, Connection } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { trafficApi } from '@/api/modules/traffic'
import type { TrafficSummaryItem, TrafficStatsItem, TrafficRealtimeItem } from '@/api/modules/traffic'
import ConfigDialog from './config-dialog.vue'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, CanvasRenderer])

const { t } = useI18n()
const loading = ref(false)
const summary = ref<TrafficSummaryItem[]>([])
const realtimeData = ref<TrafficRealtimeItem[]>([])
const configDialogRef = ref<InstanceType<typeof ConfigDialog>>()

const selectedInterface = ref('')
const dateRange = ref<[string, string] | null>(null)
const groupBy = ref<'day' | 'hour'>('day')
const statsItems = ref<TrafficStatsItem[]>([])
const statsTotalSent = ref(0)
const statsTotalRecv = ref(0)

const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
let realtimeTimer: ReturnType<typeof setInterval> | null = null

const RING_CIRCUMFERENCE = 2 * Math.PI * 52

const getRealtimeSpeed = (ifaceName: string, dir: 'up' | 'down') => {
  const item = realtimeData.value.find(r => r.name === ifaceName)
  if (!item) return 0
  return dir === 'up' ? item.speedUp : item.speedDown
}

const ringStyle = (item: TrafficSummaryItem) => {
  const pct = item.monthlyLimit > 0 ? Math.min(item.usedPercent, 100) / 100 : 0
  const offset = RING_CIRCUMFERENCE * (1 - pct)
  return {
    strokeDasharray: `${RING_CIRCUMFERENCE}`,
    strokeDashoffset: `${offset}`,
    transition: 'stroke-dashoffset 1s ease, stroke 0.5s ease',
  }
}

const ringColor = (item: TrafficSummaryItem) => {
  if (item.monthlyLimit === 0) return 'var(--xp-accent)'
  if (item.usedPercent >= 90) return '#ef4444'
  if (item.usedPercent >= 70) return '#f59e0b'
  return 'var(--xp-accent)'
}

const loadSummary = async () => {
  loading.value = true
  try {
    const res: any = await trafficApi.getSummary()
    summary.value = res.data || []
    if (summary.value.length > 0 && !selectedInterface.value) {
      selectedInterface.value = summary.value[0].interfaceName
      initDateRange()
    }
  } catch { /* handled */ }
  finally { loading.value = false }
}

const loadRealtime = async () => {
  try {
    const res: any = await trafficApi.getRealtime()
    realtimeData.value = res.data || []
  } catch { /* handled */ }
}

const initDateRange = () => {
  const item = summary.value.find(s => s.interfaceName === selectedInterface.value)
  if (item) {
    dateRange.value = [
      item.periodStart.substring(0, 10),
      item.periodEnd.substring(0, 10),
    ]
  }
}

const loadStats = async () => {
  if (!selectedInterface.value || !dateRange.value) return
  try {
    const res: any = await trafficApi.getStats({
      interfaceName: selectedInterface.value,
      startTime: dateRange.value[0],
      endTime: dateRange.value[1],
      groupBy: groupBy.value,
    })
    const data = res.data
    statsItems.value = data?.items || []
    statsTotalSent.value = data?.totalSent || 0
    statsTotalRecv.value = data?.totalRecv || 0
    await nextTick()
    renderChart()
  } catch { /* handled */ }
}

const loadAll = async () => {
  await loadSummary()
  await loadRealtime()
  if (selectedInterface.value) {
    await loadStats()
  }
}

const renderChart = () => {
  if (!chartRef.value) return
  if (!chart) {
    chart = echarts.init(chartRef.value)
  }

  const xData = statsItems.value.map(i => i.timestamp)
  const sentData = statsItems.value.map(i => i.bytesSent)
  const recvData = statsItems.value.map(i => i.bytesRecv)

  const accentColor = getComputedStyle(document.documentElement).getPropertyValue('--xp-accent').trim() || '#22d3ee'

  chart.setOption({
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(17,24,39,0.95)',
      borderColor: 'rgba(255,255,255,0.08)',
      textStyle: { color: '#f1f5f9', fontSize: 12 },
      formatter: (params: any) => {
        const time = params[0]?.axisValue || ''
        let html = `<div style="font-weight:600;margin-bottom:6px;color:#94a3b8">${time}</div>`
        for (const p of params) {
          html += `<div style="display:flex;align-items:center;gap:6px;margin:3px 0">${p.marker} <span>${p.seriesName}</span><span style="margin-left:auto;font-weight:600">${formatBytes(p.value)}</span></div>`
        }
        return html
      },
    },
    legend: {
      data: [t('traffic.upload'), t('traffic.download')],
      bottom: 0,
      textStyle: { color: '#64748b', fontSize: 11 },
      itemWidth: 12,
      itemHeight: 8,
      itemGap: 20,
    },
    grid: { left: 55, right: 16, top: 16, bottom: 36 },
    xAxis: {
      type: 'category',
      data: xData,
      axisLabel: {
        color: '#64748b',
        rotate: xData.length > 14 ? 45 : 0,
        fontSize: 10,
      },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: '#64748b',
        fontSize: 10,
        formatter: (v: number) => formatBytes(v),
      },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.04)' } },
      axisLine: { show: false },
      axisTick: { show: false },
    },
    series: [
      {
        name: t('traffic.upload'),
        type: 'bar',
        stack: 'traffic',
        data: sentData,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: accentColor },
            { offset: 1, color: accentColor + '66' },
          ]),
          borderRadius: [0, 0, 0, 0],
        },
        barMaxWidth: 28,
        emphasis: { itemStyle: { shadowBlur: 10, shadowColor: accentColor + '40' } },
      },
      {
        name: t('traffic.download'),
        type: 'bar',
        stack: 'traffic',
        data: recvData,
        itemStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: '#a78bfa' },
            { offset: 1, color: '#a78bfa66' },
          ]),
          borderRadius: [3, 3, 0, 0],
        },
        barMaxWidth: 28,
        emphasis: { itemStyle: { shadowBlur: 10, shadowColor: '#a78bfa40' } },
      },
    ],
    animationDuration: 600,
    animationEasing: 'cubicOut',
  }, true)
}

const formatBytes = (bytes?: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i >= 3 ? 2 : 1) + ' ' + units[i]
}

const formatSpeed = (bytesPerSec?: number) => {
  if (!bytesPerSec || bytesPerSec < 0) return '0 B/s'
  if (bytesPerSec < 1024) return bytesPerSec.toFixed(0) + ' B/s'
  if (bytesPerSec < 1048576) return (bytesPerSec / 1024).toFixed(1) + ' KB/s'
  return (bytesPerSec / 1048576).toFixed(2) + ' MB/s'
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return '-'
  return dateStr.substring(0, 10)
}

const openConfigDialog = (item?: any) => {
  const config = item ? {
    id: 0,
    interfaceName: item.interfaceName,
    monthlyLimit: item.monthlyLimit,
    resetDay: item.resetDay,
    enabled: item.enabled,
  } : undefined
  configDialogRef.value?.acceptParams(config)
}

const handleDelete = (interfaceName: string) => {
  ElMessageBox.confirm(
    t('traffic.deleteConfirm'),
    t('commons.tip'),
    { type: 'warning' },
  ).then(async () => {
    await trafficApi.deleteConfig(interfaceName)
    ElMessage.success(t('commons.success'))
    if (selectedInterface.value === interfaceName) {
      selectedInterface.value = ''
      statsItems.value = []
    }
    loadAll()
  }).catch(() => {})
}

let resizeHandler: (() => void) | null = null

onMounted(() => {
  loadAll()
  realtimeTimer = setInterval(loadRealtime, 3000)
  resizeHandler = () => chart?.resize()
  window.addEventListener('resize', resizeHandler)
})

onUnmounted(() => {
  chart?.dispose()
  if (realtimeTimer) clearInterval(realtimeTimer)
  if (resizeHandler) window.removeEventListener('resize', resizeHandler)
})

watch(selectedInterface, () => {
  initDateRange()
})
</script>

<style lang="scss" scoped>
.traffic-page { height: 100%; }

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
  .header-actions { display: flex; gap: 8px; }
}

/* ===== Summary Cards ===== */
.summary-cards {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.summary-card {
  flex: 1;
  min-width: 340px;
  max-width: 480px;
  position: relative;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-lg);
  overflow: hidden;
  transition: all 0.3s ease;

  &:hover {
    border-color: rgba(255,255,255,0.1);
    transform: translateY(-2px);
    .card-glow { opacity: 1; }
  }

  &.is-disabled { opacity: 0.45; pointer-events: none; }
}

.card-glow {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--xp-accent), transparent);
  opacity: 0;
  transition: opacity 0.3s;
}

.card-content {
  padding: 18px 20px;
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}

.iface-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--xp-accent-muted);
  color: var(--xp-accent);
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
  font-family: 'JetBrains Mono', monospace;
}

.card-actions { display: flex; gap: 2px; }

/* Realtime Speed */
.realtime-speed {
  display: flex;
  gap: 20px;
  margin-bottom: 16px;
  padding: 10px 14px;
  background: rgba(255,255,255,0.02);
  border-radius: var(--xp-radius-sm);
  border: 1px solid rgba(255,255,255,0.03);
}

.speed-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;

  .speed-arrow {
    font-size: 14px;
    font-weight: 700;
    width: 22px;
    height: 22px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
  }

  &.up .speed-arrow {
    background: rgba(34,211,238,0.12);
    color: #22d3ee;
  }
  &.down .speed-arrow {
    background: rgba(167,139,250,0.12);
    color: #a78bfa;
  }

  .speed-value {
    font-size: 15px;
    font-weight: 700;
    color: var(--xp-text-primary);
    font-variant-numeric: tabular-nums;
    font-family: 'JetBrains Mono', monospace;
  }
}

/* Progress Ring */
.progress-section {
  display: flex;
  gap: 20px;
  align-items: center;
}

.progress-ring-wrapper {
  position: relative;
  width: 110px;
  height: 110px;
  flex-shrink: 0;
}

.progress-ring {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.ring-bg {
  fill: none;
  stroke: rgba(255,255,255,0.04);
  stroke-width: 7;
}

.ring-fill {
  fill: none;
  stroke-width: 7;
  stroke-linecap: round;
  filter: drop-shadow(0 0 6px currentColor);
}

.ring-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;

  .ring-pct {
    font-size: 22px;
    font-weight: 800;
    color: var(--xp-text-primary);
    font-variant-numeric: tabular-nums;

    &.no-limit {
      font-size: 20px;
      color: var(--xp-text-muted);
    }
  }

  .ring-label {
    font-size: 10px;
    color: var(--xp-text-muted);
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-top: 2px;
  }
}

.quota-info {
  flex: 1;

  .quota-row {
    display: flex;
    justify-content: space-between;
    padding: 3px 0;
    font-size: 12px;
  }

  .quota-label { color: var(--xp-text-muted); }

  .quota-value {
    color: var(--xp-text-primary);
    font-weight: 500;
    font-variant-numeric: tabular-nums;

    &.highlight { color: var(--xp-accent); font-weight: 700; }
    &.period { font-size: 11px; color: var(--xp-text-secondary); }
  }
}

/* ===== Chart Section ===== */
.chart-card {
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-lg);
  padding: 20px;
  margin-bottom: 20px;
}

.chart-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.chart-container {
  height: 300px;
  margin-bottom: 12px;
}

.stats-footer {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 14px;
  background: rgba(255,255,255,0.02);
  border-radius: var(--xp-radius-sm);
  margin-bottom: 14px;
  font-size: 13px;
  color: var(--xp-text-secondary);

  .total-badge {
    margin-left: auto;
    background: var(--xp-accent-muted);
    color: var(--xp-accent);
    padding: 2px 10px;
    border-radius: 12px;
    font-weight: 600;
    font-size: 12px;
  }
}

.up-color { color: #22d3ee; }
.down-color { color: #a78bfa; }
</style>
