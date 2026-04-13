<template>
  <div class="setting-page">
    <!-- Card 1: 版本信息 -->
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
          <el-tag v-if="versionInfo.version === 'dev'" type="warning" effect="plain">{{ t('setting.dev') }}</el-tag>
          <el-tag v-else type="success" effect="plain">{{ versionInfo.version }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('setting.buildTime')">{{ versionInfo.buildTime || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="t('setting.commitHash')">
          <el-text class="mono-text">{{ versionInfo.commitHash || '-' }}</el-text>
        </el-descriptions-item>
        <el-descriptions-item :label="t('setting.goVersion')">{{ versionInfo.goVersion || '-' }}</el-descriptions-item>
      </el-descriptions>

      <div class="update-section">
        <div v-if="versionInfo.version === 'dev'" class="dev-notice">
          <el-alert :title="t('setting.devTip')" type="info" show-icon :closable="false" />
        </div>
        <template v-else>
          <div class="update-url-row">
            <el-input v-model="upgradeUrl" :placeholder="t('setting.upgradeUrlPlaceholder')" clearable style="flex: 1; margin-right: 12px">
              <template #prepend>{{ t('setting.upgradeUrl') }}</template>
            </el-input>
            <el-button type="primary" :loading="checking" :icon="Refresh" @click="handleCheckUpdate">
              {{ checking ? t('setting.checking') : t('setting.checkUpdate') }}
            </el-button>
          </div>
          <div class="update-url-row" style="margin-top: 8px">
            <span style="margin-right: 12px; white-space: nowrap; color: var(--xp-text-secondary); font-size: 13px;">{{ t('setting.autoUpgrade') }}</span>
            <el-switch v-model="autoUpgradeEnabled" @change="handleAutoUpgradeChange" />
            <el-text type="info" size="small" style="margin-left: 12px">{{ t('setting.autoUpgradeHint') }}</el-text>
          </div>
          <div class="update-url-row" style="margin-top: 8px">
            <el-input v-model="githubToken" :placeholder="t('setting.githubTokenPlaceholder')" clearable show-password style="flex: 1; margin-right: 12px">
              <template #prepend>{{ t('setting.githubToken') }}</template>
            </el-input>
            <el-button :loading="savingToken" @click="handleSaveToken">{{ t('setting.save') }}</el-button>
          </div>
          <div class="update-url-hint">
            <el-text type="info" size="small">{{ t('setting.upgradeUrlHint') }}。{{ t('setting.githubTokenHint') }}</el-text>
          </div>
          <div v-if="upgradeInfo" class="update-result">
            <el-alert v-if="!upgradeInfo.hasUpdate" :title="t('setting.noUpdate')" type="success" show-icon :closable="false" />
            <el-card v-else shadow="hover" class="update-card">
              <div class="update-card-header">
                <el-tag type="danger" effect="dark" size="large">{{ t('setting.hasUpdate') }}: {{ upgradeInfo.latestVersion }}</el-tag>
                <el-text type="info" v-if="upgradeInfo.publishDate">{{ t('setting.publishDate') }}: {{ upgradeInfo.publishDate }}</el-text>
              </div>
              <div v-if="upgradeInfo.releaseNote" class="release-note">
                <el-text tag="p" style="white-space: pre-wrap">{{ upgradeInfo.releaseNote }}</el-text>
              </div>
              <el-button type="danger" :loading="upgrading" size="large" @click="handleUpgrade">
                {{ upgrading ? t('setting.upgrading') : t('setting.doUpgrade') }}
              </el-button>
            </el-card>
          </div>
          <div v-if="upgradeLog" class="upgrade-log-section">
            <el-text tag="div" type="info" style="margin-bottom: 8px">{{ t('setting.upgradeLog') }}</el-text>
            <el-input type="textarea" :model-value="upgradeLog" :rows="8" readonly class="log-textarea" />
          </div>
        </template>
      </div>
    </el-card>

    <!-- Card 2: 外观与个性化 -->
    <el-card class="setting-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Brush /></el-icon>
            <span>{{ t('setting.appearance') }}</span>
          </div>
        </div>
      </template>
      <div class="appearance-section">
        <!-- 主题模式 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.themeMode') }}</span>
          <el-radio-group v-model="globalStore.theme" @change="(val: ThemeMode) => globalStore.setTheme(val)">
            <el-radio-button value="dark"><el-icon><Moon /></el-icon> {{ t('header.themeDark') }}</el-radio-button>
            <el-radio-button value="light"><el-icon><Sunny /></el-icon> {{ t('header.themeLight') }}</el-radio-button>
            <el-radio-button value="auto"><el-icon><Monitor /></el-icon> {{ t('header.themeAuto') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 强调色 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('header.accentColor') }}</span>
          <div class="accent-grid-large">
            <div
              v-for="preset in ACCENT_PRESETS"
              :key="preset.key"
              class="accent-swatch-large"
              :class="{ active: globalStore.accentKey === preset.key }"
              :style="{ background: preset.primary }"
              @click="selectPreset(preset.key)"
            >
              <el-icon v-if="globalStore.accentKey === preset.key" :size="16"><Check /></el-icon>
            </div>
            <div class="accent-swatch-large custom-swatch">
              <input type="color" class="swatch-color-input" :value="globalStore.accentCustom || '#22d3ee'" @input="onCustomAccent" />
            </div>
          </div>
        </div>

        <!-- 背景预设 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.bgPreset') }}</span>
          <div class="accent-grid-large">
            <el-tooltip v-for="bg in BG_PRESETS" :key="bg.key" :content="bg.name" placement="top">
              <div
                class="bg-swatch"
                :class="{ active: globalStore.bgPreset === bg.key }"
                :style="{ background: bg.preview }"
                @click="globalStore.bgPreset = bg.key"
              >
                <el-icon v-if="globalStore.bgPreset === bg.key" :size="14"><Check /></el-icon>
              </div>
            </el-tooltip>
          </div>
        </div>

        <!-- UI 字体 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.uiFont') }}</span>
          <el-select v-model="globalStore.uiFont" style="width: 200px">
            <el-option v-for="f in FONT_PRESETS" :key="f.key" :label="f.name" :value="f.key" />
          </el-select>
        </div>

        <!-- 密度 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.uiDensity') }}</span>
          <el-radio-group v-model="globalStore.uiDensity">
            <el-radio-button value="compact">{{ t('setting.densityCompact') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.densityDefault') }}</el-radio-button>
            <el-radio-button value="comfortable">{{ t('setting.densityComfortable') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 圆角 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.borderRadius') }}</span>
          <el-radio-group v-model="globalStore.borderRadiusPreset">
            <el-radio-button value="sharp">{{ t('setting.radiusSharp') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.radiusDefault') }}</el-radio-button>
            <el-radio-button value="rounded">{{ t('setting.radiusRounded') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 卡片边框 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.cardBorder') }}</span>
          <el-radio-group v-model="globalStore.cardBorderStyle">
            <el-radio-button v-for="s in CARD_BORDER_STYLES" :key="s.key" :value="s.key">{{ s.name }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 侧边栏宽度 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.sidebarWidth') }}</span>
          <el-radio-group v-model="globalStore.sidebarWidth">
            <el-radio-button value="narrow">{{ t('setting.sidebarNarrow') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.sidebarDefault') }}</el-radio-button>
            <el-radio-button value="wide">{{ t('setting.sidebarWide') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 显示服务器时钟 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.showServerClock') }}</span>
          <el-switch v-model="globalStore.showServerClock" />
        </div>

        <!-- 仪表盘刷新间隔 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.dashboardRefresh') }}</span>
          <el-select v-model="globalStore.dashboardRefreshInterval" style="width: 160px">
            <el-option :label="'2 ' + t('setting.seconds')" :value="2000" />
            <el-option :label="'5 ' + t('setting.seconds')" :value="5000" />
            <el-option :label="'10 ' + t('setting.seconds')" :value="10000" />
            <el-option :label="'30 ' + t('setting.seconds')" :value="30000" />
            <el-option :label="t('setting.disableAutoRefresh')" :value="0" />
          </el-select>
        </div>

        <!-- 减弱动画 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.reduceMotion') }}</span>
          <el-switch v-model="globalStore.reduceMotion" />
        </div>

        <el-divider />

        <!-- 终端外观 -->
        <div class="appearance-subtitle">{{ t('setting.terminalAppearance') }}</div>

        <!-- 终端配色 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.termTheme') }}</span>
          <div class="term-theme-grid">
            <div
              v-for="tt in TERMINAL_THEME_PRESETS"
              :key="tt.key"
              class="term-theme-swatch"
              :class="{ active: globalStore.termTheme === tt.key }"
              @click="globalStore.termTheme = tt.key"
            >
              <div class="term-preview" :style="{ background: tt.theme.background, color: tt.theme.foreground }">
                <span :style="{ color: tt.theme.green }">$</span>
                <span :style="{ color: tt.theme.cyan }"> ls</span>
                <span :style="{ color: tt.theme.yellow }"> -la</span>
              </div>
              <span class="term-theme-name">{{ tt.name }}</span>
            </div>
          </div>
        </div>

        <!-- 终端字体 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.termFont') }}</span>
          <el-select v-model="globalStore.termFont" style="width: 200px">
            <el-option v-for="f in TERMINAL_FONT_PRESETS" :key="f.key" :label="f.name" :value="f.key" />
          </el-select>
        </div>

        <!-- 终端字号 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.termFontSize') }}</span>
          <el-slider v-model="globalStore.termFontSize" :min="10" :max="24" :step="1" show-input style="flex: 1; max-width: 300px" />
        </div>

        <!-- 终端透明度 -->
        <div class="appearance-row">
          <span class="appearance-label">{{ t('setting.termBgOpacity') }}</span>
          <el-slider v-model="globalStore.termBgOpacity" :min="0.3" :max="1" :step="0.05" show-input style="flex: 1; max-width: 300px" />
        </div>
      </div>
    </el-card>

    <!-- Card 3: 面板与安全 -->
    <el-card class="setting-card" v-loading="loading">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Setting /></el-icon>
            <span>{{ t('setting.panelAndSecurity') }}</span>
          </div>
        </div>
      </template>
      <el-collapse v-model="activeCollapse">
        <el-collapse-item :title="t('setting.title')" name="panel">
          <el-form :model="form" label-width="140px" style="max-width: 600px">
            <el-form-item :label="t('setting.panelName')">
              <el-input v-model="form.panelName" />
            </el-form-item>
            <el-form-item :label="t('setting.serverPort')">
              <el-input-number v-model="form.port" :min="1" :max="65535" :step="1" />
              <div style="margin-top: 4px">
                <el-text type="info" size="small">{{ t('setting.portChangeHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item :label="t('setting.sessionTimeout')">
              <el-input-number v-model="form.sessionTimeout" :min="3600" :step="3600" />
            </el-form-item>
            <el-form-item :label="t('setting.securityEntrance')">
              <el-input v-model="form.securityEntrance" :placeholder="t('setting.securityEntrancePlaceholder')" clearable>
                <template #prepend>/</template>
              </el-input>
              <div style="margin-top: 4px">
                <el-text type="info" size="small">{{ t('setting.securityEntranceHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving" @click="handleSave">{{ t('setting.save') }}</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>

        <el-collapse-item :title="t('setting.agentSetting')" name="agent">
          <el-form label-width="140px" style="max-width: 600px">
            <el-form-item :label="t('setting.agentToken')">
              <div style="display: flex; gap: 8px; width: 100%">
                <el-input v-model="agentTokenForm.token" :placeholder="t('setting.agentTokenPlaceholder')" show-password clearable style="flex: 1" />
                <el-button @click="generateAgentToken">{{ t('setting.generateToken') }}</el-button>
              </div>
              <div style="margin-top: 4px">
                <el-text type="info" size="small">{{ t('setting.agentTokenHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingAgentToken" @click="handleSaveAgentToken">{{ t('setting.save') }}</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>

        <el-collapse-item :title="t('setting.accountSetting')" name="account">
          <el-form label-width="140px" style="max-width: 600px">
            <el-form-item :label="t('setting.userName')">
              <el-input v-model="accountForm.userName" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingUserName" @click="handleSaveUserName">{{ t('setting.saveUserName') }}</el-button>
            </el-form-item>
            <el-divider />
            <el-form-item :label="t('setting.oldPassword')">
              <el-input v-model="passwordForm.oldPassword" type="password" show-password autocomplete="off" />
            </el-form-item>
            <el-form-item :label="t('setting.newPassword')">
              <el-input v-model="passwordForm.newPassword" type="password" show-password autocomplete="off" />
            </el-form-item>
            <el-form-item :label="t('setting.confirmPassword')">
              <el-input v-model="passwordForm.confirmPassword" type="password" show-password autocomplete="off" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingPassword" @click="handleSavePassword">{{ t('setting.savePassword') }}</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Setting, InfoFilled, User, Brush, Moon, Sunny, Check } from '@element-plus/icons-vue'
import { getSettingInfo, updateSetting, updatePort } from '@/api/modules/setting'
import { getCurrentVersion, checkUpdate, doUpgrade, getUpgradeLog } from '@/api/modules/upgrade'
import { updatePassword } from '@/api/modules/auth'
import { useGlobalStore, type ThemeMode } from '@/store/modules/global'
import { useI18n } from 'vue-i18n'
import type { UpgradeInfo } from '@/api/interface'
import { ACCENT_PRESETS, getPresetByKey, applyAccentPalette, generatePaletteFromHex } from '@/utils/accent-colors'
import { BG_PRESETS, FONT_PRESETS, CARD_BORDER_STYLES } from '@/utils/appearance'
import { TERMINAL_THEME_PRESETS, TERMINAL_FONT_PRESETS } from '@/utils/terminal-theme'

const { t } = useI18n()
const globalStore = useGlobalStore()

const activeCollapse = ref(['panel'])

const selectPreset = (key: string) => {
  globalStore.setAccent(key)
  const preset = getPresetByKey(key)
  if (preset) applyAccentPalette(preset)
}

const onCustomAccent = (e: Event) => {
  const hex = (e.target as HTMLInputElement).value
  globalStore.setAccent('custom', hex)
  applyAccentPalette(generatePaletteFromHex(hex))
}

const loading = ref(false)
const saving = ref(false)
const form = reactive({ panelName: 'X-Panel', port: 7777, sessionTimeout: 86400, securityEntrance: '' })

const savingAgentToken = ref(false)
const agentTokenForm = reactive({ token: '' })

const savingUserName = ref(false)
const savingPassword = ref(false)
const accountForm = reactive({ userName: '' })
const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const versionInfo = reactive({ version: '', commitHash: '', buildTime: '', goVersion: '' })
const upgradeUrl = ref('')
const githubToken = ref('')
const savingToken = ref(false)
const autoUpgradeEnabled = ref(false)
const checking = ref(false)
const upgrading = ref(false)
const upgradeInfo = ref<UpgradeInfo | null>(null)
const upgradeLog = ref('')

const fetchVersion = async () => {
  try {
    const res = await getCurrentVersion()
    if (res.data) Object.assign(versionInfo, res.data)
  } catch { /* */ }
}

const handleCheckUpdate = async () => {
  checking.value = true
  upgradeInfo.value = null
  try {
    const res = await checkUpdate({ releaseUrl: upgradeUrl.value || undefined })
    if (res.data) upgradeInfo.value = res.data
  } catch { /* */ } finally { checking.value = false }
}

const handleUpgrade = async () => {
  if (!upgradeInfo.value) return
  try {
    await ElMessageBox.confirm(
      t('setting.upgradeConfirm', { version: upgradeInfo.value.latestVersion }),
      t('commons.tip'),
      { type: 'warning', confirmButtonText: t('commons.confirm'), cancelButtonText: t('commons.cancel') },
    )
  } catch { return }

  upgrading.value = true
  try {
    await doUpgrade({
      version: upgradeInfo.value.latestVersion,
      downloadUrl: upgradeInfo.value.downloadUrl,
      checksumUrl: upgradeInfo.value.checksumUrl || undefined,
    })
    ElMessage.success(t('setting.upgradeStarted'))
    pollUpgradeLog()
  } catch {
    ElMessage.error(t('setting.upgradeFailed'))
    upgrading.value = false
  }
}

let logTimer: ReturnType<typeof setInterval> | null = null
const pollUpgradeLog = () => {
  if (logTimer) clearInterval(logTimer)
  logTimer = setInterval(async () => {
    try {
      const res = await getUpgradeLog()
      if (res.data) upgradeLog.value = res.data
    } catch {
      if (logTimer) clearInterval(logTimer)
      upgrading.value = false
      setTimeout(() => window.location.reload(), 3000)
    }
  }, 2000)
}

const handleAutoUpgradeChange = async (val: boolean) => {
  try {
    await updateSetting({ key: 'AutoUpgrade', value: val ? 'enable' : 'disable' })
    ElMessage.success(t('commons.success'))
  } catch { autoUpgradeEnabled.value = !val }
}

const handleSaveToken = async () => {
  savingToken.value = true
  try {
    await updateSetting({ key: 'GitHubToken', value: githubToken.value })
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { savingToken.value = false }
}

const fetchSettings = async () => {
  loading.value = true
  try {
    const res = await getSettingInfo()
    if (res.data) {
      form.panelName = res.data.panelName || 'X-Panel'
      form.port = parseInt(res.data.serverPort) || 7777
      form.sessionTimeout = parseInt(res.data.sessionTimeout) || 86400
      form.securityEntrance = res.data.securityEntrance || ''
      githubToken.value = res.data.githubToken || ''
      accountForm.userName = res.data.userName || 'admin'
      autoUpgradeEnabled.value = res.data.autoUpgrade === 'enable'
      agentTokenForm.token = res.data.agentToken || ''
    }
  } catch { /* */ } finally { loading.value = false }
}

const handleSave = async () => {
  saving.value = true
  try {
    await updateSetting({ key: 'PanelName', value: form.panelName })
    await updatePort({ port: String(form.port) })
    await updateSetting({ key: 'SessionTimeout', value: String(form.sessionTimeout) })
    await updateSetting({ key: 'SecurityEntrance', value: form.securityEntrance })
    globalStore.setPanelName(form.panelName)
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { saving.value = false }
}

const generateAgentToken = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let token = ''
  for (let i = 0; i < 32; i++) token += chars.charAt(Math.floor(Math.random() * chars.length))
  agentTokenForm.token = token
}

const handleSaveAgentToken = async () => {
  savingAgentToken.value = true
  try {
    await updateSetting({ key: 'AgentToken', value: agentTokenForm.token })
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { savingAgentToken.value = false }
}

const handleSaveUserName = async () => {
  if (!accountForm.userName.trim()) { ElMessage.warning(t('setting.userNameRequired')); return }
  savingUserName.value = true
  try {
    await updateSetting({ key: 'UserName', value: accountForm.userName })
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { savingUserName.value = false }
}

const handleSavePassword = async () => {
  if (!passwordForm.oldPassword || !passwordForm.newPassword) { ElMessage.warning(t('setting.passwordRequired')); return }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) { ElMessage.warning(t('init.passwordMismatch')); return }
  if (passwordForm.newPassword.length < 6) { ElMessage.warning(t('init.passwordMinLength')); return }
  savingPassword.value = true
  try {
    await updatePassword({ oldPassword: passwordForm.oldPassword, newPassword: passwordForm.newPassword })
    ElMessage.success(t('setting.passwordChangedSuccess'))
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
  } catch { /* */ } finally { savingPassword.value = false }
}

onMounted(() => { fetchVersion(); fetchSettings() })
onUnmounted(() => { if (logTimer) clearInterval(logTimer) })
</script>

<style scoped lang="scss">
.setting-page { padding: 0; }
.setting-card { margin-bottom: 20px; }

.card-header { display: flex; align-items: center; justify-content: space-between; }
.card-header-title { display: flex; align-items: center; gap: 8px; font-size: 16px; font-weight: 500; }

.update-section { margin-top: 20px; }
.update-url-row { display: flex; align-items: center; margin-bottom: 4px; }
.update-url-hint { margin-bottom: 16px; font-size: 12px; }
.update-result { margin-bottom: 16px; }
.update-card { margin-top: 8px; }
.update-card-header { display: flex; align-items: center; gap: 16px; margin-bottom: 12px; }
.release-note { margin-bottom: 16px; padding: 12px; background: var(--el-fill-color-light); border-radius: 4px; max-height: 300px; overflow-y: auto; }
.upgrade-log-section { margin-top: 16px; }
.log-textarea :deep(.el-textarea__inner) { font-family: var(--xp-font-mono); font-size: 12px; background: var(--xp-bg-inset); color: var(--xp-text-primary); }
.dev-notice { margin-top: 4px; }
.mono-text { font-family: var(--xp-font-mono); font-size: 13px; }

.appearance-section { display: flex; flex-direction: column; gap: 20px; }
.appearance-row { display: flex; align-items: flex-start; gap: 16px; }
.appearance-label { font-size: 14px; color: var(--xp-text-secondary); min-width: 100px; padding-top: 6px; font-weight: 500; flex-shrink: 0; }
.appearance-subtitle { font-size: 14px; font-weight: 600; color: var(--xp-text-primary); padding-bottom: 4px; }

.accent-grid-large { display: flex; flex-wrap: wrap; gap: 10px; }

.accent-swatch-large {
  width: 36px; height: 36px; border-radius: 10px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  color: #fff; transition: all 0.2s; border: 2px solid transparent;
  &:hover { transform: scale(1.1); box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2); }
  &.active { border-color: var(--xp-text-primary); box-shadow: 0 0 0 2px var(--xp-bg-surface), 0 0 0 4px currentColor; }
  &.custom-swatch { border: 2px dashed var(--xp-border-hover); background: transparent !important; overflow: hidden; padding: 0; }
}

.swatch-color-input {
  width: 100%; height: 100%; border: none; padding: 0; background: transparent; cursor: pointer;
  &::-webkit-color-swatch-wrapper { padding: 0; }
  &::-webkit-color-swatch { border: none; border-radius: 8px; }
}

.bg-swatch {
  width: 48px; height: 32px; border-radius: 8px; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  color: rgba(255,255,255,0.7); transition: all 0.2s; border: 2px solid transparent;
  &:hover { transform: scale(1.05); box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3); }
  &.active { border-color: var(--xp-accent); box-shadow: 0 0 0 2px var(--xp-bg-surface), 0 0 0 3px var(--xp-accent); }
}

.term-theme-grid { display: flex; flex-wrap: wrap; gap: 10px; }

.term-theme-swatch {
  cursor: pointer; border-radius: 8px; border: 2px solid transparent;
  overflow: hidden; transition: all 0.2s; width: 120px;
  &:hover { border-color: var(--xp-border-hover); }
  &.active { border-color: var(--xp-accent); box-shadow: 0 0 0 1px var(--xp-accent); }
}

.term-preview {
  padding: 6px 10px; font-family: var(--xp-font-mono); font-size: 11px;
  white-space: nowrap; line-height: 1.4;
}

.term-theme-name {
  display: block; text-align: center; font-size: 11px; padding: 4px 0;
  color: var(--xp-text-secondary); background: var(--xp-bg-inset);
}
</style>
