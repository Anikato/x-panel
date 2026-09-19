<template>
  <div class="website-page xp-page-shell">
    <div class="filter-bar">
      <el-input v-model="searchInfo" :placeholder="$t('commons.search')" prefix-icon="Search" clearable class="search-input" @input="onFilterChange" />
      <el-select v-model="filterType" clearable :placeholder="$t('website.type')" class="filter-select" @change="onFilterChange">
        <el-option :label="$t('website.typeStatic')" value="static" />
        <el-option :label="$t('website.typeProxy')" value="reverse_proxy" />
      </el-select>
      <el-select v-model="filterStatus" clearable :placeholder="$t('website.status')" class="filter-select" @change="onFilterChange">
        <el-option :label="$t('website.running')" value="running" />
        <el-option :label="$t('website.stopped')" value="stopped" />
      </el-select>
      <span v-if="hasFilters" class="filter-count">{{ $t('website.filterCount', { count: total }) }}</span>
      <el-button v-if="hasFilters" link type="primary" @click="clearFilters">{{ $t('website.clearFilters') }}</el-button>
      <div class="header-actions">
        <el-button :loading="certificateBatchLoading" @click="handleCertificateBatch">
          {{ $t('website.checkAllCertificates') }}
        </el-button>
        <el-button @click="openImportDialog">{{ $t('website.importExisting') }}</el-button>
        <el-button type="primary" @click="openCreateDialog">
          <el-icon><Plus /></el-icon>
          {{ $t('website.create') }}
        </el-button>
      </div>
    </div>

    <el-table :data="websites" style="width: 100%" v-loading="loading">
      <el-table-column :label="$t('website.title')" min-width="220">
        <template #default="{ row }">
          <div class="domain-cell">
            <el-link type="primary" @click="goConfig(row.id)">{{ row.primaryDomain }}</el-link>
            <el-popover v-if="additionalDomainCount(row)" placement="bottom-start" :width="280" trigger="click">
              <template #reference>
                <button type="button" class="domain-extra">+{{ additionalDomainCount(row) }}</button>
              </template>
              <div>{{ additionalDomains(row).join(', ') }}</div>
            </el-popover>
          </div>
          <div v-if="row.remark" class="remark-text">{{ row.remark }}</div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.type')" width="150">
        <template #default="{ row }">
          <div class="type-cell">
            <el-tag :type="row.type === 'static' ? 'info' : 'warning'" size="small">
              {{ row.type === 'static' ? $t('website.typeStatic') : $t('website.typeProxy') }}
            </el-tag>
            <el-tag size="small" effect="plain">{{ modeLabel(row) }}</el-tag>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.status')" width="110">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'danger'" size="small">
            {{ row.status === 'running' ? $t('website.running') : $t('website.stopped') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.https')" min-width="140">
        <template #default="{ row }">
          <div>{{ row.sslEnable ? $t('website.httpsOn') : $t('website.httpsOff') }}</div>
          <div v-if="row.configuredCertificate && Number.isFinite(row.configuredCertificate.daysLeft)" class="muted-text">
            {{ certificateStatusLabel(row.configuredCertificate.status) }} · {{ row.configuredCertificate.daysLeft }}天
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.target')" min-width="180">
        <template #default="{ row }">
          <code class="target-path" :title="row.proxyPass || row.siteDir || row.nginxConfPath">{{ row.proxyPass || row.siteDir || row.nginxConfPath || '—' }}</code>
        </template>
      </el-table-column>
      <el-table-column :label="$t('commons.actions')" width="160" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="goConfig(row.id)">{{ $t('commons.edit') }}</el-button>
          <el-dropdown trigger="click" @command="(cmd: string) => handleMore(cmd, row)">
            <el-button link>{{ $t('website.more') }}</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="row.status === 'stopped' && row.configMode !== 'source'" command="enable">{{ $t('website.enable') }}</el-dropdown-item>
                <el-dropdown-item v-else-if="row.configMode !== 'source'" command="disable">{{ $t('website.disable') }}</el-dropdown-item>
                <el-dropdown-item command="delete" divided>{{ $t('commons.delete') }}</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty :description="hasFilters ? $t('website.noFilterResult') : $t('website.emptyHint')">
          <el-button v-if="hasFilters" @click="clearFilters">{{ $t('website.clearFilters') }}</el-button>
        </el-empty>
      </template>
    </el-table>

    <el-pagination v-if="total > 0" class="mt-pagination" :current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" @current-change="(p: number) => { page = p }" />

    <!-- 创建网站对话框 -->
    <el-dialog v-model="createDialogVisible" :title="$t('website.create')" width="540px" destroy-on-close>
      <el-form :model="createForm" label-width="100px">
        <el-form-item v-if="createForm.configMode !== 'external'" :label="$t('website.type')">
          <el-radio-group v-model="createForm.type">
            <el-radio value="static">{{ $t('website.typeStatic') }}</el-radio>
            <el-radio value="reverse_proxy">{{ $t('website.typeProxy') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('website.configMode')">
          <el-radio-group v-model="createForm.configMode">
            <el-radio value="managed">{{ $t('website.managedMode') }}</el-radio>
            <el-radio value="source">{{ $t('website.sourceMode') }}</el-radio>
            <el-radio value="external">{{ $t('website.externalMode') }}</el-radio>
          </el-radio-group>
          <div class="form-tip">{{ createForm.configMode === 'external' ? $t('website.externalModeHint') : $t('website.createSourceModeHint') }}</div>
        </el-form-item>
        <template v-if="createForm.configMode === 'external'">
          <el-form-item :label="$t('website.configPath')">
            <div class="inspect-row">
              <el-input v-model="createForm.path" placeholder="/data/site/example/conf/site.conf" @input="externalPreview = null" />
              <el-button :loading="inspectLoading" @click="handleInspectExternal">{{ $t('website.inspectConfig') }}</el-button>
            </div>
          </el-form-item>
          <el-form-item :label="$t('website.alias')">
            <el-input v-model="createForm.alias" :placeholder="externalPreview?.primaryDomain?.replace(/[.*]/g, '_') || 'example_com'" />
          </el-form-item>
          <el-form-item :label="$t('commons.description')">
            <el-input v-model="createForm.remark" />
          </el-form-item>
          <el-card v-if="externalPreview" shadow="never" class="external-preview">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item :label="$t('website.domain')">{{ externalPreview.primaryDomain }}</el-descriptions-item>
              <el-descriptions-item :label="$t('website.otherDomains')">{{ externalPreview.domains.join(', ') }}</el-descriptions-item>
              <el-descriptions-item label="HTTP / HTTPS">{{ externalPreview.httpPort || '—' }} / {{ externalPreview.httpsPort || '—' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('website.type')">{{ externalPreview.type }}</el-descriptions-item>
              <el-descriptions-item :label="$t('website.siteDir')">{{ externalPreview.root || '—' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('website.proxyPass')">{{ externalPreview.proxyPass || '—' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('website.accessLogPath')">{{ externalPreview.accessLogPath || '—' }}</el-descriptions-item>
              <el-descriptions-item :label="$t('website.errorLogPath')">{{ externalPreview.errorLogPath || '—' }}</el-descriptions-item>
              <el-descriptions-item label="证书">{{ externalPreview.certPath || '—' }}</el-descriptions-item>
              <el-descriptions-item label="私钥">{{ externalPreview.keyPath || '—' }}</el-descriptions-item>
            </el-descriptions>
            <el-alert v-for="warning in externalPreview.warnings" :key="warning" :title="warning" type="warning" :closable="false" show-icon />
          </el-card>
        </template>
        <template v-else>
        <el-form-item :label="$t('website.domain')">
          <el-input v-model="createForm.primaryDomain" placeholder="example.com" />
        </el-form-item>
        <el-form-item :label="$t('website.alias')">
          <el-input v-model="createForm.alias" :placeholder="createForm.primaryDomain ? createForm.primaryDomain.replace(/\./g, '_') : 'example_com'" />
          <div class="form-tip">{{ $t('website.aliasHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('website.otherDomains')">
          <el-input v-model="createForm.domains" placeholder="www.example.com" />
          <div class="form-tip">{{ $t('website.otherDomainsHint') }}</div>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'static'" :label="$t('website.siteDir')">
          <div style="display:flex; gap:8px; width:100%;">
            <el-input v-model="createForm.siteDir" :placeholder="`/var/www/${effectiveAlias || 'example_com'}`" style="flex:1" />
            <el-button :icon="FolderOpened" @click="openDirBrowser('create')" />
          </div>
          <div class="form-tip">{{ $t('website.siteDirHint') }}，留空自动生成 /var/www/{{ effectiveAlias || 'alias名称' }}</div>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'reverse_proxy'" :label="$t('website.proxyPass')">
          <el-input v-model="createForm.proxyPass" placeholder="http://127.0.0.1:8080" />
          <div class="form-tip">{{ $t('website.proxyPassHint') }}</div>
        </el-form-item>
        <template v-if="createForm.configMode === 'source'">
          <el-form-item :label="$t('website.accessLogPath')">
            <el-input v-model="createForm.accessLogPath" placeholder="/var/log/nginx/example.com.access.log" />
          </el-form-item>
          <el-form-item :label="$t('website.errorLogPath')">
            <el-input v-model="createForm.errorLogPath" placeholder="/var/log/nginx/example.com.error.log" />
          </el-form-item>
        </template>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="createForm.remark" />
        </el-form-item>
        <el-form-item label="HTTP 端口">
          <el-input-number
            v-model="createForm.httpPort"
            :min="0"
            :max="65535"
            style="width: 160px"
            controls-position="right"
            placeholder="0"
          />
          <div class="form-tip">留空或 0 使用默认端口 80</div>
        </el-form-item>
        <el-form-item label="HTTPS 端口">
          <el-input-number
            v-model="createForm.httpsPort"
            :min="0"
            :max="65535"
            style="width: 160px"
            controls-position="right"
            placeholder="0"
          />
          <div class="form-tip">0 = 使用默认端口 443，仅启用 SSL 后生效</div>
        </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate" :loading="createLoading">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="certificateDialogVisible" :title="$t('website.certificateCheckResults')" width="860px">
      <el-table :data="certificateBatchResults" max-height="520">
        <el-table-column prop="websiteId" label="ID" width="70" />
        <el-table-column :label="$t('website.configuredCertificate')" min-width="180">
          <template #default="{ row }">
            <el-tag :type="certificateTagType(row.configured.status)" size="small">{{ certificateStatusLabel(row.configured.status) }}</el-tag>
            <span class="result-detail">{{ row.configured.daysLeft }} 天</span>
          </template>
        </el-table-column>
        <el-table-column label="本机 Nginx" min-width="180">
          <template #default="{ row }">
            <el-tag :type="certificateTagType(row.local.status)" size="small">{{ certificateStatusLabel(row.local.status) }}</el-tag>
            <span class="result-detail">:{{ row.httpsPort }}</span>
          </template>
        </el-table-column>
        <el-table-column label="公网端点" min-width="260">
          <template #default="{ row }">
            <div v-for="endpoint in row.public" :key="endpoint.address" class="endpoint-result">
              <code>{{ endpoint.domain }}</code>
              <el-tag :type="certificateTagType(endpoint.status)" size="small" effect="plain">{{ certificateStatusLabel(endpoint.status) }}</el-tag>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 目录浏览器 Dialog -->
    <el-dialog v-model="dirBrowserVisible" title="选择目录" width="520px" destroy-on-close>
      <div class="dir-browser">
        <div class="dir-browser-bar">
          <el-input v-model="dirBrowserPath" size="small" style="flex:1" placeholder="输入路径" @keyup.enter="loadDirList" />
          <el-button size="small" :icon="RefreshRight" title="刷新" @click="loadDirList" />
          <el-button size="small" :icon="ArrowUp" title="上级目录" @click="goParentDir" />
        </div>
        <div class="dir-browser-list" v-loading="dirLoading">
          <div
            v-for="item in dirList"
            :key="item.path"
            class="dir-item"
            :class="{ 'dir-item--selected': dirBrowserPath === item.path }"
            @click="selectDir(item.path)"
            @dblclick="enterDir(item.path)"
          >
            <el-icon class="dir-folder-icon"><Folder /></el-icon>
            <span>{{ item.name }}</span>
          </div>
          <div v-if="dirList.length === 0 && !dirLoading" class="dir-empty">无子目录（双击目录进入，单击选中）</div>
        </div>
        <div class="dir-browser-current">当前选择：<code>{{ dirBrowserPath }}</code></div>
      </div>
      <template #footer>
        <el-button @click="dirBrowserVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmDirSelect">选择此目录</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { FolderOpened, RefreshRight, ArrowUp, Folder } from '@element-plus/icons-vue'
import {
  searchWebsite, createWebsite, deleteWebsite, enableWebsite, disableWebsite,
  inspectExternalNginxSite, createExternalNginxSite, checkWebsiteCertificateHealthBatch,
} from '@/api/modules/website'
import { listFiles } from '@/api/modules/file'
import type { ExternalNginxSitePreview, Website, WebsiteCertificateHealth } from '@/api/interface'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const hasFilters = computed(() => Boolean(searchInfo.value || filterType.value || filterStatus.value))

const loading = ref(false)
const websites = ref<Website[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchInfo = ref('')
const filterType = ref('')
const filterStatus = ref('')

const createDialogVisible = ref(false)
const createLoading = ref(false)
const inspectLoading = ref(false)
const externalPreview = ref<ExternalNginxSitePreview | null>(null)
const certificateBatchLoading = ref(false)
const certificateDialogVisible = ref(false)
const certificateBatchResults = ref<WebsiteCertificateHealth[]>([])
const createForm = ref({
  primaryDomain: '',
  alias: '',
  domains: '',
  type: 'static',
  configMode: 'managed',
  remark: '',
  siteDir: '',
  proxyPass: '',
  accessLogPath: '',
  errorLogPath: '',
  httpPort: 0,
  httpsPort: 0,
  path: '',
})

// alias 有效值：优先用填写的，其次由域名推导
const effectiveAlias = computed(() => {
  if (createForm.value.alias.trim()) return createForm.value.alias.trim()
  if (createForm.value.primaryDomain) return createForm.value.primaryDomain.replace(/[.*]/g, '_')
  return ''
})

// --- 目录浏览器 ---
const dirBrowserVisible = ref(false)
const dirBrowserPath = ref('/var/www')
const dirList = ref<{ name: string; path: string }[]>([])
const dirLoading = ref(false)
let dirBrowserTarget = '' // 'create' | 'config'

const loadDirList = async () => {
  dirLoading.value = true
  try {
    const res = await listFiles({ path: dirBrowserPath.value, showHidden: false })
    const items = res.data?.items || []
    dirList.value = items
      .filter((f: any) => f.isDir)
      .map((f: any) => ({ name: f.name, path: f.path }))
  } catch {
    dirList.value = []
  } finally {
    dirLoading.value = false
  }
}

const openDirBrowser = (target: string) => {
  dirBrowserTarget = target
  // 用当前输入的路径初始化，若为空则默认 /var/www
  const cur = target === 'create' ? createForm.value.siteDir : ''
  dirBrowserPath.value = cur || '/var/www'
  dirBrowserVisible.value = true
  loadDirList()
}

const enterDir = (path: string) => {
  dirBrowserPath.value = path
  loadDirList()
}

const goParentDir = () => {
  const parts = dirBrowserPath.value.split('/').filter(Boolean)
  parts.pop()
  dirBrowserPath.value = '/' + parts.join('/')
  loadDirList()
}

const selectDir = (path: string) => {
  dirBrowserPath.value = path
}

const confirmDirSelect = () => {
  if (dirBrowserTarget === 'create') {
    createForm.value.siteDir = dirBrowserPath.value
  }
  dirBrowserVisible.value = false
}
// --- /目录浏览器 ---

const loadWebsites = async () => {
  loading.value = true
  try {
    const res = await searchWebsite({
      page: page.value,
      pageSize: pageSize.value,
      info: searchInfo.value,
      type: filterType.value,
      status: filterStatus.value,
    })
    websites.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch { websites.value = [] }
  finally { loading.value = false }
}

const openCreateDialog = () => {
  createForm.value = { primaryDomain: '', alias: '', domains: '', type: 'static', configMode: 'managed', remark: '', siteDir: '', proxyPass: '', accessLogPath: '', errorLogPath: '', httpPort: 0, httpsPort: 0, path: '' }
  externalPreview.value = null
  createDialogVisible.value = true
}

const openImportDialog = () => {
  openCreateDialog()
  createForm.value.configMode = 'external'
}

const onFilterChange = () => {
  page.value = 1
  persistQuery()
  loadWebsites()
}

const clearFilters = () => {
  searchInfo.value = ''
  filterType.value = ''
  filterStatus.value = ''
  onFilterChange()
}

const persistQuery = () => {
  const query: Record<string, string> = {}
  if (searchInfo.value) query.q = searchInfo.value
  if (filterType.value) query.type = filterType.value
  if (filterStatus.value) query.status = filterStatus.value
  if (page.value > 1) query.page = String(page.value)
  router.replace({ query })
}

const readQuery = () => {
  if (typeof route.query.q === 'string') searchInfo.value = route.query.q
  if (typeof route.query.type === 'string') filterType.value = route.query.type
  if (typeof route.query.status === 'string') filterStatus.value = route.query.status
  if (typeof route.query.page === 'string') {
    const next = Number(route.query.page)
    if (Number.isFinite(next) && next > 0) page.value = next
  }
}

const handleInspectExternal = async () => {
  if (!createForm.value.path.trim()) { ElMessage.warning(t('website.configPathRequired')); return }
  inspectLoading.value = true
  try {
    const res = await inspectExternalNginxSite(createForm.value.path.trim())
    externalPreview.value = res.data
    createForm.value.path = res.data.path
    createForm.value.primaryDomain = res.data.primaryDomain
    createForm.value.domains = res.data.domains.join(',')
    createForm.value.type = res.data.type
    createForm.value.siteDir = res.data.root
    createForm.value.proxyPass = res.data.proxyPass
    createForm.value.accessLogPath = res.data.accessLogPath
    createForm.value.errorLogPath = res.data.errorLogPath
    createForm.value.httpPort = res.data.httpPort
    createForm.value.httpsPort = res.data.httpsPort
  } finally { inspectLoading.value = false }
}

const handleCreate = async () => {
  if (createForm.value.configMode === 'external') {
    if (!externalPreview.value || externalPreview.value.path !== createForm.value.path) {
      ElMessage.warning(t('website.inspectBeforeCreate'))
      return
    }
    createLoading.value = true
    try {
      await createExternalNginxSite({ path: createForm.value.path, alias: createForm.value.alias, remark: createForm.value.remark })
      ElMessage.success(t('commons.success'))
      createDialogVisible.value = false
      loadWebsites()
    } finally { createLoading.value = false }
    return
  }
  if (!createForm.value.primaryDomain) { ElMessage.warning('请输入域名'); return }
  if (createForm.value.configMode !== 'source' && createForm.value.type === 'reverse_proxy' && !createForm.value.proxyPass) { ElMessage.warning('请输入代理地址'); return }
  if (createForm.value.configMode === 'source' && !createForm.value.siteDir) { ElMessage.warning('请输入网站目录'); return }
  createLoading.value = true
  try {
    await createWebsite(createForm.value)
    ElMessage.success(t('commons.success'))
    createDialogVisible.value = false
    loadWebsites()
  } catch {}
  finally { createLoading.value = false }
}

const handleEnable = async (row: Website) => {
  try {
    await ElMessageBox.confirm(t('website.enableConfirm'), t('commons.tip'), { type: 'info' })
    await enableWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const handleDisable = async (row: Website) => {
  try {
    await ElMessageBox.confirm(t('website.disableConfirm'), t('commons.tip'), { type: 'warning' })
    await disableWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const handleDelete = async (row: Website) => {
  try {
    const message = row.nginxConfPath ? t('website.unregisterExternalConfirm') : t('website.deleteConfirm')
    await ElMessageBox.confirm(message, t('commons.tip'), { type: row.nginxConfPath ? 'warning' : 'error' })
    await deleteWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const additionalDomains = (row: Website) => {
  return row.domains.split(',').map(item => item.trim()).filter(item => item && item !== row.primaryDomain)
}

const additionalDomainCount = (row: Website) => additionalDomains(row).length

const modeLabel = (row: Website) => {
  if (row.nginxConfPath || row.configMode === 'external') return t('website.modeExternal')
  if (row.configMode === 'source') return t('website.modeSource')
  return t('website.modeManaged')
}

const handleMore = (cmd: string, row: Website) => {
  if (cmd === 'enable') handleEnable(row)
  if (cmd === 'disable') handleDisable(row)
  if (cmd === 'delete') handleDelete(row)
}

const certificateTagType = (status: string): 'success' | 'warning' | 'info' | 'danger' => {
  if (status === 'valid') return 'success'
  if (status === 'expiring' || status === 'unavailable' || status === 'not_checked') return 'warning'
  if (status === 'not_configured') return 'info'
  return 'danger'
}

const certificateStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    valid: '有效', expiring: '即将过期', expired: '已过期', not_yet_valid: '尚未生效',
    domain_mismatch: '域名不匹配', key_mismatch: '私钥不匹配', unreadable: '无法读取',
    unavailable: '不可用', untrusted: '证书链不受信任', not_checked: '未检测', not_configured: '未配置',
  }
  return labels[status] || status
}

const handleCertificateBatch = async () => {
  certificateBatchLoading.value = true
  try {
    const res = await checkWebsiteCertificateHealthBatch({ all: true })
    certificateBatchResults.value = res.data || []
    certificateDialogVisible.value = true
  } finally { certificateBatchLoading.value = false }
}

const goConfig = (id: number) => {
  router.push(`/website/websites/${id}`)
}

watch(page, () => {
  persistQuery()
  loadWebsites()
})

onMounted(() => {
  readQuery()
  loadWebsites()
})
</script>

<style lang="scss" scoped>
.website-page {
  height: 100%;
}

.header-actions,
.inspect-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.inspect-row {
  width: 100%;
}

.filter-bar {
  .header-actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    margin-left: auto;
    gap: 8px;
  }

  .search-input {
    flex: 1 1 260px;
    min-width: 240px;
    max-width: 320px;
  }

  .filter-select {
    flex: 0 0 160px;
    width: 160px;
  }

  .filter-count {
    color: var(--xp-text-muted);
    font-size: 12px;
  }
}

.mt-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.domain-cell {
  display: flex;
  align-items: center;
  gap: 6px;

  .ssl-badge {
    font-size: 10px;
    padding: 0 4px;
    height: 18px;
    line-height: 18px;
  }

  .domain-extra {
    font-size: 11px;
    padding: 1px 5px;
    color: var(--xp-accent);
    background: var(--xp-accent-muted);
    border: 0;
    border-radius: 3px;
    cursor: pointer;
  }
}

.remark-text,
.target-path {
  margin-top: 4px;
  color: var(--xp-text-muted);
  font-size: 12px;
}

.target-path {
  display: block;
  overflow: hidden;
  font-family: var(--xp-font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-cell {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.filter-count {
  color: var(--xp-text-muted);
  font-size: 12px;
}

.config-path {
  display: block;
  max-width: 420px;
  margin-top: 5px;
  overflow: hidden;
  color: var(--xp-text-muted);
  font-family: var(--xp-font-mono);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.muted-text,
.result-detail {
  margin-left: 6px;
  color: var(--xp-text-muted);
  font-size: 12px;
}

.external-preview {
  margin: 0 0 16px 100px;

  :deep(.el-alert) {
    margin-top: 8px;
  }
}

.endpoint-result {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 28px;
}

.form-tip {
  font-size: 12px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}

.dir-browser {
  display: flex;
  flex-direction: column;
  gap: 10px;

  .dir-browser-bar {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .dir-browser-list {
    height: 260px;
    overflow-y: auto;
    border: 1px solid var(--xp-border);
    border-radius: var(--xp-radius-sm);
    padding: 4px;

    .dir-folder-icon { color: var(--xp-warning); }

    .dir-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 10px;
      border-radius: var(--xp-radius-sm);
      cursor: pointer;
      font-size: 13px;
      user-select: none;
      transition: background 0.15s;

      &:hover { background: var(--xp-bg-inset); }
      &--selected { background: var(--xp-accent-muted); color: var(--xp-accent); }
    }

    .dir-empty {
      text-align: center;
      color: var(--xp-text-muted);
      font-size: 13px;
      padding: 40px 0;
    }
  }

  .dir-browser-current {
    font-size: 12px;
    color: var(--xp-text-muted);
    code { font-size: 12px; color: var(--xp-accent); }
  }
}
</style>
