<template>
  <div class="ssl-page">
    <div class="ssl-header">
      <h3>{{ $t('ssl.title') }}</h3>
      <div class="header-actions">
        <el-button size="small" type="info" plain @click="sslDirVisible = true">
          <el-icon><FolderOpened /></el-icon>
          {{ $t('ssl.certDir') }}
        </el-button>
        <el-dropdown @command="handleExportImport" trigger="click">
          <el-button size="small" type="info" plain>
            <el-icon><Download /></el-icon>
            {{ $t('ssl.importExport') }}
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="export">{{ $t('ssl.exportAccounts') }}</el-dropdown-item>
              <el-dropdown-item command="import">{{ $t('ssl.importAccounts') }}</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="ssl-tabs">
      <!-- 证书列表 -->
      <el-tab-pane :label="$t('ssl.certificates')" name="certs">
        <div class="tab-toolbar">
          <el-input v-model="certSearch" :placeholder="$t('commons.search')" prefix-icon="Search" size="small" clearable class="search-input" @input="loadCerts" />
          <el-button size="small" type="primary" @click="openCertDialog()">
            <el-icon><Plus /></el-icon>
            {{ $t('ssl.applyCert') }}
          </el-button>
          <el-button size="small" type="success" plain @click="uploadVisible = true">
            <el-icon><Upload /></el-icon>
            {{ $t('ssl.uploadCert') }}
          </el-button>
        </div>
        <el-table :data="certs" style="width: 100%">
          <el-table-column prop="primaryDomain" :label="$t('ssl.domain')" min-width="180">
            <template #default="{ row }">
              <div class="domain-cell">
                <span class="primary-domain">{{ row.primaryDomain }}</span>
                <span v-if="row.domains" class="other-domains">+{{ row.domains.split(',').length }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="$t('ssl.status')" width="110">
            <template #default="{ row }">
              <el-popover v-if="row.status === 'error' && row.message" placement="bottom" :width="360" trigger="hover">
                <template #reference>
                  <el-tag type="danger" size="small" style="cursor:pointer">错误</el-tag>
                </template>
                <div class="error-popover">{{ row.message }}</div>
              </el-popover>
              <el-tag v-else :type="statusType(row.status)" size="small">
                <el-icon v-if="row.status === 'applying'" class="is-loading"><Loading /></el-icon>
                {{ statusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('ssl.provider')" width="100">
            <template #default="{ row }">
              {{ row.type === 'upload' ? '手动上传' : row.provider?.toUpperCase() }}
            </template>
          </el-table-column>
          <el-table-column :label="$t('ssl.expireDate')" width="170">
            <template #default="{ row }">
              <span :class="{ 'text-danger': isExpiring(row.expireDate) }">
                {{ row.expireDate ? formatDate(row.expireDate) : '-' }}
              </span>
            </template>
          </el-table-column>
          <el-table-column prop="autoRenew" :label="$t('ssl.autoRenew')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.autoRenew ? 'success' : 'info'" size="small">
                {{ row.autoRenew ? '是' : '否' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="日志" width="70">
            <template #default="{ row }">
              <el-button v-if="row.type !== 'upload'" link type="primary" size="small" @click="handleViewLog(row.id)">
                查看
              </el-button>
            </template>
          </el-table-column>
          <el-table-column :label="$t('commons.actions')" width="220" fixed="right">
            <template #default="{ row }">
              <el-button v-if="row.status !== 'applied'" link type="primary" size="small" @click="handleApply(row.id)">
                {{ $t('ssl.apply') }}
              </el-button>
              <el-button v-if="row.status === 'applied' && row.type !== 'upload'" link type="success" size="small" @click="handleRenew(row.id)">
                {{ $t('ssl.renew') }}
              </el-button>
              <el-button link type="info" size="small" @click="handleDetail(row.id)">
                {{ $t('ssl.detail') }}
              </el-button>
              <el-button link type="danger" size="small" @click="handleDeleteCert(row.id)">
                {{ $t('commons.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination v-if="certTotal > 0" class="mt-pagination" :current-page="certPage" :page-size="certPageSize" :total="certTotal" layout="total, prev, pager, next" @current-change="(p: number) => { certPage = p; loadCerts() }" />
      </el-tab-pane>

      <!-- ACME 账户 -->
      <el-tab-pane :label="$t('ssl.acmeAccounts')" name="acme">
        <div class="tab-toolbar">
          <div></div>
          <el-button size="small" type="primary" @click="acmeDialogVisible = true">
            <el-icon><Plus /></el-icon>
            {{ $t('ssl.addAcme') }}
          </el-button>
        </div>
        <el-table :data="acmeAccounts" style="width: 100%">
          <el-table-column prop="email" label="Email" min-width="200" />
          <el-table-column prop="type" :label="$t('ssl.caType')" width="140">
            <template #default="{ row }">
              <el-tag size="small">{{ caLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="keyType" :label="$t('ssl.keyType')" width="100" />
          <el-table-column prop="url" label="URL" min-width="200" show-overflow-tooltip />
          <el-table-column :label="$t('commons.actions')" width="100" fixed="right">
            <template #default="{ row }">
              <el-button link type="danger" size="small" @click="handleDeleteAcme(row.id)">
                {{ $t('commons.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- DNS 账户 -->
      <el-tab-pane :label="$t('ssl.dnsAccounts')" name="dns">
        <div class="tab-toolbar">
          <div></div>
          <el-button size="small" type="primary" @click="openDnsDialog()">
            <el-icon><Plus /></el-icon>
            {{ $t('ssl.addDns') }}
          </el-button>
        </div>
        <el-table :data="dnsAccounts" style="width: 100%">
          <el-table-column prop="name" :label="$t('commons.name')" min-width="150" />
          <el-table-column prop="type" :label="$t('ssl.dnsProvider')" width="150">
            <template #default="{ row }">
              <el-tag size="small">{{ row.type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('commons.actions')" width="150" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="openDnsDialog(row)">
                {{ $t('commons.edit') }}
              </el-button>
              <el-button link type="danger" size="small" @click="handleDeleteDns(row.id)">
                {{ $t('commons.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- 申请证书对话框 -->
    <el-dialog v-model="certDialogVisible" :title="$t('ssl.applyCert')" width="560px" destroy-on-close>
      <el-form :model="certForm" label-width="110px">
        <el-form-item :label="$t('ssl.primaryDomain')">
          <el-input v-model="certForm.primaryDomain" placeholder="example.com" />
        </el-form-item>
        <el-form-item :label="$t('ssl.otherDomains')">
          <el-input v-model="certForm.otherDomains" placeholder="www.example.com,api.example.com" />
          <div class="form-tip">多个域名用逗号分隔</div>
        </el-form-item>
        <el-form-item :label="$t('ssl.acmeAccount')">
          <el-select v-model="certForm.acmeAccountID" style="width: 100%">
            <el-option v-for="a in acmeAccounts" :key="a.id" :label="`${a.email} (${caLabel(a.type)})`" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('ssl.verifyMethod')">
          <el-radio-group v-model="certForm.provider">
            <el-radio value="dns">DNS 验证</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="certForm.provider === 'dns'" :label="$t('ssl.dnsAccount')">
          <el-select v-model="certForm.dnsAccountID" style="width: 100%">
            <el-option v-for="d in dnsAccounts" :key="d.id" :label="`${d.name} (${d.type})`" :value="d.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('ssl.keyType')">
          <el-select v-model="certForm.keyType" style="width: 100%">
            <el-option label="RSA 2048" value="2048" />
            <el-option label="RSA 3072" value="3072" />
            <el-option label="RSA 4096" value="4096" />
            <el-option label="EC P256" value="P256" />
            <el-option label="EC P384" value="P384" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('ssl.autoRenew')">
          <el-switch v-model="certForm.autoRenew" />
        </el-form-item>
        <el-form-item :label="$t('ssl.applyNow')">
          <el-switch v-model="certForm.apply" />
          <div class="form-tip">开启后将立即开始申请证书</div>
        </el-form-item>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="certForm.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="certDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateCert" :loading="certSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 上传证书对话框 -->
    <el-dialog v-model="uploadVisible" :title="$t('ssl.uploadCert')" width="560px" destroy-on-close>
      <el-form :model="uploadForm" label-width="90px">
        <el-form-item :label="$t('ssl.certificate')">
          <el-input v-model="uploadForm.certificate" type="textarea" :rows="6" placeholder="-----BEGIN CERTIFICATE-----" />
        </el-form-item>
        <el-form-item :label="$t('ssl.privateKey')">
          <el-input v-model="uploadForm.privateKey" type="textarea" :rows="6" placeholder="-----BEGIN RSA PRIVATE KEY-----" />
        </el-form-item>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="uploadForm.description" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="uploadVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleUploadCert" :loading="uploadSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- ACME 账户对话框 -->
    <el-dialog v-model="acmeDialogVisible" :title="$t('ssl.addAcme')" width="480px" destroy-on-close>
      <el-form :model="acmeForm" label-width="100px">
        <el-form-item label="Email">
          <el-input v-model="acmeForm.email" placeholder="admin@example.com" />
        </el-form-item>
        <el-form-item :label="$t('ssl.caType')">
          <el-select v-model="acmeForm.type" style="width: 100%">
            <el-option label="Let's Encrypt" value="letsencrypt" />
            <el-option label="ZeroSSL" value="zerossl" />
            <el-option label="Buypass" value="buypass" />
            <el-option label="Google Trust" value="google" />
            <el-option label="自定义 CA" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="acmeForm.type === 'custom'" label="CA URL">
          <el-input v-model="acmeForm.caDirURL" placeholder="https://acme.example.com/directory" />
        </el-form-item>
        <el-form-item :label="$t('ssl.keyType')">
          <el-select v-model="acmeForm.keyType" style="width: 100%">
            <el-option label="RSA 2048" value="2048" />
            <el-option label="RSA 4096" value="4096" />
            <el-option label="EC P256" value="P256" />
            <el-option label="EC P384" value="P384" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="acmeDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateAcme" :loading="acmeSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- DNS 账户对话框 -->
    <el-dialog v-model="dnsDialogVisible" :title="dnsEditMode ? $t('ssl.editDns') : $t('ssl.addDns')" width="480px" destroy-on-close>
      <el-form :model="dnsForm" label-width="90px">
        <el-form-item :label="$t('commons.name')">
          <el-input v-model="dnsForm.name" placeholder="我的阿里云" />
        </el-form-item>
        <el-form-item :label="$t('ssl.dnsProvider')">
          <el-select v-model="dnsForm.type" style="width: 100%" @change="handleDnsTypeChange">
            <el-option v-for="p in dnsProviders" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
        </el-form-item>
        <template v-if="currentDnsFields.length > 0">
          <el-form-item v-for="field in currentDnsFields" :key="field" :label="field">
            <el-input v-model="dnsForm.authorization[field]" :type="field.includes('Key') || field.includes('Secret') || field.includes('Token') || field.includes('token') ? 'password' : 'text'" show-password />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dnsDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmitDns" :loading="dnsSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- 证书详情对话框 -->
    <el-dialog v-model="detailVisible" :title="$t('ssl.detail')" width="600px" destroy-on-close>
      <template v-if="certDetail">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('ssl.primaryDomain')">{{ certDetail.primaryDomain }}</el-descriptions-item>
          <el-descriptions-item :label="$t('ssl.status')">
            <el-tag :type="statusType(certDetail.status)" size="small">{{ statusLabel(certDetail.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('ssl.expireDate')">{{ certDetail.expireDate ? formatDate(certDetail.expireDate) : '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('ssl.startDate')">{{ certDetail.startDate ? formatDate(certDetail.startDate) : '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('ssl.certDir')" :span="2">{{ certDetail.filePath || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-divider />
        <div class="cert-pem-section" v-if="certDetail.pem">
          <div class="pem-label">{{ $t('ssl.certificate') }} (fullchain.pem)</div>
          <el-input :model-value="certDetail.pem" type="textarea" :rows="6" readonly />
          <div class="pem-label mt-12">{{ $t('ssl.privateKey') }} (privkey.pem)</div>
          <el-input :model-value="certDetail.privateKey" type="textarea" :rows="4" readonly />
        </div>
      </template>
    </el-dialog>

    <!-- SSL 路径设置 -->
    <el-dialog v-model="sslDirVisible" :title="$t('ssl.certDir')" width="460px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item :label="$t('ssl.storagePath')">
          <el-input v-model="sslDir" placeholder="/opt/xpanel/ssl" />
          <div class="form-tip">证书将统一存放在此路径下，默认为安装目录/ssl</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="sslDirVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleUpdateSSLDir">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 日志查看对话框 -->
    <el-dialog v-model="logVisible" title="申请日志" width="680px" destroy-on-close>
      <div class="ssl-log-container">
        <div v-if="logLoading" class="log-loading">加载中...</div>
        <pre v-else class="ssl-log-content">{{ logContent || '暂无日志' }}</pre>
      </div>
      <template #footer>
        <el-button size="small" @click="handleRefreshLog">刷新</el-button>
        <el-button size="small" @click="logVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <!-- 导入文件 input（隐藏） -->
    <input ref="importInput" type="file" accept=".json" style="display: none" @change="handleImportFile" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import {
  searchCertificate, createCertificate, uploadCertificate, deleteCertificate,
  getCertificateDetail, applyCertificate, renewCertificate, getCertificateLog,
  listAcmeAccount, createAcmeAccount, deleteAcmeAccount,
  listDnsAccount, createDnsAccount, updateDnsAccount, deleteDnsAccount,
  exportAccounts, importAccounts,
  getSSLDir, updateSSLDir, getDnsProviders,
} from '@/api/modules/ssl'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('certs')

// --- 证书 ---
const certs = ref<any[]>([])
const certTotal = ref(0)
const certPage = ref(1)
const certPageSize = ref(20)
const certSearch = ref('')
const certDialogVisible = ref(false)
const certSubmitting = ref(false)
const uploadVisible = ref(false)
const uploadSubmitting = ref(false)
const detailVisible = ref(false)
const certDetail = ref<any>(null)

const defaultCertForm = () => ({
  primaryDomain: '',
  otherDomains: '',
  provider: 'dns',
  acmeAccountID: undefined as number | undefined,
  dnsAccountID: undefined as number | undefined,
  keyType: '2048',
  autoRenew: true,
  apply: true,
  description: '',
})
const certForm = ref(defaultCertForm())
const uploadForm = ref({ certificate: '', privateKey: '', description: '' })

// --- ACME 账户 ---
const acmeAccounts = ref<any[]>([])
const acmeDialogVisible = ref(false)
const acmeSubmitting = ref(false)
const acmeForm = ref({ email: '', type: 'letsencrypt', keyType: '2048', caDirURL: '' })

// --- DNS 账户 ---
const dnsAccounts = ref<any[]>([])
const dnsDialogVisible = ref(false)
const dnsSubmitting = ref(false)
const dnsEditMode = ref(false)
const dnsProviders = ref<any[]>([])
const dnsForm = ref({ id: 0, name: '', type: '', authorization: {} as Record<string, string> })

// --- SSL 路径 ---
const sslDirVisible = ref(false)
const sslDir = ref('')

// --- 导入 ---
const importInput = ref<HTMLInputElement | null>(null)

// --- 日志 ---
const logVisible = ref(false)
const logLoading = ref(false)
const logContent = ref('')
const logCertId = ref(0)
let logPollTimer: ReturnType<typeof setInterval> | null = null

// --- 申请中自动刷新 ---
let applyingPollTimer: ReturnType<typeof setInterval> | null = null

const currentDnsFields = computed(() => {
  const p = dnsProviders.value.find((x: any) => x.value === dnsForm.value.type)
  return p ? p.fields.split(',') : []
})

// --- 加载数据 ---
const loadCerts = async () => {
  try {
    const res = await searchCertificate({ page: certPage.value, pageSize: certPageSize.value, info: certSearch.value })
    certs.value = res.data?.items || []
    certTotal.value = res.data?.total || 0
  } catch { certs.value = [] }
}

const loadAcmeAccounts = async () => {
  try {
    const res = await listAcmeAccount()
    acmeAccounts.value = res.data || []
  } catch { acmeAccounts.value = [] }
}

const loadDnsAccounts = async () => {
  try {
    const res = await listDnsAccount()
    dnsAccounts.value = res.data || []
  } catch { dnsAccounts.value = [] }
}

const loadDnsProviders = async () => {
  try {
    const res = await getDnsProviders()
    dnsProviders.value = res.data || []
  } catch { dnsProviders.value = [] }
}

const loadSSLDir = async () => {
  try {
    const res = await getSSLDir()
    sslDir.value = res.data || ''
  } catch {}
}

// --- 证书操作 ---
const openCertDialog = () => {
  certForm.value = defaultCertForm()
  if (acmeAccounts.value.length > 0) certForm.value.acmeAccountID = acmeAccounts.value[0].id
  if (dnsAccounts.value.length > 0) certForm.value.dnsAccountID = dnsAccounts.value[0].id
  certDialogVisible.value = true
}

const handleCreateCert = async () => {
  if (!certForm.value.primaryDomain) { ElMessage.warning('请输入主域名'); return }
  if (!certForm.value.acmeAccountID) { ElMessage.warning('请选择 ACME 账户'); return }
  certSubmitting.value = true
  try {
    await createCertificate(certForm.value as any)
    ElMessage.success('创建成功')
    certDialogVisible.value = false
    loadCerts()
  } catch { ElMessage.error('创建失败') }
  finally { certSubmitting.value = false }
}

const handleUploadCert = async () => {
  if (!uploadForm.value.certificate || !uploadForm.value.privateKey) {
    ElMessage.warning('请填写证书和私钥'); return
  }
  uploadSubmitting.value = true
  try {
    await uploadCertificate(uploadForm.value)
    ElMessage.success('上传成功')
    uploadVisible.value = false
    loadCerts()
  } catch { ElMessage.error('上传失败') }
  finally { uploadSubmitting.value = false }
}

const handleApply = async (id: number) => {
  try {
    await applyCertificate(id)
    ElMessage.success('申请已提交，可点击"查看"跟踪日志')
    setTimeout(loadCerts, 2000)
    startApplyingPoll()
  } catch { ElMessage.error('申请失败') }
}

const handleRenew = async (id: number) => {
  await ElMessageBox.confirm('确定要续签该证书吗？', '提示')
  try {
    await renewCertificate(id)
    ElMessage.success('续签已提交，可点击"查看"跟踪日志')
    setTimeout(loadCerts, 2000)
    startApplyingPoll()
  } catch { ElMessage.error('续签失败') }
}

const handleDetail = async (id: number) => {
  try {
    const res = await getCertificateDetail(id)
    certDetail.value = res.data
    detailVisible.value = true
  } catch { ElMessage.error('获取详情失败') }
}

const handleDeleteCert = async (id: number) => {
  await ElMessageBox.confirm('确定要删除该证书吗？证书文件也会被清除。', '提示', { type: 'warning' })
  try {
    await deleteCertificate(id)
    ElMessage.success('删除成功')
    loadCerts()
  } catch { ElMessage.error('删除失败') }
}

// --- 日志查看 ---
const handleViewLog = async (id: number) => {
  logCertId.value = id
  logVisible.value = true
  logLoading.value = true
  try {
    const res = await getCertificateLog(id)
    logContent.value = res.data || '暂无日志'
  } catch { logContent.value = '获取日志失败' }
  finally { logLoading.value = false }

  // 如果证书正在申请中，自动轮询日志
  const cert = certs.value.find((c: any) => c.id === id)
  if (cert && cert.status === 'applying') {
    startLogPoll(id)
  }
}

const handleRefreshLog = async () => {
  if (!logCertId.value) return
  logLoading.value = true
  try {
    const res = await getCertificateLog(logCertId.value)
    logContent.value = res.data || '暂无日志'
  } catch { logContent.value = '获取日志失败' }
  finally { logLoading.value = false }
}

const startLogPoll = (id: number) => {
  stopLogPoll()
  logPollTimer = setInterval(async () => {
    try {
      const res = await getCertificateLog(id)
      logContent.value = res.data || '暂无日志'
    } catch { /* ignore */ }
    // 检查是否还在 applying
    const cert = certs.value.find((c: any) => c.id === id)
    if (!cert || cert.status !== 'applying') {
      stopLogPoll()
      loadCerts() // 刷新列表以更新状态
    }
  }, 3000)
}

const stopLogPoll = () => {
  if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null }
}

// 启动申请中状态自动检测
const startApplyingPoll = () => {
  stopApplyingPoll()
  applyingPollTimer = setInterval(() => {
    const hasApplying = certs.value.some((c: any) => c.status === 'applying')
    if (hasApplying) {
      loadCerts()
    } else {
      stopApplyingPoll()
    }
  }, 5000)
}

const stopApplyingPoll = () => {
  if (applyingPollTimer) { clearInterval(applyingPollTimer); applyingPollTimer = null }
}

// --- ACME 账户操作 ---
const handleCreateAcme = async () => {
  if (!acmeForm.value.email) { ElMessage.warning('请输入邮箱'); return }
  acmeSubmitting.value = true
  try {
    await createAcmeAccount(acmeForm.value)
    ElMessage.success('注册成功')
    acmeDialogVisible.value = false
    loadAcmeAccounts()
  } catch { ElMessage.error('注册失败，请检查网络') }
  finally { acmeSubmitting.value = false }
}

const handleDeleteAcme = async (id: number) => {
  await ElMessageBox.confirm('确定要删除该 ACME 账户吗？', '提示', { type: 'warning' })
  try {
    await deleteAcmeAccount(id)
    ElMessage.success('删除成功')
    loadAcmeAccounts()
  } catch { ElMessage.error('删除失败') }
}

// --- DNS 账户操作 ---
const openDnsDialog = (row?: any) => {
  if (row) {
    dnsEditMode.value = true
    dnsForm.value = { id: row.id, name: row.name, type: row.type, authorization: { ...(row.authorization || {}) } }
  } else {
    dnsEditMode.value = false
    dnsForm.value = { id: 0, name: '', type: dnsProviders.value[0]?.value || '', authorization: {} }
  }
  dnsDialogVisible.value = true
}

const handleDnsTypeChange = () => {
  dnsForm.value.authorization = {}
}

const handleSubmitDns = async () => {
  if (!dnsForm.value.name || !dnsForm.value.type) { ElMessage.warning('请填写完整信息'); return }
  dnsSubmitting.value = true
  try {
    if (dnsEditMode.value) {
      await updateDnsAccount(dnsForm.value as any)
    } else {
      await createDnsAccount(dnsForm.value as any)
    }
    ElMessage.success('操作成功')
    dnsDialogVisible.value = false
    loadDnsAccounts()
  } catch { ElMessage.error('操作失败') }
  finally { dnsSubmitting.value = false }
}

const handleDeleteDns = async (id: number) => {
  await ElMessageBox.confirm('确定要删除该 DNS 账户吗？', '提示', { type: 'warning' })
  try {
    await deleteDnsAccount(id)
    ElMessage.success('删除成功')
    loadDnsAccounts()
  } catch { ElMessage.error('删除失败') }
}

// --- SSL 路径 ---
const handleUpdateSSLDir = async () => {
  if (!sslDir.value) { ElMessage.warning('请输入路径'); return }
  try {
    await updateSSLDir(sslDir.value)
    ElMessage.success('设置成功')
    sslDirVisible.value = false
  } catch { ElMessage.error('设置失败') }
}

// --- 导入导出 ---
const handleExportImport = async (cmd: string) => {
  if (cmd === 'export') {
    try {
      const res = await exportAccounts()
      const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `xpanel-ssl-accounts-${new Date().toISOString().slice(0, 10)}.json`
      a.click()
      URL.revokeObjectURL(url)
      ElMessage.success('导出成功')
    } catch { ElMessage.error('导出失败') }
  } else {
    importInput.value?.click()
  }
}

const handleImportFile = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = async (ev) => {
    try {
      const data = JSON.parse(ev.target?.result as string)
      await importAccounts(data)
      ElMessage.success('导入成功')
      loadAcmeAccounts()
      loadDnsAccounts()
    } catch { ElMessage.error('导入失败，请检查文件格式') }
  }
  reader.readAsText(file)
  ;(e.target as HTMLInputElement).value = ''
}

// --- 工具函数 ---
const statusType = (s: string) => {
  const map: Record<string, string> = { applied: 'success', applying: 'warning', error: 'danger', ready: 'info' }
  return (map[s] || 'info') as any
}

const statusLabel = (s: string) => {
  const map: Record<string, string> = { applied: '已签发', applying: '申请中', error: '错误', ready: '待申请' }
  return map[s] || s
}

const caLabel = (t: string) => {
  const map: Record<string, string> = { letsencrypt: "Let's Encrypt", zerossl: 'ZeroSSL', buypass: 'Buypass', google: 'Google Trust', custom: '自定义' }
  return map[t] || t
}

const formatDate = (d: string) => {
  return new Date(d).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

const isExpiring = (d: string) => {
  if (!d) return false
  const diff = new Date(d).getTime() - Date.now()
  return diff < 30 * 24 * 60 * 60 * 1000 // 30 天内
}

onMounted(() => {
  loadCerts()
  loadAcmeAccounts()
  loadDnsAccounts()
  loadDnsProviders()
  loadSSLDir()
})

onUnmounted(() => {
  stopLogPoll()
  stopApplyingPoll()
})
</script>

<style lang="scss" scoped>
.ssl-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.ssl-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 {
    margin: 0;
    font-size: 16px;
    color: var(--xp-text-primary);
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }
}

.ssl-tabs {
  flex: 1;

  :deep(.el-tabs__header) {
    margin-bottom: 12px;
  }
}

.tab-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 8px;

  .search-input {
    width: 240px;
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

  .primary-domain {
    font-weight: 500;
  }

  .other-domains {
    font-size: 11px;
    padding: 1px 5px;
    border-radius: 3px;
    background: rgba(34, 211, 238, 0.12);
    color: var(--xp-accent);
  }
}

.text-danger {
  color: var(--xp-danger);
}

.form-tip {
  font-size: 12px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}

.error-popover {
  font-size: 13px;
  color: #ef4444;
  word-break: break-all;
  max-height: 200px;
  overflow-y: auto;
}

.ssl-log-container {
  background: var(--xp-bg-deep, #0d1117);
  border-radius: 6px;
  padding: 16px;
  max-height: 450px;
  overflow-y: auto;
}

.ssl-log-content {
  font-family: 'Courier New', Consolas, monospace;
  font-size: 12px;
  line-height: 1.7;
  color: #c9d1d9;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}

.log-loading {
  text-align: center;
  color: var(--xp-text-muted);
  padding: 40px 0;
}

.cert-pem-section {
  .pem-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--xp-text-secondary);
    margin-bottom: 6px;
  }

  .mt-12 {
    margin-top: 12px;
  }
}
</style>
