<template>
  <div class="setting-page xp-settings-page">
    <div class="settings-heading">
      <h2>{{ t('setting.title') }}</h2>
      <p>{{ t('setting.pageDesc') }}</p>
    </div>

    <div class="xp-settings-layout">
      <aside class="xp-settings-nav">
        <button
          v-for="section in settingSections"
          :key="section.id"
          type="button"
          class="xp-settings-nav-item"
          :class="{ active: activeSection === section.id }"
          @click="activeSection = section.id"
        >
          <el-icon><component :is="section.icon" /></el-icon>
          <span>{{ section.title }}</span>
        </button>
      </aside>

      <main class="xp-settings-content">
    <el-card v-show="activeSection === 'update'" id="setting-version" class="setting-card xp-section-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><InfoFilled /></el-icon>
            <span>{{ t('setting.versionAndUpgrade') }}</span>
          </div>
        </div>
      </template>
      <div class="version-compact">
        <div>
          <div class="version-name">{{ versionInfo.version || t('setting.dev') }}</div>
          <div class="version-meta">
            {{ [versionInfo.commitHash, versionInfo.buildTime].filter(Boolean).join(' · ') || t('setting.buildDetails') }}
          </div>
        </div>
        <div class="version-actions">
          <div class="xp-setting-line">
            <span class="xp-setting-line-label">{{ t('setting.autoUpgrade') }}</span>
            <el-switch v-model="autoUpgradeEnabled" @change="handleAutoUpgradeChange" />
          </div>
          <el-button type="primary" :icon="Refresh" :loading="checking" @click="handleCheckUpdate">
            {{ checking ? t('setting.checking') : t('setting.checkUpdate') }}
          </el-button>
        </div>
      </div>
      <el-text type="info" size="small">{{ t('setting.autoUpgradeHint') }}</el-text>

      <div class="update-section">
        <div v-if="versionInfo.version === 'dev'" class="dev-notice">
          <el-alert :title="t('setting.devTip')" type="info" show-icon :closable="false" />
        </div>
        <template v-else>
          <el-collapse>
            <el-collapse-item :title="t('setting.advancedUpdate')" name="advanced-update">
              <div class="xp-inline-form">
                <el-input v-model="upgradeUrl" :placeholder="t('setting.upgradeUrlPlaceholder')" clearable>
                  <template #prepend>{{ t('setting.upgradeUrl') }}</template>
                </el-input>
              </div>
              <p class="setting-inline-hint">{{ t('setting.upgradeUrlHint') }}</p>
              <div class="xp-inline-form">
                <el-input v-model="githubToken" :placeholder="githubTokenSet ? t('setting.secretConfiguredPlaceholder') : t('setting.githubTokenPlaceholder')" clearable show-password>
                  <template #prepend>{{ t('setting.githubToken') }}</template>
                </el-input>
                <span v-if="githubTokenSet" class="xp-secret-row">{{ t('setting.secretConfigured') }}</span>
                <el-button :loading="savingToken" :disabled="!githubToken" @click="handleSaveToken">{{ t('setting.save') }}</el-button>
              </div>
              <p class="setting-inline-hint">{{ t('setting.githubTokenHint') }}</p>
            </el-collapse-item>
          </el-collapse>
          <div v-if="upgradeInfo" class="update-result">
            <el-alert v-if="!upgradeInfo.hasUpdate" :title="t('setting.noUpdate')" type="success" show-icon :closable="false" />
            <el-card v-else shadow="hover" class="update-card">
              <div class="update-card-header">
                <el-tag type="danger" effect="dark" size="large">{{ t('setting.hasUpdate') }}: {{ upgradeInfo.latestVersion }}</el-tag>
                <el-text type="info" v-if="upgradeInfo.publishDate">{{ t('setting.publishDate') }}: {{ upgradeInfo.publishDate }}</el-text>
              </div>
              <div v-if="upgradeInfo.releaseNote" class="release-note">
                <el-text tag="p" class="xp-pre-wrap">{{ upgradeInfo.releaseNote }}</el-text>
              </div>
              <el-button type="danger" :loading="upgrading" size="large" @click="handleUpgrade">
                {{ upgrading ? t('setting.upgrading') : t('setting.doUpgrade') }}
              </el-button>
            </el-card>
          </div>
          <div v-if="upgradeLog" class="upgrade-log-section">
            <el-text tag="div" type="info" class="xp-mb-8">{{ t('setting.upgradeLog') }}</el-text>
            <el-input type="textarea" :model-value="upgradeLog" :rows="8" readonly class="log-textarea" />
          </div>
        </template>
      </div>
    </el-card>

    <el-card v-show="activeSection === 'appearance'" id="setting-appearance" class="setting-card xp-section-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Brush /></el-icon>
            <span>{{ t('setting.appearance') }}</span>
          </div>
        </div>
      </template>
      <AppearancePanel />
    </el-card>

    <el-card v-show="activeSection === 'panel'" id="setting-security" class="setting-card xp-section-card" v-loading="loading">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Setting /></el-icon>
            <span>{{ t('setting.panelSection') }}</span>
          </div>
        </div>
      </template>
      <el-collapse v-model="panelCollapse">
        <el-collapse-item :title="t('setting.panelSection')" name="panel">
          <el-form :model="form" label-width="140px" class="xp-form-narrow">
            <el-form-item :label="t('setting.panelName')">
              <el-input v-model="form.panelName" />
            </el-form-item>
            <el-form-item :label="t('setting.serverPort')">
              <el-input-number v-model="form.port" :min="1" :max="65535" :step="1" />
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ t('setting.portChangeHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item :label="t('setting.sessionTimeout')">
              <el-input-number v-model="form.sessionTimeout" :min="3600" :step="3600" />
            </el-form-item>
            <el-form-item :label="t('setting.securityEntrance')">
              <el-input v-model="form.securityEntrance" :placeholder="securityEntranceSet ? t('setting.secretConfiguredPlaceholder') : t('setting.securityEntrancePlaceholder')" clearable>
                <template #prepend>/</template>
              </el-input>
              <div v-if="securityEntranceSet" class="xp-secret-row">
                <span>{{ t('setting.secretConfigured') }}</span>
                <button type="button" class="xp-secret-clear" @click="clearSettingSecret('SecurityEntrance')">{{ t('setting.clearSecret') }}</button>
              </div>
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ t('setting.securityEntranceHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="saving" @click="handleSave">{{ t('setting.save') }}</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>

        <el-collapse-item :title="t('setting.panelHttpsCert')" name="panelSsl">
          <div v-loading="loadingPanelSSL">
            <el-alert type="info" :closable="false" class="xp-mb-12">
              {{ t('setting.panelHttpsCertHint') }}
            </el-alert>
            <el-descriptions :column="1" border size="small" class="xp-desc-panel">
              <el-descriptions-item :label="t('setting.panelHttpsEnabled')">
                {{ panelSSLInfo.enable ? t('setting.panelHttpsOn') : t('setting.panelHttpsOff') }}
              </el-descriptions-item>
              <el-descriptions-item v-if="panelSSLInfo.primaryDomain" :label="t('setting.panelHttpsBoundDomain')">
                {{ panelSSLInfo.primaryDomain }}
              </el-descriptions-item>
              <el-descriptions-item :label="t('setting.panelHttpsCertPath')">
                <el-text class="mono-text" size="small">{{ panelSSLInfo.certPath || '—' }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item :label="t('setting.panelHttpsKeyPath')">
                <el-text class="mono-text" size="small">{{ panelSSLInfo.keyPath || '—' }}</el-text>
              </el-descriptions-item>
            </el-descriptions>
            <el-form label-width="160px" class="xp-form-medium">
              <el-form-item :label="t('setting.panelHttpsSelectCert')">
                <el-select
                  v-model="panelSSLCertSelect"
                  filterable
                  clearable
                  :placeholder="t('setting.panelHttpsSelectCert')"
                  class="xp-input-wide"
                >
                  <el-option
                    v-for="c in readyCerts"
                    :key="c.id"
                    :label="`${c.primaryDomain} (ID ${c.id})`"
                    :value="c.id"
                  />
                </el-select>
                <div v-if="readyCerts.length === 0" class="xp-form-tip">
                  <el-text type="warning" size="small">{{ t('setting.panelHttpsNoReadyCert') }}</el-text>
                </div>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="savingPanelSSL" @click="handleSavePanelSSL">{{ t('setting.save') }}</el-button>
                <el-button @click="handleRestartPanelForSsl">{{ t('home.restartPanel') }}</el-button>
              </el-form-item>
              <el-text type="info" size="small">{{ t('setting.panelHttpsRestartHint') }}</el-text>
            </el-form>
          </div>
        </el-collapse-item>

        <el-collapse-item :title="t('setting.proxy')" name="proxy">
          <el-form label-width="140px" class="xp-form-medium">
            <el-form-item :label="t('setting.proxyType')">
              <el-radio-group v-model="proxyForm.type">
                <el-radio-button value="mix">
                  {{ t('setting.proxyTypeMix') }}
                </el-radio-button>
                <el-radio-button value="http">
                  {{ t('setting.proxyTypeHttp') }}
                </el-radio-button>
                <el-radio-button value="socks5">
                  {{ t('setting.proxyTypeSocks5') }}
                </el-radio-button>
              </el-radio-group>
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ proxyTypeDesc }}</el-text>
              </div>
            </el-form-item>
            <el-form-item :label="t('setting.proxyAddress')">
              <el-input v-model="proxyForm.address" :placeholder="proxyAddressSet ? t('setting.secretConfiguredPlaceholder') : proxyAddressPlaceholder" clearable />
              <div v-if="proxyAddressSet" class="xp-secret-row">
                <span>{{ t('setting.secretConfigured') }}</span>
                <button type="button" class="xp-secret-clear" @click="clearSettingSecret('ProxyAddress')">{{ t('setting.clearSecret') }}</button>
              </div>
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ proxyAddressHint }}</el-text>
              </div>
            </el-form-item>
            <el-form-item :label="t('setting.proxyNoProxy')">
              <el-input v-model="proxyForm.noProxy" placeholder="localhost,127.0.0.1,::1" clearable />
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ t('setting.proxyNoProxyHint') }}</el-text>
              </div>
            </el-form-item>
            <el-alert
              v-if="proxyForm.type === 'socks5'"
              type="warning"
              :title="t('setting.proxySocks5Warning')"
              show-icon
              :closable="false"
              class="xp-mb-16"
            />
            <el-form-item :label="t('setting.proxyEnable')">
              <div class="xp-setting-line">
                <el-switch v-model="proxyForm.enable" :loading="savingProxy" @change="handleProxyToggle" />
                <el-button :loading="testingProxy" :disabled="!proxyForm.address" @click="handleTestProxy">
                  {{ testingProxy ? t('setting.proxyTesting') : t('setting.proxyTest') }}
                </el-button>
              </div>
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ t('setting.proxyHint') }}</el-text>
              </div>
              <div class="xp-form-tip">
                <el-text type="info" size="small">
                  {{ proxyForm.type === 'socks5' ? t('setting.proxyCoverageSocks5') : t('setting.proxyCoverage') }}
                </el-text>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingProxy" @click="handleSaveProxy">{{ t('setting.save') }}</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </el-card>

    <el-card v-show="activeSection === 'account'" id="setting-account" class="setting-card xp-section-card" v-loading="loading">
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><User /></el-icon>
            <span>{{ t('setting.accountAndSecurity') }}</span>
          </div>
        </div>
      </template>
      <el-collapse v-model="accountCollapse">
        <el-collapse-item :title="t('setting.accountSetting')" name="account">
          <el-form label-width="140px" class="xp-form-narrow">
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
        <el-collapse-item :title="t('setting.agentSetting')" name="agent">
          <el-form label-width="140px" class="xp-form-narrow">
            <el-form-item :label="t('setting.agentToken')">
              <div class="xp-inline-form">
                <el-input v-model="agentTokenForm.token" :placeholder="agentTokenSet ? t('setting.secretConfiguredPlaceholder') : t('setting.agentTokenPlaceholder')" show-password clearable />
                <span v-if="agentTokenSet" class="xp-secret-row">{{ t('setting.secretConfigured') }}</span>
                <el-button @click="generateAgentToken">{{ t('setting.generateToken') }}</el-button>
              </div>
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ t('setting.agentTokenHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingAgentToken" :disabled="!agentTokenForm.token" @click="handleSaveAgentToken">{{ t('setting.save') }}</el-button>
            </el-form-item>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </el-card>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, markRaw } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Setting, InfoFilled, Brush, User } from '@element-plus/icons-vue'
import { getSettingInfo, updateSetting, updatePort, testProxy, getPanelSSL, updatePanelSSL, restartPanel } from '@/api/modules/setting'
import { searchCertificate } from '@/api/modules/ssl'
import { getCurrentVersion, checkUpdate, doUpgrade, getUpgradeLog } from '@/api/modules/upgrade'
import { updatePassword } from '@/api/modules/auth'
import { useGlobalStore } from '@/store/modules/global'
import { useI18n } from 'vue-i18n'
import type { UpgradeInfo, Certificate } from '@/api/interface'
import AppearancePanel from './appearance-panel.vue'

const { t } = useI18n()
const globalStore = useGlobalStore()

const activeSection = ref('appearance')
const settingSections = computed(() => [
  { id: 'appearance', title: t('setting.appearance'), icon: markRaw(Brush) },
  { id: 'panel', title: t('setting.panelSection'), icon: markRaw(Setting) },
  { id: 'account', title: t('setting.accountAndSecurity'), icon: markRaw(User) },
  { id: 'update', title: t('setting.versionAndUpgrade'), icon: markRaw(InfoFilled) },
])

const panelCollapse = ref(['panel'])
const accountCollapse = ref(['account'])

const panelSSLInfo = reactive({
  enable: false,
  certPath: '',
  keyPath: '',
  certificateId: 0,
  primaryDomain: '',
})
const readyCerts = ref<Certificate[]>([])
const panelSSLCertSelect = ref<number | undefined>(undefined)
const loadingPanelSSL = ref(false)
const savingPanelSSL = ref(false)

const fetchPanelSSL = async () => {
  loadingPanelSSL.value = true
  try {
    const res = await getPanelSSL() as { data?: typeof panelSSLInfo }
    if (res.data) {
      Object.assign(panelSSLInfo, res.data)
      panelSSLCertSelect.value = res.data.certificateId ? res.data.certificateId : undefined
    }
  } catch {
    /* ignore */
  } finally {
    loadingPanelSSL.value = false
  }
}

const fetchReadyCertificates = async () => {
  try {
    const res = await searchCertificate({ page: 1, pageSize: 100, info: '' })
    const items = (res as { data?: { items?: Certificate[] } }).data?.items ?? []
    // ready：已创建未签发；applied：ACME/上传/证书同步成功后落盘
    readyCerts.value = items.filter((c) => c.status === 'ready' || c.status === 'applied')
  } catch {
    readyCerts.value = []
  }
}

const handleSavePanelSSL = async () => {
  if (!panelSSLCertSelect.value) {
    ElMessage.warning(t('setting.panelHttpsSelectRequired'))
    return
  }
  savingPanelSSL.value = true
  try {
    await updatePanelSSL({ certificateId: panelSSLCertSelect.value })
    ElMessage.success(t('setting.panelHttpsSaveSuccess'))
    await fetchPanelSSL()
  } catch {
    /* ElMessage from http */
  } finally {
    savingPanelSSL.value = false
  }
}

const handleRestartPanelForSsl = async () => {
  try {
    await ElMessageBox.confirm(t('home.restartPanelConfirm'), t('commons.tip'), {
      type: 'warning',
      confirmButtonText: t('commons.confirm'),
      cancelButtonText: t('commons.cancel'),
    })
  } catch {
    return
  }
  try {
    await restartPanel()
    ElMessage.success(t('home.restartPanelSuccess'))
  } catch {
    /* ignore */
  }
}

const loading = ref(false)
const saving = ref(false)
const form = reactive({ panelName: globalStore.panelName || 'X-Panel', port: 7777, sessionTimeout: 86400, securityEntrance: '' })
const securityEntranceSet = ref(false)

const savingAgentToken = ref(false)
const agentTokenForm = reactive({ token: '' })
const agentTokenSet = ref(false)

const savingProxy = ref(false)
const testingProxy = ref(false)
const proxyForm = reactive({ type: 'mix', address: '', noProxy: 'localhost,127.0.0.1,::1', enable: false })
const proxyAddressSet = ref(false)

const proxyTypeDesc = computed(() => {
  const map: Record<string, string> = {
    mix: t('setting.proxyTypeMixDesc'),
    http: t('setting.proxyTypeHttpDesc'),
    socks5: t('setting.proxyTypeSocks5Desc'),
  }
  return map[proxyForm.type] || ''
})

const proxyAddressPlaceholder = computed(() => {
  const map: Record<string, string> = {
    mix: t('setting.proxyAddressPlaceholderMix'),
    http: t('setting.proxyAddressPlaceholderHttp'),
    socks5: t('setting.proxyAddressPlaceholderSocks5'),
  }
  return map[proxyForm.type] || ''
})

const proxyAddressHint = computed(() => {
  const map: Record<string, string> = {
    mix: t('setting.proxyAddressHintMix'),
    http: t('setting.proxyAddressHintHttp'),
    socks5: t('setting.proxyAddressHintSocks5'),
  }
  return map[proxyForm.type] || ''
})

const savingUserName = ref(false)
const savingPassword = ref(false)
const accountForm = reactive({ userName: '' })
const passwordForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })

const versionInfo = reactive({ version: '', commitHash: '', buildTime: '', goVersion: '' })
const upgradeUrl = ref('')
const githubToken = ref('')
const githubTokenSet = ref(false)
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
    const customURL = upgradeUrl.value.trim()
    if (customURL) {
      await updateSetting({ key: 'UpgradeURL', value: customURL })
    }
    const res = await checkUpdate({ releaseUrl: customURL || undefined })
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
    githubToken.value = ''
    githubTokenSet.value = true
    ElMessage.success(t('commons.success'))
  } catch { /* */ } finally { savingToken.value = false }
}

const fetchSettings = async () => {
  loading.value = true
  try {
    const res = await getSettingInfo()
    if (res.data) {
      if (res.data.panelName) form.panelName = res.data.panelName
      form.port = parseInt(res.data.serverPort) || 7777
      form.sessionTimeout = parseInt(res.data.sessionTimeout) || 86400
      form.securityEntrance = ''
      securityEntranceSet.value = res.data.securityEntranceSet === true
      upgradeUrl.value = res.data.upgradeUrl || ''
      githubToken.value = ''
      githubTokenSet.value = res.data.githubTokenSet === true
      accountForm.userName = res.data.userName || 'admin'
      autoUpgradeEnabled.value = res.data.autoUpgrade === 'enable'
      agentTokenForm.token = ''
      agentTokenSet.value = res.data.agentTokenSet === true
      proxyForm.type = res.data.proxyType || 'mix'
      proxyForm.address = ''
      proxyAddressSet.value = res.data.proxyAddressSet === true
      proxyForm.noProxy = res.data.proxyNoProxy || 'localhost,127.0.0.1,::1'
      proxyForm.enable = res.data.proxyEnable === 'enable'
    }
  } catch { /* */ } finally { loading.value = false }
}

const handleSave = async () => {
  const panelName = form.panelName.trim()
  if (!panelName) {
    ElMessage.warning(t('setting.panelNameRequired'))
    return
  }
  saving.value = true
  try {
    await updateSetting({ key: 'PanelName', value: panelName })
    await updatePort({ port: String(form.port) })
    await updateSetting({ key: 'SessionTimeout', value: String(form.sessionTimeout) })
    await updateSetting({ key: 'SecurityEntrance', value: form.securityEntrance })
    if (form.securityEntrance) {
      securityEntranceSet.value = true
      form.securityEntrance = ''
    }
    form.panelName = panelName
    globalStore.setPanelName(panelName)
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
    agentTokenForm.token = ''
    agentTokenSet.value = true
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

const handleSaveProxy = async () => {
  savingProxy.value = true
  try {
    await updateSetting({ key: 'ProxyType', value: proxyForm.type })
    await updateSetting({ key: 'ProxyAddress', value: proxyForm.address })
    if (proxyForm.address) {
      proxyAddressSet.value = true
      proxyForm.address = ''
    }
    await updateSetting({ key: 'ProxyNoProxy', value: proxyForm.noProxy })
    await updateSetting({ key: 'ProxyEnable', value: proxyForm.enable ? 'enable' : 'disable' })
    ElMessage.success(t('commons.success'))
    if (proxyForm.enable) {
      ElMessage.info(t('setting.proxyRestartHint'))
    }
  } catch { /* */ } finally { savingProxy.value = false }
}

const handleProxyToggle = async (val: boolean) => {
  if (val && !proxyForm.address.trim() && !proxyAddressSet.value) {
    proxyForm.enable = false
    ElMessage.warning(proxyAddressPlaceholder.value)
    return
  }
  savingProxy.value = true
  try {
    if (val) {
      await updateSetting({ key: 'ProxyType', value: proxyForm.type })
      await updateSetting({ key: 'ProxyAddress', value: proxyForm.address })
      await updateSetting({ key: 'ProxyNoProxy', value: proxyForm.noProxy })
    }
    await updateSetting({ key: 'ProxyEnable', value: val ? 'enable' : 'disable' })
    ElMessage.success(t('commons.success'))
    ElMessage.info(t('setting.proxyRestartHint'))
  } catch { proxyForm.enable = !val } finally { savingProxy.value = false }
}

const clearSettingSecret = async (key: 'SecurityEntrance' | 'ProxyAddress') => {
  await updateSetting({ key, value: '', clear: true })
  if (key === 'SecurityEntrance') {
    securityEntranceSet.value = false
    form.securityEntrance = ''
  } else {
    proxyAddressSet.value = false
    proxyForm.address = ''
  }
  ElMessage.success(t('commons.success'))
}

const handleTestProxy = async () => {
  testingProxy.value = true
  try {
    await testProxy({ address: proxyForm.address })
    ElMessage.success(t('setting.proxyTestSuccess'))
  } catch {
    ElMessage.error(t('setting.proxyTestFail'))
  } finally { testingProxy.value = false }
}

onMounted(() => {
  fetchVersion()
  fetchSettings()
  fetchPanelSSL()
  fetchReadyCertificates()
})
onUnmounted(() => { if (logTimer) clearInterval(logTimer) })
</script>
