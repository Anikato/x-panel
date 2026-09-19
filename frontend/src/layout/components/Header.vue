<template>
  <div class="header">
    <div class="header-left">
      <div
        class="icon-btn"
        role="button"
        tabindex="0"
        :aria-label="globalStore.menuCollapse ? t('header.expandMenu') : t('header.collapseMenu')"
        @click="globalStore.toggleMenuCollapse"
        @keydown.enter.prevent="globalStore.toggleMenuCollapse"
        @keydown.space.prevent="globalStore.toggleMenuCollapse"
      >
        <el-icon :size="18">
          <Fold v-if="!globalStore.menuCollapse" />
          <Expand v-else />
        </el-icon>
      </div>

      <el-popover placement="bottom-start" :width="360" trigger="click" popper-class="xp-float">
        <template #reference>
          <button type="button" class="server-chip" :aria-label="serverChipLabel" :title="connectionLabel">
            <span class="status-dot" :class="connectionStatus" />
            <span class="server-name">{{ globalStore.serverInfo?.hostname || globalStore.panelName || 'X-Panel' }}</span>
          </button>
        </template>
        <div class="server-summary">
          <div class="summary-row"><span>{{ t('home.hostname') }}</span><strong>{{ globalStore.serverInfo?.hostname || '—' }}</strong></div>
          <div class="summary-row"><span>{{ t('home.os') }}</span><strong>{{ osLabel }}</strong></div>
          <div class="summary-row"><span>{{ t('home.arch') }}</span><strong>{{ globalStore.serverInfo?.kernelArch || '—' }}</strong></div>
          <div class="summary-row"><span>{{ t('home.uptime') }}</span><strong>{{ formatUptime(globalStore.serverInfo?.uptime || 0) }}</strong></div>
          <div v-if="globalStore.showServerClock && serverClock" class="summary-row">
            <span>{{ t('header.serverClock') }}</span><strong class="mono">{{ serverClock }}</strong>
          </div>
          <div class="summary-row"><span>{{ t('home.panelVersion') }}</span><strong>{{ globalStore.version || '—' }}</strong></div>
          <div class="summary-actions">
            <el-button size="small" @click="handleRestartPanel">{{ t('home.restartPanel') }}</el-button>
            <el-button size="small" type="danger" plain @click="handleRebootServer">{{ t('home.rebootServer') }}</el-button>
          </div>
        </div>
      </el-popover>
    </div>

    <div class="header-right">
      <button type="button" class="search-entry" @click="searchOpen = true">
        <el-icon><Search /></el-icon>
        <span>{{ t('header.search') }}</span>
        <kbd>{{ searchShortcut }}</kbd>
      </button>

      <el-tooltip :content="t('header.quickTerminal')" placement="bottom">
        <div
          class="icon-btn"
          :class="{ active: globalStore.floatTermVisible }"
          role="button"
          tabindex="0"
          :aria-label="t('header.quickTerminal')"
          @click="toggleFloatTerm"
          @keydown.enter.prevent="toggleFloatTerm"
          @keydown.space.prevent="toggleFloatTerm"
        >
          <el-icon :size="16"><Monitor /></el-icon>
        </div>
      </el-tooltip>

      <el-tooltip :content="t('header.tasks')" placement="bottom">
        <div
          class="icon-btn"
          role="button"
          tabindex="0"
          :aria-label="t('header.tasks')"
          @click="taskOpen = true"
          @keydown.enter.prevent="taskOpen = true"
          @keydown.space.prevent="taskOpen = true"
        >
          <el-badge :value="taskCount" :hidden="taskCount <= 0" :max="99">
            <el-icon :size="16"><List /></el-icon>
          </el-badge>
        </div>
      </el-tooltip>

      <el-popover
        placement="bottom-end"
        :width="380"
        trigger="click"
        popper-class="xp-notification-popper"
        :offset="10"
        @show="fetchRecentNotifications"
      >
        <template #reference>
          <div
            class="icon-btn"
            role="button"
            tabindex="0"
            :aria-label="t('notification.title')"
          >
            <el-badge :value="unreadNotifications" :hidden="unreadNotifications <= 0" :max="99">
              <el-icon :size="16"><Bell /></el-icon>
            </el-badge>
          </div>
        </template>
        <div class="notification-panel">
          <div class="notification-panel-head">
            <strong>{{ t('notification.title') }}</strong>
            <el-button link type="primary" @click="openNotifications">{{ t('notification.viewAll') }}</el-button>
          </div>
          <div v-if="recentNotifications.length === 0" class="notification-empty">{{ t('commons.noData') }}</div>
          <div v-else class="notification-recent-list">
            <div
              v-for="item in recentNotifications"
              :key="item.id"
              class="notification-recent-item"
              :class="{ unread: !item.readAt }"
              @click="openNotificationItem(item)"
            >
              <span class="type-dot" :class="item.type"></span>
              <div class="notification-recent-main">
                <div class="notification-recent-title">{{ item.title }}</div>
                <div v-if="item.content" class="notification-recent-content">{{ item.content }}</div>
              </div>
            </div>
          </div>
        </div>
      </el-popover>

      <el-tooltip :content="themeLabel" placement="bottom">
        <div
          class="icon-btn"
          role="button"
          tabindex="0"
          :aria-label="themeLabel"
          @click="cycleColorMode"
          @keydown.enter.prevent="cycleColorMode"
          @keydown.space.prevent="cycleColorMode"
        >
          <el-icon :size="16">
            <Moon v-if="appearanceStore.preference.mode === 'dark'" />
            <Sunny v-else-if="appearanceStore.preference.mode === 'light'" />
            <Monitor v-else />
          </el-icon>
        </div>
      </el-tooltip>

      <el-dropdown @command="handleCommand" trigger="click">
        <div class="user-dropdown">
          <div class="user-avatar">
            <UserAvatar :name="userStore.name || 'admin'" />
          </div>
          <span class="username">{{ userStore.name || 'admin' }}</span>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="appearance">
              <el-icon><Brush /></el-icon>{{ t('setting.appearance') }}
            </el-dropdown-item>
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

    <el-dialog v-model="searchOpen" :show-close="false" width="520px" append-to-body class="command-search-dialog" @opened="focusSearch">
      <el-input
        ref="searchInput"
        v-model="searchKeyword"
        :placeholder="t('header.searchPlaceholder')"
        prefix-icon="Search"
        @keydown.enter.prevent="openHit(searchHits[0])"
      />
      <div class="search-hits">
        <button
          v-for="hit in searchHits"
          :key="hit.id + hit.path"
          type="button"
          class="search-hit"
          @click="openHit(hit)"
        >
          <span>{{ t(hit.titleKey) }}</span>
          <code>{{ hit.path }}</code>
        </button>
        <div v-if="searchKeyword && searchHits.length === 0" class="search-empty">{{ t('header.searchEmpty') }}</div>
      </div>
    </el-dialog>

    <TaskDrawer v-model="taskOpen" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { useGlobalStore } from '@/store/modules/global'
import { useAppearanceStore } from '@/store/modules/appearance'
import type { ThemeMode } from '@/theme/types.ts'
import { useUserStore } from '@/store/modules/user'
import { useUploadStore } from '@/store/modules/upload'
import { useFileTaskStore } from '@/store/modules/fileTask'
import { logout as logoutApi } from '@/api/modules/auth'
import { getSystemStats } from '@/api/modules/monitor'
import { getCurrentVersion } from '@/api/modules/upgrade'
import { rebootServer, restartPanel } from '@/api/modules/setting'
import {
  getNotificationSummary,
  getRecentNotifications,
  markNotificationsRead,
} from '@/api/modules/notification'
import { useI18n } from 'vue-i18n'
import type { NotificationItem } from '@/api/interface'
import { searchNavigation, type NavHit } from '@/navigation/registry'
import TaskDrawer from './TaskDrawer.vue'
import UserAvatar from './UserAvatar.vue'
import { reachStatus } from '../reach-status.ts'
import { searchShortcutLabel } from '../search-shortcut.ts'

const router = useRouter()
const globalStore = useGlobalStore()
const appearanceStore = useAppearanceStore()
const userStore = useUserStore()
const uploadStore = useUploadStore()
const fileTaskStore = useFileTaskStore()
const { t } = useI18n()

const searchShortcut = searchShortcutLabel()
const searchOpen = ref(false)
const searchKeyword = ref('')
const searchInput = ref<{ focus?: () => void } | null>(null)
const taskOpen = ref(false)
const serverClock = ref('')
const reachFailStreak = ref(0)
const connectionStatus = computed(() => reachStatus(reachFailStreak.value))
const connectionLabel = computed(() => {
  const labels = {
    online: t('header.panelReachable'),
    warning: t('header.connectionUnstable'),
    offline: t('header.panelDisconnected'),
  }
  return labels[connectionStatus.value]
})
const serverChipLabel = computed(() => `${t('header.serverSummary')} · ${connectionLabel.value}`)
const unreadNotifications = ref(0)
const recentNotifications = ref<NotificationItem[]>([])
const popupShown = new Set<number>()

const themeLabel = computed(() => {
  const labels = { dark: t('header.themeDark'), light: t('header.themeLight'), auto: t('header.themeAuto') }
  return labels[appearanceStore.preference.mode] || labels.dark
})

const cycleColorMode = () => {
  const order: ThemeMode[] = ['dark', 'light', 'auto']
  const idx = order.indexOf(appearanceStore.preference.mode)
  appearanceStore.applyImmediate({ mode: order[(idx + 1) % order.length] })
}

const osLabel = computed(() => {
  const info = globalStore.serverInfo
  if (!info) return '—'
  return `${info.platform} ${info.platformVersion}`.trim()
})

const searchHits = computed(() => searchNavigation(searchKeyword.value).slice(0, 12))

const taskCount = computed(() => {
  const uploads = uploadStore.queue.filter((item) => !item.error && item.progress < 100).length
  return uploads + fileTaskStore.runningCount
})

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
      reachFailStreak.value = 0
    }
  } catch {
    reachFailStreak.value += 1
  }
}

const fetchVersion = async () => {
  try {
    const res = await getCurrentVersion()
    if (res.data) {
      globalStore.setVersion(res.data.version === 'dev' ? 'dev' : res.data.version)
    }
  } catch { /* ignore */ }
}

const fetchNotificationSummary = async () => {
  try {
    const res: any = await getNotificationSummary()
    unreadNotifications.value = res.data?.unread || 0
  } catch { /* ignore */ }
}

const fetchRecentNotifications = async () => {
  try {
    const res: any = await getRecentNotifications()
    const items = res.data || []
    recentNotifications.value = items
    items
      .filter((item: NotificationItem) => item.popup && !item.readAt && !popupShown.has(item.id))
      .slice(0, 3)
      .forEach((item: NotificationItem) => {
        popupShown.add(item.id)
        ElNotification({
          title: item.title,
          message: item.content || '',
          type: item.type === 'error' ? 'error' : item.type,
          duration: item.type === 'error' ? 8000 : 4500,
          position: 'bottom-right',
          onClick: () => openNotificationItem(item),
        })
      })
  } catch { /* ignore */ }
}

const openNotificationItem = async (item: NotificationItem) => {
  if (!item.readAt) {
    await markNotificationsRead({ ids: [item.id] })
    await fetchNotificationSummary()
    await fetchRecentNotifications()
  }
  if (item.targetUrl) router.push(item.targetUrl)
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
  } else if (command === 'appearance') {
    router.push({ path: '/setting', hash: '#setting-appearance' })
  } else if (command === 'password') {
    router.push('/setting')
  }
}

const toggleFloatTerm = () => {
  if (globalStore.floatTermVisible && globalStore.floatTermMinimized) {
    globalStore.floatTermMinimized = false
  } else {
    globalStore.floatTermVisible = !globalStore.floatTermVisible
    if (globalStore.floatTermVisible) globalStore.floatTermMinimized = false
  }
}

const openNotifications = () => router.push('/notifications')

const focusSearch = () => searchInput.value?.focus?.()

const openHit = (hit?: NavHit) => {
  if (!hit) return
  searchOpen.value = false
  searchKeyword.value = ''
  router.push({ path: hit.path, query: hit.query })
}

const isEditableTarget = (target: EventTarget | null) => {
  if (!(target instanceof HTMLElement)) return false
  const tag = target.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || target.isContentEditable) return true
  return Boolean(target.closest('.xterm, .monaco-editor, textarea, input'))
}

const onGlobalKeydown = (event: KeyboardEvent) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    if (isEditableTarget(event.target)) return
    event.preventDefault()
    searchOpen.value = true
  }
}

let serverInfoTimer: ReturnType<typeof setInterval> | null = null
let clockTimer: ReturnType<typeof setInterval> | null = null
let notificationTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  fetchServerInfo()
  fetchVersion()
  fetchNotificationSummary()
  fetchRecentNotifications()
  serverInfoTimer = setInterval(fetchServerInfo, 30000)
  clockTimer = setInterval(updateClock, 1000)
  notificationTimer = setInterval(() => {
    fetchNotificationSummary()
    fetchRecentNotifications()
  }, 30000)
  window.addEventListener('keydown', onGlobalKeydown)
})

onUnmounted(() => {
  if (serverInfoTimer) clearInterval(serverInfoTimer)
  if (clockTimer) clearInterval(clockTimer)
  if (notificationTimer) clearInterval(notificationTimer)
  window.removeEventListener('keydown', onGlobalKeydown)
})
</script>

<style lang="scss" scoped>
.header {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  height: var(--xp-header-height);
  padding: 0 16px 0 12px;
  background: var(--xp-bg-header);
  border-bottom: 1px solid var(--xp-border);
}

.header-left,
.header-right {
  display: flex;
  align-items: center;
  min-width: 0;
  height: 100%;
  gap: 8px;
}

.header-right {
  :deep(.el-tooltip__trigger),
  :deep(.el-only-child__content),
  :deep(.el-dropdown) {
    display: inline-flex;
    align-items: center;
    height: 32px;
    line-height: 1;
  }
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  line-height: 1;
  color: var(--xp-text-secondary);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  :deep(.el-badge) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
    height: 16px;
  }

  :deep(.el-icon) {
    display: inline-flex;
    align-items: center;
    justify-content: center;
  }

  &:hover,
  &.active {
    color: var(--xp-accent);
    background: var(--xp-accent-muted);
  }
}

.server-chip {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  max-width: 360px;
  height: 32px;
  padding: 0 10px;
  gap: 8px;
  line-height: 1;
  color: var(--xp-text-primary);
  background: transparent;
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  &:hover {
    border-color: var(--xp-accent);
  }
}

.server-name {
  overflow: hidden;
  font-size: 13px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-entry {
  display: inline-flex;
  align-items: center;
  min-width: 180px;
  height: 32px;
  padding: 0 10px;
  gap: 8px;
  line-height: 1;
  color: var(--xp-text-muted);
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  span {
    flex: 1;
    text-align: left;
    font-size: 13px;
    line-height: 1;
  }

  kbd {
    color: var(--xp-text-muted);
    font-size: 11px;
    line-height: 1;
  }
}

.user-dropdown {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 8px;
  gap: 8px;
  line-height: 1;
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  &:hover {
    background: var(--xp-accent-muted);
  }
}

.user-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--xp-on-accent, #fff);
  background: var(--xp-accent);
  border-radius: 50%;
}

.username {
  max-width: 100px;
  overflow: hidden;
  color: var(--xp-text-secondary);
  font-size: 13px;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.server-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--xp-text-muted);
  font-size: 12px;

  strong {
    color: var(--xp-text-primary);
    font-weight: 600;
  }
}

.summary-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 8px;
  gap: 8px;
}

.notification-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.notification-empty {
  padding: 18px 0;
  color: var(--xp-text-secondary);
  font-size: 13px;
  text-align: center;
}

.notification-recent-list {
  display: flex;
  flex-direction: column;
  max-height: 360px;
  overflow: auto;
  gap: 4px;
}

.notification-recent-item {
  display: flex;
  padding: 8px;
  gap: 10px;
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  &:hover {
    background: var(--xp-accent-muted);
  }
}

.notification-recent-title {
  color: var(--xp-text-secondary);
  font-size: 13px;
}

.unread .notification-recent-title {
  color: var(--xp-text-primary);
  font-weight: 700;
}

.notification-recent-content {
  overflow: hidden;
  color: var(--xp-text-muted);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.search-hits {
  display: flex;
  flex-direction: column;
  max-height: 360px;
  margin-top: 12px;
  overflow: auto;
}

.search-hit {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 36px;
  padding: 0 10px;
  color: var(--xp-text-primary);
  background: transparent;
  border: 0;
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  &:hover {
    background: var(--xp-accent-muted);
  }

  code {
    color: var(--xp-text-muted);
    font-size: 12px;
  }
}

.search-empty {
  padding: 20px 0;
  color: var(--xp-text-muted);
  text-align: center;
}

@media (max-width: 1100px) {
  .search-entry span,
  .search-entry kbd,
  .username {
    display: none;
  }

  .search-entry {
    min-width: 32px;
    width: 32px;
    padding: 0;
    justify-content: center;
  }
}
</style>
