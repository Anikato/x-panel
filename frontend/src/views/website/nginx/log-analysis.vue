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
            <el-col :xs="12" :sm="6" :md="5">
              <div class="summary-card">
                <div class="summary-value">{{ formatNumber(analysis.totalRequests) }}</div>
                <div class="summary-label">{{ $t('nginx.totalRequests') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6" :md="5">
              <div class="summary-card">
                <div class="summary-value">{{ formatNumber(analysis.uniqueIPs) }}</div>
                <div class="summary-label">{{ $t('nginx.uniqueIPs') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6" :md="5">
              <div class="summary-card">
                <div class="summary-value">{{ formatBytes(analysis.totalBytes) }}</div>
                <div class="summary-label">{{ $t('nginx.totalTraffic') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6" :md="5">
              <div class="summary-card" :class="errorRateClass">
                <div class="summary-value">{{ analysis.errorRate?.toFixed(1) || '0.0' }}%</div>
                <div class="summary-label">{{ $t('nginx.errorRate') }}</div>
              </div>
            </el-col>
            <el-col :xs="12" :sm="6" :md="4">
              <div class="summary-card" :class="{ 'threat-card': analysis.threatRequests > 0 }">
                <div class="summary-value">{{ formatNumber(analysis.threatRequests) }}</div>
                <div class="summary-label">{{ $t('nginx.threatRequests') }}</div>
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

          <!-- Top IP + Top URL -->
          <el-row :gutter="16" style="margin-top: 16px">
            <el-col :xs="24" :lg="12">
              <el-card shadow="never" class="rank-card">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.topIPs') }}</span>
                </template>
                <el-table :data="analysis.topIps || []" size="small" stripe :show-header="true" max-height="320"
                  :row-class-name="ipRowClass">
                  <el-table-column type="index" width="36" />
                  <el-table-column :label="$t('nginx.ip')" min-width="130">
                    <template #default="{ row }">
                      <span class="mono-text drilldown-link" @click="handleDrilldown('ip', row.name)">{{ row.name }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.location')" min-width="100">
                    <template #default="{ row }">
                      <span v-if="row.country" class="location-text">{{ row.country }}<template v-if="row.city"> / {{ row.city }}</template></span>
                      <span v-else class="muted-text">-</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.requests')" width="80" align="right">
                    <template #default="{ row }">
                      <span class="count-text">{{ formatNumber(row.count) }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.logStatus')" width="80" align="center">
                    <template #default="{ row }">
                      <el-tag v-if="row.banned" type="danger" size="small" effect="dark">{{ $t('nginx.banned') }}</el-tag>
                      <span v-else-if="isHighTraffic(row)" class="high-traffic-tag">{{ $t('nginx.highTraffic') }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.actions')" width="80" align="center">
                    <template #default="{ row }">
                      <el-popconfirm v-if="!row.banned"
                        :title="$t('nginx.banConfirm', { ip: row.name })"
                        @confirm="handleBanIP(row.name)">
                        <template #reference>
                          <el-button type="danger" text size="small" :loading="banLoading[row.name]">{{ $t('nginx.ban') }}</el-button>
                        </template>
                      </el-popconfirm>
                      <el-popconfirm v-else
                        :title="$t('nginx.unbanConfirm', { ip: row.name })"
                        @confirm="handleUnbanIP(row.name)">
                        <template #reference>
                          <el-button type="warning" text size="small" :loading="banLoading[row.name]">{{ $t('nginx.unban') }}</el-button>
                        </template>
                      </el-popconfirm>
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
                      <span class="mono-text url-cell drilldown-link" @click="handleDrilldown('url', row.name)">{{ row.name }}</span>
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

          <!-- 威胁检测 -->
          <el-row v-if="analysis.threatRequests > 0" :gutter="16" style="margin-top: 16px">
            <el-col :xs="24" :lg="10">
              <el-card shadow="never" class="rank-card threat-section">
                <template #header>
                  <div class="threat-header">
                    <span class="chart-title">{{ $t('nginx.attackTypes') }}</span>
                    <el-tag type="danger" size="small" effect="plain">{{ formatNumber(analysis.threatRequests) }} {{ $t('nginx.requests') }}</el-tag>
                  </div>
                </template>
                <div ref="threatChartRef" class="chart-container" style="height: 220px"></div>
              </el-card>
            </el-col>
            <el-col :xs="24" :lg="14">
              <el-card shadow="never" class="rank-card threat-section">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.threatIPs') }}</span>
                </template>
                <el-table :data="analysis.threatIPs || []" size="small" stripe :show-header="true" max-height="260">
                  <el-table-column type="index" width="36" />
                  <el-table-column :label="$t('nginx.ip')" min-width="130">
                    <template #default="{ row }">
                      <span class="mono-text drilldown-link" @click="handleDrilldown('ip', row.name)">{{ row.name }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.location')" min-width="100">
                    <template #default="{ row }">
                      <span v-if="row.country" class="location-text">{{ row.country }}<template v-if="row.city"> / {{ row.city }}</template></span>
                      <span v-else class="muted-text">-</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.threatCount')" width="80" align="right">
                    <template #default="{ row }">
                      <span class="count-text threat-count">{{ formatNumber(row.count) }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.logStatus')" width="80" align="center">
                    <template #default="{ row }">
                      <el-tag v-if="row.banned" type="danger" size="small" effect="dark">{{ $t('nginx.banned') }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.actions')" width="80" align="center">
                    <template #default="{ row }">
                      <el-popconfirm v-if="!row.banned"
                        :title="$t('nginx.banConfirm', { ip: row.name })"
                        @confirm="handleBanIP(row.name)">
                        <template #reference>
                          <el-button type="danger" text size="small" :loading="banLoading[row.name]">{{ $t('nginx.ban') }}</el-button>
                        </template>
                      </el-popconfirm>
                      <el-popconfirm v-else
                        :title="$t('nginx.unbanConfirm', { ip: row.name })"
                        @confirm="handleUnbanIP(row.name)">
                        <template #reference>
                          <el-button type="warning" text size="small" :loading="banLoading[row.name]">{{ $t('nginx.unban') }}</el-button>
                        </template>
                      </el-popconfirm>
                    </template>
                  </el-table-column>
                </el-table>
              </el-card>
            </el-col>
          </el-row>

          <!-- 爬虫检测 -->
          <el-row v-if="analysis.crawlerRequests > 0" :gutter="16" style="margin-top: 16px">
            <el-col :xs="24" :lg="10">
              <el-card shadow="never" class="rank-card crawler-section">
                <template #header>
                  <div class="threat-header">
                    <span class="chart-title">{{ $t('nginx.crawlerDetection') }}</span>
                    <el-tag type="info" size="small" effect="plain">{{ formatNumber(analysis.crawlerRequests) }} {{ $t('nginx.requests') }}</el-tag>
                  </div>
                </template>
                <div ref="crawlerChartRef" class="chart-container" style="height: 220px"></div>
              </el-card>
            </el-col>
            <el-col :xs="24" :lg="14">
              <el-card shadow="never" class="rank-card crawler-section">
                <template #header>
                  <span class="chart-title">{{ $t('nginx.crawlerRanking') }}</span>
                </template>
                <el-table :data="analysis.topCrawlers || []" size="small" stripe :show-header="true" max-height="260">
                  <el-table-column type="index" width="36" />
                  <el-table-column :label="$t('nginx.crawlerName')" min-width="150">
                    <template #default="{ row }">
                      <span>{{ row.name }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.requests')" width="100" align="right">
                    <template #default="{ row }">
                      <span class="count-text">{{ formatNumber(row.count) }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.percentage')" width="100" align="right">
                    <template #default="{ row }">
                      <span class="muted-text">{{ analysis.totalRequests ? ((row.count / analysis.totalRequests) * 100).toFixed(1) : '0' }}%</span>
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
                  <el-table-column :label="$t('nginx.browser')" width="160">
                    <template #default="{ row }">
                      <span>{{ parseUA(row.name).browser }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.os')" width="150">
                    <template #default="{ row }">
                      <span>{{ parseUA(row.name).os }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column :label="$t('nginx.userAgent')" min-width="300" show-overflow-tooltip>
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

    <!-- 下钻弹窗 -->
    <el-dialog v-model="drilldownVisible" :title="drilldownTitle" width="720px" destroy-on-close>
      <div v-loading="drilldownLoading">
        <template v-if="drilldownURLs.length > 0">
          <h4 style="margin: 0 0 8px; font-size: 13px; color: var(--xp-text-secondary)">{{ $t('nginx.relatedURLs') }}</h4>
          <el-table :data="drilldownURLs" size="small" stripe max-height="200" style="margin-bottom: 16px">
            <el-table-column type="index" width="36" />
            <el-table-column :label="$t('nginx.url')" min-width="300">
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
        </template>

        <h4 v-if="drilldownURLs.length > 0" style="margin: 0 0 8px; font-size: 13px; color: var(--xp-text-secondary)">{{ $t('nginx.relatedIPs') }}</h4>
        <el-table :data="drilldownIPs" size="small" stripe max-height="360">
          <el-table-column type="index" width="36" />
          <el-table-column :label="$t('nginx.ip')" min-width="130">
            <template #default="{ row }">
              <span class="mono-text">{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('nginx.location')" min-width="100">
            <template #default="{ row }">
              <span v-if="row.country" class="location-text">{{ row.country }}<template v-if="row.city"> / {{ row.city }}</template></span>
              <span v-else class="muted-text">-</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('nginx.requests')" width="80" align="right">
            <template #default="{ row }">
              <span class="count-text">{{ formatNumber(row.count) }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('nginx.logStatus')" width="80" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.banned" type="danger" size="small" effect="dark">{{ $t('nginx.banned') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('nginx.actions')" width="80" align="center">
            <template #default="{ row }">
              <el-popconfirm v-if="!row.banned"
                :title="$t('nginx.banConfirm', { ip: row.name })"
                @confirm="handleBanIP(row.name)">
                <template #reference>
                  <el-button type="danger" text size="small" :loading="banLoading[row.name]">{{ $t('nginx.ban') }}</el-button>
                </template>
              </el-popconfirm>
              <el-popconfirm v-else
                :title="$t('nginx.unbanConfirm', { ip: row.name })"
                @confirm="handleUnbanIP(row.name)">
                <template #reference>
                  <el-button type="warning" text size="small" :loading="banLoading[row.name]">{{ $t('nginx.unban') }}</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>

        <el-empty v-if="!drilldownLoading && drilldownIPs.length === 0" :description="$t('nginx.noLogData')" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh } from '@element-plus/icons-vue'
import { detectNginxSites, analyzeNginxSiteLog, tailNginxLog, drilldownNginxLog } from '@/api/modules/website'
import { banFail2banIP, unbanFail2banIP } from '@/api/modules/toolbox'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import { UAParser } from 'ua-parser-js'

const { t } = useI18n()

const uaParser = new UAParser()
const uaCache = new Map<string, { browser: string; os: string }>()

const parseUA = (ua: string): { browser: string; os: string } => {
  if (!ua) return { browser: '-', os: '-' }
  const cached = uaCache.get(ua)
  if (cached) return cached
  uaParser.setUA(ua)
  const b = uaParser.getBrowser()
  const o = uaParser.getOS()
  const result = {
    browser: [b.name, b.version?.split('.')[0]].filter(Boolean).join(' ') || ua.split('/')[0] || '-',
    os: [o.name, o.version].filter(Boolean).join(' ') || '-',
  }
  uaCache.set(ua, result)
  return result
}

const selectedSite = ref('')
const timeRange = ref('24h')
const subTab = ref('overview')
const analyzing = ref(false)

const sites = ref<any[]>([])
const analysis = reactive<any>({
  totalRequests: 0, uniqueIPs: 0, totalBytes: 0, errorRate: 0,
  statusCodes: {}, topUrls: [], topIps: [], topUserAgents: [],
  hourlyStats: [], dailyStats: [],
  threatRequests: 0, threatIPs: [], topThreats: [],
  crawlerRequests: 0, topCrawlers: [],
})

const drilldownVisible = ref(false)
const drilldownLoading = ref(false)
const drilldownTitle = ref('')
const drilldownIPs = ref<any[]>([])
const drilldownURLs = ref<any[]>([])

const trendChartRef = ref<HTMLElement>()
const statusChartRef = ref<HTMLElement>()
const threatChartRef = ref<HTMLElement>()
const crawlerChartRef = ref<HTMLElement>()
let trendChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null
let threatChart: echarts.ECharts | null = null
let crawlerChart: echarts.ECharts | null = null

const logLines = ref(200)
const errorLogLines = ref(200)
const accessLogContent = ref('')
const accessLogPath = ref('')
const accessLogLoading = ref(false)
const errorLogContent = ref('')
const errorLogPath = ref('')
const errorLogLoading = ref(false)
const banLoading = reactive<Record<string, boolean>>({})

const errorRateClass = computed(() => {
  const rate = analysis.errorRate || 0
  if (rate > 10) return 'error-card'
  if (rate > 5) return 'warning-card'
  return ''
})

const isHighTraffic = (row: any) => {
  if (!analysis.totalRequests || analysis.totalRequests === 0) return false
  return row.count / analysis.totalRequests > 0.3
}

const ipRowClass = ({ row }: { row: any }) => {
  if (row.banned) return 'banned-row'
  if (isHighTraffic(row)) return 'high-traffic-row'
  return ''
}

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

const emptyAnalysis = {
  totalRequests: 0, uniqueIPs: 0, totalBytes: 0, errorRate: 0,
  statusCodes: {}, topUrls: [], topIps: [], topUserAgents: [],
  hourlyStats: [], dailyStats: [],
  threatRequests: 0, threatIPs: [], topThreats: [],
  crawlerRequests: 0, topCrawlers: [],
}

const handleAnalyze = async () => {
  analyzing.value = true
  try {
    const res = await analyzeNginxSiteLog({ site: selectedSite.value, timeRange: timeRange.value })
    Object.assign(analysis, { ...emptyAnalysis, ...res.data })
    await nextTick()
    renderCharts()
  } catch {
    Object.assign(analysis, emptyAnalysis)
  } finally {
    analyzing.value = false
  }
}

const handleBanIP = async (ip: string) => {
  banLoading[ip] = true
  try {
    await banFail2banIP(ip)
    ElMessage.success(t('nginx.banSuccess', { ip }))
    handleAnalyze()
  } catch {} finally { banLoading[ip] = false }
}

const handleUnbanIP = async (ip: string) => {
  banLoading[ip] = true
  try {
    await unbanFail2banIP(ip, 'sshd')
    ElMessage.success(t('nginx.unbanSuccess', { ip }))
    handleAnalyze()
  } catch {} finally { banLoading[ip] = false }
}

const handleDrilldown = async (filterType: string, filterValue: string) => {
  if (filterType === 'url') {
    drilldownTitle.value = `${t('nginx.drilldownURL')}: ${filterValue}`
  } else if (filterType === 'ip') {
    drilldownTitle.value = `${t('nginx.drilldownIP')}: ${filterValue}`
  } else {
    drilldownTitle.value = `${t('nginx.drilldownThreat')}: ${filterValue}`
  }
  drilldownIPs.value = []
  drilldownURLs.value = []
  drilldownVisible.value = true
  drilldownLoading.value = true
  try {
    const res = await drilldownNginxLog({
      site: selectedSite.value,
      timeRange: timeRange.value,
      filterType,
      filterValue,
    })
    drilldownIPs.value = res.data?.ips || []
    drilldownURLs.value = res.data?.urls || []
  } catch { /* empty */ }
  finally { drilldownLoading.value = false }
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

const threatColors = ['#f56c6c', '#e6a23c', '#ff8c6b', '#c45656', '#fab6b6', '#b88230', '#f89898']

const renderCharts = () => {
  renderTrendChart()
  renderStatusChart()
  if (analysis.threatRequests > 0) {
    nextTick(() => renderThreatChart())
  }
  if (analysis.crawlerRequests > 0) {
    nextTick(() => renderCrawlerChart())
  }
}

const renderTrendChart = () => {
  if (!trendChartRef.value) return
  if (!trendChart) {
    trendChart = echarts.init(trendChartRef.value)
  }

  const isHourly = ['1h', '6h', '24h'].includes(timeRange.value)
  const stats = isHourly ? (analysis.hourlyStats || []) : (analysis.dailyStats || [])

  const xData = stats.map((s: any) => {
    if (isHourly) return s.time.substring(11, 16)
    return s.time.substring(5)
  })
  const reqData = stats.map((s: any) => s.requests)
  const byteData = stats.map((s: any) => +(s.bytes / 1024).toFixed(1))

  trendChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    grid: { left: 50, right: 50, top: 30, bottom: 30 },
    xAxis: { type: 'category', data: xData, axisLabel: { fontSize: 11 } },
    yAxis: [
      { type: 'value', name: t('nginx.requests'), axisLabel: { fontSize: 11 } },
      { type: 'value', name: 'KB', axisLabel: { fontSize: 11 } },
    ],
    series: [
      {
        name: t('nginx.requests'), type: 'bar', data: reqData,
        itemStyle: { color: '#409eff', borderRadius: [3, 3, 0, 0] }, barMaxWidth: 20,
      },
      {
        name: t('nginx.totalTraffic'), type: 'line', yAxisIndex: 1, data: byteData,
        smooth: true, lineStyle: { color: '#e6a23c', width: 2 },
        itemStyle: { color: '#e6a23c' }, areaStyle: { color: 'rgba(230,162,60,0.1)' },
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
    name, value, itemStyle: { color: statusColors[name] || '#909399' },
  }))

  if (data.length === 0) { statusChart.clear(); return }

  statusChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie', radius: ['40%', '70%'], center: ['50%', '50%'],
      avoidLabelOverlap: true, label: { show: true, formatter: '{b}\n{d}%', fontSize: 12 }, data,
    }],
  }, true)
}

const renderThreatChart = () => {
  if (!threatChartRef.value) return
  if (!threatChart) {
    threatChart = echarts.init(threatChartRef.value)
    threatChart.on('click', (params: any) => {
      if (params.name) handleDrilldown('threat', params.name)
    })
  }

  const threats = analysis.topThreats || []
  if (threats.length === 0) { threatChart.clear(); return }

  const names = threats.map((t: any) => t.name)
  const values = threats.map((t: any) => t.count)

  threatChart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    grid: { left: 110, right: 40, top: 10, bottom: 20 },
    xAxis: { type: 'value', axisLabel: { fontSize: 11 } },
    yAxis: { type: 'category', data: names.reverse(), axisLabel: { fontSize: 11, width: 90, overflow: 'truncate' } },
    series: [{
      type: 'bar', data: values.reverse(), barMaxWidth: 18, cursor: 'pointer',
      itemStyle: { borderRadius: [0, 3, 3, 0], color: (params: any) => threatColors[params.dataIndex % threatColors.length] },
    }],
  }, true)
}

const crawlerColors = ['#67c23a', '#409eff', '#e6a23c', '#f56c6c', '#909399', '#b37feb', '#36cfc9', '#ff85c0', '#ffc53d', '#597ef7', '#73d13d', '#ff7a45', '#9254de']

const renderCrawlerChart = () => {
  if (!crawlerChartRef.value) return
  if (!crawlerChart) {
    crawlerChart = echarts.init(crawlerChartRef.value)
  }
  const crawlers = analysis.topCrawlers || []
  if (crawlers.length === 0) { crawlerChart.clear(); return }

  const data = crawlers.map((c: any, i: number) => ({
    name: c.name, value: c.count,
    itemStyle: { color: crawlerColors[i % crawlerColors.length] },
  }))

  crawlerChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie', radius: ['35%', '65%'], center: ['50%', '50%'],
      avoidLabelOverlap: true,
      label: { show: true, formatter: '{b}\n{d}%', fontSize: 11 },
      data,
    }],
  }, true)
}

const handleResize = () => {
  trendChart?.resize()
  statusChart?.resize()
  threatChart?.resize()
  crawlerChart?.resize()
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
  threatChart?.dispose()
  crawlerChart?.dispose()
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

      &.warning-card {
        border-color: var(--el-color-warning-light-5);
        .summary-value { color: var(--el-color-warning); }
      }

      &.threat-card {
        border-color: var(--el-color-danger-light-5);
        background: var(--el-color-danger-light-9);
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

  .threat-section {
    :deep(.el-card__header) {
      border-left: 3px solid var(--el-color-danger);
    }
  }

  .threat-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .threat-count { color: var(--el-color-danger) !important; }

  .high-traffic-tag {
    font-size: 11px;
    color: var(--el-color-warning);
    font-weight: 600;
  }

  :deep(.banned-row) {
    background-color: var(--el-color-danger-light-9) !important;
  }

  :deep(.high-traffic-row) {
    background-color: var(--el-color-warning-light-9) !important;
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

  .drilldown-link {
    cursor: pointer;
    color: var(--el-color-primary);
    &:hover { text-decoration: underline; }
  }

  .crawler-section {
    :deep(.el-card__header) {
      border-left: 3px solid var(--el-color-success);
    }
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
