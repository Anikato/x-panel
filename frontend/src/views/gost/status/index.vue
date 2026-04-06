<template>
  <div class="gost-status-page">
    <div class="page-header">
      <h3>{{ $t('gost.status') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadStatus" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <!-- 未安装 -->
    <template v-if="!status.isInstalled && !installing">
      <el-card shadow="never" class="install-card">
        <el-empty :description="$t('gost.notInstalled')">
          <template #image>
            <el-icon :size="64" color="var(--xp-text-muted)"><Promotion /></el-icon>
          </template>
          <div class="install-actions">
            <el-button type="primary" @click="handleInstall">
              {{ $t('gost.install') }}
            </el-button>
          </div>
        </el-empty>
      </el-card>
    </template>

    <!-- 安装中 -->
    <template v-if="installing">
      <el-card shadow="never">
        <template #header>
          <span>{{ $t('gost.installProgress') }}</span>
        </template>
        <div class="progress-content">
          <el-progress
            :percentage="installProgress.percent"
            :status="installProgress.phase === 'error' ? 'exception' : installProgress.phase === 'done' ? 'success' : undefined"
            :stroke-width="18"
            :text-inside="true"
          />
          <div class="progress-phase" style="margin-top: 12px;">
            <el-tag
              :type="installProgress.phase === 'error' ? 'danger' : installProgress.phase === 'done' ? 'success' : 'primary'"
              size="small"
            >
              {{ installProgress.phase }}
            </el-tag>
            <span style="margin-left: 8px;">{{ installProgress.message }}</span>
          </div>
        </div>
      </el-card>
    </template>

    <!-- 已安装 -->
    <template v-if="status.isInstalled && !installing">
      <el-alert type="success" :closable="false" show-icon style="margin-bottom: 16px;">
        {{ $t('gost.infoNote') }}
      </el-alert>

      <el-row :gutter="16" class="info-row">
        <el-col :span="5">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('commons.status') }}</div>
            <div class="stat-value">
              <el-tag :type="status.isRunning ? 'success' : 'danger'" size="large" effect="dark" round>
                {{ status.isRunning ? $t('gost.running') : $t('gost.stopped') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="5">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('gost.version') }}</div>
            <div class="stat-value version-text">{{ status.version || '-' }}</div>
          </el-card>
        </el-col>
        <el-col :span="5">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('gost.apiStatus') }}</div>
            <div class="stat-value">
              <el-tag :type="status.apiReady ? 'success' : 'danger'" size="large" effect="dark" round>
                {{ status.apiReady ? $t('gost.apiReady') : $t('gost.apiUnreachable') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="5">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('gost.autoStart') }}</div>
            <div class="stat-value">
              <el-tag type="success" size="large" effect="dark" round>{{ $t('gost.autoStartEnabled') }}</el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="4">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('gost.sync') }}</div>
            <div class="stat-value">
              <el-button type="primary" size="small" @click="handleSync" :loading="syncLoading" :disabled="!status.apiReady">
                {{ $t('gost.sync') }}
              </el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16" style="margin-top: 16px;">
        <el-col :span="12">
          <el-card shadow="never">
            <template #header>
              <span>{{ $t('commons.operate') }}</span>
            </template>
            <div class="operate-buttons">
              <el-button type="success" :disabled="status.isRunning" @click="handleOperate('start')" :loading="operateLoading === 'start'">
                <el-icon><VideoPlay /></el-icon>{{ $t('gost.start') }}
              </el-button>
              <el-button type="danger" :disabled="!status.isRunning" @click="handleOperate('stop')" :loading="operateLoading === 'stop'">
                <el-icon><VideoPause /></el-icon>{{ $t('gost.stop') }}
              </el-button>
              <el-button type="primary" :disabled="!status.isRunning" @click="handleOperate('restart')" :loading="operateLoading === 'restart'">
                <el-icon><RefreshRight /></el-icon>{{ $t('gost.restart') }}
              </el-button>
              <el-divider direction="vertical" />
              <el-button type="danger" plain @click="handleUninstall">
                <el-icon><Delete /></el-icon>{{ $t('gost.uninstall') }}
              </el-button>
            </div>
          </el-card>
        </el-col>
        <el-col :span="12">
          <el-card shadow="never">
            <template #header>
              <span>{{ $t('gost.checkUpdate') }}</span>
            </template>
            <div v-if="!updateInfo">
              <el-button type="primary" plain @click="handleCheckUpdate" :loading="checkUpdateLoading">
                <el-icon><Upload /></el-icon>{{ $t('gost.checkUpdate') }}
              </el-button>
            </div>
            <div v-else>
              <el-descriptions :column="1" border size="small">
                <el-descriptions-item :label="$t('gost.currentVersion')">{{ updateInfo.currentVersion }}</el-descriptions-item>
                <el-descriptions-item :label="$t('gost.latestVersion')">
                  <a :href="updateInfo.releaseURL" target="_blank" style="color: var(--el-color-primary)">{{ updateInfo.latestVersion }}</a>
                </el-descriptions-item>
              </el-descriptions>
              <div style="margin-top: 12px; display: flex; align-items: center; gap: 8px;">
                <el-button
                  v-if="updateInfo.hasUpdate"
                  type="warning"
                  @click="handleUpgrade"
                  :loading="upgrading"
                >
                  {{ $t('gost.upgradeNow') }}
                </el-button>
                <el-tag v-else type="success">{{ $t('gost.alreadyLatest') }}</el-tag>
                <el-button text @click="handleCheckUpdate" :loading="checkUpdateLoading">
                  <el-icon><Refresh /></el-icon>
                </el-button>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Refresh, Promotion, VideoPlay, VideoPause, RefreshRight, Delete, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getGostStatus, installGost, getGostInstallProgress, uninstallGost, operateGost, syncGost, checkGostUpdate, upgradeGost } from '@/api/modules/gost'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const installing = ref(false)
const operateLoading = ref('')
const syncLoading = ref(false)
const checkUpdateLoading = ref(false)
const upgrading = ref(false)
const updateInfo = ref<any>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

const status = ref({
  isInstalled: false,
  isRunning: false,
  version: '',
  apiReady: false,
})

const installProgress = ref({
  phase: 'idle',
  message: '',
  percent: 0,
})

const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getGostStatus()
    if (res.data) {
      status.value = res.data
    }
  } finally {
    loading.value = false
  }
}

const handleInstall = async () => {
  installing.value = true
  try {
    await installGost()
    startPollProgress()
  } catch {
    installing.value = false
  }
}

const startPollProgress = () => {
  pollTimer = setInterval(async () => {
    try {
      const res = await getGostInstallProgress()
      if (res.data) {
        installProgress.value = res.data
        if (res.data.phase === 'done' || res.data.phase === 'error') {
          stopPollProgress()
          if (res.data.phase === 'done') {
            ElMessage.success(res.data.message)
            installing.value = false
            upgrading.value = false
            updateInfo.value = null
            await loadStatus()
          }
        }
      }
    } catch {
      stopPollProgress()
    }
  }, 1500)
}

const stopPollProgress = () => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

const handleUninstall = async () => {
  await ElMessageBox.confirm(t('gost.uninstallConfirm'), t('commons.warning'), { type: 'warning' })
  try {
    await uninstallGost()
    ElMessage.success(t('commons.operationSuccess'))
    await loadStatus()
  } catch { /* handled by interceptor */ }
}

const handleOperate = async (op: string) => {
  operateLoading.value = op
  try {
    await operateGost(op)
    ElMessage.success(t('commons.operationSuccess'))
    setTimeout(() => loadStatus(), 1000)
  } finally {
    operateLoading.value = ''
  }
}

const handleSync = async () => {
  syncLoading.value = true
  try {
    await syncGost()
    ElMessage.success(t('gost.syncSuccess'))
  } finally {
    syncLoading.value = false
  }
}

const handleCheckUpdate = async () => {
  checkUpdateLoading.value = true
  try {
    const res = await checkGostUpdate()
    if (res.data) {
      updateInfo.value = res.data
    }
  } finally {
    checkUpdateLoading.value = false
  }
}

const handleUpgrade = async () => {
  if (!updateInfo.value?.latestVersion) return
  await ElMessageBox.confirm(
    t('gost.upgradeConfirm', { version: updateInfo.value.latestVersion }),
    t('commons.warning'),
    { type: 'warning' }
  )
  upgrading.value = true
  installing.value = true
  try {
    await upgradeGost(updateInfo.value.latestVersion)
    startPollProgress()
  } catch {
    upgrading.value = false
    installing.value = false
  }
}

onMounted(() => loadStatus())
onUnmounted(() => stopPollProgress())
</script>

<style lang="scss" scoped>
.gost-status-page {
  padding: 0;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.install-card {
  text-align: center;
  .install-actions {
    margin-top: 16px;
  }
}
.info-row {
  .stat-card {
    text-align: center;
    .stat-title {
      font-size: 13px;
      color: var(--xp-text-muted);
      margin-bottom: 12px;
    }
    .stat-value { font-size: 15px; font-weight: 600; }
    .version-text { font-family: monospace; }
  }
}
.operate-buttons {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.progress-content {
  padding: 10px 0;
}
</style>
