<template>
  <Teleport to="body">
    <!-- 最小化状态：底部右侧小条 -->
    <div
      v-if="globalStore.floatTermVisible && globalStore.floatTermMinimized"
      class="float-term-bar"
      @click="globalStore.floatTermMinimized = false"
    >
      <el-icon :size="13"><Monitor /></el-icon>
      <span>{{ t('terminal.title') }}</span>
      <span class="term-bar-dot" />
      <el-icon :size="12" class="bar-close" @click.stop="globalStore.floatTermVisible = false"><Close /></el-icon>
    </div>

    <button
      v-if="!globalStore.floatTermVisible && sessionStore.surface === 'float' && sessionStore.items.length"
      type="button"
      class="float-term-sessions"
      @click="showBackgroundTabs"
    >
      {{ t('terminal.backgroundHint', { count: sessionStore.items.length }) }}
    </button>

    <!-- 展开状态：悬浮面板 -->
    <div
      v-show="globalStore.floatTermVisible && !globalStore.floatTermMinimized"
      class="float-term-panel"
      :style="panelStyle"
    >
      <!-- 标题栏（拖拽手柄） -->
      <div class="float-term-header" @mousedown="startDrag">
        <div class="float-term-title">
          <el-icon :size="13"><Monitor /></el-icon>
          <span>{{ t('terminal.title') }}</span>
          <span v-if="tabs.length > 1" class="tab-count">{{ t('terminal.tabCount', { count: tabs.length }) }}</span>
          <span v-if="activeTabInfo" class="conn-state" :class="activeTabInfo.status">
            <span class="conn-dot" />
            {{ connectionText(activeTabInfo) }}
          </span>
        </div>
        <div class="float-term-actions" @mousedown.stop>
          <!-- 字体大小 -->
          <el-icon :size="11" class="action-btn" @click="changeFontSize(-1)"><Minus /></el-icon>
          <span class="fs-label">{{ globalStore.termFontSize }}</span>
          <el-icon :size="11" class="action-btn" @click="changeFontSize(1)"><Plus /></el-icon>
          <button
            v-if="activeTabInfo?.locked"
            type="button"
            class="action-text"
            @click="disconnectActive"
          >{{ t('terminal.disconnect') }}</button>
          <button
            v-if="activeTabInfo?.locked"
            type="button"
            class="action-text danger"
            @click="terminateTab(activeTabInfo)"
          >{{ sessionStatus(activeTabInfo) === 'exited' ? t('terminal.dismiss') : t('terminal.terminate') }}</button>
          <!-- 重连 -->
          <el-icon
            v-if="activeTabInfo && activeTabInfo.status !== 'connected' && activeTabInfo.status !== 'connecting'"
            :size="13"
            class="action-btn"
            :title="t('terminal.reconnect')"
            @click="reconnectTab(activeTabInfo, true)"
          ><RefreshRight /></el-icon>
          <!-- 新建标签 -->
          <el-icon :size="13" class="action-btn" :title="t('terminal.newTerminal')" @click="addLocalTab"><Plus /></el-icon>
          <!-- 最小化 -->
          <el-icon :size="13" class="action-btn" :title="t('commons.minimize')" @click="globalStore.floatTermMinimized = true"><SemiSelect /></el-icon>
          <!-- 关闭 -->
          <el-icon :size="13" class="action-btn close-btn" :title="t('commons.close')" @click="globalStore.floatTermVisible = false"><Close /></el-icon>
        </div>
      </div>

      <!-- 标签栏（多标签时显示） -->
      <div v-if="tabs.length" class="float-term-tabs">
        <div
          v-for="(tab, idx) in tabs"
          :key="tab.id"
          class="ft-tab"
          :class="{ active: activeTab === tab.id, disconnected: tab.status === 'disconnected', reconnecting: tab.status === 'reconnecting' }"
          @click="switchTab(tab.id)"
        >
          <button
            v-if="!tab.hostId"
            type="button"
            class="ft-tab-lock"
            :class="{ locked: tab.locked }"
            :title="tab.locked ? t('terminal.unlockBackground') : t('terminal.lockBackground')"
            @click.stop="toggleLock(tab)"
          >
            <el-icon :size="11"><Lock v-if="tab.locked" /><Unlock v-else /></el-icon>
          </button>
          <span class="ft-tab-dot" />
          <span>{{ tab.title }}</span>
          <button type="button" class="ft-tab-close" :title="t('commons.close')" @click.stop="requestCloseTab(idx)">
            <el-icon :size="10"><Close /></el-icon>
          </button>
        </div>
      </div>

      <!-- 终端区域 -->
      <div class="float-term-body xp-term-stage" ref="bodyRef">
        <div v-if="tabs.length === 0" class="ft-empty">{{ t('terminal.backgroundEmpty') }}</div>
        <div
          v-for="tab in tabs"
          :key="tab.id"
          :ref="(el) => setTermRef(tab.id, el as HTMLElement)"
          class="ft-term-instance"
          :class="{ active: activeTab === tab.id }"
        />
      </div>

      <!-- 调整大小手柄 -->
      <div class="resize-handle resize-se" @mousedown.stop="startResize($event, 'se')" />
      <div class="resize-handle resize-e"  @mousedown.stop="startResize($event, 'e')" />
      <div class="resize-handle resize-s"  @mousedown.stop="startResize($event, 's')" />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted, onBeforeUnmount, nextTick, reactive } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { useGlobalStore } from '@/store/modules/global'
import { useAppearanceStore } from '@/store/modules/appearance'
import { getTermThemeByKey, getTermFontByKey, applyBgOpacity, loadTerminalIconFont } from '@/utils/terminal-theme'
import { normalizeTerminalCwd } from '@/utils/terminal-cwd'
import { buildTerminalWsUrl } from '@/utils/terminal-ws'
import { issueAccessTicket } from '@/api/modules/auth'
import { readTerminalServerFrame } from '@/utils/terminal-frame'
import { installTerminalClipboard } from '@/utils/terminal-clipboard'
import { toggleBackgroundLock } from '@/utils/terminal-lock'
import { useTerminalSessionStore, type BackgroundTerminal } from '@/store/modules/terminal-session'
import { ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const globalStore = useGlobalStore()
const appearance = useAppearanceStore()
const sessionStore = useTerminalSessionStore()
const bodyRef = ref<HTMLElement | null>(null)

// ==================== 面板位置/尺寸 ====================
const pos = reactive({ x: window.innerWidth - 820, y: window.innerHeight - 480 })
const size = reactive({ w: 800, h: 440 })

const panelStyle = ref({})
const updateStyle = () => {
  panelStyle.value = {
    left: `${Math.max(0, Math.min(pos.x, window.innerWidth - size.w))}px`,
    top:  `${Math.max(0, Math.min(pos.y, window.innerHeight - size.h))}px`,
    width:  `${size.w}px`,
    height: `${size.h}px`,
  }
}
updateStyle()

// 拖拽
let dragging = false
let dragOffX = 0, dragOffY = 0
const startDrag = (e: MouseEvent) => {
  dragging = true
  dragOffX = e.clientX - pos.x
  dragOffY = e.clientY - pos.y
}
const onMouseMove = (e: MouseEvent) => {
  if (dragging) {
    pos.x = e.clientX - dragOffX
    pos.y = e.clientY - dragOffY
    updateStyle()
  }
  if (resizing) {
    const dx = e.clientX - resizeStart.x
    const dy = e.clientY - resizeStart.y
    if (resizeDir.includes('e')) size.w = Math.max(400, resizeStart.w + dx)
    if (resizeDir.includes('s')) size.h = Math.max(200, resizeStart.h + dy)
    updateStyle()
    fitActive()
  }
}
const onMouseUp = () => { dragging = false; resizing = false }

// 调整大小
let resizing = false
let resizeDir = ''
const resizeStart = { x: 0, y: 0, w: 0, h: 0 }
const startResize = (e: MouseEvent, dir: string) => {
  resizing = true
  resizeDir = dir
  resizeStart.x = e.clientX; resizeStart.y = e.clientY
  resizeStart.w = size.w;    resizeStart.h = size.h
}

// ==================== 终端 Tab ====================
type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'reconnecting'

interface TermTab {
  id: string; title: string; hostId?: number
  terminal?: Terminal; fitAddon?: FitAddon; ws?: WebSocket
  _observer?: ResizeObserver
  status: ConnectionStatus
  reconnectAttempts: number
  reconnectTimer?: ReturnType<typeof setTimeout>
  reconnectDelay?: number
  closing?: boolean
  suppressReconnect?: boolean
  connectGeneration?: number
  cwd?: string
  sessionId?: string
  locked?: boolean
}

const tabs = ref<TermTab[]>([])
const activeTab = ref('')
const termRefs: Record<string, HTMLElement | null> = {}
let tabCounter = 0
const reconnectDelays = [1000, 2000, 5000, 10000]

const setTermRef = (id: string, el: HTMLElement | null) => { if (el) termRefs[id] = el }

const activeTabInfo = computed(() => tabs.value.find(t => t.id === activeTab.value))

const connectionText = (tab: TermTab) => {
  if (tab.status === 'connected') return t('terminal.connected')
  if (tab.status === 'connecting') return t('terminal.connecting')
  if (tab.status === 'reconnecting') return t('terminal.reconnecting', { seconds: Math.ceil((tab.reconnectDelay || 0) / 1000) })
  return t('terminal.disconnected')
}

const fitActive = () => {
  nextTick(() => {
    const tab = tabs.value.find(t => t.id === activeTab.value)
    try { tab?.fitAddon?.fit() } catch { /* */ }
  })
}

const getWsUrl = async (tab: TermTab) => {
  const res = await issueAccessTicket({ scope: 'terminal' })
  return buildTerminalWsUrl({
    protocol: location.protocol,
    host: location.host,
    ticket: res.data.ticket,
    hostId: tab.sessionId ? undefined : tab.hostId,
    cwd: tab.sessionId ? undefined : tab.cwd,
    sessionId: tab.sessionId,
  })
}

const sessionStatus = (tab?: TermTab) =>
  sessionStore.items.find((item) => item.id === tab?.sessionId)?.status

const handleTerminalControl = (tab: TermTab, op: string, id: string, message: string) => {
  if (!tabs.value.includes(tab)) return
  const term = tab.terminal
  if (op === 'session') {
    tab.sessionId = id
    tab.locked = message === 'true'
    return
  }
  if (op === 'detached') {
    tab.closing = true
    clearReconnectTimer(tab)
    void (async () => {
      await sessionStore.refresh().catch(() => {})
      const idx = tabs.value.findIndex((item) => item.id === tab.id)
      if (idx >= 0) closeTab(idx)
    })()
    return
  }
  if (tab.closing) return
  if (op === 'detach_rejected') {
    term?.write(`\r\n\x1b[31m${message || t('terminal.detachFailed')}\x1b[0m\r\n`)
    return
  }
  if (op === 'taken') {
    tab.suppressReconnect = true
    term?.write(`\r\n\x1b[33m${t('terminal.attachedElsewhere')}\x1b[0m\r\n`)
    return
  }
  if (op === 'missing') {
    tab.suppressReconnect = true
    term?.write(`\r\n\x1b[31m${message || t('terminal.sessionGone')}\x1b[0m\r\n`)
    sessionStore.refresh().catch(() => {})
    return
  }
  if (op === 'exited') {
    tab.suppressReconnect = true
    tab.status = 'disconnected'
    sessionStore.refresh().catch(() => {})
    return
  }
  if (op === 'overflow') {
    term?.reset()
    term?.write(`\r\n\x1b[33m${t('terminal.outputOverflow')}\x1b[0m\r\n`)
    connectWebSocket(tab)
  }
}

const toggleLock = (tab: TermTab) => toggleBackgroundLock(tab, sessionStore, t)

const showBackgroundTabs = () => {
  sessionStore.surface = 'float'
  globalStore.floatTermVisible = true
  globalStore.floatTermMinimized = false
}

const ensureSessionTabs = async () => {
  await sessionStore.refresh().catch(() => {})
  for (const item of sessionStore.items) {
    if (tabs.value.some((tab) => tab.sessionId === item.id)) continue
    await openBackgroundSession(item)
  }
}

const disconnectActive = () => {
  const tab = activeTabInfo.value
  if (!tab) return
  const idx = tabs.value.findIndex((item) => item.id === tab.id)
  if (idx >= 0) void requestCloseTab(idx)
}

const terminateTab = async (tab?: TermTab) => {
  if (!tab?.sessionId) return
  try {
    await sessionStore.terminate(tab.sessionId)
  } catch {
    return
  }
  const idx = tabs.value.findIndex((item) => item.id === tab.id)
  if (idx >= 0) closeTab(idx)
}

const openBackgroundSession = async (item: BackgroundTerminal) => {
  globalStore.floatTermVisible = true
  globalStore.floatTermMinimized = false
  const existing = tabs.value.find((tab) => tab.sessionId === item.id)
  if (existing) {
    switchTab(existing.id)
    return
  }
  const tab = createTab(item.name, item.cwd)
  tab.sessionId = item.id
  tab.locked = true
  tabs.value.push(tab)
  activeTab.value = tab.id
  await createTerminal(tab)
}

const sendResize = (ws: WebSocket, rows: number, cols: number) => {
  const data = JSON.stringify({ rows, cols })
  const msg = new Uint8Array(1 + data.length)
  msg[0] = 1
  for (let i = 0; i < data.length; i++) msg[i + 1] = data.charCodeAt(i)
  ws.send(msg)
}

const clearReconnectTimer = (tab: TermTab) => {
  if (tab.reconnectTimer) {
    clearTimeout(tab.reconnectTimer)
    tab.reconnectTimer = undefined
  }
}

const scheduleReconnect = (tab: TermTab) => {
  if (tab.closing || !tabs.value.includes(tab)) return
  clearReconnectTimer(tab)
  const delay = reconnectDelays[Math.min(tab.reconnectAttempts, reconnectDelays.length - 1)]
  tab.reconnectDelay = delay
  tab.status = 'reconnecting'
  tab.terminal?.write(`\r\n\x1b[33m${t('terminal.reconnecting', { seconds: Math.ceil(delay / 1000) })}\x1b[0m\r\n`)
  tab.reconnectTimer = setTimeout(() => {
    tab.reconnectAttempts++
    connectWebSocket(tab)
  }, delay)
}

const connectWebSocket = async (tab: TermTab) => {
  if (!tab.terminal || tab.closing || !tabs.value.includes(tab)) return
  const generation = (tab.connectGeneration || 0) + 1
  tab.connectGeneration = generation
  clearReconnectTimer(tab)
  tab.suppressReconnect = false
  tab.status = tab.reconnectAttempts > 0 ? 'reconnecting' : 'connecting'

  const previous = tab.ws
  if (previous && previous.readyState !== WebSocket.CLOSED) {
    previous.onclose = null
    previous.onerror = null
    previous.onmessage = null
    previous.onopen = null
    previous.close()
  }

  let url = ''
  try {
    url = await getWsUrl(tab)
  } catch {
    if (generation !== tab.connectGeneration || tab.closing || !tabs.value.includes(tab)) return
    tab.status = 'disconnected'
    tab.terminal?.write(`\r\n\x1b[31m${t('terminal.connError')}\x1b[0m\r\n`)
    scheduleReconnect(tab)
    return
  }
  if (generation !== tab.connectGeneration || tab.closing || !tabs.value.includes(tab)) return

  const term = tab.terminal
  const ws = new WebSocket(url)
  ws.binaryType = 'arraybuffer'
  if (generation !== tab.connectGeneration || tab.closing || !tabs.value.includes(tab)) {
    ws.close()
    return
  }
  if (tab.sessionId) term.reset()
  tab.ws = ws

  ws.onopen = () => {
    if (tab.ws !== ws) return
    tab.status = 'connected'
    tab.reconnectAttempts = 0
    tab.reconnectDelay = undefined
    sendResize(ws, term.rows, term.cols)
    term.focus()
  }
  ws.onmessage = (e) => {
    if (tab.ws !== ws) return
    const frame = readTerminalServerFrame(e.data)
    if (frame.kind === 'control') {
      handleTerminalControl(tab, frame.op, frame.id, frame.message)
      return
    }
    if ('bytes' in frame) term.write(frame.bytes)
    else term.write(frame.text)
  }
  ws.onclose = () => {
    if (tab.ws !== ws) return
    if (tab.closing || tab.suppressReconnect || !tabs.value.includes(tab)) return
    if (!tab.locked) tab.sessionId = undefined
    tab.status = 'disconnected'
    term.write(`\r\n\x1b[31m${t('terminal.disconnected')}\x1b[0m\r\n`)
    scheduleReconnect(tab)
  }
  ws.onerror = () => {
    if (tab.ws !== ws || tab.closing) return
    term.write(`\r\n\x1b[31m${t('terminal.connError')}\x1b[0m\r\n`)
  }
}

const reconnectTab = (tab: TermTab, manual = false) => {
  if (!tab.terminal) return
  clearReconnectTimer(tab)
  tab.reconnectAttempts = 0
  if (manual) tab.terminal.write(`\r\n\x1b[33m${t('terminal.reconnect')}...\x1b[0m\r\n`)
  connectWebSocket(tab)
}

const createTerminal = async (tab: TermTab) => {
  await nextTick()
  const el = termRefs[tab.id]
  if (!el) return
  await loadTerminalIconFont()
  if (!termRefs[tab.id] || tab.closing || !tabs.value.includes(tab)) return

  const term = new Terminal({
    cursorBlink: true, cursorStyle: 'bar',
    fontSize: globalStore.termFontSize,
    fontFamily: getTermFontByKey(globalStore.termFont),
    theme: applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity),
    scrollback: 10000, allowProposedApi: true, allowTransparency: true,
  })
  installTerminalClipboard(term)

  term.attachCustomKeyEventHandler((e: KeyboardEvent) => {
    if (e.ctrlKey && e.shiftKey && ['c','v'].includes(e.key.toLowerCase())) return false
    return true
  })

  const fit = new FitAddon()
  term.loadAddon(fit)
  term.open(el)
  tab.terminal = term; tab.fitAddon = fit

  const obs = new ResizeObserver(() => { if (activeTab.value === tab.id) { try { fit.fit() } catch { /* */ } } })
  obs.observe(el); tab._observer = obs

  setTimeout(() => { try { fit.fit() } catch { /* */ }; term.focus() }, 100)

  term.onData((d: string) => {
    if (!tab.ws || tab.ws.readyState !== WebSocket.OPEN) return
    tab.ws.send(d)
  })
  term.onResize(({ rows, cols }) => {
    if (tab.ws?.readyState === WebSocket.OPEN) sendResize(tab.ws, rows, cols)
  })
  connectWebSocket(tab)
}

const createTab = (title: string, cwd?: string): TermTab => {
  tabCounter++
  return reactive<TermTab>({
    id: `ft-${tabCounter}`,
    title,
    status: 'connecting',
    reconnectAttempts: 0,
    cwd,
  })
}

const addLocalTab = async () => {
  const tab = createTab(`终端 ${tabCounter + 1}`)
  tabs.value.push(tab)
  activeTab.value = tab.id
  await createTerminal(tab)
}

const switchTab = (id: string) => {
  activeTab.value = id
  nextTick(() => {
    const tab = tabs.value.find(t => t.id === id)
    try { tab?.fitAddon?.fit() } catch { /* */ }
    tab?.terminal?.focus()
  })
}

const requestCloseTab = async (idx: number) => {
  const tab = tabs.value[idx]
  if (!tab || tab.closing) return
  if (tab.locked) {
    try {
      await ElMessageBox.confirm(t('terminal.leaveLockedConfirm'), t('commons.tip'), {
        type: 'warning',
        confirmButtonText: t('terminal.leaveLocked'),
        cancelButtonText: t('commons.cancel'),
      })
    } catch {
      return
    }
    if (tab.closing) return
    idx = tabs.value.indexOf(tab)
    if (idx < 0) return
  }
  closeTab(idx)
}

const closeTab = (idx: number) => {
  const tab = tabs.value[idx]
  if (!tab || tab.closing) return
  tab.closing = true
  clearReconnectTimer(tab)
  tab.ws?.close(); tab.terminal?.dispose(); tab._observer?.disconnect()
  tabs.value.splice(idx, 1)
  if (activeTab.value === tab.id && tabs.value.length > 0) {
    activeTab.value = tabs.value[Math.min(idx, tabs.value.length - 1)].id
    nextTick(() => { const t = tabs.value.find(x => x.id === activeTab.value); try { t?.fitAddon?.fit() } catch { /* */ } })
  }
  if (tabs.value.length === 0 && sessionStore.items.length === 0) globalStore.floatTermVisible = false
}

watch(
  () => sessionStore.items.map((item) => item.id).join('\n'),
  (joined) => {
    const ids = new Set(joined ? joined.split('\n') : [])
    for (let i = tabs.value.length - 1; i >= 0; i--) {
      const tab = tabs.value[i]
      if (tab.locked && tab.sessionId && !ids.has(tab.sessionId)) closeTab(i)
    }
  },
)

const changeFontSize = (delta: number) => {
  const n = Math.max(10, Math.min(24, globalStore.termFontSize + delta))
  if (n === globalStore.termFontSize) return
  appearance.setLiveOverride('termFontSize', n)
}

// 主题/字体联动
watch(() => globalStore.termTheme, () => {
  const theme = applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity)
  tabs.value.forEach(t => { if (t.terminal) t.terminal.options.theme = theme })
})
watch(() => globalStore.termBgOpacity, () => {
  const theme = applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity)
  tabs.value.forEach(t => { if (t.terminal) t.terminal.options.theme = theme })
})
watch(() => globalStore.termFont, () => {
  const font = getTermFontByKey(globalStore.termFont)
  tabs.value.forEach(t => { if (t.terminal) { t.terminal.options.fontFamily = font; t.fitAddon?.fit() } })
})
watch(() => globalStore.termFontSize, (size) => {
  tabs.value.forEach(t => { if (t.terminal) { t.terminal.options.fontSize = size; t.fitAddon?.fit() } })
})

// 显示时自动打开第一个终端；最小化恢复时 refit
watch(() => globalStore.floatTermVisible, async (visible) => {
  if (!visible) return
  if (sessionStore.surface === 'float') await ensureSessionTabs()
  if (tabs.value.length === 0 && !globalStore.terminalTrigger) await addLocalTab()
  if (!globalStore.floatTermMinimized) setTimeout(fitActive, 150)
})
watch(() => sessionStore.surface, (surface) => {
  if (surface !== 'page') return
  for (let i = tabs.value.length - 1; i >= 0; i--) {
    if (tabs.value[i].locked) closeTab(i)
  }
  if (tabs.value.length === 0) globalStore.floatTermVisible = false
})
watch(() => globalStore.floatTermMinimized, (min) => {
  if (!min) setTimeout(fitActive, 150)
})

watch(() => globalStore.terminalTrigger, async (trigger) => {
  if (trigger) {
    const cwd = normalizeTerminalCwd(trigger.cwd)
    globalStore.terminalTrigger = null // reset trigger
    const tab = createTab(cwd.split('/').pop() || cwd || '终端', cwd)
    tabs.value.push(tab)
    activeTab.value = tab.id
    await createTerminal(tab)
  }
})

let sessionTimer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
  window.addEventListener('resize', updateStyle)
  sessionStore.refresh().then(() => {
    if (sessionStore.surface === 'float' && sessionStore.items.length) {
      globalStore.floatTermVisible = true
      globalStore.floatTermMinimized = false
    }
  }).catch(() => {})
  sessionTimer = setInterval(() => sessionStore.refresh().catch(() => {}), 15000)
})

onBeforeUnmount(() => {
  if (sessionTimer) clearInterval(sessionTimer)
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
  window.removeEventListener('resize', updateStyle)
  tabs.value.forEach(t => {
    t.closing = true
    clearReconnectTimer(t)
    t.ws?.close(); t.terminal?.dispose(); t._observer?.disconnect()
  })
})
</script>

<style lang="scss" scoped>
.float-term-panel {
  position: fixed;
  z-index: 3000;
  display: flex;
  flex-direction: column;
  background: var(--xp-terminal-bg);
  border: 1px solid var(--xp-accent-muted);
  border-radius: var(--xp-radius);
  box-shadow: 0 20px 60px rgba(0,0,0,0.6), var(--xp-accent-glow);
  overflow: hidden;
  user-select: none;
}

.float-term-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  height: 36px;
  background: color-mix(in srgb, var(--xp-bg-elevated) 88%, transparent);
  border-bottom: 1px solid var(--xp-border-light);
  cursor: move;
  flex-shrink: 0;
}

.float-term-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--xp-accent);

  .tab-count {
    font-size: 10px;
    color: var(--xp-text-muted);
    font-weight: 400;
  }

  .conn-state {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: 4px;
    font-size: 10px;
    color: var(--xp-text-muted);
    font-weight: 400;

    .conn-dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      background: var(--xp-warning);
    }

    &.connected .conn-dot { background: var(--xp-success); box-shadow: 0 0 6px color-mix(in srgb, var(--xp-success) 75%, transparent); }
    &.disconnected .conn-dot { background: var(--xp-danger); box-shadow: 0 0 6px color-mix(in srgb, var(--xp-danger) 65%, transparent); }
    &.reconnecting .conn-dot,
    &.connecting .conn-dot { background: var(--xp-warning); box-shadow: 0 0 6px color-mix(in srgb, var(--xp-warning) 65%, transparent); }
  }
}

.float-term-actions {
  display: flex;
  align-items: center;
  gap: 6px;

  .fs-label {
    font-size: 10px;
    color: var(--xp-text-muted);
    font-family: monospace;
    min-width: 16px;
    text-align: center;
  }

  .action-btn {
    color: var(--xp-text-muted);
    cursor: pointer;
    padding: 3px;
    border-radius: var(--xp-radius-sm);
    transition: all 0.15s;

    &:hover { color: var(--xp-accent); background: var(--xp-accent-muted); }
    &.close-btn:hover { color: var(--xp-danger); background: color-mix(in srgb, var(--xp-danger) 16%, transparent); }
  }
}

.float-term-tabs {
  display: flex;
  gap: 2px;
  padding: 4px 8px 0;
  background: color-mix(in srgb, var(--xp-bg-elevated) 70%, transparent);
  flex-shrink: 0;

  .ft-tab {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 3px 10px;
    font-size: 11px;
    border-radius: var(--xp-radius-sm) var(--xp-radius-sm) 0 0;
    cursor: pointer;
    color: var(--xp-text-muted);
    background: color-mix(in srgb, var(--xp-bg-surface) 80%, transparent);
    transition: color 0.15s, background 0.15s;

    &.active { color: var(--xp-accent); background: var(--xp-accent-muted); }
    &:hover:not(.active) { color: var(--xp-text-secondary); background: var(--xp-accent-muted); }

    .ft-tab-dot {
      width: 5px;
      height: 5px;
      border-radius: 50%;
      background: var(--xp-success);
      flex-shrink: 0;
    }

    &.disconnected .ft-tab-dot { background: var(--xp-danger); }
    &.reconnecting .ft-tab-dot { background: var(--xp-warning); }

    .ft-tab-close {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 16px;
      height: 16px;
      margin-left: 6px;
      padding: 0;
      border: 0;
      border-radius: var(--xp-radius-sm);
      background: transparent;
      color: inherit;
      opacity: 0.5;
      cursor: pointer;
      &:hover { opacity: 1; color: var(--xp-danger); }
    }
  }
}

.float-term-body {
  flex: 1;
  min-height: 0;
  position: relative;
  padding: 0;
  overflow: hidden;
  user-select: text;

  .ft-term-instance {
    position: absolute;
    inset: 0;
    display: none;

    &.active { display: block; }

    :deep(.xterm) { height: 100%; }
  }

  .ft-empty {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    color: var(--xp-text-muted);
    font-size: 12px;
  }
}

.float-term-sessions {
  position: fixed;
  right: 24px;
  bottom: 0;
  z-index: 3000;
  max-width: min(560px, calc(100vw - 32px));
  padding: 8px 12px;
  background: var(--xp-terminal-bg);
  border: 1px solid var(--xp-accent-muted);
  border-bottom: none;
  border-radius: var(--xp-radius-sm) var(--xp-radius-sm) 0 0;
  box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.4);
  color: var(--xp-accent);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.ft-tab-lock {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  padding: 0;
  border: 0;
  border-radius: var(--xp-radius-sm);
  background: transparent;
  color: var(--xp-text-muted);
  cursor: pointer;

  &.locked { color: var(--xp-accent); }
  &:hover { background: var(--xp-accent-muted); color: var(--xp-accent); }
}

.action-text {
  border: 0;
  background: transparent;
  color: var(--xp-text-muted);
  font: inherit;
  font-size: 11px;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: var(--xp-radius-sm);

  &:hover { color: var(--xp-accent); background: var(--xp-accent-muted); }
  &.danger:hover { color: var(--xp-danger); background: color-mix(in srgb, var(--xp-danger) 12%, transparent); }
}

// 最小化小条
.float-term-bar {
  position: fixed;
  bottom: 0;
  right: 24px;
  z-index: 3000;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  height: 32px;
  background: var(--xp-terminal-bg);
  border: 1px solid var(--xp-accent-muted);
  border-bottom: none;
  border-radius: var(--xp-radius-sm) var(--xp-radius-sm) 0 0;
  font-size: 12px;
  color: var(--xp-accent);
  cursor: pointer;
  box-shadow: 0 -4px 20px rgba(0,0,0,0.4);
  user-select: none;
  transition: background 0.15s;

  &:hover { background: var(--xp-accent-muted); }

  .term-bar-dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: var(--xp-success);
    box-shadow: 0 0 6px color-mix(in srgb, var(--xp-success) 80%, transparent);
    flex-shrink: 0;
  }

  .bar-close {
    margin-left: 4px;
    color: var(--xp-text-muted);
    &:hover { color: var(--xp-danger); }
  }
}

</style>

<style lang="scss">
/* The float panel is z-index 3000. Element Plus message boxes start near 2000, so a confirm opened from a tab sits under the panel. */
.el-overlay.is-message-box {
  z-index: 4100 !important;
}
</style>

<style lang="scss" scoped>
// 调整大小手柄
.resize-handle {
  position: absolute;
  z-index: 10;

  &.resize-se {
    bottom: 0; right: 0;
    width: 14px; height: 14px;
    cursor: se-resize;
  }
  &.resize-e {
    top: 36px; right: 0;
    width: 6px;
    bottom: 14px;
    cursor: e-resize;
  }
  &.resize-s {
    bottom: 0; left: 14px; right: 14px;
    height: 6px;
    cursor: s-resize;
  }
}
</style>
