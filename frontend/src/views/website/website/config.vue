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
        <el-button v-if="detail.status === 'stopped'" type="success" size="small" @click="handleEnable">{{ $t('website.enable') }}</el-button>
        <el-button v-else type="warning" size="small" @click="handleDisable">{{ $t('website.disable') }}</el-button>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="config-tabs">
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

      <!-- 配置预览 -->
      <el-tab-pane :label="$t('website.configPreview')" name="preview">
        <div class="config-preview">
          <el-button size="small" style="margin-bottom: 8px" @click="loadDetail">{{ $t('commons.refresh') }}</el-button>
          <pre class="preview-content">{{ detail.nginxConfig || '暂无配置' }}</pre>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Delete, Plus } from '@element-plus/icons-vue'
import { getWebsiteDetail, updateWebsite, enableWebsite, disableWebsite, getWebsiteLog } from '@/api/modules/website'
import { searchCertificate } from '@/api/modules/ssl'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const saving = ref(false)
const activeTab = ref('basic')
const detail = ref<any>({})
const certList = ref<any[]>([])
const redirects = ref<any[]>([])

// 日志
const logType = ref('access')
const logContent = ref<string | null>(null)

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
    // 解析 redirects JSON
    try {
      redirects.value = detail.value.redirects ? JSON.parse(detail.value.redirects) : []
    } catch { redirects.value = [] }
  } catch { router.push('/website/websites') }
  finally { loading.value = false }
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

onMounted(() => {
  loadDetail()
  loadCerts()
})
</script>

<style lang="scss" scoped>
.website-config-page {
  height: 100%;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  .header-left {
    display: flex;
    align-items: center;
    gap: 10px;

    h3 {
      margin: 0;
      font-size: 16px;
      color: var(--xp-text-primary);
    }
  }
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
    background: var(--xp-bg-deep, #0d1117);
    border-radius: 6px;
    padding: 16px;
    max-height: 450px;
    overflow-y: auto;
  }

  .log-content {
    font-family: 'Courier New', Consolas, monospace;
    font-size: 12px;
    line-height: 1.7;
    color: #c9d1d9;
    white-space: pre-wrap;
    word-break: break-all;
    margin: 0;
  }
}

.config-preview {
  .preview-content {
    background: var(--xp-bg-deep, #0d1117);
    border-radius: 6px;
    padding: 16px;
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    font-size: 13px;
    line-height: 1.6;
    color: #c9d1d9;
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
</style>
