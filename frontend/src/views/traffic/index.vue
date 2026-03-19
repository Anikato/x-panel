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

    <!-- Summary Cards -->
    <div class="summary-cards" v-if="summary.length > 0">
      <el-card
        v-for="item in summary"
        :key="item.interfaceName"
        shadow="never"
        class="summary-card"
        :class="{ 'is-disabled': !item.enabled }"
      >
        <div class="card-header">
          <span class="iface-name">{{ item.interfaceName }}</span>
          <div class="card-actions">
            <el-button text size="small" @click="openConfigDialog(item)">
              <el-icon><Setting /></el-icon>
            </el-button>
            <el-button text size="small" type="danger" @click="handleDelete(item.interfaceName)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
        </div>

        <div class="card-body">
          <el-progress
            type="dashboard"
            :percentage="item.monthlyLimit > 0 ? Math.round(item.usedPercent) : 0"
            :color="progressColor"
            :width="90"
          >
            <template #default="{ percentage }">
              <span class="progress-inner" v-if="item.monthlyLimit > 0">{{ percentage }}%</span>
              <span class="progress-inner no-limit" v-else>--</span>
            </template>
          </el-progress>

          <div class="card-info">
            <div class="info-row">
              <span class="info-label">{{ $t('traffic.monthlyQuota') }}</span>
              <span class="info-value">{{ item.monthlyLimit > 0 ? formatBytes(item.monthlyLimit) : $t('traffic.unlimited') }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">{{ $t('traffic.used') }}</span>
              <span class="info-value">{{ formatBytes(item.totalUsed) }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">{{ $t('traffic.upload') }}</span>
              <span class="info-value up-color">↑ {{ formatBytes(item.totalSent) }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">{{ $t('traffic.download') }}</span>
              <span class="info-value down-color">↓ {{ formatBytes(item.totalRecv) }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">{{ $t('traffic.billingPeriod') }}</span>
              <span class="info-value period-text">{{ formatDate(item.periodStart) }} ~ {{ formatDate(item.periodEnd) }}</span>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <el-empty v-else-if="!loading" :description="$t('traffic.noConfig')">
      <el-button type="primary" @click="openConfigDialog()">{{ $t('traffic.addConfig') }}</el-button>
    </el-empty>

    <!-- Chart Section -->
    <el-card shadow="never" class="chart-section" v-if="summary.length > 0">
      <div class="chart-toolbar">
        <el-select v-model="selectedInterface" style="width: 160px" @change="loadStats">
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
          style="width: 280px"
        />
        <el-radio-group v-model="groupBy" @change="loadStats">
          <el-radio-button value="day">{{ $t('traffic.byDay') }}</el-radio-button>
          <el-radio-button value="hour">{{ $t('traffic.byHour') }}</el-radio-button>
        </el-radio-group>
      </div>

      <div ref="chartRef" class="chart-container"></div>

      <!-- Data Table -->
      <el-table :data="statsItems" size="small" class="stats-table" max-height="360">
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

      <div class="stats-summary" v-if="statsTotalSent > 0 || statsTotalRecv > 0">
        {{ $t('traffic.periodTotal') }}:
        ↑ {{ formatBytes(statsTotalSent) }} &nbsp; ↓ {{ formatBytes(statsTotalRecv) }} &nbsp;
        {{ $t('traffic.total') }}: {{ formatBytes(statsTotalSent + statsTotalRecv) }}
      </div>
    </el-card>

    <ConfigDialog ref="configDialogRef" @refresh="loadAll" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, Setting, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { trafficApi } from '@/api/modules/traffic'
import type { TrafficSummaryItem, TrafficStatsItem } from '@/api/modules/traffic'
import ConfigDialog from './config-dialog.vue'
import * as echarts from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, CanvasRenderer])

const { t } = useI18n()
const loading = ref(false)
const summary = ref<TrafficSummaryItem[]>([])
const configDialogRef = ref<InstanceType<typeof ConfigDialog>>()

const selectedInterface = ref('')
const dateRange = ref<[string, string] | null>(null)
const groupBy = ref<'day' | 'hour'>('day')
const statsItems = ref<TrafficStatsItem[]>([])
const statsTotalSent = ref(0)
const statsTotalRecv = ref(0)

const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null

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

  chart.setOption({
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const time = params[0]?.axisValue || ''
        let html = `<div style="font-weight:600;margin-bottom:4px">${time}</div>`
        for (const p of params) {
          html += `<div>${p.marker} ${p.seriesName}: ${formatBytes(p.value)}</div>`
        }
        return html
      },
    },
    legend: {
      data: [t('traffic.upload'), t('traffic.download')],
      bottom: 0,
      textStyle: { color: '#999' },
    },
    grid: { left: 60, right: 20, top: 20, bottom: 40 },
    xAxis: {
      type: 'category',
      data: xData,
      axisLabel: {
        color: '#999',
        rotate: xData.length > 15 ? 45 : 0,
        fontSize: 11,
      },
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        color: '#999',
        formatter: (v: number) => formatBytes(v),
      },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
    },
    series: [
      {
        name: t('traffic.upload'),
        type: 'bar',
        stack: 'traffic',
        data: sentData,
        itemStyle: { color: '#22d3ee', borderRadius: [0, 0, 0, 0] },
        barMaxWidth: 32,
      },
      {
        name: t('traffic.download'),
        type: 'bar',
        stack: 'traffic',
        data: recvData,
        itemStyle: { color: '#a78bfa', borderRadius: [4, 4, 0, 0] },
        barMaxWidth: 32,
      },
    ],
  }, true)
}

const progressColor = (percentage: number) => {
  if (percentage < 50) return '#22d3ee'
  if (percentage < 80) return '#f59e0b'
  return '#ef4444'
}

const formatBytes = (bytes?: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i >= 3 ? 2 : 1) + ' ' + units[i]
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
  resizeHandler = () => chart?.resize()
  window.addEventListener('resize', resizeHandler)
})

onUnmounted(() => {
  chart?.dispose()
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
  margin-bottom: 16px;
  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
  .header-actions { display: flex; gap: 8px; }
}

.summary-cards {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.summary-card {
  flex: 1;
  min-width: 320px;
  max-width: 460px;

  &.is-disabled { opacity: 0.5; }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;

    .iface-name {
      font-size: 15px;
      font-weight: 600;
      color: var(--xp-text-primary);
      font-family: 'JetBrains Mono', monospace;
    }

    .card-actions { display: flex; gap: 2px; }
  }

  .card-body {
    display: flex;
    gap: 20px;
    align-items: center;
  }

  .card-info {
    flex: 1;
    .info-row {
      display: flex;
      justify-content: space-between;
      padding: 3px 0;
      font-size: 12px;
    }
    .info-label { color: var(--xp-text-muted); }
    .info-value { color: var(--xp-text-primary); font-weight: 500; }
    .period-text { font-size: 11px; }
  }
}

.progress-inner {
  font-size: 18px;
  font-weight: 700;
  color: var(--xp-text-primary);
  &.no-limit { color: var(--xp-text-muted); font-size: 14px; }
}

.chart-section {
  margin-bottom: 16px;

  .chart-toolbar {
    display: flex;
    gap: 12px;
    align-items: center;
    margin-bottom: 16px;
    flex-wrap: wrap;
  }

  .chart-container {
    height: 320px;
    margin-bottom: 16px;
  }
}

.stats-table {
  margin-bottom: 8px;
}

.stats-summary {
  text-align: right;
  font-size: 13px;
  color: var(--xp-text-secondary);
  padding: 8px 4px 0;
  border-top: 1px solid var(--xp-border-light);
}

.up-color { color: #22d3ee; }
.down-color { color: #a78bfa; }
</style>
