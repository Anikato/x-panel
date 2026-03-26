<template>
  <div class="website-config-page" v-loading="loading">
    <div class="page-header">
      <div class="header-left">
        <el-button size="small" :icon="ArrowLeft" @click="router.push('/website/websites')" />
        <h3>{{ detail.primaryDomain || '...' }}</h3>
        <el-tag :type="detail.status === 'running' ? 'success' : 'danger'" size="small" effect="dark" round>
          {{ detail.status === 'running' ? $t('website.running') : $t('website.stopped') }}
        </el-tag>
        <el-tag :type="detail.type === 'static' ? 'info' : 'warning'" size="small">
          {{ detail.type === 'static' ? $t('website.typeStatic') : $t('website.typeProxy') }}
        </el-tag>
      </div>
      <div class="header-right">
        <el-radio-group v-model="configMode" size="small" class="mode-switcher" @change="handleModeSwitch">
          <el-radio-button value="managed">{{ $t('website.managedMode') }}</el-radio-button>
          <el-radio-button value="source">{{ $t('website.sourceMode') }}</el-radio-button>
        </el-radio-group>
        <el-button v-if="detail.status === 'stopped'" type="success" size="small" @click="handleEnable">{{ $t('website.enable') }}</el-button>
        <el-button v-else type="warning" size="small" @click="handleDisable">{{ $t('website.disable') }}</el-button>
      </div>
    </div>

    <!-- 源码模式提示 -->
    <el-alert
      v-if="configMode === 'source'"
      :title="$t('website.sourceModeHint')"
      type="info"
      show-icon
      :closable="false"
      class="mode-alert"
    />

    <el-tabs v-if="configMode === 'managed'" v-model="activeTab" class="config-tabs">
      <!-- 基本设置 -->
      <el-tab-pane :label="$t('website.basicSetting')" name="basic">
        <el-form :model="detail" label-width="120px" class="config-form">
          <el-form-item :label="$t('website.domain')">
            <el-input v-model="detail.primaryDomain" />
          </el-form-item>
          <el-form-item :label="$t('website.otherDomains')">
            <el-input v-model="detail.domains" :placeholder="$t('website.otherDomainsHint')" />
          </el-form-item>
          <template v-if="detail.type === 'static'">
            <el-form-item :label="$t('website.siteDir')">
              <el-input v-model="detail.siteDir" />
            </el-form-item>
            <el-form-item :label="$t('website.indexFile')">
              <el-input v-model="detail.indexFile" />
              <div class="form-tip">{{ $t('website.indexFileHint') }}</div>
            </el-form-item>
          </template>
          <el-form-item :label="$t('website.defaultServer')">
            <el-switch v-model="detail.defaultServer" />
            <div class="form-tip">{{ $t('website.defaultServerHint') }}</div>
          </el-form-item>
          <el-form-item :label="$t('commons.description')">
            <el-input v-model="detail.remark" type="textarea" :rows="2" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 反向代理 (仅反向代理类型) -->
      <el-tab-pane v-if="detail.type === 'reverse_proxy'" :label="$t('website.proxySetting')" name="proxy">
        <el-form :model="detail" label-width="120px" class="config-form">
          <el-form-item :label="$t('website.proxyPass')">
            <el-input v-model="detail.proxyPass" placeholder="http://127.0.0.1:8080" />
            <div class="form-tip">{{ $t('website.proxyPassHint') }}</div>
          </el-form-item>
          <el-form-item :label="$t('website.webSocket')">
            <el-switch v-model="detail.webSocket" />
          </el-form-item>
          <el-form-item :label="$t('website.upstream')">
            <el-input v-model="detail.upstream" type="textarea" :rows="6" class="code-textarea" :placeholder="$t('website.upstreamPlaceholder')" />
            <div class="form-tip">{{ $t('website.upstreamHint') }}</div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- HTTPS -->
      <el-tab-pane :label="$t('website.httpsSetting')" name="https">
        <el-form :model="detail" label-width="120px" class="config-form">
          <el-form-item :label="$t('website.sslEnable')">
            <el-switch v-model="detail.sslEnable" />
          </el-form-item>
          <template v-if="detail.sslEnable">
            <el-form-item :label="$t('website.selectCert')">
              <el-select v-model="detail.certificateID" style="width: 100%">
                <el-option v-for="c in certList" :key="c.id" :label="`${c.primaryDomain} (${c.status === 'applied' ? '已签发' : c.status})`" :value="c.id" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('website.httpConfig')">
              <el-select v-model="detail.httpConfig" style="width: 100%">
                <el-option :label="$t('website.httpConfigHTTPSRedirect')" value="HTTPSRedirect" />
                <el-option :label="$t('website.httpConfigHTTPAlso')" value="HTTPAlso" />
                <el-option :label="$t('website.httpConfigHTTPSOnly')" value="httpsOnly" />
                <el-option :label="$t('website.httpConfigHTTPOnly')" value="httpOnly" />
              </el-select>
            </el-form-item>
            <el-form-item label="HTTP/2">
              <el-switch v-model="detail.http2Enable" />
              <div class="form-tip">{{ $t('website.http2Hint') }}</div>
            </el-form-item>
            <el-form-item :label="$t('website.hsts')">
              <el-switch v-model="detail.hsts" />
              <div class="form-tip">{{ $t('website.hstsHint') }}</div>
            </el-form-item>
            <el-form-item :label="$t('website.sslProtocols')">
              <el-input v-model="detail.sslProtocols" placeholder="TLSv1.2 TLSv1.3" />
            </el-form-item>
          </template>
          <el-form-item>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 伪静态 -->
      <el-tab-pane :label="$t('website.rewriteSetting')" name="rewrite">
        <el-form :model="detail" label-width="0" class="config-form">
          <div class="rewrite-presets">
            <span class="preset-label">{{ $t('website.rewritePreset') }}:</span>
            <el-button size="small" plain v-for="p in rewritePresets" :key="p.name" @click="detail.rewrite = p.content">{{ p.name }}</el-button>
          </div>
          <el-input v-model="detail.rewrite" type="textarea" :rows="12" :placeholder="$t('website.rewriteHint')" class="code-textarea" />
          <div style="margin-top: 12px">
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </div>
        </el-form>
      </el-tab-pane>

      <!-- 重定向 -->
      <el-tab-pane :label="$t('website.redirectSetting')" name="redirect">
        <div class="redirect-section">
          <el-button size="small" type="primary" @click="addRedirect" style="margin-bottom: 12px">
            <el-icon><Plus /></el-icon>
            {{ $t('website.addRedirect') }}
          </el-button>
          <el-table :data="redirects" style="width: 100%">
            <el-table-column :label="$t('website.redirectSource')" min-width="180">
              <template #default="{ row }">
                <el-input v-model="row.source" size="small" placeholder="/old-path" />
              </template>
            </el-table-column>
            <el-table-column :label="$t('website.redirectTarget')" min-width="250">
              <template #default="{ row }">
                <el-input v-model="row.target" size="small" placeholder="https://new.com/path" />
              </template>
            </el-table-column>
            <el-table-column :label="$t('website.redirectType')" width="120">
              <template #default="{ row }">
                <el-select v-model="row.type" size="small">
                  <el-option label="301 永久" :value="301" />
                  <el-option label="302 临时" :value="302" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column width="60">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="redirects.splice($index, 1)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div style="margin-top: 12px">
            <el-button type="primary" @click="handleSaveRedirects" :loading="saving">{{ $t('commons.save') }}</el-button>
          </div>
        </div>
      </el-tab-pane>

      <!-- 流量限制 -->
      <el-tab-pane :label="$t('website.trafficSetting')" name="traffic">
        <el-form :model="detail" label-width="130px" class="config-form">
          <el-form-item :label="$t('website.limitRate')">
            <el-input v-model="detail.limitRate" placeholder="1m" style="width: 200px" />
            <div class="form-tip">{{ $t('website.limitRateHint') }}</div>
          </el-form-item>
          <el-form-item :label="$t('website.limitConn')">
            <el-input-number v-model="detail.limitConn" :min="0" :max="100000" />
            <div class="form-tip">{{ $t('website.limitConnHint') }}</div>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 安全 -->
      <el-tab-pane :label="$t('website.securitySetting')" name="security">
        <el-form :model="detail" label-width="130px" class="config-form">
          <el-divider content-position="left">{{ $t('website.basicAuth') }}</el-divider>
          <el-form-item :label="$t('website.basicAuth')">
            <el-switch v-model="detail.basicAuth" />
          </el-form-item>
          <template v-if="detail.basicAuth">
            <el-form-item :label="$t('website.basicUser')">
              <el-input v-model="detail.basicUser" style="width: 300px" />
            </el-form-item>
            <el-form-item :label="$t('website.basicPassword')">
              <el-input v-model="detail.basicPassword" type="password" show-password style="width: 300px" />
            </el-form-item>
          </template>

          <el-divider content-position="left">{{ $t('website.antiLeech') }}</el-divider>
          <el-form-item :label="$t('website.antiLeech')">
            <el-switch v-model="detail.antiLeech" />
          </el-form-item>
          <template v-if="detail.antiLeech">
            <el-form-item :label="$t('website.leechReferers')">
              <el-input v-model="detail.leechReferers" type="textarea" :rows="3" />
              <div class="form-tip">{{ $t('website.leechReferersHint') }}</div>
            </el-form-item>
          </template>

          <el-form-item>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <!-- 日志 -->
      <el-tab-pane :label="$t('website.logSetting')" name="log">
        <el-form :model="detail" label-width="120px" class="config-form">
          <el-form-item :label="$t('website.accessLog')">
            <el-switch v-model="detail.accessLog" />
          </el-form-item>
          <el-form-item :label="$t('website.errorLog')">
            <el-switch v-model="detail.errorLog" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </el-form-item>
        </el-form>

        <el-divider />
        <div class="log-viewer-section">
          <div class="log-toolbar">
            <el-radio-group v-model="logType" size="small">
              <el-radio-button value="access">{{ $t('website.accessLog') }}</el-radio-button>
              <el-radio-button value="error">{{ $t('website.errorLog') }}</el-radio-button>
            </el-radio-group>
            <el-button size="small" @click="loadLog">{{ $t('website.viewLog') }}</el-button>
          </div>
          <div class="log-container" v-if="logContent !== null">
            <pre class="log-content">{{ logContent || '暂无日志' }}</pre>
          </div>
        </div>
      </el-tab-pane>

      <!-- 日志分析 -->
      <el-tab-pane :label="$t('website.logAnalysis')" name="logAnalysis">
        <div class="log-analysis-section">
          <div class="analysis-toolbar">
            <el-radio-group v-model="analysisDays" size="small" @change="loadLogAnalysis">
              <el-radio-button :value="1">{{ $t('website.today') }}</el-radio-button>
              <el-radio-button :value="7">{{ $t('website.last7days') }}</el-radio-button>
              <el-radio-button :value="30">{{ $t('website.last30days') }}</el-radio-button>
            </el-radio-group>
            <el-button size="small" @click="loadLogAnalysis" :loading="analysisLoading">{{ $t('commons.refresh') }}</el-button>
          </div>

          <!-- 概览卡片 -->
          <div v-if="logAnalysisData" class="analysis-overview">
            <el-row :gutter="16">
              <el-col :span="6">
                <el-card shadow="never" class="stat-card">
                  <div class="stat-value">{{ formatNumber(logAnalysisData.totalRequests) }}</div>
                  <div class="stat-label">{{ $t('website.totalRequests') }}</div>
                </el-card>
              </el-col>
              <el-col :span="6">
                <el-card shadow="never" class="stat-card">
                  <div class="stat-value">{{ formatNumber(logAnalysisData.uniqueIPs) }}</div>
                  <div class="stat-label">{{ $t('website.uniqueVisitors') }}</div>
                </el-card>
              </el-col>
              <el-col :span="6">
                <el-card shadow="never" class="stat-card">
                  <div class="stat-value">{{ formatBytes(logAnalysisData.totalBytes) }}</div>
                  <div class="stat-label">{{ $t('website.totalTraffic') }}</div>
                </el-card>
              </el-col>
              <el-col :span="6">
                <el-card shadow="never" class="stat-card" :class="{ 'error-card': logAnalysisData.errorRate > 5 }">
                  <div class="stat-value">{{ logAnalysisData.errorRate.toFixed(1) }}%</div>
                  <div class="stat-label">{{ $t('website.errorRate') }}</div>
                </el-card>
              </el-col>
            </el-row>
          </div>

          <!-- 图表 -->
          <div v-if="logAnalysisData" class="analysis-charts">
            <el-row :gutter="16">
              <el-col :span="16">
                <el-card shadow="never">
                  <template #header>{{ $t('website.requestTrend') }}</template>
                  <div ref="trendChartRef" class="chart-container" />
                </el-card>
              </el-col>
              <el-col :span="8">
                <el-card shadow="never">
                  <template #header>{{ $t('website.statusCodeDist') }}</template>
                  <div ref="statusChartRef" class="chart-container" />
                </el-card>
              </el-col>
            </el-row>
          </div>

          <!-- Top 排行 -->
          <div v-if="logAnalysisData" class="analysis-rankings">
            <el-row :gutter="16">
              <el-col :span="12">
                <el-card shadow="never">
                  <template #header>{{ $t('website.topURLs') }}</template>
                  <el-table :data="logAnalysisData.topUrls || []" size="small" stripe>
                    <el-table-column label="URL" prop="name" show-overflow-tooltip />
                    <el-table-column :label="$t('website.visits')" prop="count" width="100" align="right" />
                  </el-table>
                </el-card>
              </el-col>
              <el-col :span="12">
                <el-card shadow="never">
                  <template #header>{{ $t('website.topIPs') }}</template>
                  <el-table :data="logAnalysisData.topIps || []" size="small" stripe>
                    <el-table-column label="IP" prop="name" />
                    <el-table-column :label="$t('website.visits')" prop="count" width="100" align="right" />
                  </el-table>
                </el-card>
              </el-col>
            </el-row>
          </div>

          <el-empty v-if="logAnalysisData && logAnalysisData.totalRequests === 0" :description="$t('website.noLogData')" />
        </div>
      </el-tab-pane>

      <!-- 自定义配置 -->
      <el-tab-pane :label="$t('website.customSetting')" name="custom">
        <el-form :model="detail" label-width="0" class="config-form">
          <div class="form-tip" style="margin-bottom: 8px">{{ $t('website.customNginxHint') }}</div>
          <el-input v-model="detail.customNginx" type="textarea" :rows="12" class="code-textarea" />
          <div style="margin-top: 12px">
            <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
          </div>
        </el-form>
      </el-tab-pane>

      <!-- 配置预览 (托管模式) -->
      <el-tab-pane :label="$t('website.configPreview')" name="preview">
        <div class="config-preview">
          <el-button size="small" style="margin-bottom: 8px" @click="loadDetail">{{ $t('commons.refresh') }}</el-button>
          <pre class="preview-content">{{ detail.nginxConfig || '暂无配置' }}</pre>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 源码编辑模式 -->
    <div v-if="configMode === 'source'" class="source-editor-section">
      <div class="source-editor-toolbar">
        <span class="source-file-label">{{ detail.alias }}.conf</span>
        <div class="source-actions">
          <el-button size="small" @click="loadSourceConf">{{ $t('commons.refresh') }}</el-button>
          <el-button size="small" type="primary" @click="handleSaveSource" :loading="sourceSaving">{{ $t('commons.save') }}</el-button>
        </div>
      </div>
      <div ref="monacoContainerRef" class="source-editor-container" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Delete, Plus } from '@element-plus/icons-vue'
import { getWebsiteDetail, updateWebsite, enableWebsite, disableWebsite, getWebsiteLog, getSiteConfContent, saveSiteConfContent, switchConfigMode, analyzeNginxLog } from '@/api/modules/website'
import { searchCertificate } from '@/api/modules/ssl'
import type { Certificate } from '@/api/interface'
import * as monaco from 'monaco-editor'
import * as echarts from 'echarts/core'
import { BarChart, PieChart, LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'

echarts.use([BarChart, PieChart, LineChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent, CanvasRenderer])

interface WebsiteDetail {
  id: number
  primaryDomain: string
  domains: string
  type: string
  status: string
  sslEnable: boolean
  remark: string
  siteDir: string
  proxyPass: string
  indexFile: string
  defaultServer: boolean
  webSocket: boolean
  certificateID: number
  httpConfig: string
  http2Enable: boolean
  hsts: boolean
  sslProtocols: string
  rewrite: string
  redirects: string
  limitRate: string
  limitConn: number
  basicAuth: boolean
  basicUser: string
  basicPassword: string
  antiLeech: boolean
  leechReferers: string
  accessLog: boolean
  errorLog: boolean
  upstream: string
  customNginx: string
  nginxConfig: string
  alias: string
  configMode: string
}

interface RedirectItem {
  source: string
  target: string
  type: number
}

interface TimeStat {
  time: string
  requests: number
  bytes: number
}

interface LogAnalysisData {
  totalRequests: number
  uniqueIPs: number
  totalBytes: number
  errorRate: number
  hourlyStats: TimeStat[]
  dailyStats: TimeStat[]
  statusCodes: Record<string, number>
  topUrls: { name: string; count: number }[]
  topIps: { name: string; count: number }[]
}

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const activeTab = ref('basic')
const detail = ref<Partial<WebsiteDetail>>({})
const certList = ref<Certificate[]>([])
const redirects = ref<RedirectItem[]>([])

// Config mode
const configMode = ref<'managed' | 'source'>('managed')
const sourceSaving = ref(false)
const monacoContainerRef = ref<HTMLElement>()
let monacoEditor: monaco.editor.IStandaloneCodeEditor | null = null

// 日志
const logType = ref('access')
const logContent = ref<string | null>(null)

// 日志分析
const analysisDays = ref(1)
const analysisLoading = ref(false)
const logAnalysisData = ref<LogAnalysisData | null>(null)
const trendChartRef = ref<HTMLElement>()
const statusChartRef = ref<HTMLElement>()
let trendChart: echarts.ECharts | null = null
let statusChart: echarts.ECharts | null = null

const siteId = Number(route.params.id)

const rewritePresets = [
  { name: 'Vue/React SPA', content: 'location / {\n    try_files $uri $uri/ /index.html;\n}' },
  { name: 'WordPress', content: 'location / {\n    try_files $uri $uri/ /index.php?$args;\n}' },
  { name: 'Laravel', content: 'location / {\n    try_files $uri $uri/ /index.php?$query_string;\n}' },
]

const loadDetail = async () => {
  loading.value = true
  try {
    const res = await getWebsiteDetail(siteId)
    detail.value = res.data || {}
    try {
      redirects.value = detail.value.redirects ? JSON.parse(detail.value.redirects) : []
    } catch { redirects.value = [] }
  } catch {
    router.push('/website/websites')
  } finally {
    loading.value = false
  }
}

const loadCerts = async () => {
  try {
    const res = await searchCertificate({ page: 1, pageSize: 100 })
    certList.value = res.data?.items || []
  } catch { certList.value = [] }
}

const handleSave = async () => {
  saving.value = true
  try {
    await updateWebsite({ ...detail.value })
    ElMessage.success(t('commons.success'))
    loadDetail()
  } catch {}
  finally { saving.value = false }
}

const handleSaveRedirects = async () => {
  detail.value.redirects = JSON.stringify(redirects.value)
  await handleSave()
}

const handleEnable = async () => {
  try {
    await ElMessageBox.confirm(t('website.enableConfirm'), t('commons.tip'))
    await enableWebsite(siteId)
    ElMessage.success(t('commons.success'))
    loadDetail()
  } catch {}
}

const handleDisable = async () => {
  try {
    await ElMessageBox.confirm(t('website.disableConfirm'), t('commons.tip'), { type: 'warning' })
    await disableWebsite(siteId)
    ElMessage.success(t('commons.success'))
    loadDetail()
  } catch {}
}

const addRedirect = () => {
  redirects.value.push({ source: '', target: '', type: 301 })
}

const loadLog = async () => {
  try {
    const res = await getWebsiteLog({ id: siteId, type: logType.value, tail: 200 })
    logContent.value = res.data || '暂无日志'
  } catch { logContent.value = '获取日志失败' }
}

// --- 日志分析 ---

const loadLogAnalysis = async () => {
  analysisLoading.value = true
  try {
    const res = await analyzeNginxLog(siteId, analysisDays.value)
    logAnalysisData.value = res.data
    await nextTick()
    renderTrendChart()
    renderStatusChart()
  } catch { /* ignore */ }
  finally { analysisLoading.value = false }
}

const renderTrendChart = () => {
  if (!trendChartRef.value || !logAnalysisData.value) return
  if (trendChart) trendChart.dispose()
  trendChart = echarts.init(trendChartRef.value)

  const data = analysisDays.value <= 1 ? logAnalysisData.value.hourlyStats : logAnalysisData.value.dailyStats
  if (!data || !data.length) { trendChart.clear(); return }

  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: data.map((p: TimeStat) => analysisDays.value <= 1 ? p.time.slice(11) : p.time.slice(5)),
      axisLabel: { fontSize: 11 },
    },
    yAxis: [
      { type: 'value', name: '请求数', axisLabel: { fontSize: 11 } },
      { type: 'value', name: '流量', axisLabel: { fontSize: 11, formatter: (v: number) => formatBytes(v) } },
    ],
    series: [
      { name: '请求数', type: 'bar', data: data.map((p: TimeStat) => p.requests), itemStyle: { color: '#409EFF' } },
      { name: '流量', type: 'line', yAxisIndex: 1, data: data.map((p: TimeStat) => p.bytes), itemStyle: { color: '#67C23A' }, smooth: true },
    ],
  })
}

const renderStatusChart = () => {
  if (!statusChartRef.value || !logAnalysisData.value) return
  if (statusChart) statusChart.dispose()
  statusChart = echarts.init(statusChartRef.value)

  const codes = logAnalysisData.value.statusCodes || {}
  const colorMap: Record<string, string> = { '2xx': '#67C23A', '3xx': '#409EFF', '4xx': '#E6A23C', '5xx': '#F56C6C' }
  const data = Object.entries(codes).map(([name, value]) => ({
    name,
    value,
    itemStyle: { color: colorMap[name] || '#909399' },
  }))

  if (!data.length) { statusChart.clear(); return }

  statusChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      data,
      label: { formatter: '{b}\n{d}%', fontSize: 12 },
    }],
  })
}

const formatNumber = (n: number) => {
  if (!n) return '0'
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) { val /= 1024; i++ }
  return val.toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}

// --- 源码模式 ---

const initMonacoEditor = (content: string) => {
  if (monacoEditor) {
    monacoEditor.setValue(content)
    return
  }
  if (!monacoContainerRef.value) return
  monacoEditor = monaco.editor.create(monacoContainerRef.value, {
    value: content,
    language: 'plaintext',
    theme: 'vs-dark',
    fontSize: 13,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
    minimap: { enabled: false },
    scrollBeyondLastLine: false,
    lineNumbers: 'on',
    automaticLayout: true,
    tabSize: 4,
    wordWrap: 'on',
  })
}

const disposeMonacoEditor = () => {
  if (monacoEditor) {
    monacoEditor.dispose()
    monacoEditor = null
  }
}

const loadSourceConf = async () => {
  try {
    const res = await getSiteConfContent(siteId)
    const content = res.data || ''
    await nextTick()
    initMonacoEditor(content)
  } catch { ElMessage.error('加载配置失败') }
}

const handleSaveSource = async () => {
  if (!monacoEditor) return
  const content = monacoEditor.getValue()
  if (!content.trim()) { ElMessage.warning('配置内容不能为空'); return }
  sourceSaving.value = true
  try {
    await saveSiteConfContent(siteId, content)
    ElMessage.success(t('commons.success'))
  } catch {}
  finally { sourceSaving.value = false }
}

const handleModeSwitch = async (val: string | number | boolean) => {
  const mode = val as string
  if (mode === 'source') {
    try {
      await switchConfigMode(siteId, 'source')
      await nextTick()
      loadSourceConf()
    } catch {
      configMode.value = 'managed'
    }
  } else {
    try {
      await ElMessageBox.confirm(t('website.switchToManagedConfirm'), t('commons.tip'), { type: 'warning' })
      await switchConfigMode(siteId, 'managed')
      disposeMonacoEditor()
      loadDetail()
    } catch {
      configMode.value = 'source'
    }
  }
}

watch(configMode, (val) => {
  if (val !== 'source') {
    disposeMonacoEditor()
  }
})

watch(activeTab, (val) => {
  if (val === 'logAnalysis' && !logAnalysisData.value) {
    loadLogAnalysis()
  }
})

onMounted(() => {
  loadDetail().then(() => {
    configMode.value = detail.value.configMode === 'source' ? 'source' : 'managed'
    if (configMode.value === 'source') {
      nextTick(() => loadSourceConf())
    }
  })
  loadCerts()
})

onBeforeUnmount(() => {
  disposeMonacoEditor()
  if (trendChart) { trendChart.dispose(); trendChart = null }
  if (statusChart) { statusChart.dispose(); statusChart = null }
})
</script>

<style lang="scss" scoped>
.website-config-page {
  height: 100%;
}

.page-header .header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.config-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 16px;
  }
}

.config-form {
  max-width: 700px;
}

.form-tip {
  font-size: 12px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}

.code-textarea {
  :deep(textarea) {
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    font-size: 13px;
    line-height: 1.6;
  }
}

.rewrite-presets {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .preset-label {
    font-size: 13px;
    color: var(--xp-text-secondary);
  }
}

.log-viewer-section {
  .log-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
  }

  .log-container {
    background: var(--xp-bg-inset);
    border-radius: 6px;
    padding: 16px;
    max-height: 450px;
    overflow-y: auto;
  }

  .log-content {
    font-family: var(--xp-font-mono);
    font-size: 12px;
    line-height: 1.7;
    color: var(--xp-text-secondary);
    white-space: pre-wrap;
    word-break: break-all;
    margin: 0;
  }
}

.config-preview {
  .preview-content {
    background: var(--xp-bg-inset);
    border-radius: 6px;
    padding: 16px;
    font-family: var(--xp-font-mono);
    font-size: 13px;
    line-height: 1.6;
    color: var(--xp-text-secondary);
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 600px;
    overflow-y: auto;
    margin: 0;
  }
}

.redirect-section {
  max-width: 900px;
}

.mode-switcher {
  margin-right: 12px;
}

.mode-alert {
  margin-bottom: 16px;
}

.source-editor-section {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 250px);
  min-height: 400px;
}

.source-editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  margin-bottom: 8px;

  .source-file-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--xp-accent);
    font-family: 'Fira Code', 'Consolas', monospace;
  }

  .source-actions {
    display: flex;
    gap: 8px;
  }
}

.source-editor-container {
  flex: 1;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--xp-border-light);
}

.log-analysis-section {
  .analysis-toolbar {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }

  .analysis-overview {
    margin-bottom: 16px;
  }

  .stat-card {
    text-align: center;
    .stat-value {
      font-size: 28px;
      font-weight: 700;
      color: var(--xp-text-primary);
      line-height: 1.2;
    }
    .stat-label {
      font-size: 13px;
      color: var(--xp-text-secondary);
      margin-top: 4px;
    }
    &.error-card .stat-value { color: var(--el-color-danger); }
  }

  .analysis-charts {
    margin-bottom: 16px;
  }

  .chart-container {
    height: 300px;
    width: 100%;
  }

  .analysis-rankings {
    margin-bottom: 16px;
  }
}
</style>
