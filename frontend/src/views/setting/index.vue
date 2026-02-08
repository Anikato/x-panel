<template>
  <div class="setting-page">
    <!-- 版本信息 -->
    <el-card class="setting-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><InfoFilled /></el-icon>
            <span>{{ t('setting.versionInfo') }}</span>
          </div>
        </div>
      </template>
      <el-descriptions :column="2" border>
        <el-descriptions-item :label="t('setting.currentVersion')">
          <el-tag v-if="versionInfo.version === 'dev'" type="warning" effect="plain">
            {{ t('setting.dev') }}
          </el-tag>
          <el-tag v-else type="success" effect="plain">
            {{ versionInfo.version }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('setting.buildTime')">
          {{ versionInfo.buildTime || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('setting.commitHash')">
          <el-text class="mono-text">{{ versionInfo.commitHash || '-' }}</el-text>
        </el-descriptions-item>
        <el-descriptions-item :label="t('setting.goVersion')">
          {{ versionInfo.goVersion || '-' }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 更新区域 -->
      <div class="update-section">
        <div v-if="versionInfo.version === 'dev'" class="dev-notice">
          <el-alert :title="t('setting.devTip')" type="info" show-icon :closable="false" />
        </div>
        <template v-else>
          <!-- 更新源配置 -->
          <div class="update-url-row">
            <el-input
              v-model="upgradeUrl"
              :placeholder="t('setting.upgradeUrlPlaceholder')"
              clearable
              style="flex: 1; margin-right: 12px"
            >
              <template #prepend>{{ t('setting.upgradeUrl') }}</template>
            </el-input>
            <el-button
              type="primary"
              :loading="checking"
              :icon="Refresh"
              @click="handleCheckUpdate"
            >
              {{ checking ? t('setting.checking') : t('setting.checkUpdate') }}
            </el-button>
          </div>

          <!-- GitHub Token（私有仓库必须） -->
          <div class="update-url-row" style="margin-top: 8px">
            <el-input
              v-model="githubToken"
              :placeholder="t('setting.githubTokenPlaceholder')"
              clearable
              show-password
              style="flex: 1; margin-right: 12px"
            >
              <template #prepend>{{ t('setting.githubToken') }}</template>
            </el-input>
            <el-button
              :loading="savingToken"
              @click="handleSaveToken"
            >
              {{ t('setting.save') }}
            </el-button>
          </div>
          <div class="update-url-hint">
            <el-text type="info" size="small">
              {{ t('setting.upgradeUrlHint') }}。{{ t('setting.githubTokenHint') }}
            </el-text>
          </div>

          <!-- 更新结果 -->
          <div v-if="upgradeInfo" class="update-result">
            <el-alert
              v-if="!upgradeInfo.hasUpdate"
              :title="t('setting.noUpdate')"
              type="success"
              show-icon
              :closable="false"
            />
            <el-card v-else shadow="hover" class="update-card">
              <div class="update-card-header">
                <el-tag type="danger" effect="dark" size="large">
                  {{ t('setting.hasUpdate') }}: {{ upgradeInfo.latestVersion }}
                </el-tag>
                <el-text type="info" v-if="upgradeInfo.publishDate">
                  {{ t('setting.publishDate') }}: {{ upgradeInfo.publishDate }}
                </el-text>
              </div>
              <div v-if="upgradeInfo.releaseNote" class="release-note">
                <el-text tag="p" style="white-space: pre-wrap">{{ upgradeInfo.releaseNote }}</el-text>
              </div>
              <el-button
                type="danger"
                :loading="upgrading"
                size="large"
                @click="handleUpgrade"
              >
                {{ upgrading ? t('setting.upgrading') : t('setting.doUpgrade') }}
              </el-button>
            </el-card>
          </div>

          <!-- 升级日志 -->
          <div v-if="upgradeLog" class="upgrade-log-section">
            <el-text tag="div" type="info" style="margin-bottom: 8px">
              {{ t('setting.upgradeLog') }}
            </el-text>
            <el-input
              type="textarea"
              :model-value="upgradeLog"
              :rows="8"
              readonly
              class="log-textarea"
            />
          </div>
        </template>
      </div>
    </el-card>

    <!-- 面板设置 -->
    <el-card v-loading="loading" class="setting-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Setting /></el-icon>
            <span>{{ t('setting.title') }}</span>
          </div>
        </div>
      </template>
      <el-form :model="form" label-width="140px" style="max-width: 600px">
        <el-form-item :label="t('setting.panelName')">
          <el-input v-model="form.panelName" />
        </el-form-item>
        <el-form-item :label="t('setting.sessionTimeout')">
          <el-input-number v-model="form.sessionTimeout" :min="3600" :step="3600" />
        </el-form-item>
        <el-form-item :label="t('setting.securityEntrance')">
          <el-input
            v-model="form.securityEntrance"
            :placeholder="t('setting.securityEntrancePlaceholder')"
            clearable
          >
            <template #prepend>/</template>
          </el-input>
          <div style="margin-top: 4px">
            <el-text type="info" size="small">
              {{ t('setting.securityEntranceHint') }}
            </el-text>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleSave">
            {{ t('setting.save') }}
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Setting, InfoFilled } from '@element-plus/icons-vue'
import { getSettingInfo, updateSetting } from '@/api/modules/setting'
import { getCurrentVersion, checkUpdate, doUpgrade, getUpgradeLog } from '@/api/modules/upgrade'
import { useGlobalStore } from '@/store/modules/global'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const globalStore = useGlobalStore()

// 面板设置
const loading = ref(false)
const saving = ref(false)
const form = reactive({ panelName: 'X-Panel', sessionTimeout: 86400, securityEntrance: '' })

// 版本与升级
const versionInfo = reactive({
  version: '',
  commitHash: '',
  buildTime: '',
  goVersion: '',
})
const upgradeUrl = ref('')
const githubToken = ref('')
const savingToken = ref(false)
const checking = ref(false)
const upgrading = ref(false)
const upgradeInfo = ref<any>(null)
const upgradeLog = ref('')

// 加载当前版本信息
const fetchVersion = async () => {
  try {
    const res: any = await getCurrentVersion()
    if (res.data) {
      Object.assign(versionInfo, res.data)
    }
  } catch { /* */ }
}

// 检查更新
const handleCheckUpdate = async () => {
  checking.value = true
  upgradeInfo.value = null
  try {
    const res: any = await checkUpdate({
      releaseUrl: upgradeUrl.value || undefined,
    })
    if (res.data) {
      upgradeInfo.value = res.data
    }
  } catch { /* */ } finally {
    checking.value = false
  }
}

// 执行升级
const handleUpgrade = async () => {
  if (!upgradeInfo.value) return

  try {
    await ElMessageBox.confirm(
      t('setting.upgradeConfirm', { version: upgradeInfo.value.latestVersion }),
      t('commons.tip'),
      { type: 'warning', confirmButtonText: t('commons.confirm'), cancelButtonText: t('commons.cancel') },
    )
  } catch {
    return
  }

  upgrading.value = true
  try {
    await doUpgrade({
      version: upgradeInfo.value.latestVersion,
      downloadUrl: upgradeInfo.value.downloadUrl,
      checksumUrl: upgradeInfo.value.checksumUrl || undefined,
    })
    ElMessage.success(t('setting.upgradeStarted'))

    // 轮询升级日志
    pollUpgradeLog()
  } catch {
    ElMessage.error(t('setting.upgradeFailed'))
    upgrading.value = false
  }
}

// 轮询升级日志
let logTimer: ReturnType<typeof setInterval> | null = null
const pollUpgradeLog = () => {
  if (logTimer) clearInterval(logTimer)
  logTimer = setInterval(async () => {
    try {
      const res: any = await getUpgradeLog()
      if (res.data) {
        upgradeLog.value = res.data
      }
    } catch {
      // 服务器可能已重启
      if (logTimer) clearInterval(logTimer)
      upgrading.value = false
      // 几秒后刷新页面
      setTimeout(() => window.location.reload(), 3000)
    }
  }, 2000)
}

// 保存 GitHub Token
const handleSaveToken = async () => {
  savingToken.value = true
  try {
    await updateSetting({ key: 'GitHubToken', value: githubToken.value })
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { savingToken.value = false }
}

// 面板设置相关
const fetchSettings = async () => {
  loading.value = true
  try {
    const res: any = await getSettingInfo()
    if (res.data) {
      form.panelName = res.data.panelName || 'X-Panel'
      form.sessionTimeout = parseInt(res.data.sessionTimeout) || 86400
      form.securityEntrance = res.data.securityEntrance || ''
      githubToken.value = res.data.githubToken || ''
    }
  } catch { /* */ } finally { loading.value = false }
}

const handleSave = async () => {
  saving.value = true
  try {
    await updateSetting({ key: 'PanelName', value: form.panelName })
    await updateSetting({ key: 'SessionTimeout', value: String(form.sessionTimeout) })
    await updateSetting({ key: 'SecurityEntrance', value: form.securityEntrance })
    globalStore.setPanelName(form.panelName)
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { saving.value = false }
}

onMounted(() => {
  fetchVersion()
  fetchSettings()
})

onUnmounted(() => {
  if (logTimer) clearInterval(logTimer)
})
</script>

<style scoped>
.setting-page {
  padding: 0;
}

.setting-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-header-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 500;
}

.update-section {
  margin-top: 20px;
}

.update-url-row {
  display: flex;
  align-items: center;
  margin-bottom: 4px;
}

.update-url-hint {
  margin-bottom: 16px;
  font-size: 12px;
}

.update-result {
  margin-bottom: 16px;
}

.update-card {
  margin-top: 8px;
}

.update-card-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
}

.release-note {
  margin-bottom: 16px;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  max-height: 300px;
  overflow-y: auto;
}

.upgrade-log-section {
  margin-top: 16px;
}

.log-textarea :deep(.el-textarea__inner) {
  font-family: 'Courier New', Courier, monospace;
  font-size: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
}

.mono-text {
  font-family: 'Courier New', Courier, monospace;
}

.dev-notice {
  margin-top: 4px;
}
</style>
