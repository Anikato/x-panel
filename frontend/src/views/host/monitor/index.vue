<template>
  <div class="monitor-page xp-page-shell">
    <div class="monitor-toolbar-row">
      <p class="monitor-hint">{{ $t('monitor.historyHint') }}</p>
      <router-link class="overview-link" to="/home">{{ $t('home.overview') }}</router-link>
    </div>

    <div class="history-toolbar">
        <div class="time-shortcuts">
          <el-button v-for="s in shortcuts" :key="s.label" size="small" :type="activeShortcut === s.label ? 'primary' : ''" @click="applyShortcut(s)">{{ s.label }}</el-button>
        </div>
        <div class="toolbar-right">
          <el-date-picker v-model="timeRange" type="datetimerange" :start-placeholder="$t('monitor.startTime')" :end-placeholder="$t('monitor.endTime')" size="small" style="max-width: 360px" @change="loadHistory" />
          <el-button size="small" :icon="Setting" @click="showSettingDialog = true" />
        </div>
      </div>

      <!-- 监控设置对话框 -->
      <el-dialog v-model="showSettingDialog" :title="$t('monitor.monitorSetting')" width="480px" :close-on-click-modal="false">
        <el-form label-width="110px" v-loading="settingLoading">
          <el-form-item :label="$t('monitor.monitorStatus')">
            <el-switch v-model="monitorEnabled" :active-text="$t('monitor.enableMonitor')" :inactive-text="$t('monitor.disableMonitor')" @change="onSettingChange('MonitorStatus', monitorEnabled ? 'enable' : 'disable')" />
          </el-form-item>
          <el-form-item :label="$t('monitor.monitorInterval')">
            <el-select v-model="monitorInterval" style="width: 100%" @change="onSettingChange('MonitorInterval', String(monitorInterval))">
              <el-option v-for="v in [60, 120, 300, 600]" :key="v" :label="$t('monitor.intervalSeconds', { n: v })" :value="String(v)" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('monitor.monitorStoreDays')">
            <el-select v-model="monitorDays" style="width: 100%" @change="onSettingChange('MonitorStoreDays', String(monitorDays))">
              <el-option v-for="v in [1, 3, 7, 14, 30]" :key="v" :label="$t('monitor.retentionDays', { n: v })" :value="String(v)" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('monitor.defaultNetwork')">
            <el-select v-model="defaultNet" style="width: 100%" @change="onSettingChange('DefaultNetwork', defaultNet)">
              <el-option v-for="o in netOptions" :key="o" :label="o === 'all' ? $t('commons.all') : o" :value="o" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('monitor.defaultIO')">
            <el-select v-model="defaultIO" style="width: 100%" @change="onSettingChange('DefaultIO', defaultIO)">
              <el-option v-for="o in ioOptions" :key="o" :label="o === 'all' ? $t('commons.all') : o" :value="o" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-popconfirm :title="$t('monitor.cleanConfirm')" @confirm="handleCleanData">
              <template #reference>
                <el-button type="danger" plain size="small">{{ $t('monitor.cleanData') }}</el-button>
              </template>
            </el-popconfirm>
          </el-form-item>
        </el-form>
      </el-dialog>

      <!-- 负载（全宽） -->
      <article class="xp-deck chart-card">
        <div class="xp-section" style="margin-top:0"><h3 class="chart-title">{{ $t('monitor.load') }}</h3></div>
        <div ref="loadChartRef" class="chart-container"></div>
      </article>

      <!-- CPU + 内存 -->
      <el-row :gutter="14" class="chart-grid-row">
        <el-col :xs="24" :md="12">
          <article class="xp-deck chart-card">
            <div class="xp-section" style="margin-top:0"><h3 class="chart-title">CPU</h3></div>
            <div ref="cpuChartRef" class="chart-container"></div>
          </article>
        </el-col>
        <el-col :xs="24" :md="12">
          <article class="xp-deck chart-card">
            <div class="xp-section" style="margin-top:0"><h3 class="chart-title">{{ $t('monitor.memory') }}</h3></div>
            <div ref="memChartRef" class="chart-container"></div>
          </article>
        </el-col>
      </el-row>

      <!-- IO + 网络 -->
      <el-row :gutter="14" class="chart-grid-row">
        <el-col :xs="24" :md="12">
          <article class="xp-deck chart-card">
            <div class="xp-section" style="margin-top:0">
                <h3 class="chart-title">{{ $t('monitor.disk') }} I/O</h3>
                <el-select v-model="ioChoose" size="small" style="width: 120px" @change="loadHistory">
                  <el-option v-for="o in ioOptions" :key="o" :label="o === 'all' ? $t('commons.all') : o" :value="o" />
                </el-select>
            </div>
            <div ref="ioChartRef" class="chart-container"></div>
          </article>
        </el-col>
        <el-col :xs="24" :md="12">
          <article class="xp-deck chart-card">
            <div class="xp-section" style="margin-top:0">
                <h3 class="chart-title">{{ $t('monitor.network') }}</h3>
                <el-select v-model="netChoose" size="small" style="width: 120px" @change="loadHistory">
                  <el-option v-for="o in netOptions" :key="o" :label="o === 'all' ? $t('commons.all') : o" :value="o" />
                </el-select>
            </div>
            <div ref="netChartRef" class="chart-container"></div>
          </article>
        </el-col>
      </el-row>

      <!-- 硬件温度（仅物理机有数据时显示） -->
      <article class="xp-deck chart-card" v-show="hasSensorData">
        <div class="xp-section" style="margin-top:0"><h3 class="chart-title">{{ $t('monitor.temperature') }}</h3></div>
        <div ref="sensorChartRef" class="chart-container"></div>
      </article>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Setting } from '@element-plus/icons-vue'
import { loadMonitorHistory, getIOOptions as fetchIOOptions, getNetworkOptions as fetchNetOptions, getMonitorSetting, updateMonitorSetting, cleanMonitorData } from '@/api/modules/monitor'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent, DataZoomComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import { chartTokens, colorAlpha, onAppearanceChange } from '@/theme'

echarts.use([LineChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent, CanvasRenderer])

const { t } = useI18n()

// ==================== History ====================
const loadChartRef = ref<HTMLDivElement>()
const cpuChartRef = ref<HTMLDivElement>()
const memChartRef = ref<HTMLDivElement>()
const ioChartRef = ref<HTMLDivElement>()
const netChartRef = ref<HTMLDivElement>()
const sensorChartRef = ref<HTMLDivElement>()

let loadChart: echarts.ECharts | null = null
let cpuChart: echarts.ECharts | null = null
let memChart: echarts.ECharts | null = null
let ioChart: echarts.ECharts | null = null
let netChart: echarts.ECharts | null = null
let sensorChart: echarts.ECharts | null = null

const hasSensorData = ref(false)

const ioOptions = ref<string[]>(['all'])
const netOptions = ref<string[]>(['all'])
const ioChoose = ref('all')
const netChoose = ref('all')

const timeRange = ref<[Date, Date]>([new Date(Date.now() - 6 * 3600000), new Date()])
const activeShortcut = ref('6h')

const shortcuts = [
  { label: '1h', ms: 3600000 },
  { label: '6h', ms: 6 * 3600000 },
  { label: '24h', ms: 24 * 3600000 },
  { label: '7d', ms: 7 * 24 * 3600000 },
]

const applyShortcut = (s: { label: string; ms: number }) => {
  activeShortcut.value = s.label
  timeRange.value = [new Date(Date.now() - s.ms), new Date()]
  loadHistory()
}

const baseChartOption = (): echarts.EChartsCoreOption => {
  const t = chartTokens()
  return {
    backgroundColor: 'transparent',
    grid: { top: 30, right: 20, bottom: 60, left: 50 },
    tooltip: { trigger: 'axis', backgroundColor: t.tooltipBg, borderColor: 'transparent', textStyle: { color: t.tooltipText, fontSize: 12 } },
    xAxis: { type: 'time', axisLabel: { color: t.muted, fontSize: 10 }, axisLine: { lineStyle: { color: t.border } }, splitLine: { show: false } },
    dataZoom: [{ type: 'inside' }, { type: 'slider', height: 20, bottom: 8, borderColor: 'transparent', backgroundColor: colorAlpha(t.primaryText, 0.04), fillerColor: colorAlpha(t.muted, 0.2), handleStyle: { color: t.muted } }],
  }
}

const initCharts = () => {
  const init = (el: HTMLDivElement | undefined) => el ? echarts.init(el) : null
  loadChart = init(loadChartRef.value)
  cpuChart = init(cpuChartRef.value)
  memChart = init(memChartRef.value)
  ioChart = init(ioChartRef.value)
  netChart = init(netChartRef.value)
  sensorChart = init(sensorChartRef.value)
}

const disposeCharts = () => {
  ;[loadChart, cpuChart, memChart, ioChart, netChart, sensorChart].forEach(c => c?.dispose())
  loadChart = cpuChart = memChart = ioChart = netChart = sensorChart = null
}

const loadHistory = async () => {
  if (!timeRange.value?.[0] || !timeRange.value?.[1]) return
  const startTime = timeRange.value[0].toISOString()
  const endTime = timeRange.value[1].toISOString()

  try {
    const res = await loadMonitorHistory({ param: 'all', io: ioChoose.value, network: netChoose.value, startTime, endTime })
    const allData = res.data || []
    const baseData = allData.find((d: any) => d.param === 'base')
    const ioData = allData.find((d: any) => d.param === 'io')
    const networkData = allData.find((d: any) => d.param === 'network')
    const sensorData = allData.find((d: any) => d.param === 'sensor')

    const tokens = chartTokens()
    const axisLabel = { color: tokens.muted }
    const splitLine = { lineStyle: { color: tokens.borderLight } }
    const legendText = { color: tokens.text }

    if (baseData && loadChart) {
      const dates = baseData.date || []
      const values = baseData.value || []
      loadChart.setOption({
        ...baseChartOption(),
        yAxis: { type: 'value', name: '', axisLabel: { ...axisLabel, formatter: '{value}' }, splitLine },
        legend: { data: ['Load1', 'Load5', 'Load15'], textStyle: legendText, top: 0 },
        series: [
          { name: 'Load1', type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.cpuLoad1 ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.accent }, areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: colorAlpha(tokens.accent, 0.15) }, { offset: 1, color: colorAlpha(tokens.accent, 0) }]) } },
          { name: 'Load5', type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.cpuLoad5 ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.secondary } },
          { name: 'Load15', type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.cpuLoad15 ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.success } },
        ],
      })
    }

    if (baseData && cpuChart) {
      const dates = baseData.date || []
      const values = baseData.value || []
      cpuChart.setOption({
        ...baseChartOption(),
        yAxis: { type: 'value', max: 100, axisLabel: { ...axisLabel, formatter: '{value}%' }, splitLine },
        series: [{
          name: 'CPU', type: 'line', smooth: true, symbol: 'none',
          data: dates.map((d: string, i: number) => [d, values[i]?.cpu?.toFixed(1) ?? 0]),
          lineStyle: { width: 1.5 }, itemStyle: { color: tokens.accent },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: colorAlpha(tokens.accent, 0.2) }, { offset: 1, color: colorAlpha(tokens.accent, 0) }]) },
        }],
      })
    }

    if (baseData && memChart) {
      const dates = baseData.date || []
      const values = baseData.value || []
      memChart.setOption({
        ...baseChartOption(),
        yAxis: { type: 'value', max: 100, axisLabel: { ...axisLabel, formatter: '{value}%' }, splitLine },
        series: [{
          name: t('monitor.memory'), type: 'line', smooth: true, symbol: 'none',
          data: dates.map((d: string, i: number) => [d, values[i]?.memory?.toFixed(1) ?? 0]),
          lineStyle: { width: 1.5 }, itemStyle: { color: tokens.secondary },
          areaStyle: { color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [{ offset: 0, color: colorAlpha(tokens.secondary, 0.2) }, { offset: 1, color: colorAlpha(tokens.secondary, 0) }]) },
        }],
      })
    }

    if (ioData && ioChart) {
      const dates = ioData.date || []
      const values = ioData.value || []
      ioChart.setOption({
        ...baseChartOption(),
        yAxis: { type: 'value', axisLabel: { ...axisLabel, formatter: (v: number) => formatBytesShort(v) + '/s' }, splitLine },
        legend: { data: [t('monitor.read'), t('monitor.write')], textStyle: legendText, top: 0 },
        series: [
          { name: t('monitor.read'), type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.read ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.success } },
          { name: t('monitor.write'), type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.write ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.warning } },
        ],
      })
    }

    if (networkData && netChart) {
      const dates = networkData.date || []
      const values = networkData.value || []
      netChart.setOption({
        ...baseChartOption(),
        yAxis: { type: 'value', axisLabel: { ...axisLabel, formatter: (v: number) => v.toFixed(0) + ' KB/s' }, splitLine },
        legend: { data: [t('monitor.upload'), t('monitor.download')], textStyle: legendText, top: 0 },
        series: [
          { name: t('monitor.upload'), type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.up?.toFixed(1) ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.up } },
          { name: t('monitor.download'), type: 'line', smooth: true, symbol: 'none', data: dates.map((d: string, i: number) => [d, values[i]?.down?.toFixed(1) ?? 0]), lineStyle: { width: 1.5 }, itemStyle: { color: tokens.down } },
        ],
      })
    }

    hasSensorData.value = !!(sensorData && sensorData.date?.length)
    if (hasSensorData.value && sensorChart) {
      // 按传感器名分组，每个传感器一条曲线
      const dates = sensorData.date || []
      const values = sensorData.value || []
      const seriesMap = new Map<string, [string, number][]>()
      values.forEach((v: any, i: number) => {
        const arr = seriesMap.get(v.name) || []
        arr.push([dates[i], v.temp])
        seriesMap.set(v.name, arr)
      })
      const colors = tokens.series
      const names = [...seriesMap.keys()].slice(0, 8)
      sensorChart.setOption({
        ...baseChartOption(),
        yAxis: { type: 'value', axisLabel: { ...axisLabel, formatter: '{value}°C' }, splitLine },
        legend: { data: names.map(n => n.replace(/_/g, ' ')), textStyle: legendText, top: 0 },
        series: names.map((n, idx) => ({
          name: n.replace(/_/g, ' '), type: 'line', smooth: true, symbol: 'none',
          data: seriesMap.get(n), lineStyle: { width: 1.5 }, itemStyle: { color: colors[idx % colors.length] },
        })),
      }, true)
      nextTick(() => sensorChart?.resize())
    }
  } catch { /* */ }
}

const formatBytesShort = (b: number) => {
  if (b < 1024) return b.toFixed(0) + ' B'
  if (b < 1048576) return (b / 1024).toFixed(0) + ' KB'
  if (b < 1073741824) return (b / 1048576).toFixed(1) + ' MB'
  return (b / 1073741824).toFixed(1) + ' GB'
}

// ==================== Settings ====================
const showSettingDialog = ref(false)
const settingLoading = ref(false)
const monitorEnabled = ref(true)
const monitorInterval = ref('300')
const monitorDays = ref('7')
const defaultNet = ref('all')
const defaultIO = ref('all')

const loadSettings = async () => {
  settingLoading.value = true
  try {
    const res = await getMonitorSetting()
    const s = res.data
    monitorEnabled.value = s.monitorStatus === 'enable'
    monitorInterval.value = s.monitorInterval || '300'
    monitorDays.value = s.monitorStoreDays || '7'
    defaultNet.value = s.defaultNetwork || 'all'
    defaultIO.value = s.defaultIO || 'all'
  } catch { /* */ }
  finally { settingLoading.value = false }
}

const onSettingChange = async (key: string, value: string) => {
  try {
    await updateMonitorSetting({ key, value })
    ElMessage.success(t('commons.saveSuccess'))
  } catch { /* */ }
}

const handleCleanData = async () => {
  try {
    await cleanMonitorData()
    ElMessage.success(t('commons.operationSuccess'))
    loadHistory()
  } catch { /* */ }
}

const loadDeviceOptions = async () => {
  try {
    const [ioRes, netRes] = await Promise.all([fetchIOOptions(), fetchNetOptions()])
    ioOptions.value = ioRes.data || ['all']
    netOptions.value = netRes.data || ['all']
  } catch { /* */ }
}

const handleResize = () => {
  ;[loadChart, cpuChart, memChart, ioChart, netChart, sensorChart].forEach(c => c?.resize())
}

let stopAppearance: (() => void) | undefined
onMounted(async () => {
  window.addEventListener('resize', handleResize)
  await Promise.all([loadDeviceOptions(), loadSettings()])
  await nextTick()
  initCharts()
  loadHistory()
  stopAppearance = onAppearanceChange(() => { loadHistory() })
})
onUnmounted(() => {
  disposeCharts()
  window.removeEventListener('resize', handleResize)
  stopAppearance?.()
})
</script>

<style lang="scss" scoped>
.monitor-toolbar-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.monitor-hint {
  margin: 0;
  color: var(--xp-text-muted);
  font-size: 13px;
}

.overview-link {
  color: var(--xp-accent);
  font-size: 12px;
  text-decoration: none;
  white-space: nowrap;
  &:hover { text-decoration: underline; }
}

.history-toolbar {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  margin-bottom: 16px; flex-wrap: wrap;
  .time-shortcuts { display: flex; gap: 4px; }
  .toolbar-right { display: flex; align-items: center; gap: 8px; }
}

.chart-card { margin-bottom: 14px; border-left-width: 3px; }
.chart-container { height: clamp(300px, 25vh, 420px); width: 100%; }
.chart-title { font-weight: 600; font-size: 13px; color: var(--xp-text-primary); }
.chart-hd-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; }

.chart-grid-row {
  margin-bottom: 2px;
}

@media (min-width: 1440px) {
  .chart-container {
    height: 360px;
  }
}

</style>
