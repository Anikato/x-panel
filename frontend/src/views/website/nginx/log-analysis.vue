<template>
  <div class="log-analysis">
    <!-- 顶部工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <el-select v-model="selectedSite" :placeholder="$t('nginx.selectSite')" style="width: 280px" @change="handleSiteChange">
          <el-option :label="$t('nginx.allSites')" value="" />
          <el-option v-for="site in sites" :key="site.name" :label="site.name" :value="site.name">
            <span>{{ site.name }}</span>
            <span class="site-conf-hint">{{ site.confFile.split('/').pop() }}</span>
          </el-option>
        </el-select>
        <el-select v-model="timeRange" style="width: 160px" @change="handleAnalyze">
          <el-option :label="$t('nginx.last1h')" value="1h" />
          <el-option :label="$t('nginx.last6h')" value="6h" />
          <el-option :label="$t('nginx.last24h')" value="24h" />
          <el-option :label="$t('nginx.last7d')" value="7d" />
          <el-option :label="$t('nginx.last30d')" value="30d" />
        </el-select>
      </div>
      <el-button :icon="Refresh" @click="handleAnalyze" :loading="analyzing" size="small">
        {{ $t('nginx.refreshLog') }}
      </el-button>
    </div>

    <!-- 子 Tab -->
    <el-tabs v-model="subTab" class="log-tabs">
      <!-- 统计概览 -->
      <el-tab-pane :label="$t('nginx.overview')" name="overview">
        <div v-loading="analyzing" class="overview-content">
          <!-- 概要卡片 -->
          <el-row :gutter="16" class="summary-row">
            <el-col :xs="12" :sm="6">
              <div class="summary-card">
                <div class="summary-value">{{ formatNumber(analysis.totalRequests) }}</div>
                <div class="summary-label">{{ $t('nginx.totalRequests') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6">
              <div class="summary-card">
                <div class="summary-value">{{ formatNumber(analysis.uniqueIPs) }}</div>
                <div class="summary-label">{{ $t('nginx.uniqueIPs') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6">
              <div class="summary-card">
                <div class="summary-value">{{ formatBytes(analysis.totalBytes) }}</div>
                <div class="summary-label">{{ $t('nginx.totalTraffic') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6">
              <div class="summary-card" :class="{ 'error-card': analysis.errorRate > 10 }">
                <div class="summary-value">{{ analysis.errorRate?.toFixed(1) || '0.0' }}%</div>
                <div class="summary-label">{{ $t('nginx.errorRate') }}</div>
              </div>
            </el-col>
          </el-row>

          <!-- 图表行 -->
          <el-row :gutter="16" style="margin-top: 16px">
            <el-col :xs="24" :lg="16">
              <el-card shadow="never" class="chart-card">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.requestTrend') }}</span>
                </template>
                <div ref="trendChartRef" class="chart-container"></div>
              </el-card>
            </el-col>
            <el-col :xs="24" :lg="8">
              <el-card shadow="never" class="chart-card">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.statusCodeDist') }}</span>
                </template>
                <div ref="statusChartRef" class="chart-container"></div>
              </el-card>
            </el-col>
          </el-row>

          <!-- Top 排行 -->
          <el-row :gutter="16" style="margin-top: 16px">
            <el-col :xs="24" :lg="12">
              <el-card shadow="never" class="rank-card">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.topIPs') }}</span>
                </template>
                <el-table :data="analysis.topIps || []" size="small" stripe :show-header="true" max-height="320">
                  <el-table-column type="index" width="36" />
                  <el-table-column :label="$t('nginx.ip')" min-width="140">
                    <template #default="{ row }">
                      <span class="mono-text">{{ row.name }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.location')" min-width="120">
                    <template #default="{ row }">
                      <span v-if="row.country" class="location-text">{{ row.country }}<template v-if="row.city"> / {{ row.city }}</template></span>
                      <span v-else class="muted-text">-</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.requests')" width="90" align="right">
                    <template #default="{ row }">
                      <span class="count-text">{{ formatNumber(row.count) }}</span>
                    </template>
                  </el-table-column>
                </el-table>
              </el-card>
            </el-col>
            <el-col :xs="24" :lg="12">
              <el-card shadow="never" class="rank-card">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.topURLs') }}</span>
                </template>
                <el-table :data="analysis.topUrls || []" size="small" stripe :show-header="true" max-height="320">
                  <el-table-column type="index" width="36" />
                  <el-table-column :label="$t('nginx.url')" min-width="240">
                    <template #default="{ row }">
                      <span class="mono-text url-cell">{{ row.name }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.requests')" width="90" align="right">
                    <template #default="{ row }">
                      <span class="count-text">{{ formatNumber(row.count) }}</span>
                    </template>
                  </el-table-column>
                </el-table>
              </el-card>
            </el-col>
          </el-row>

          <!-- User-Agent 排行 -->
          <el-row :gutter="16" style="margin-top: 16px">
            <el-col :span="24">
              <el-card shadow="never" class="rank-card">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.topUserAgents') }}</span>
                </template>
                <el-table :data="analysis.topUserAgents || []" size="small" stripe :show-header="true" max-height="280">
                  <el-table-column type="index" width="36" />
                  <el-table-column :label="$t('nginx.userAgent')" min-width="400">
                    <template #default="{ row }">
                      <span class="mono-text ua-cell">{{ row.name || '-' }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.requests')" width="100" align="right">
                    <template #default="{ row }">
                      <span class="count-text">{{ formatNumber(row.count) }}</span>
                    </template>
                  </el-table-column>
                </el-table>
              </el-card>
            </el-col>
          </el-row>

          <!-- 无数据 -->
          <el-empty v-if="!analyzing && analysis.totalRequests === 0" :description="$t('nginx.noLogData')" />
        </div>
      </el-tab-pane>

      <!-- 访问日志 -->
      <el-tab-pane :label="$t('nginx.accessLog')" name="access" lazy>
        <div class="log-viewer-toolbar">
          <el-select v-model="logLines" size="small" style="width: 120px" @change="loadAccessLog">
            <el-option label="100" :value="100" />
            <el-option label="200" :value="200" />
            <el-option label="500" :value="500" />
            <el-option label="1000" :value="1000" />
            <el-option label="3000" :value="3000" />
          </el-select>
          <span class="log-lines-label">{{ $t('nginx.logLines') }}</span>
          <el-button size="small" :icon="Refresh" @click="loadAccessLog" :loading="accessLogLoading">
            {{ $t('nginx.refreshLog') }}
          </el-button>
          <span v-if="accessLogPath" class="log-path-hint">{{ $t('nginx.logPath') }}: {{ accessLogPath }}</span>
        </div>
        <div class="log-viewer" v-loading="accessLogLoading">
          <pre>{{ accessLogContent || $t('nginx.noLogData') }}</pre>
        </div>
      </el-tab-pane>

      <!-- 错误日志 -->
      <el-tab-pane :label="$t('nginx.errorLog')" name="error" lazy>
        <div class="log-viewer-toolbar">
          <el-select v-model="errorLogLines" size="small" style="width: 120px" @change="loadErrorLog">
            <el-option label="100" :value="100" />
            <el-option label="200" :value="200" />
            <el-option label="500" :value="500" />
            <el-option label="1000" :value="1000" />
          </el-select>
          <span class="log-lines-label">{{ $t('nginx.logLines') }}</span>
          <el-button size="small" :icon="Refresh" @click="loadErrorLog" :loading="errorLogLoading">
            {{ $t('nginx.refreshLog') }}
          </el-button>
          <span v-if="errorLogPath" class="log-path-hint">{{ $t('nginx.logPath') }}: {{ errorLogPath }}</span>
        </div>
        <div class="log-viewer" v-loading="errorLogLoading">
          <pre>{{ errorLogContent || $t('nginx.noLogData') }}</pre>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh } from '@element-plus/icons-vue'
import { detectNginxSites, analyzeNginxSiteLog, tailNginxLog } from '@/api/modules/website'
import * as echarts from 'echarts'

const { t } = useI18n()

const selectedSite = ref('')
const timeRange = ref('24h')
const subTab = ref('overview')
const analyzing = ref(false)

const sites = ref<any[]>([])
const analysis = reactive<any>({
  totalRequests: 0, uniqueIPs: 0, totalBytes: 0, errorRate: 0,
  statusCodes: {}, topUrls: [], topIps: [], topUserAgents: [],
  hourlyStats: [], dailyStats: [],
})

const trendChartRef = ref<HTMLElement>()
const statusChartRef = ref<HTMLElement>()
let trendChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null

const logLines = ref(200)
const errorLogLines = ref(200)
const accessLogContent = ref('')
const accessLogPath = ref('')
const accessLogLoading = ref(false)
const errorLogContent = ref('')
const errorLogPath = ref('')
const errorLogLoading = ref(false)

const loadSites = async () => {
  try {
    const res = await detectNginxSites()
    sites.value = res.data || []
  } catch { sites.value = [] }
}

const handleSiteChange = () => {
  handleAnalyze()
  if (subTab.value === 'access') loadAccessLog()
  if (subTab.value === 'error') loadErrorLog()
}

const handleAnalyze = async () => {
  analyzing.value = true
  try {
    const res = await analyzeNginxSiteLog({ site: selectedSite.value, timeRange: timeRange.value })
    Object.assign(analysis, res.data || {})
    await nextTick()
    renderCharts()
  } catch {
    Object.assign(analysis, { totalRequests: 0, uniqueIPs: 0, totalBytes: 0, errorRate: 0, statusCodes: {}, topUrls: [], topIps: [], topUserAgents: [], hourlyStats: [], dailyStats: [] })
  } finally {
    analyzing.value = false
  }
}

const loadAccessLog = async () => {
  accessLogLoading.value = true
  try {
    const res = await tailNginxLog({ site: selectedSite.value, type: 'access', lines: logLines.value })
    accessLogContent.value = res.data?.content || ''
    accessLogPath.value = res.data?.path || ''
  } catch { accessLogContent.value = '' }
  finally { accessLogLoading.value = false }
}

const loadErrorLog = async () => {
  errorLogLoading.value = true
  try {
    const res = await tailNginxLog({ site: selectedSite.value, type: 'error', lines: errorLogLines.value })
    errorLogContent.value = res.data?.content || ''
    errorLogPath.value = res.data?.path || ''
  } catch { errorLogContent.value = '' }
  finally { errorLogLoading.value = false }
}

watch(subTab, (val) => {
  if (val === 'access' && !accessLogContent.value) loadAccessLog()
  if (val === 'error' && !errorLogContent.value) loadErrorLog()
})

const statusColors: Record<string, string> = {
  '2xx': '#67c23a', '3xx': '#e6a23c', '4xx': '#f56c6c', '5xx': '#909399',
}

const renderCharts = () => {
  renderTrendChart()
  renderStatusChart()
}

const renderTrendChart = () => {
  if (!trendChartRef.value) return
  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }

  const isHourly = ['1h', '6h', '24h'].includes(timeRange.value)
  const stats = isHourly ? (analysis.hourlyStats || []) : (analysis.dailyStats || [])

  const xData = stats.map((s: any) => {
    if (isHourly) {
      return s.time.substring(11, 16)
    }
    return s.time.substring(5)
  })
  const reqData = stats.map((s: any) => s.requests)
  const byteData = stats.map((s: any) => +(s.bytes / 1024).toFixed(1))

  trendChart.setOption({
    tooltip: {
      trigger: 'axis',
      axisPointer: { type: 'cross' },
    },
    grid: { left: 50, right: 50, top: 30, bottom: 30 },
    xAxis: { type: 'category', data: xData, axisLabel: { fontSize: 11 } },
    yAxis: [
      { type: 'value', name: t('nginx.requests'), axisLabel: { fontSize: 11 } },
      { type: 'value', name: 'KB', axisLabel: { fontSize: 11 } },
    ],
    series: [
      {
        name: t('nginx.requests'),
        type: 'bar',
        data: reqData,
        itemStyle: { color: '#409eff', borderRadius: [3, 3, 0, 0] },
        barMaxWidth: 20,
      },
      {
        name: t('nginx.totalTraffic'),
        type: 'line',
        yAxisIndex: 1,
        data: byteData,
        smooth: true,
        lineStyle: { color: '#e6a23c', width: 2 },
        itemStyle: { color: '#e6a23c' },
        areaStyle: { color: 'rgba(230,162,60,0.1)' },
      },
    ],
  }, true)
}

const renderStatusChart = () => {
  if (!statusChartRef.value) return
  if (!statusChart) {
    statusChart = echarts.init(statusChartRef.value)
  }

  const codes = analysis.statusCodes || {}
  const data = Object.entries(codes).map(([name, value]) => ({
    name,
    value,
    itemStyle: { color: statusColors[name] || '#909399' },
  }))

  if (data.length === 0) {
    statusChart.clear()
    return
  }

  statusChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      center: ['50%', '50%'],
      avoidLabelOverlap: true,
      label: { show: true, formatter: '{b}\n{d}%', fontSize: 12 },
      data,
    }],
  }, true)
}

const handleResize = () => {
  trendChart?.resize()
  statusChart?.resize()
}

const formatNumber = (n: number) => {
  if (!n) return '0'
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return String(n)
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let b = bytes
  while (b >= 1024 && i < units.length - 1) { b /= 1024; i++ }
  return b.toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}

onMounted(async () => {
  await loadSites()
  handleAnalyze()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  trendChart?.dispose()
  statusChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style lang="scss" scoped>
.log-analysis {
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
    gap: 12px;

    .toolbar-left {
      display: flex;
      align-items: center;
      gap: 10px;
    }
  }

  .site-conf-hint {
    float: right;
    font-size: 12px;
    color: var(--xp-text-muted);
  }

  .summary-row {
    .summary-card {
      background: var(--xp-bg-card);
      border: 1px solid var(--xp-border-light);
      border-radius: var(--xp-radius);
      padding: 18px 16px;
      text-align: center;
      transition: border-color 0.2s;

      &:hover { border-color: var(--el-color-primary-light-5); }

      &.error-card {
        border-color: var(--el-color-danger-light-5);
        .summary-value { color: var(--el-color-danger); }
      }
    }

    .summary-value {
      font-size: 28px;
      font-weight: 700;
      color: var(--xp-text-primary);
      line-height: 1.2;
      font-variant-numeric: tabular-nums;
    }

    .summary-label {
      font-size: 13px;
      color: var(--xp-text-muted);
      margin-top: 6px;
    }
  }

  .chart-card, .rank-card {
    :deep(.el-card__header) {
      padding: 12px 16px;
      border-bottom: 1px solid var(--xp-border-light);
    }
    :deep(.el-card__body) { padding: 12px; }
  }

  .chart-title {
    font-size: 14px;
    font-weight: 600;
    color: var(--xp-text-primary);
  }

  .chart-container {
    width: 100%;
    height: 280px;
  }

  .mono-text {
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    font-size: 12px;
  }

  .url-cell, .ua-cell {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
  }

  .location-text { font-size: 12px; color: var(--xp-text-secondary); }
  .muted-text { font-size: 12px; color: var(--xp-text-muted); }
  .count-text { font-weight: 600; font-variant-numeric: tabular-nums; }

  .log-viewer-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;

    .log-lines-label {
      font-size: 13px;
      color: var(--xp-text-muted);
      margin-right: 8px;
    }

    .log-path-hint {
      margin-left: auto;
      font-size: 12px;
      color: var(--xp-text-muted);
      font-family: 'JetBrains Mono', 'Fira Code', monospace;
    }
  }

  .log-viewer {
    background: var(--xp-bg-inset);
    border: 1px solid var(--xp-border-light);
    border-radius: var(--xp-radius);
    padding: 12px;
    max-height: 600px;
    overflow: auto;

    pre {
      margin: 0;
      font-size: 12px;
      line-height: 1.6;
      font-family: 'JetBrains Mono', 'Fira Code', monospace;
      color: var(--xp-text-primary);
      white-space: pre-wrap;
      word-break: break-all;
    }
  }

  .log-tabs {
    :deep(.el-tabs__header) { margin-bottom: 12px; }
  }
}
</style>
