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
            <el-button type="primary" @click="showInstallDialog = true">
              {{ $t('nginx.install') }}
            </el-button>
            <el-button @click="handleCheckDeps">
              {{ $t('nginx.checkDeps') }}
            </el-button>
          </div>
        </el-empty>
      </el-card>

      <!-- 依赖检查结果 -->
      <el-card v-if="depsChecked" shadow="never" class="deps-card">
        <template #header>
          <span>{{ $t('nginx.checkDeps') }}</span>
        </template>
        <el-result v-if="depsAllSatisfied" icon="success" :title="$t('nginx.depsOk')" />
        <div v-else>
          <el-alert :title="$t('nginx.depsMissing')" type="warning" :closable="false" show-icon>
            <template #default>
              <ul class="deps-list">
                <li v-for="dep in depsMissing" :key="dep">{{ dep }}</li>
              </ul>
            </template>
          </el-alert>
        </div>
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
            <div class="stat-title">{{ $t('nginx.version') }}</div>
            <div class="stat-value version-text">{{ status.version || '-' }}</div>
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
        </div>
      </el-card>

      <!-- 安装信息 -->
      <el-card shadow="never" class="detail-card">
        <template #header>
          <span>{{ $t('nginx.installDir') }}</span>
        </template>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="$t('nginx.installDir')">{{ status.installDir }}</el-descriptions-item>
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
    </template>

    <!-- 安装对话框 -->
    <el-dialog v-model="showInstallDialog" :title="$t('nginx.install')" width="460px" :close-on-click-modal="false">
      <el-form :model="installForm" label-width="100px">
        <el-form-item :label="$t('nginx.installVersion')">
          <el-input v-model="installForm.version" :placeholder="$t('nginx.installVersionPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showInstallDialog = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleInstall" :loading="installLoading">{{ $t('commons.confirm') }}</el-button>
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
  Document, Delete, Box, Checked,
} from '@element-plus/icons-vue'
import {
  getNginxStatus,
  operateNginx,
  testNginxConfig,
  installNginx,
  getInstallProgress,
  uninstallNginx,
  checkNginxDeps,
} from '@/api/modules/nginx'

const { t } = useI18n()

// 状态数据
const loading = ref(false)
const status = ref<any>({})
const operateLoading = ref('')
const testLoading = ref(false)
const testResult = ref<any>(null)

// 安装相关
const showInstallDialog = ref(false)
const installLoading = ref(false)
const installing = ref(false)
const installProgress = ref<any>({ phase: 'idle', message: '', percent: 0 })
let progressTimer: ReturnType<typeof setInterval> | null = null

const installForm = reactive({
  version: '1.26.2',
})

// 依赖检查
const depsChecked = ref(false)
const depsAllSatisfied = ref(false)
const depsMissing = ref<string[]>([])

// 加载状态
const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getNginxStatus()
    status.value = res.data || {}
  } catch { /* handled by interceptor */ }
  finally { loading.value = false }
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

// 检查依赖
const handleCheckDeps = async () => {
  try {
    const res = await checkNginxDeps()
    depsChecked.value = true
    depsAllSatisfied.value = res.data?.allSatisfied || false
    depsMissing.value = res.data?.missing || []
  } catch { /* handled */ }
}

// 安装 Nginx
const handleInstall = async () => {
  if (!installForm.version) {
    ElMessage.warning(t('nginx.installVersionPlaceholder'))
    return
  }
  try {
    await ElMessageBox.confirm(
      t('nginx.installConfirm', { version: installForm.version }),
      t('commons.tip'),
      { type: 'warning' },
    )
  } catch { return }

  installLoading.value = true
  try {
    await installNginx(installForm.version)
    showInstallDialog.value = false
    installing.value = true
    startProgressPolling()
  } catch { /* handled */ }
  finally { installLoading.value = false }
}

// 卸载 Nginx
const handleUninstall = async () => {
  try {
    await ElMessageBox.confirm(t('nginx.uninstallConfirm'), t('commons.tip'), {
      type: 'error',
      confirmButtonText: t('commons.confirm'),
      cancelButtonText: t('commons.cancel'),
    })
  } catch { return }

  try {
    await uninstallNginx()
    ElMessage.success(t('commons.success'))
    await loadStatus()
    testResult.value = null
  } catch { /* handled */ }
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
          installing.value = false
          await loadStatus()
        }
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
  if (phase === 'compile') return 'warning'
  return 'info'
})

const phaseLabel = computed(() => {
  const map: Record<string, string> = {
    idle: t('nginx.phaseIdle'),
    download: t('nginx.phaseDownload'),
    configure: t('nginx.phaseConfigure'),
    compile: t('nginx.phaseCompile'),
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

onMounted(() => loadStatus())
onUnmounted(() => stopProgressPolling())
</script>

<style lang="scss" scoped>
.nginx-page {
  height: 100%;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 {
    margin: 0;
    font-size: 16px;
    color: var(--xp-text-primary);
  }
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
}

.operate-card {
  margin-bottom: 16px;
}

.operate-buttons {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
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

.deps-card {
  margin-bottom: 16px;

  .deps-list {
    margin: 8px 0 0;
    padding-left: 20px;

    li {
      line-height: 1.8;
      color: var(--xp-text-secondary);
    }
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
</style>
