<template>
  <div class="nginx-page">
    <div class="page-header">
      <h3>{{ $t('nginx.title') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadStatus" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <!-- 未安装状态 -->
    <template v-if="!status.isInstalled && !installing">
      <el-card shadow="never" class="install-card">
        <el-empty :description="$t('nginx.notInstalled')">
          <template #image>
            <el-icon :size="64" color="var(--xp-text-muted)"><Box /></el-icon>
          </template>
          <div class="install-actions">
            <el-button type="primary" @click="handleShowInstall">
              {{ $t('nginx.install') }}
            </el-button>
          </div>
        </el-empty>
      </el-card>
    </template>

    <!-- 安装中 -->
    <template v-if="installing">
      <el-card shadow="never" class="progress-card">
        <template #header>
          <span>{{ $t('nginx.installProgress') }}</span>
        </template>
        <div class="progress-content">
          <el-progress :percentage="installProgress.percent" :status="progressStatus" :stroke-width="18" :text-inside="true" />
          <div class="progress-phase">
            <el-tag :type="phaseTagType" size="small">{{ phaseLabel }}</el-tag>
            <span class="progress-msg">{{ installProgress.message }}</span>
          </div>
        </div>
      </el-card>
    </template>

    <!-- 已安装 — 状态面板 -->
    <template v-if="status.isInstalled && !installing">
      <!-- 双安装警告 -->
      <el-alert v-if="status.hasBothInstalled" type="warning" show-icon :closable="false" style="margin-bottom: 16px">
        <template #title>{{ $t('nginx.bothInstalledTitle') }}</template>
        <div style="margin-bottom: 8px">
          {{ $t('nginx.bothInstalledDesc') }}
        </div>
        <div style="display: flex; gap: 8px;">
          <el-popconfirm
            :title="$t('nginx.uninstallConfirm', { mode: status.systemMode ? $t('nginx.prefixMode') : $t('nginx.systemMode') })"
            @confirm="handleUninstallInactive"
          >
            <template #reference>
              <el-button size="small" type="warning" :loading="uninstalling">
                {{ $t('nginx.uninstallInactive', { mode: status.systemMode ? $t('nginx.prefixMode') : $t('nginx.systemMode') }) }}
              </el-button>
            </template>
          </el-popconfirm>
        </div>
      </el-alert>

      <el-tabs v-model="mainTab" class="nginx-tabs">
        <el-tab-pane :label="$t('nginx.status')" name="status">

      <!-- 信息卡片 -->
      <el-row :gutter="16" class="info-row">
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nginx.status') }}</div>
            <div class="stat-value">
              <el-tag :type="status.isRunning ? 'success' : 'danger'" size="large" effect="dark" round>
                {{ status.isRunning ? $t('nginx.running') : $t('nginx.stopped') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">
              {{ $t('nginx.version') }}
              <el-button link type="primary" size="small" :loading="updateCheckLoading" @click="handleCheckUpdate" style="margin-left: 6px">
                {{ $t('nginx.checkUpdate') }}
              </el-button>
            </div>
            <div class="stat-value version-text">{{ status.version || '-' }}</div>
            <div v-if="updateInfo.hasUpdate" class="update-hint">
              <el-tag type="warning" size="small" effect="plain">
                {{ $t('nginx.newVersionAvailable', { version: updateInfo.availableVersion }) }}
              </el-tag>
              <el-button type="warning" size="small" @click="handleUpgrade" :loading="upgradeLoading" style="margin-left: 6px">
                <el-icon><Upload /></el-icon>{{ $t('nginx.upgrade') }}
              </el-button>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nginx.pid') }}</div>
            <div class="stat-value version-text">{{ status.isRunning ? status.pid : '-' }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nginx.configOK') }}</div>
            <div class="stat-value">
              <el-tag :type="status.configOK ? 'success' : 'danger'" size="large" effect="dark" round>
                {{ status.configOK ? $t('nginx.configValid') : $t('nginx.configInvalid') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 操作按钮 -->
      <el-card shadow="never" class="operate-card">
        <template #header>
          <span>{{ $t('commons.operate') }}</span>
        </template>
        <div class="operate-buttons">
          <el-button type="success" :disabled="status.isRunning" @click="handleOperate('start')" :loading="operateLoading === 'start'">
            <el-icon><VideoPlay /></el-icon>{{ $t('nginx.start') }}
          </el-button>
          <el-button type="danger" :disabled="!status.isRunning" @click="handleOperate('stop')" :loading="operateLoading === 'stop'">
            <el-icon><VideoPause /></el-icon>{{ $t('nginx.stop') }}
          </el-button>
          <el-button type="primary" :disabled="!status.isRunning" @click="handleOperate('reload')" :loading="operateLoading === 'reload'">
            <el-icon><RefreshRight /></el-icon>{{ $t('nginx.reload') }}
          </el-button>
          <el-button :disabled="!status.isRunning" @click="handleOperate('reopen')" :loading="operateLoading === 'reopen'">
            <el-icon><Document /></el-icon>{{ $t('nginx.reopen') }}
          </el-button>
          <el-button type="warning" :disabled="!status.isRunning" @click="handleOperate('quit')" :loading="operateLoading === 'quit'">
            <el-icon><SwitchButton /></el-icon>{{ $t('nginx.quit') }}
          </el-button>
          <el-divider direction="vertical" />
          <el-button @click="handleTestConfig" :loading="testLoading">
            <el-icon><Checked /></el-icon>{{ $t('nginx.testConfig') }}
          </el-button>
          <el-button type="danger" plain @click="handleUninstall">
            <el-icon><Delete /></el-icon>{{ $t('nginx.uninstall') }}
          </el-button>
          <el-divider direction="vertical" />
          <div class="autostart-toggle">
            <span class="autostart-label">{{ $t('nginx.autoStart') }}</span>
            <el-switch
              v-model="status.autoStart"
              :loading="autoStartLoading"
              @change="handleAutoStart"
            />
          </div>
        </div>
      </el-card>

      <!-- 安装信息 -->
      <el-card shadow="never" class="detail-card">
        <template #header>
          <span>{{ $t('nginx.installDir') }}</span>
        </template>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('nginx.mode')">
            <el-tag :type="status.systemMode ? 'success' : 'info'" size="small">
              {{ status.systemMode ? $t('nginx.systemMode') : $t('nginx.prefixMode') }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="$t('nginx.installDir')">{{ status.systemMode ? '/etc/nginx' : status.installDir }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nginx.version')">{{ status.version || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nginx.startedAt')">{{ status.isRunning ? formatTime(status.startedAt) : '-' }}</el-descriptions-item>
          <el-descriptions-item :label="$t('nginx.pid')">{{ status.isRunning ? status.pid : '-' }}</el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 配置测试结果 -->
      <el-card v-if="testResult !== null" shadow="never" class="test-card">
        <template #header>
          <span>{{ $t('nginx.testOutput') }}</span>
        </template>
        <el-alert :type="testResult.success ? 'success' : 'error'" :title="testResult.success ? $t('nginx.testSuccess') : $t('nginx.testFail')" :closable="false" show-icon />
        <pre class="config-output" v-if="testResult.output">{{ testResult.output }}</pre>
      </el-card>

        </el-tab-pane>

        <!-- 配置文件编辑 Tab -->
        <el-tab-pane :label="$t('website.confEditor')" name="config">
          <div class="conf-editor-section">
            <el-row :gutter="16">
              <el-col :span="6">
                <div class="conf-file-list">
                  <div class="conf-file-item" :class="{ active: activeConfFile === '__main__' }" @click="loadMainConf">
                    <el-icon><Setting /></el-icon>
                    <span>nginx.conf</span>
                  </div>
                  <el-divider style="margin: 8px 0">
                    <span style="font-size: 12px">conf.d/</span>
                  </el-divider>
                  <template v-if="confFiles.length > 0">
                    <div v-for="f in confFiles" :key="f.name" class="conf-file-item" :class="{ active: activeConfFile === f.name }" @click="loadConfFile(f.name)">
                      <el-icon><Document /></el-icon>
                      <span>{{ f.name }}</span>
                    </div>
                  </template>
                  <div v-else class="no-conf-files">{{ $t('website.noConfFiles') }}</div>
                </div>
              </el-col>
              <el-col :span="18">
                <div class="conf-editor-header">
                  <span class="conf-file-name">{{ activeConfFile === '__main__' ? 'nginx.conf' : activeConfFile || '选择文件' }}</span>
                  <el-button size="small" type="primary" @click="handleSaveConf" :loading="confSaving" :disabled="!activeConfFile">
                    {{ $t('website.saveConf') }}
                  </el-button>
                </div>
                <el-input v-model="confContent" type="textarea" :rows="24" class="conf-editor-textarea" :placeholder="activeConfFile ? '' : '请从左侧选择配置文件'" />
              </el-col>
            </el-row>
          </div>
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- 安装对话框 -->
    <el-dialog v-model="showInstallDialog" :title="$t('nginx.install')" width="520px" :close-on-click-modal="false">
      <el-form :model="installForm" label-width="100px">
        <el-form-item :label="$t('nginx.installMethod')">
          <el-radio-group v-model="installForm.method" @change="onInstallMethodChange">
            <el-radio value="apt">
              {{ $t('nginx.installMethodApt') }}
              <el-tag type="success" size="small" style="margin-left: 6px">{{ $t('nginx.recommended') }}</el-tag>
            </el-radio>
            <el-radio value="precompiled">{{ $t('nginx.installMethodPrecompiled') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="installForm.method === 'apt'">
          <el-alert :title="$t('nginx.installAptDesc')" type="info" :closable="false" show-icon />
        </el-form-item>
        <template v-if="installForm.method === 'precompiled'">
          <el-form-item :label="$t('nginx.installVersion')">
            <el-select
              v-model="installForm.version"
              :placeholder="$t('nginx.selectVersion')"
              :loading="versionsLoading"
              filterable
              style="width: 100%"
            >
              <el-option
                v-for="v in availableVersions"
                :key="v.version"
                :label="v.version"
                :value="v.version"
              >
                <div class="version-option">
                  <span>{{ v.version }}</span>
                  <span class="version-date">{{ formatDate(v.publishedAt) }}</span>
                </div>
              </el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="availableVersions.length === 0 && !versionsLoading">
            <el-alert :title="$t('nginx.noVersions')" type="warning" :closable="false" show-icon />
            <el-input v-model="installForm.version" :placeholder="$t('nginx.installVersionPlaceholder')" style="margin-top: 8px" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="showInstallDialog = false">{{ $t('commons.cancel') }}</el-button>
        <el-button
          type="primary"
          @click="handleInstall"
          :loading="installLoading"
          :disabled="installForm.method === 'precompiled' && !installForm.version"
        >{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Refresh, VideoPlay, VideoPause, RefreshRight, SwitchButton,
  Document, Delete, Box, Checked, Setting, Upload,
} from '@element-plus/icons-vue'
import {
  getNginxStatus,
  operateNginx,
  testNginxConfig,
  installNginx,
  getInstallProgress,
  uninstallNginx,
  listNginxVersions,
  setNginxAutoStart,
  checkNginxUpdate,
  upgradeNginx,
} from '@/api/modules/nginx'
import {
  getNginxMainConf,
  saveNginxMainConf,
  listNginxConfFiles,
  getNginxConfFile,
  saveNginxConfFile,
} from '@/api/modules/website'
import type { NginxStatus, NginxVersion, NginxTestResult, NginxInstallProgress, ConfFile } from '@/api/interface'

const { t } = useI18n()

const mainTab = ref('status')

// 状态数据
const loading = ref(false)
const status = ref<Partial<NginxStatus>>({})
const operateLoading = ref('')
const testLoading = ref(false)
const testResult = ref<NginxTestResult | null>(null)
const autoStartLoading = ref(false)

// 安装相关
const showInstallDialog = ref(false)
const installLoading = ref(false)
const installing = ref(false)
const uninstalling = ref(false)
const installProgress = ref<NginxInstallProgress>({ phase: 'idle', message: '', percent: 0 })
let progressTimer: ReturnType<typeof setInterval> | null = null

const installForm = reactive({
  method: 'apt' as 'apt' | 'precompiled',
  version: '',
})

// 版本更新
const updateCheckLoading = ref(false)
const upgradeLoading = ref(false)
const updateInfo = reactive({ hasUpdate: false, availableVersion: '' })

// 可用版本列表
const availableVersions = ref<NginxVersion[]>([])
const versionsLoading = ref(false)

// 加载状态
const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getNginxStatus()
    status.value = res.data || {}
  } catch { /* handled by interceptor */ }
  finally { loading.value = false }
}

// 获取可用版本
const loadVersions = async () => {
  versionsLoading.value = true
  try {
    const res = await listNginxVersions()
    availableVersions.value = res.data || []
    // 默认选择第一个版本
    if (availableVersions.value.length > 0 && !installForm.version) {
      installForm.version = availableVersions.value[0].version
    }
  } catch { /* handled */ }
  finally { versionsLoading.value = false }
}

// 操作 Nginx
const handleOperate = async (operation: string) => {
  operateLoading.value = operation
  try {
    await operateNginx(operation)
    ElMessage.success(t('commons.success'))
    await loadStatus()
  } catch { /* handled */ }
  finally { operateLoading.value = '' }
}

// 开机自启
const handleAutoStart = async (val: boolean) => {
  autoStartLoading.value = true
  try {
    await setNginxAutoStart(val as boolean)
    ElMessage.success(t('commons.success'))
  } catch {
    status.value.autoStart = !val
  }
  finally { autoStartLoading.value = false }
}

// 配置测试
const handleTestConfig = async () => {
  testLoading.value = true
  try {
    const res = await testNginxConfig()
    testResult.value = res.data
    if (res.data?.success) {
      ElMessage.success(t('nginx.testSuccess'))
    } else {
      ElMessage.error(t('nginx.testFail'))
    }
  } catch { /* handled */ }
  finally { testLoading.value = false }
}

// 显示安装对话框
const handleShowInstall = () => {
  installForm.method = 'apt'
  installForm.version = ''
  showInstallDialog.value = true
}

// 切换安装方式
const onInstallMethodChange = (val: string) => {
  if (val === 'precompiled' && availableVersions.value.length === 0) {
    loadVersions()
  }
}

// 安装 Nginx
const handleInstall = async () => {
  if (installForm.method === 'precompiled' && !installForm.version) {
    ElMessage.warning(t('nginx.selectVersion'))
    return
  }
  const confirmMsg = installForm.method === 'apt'
    ? t('nginx.installAptConfirm')
    : t('nginx.installConfirm', { version: installForm.version })
  try {
    await ElMessageBox.confirm(confirmMsg, t('commons.tip'), { type: 'warning' })
  } catch { return }

  installLoading.value = true
  try {
    await installNginx(installForm.method, installForm.method === 'precompiled' ? installForm.version : undefined)
    showInstallDialog.value = false
    installing.value = true
    startProgressPolling()
  } catch { /* handled */ }
  finally { installLoading.value = false }
}

// 卸载 Nginx
const handleUninstall = async () => {
  const siteCount = status.value.websiteCount || 0
  let forceCleanup = false

  if (siteCount > 0) {
    try {
      await ElMessageBox.confirm(
        t('nginx.uninstallHasSites', { count: siteCount }),
        t('commons.tip'),
        {
          type: 'error',
          confirmButtonText: t('nginx.uninstallForce'),
          cancelButtonText: t('commons.cancel'),
          dangerouslyUseHTMLString: true,
        },
      )
      forceCleanup = true
    } catch { return }
  } else {
    try {
      await ElMessageBox.confirm(t('nginx.uninstallConfirm'), t('commons.tip'), {
        type: 'error',
        confirmButtonText: t('commons.confirm'),
        cancelButtonText: t('commons.cancel'),
      })
    } catch { return }
  }

  try {
    await uninstallNginx(forceCleanup)
    ElMessage.success(t('commons.success'))
    await loadStatus()
    testResult.value = null
  } catch { /* handled */ }
}

const handleUninstallInactive = async () => {
  uninstalling.value = true
  try {
    const mode = status.value.systemMode ? 'prefix' : 'system'
    await uninstallNginx(false, mode)
    ElMessage.success(t('commons.success'))
    await loadStatus()
  } catch { /* handled */ }
  finally { uninstalling.value = false }
}

// 检查更新
const handleCheckUpdate = async () => {
  updateCheckLoading.value = true
  try {
    const res = await checkNginxUpdate()
    const data = res.data
    updateInfo.hasUpdate = data?.hasUpdate || false
    updateInfo.availableVersion = data?.availableVersion || ''
    if (!data?.hasUpdate) {
      ElMessage.success(t('nginx.alreadyLatest'))
    }
  } catch { /* handled */ }
  finally { updateCheckLoading.value = false }
}

// 升级 Nginx
const handleUpgrade = async () => {
  const ver = updateInfo.availableVersion
  try {
    await ElMessageBox.confirm(
      t('nginx.upgradeConfirm', { current: status.value.version, target: ver }),
      t('commons.tip'),
      { type: 'warning' },
    )
  } catch { return }

  upgradeLoading.value = true
  try {
    await upgradeNginx(status.value.systemMode ? undefined : ver)
    installing.value = true
    startProgressPolling()
  } catch { /* handled */ }
  finally { upgradeLoading.value = false }
}

// 安装进度轮询
const startProgressPolling = () => {
  stopProgressPolling()
  progressTimer = setInterval(async () => {
    try {
      const res = await getInstallProgress()
      installProgress.value = res.data || {}
      if (res.data?.phase === 'done' || res.data?.phase === 'error') {
        stopProgressPolling()
        if (res.data?.phase === 'done') {
          ElMessage.success(res.data.message)
        } else {
          ElMessage.error(res.data.message || t('nginx.installFailed'))
        }
        installing.value = false
        updateInfo.hasUpdate = false
        updateInfo.availableVersion = ''
        await loadStatus()
      }
    } catch { /* retry */ }
  }, 2000)
}

const stopProgressPolling = () => {
  if (progressTimer) {
    clearInterval(progressTimer)
    progressTimer = null
  }
}

// 进度状态计算
const progressStatus = computed(() => {
  const phase = installProgress.value?.phase
  if (phase === 'done') return 'success'
  if (phase === 'error') return 'exception'
  return undefined
})

const phaseTagType = computed(() => {
  const phase = installProgress.value?.phase
  if (phase === 'done') return 'success'
  if (phase === 'error') return 'danger'
  if (phase === 'verify') return 'warning'
  return 'info'
})

const phaseLabel = computed(() => {
  const map: Record<string, string> = {
    idle: t('nginx.phaseIdle'),
    download: t('nginx.phaseDownload'),
    verify: t('nginx.phaseVerify'),
    install: t('nginx.phaseInstall'),
    done: t('nginx.phaseDone'),
    error: t('nginx.phaseError'),
  }
  return map[installProgress.value?.phase] || installProgress.value?.phase
})

// 时间格式化
const formatTime = (timeStr?: string) => {
  if (!timeStr) return '-'
  try {
    const d = new Date(timeStr)
    if (isNaN(d.getTime())) return '-'
    return d.toLocaleString('zh-CN', { hour12: false })
  } catch { return '-' }
}

const formatDate = (dateStr?: string) => {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return ''
    return d.toLocaleDateString('zh-CN')
  } catch { return '' }
}

// --- 配置文件编辑 ---
const confFiles = ref<ConfFile[]>([])
const activeConfFile = ref('')
const confContent = ref('')
const confSaving = ref(false)

const loadConfFilesList = async () => {
  try {
    const res = await listNginxConfFiles()
    confFiles.value = res.data || []
  } catch { confFiles.value = [] }
}

const loadMainConf = async () => {
  activeConfFile.value = '__main__'
  try {
    const res = await getNginxMainConf()
    confContent.value = res.data || ''
  } catch { confContent.value = '' }
}

const loadConfFile = async (name: string) => {
  activeConfFile.value = name
  try {
    const res = await getNginxConfFile(name)
    confContent.value = res.data || ''
  } catch { confContent.value = '' }
}

const handleSaveConf = async () => {
  if (!activeConfFile.value || !confContent.value) return
  try {
    await ElMessageBox.confirm(t('website.saveConfConfirm'), t('commons.tip'), { type: 'warning' })
  } catch { return }

  confSaving.value = true
  try {
    if (activeConfFile.value === '__main__') {
      await saveNginxMainConf(confContent.value)
    } else {
      const confDir = status.value.systemMode ? '/etc/nginx/sites-available' : (status.value.installDir ? `${status.value.installDir}/conf/conf.d` : '')
      await saveNginxConfFile(`${confDir}/${activeConfFile.value}`, confContent.value)
    }
    ElMessage.success(t('website.confSaved'))
  } catch {}
  finally { confSaving.value = false }
}

onMounted(() => {
  loadStatus()
  loadConfFilesList()
})
onUnmounted(() => stopProgressPolling())
</script>

<style lang="scss" scoped>
.nginx-page {
  height: 100%;
}

.info-row {
  margin-bottom: 16px;
}

.stat-card {
  text-align: center;
  min-height: 120px;
  display: flex;
  flex-direction: column;
  justify-content: center;

  .stat-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--xp-text-secondary);
    margin-bottom: 12px;
  }

  .stat-value {
    font-size: 14px;
  }

  .version-text {
    font-size: 22px;
    font-weight: 600;
    color: var(--xp-accent);
  }

  .update-hint {
    margin-top: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
}

.operate-card {
  margin-bottom: 16px;
}

.operate-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;

  .autostart-toggle {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .autostart-label {
    font-size: 13px;
    color: var(--xp-text-secondary);
    white-space: nowrap;
  }
}

.detail-card {
  margin-bottom: 16px;
}

.test-card {
  margin-bottom: 16px;

  .config-output {
    margin-top: 12px;
    padding: 12px;
    background: var(--xp-bg-darker, #0d1117);
    border-radius: var(--xp-radius-sm);
    color: var(--xp-text-secondary);
    font-family: 'Fira Code', 'Cascadia Code', 'Consolas', monospace;
    font-size: 12px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 300px;
    overflow-y: auto;
  }
}

.install-card {
  margin-bottom: 16px;

  .install-actions {
    margin-top: 16px;
    display: flex;
    gap: 12px;
    justify-content: center;
  }
}

.progress-card {
  margin-bottom: 16px;
}

.progress-content {
  padding: 16px 0;

  .progress-phase {
    margin-top: 16px;
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .progress-msg {
    color: var(--xp-text-secondary);
    font-size: 13px;
  }
}

.version-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.version-date {
  font-size: 12px;
  color: var(--xp-text-muted, #666);
}

.nginx-tabs {
  :deep(.el-tabs__header) {
    margin-bottom: 16px;
  }
}

.conf-editor-section {
  min-height: 500px;
}

.conf-file-list {
  background: var(--xp-bg-inset);
  border-radius: var(--xp-radius-sm);
  padding: 8px;
  min-height: 400px;
}

.conf-file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--xp-radius-sm);
  cursor: pointer;
  font-size: 13px;
  color: var(--xp-text-secondary);
  transition: all 0.15s;

  &:hover {
    background: var(--xp-accent-muted);
    color: var(--xp-text-primary);
  }

  &.active {
    background: var(--xp-accent-muted);
    color: var(--xp-accent);
    font-weight: 500;
  }
}

.no-conf-files {
  text-align: center;
  color: var(--xp-text-muted);
  font-size: 12px;
  padding: 20px 0;
}

.conf-editor-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;

  .conf-file-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--xp-text-primary);
  }
}

.conf-editor-textarea {
  :deep(textarea) {
    font-family: var(--xp-font-mono);
    font-size: 13px;
    line-height: 1.6;
    background: var(--xp-bg-inset);
    color: var(--xp-text-secondary);
  }
}
</style>
