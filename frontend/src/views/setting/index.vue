<template>
  <div class="setting-page xp-settings-page">
    <div class="xp-page-hero">
      <div>
        <div class="xp-page-eyebrow">{{ t('setting.pageEyebrow') }}</div>
        <h2>{{ t('setting.title') }}</h2>
        <p>{{ t('setting.pageDesc') }}</p>
      </div>
      <div class="xp-page-hero-actions">
        <el-tag effect="plain">{{ versionInfo.version || t('setting.dev') }}</el-tag>
        <el-button type="primary" :icon="Refresh" :loading="checking" @click="handleCheckUpdate">
          {{ checking ? t('setting.checking') : t('setting.checkUpdate') }}
        </el-button>
      </div>
    </div>

    <div class="xp-settings-layout">
      <aside class="xp-settings-nav">
        <button
          v-for="section in settingSections"
          :key="section.id"
          type="button"
          class="xp-settings-nav-item"
          @click="scrollToSection(section.id)"
        >
          <el-icon><component :is="section.icon" /></el-icon>
          <span>{{ section.title }}</span>
        </button>
      </aside>

      <main class="xp-settings-content">
    <!-- Card 1: 版本信息 -->
    <el-card id="setting-version" class="setting-card xp-section-card">
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
          <div class="xp-inline-form">
            <el-input v-model="upgradeUrl" :placeholder="t('setting.upgradeUrlPlaceholder')" clearable>
              <template #prepend>{{ t('setting.upgradeUrl') }}</template>
            </el-input>
            <el-button type="primary" :loading="checking" :icon="Refresh" @click="handleCheckUpdate">
              {{ checking ? t('setting.checking') : t('setting.checkUpdate') }}
            </el-button>
          </div>
          <div class="xp-setting-line">
            <span class="xp-setting-line-label">{{ t('setting.autoUpgrade') }}</span>
            <el-switch v-model="autoUpgradeEnabled" @change="handleAutoUpgradeChange" />
            <el-text type="info" size="small">{{ t('setting.autoUpgradeHint') }}</el-text>
          </div>
          <div class="xp-inline-form">
            <el-input v-model="githubToken" :placeholder="t('setting.githubTokenPlaceholder')" clearable show-password>
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

    <!-- Card 2: 外观与个性化 -->
    <el-card id="setting-appearance" class="setting-card xp-section-card">
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
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.themeMode') }}</span>
          <el-radio-group v-model="globalStore.theme" @change="(val: ThemeMode) => globalStore.setTheme(val)">
            <el-radio-button value="dark"><el-icon><Moon /></el-icon> {{ t('header.themeDark') }}</el-radio-button>
            <el-radio-button value="light"><el-icon><Sunny /></el-icon> {{ t('header.themeLight') }}</el-radio-button>
            <el-radio-button value="auto"><el-icon><Monitor /></el-icon> {{ t('header.themeAuto') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 强调色 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('header.accentColor') }}</span>
          <div class="accent-grid-large">
              <button
              v-for="preset in ACCENT_PRESETS"
              :key="preset.key"
                type="button"
              class="accent-swatch-large"
              :class="{ active: globalStore.accentKey === preset.key }"
              :style="{ background: preset.primary }"
                :aria-label="preset.name"
              @click="selectPreset(preset.key)"
            >
              <el-icon v-if="globalStore.accentKey === preset.key" :size="16"><Check /></el-icon>
              </button>
            <div class="accent-swatch-large custom-swatch">
              <input type="color" class="swatch-color-input" :value="globalStore.accentCustom || '#22d3ee'" @input="onCustomAccent" />
            </div>
          </div>
        </div>

        <!-- 背景预设 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.bgPreset') }}</span>
          <div class="accent-grid-large">
            <el-tooltip v-for="bg in BG_PRESETS" :key="bg.key" :content="bgPresetLabel(bg.key)" placement="top">
              <button
                type="button"
                class="bg-swatch"
                :class="{ active: globalStore.bgPreset === bg.key }"
                :style="{ background: bg.preview }"
                :aria-label="bgPresetLabel(bg.key)"
                @click="globalStore.bgPreset = bg.key"
              >
                <el-icon v-if="globalStore.bgPreset === bg.key" :size="14"><Check /></el-icon>
              </button>
            </el-tooltip>
          </div>
        </div>

        <!-- UI 字体 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.uiFont') }}</span>
          <el-select v-model="globalStore.uiFont" class="xp-select-md">
            <el-option v-for="f in FONT_PRESETS" :key="f.key" :label="f.name" :value="f.key" />
          </el-select>
        </div>

        <!-- 密度 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.uiDensity') }}</span>
          <el-radio-group v-model="globalStore.uiDensity">
            <el-radio-button value="compact">{{ t('setting.densityCompact') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.densityDefault') }}</el-radio-button>
            <el-radio-button value="comfortable">{{ t('setting.densityComfortable') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 圆角 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.borderRadius') }}</span>
          <el-radio-group v-model="globalStore.borderRadiusPreset">
            <el-radio-button value="sharp">{{ t('setting.radiusSharp') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.radiusDefault') }}</el-radio-button>
            <el-radio-button value="rounded">{{ t('setting.radiusRounded') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 卡片边框 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.cardBorder') }}</span>
          <el-radio-group v-model="globalStore.cardBorderStyle">
            <el-radio-button v-for="s in cardBorderOptions" :key="s.key" :value="s.key">{{ s.name }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 侧边栏宽度 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.sidebarWidth') }}</span>
          <el-radio-group v-model="globalStore.sidebarWidth">
            <el-radio-button value="narrow">{{ t('setting.sidebarNarrow') }}</el-radio-button>
            <el-radio-button value="default">{{ t('setting.sidebarDefault') }}</el-radio-button>
            <el-radio-button value="wide">{{ t('setting.sidebarWide') }}</el-radio-button>
          </el-radio-group>
        </div>

        <!-- 显示服务器时钟 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.showServerClock') }}</span>
          <el-switch v-model="globalStore.showServerClock" />
        </div>

        <!-- 仪表盘刷新间隔 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.dashboardRefresh') }}</span>
          <el-select v-model="globalStore.dashboardRefreshInterval" class="xp-select-sm">
            <el-option :label="'2 ' + t('setting.seconds')" :value="2000" />
            <el-option :label="'5 ' + t('setting.seconds')" :value="5000" />
            <el-option :label="'10 ' + t('setting.seconds')" :value="10000" />
            <el-option :label="'30 ' + t('setting.seconds')" :value="30000" />
            <el-option :label="t('setting.disableAutoRefresh')" :value="0" />
          </el-select>
        </div>

        <!-- 减弱动画 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.reduceMotion') }}</span>
          <el-switch v-model="globalStore.reduceMotion" />
        </div>

        <el-divider />

        <!-- 终端外观 -->
        <div class="appearance-subtitle">{{ t('setting.terminalAppearance') }}</div>

        <!-- 终端配色 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.termTheme') }}</span>
          <div class="term-theme-grid">
            <button
              v-for="tt in TERMINAL_THEME_PRESETS"
              :key="tt.key"
              type="button"
              class="term-theme-swatch"
              :class="{ active: globalStore.termTheme === tt.key }"
              :aria-label="tt.name"
              @click="globalStore.termTheme = tt.key"
            >
              <div class="term-preview" :style="{ background: tt.theme.background, color: tt.theme.foreground }">
                <span :style="{ color: tt.theme.green }">$</span>
                <span :style="{ color: tt.theme.cyan }"> ls</span>
                <span :style="{ color: tt.theme.yellow }"> -la</span>
              </div>
              <span class="term-theme-name">{{ tt.name }}</span>
            </button>
          </div>
        </div>

        <!-- 终端字体 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.termFont') }}</span>
          <el-select v-model="globalStore.termFont" class="xp-select-md">
            <el-option v-for="f in TERMINAL_FONT_PRESETS" :key="f.key" :label="f.name" :value="f.key" />
          </el-select>
        </div>

        <!-- 终端字号 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.termFontSize') }}</span>
          <el-slider v-model="globalStore.termFontSize" :min="10" :max="24" :step="1" show-input class="xp-slider-md" />
        </div>

        <!-- 终端透明度 -->
        <div class="xp-setting-row">
          <span class="appearance-label">{{ t('setting.termBgOpacity') }}</span>
          <el-slider v-model="globalStore.termBgOpacity" :min="0.3" :max="1" :step="0.05" show-input class="xp-slider-md" />
        </div>
      </div>
    </el-card>

    <!-- Card 3: 面板与安全 -->
    <el-card id="setting-security" class="setting-card xp-section-card" v-loading="loading">
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
              <el-input v-model="form.securityEntrance" :placeholder="t('setting.securityEntrancePlaceholder')" clearable>
                <template #prepend>/</template>
              </el-input>
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

        <el-collapse-item :title="t('setting.agentSetting')" name="agent">
          <el-form label-width="140px" class="xp-form-narrow">
            <el-form-item :label="t('setting.agentToken')">
              <div class="xp-inline-form">
                <el-input v-model="agentTokenForm.token" :placeholder="t('setting.agentTokenPlaceholder')" show-password clearable />
                <el-button @click="generateAgentToken">{{ t('setting.generateToken') }}</el-button>
              </div>
              <div class="xp-form-tip">
                <el-text type="info" size="small">{{ t('setting.agentTokenHint') }}</el-text>
              </div>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="savingAgentToken" @click="handleSaveAgentToken">{{ t('setting.save') }}</el-button>
            </el-form-item>
          </el-form>
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
              <el-input v-model="proxyForm.address" :placeholder="proxyAddressPlaceholder" clearable />
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
      </el-collapse>
    </el-card>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Setting, InfoFilled, Brush, Moon, Sunny, Check } from '@element-plus/icons-vue'
import { getSettingInfo, updateSetting, updatePort, testProxy, getPanelSSL, updatePanelSSL, restartPanel } from '@/api/modules/setting'
import { searchCertificate } from '@/api/modules/ssl'
import { getCurrentVersion, checkUpdate, doUpgrade, getUpgradeLog } from '@/api/modules/upgrade'
import { updatePassword } from '@/api/modules/auth'
import { useGlobalStore, type ThemeMode } from '@/store/modules/global'
import { useI18n } from 'vue-i18n'
import type { UpgradeInfo, Certificate } from '@/api/interface'
import { ACCENT_PRESETS, getPresetByKey, applyAccentPalette, generatePaletteFromHex } from '@/utils/accent-colors'
import { BG_PRESETS, FONT_PRESETS, CARD_BORDER_STYLES } from '@/utils/appearance'
import { TERMINAL_THEME_PRESETS, TERMINAL_FONT_PRESETS } from '@/utils/terminal-theme'

const { t } = useI18n()
const globalStore = useGlobalStore()

const settingSections = computed(() => [
  { id: 'setting-version', title: t('setting.versionAndUpgrade'), icon: 'InfoFilled' },
  { id: 'setting-appearance', title: t('setting.appearance'), icon: 'Brush' },
  { id: 'setting-security', title: t('setting.panelAndSecurity'), icon: 'Setting' },
])

const cardBorderLabelKeys: Record<string, string> = {
  'accent-left': 'setting.cardBorderAccentLeft',
  full: 'setting.cardBorderFull',
  'shadow-only': 'setting.cardBorderShadowOnly',
}

const cardBorderOptions = computed(() =>
  CARD_BORDER_STYLES.map((item) => ({
    ...item,
    name: t(cardBorderLabelKeys[item.key] || 'setting.cardBorderFull'),
  })),
)

const bgPresetLabel = (key: string) => t(`setting.bgPresetNames.${key}`)

const scrollToSection = (id: string) => {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const activeCollapse = ref(['panel', 'panelSsl', 'agent', 'proxy', 'account'])

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

const savingProxy = ref(false)
const testingProxy = ref(false)
const proxyForm = reactive({ type: 'mix', address: '', noProxy: 'localhost,127.0.0.1,::1', enable: false })

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
      upgradeUrl.value = res.data.upgradeUrl || ''
      githubToken.value = res.data.githubToken || ''
      accountForm.userName = res.data.userName || 'admin'
      autoUpgradeEnabled.value = res.data.autoUpgrade === 'enable'
      agentTokenForm.token = res.data.agentToken || ''
      proxyForm.type = res.data.proxyType || 'mix'
      proxyForm.address = res.data.proxyAddress || ''
      proxyForm.noProxy = res.data.proxyNoProxy || 'localhost,127.0.0.1,::1'
      proxyForm.enable = res.data.proxyEnable === 'enable'
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

const handleSaveProxy = async () => {
  savingProxy.value = true
  try {
    await updateSetting({ key: 'ProxyType', value: proxyForm.type })
    await updateSetting({ key: 'ProxyAddress', value: proxyForm.address })
    await updateSetting({ key: 'ProxyNoProxy', value: proxyForm.noProxy })
    await updateSetting({ key: 'ProxyEnable', value: proxyForm.enable ? 'enable' : 'disable' })
    ElMessage.success(t('commons.success'))
    if (proxyForm.enable) {
      ElMessage.info(t('setting.proxyRestartHint'))
    }
  } catch { /* */ } finally { savingProxy.value = false }
}

const handleProxyToggle = async (val: boolean) => {
  if (val && !proxyForm.address.trim()) {
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
