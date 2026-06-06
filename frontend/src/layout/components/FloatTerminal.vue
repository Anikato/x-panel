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
      <div v-if="tabs.length > 1" class="float-term-tabs">
        <div
          v-for="(tab, idx) in tabs"
          :key="tab.id"
          class="ft-tab"
          :class="{ active: activeTab === tab.id, disconnected: tab.status === 'disconnected', reconnecting: tab.status === 'reconnecting' }"
          @click="switchTab(tab.id)"
        >
          <span class="ft-tab-dot" />
          <span>{{ tab.title }}</span>
          <el-icon :size="10" class="ft-tab-close" @click.stop="closeTab(idx)"><Close /></el-icon>
        </div>
      </div>

      <!-- 终端区域 -->
      <div class="float-term-body" ref="bodyRef">
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
import { getTermThemeByKey, getTermFontByKey, applyBgOpacity } from '@/utils/terminal-theme'
import { buildInitialCwdCommand, normalizeTerminalCwd } from '@/utils/terminal-cwd'
import { getToken } from '@/utils/auth'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const globalStore = useGlobalStore()
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
  initialCwd?: string
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

const getWsUrl = (hostId?: number) => {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = getToken()
  let url = `${proto}//${location.host}/api/v1/terminal?token=${token}`
  if (hostId) url += `&id=${hostId}`
  return url
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

const connectWebSocket = (tab: TermTab) => {
  if (!tab.terminal) return
  clearReconnectTimer(tab)
  tab.closing = false
  tab.status = tab.reconnectAttempts > 0 ? 'reconnecting' : 'connecting'

  if (tab.ws && tab.ws.readyState !== WebSocket.CLOSED) {
    tab.ws.onclose = null
    tab.ws.onerror = null
    tab.ws.onmessage = null
    tab.ws.onopen = null
    tab.ws.close()
  }

  const term = tab.terminal
  const ws = new WebSocket(getWsUrl(tab.hostId))
  ws.binaryType = 'arraybuffer'
  tab.ws = ws

  ws.onopen = () => {
    tab.status = 'connected'
    tab.reconnectAttempts = 0
    tab.reconnectDelay = undefined
    sendResize(ws, term.rows, term.cols)
    term.focus()
    if (tab.initialCwd) {
      setTimeout(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          ws.send(buildInitialCwdCommand(tab.initialCwd))
        }
      }, 300)
      tab.initialCwd = undefined
    }
  }
  ws.onmessage = (e) => {
    if (e.data instanceof ArrayBuffer) term.write(new Uint8Array(e.data))
    else term.write(e.data)
  }
  ws.onclose = () => {
    if (tab.closing || !tabs.value.includes(tab)) return
    tab.status = 'disconnected'
    term.write(`\r\n\x1b[31m${t('terminal.disconnected')}\x1b[0m\r\n`)
    scheduleReconnect(tab)
  }
  ws.onerror = () => {
    if (!tab.closing) term.write(`\r\n\x1b[31m${t('terminal.connError')}\x1b[0m\r\n`)
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

  const term = new Terminal({
    cursorBlink: true, cursorStyle: 'bar',
    fontSize: globalStore.termFontSize,
    fontFamily: getTermFontByKey(globalStore.termFont),
    theme: applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity),
    scrollback: 10000, allowProposedApi: true,
  })

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

const addLocalTab = async () => {
  tabCounter++
  const tab: TermTab = {
    id: `ft-${tabCounter}`,
    title: `终端 ${tabCounter}`,
    status: 'connecting',
    reconnectAttempts: 0,
  }
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

const closeTab = (idx: number) => {
  const tab = tabs.value[idx]
  tab.closing = true
  clearReconnectTimer(tab)
  tab.ws?.close(); tab.terminal?.dispose(); tab._observer?.disconnect()
  tabs.value.splice(idx, 1)
  if (activeTab.value === tab.id && tabs.value.length > 0) {
    activeTab.value = tabs.value[Math.min(idx, tabs.value.length - 1)].id
    nextTick(() => { const t = tabs.value.find(x => x.id === activeTab.value); try { t?.fitAddon?.fit() } catch { /* */ } })
  }
  if (tabs.value.length === 0) globalStore.floatTermVisible = false
}

const changeFontSize = (delta: number) => {
  const n = Math.max(10, Math.min(24, globalStore.termFontSize + delta))
  if (n === globalStore.termFontSize) return
  globalStore.termFontSize = n
  for (const tab of tabs.value) { if (tab.terminal) { tab.terminal.options.fontSize = n; tab.fitAddon?.fit() } }
}

// 主题/字体联动
watch(() => globalStore.termTheme, () => {
  const theme = applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity)
  tabs.value.forEach(t => { if (t.terminal) t.terminal.options.theme = theme })
})
watch(() => globalStore.termFont, () => {
  const font = getTermFontByKey(globalStore.termFont)
  tabs.value.forEach(t => { if (t.terminal) { t.terminal.options.fontFamily = font; t.fitAddon?.fit() } })
})

// 显示时自动打开第一个终端；最小化恢复时 refit
watch(() => globalStore.floatTermVisible, async (visible) => {
  if (visible && tabs.value.length === 0) {
    await nextTick()
    await addLocalTab()
  }
  if (visible && !globalStore.floatTermMinimized) {
    setTimeout(fitActive, 150)
  }
})
watch(() => globalStore.floatTermMinimized, (min) => {
  if (!min) setTimeout(fitActive, 150)
})

watch(() => globalStore.terminalTrigger, async (trigger) => {
  if (trigger) {
    const cwd = normalizeTerminalCwd(trigger.cwd)
    globalStore.terminalTrigger = null // reset trigger
    tabCounter++
    const tab: TermTab = {
      id: `ft-${tabCounter}`,
      title: cwd.split('/').pop() || cwd || '终端',
      status: 'connecting',
      reconnectAttempts: 0,
      initialCwd: cwd,
    }
    tabs.value.push(tab)
    activeTab.value = tab.id
    await createTerminal(tab)
  }
})

onMounted(() => {
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
  window.addEventListener('resize', updateStyle)
})

onBeforeUnmount(() => {
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
  background: #0d1117;
  border: 1px solid var(--xp-accent-muted);
  border-radius: 10px;
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
  background: rgba(255,255,255,0.04);
  border-bottom: 1px solid rgba(255,255,255,0.06);
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
      background: #f59e0b;
    }

    &.connected .conn-dot { background: #22c55e; box-shadow: 0 0 6px rgba(34,197,94,0.75); }
    &.disconnected .conn-dot { background: #ef4444; box-shadow: 0 0 6px rgba(239,68,68,0.65); }
    &.reconnecting .conn-dot,
    &.connecting .conn-dot { background: #f59e0b; box-shadow: 0 0 6px rgba(245,158,11,0.65); }
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
    border-radius: 4px;
    transition: all 0.15s;

    &:hover { color: var(--xp-accent); background: var(--xp-accent-muted); }
    &.close-btn:hover { color: #ff6b6b; background: rgba(255,107,107,0.15); }
  }
}

.float-term-tabs {
  display: flex;
  gap: 2px;
  padding: 4px 8px 0;
  background: rgba(255,255,255,0.02);
  flex-shrink: 0;

  .ft-tab {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 3px 10px;
    font-size: 11px;
    border-radius: 4px 4px 0 0;
    cursor: pointer;
    color: var(--xp-text-muted);
    background: rgba(255,255,255,0.03);
    transition: all 0.15s;

    &.active { color: var(--xp-accent); background: var(--xp-accent-muted); }
    &:hover:not(.active) { color: var(--xp-text-secondary); background: rgba(255,255,255,0.06); }

    .ft-tab-dot {
      width: 5px;
      height: 5px;
      border-radius: 50%;
      background: #22c55e;
      flex-shrink: 0;
    }

    &.disconnected .ft-tab-dot { background: #ef4444; }
    &.reconnecting .ft-tab-dot { background: #f59e0b; }

    .ft-tab-close {
      opacity: 0.5;
      &:hover { opacity: 1; color: #ff6b6b; }
    }
  }
}

.float-term-body {
  flex: 1;
  min-height: 0;
  position: relative;
  padding: 6px;
  user-select: text;

  .ft-term-instance {
    position: absolute;
    inset: 6px;
    display: none;

    &.active { display: block; }

    :deep(.xterm) { height: 100%; }
    :deep(.xterm-viewport) { border-radius: 4px; }
  }
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
  background: #0d1117;
  border: 1px solid var(--xp-accent-muted);
  border-bottom: none;
  border-radius: 6px 6px 0 0;
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
    background: #4ade80;
    box-shadow: 0 0 6px #4ade80;
    flex-shrink: 0;
  }

  .bar-close {
    margin-left: 4px;
    color: var(--xp-text-muted);
    &:hover { color: #ff6b6b; }
  }
}

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
