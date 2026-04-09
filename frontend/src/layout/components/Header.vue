<template>
  <div class="header">
    <div class="header-left">
      <div class="collapse-btn" @click="globalStore.toggleMenuCollapse">
        <el-icon :size="18">
          <Fold v-if="!globalStore.menuCollapse" />
          <Expand v-else />
        </el-icon>
      </div>
      <!-- 服务器信息 -->
      <div class="server-info" v-if="globalStore.serverInfo">
        <div class="server-identity">
          <el-icon :size="16" color="var(--xp-accent)"><Monitor /></el-icon>
          <span class="server-hostname">{{ globalStore.serverInfo.hostname }}</span>
        </div>
        <el-tag size="small" effect="dark" round>{{ globalStore.version || '...' }}</el-tag>
        <el-tag size="small" effect="plain" round type="info">
          {{ globalStore.serverInfo.platform }} {{ globalStore.serverInfo.platformVersion }}
        </el-tag>
        <el-tag size="small" effect="plain" round type="info">
          {{ globalStore.serverInfo.kernelArch }}
        </el-tag>
        <el-tag v-if="globalStore.serverInfo.virtualization" size="small" effect="plain" round type="warning">
          {{ globalStore.serverInfo.virtualization }}
        </el-tag>
        <div class="server-uptime">
          <el-icon :size="12"><Clock /></el-icon>
          <span>{{ t('home.uptime') }}: {{ formatUptime(globalStore.serverInfo.uptime) }}</span>
        </div>
        <el-tooltip v-if="serverClock" :content="globalStore.serverInfo.timezone" placement="bottom">
          <div class="server-clock">
            <el-icon :size="12"><Timer /></el-icon>
            <span>{{ serverClock }}</span>
          </div>
        </el-tooltip>
        <el-button-group size="small" class="server-actions">
          <el-button type="warning" text size="small" @click="handleRestartPanel">
            <el-icon><RefreshRight /></el-icon>{{ t('home.restartPanel') }}
          </el-button>
          <el-button type="danger" text size="small" @click="handleRebootServer">
            <el-icon><SwitchButton /></el-icon>{{ t('home.rebootServer') }}
          </el-button>
        </el-button-group>
      </div>
    </div>
    <div class="header-right">
      <el-select
        v-model="currentNode"
        size="small"
        style="width: 160px; margin-right: 4px"
        @change="onNodeChange"
      >
        <el-option :label="t('node.local')" :value="0" />
        <el-option v-for="n in nodes" :key="n.id" :label="n.name" :value="n.id" />
      </el-select>

      <!-- 主题色选择 -->
      <el-popover placement="bottom" :width="240" trigger="click" :show-arrow="true">
        <template #reference>
          <div class="theme-btn">
            <div class="accent-dot" :style="{ background: currentAccentColor }"></div>
          </div>
        </template>
        <div class="accent-panel">
          <div class="accent-section">
            <div class="accent-panel-title">{{ t('header.accentColor') }}</div>
            <div class="accent-grid">
              <div
                v-for="preset in ACCENT_PRESETS"
                :key="preset.key"
                class="accent-swatch"
                :class="{ active: globalStore.accentKey === preset.key }"
                :style="{ background: preset.primary }"
                :title="preset.name"
                @click="selectAccent(preset.key)"
              >
                <el-icon v-if="globalStore.accentKey === preset.key" :size="12"><Check /></el-icon>
              </div>
            </div>
          </div>
          <div class="accent-custom-row">
            <span class="accent-custom-label">{{ t('header.customColor') }}</span>
            <input
              type="color"
              class="accent-color-input"
              :value="globalStore.accentCustom || '#22d3ee'"
              @input="onCustomColor"
            />
          </div>
        </div>
      </el-popover>

      <!-- 深浅模式切换 -->
      <el-tooltip :content="themeLabel" placement="bottom">
        <div class="theme-btn" @click="globalStore.cycleTheme()">
          <el-icon :size="16">
            <Moon v-if="globalStore.theme === 'dark'" />
            <Sunny v-else-if="globalStore.theme === 'light'" />
            <Monitor v-else />
          </el-icon>
        </div>
      </el-tooltip>

      <el-dropdown @command="handleCommand" trigger="click">
        <div class="user-dropdown">
          <div class="user-avatar">
            <el-icon :size="14"><UserFilled /></el-icon>
          </div>
          <span class="username">{{ userStore.name || 'admin' }}</span>
          <el-icon :size="12" class="arrow"><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="password">
              <el-icon><Lock /></el-icon>{{ t('header.changePassword') }}
            </el-dropdown-item>
            <el-dropdown-item command="logout" divided>
              <el-icon><SwitchButton /></el-icon>{{ t('header.logout') }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { useGlobalStore } from '@/store/modules/global'
import { useUserStore } from '@/store/modules/user'
import { logout as logoutApi } from '@/api/modules/auth'
import { listNodes } from '@/api/modules/node'
import { getSystemStats } from '@/api/modules/monitor'
import { getCurrentVersion } from '@/api/modules/upgrade'
import { rebootServer, restartPanel } from '@/api/modules/setting'
import { useI18n } from 'vue-i18n'
import type { NodeItem } from '@/api/interface'
import { Moon, Sunny, Check, Clock, RefreshRight, Timer } from '@element-plus/icons-vue'
import { ACCENT_PRESETS, getPresetByKey, applyAccentPalette, generatePaletteFromHex } from '@/utils/accent-colors'

const route = useRoute()
const router = useRouter()
const globalStore = useGlobalStore()
const userStore = useUserStore()
const { t } = useI18n()

const themeLabel = computed(() => {
  const labels = { dark: t('header.themeDark'), light: t('header.themeLight'), auto: t('header.themeAuto') }
  return labels[globalStore.theme] || labels.dark
})

const currentAccentColor = computed(() => {
  if (globalStore.accentKey === 'custom' && globalStore.accentCustom) return globalStore.accentCustom
  return getPresetByKey(globalStore.accentKey)?.primary || '#22d3ee'
})

const selectAccent = (key: string) => {
  globalStore.setAccent(key)
  const preset = getPresetByKey(key)
  if (preset) applyAccentPalette(preset)
}

const onCustomColor = (e: Event) => {
  const hex = (e.target as HTMLInputElement).value
  globalStore.setAccent('custom', hex)
  applyAccentPalette(generatePaletteFromHex(hex))
}

const nodes = ref<NodeItem[]>([])
const currentNode = ref(globalStore.currentNodeID || 0)

const loadNodes = async () => {
  try {
    const res = await listNodes()
    nodes.value = res.data || []
  } catch { /* ignore */ }
}

const onNodeChange = (val: number) => {
  const node = nodes.value.find((n: NodeItem) => n.id === val)
  globalStore.setCurrentNode(val, node ? node.name : '')
  window.location.reload()
}

let serverInfoTimer: ReturnType<typeof setInterval> | null = null
let clockTimer: ReturnType<typeof setInterval> | null = null
const serverClock = ref('')

const extractIANA = (tz: string): string => {
  const match = tz.match(/^([A-Za-z_/]+)/)
  return match ? match[1] : tz
}

const updateClock = () => {
  const rawTz = globalStore.serverInfo?.timezone
  if (!rawTz) { serverClock.value = ''; return }
  try {
    const iana = extractIANA(rawTz)
    const fmt = new Intl.DateTimeFormat('zh-CN', {
      timeZone: iana,
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit', second: '2-digit',
      hour12: false, timeZoneName: 'short',
    })
    serverClock.value = fmt.format(new Date())
  } catch {
    serverClock.value = ''
  }
}

const fetchServerInfo = async () => {
  try {
    const res = await getSystemStats()
    const h = res.data?.host
    if (h) {
      globalStore.setServerInfo({
        hostname: h.hostname || '',
        platform: h.platform || '',
        platformVersion: h.platformVersion || '',
        kernelArch: h.kernelArch || '',
        virtualization: h.virtualization || '',
        uptime: res.data.uptime || 0,
        timezone: h.timezone || '',
      })
      updateClock()
    }
  } catch { /* ignore */ }
}

const fetchVersion = async () => {
  try {
    const res = await getCurrentVersion()
    if (res.data) {
      globalStore.setVersion(res.data.version === 'dev' ? 'dev' : res.data.version)
    }
  } catch { /* ignore */ }
}

const formatUptime = (seconds: number) => {
  if (!seconds) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (d > 0) parts.push(`${d} ${t('monitor.days')}`)
  if (h > 0) parts.push(`${h} ${t('monitor.hours')}`)
  parts.push(`${m} ${t('monitor.minutes')}`)
  return parts.join(' ')
}

const handleRebootServer = async () => {
  await ElMessageBox.confirm(t('home.rebootConfirm'), t('commons.tip'), { type: 'warning', confirmButtonText: t('home.rebootServer') })
  await rebootServer()
  ElMessage.success(t('home.rebootSuccess'))
}

const handleRestartPanel = async () => {
  await ElMessageBox.confirm(t('home.restartPanelConfirm'), t('commons.tip'), { type: 'warning' })
  await restartPanel()
  ElMessage.success(t('home.restartPanelSuccess'))
}

onMounted(() => {
  loadNodes()
  fetchServerInfo()
  fetchVersion()
  serverInfoTimer = setInterval(fetchServerInfo, 30000)
  clockTimer = setInterval(updateClock, 1000)
})

onUnmounted(() => {
  if (serverInfoTimer) clearInterval(serverInfoTimer)
  if (clockTimer) clearInterval(clockTimer)
})

const breadcrumbs = computed(() => {
  return route.matched
    .filter((item) => item.meta?.title)
    .map((item) => ({
      path: item.path,
      title: t(item.meta.title as string),
    }))
})

const handleCommand = async (command: string) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm(t('header.logoutConfirm'), t('commons.tip'), {
        type: 'warning',
        confirmButtonText: t('commons.confirm'),
        cancelButtonText: t('commons.cancel'),
      })
      await logoutApi()
      userStore.logout()
      globalStore.setLogin(false)
      router.push('/login')
    } catch {
      // cancelled
    }
  } else if (command === 'password') {
    router.push('/setting')
  }
}
</script>

<style lang="scss" scoped>
.header {
  height: var(--xp-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: var(--xp-bg-header);
  backdrop-filter: blur(16px) saturate(1.8);
  border-bottom: 1px solid var(--xp-border-light);
  flex-shrink: 0;
  position: relative;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  overflow: hidden;
  flex: 1;
  min-width: 0;

  .server-info {
    display: flex;
    align-items: center;
    gap: 8px;
    overflow: hidden;
    flex-wrap: nowrap;
    min-width: 0;
  }

  .server-identity {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .server-hostname {
    font-weight: 700;
    font-size: 14px;
    color: var(--xp-text-primary);
    white-space: nowrap;
  }

  .server-uptime {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--xp-accent);
    background: var(--xp-accent-muted);
    padding: 2px 10px;
    border-radius: 12px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .server-clock {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    color: var(--xp-text-secondary);
    background: rgba(255, 255, 255, 0.04);
    padding: 2px 10px;
    border-radius: 12px;
    white-space: nowrap;
    flex-shrink: 0;
    font-family: var(--xp-font-mono);
    font-variant-numeric: tabular-nums;
  }

  .server-actions {
    flex-shrink: 0;
  }

  .collapse-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--xp-radius-sm);
    color: var(--xp-text-secondary);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
      color: var(--xp-accent);
    }
  }
}

.header-right {
  display: flex;
  align-items: center;
  gap: 8px;

  .theme-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--xp-radius-sm);
    color: var(--xp-text-secondary);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
      color: var(--xp-accent);
    }

    .accent-dot {
      width: 18px;
      height: 18px;
      border-radius: 50%;
      border: 2px solid rgba(255, 255, 255, 0.2);
      transition: all 0.2s;
    }
  }

  .user-dropdown {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    padding: 4px 10px;
    border-radius: var(--xp-radius-sm);
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
    }

    .user-avatar {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(135deg, var(--xp-accent), var(--xp-accent-secondary));
      border-radius: 50%;
      color: #0b0e14;
    }

    .username {
      font-size: 13px;
      color: var(--xp-text-secondary);
      max-width: 100px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .arrow {
      color: var(--xp-text-muted);
    }
  }
}
</style>

<style lang="scss">
.accent-panel {
  .accent-section {
    margin-bottom: 12px;
  }

  .accent-panel-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--xp-text-muted);
    letter-spacing: 0.5px;
    margin-bottom: 10px;
  }

  .accent-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
    justify-items: center;
  }

  .accent-swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
    transition: all 0.2s;
    border: 2px solid transparent;
    flex-shrink: 0;

    &:hover {
      transform: scale(1.15);
    }

    &.active {
      border-color: var(--xp-text-primary);
      box-shadow: 0 0 0 2px var(--xp-bg-surface), 0 0 0 3px var(--xp-accent);
    }
  }

  .accent-custom-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 10px;
    border-top: 1px solid var(--xp-border-light);
  }

  .accent-custom-label {
    font-size: 12px;
    color: var(--xp-text-secondary);
  }

  .accent-color-input {
    width: 32px;
    height: 28px;
    border: 1px solid var(--xp-border);
    border-radius: 6px;
    padding: 2px;
    background: transparent;
    cursor: pointer;

    &::-webkit-color-swatch-wrapper { padding: 2px; }
    &::-webkit-color-swatch { border-radius: 4px; border: none; }
  }
}
</style>
