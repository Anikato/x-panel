<template>
  <div class="haproxy-status-page">
    <div class="page-header">
      <h3>{{ $t('haproxy.status') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadStatus" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <template v-if="!status.isInstalled && !installing">
      <el-card shadow="never" class="install-card">
        <el-empty :description="$t('haproxy.notInstalled')">
          <template #image>
            <el-icon :size="64" color="var(--xp-text-muted)"><Aim /></el-icon>
          </template>
          <el-button type="primary" @click="handleInstall">{{ $t('haproxy.install') }}</el-button>
        </el-empty>
      </el-card>
    </template>

    <template v-if="installing">
      <el-card shadow="never">
        <template #header>{{ $t('haproxy.installProgress') }}</template>
        <el-progress
          :percentage="installProgress.percent"
          :status="installProgress.phase === 'error' ? 'exception' : installProgress.phase === 'done' ? 'success' : undefined"
          :stroke-width="18"
          :text-inside="true"
        />
        <div style="margin-top: 12px;">
          <el-tag :type="installProgress.phase === 'error' ? 'danger' : installProgress.phase === 'done' ? 'success' : 'primary'" size="small">
            {{ installProgress.phase }}
          </el-tag>
          <span style="margin-left: 8px;">{{ installProgress.message }}</span>
        </div>
      </el-card>
    </template>

    <template v-if="status.isInstalled && !installing">
      <el-alert type="success" :closable="false" show-icon style="margin-bottom: 16px;">
        {{ $t('haproxy.infoNote') }}
      </el-alert>

      <el-row :gutter="16">
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('commons.status') }}</div>
            <div class="stat-value">
              <el-tag :type="status.isRunning ? 'success' : 'danger'" size="large" effect="dark" round>
                {{ status.isRunning ? $t('haproxy.running') : $t('haproxy.stopped') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('haproxy.version') }}</div>
            <div class="stat-value version-text">{{ status.version || '-' }}</div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('haproxy.socketReady') }}</div>
            <div class="stat-value">
              <el-tag :type="status.socketReady ? 'success' : 'danger'" size="large" effect="dark" round>
                {{ status.socketReady ? $t('haproxy.socketOk') : $t('haproxy.socketBad') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('haproxy.autoStart') }}</div>
            <div class="stat-value">
              <el-tag :type="status.autoStart ? 'success' : 'info'" size="large" effect="dark" round>
                {{ status.autoStart ? $t('haproxy.autoStartEnabled') : '-' }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="16" style="margin-top: 16px;">
        <el-col :span="14">
          <el-card shadow="never">
            <template #header>{{ $t('commons.operate') }}</template>
            <div class="operate-buttons">
              <el-button type="success" :disabled="status.isRunning" @click="handleOperate('start')" :loading="operateLoading === 'start'">
                <el-icon><VideoPlay /></el-icon>{{ $t('haproxy.start') }}
              </el-button>
              <el-button type="danger" :disabled="!status.isRunning" @click="handleOperate('stop')" :loading="operateLoading === 'stop'">
                <el-icon><VideoPause /></el-icon>{{ $t('haproxy.stop') }}
              </el-button>
              <el-button type="primary" :disabled="!status.isRunning" @click="handleOperate('reload')" :loading="operateLoading === 'reload'">
                <el-icon><RefreshRight /></el-icon>{{ $t('haproxy.reload') }}
              </el-button>
              <el-button type="warning" :disabled="!status.isRunning" @click="handleOperate('restart')" :loading="operateLoading === 'restart'">
                <el-icon><Refresh /></el-icon>{{ $t('haproxy.restart') }}
              </el-button>
              <el-divider direction="vertical" />
              <el-button type="danger" plain @click="handleUninstall">
                <el-icon><Delete /></el-icon>{{ $t('haproxy.uninstall') }}
              </el-button>
            </div>
          </el-card>
        </el-col>
        <el-col :span="10">
          <el-card shadow="never">
            <template #header>{{ $t('haproxy.checkUpdate') }}</template>
            <div v-if="!updateInfo">
              <el-button type="primary" plain @click="handleCheckUpdate" :loading="checkUpdateLoading">
                <el-icon><Upload /></el-icon>{{ $t('haproxy.checkUpdate') }}
              </el-button>
            </div>
            <div v-else>
              <el-descriptions :column="1" border size="small">
                <el-descriptions-item :label="$t('haproxy.currentVersion')">{{ updateInfo.currentVersion || '-' }}</el-descriptions-item>
                <el-descriptions-item :label="$t('haproxy.availableVersion')">{{ updateInfo.availableVersion || '-' }}</el-descriptions-item>
              </el-descriptions>
              <div style="margin-top: 12px; display: flex; gap: 8px;">
                <el-button v-if="updateInfo.hasUpdate" type="warning" @click="handleUpgrade" :loading="upgrading">
                  {{ $t('haproxy.upgradeNow') }}
                </el-button>
                <el-tag v-else type="success">{{ $t('haproxy.alreadyLatest') }}</el-tag>
                <el-button text @click="handleCheckUpdate" :loading="checkUpdateLoading">
                  <el-icon><Refresh /></el-icon>
                </el-button>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-card shadow="never" style="margin-top: 16px;">
        <template #header>{{ $t('haproxy.statsSection') }}</template>
        <el-descriptions :column="3" border size="small">
          <el-descriptions-item :label="$t('haproxy.statsBind')"><code>{{ status.statsBind }}</code></el-descriptions-item>
          <el-descriptions-item :label="$t('haproxy.statsURI')"><code>{{ status.statsURI }}</code></el-descriptions-item>
          <el-descriptions-item :label="$t('haproxy.statsUser')"><code>{{ status.statsUser }}</code></el-descriptions-item>
        </el-descriptions>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Refresh, Aim, VideoPlay, VideoPause, RefreshRight, Delete, Upload } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  getHAProxyStatus, installHAProxy, getHAProxyInstallProgress, uninstallHAProxy,
  operateHAProxy, checkHAProxyUpdate, upgradeHAProxy,
} from '@/api/modules/haproxy'

const { t } = useI18n()
const loading = ref(false)
const installing = ref(false)
const operateLoading = ref('')
const checkUpdateLoading = ref(false)
const upgrading = ref(false)
const updateInfo = ref<any>(null)
let pollTimer: ReturnType<typeof setInterval> | null = null

const status = ref<any>({
  isInstalled: false,
  isRunning: false,
  version: '',
  socketReady: false,
  autoStart: false,
  statsBind: '',
  statsURI: '',
  statsUser: '',
})

const installProgress = ref({ phase: 'idle', message: '', percent: 0 })

const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getHAProxyStatus()
    if (res.data) status.value = res.data
  } finally {
    loading.value = false
  }
}

const handleInstall = async () => {
  installing.value = true
  try {
    await installHAProxy()
    startPollProgress()
  } catch {
    installing.value = false
  }
}

const startPollProgress = () => {
  pollTimer = setInterval(async () => {
    try {
      const res = await getHAProxyInstallProgress()
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
  await ElMessageBox.confirm(t('haproxy.uninstallConfirm'), t('commons.warning'), { type: 'warning' })
  try {
    await uninstallHAProxy()
    ElMessage.success(t('commons.operationSuccess'))
    await loadStatus()
  } catch { /* ignore */ }
}

const handleOperate = async (op: string) => {
  operateLoading.value = op
  try {
    await operateHAProxy(op)
    ElMessage.success(t('commons.operationSuccess'))
    setTimeout(() => loadStatus(), 800)
  } finally {
    operateLoading.value = ''
  }
}

const handleCheckUpdate = async () => {
  checkUpdateLoading.value = true
  try {
    const res = await checkHAProxyUpdate()
    if (res.data) updateInfo.value = res.data
  } finally {
    checkUpdateLoading.value = false
  }
}

const handleUpgrade = async () => {
  await ElMessageBox.confirm(t('haproxy.upgradeConfirm'), t('commons.warning'), { type: 'warning' })
  upgrading.value = true
  installing.value = true
  try {
    await upgradeHAProxy()
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
.haproxy-status-page { padding: 0; }
.page-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.install-card { text-align: center; }
.stat-card {
  text-align: center;
  .stat-title { font-size: 13px; color: var(--xp-text-muted); margin-bottom: 12px; }
  .stat-value { font-size: 15px; font-weight: 600; }
  .version-text { font-family: monospace; }
}
.operate-buttons {
  display: flex; flex-wrap: wrap; gap: 8px; align-items: center;
}
</style>
