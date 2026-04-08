<template>
  <div class="terminal-page">
    <!-- 顶部导航栏 -->
    <div class="terminal-header">
      <div class="header-left">
        <el-radio-group v-model="currentView" size="small" class="view-switcher">
          <el-radio-button value="terminal">
            <el-icon><Monitor /></el-icon>
            <span>{{ $t('terminal.title') }}</span>
          </el-radio-button>
          <el-radio-button value="hosts">
            <el-icon><Connection /></el-icon>
            <span>{{ $t('terminal.hostManage') }}</span>
          </el-radio-button>
          <el-radio-button value="commands">
            <el-icon><Promotion /></el-icon>
            <span>{{ $t('terminal.quickCommand') }}</span>
          </el-radio-button>
        </el-radio-group>
      </div>
      <div class="header-right" v-if="currentView === 'terminal'">
        <div class="term-settings">
          <el-tooltip :content="$t('terminal.fontSize')" placement="bottom">
            <div class="font-size-control">
              <el-icon :size="12" @click="changeFontSize(-1)" class="fs-btn"><Minus /></el-icon>
              <span class="fs-value">{{ termFontSize }}</span>
              <el-icon :size="12" @click="changeFontSize(1)" class="fs-btn"><Plus /></el-icon>
            </div>
          </el-tooltip>
        </div>
        <el-button size="small" type="primary" plain @click="showCommandPalette = true">
          <el-icon><Search /></el-icon>
          {{ $t('terminal.quickCommand') }}
          <span class="shortcut-hint">Ctrl+Shift+P</span>
        </el-button>
        <el-popover placement="bottom" :width="360" trigger="click">
          <template #reference>
            <el-button size="small" type="info" plain>
              <el-icon><Promotion /></el-icon>
              {{ $t('terminal.batchInput') }}
            </el-button>
          </template>
          <div class="batch-input-panel">
            <el-input
              v-model="batchCommand"
              :placeholder="$t('terminal.batchInputPlaceholder')"
              type="textarea"
              :rows="3"
              resize="none"
            />
            <div class="batch-targets">
              <el-checkbox v-model="batchAllTerminals" @change="toggleBatchAll">
                {{ $t('terminal.allTerminals') }}
              </el-checkbox>
              <div class="batch-tab-list">
                <el-checkbox
                  v-for="tab in tabs"
                  :key="tab.id"
                  :model-value="batchTargets.has(tab.id)"
                  @change="(v: boolean | string | number) => toggleBatchTarget(tab.id, v as boolean)"
                  size="small"
                >
                  {{ tab.title }}
                </el-checkbox>
              </div>
            </div>
            <el-button type="primary" size="small" class="batch-send-btn" @click="sendBatchCommand">
              {{ $t('terminal.batchSend') }}
            </el-button>
          </div>
        </el-popover>
      </div>
    </div>

    <!-- 终端视图 -->
    <div v-show="currentView === 'terminal'" class="terminal-main">
      <div class="terminal-sidebar">
        <div class="sidebar-section">
          <div class="section-title">
            <span>{{ $t('terminal.localTerminal') }}</span>
          </div>
          <div
            class="host-item local"
            @click="addLocalTab"
          >
            <el-icon :size="16"><Monitor /></el-icon>
            <span>{{ $t('terminal.connLocal') }}</span>
            <el-icon class="add-icon" :size="14"><Plus /></el-icon>
          </div>
        </div>
        <div class="sidebar-section">
          <div class="section-title">
            <span>{{ $t('terminal.remoteTerminal') }}</span>
            <el-button
              link
              type="primary"
              size="small"
              @click="currentView = 'hosts'"
            >
              <el-icon :size="14"><Setting /></el-icon>
            </el-button>
          </div>
          <div v-if="hostTree.length === 0" class="empty-hosts">
            <span class="empty-text">{{ $t('host.noHost') }}</span>
          </div>
          <template v-for="group in hostTree" :key="group.label">
            <div class="group-label" v-if="group.children && group.children.length > 0">
              {{ group.label }}
            </div>
            <div
              v-for="host in group.children"
              :key="host.id"
              class="host-item"
              @click="addRemoteTab(host.id, host.label)"
            >
              <el-icon :size="16"><Connection /></el-icon>
              <span class="host-name">{{ host.label }}</span>
            </div>
          </template>
        </div>
        <div class="sidebar-section" v-if="commandList.length > 0">
          <div class="section-title">
            <span>{{ $t('terminal.quickCommand') }}</span>
          </div>
          <div
            v-for="cmd in commandList"
            :key="cmd.id"
            class="cmd-item"
            @click="executeCommand(cmd.command)"
            :title="cmd.command"
          >
            <el-icon :size="14"><Promotion /></el-icon>
            <span>{{ cmd.name }}</span>
          </div>
        </div>
      </div>

      <div class="terminal-content">
        <div class="terminal-tabs">
          <div
            v-for="(tab, idx) in tabs"
            :key="tab.id"
            class="terminal-tab"
            :class="{ active: activeTab === tab.id }"
            @click="switchTab(tab.id)"
          >
            <el-icon :size="14" :class="{ remote: !!tab.hostId }">
              <Connection v-if="tab.hostId" />
              <Monitor v-else />
            </el-icon>
            <span>{{ tab.title }}</span>
            <span v-if="tab.hostId" class="tab-badge">SSH</span>
            <el-icon
              v-if="tabs.length > 1"
              class="tab-close"
              :size="12"
              @click.stop="closeTab(idx)"
            >
              <Close />
            </el-icon>
          </div>
          <div class="terminal-tab add-tab" @click="addLocalTab">
            <el-icon :size="14"><Plus /></el-icon>
          </div>
        </div>
        <div class="terminal-container" @click="focusActiveTerminal">
          <div
            v-for="tab in tabs"
            :key="tab.id"
            :ref="(el: unknown) => setTermRef(tab.id, el as HTMLElement)"
            class="terminal-instance"
            :class="{ active: activeTab === tab.id }"
          />
        </div>
      </div>
    </div>

    <!-- 主机管理视图 -->
    <div v-if="currentView === 'hosts'" class="sub-view">
      <HostManage @connect="handleHostConnect" @back="currentView = 'terminal'" />
    </div>

    <!-- 快速命令视图 -->
    <div v-if="currentView === 'commands'" class="sub-view">
      <CommandManage @execute="handleCommandExecute" @back="currentView = 'terminal'" />
    </div>

    <!-- 命令面板 (Ctrl+P) -->
    <Teleport to="body">
      <div v-if="showCommandPalette" class="command-palette-mask" @click="showCommandPalette = false">
        <div class="command-palette" @click.stop>
          <el-input
            ref="paletteInputRef"
            v-model="paletteSearch"
            :placeholder="$t('terminal.searchCommand')"
            size="large"
            clearable
            @keydown.enter="executePaletteCommand"
            @keydown.escape="showCommandPalette = false"
            @keydown.down.prevent="paletteMoveDown"
            @keydown.up.prevent="paletteMoveUp"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <div class="palette-results" v-if="filteredCommands.length > 0">
            <div
              v-for="(cmd, idx) in filteredCommands"
              :key="cmd.id"
              class="palette-item"
              :class="{ active: paletteIndex === idx }"
              @click="executePaletteItem(cmd)"
              @mouseenter="paletteIndex = idx"
            >
              <span class="palette-name">{{ cmd.name }}</span>
              <code class="palette-cmd">{{ cmd.command }}</code>
            </div>
          </div>
          <div v-else-if="paletteSearch" class="palette-empty">
            {{ $t('command.noCommand') }}
          </div>
          <div class="palette-hint">
            <span>↑↓ {{ $t('terminal.navigate') }}</span>
            <span>Enter {{ $t('terminal.executeCmd') }}</span>
            <span>Esc {{ $t('commons.close') }}</span>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { getHostTree, getCommandTree } from '@/api/modules/host'
import { ElMessage } from 'element-plus'
import type { HostTreeGroup, CommandItem, CommandGroup } from '@/api/interface'
import { Search, Minus } from '@element-plus/icons-vue'
import HostManage from './host/index.vue'
import CommandManage from './command/index.vue'

interface TermTab {
  id: string
  title: string
  hostId?: number
  terminal?: Terminal
  fitAddon?: FitAddon
  ws?: WebSocket
  _resizeObserver?: ResizeObserver
}

const { t } = useI18n()
const currentView = ref<'terminal' | 'hosts' | 'commands'>('terminal')
const tabs = ref<TermTab[]>([])
const activeTab = ref('')
const termFontSize = computed(() => globalStore.termFontSize)

const changeFontSize = (delta: number) => {
  const newSize = Math.max(10, Math.min(24, globalStore.termFontSize + delta))
  if (newSize === globalStore.termFontSize) return
  globalStore.termFontSize = newSize
  for (const tab of tabs.value) {
    if (tab.terminal) {
      tab.terminal.options.fontSize = newSize
      tab.fitAddon?.fit()
    }
  }
}

watch(() => globalStore.termTheme, () => {
  const theme = applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity)
  for (const tab of tabs.value) {
    if (tab.terminal) tab.terminal.options.theme = theme
  }
})

watch(() => globalStore.termFont, () => {
  const font = getTermFontByKey(globalStore.termFont)
  for (const tab of tabs.value) {
    if (tab.terminal) { tab.terminal.options.fontFamily = font; tab.fitAddon?.fit() }
  }
})

watch(() => globalStore.termBgOpacity, () => {
  const theme = applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity)
  for (const tab of tabs.value) {
    if (tab.terminal) tab.terminal.options.theme = theme
  }
})
const termRefs: Record<string, HTMLElement | null> = {}
let tabCounter = 0
const batchCommand = ref('')
const hostTree = ref<HostTreeGroup[]>([])
const commandList = ref<CommandItem[]>([])

// 批量输入目标选择
const batchTargets = ref<Set<string>>(new Set())
const batchAllTerminals = ref(true)

const toggleBatchAll = (val: boolean | string | number) => {
  batchTargets.value.clear()
  if (val) {
    for (const tab of tabs.value) batchTargets.value.add(tab.id)
  }
}

const toggleBatchTarget = (id: string, checked: boolean) => {
  if (checked) {
    batchTargets.value.add(id)
  } else {
    batchTargets.value.delete(id)
  }
  batchAllTerminals.value = batchTargets.value.size === tabs.value.length
}

// 命令面板
const showCommandPalette = ref(false)
const paletteSearch = ref('')
const paletteIndex = ref(0)
const paletteInputRef = ref<{ focus: () => void } | null>(null)

const filteredCommands = computed(() => {
  const q = paletteSearch.value.toLowerCase()
  if (!q) return commandList.value
  return commandList.value.filter((cmd) =>
    cmd.name.toLowerCase().includes(q) || cmd.command.toLowerCase().includes(q)
  )
})

watch(showCommandPalette, (val) => {
  if (val) {
    paletteSearch.value = ''
    paletteIndex.value = 0
    nextTick(() => paletteInputRef.value?.focus())
  } else {
    nextTick(() => focusActiveTerminal())
  }
})

watch(paletteSearch, () => {
  paletteIndex.value = 0
})

const paletteMoveDown = () => {
  if (paletteIndex.value < filteredCommands.value.length - 1) paletteIndex.value++
}
const paletteMoveUp = () => {
  if (paletteIndex.value > 0) paletteIndex.value--
}

const executePaletteCommand = () => {
  const cmds = filteredCommands.value
  if (cmds.length > 0 && paletteIndex.value < cmds.length) {
    executePaletteItem(cmds[paletteIndex.value])
  }
}

const executePaletteItem = (cmd: CommandItem) => {
  showCommandPalette.value = false
  executeCommand(cmd.command)
}

// 全局快捷键 Ctrl+Shift+P（避免与 vim/tmux 等程序的 Ctrl+P 冲突）
const handleGlobalKeydown = (e: KeyboardEvent) => {
  if (e.ctrlKey && e.shiftKey && e.key.toLowerCase() === 'p' && currentView.value === 'terminal') {
    e.preventDefault()
    showCommandPalette.value = !showCommandPalette.value
  }
}

const setTermRef = (id: string, el: HTMLElement | null) => {
  if (el) termRefs[id] = el
}

const focusActiveTerminal = () => {
  const tab = tabs.value.find((t) => t.id === activeTab.value)
  tab?.terminal?.focus()
}

const getWsUrl = (hostId?: number) => {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = sessionStorage.getItem('token')
  let url = `${proto}//${location.host}/api/v1/terminal?token=${token}`
  if (hostId) url += `&id=${hostId}`
  return url
}

import { getTermThemeByKey, getTermFontByKey, applyBgOpacity } from '@/utils/terminal-theme'
import { useGlobalStore } from '@/store/modules/global'

const globalStore = useGlobalStore()

const sendResize = (ws: WebSocket, rows: number, cols: number) => {
  const resizeData = JSON.stringify({ rows, cols })
  const msg = new Uint8Array(1 + resizeData.length)
  msg[0] = 1
  for (let i = 0; i < resizeData.length; i++) {
    msg[i + 1] = resizeData.charCodeAt(i)
  }
  ws.send(msg)
}

const createTerminal = async (tab: TermTab) => {
  await nextTick()
  const el = termRefs[tab.id]
  if (!el) return

  const terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    fontSize: globalStore.termFontSize,
    fontFamily: getTermFontByKey(globalStore.termFont),
    theme: applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity),
    scrollback: 10000,
    allowProposedApi: true,
  })

  // 自定义按键处理：解决 vim/tmux 等程序的快捷键冲突
  terminal.attachCustomKeyEventHandler((event: KeyboardEvent) => {
    // Ctrl+Shift+C/V: 让浏览器处理（复制/粘贴）
    if (event.ctrlKey && event.shiftKey && ['c', 'v'].includes(event.key.toLowerCase())) {
      return false
    }
    // F11: 让浏览器处理全屏切换
    if (event.key === 'F11') return false
    // F12: 让浏览器处理开发者工具
    if (event.key === 'F12') return false
    // 其他所有按键（包括 Esc, Ctrl+C, Ctrl+Z, 方向键等）都交给终端
    return true
  })

  const fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(el)

  tab.terminal = terminal
  tab.fitAddon = fitAddon

  // 使用 ResizeObserver 精确监听容器尺寸变化，替代 window resize
  const observer = new ResizeObserver(() => {
    if (activeTab.value === tab.id && tab.fitAddon) {
      try { tab.fitAddon.fit() } catch { /* ignore fit errors during teardown */ }
    }
  })
  observer.observe(el)
  tab._resizeObserver = observer

  // 延迟首次 fit，确保 DOM 完全渲染
  setTimeout(() => {
    try { fitAddon.fit() } catch { /* */ }
    terminal.focus()
  }, 100)

  const ws = new WebSocket(getWsUrl(tab.hostId))
  ws.binaryType = 'arraybuffer'
  tab.ws = ws

  ws.onopen = () => {
    sendResize(ws, terminal.rows, terminal.cols)
    terminal.focus()
  }

  ws.onmessage = (e: MessageEvent) => {
    if (e.data instanceof ArrayBuffer) {
      terminal.write(new Uint8Array(e.data))
    } else {
      terminal.write(e.data)
    }
  }

  ws.onclose = () => {
    terminal.write(`\r\n\x1b[31m${t('terminal.disconnected')}\x1b[0m\r\n`)
  }

  ws.onerror = () => {
    terminal.write(`\r\n\x1b[31m${t('terminal.connError')}\x1b[0m\r\n`)
  }

  terminal.onData((data: string) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  terminal.onResize(({ rows, cols }) => {
    if (ws.readyState === WebSocket.OPEN) {
      sendResize(ws, rows, cols)
    }
  })
}

const addLocalTab = async () => {
  tabCounter++
  const tab: TermTab = {
    id: `term-${tabCounter}`,
    title: `${t('terminal.localTerminal')} ${tabCounter}`,
  }
  tabs.value.push(tab)
  activeTab.value = tab.id
  currentView.value = 'terminal'
  await createTerminal(tab)
}

const addRemoteTab = async (hostId: number, label: string) => {
  tabCounter++
  const tab: TermTab = {
    id: `term-${tabCounter}`,
    title: label,
    hostId,
  }
  tabs.value.push(tab)
  activeTab.value = tab.id
  currentView.value = 'terminal'
  await createTerminal(tab)
}

const switchTab = (id: string) => {
  activeTab.value = id
  nextTick(() => {
    const tab = tabs.value.find((t) => t.id === id)
    if (tab?.fitAddon) tab.fitAddon.fit()
    tab?.terminal?.focus()
  })
}

const closeTab = (idx: number) => {
  const tab = tabs.value[idx]
  if (tab.ws) tab.ws.close()
  if (tab.terminal) tab.terminal.dispose()
  if (tab._resizeObserver) tab._resizeObserver.disconnect()
  tabs.value.splice(idx, 1)
  if (activeTab.value === tab.id && tabs.value.length > 0) {
    activeTab.value = tabs.value[Math.min(idx, tabs.value.length - 1)].id
  }
}

const sendBatchCommand = () => {
  if (!batchCommand.value.trim()) return
  const cmd = batchCommand.value + '\n'
  let count = 0
  for (const tab of tabs.value) {
    if (batchAllTerminals.value || batchTargets.value.has(tab.id)) {
      if (tab.ws && tab.ws.readyState === WebSocket.OPEN) {
        tab.ws.send(cmd)
        count++
      }
    }
  }
  batchCommand.value = ''
  if (count > 0) {
    ElMessage.success(t('terminal.batchSent', { count }))
  } else {
    ElMessage.warning(t('terminal.noActiveTerminal'))
  }
}

const executeCommand = (cmd: string) => {
  const tab = tabs.value.find((t) => t.id === activeTab.value)
  if (tab?.ws && tab.ws.readyState === WebSocket.OPEN) {
    tab.ws.send(cmd + '\n')
  } else {
    ElMessage.warning(t('terminal.noActiveTerminal'))
  }
}

const handleHostConnect = (hostId: number, label: string) => {
  addRemoteTab(hostId, label)
}

const handleCommandExecute = (cmd: string) => {
  currentView.value = 'terminal'
  nextTick(() => executeCommand(cmd))
}

const loadHostTree = async () => {
  try {
    const res = await getHostTree()
    hostTree.value = res.data || []
  } catch {
    hostTree.value = []
  }
}

const loadCommands = async () => {
  try {
    const res = await getCommandTree()
    const tree: CommandGroup[] = res.data || []
    const flat: CommandItem[] = []
    for (const g of tree) {
      if (g.children) {
        flat.push(...g.children)
      }
    }
    commandList.value = flat
  } catch {
    commandList.value = []
  }
}

watch(currentView, (val) => {
  if (val === 'terminal') {
    loadHostTree()
    loadCommands()
    nextTick(() => {
      const tab = tabs.value.find((t) => t.id === activeTab.value)
      if (tab?.fitAddon) tab.fitAddon.fit()
      // 延迟聚焦，确保 DOM 已切换完毕
      setTimeout(() => tab?.terminal?.focus(), 50)
    })
  }
})

onMounted(() => {
  loadHostTree()
  loadCommands()
  addLocalTab()
  document.addEventListener('keydown', handleGlobalKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
  for (const tab of tabs.value) {
    if (tab.ws) tab.ws.close()
    if (tab.terminal) tab.terminal.dispose()
    if (tab._resizeObserver) tab._resizeObserver.disconnect()
  }
})
</script>

<style lang="scss" scoped>
.terminal-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - var(--xp-header-height) - 40px);
}

.terminal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0 12px 0;
  flex-shrink: 0;

  .view-switcher {
    :deep(.el-radio-button__inner) {
      display: flex;
      align-items: center;
      gap: 6px;
      background: var(--xp-bg-surface);
      border-color: var(--xp-border);
      color: var(--xp-text-secondary);
      font-size: 13px;
    }
    :deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
      background: var(--xp-accent-muted);
      border-color: var(--xp-accent);
      color: var(--xp-accent);
      box-shadow: -1px 0 0 0 var(--xp-accent);
    }
  }
}

.term-settings {
  display: flex;
  align-items: center;
  gap: 8px;
}

.font-size-control {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-sm);
  padding: 2px 6px;
  height: 28px;

  .fs-btn {
    cursor: pointer;
    color: var(--xp-text-muted);
    padding: 2px;
    border-radius: 3px;
    transition: all 0.15s;

    &:hover {
      color: var(--xp-accent);
      background: var(--xp-accent-muted);
    }
  }

  .fs-value {
    font-size: 11px;
    font-family: 'JetBrains Mono', monospace;
    color: var(--xp-text-secondary);
    min-width: 18px;
    text-align: center;
  }
}

.shortcut-hint {
  font-size: 10px;
  padding: 1px 5px;
  margin-left: 4px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.08);
  color: var(--xp-text-muted);
  font-family: monospace;
}

.batch-input-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;

  .batch-targets {
    font-size: 12px;
    .batch-tab-list {
      display: flex;
      flex-direction: column;
      gap: 2px;
      padding-left: 24px;
      margin-top: 4px;
    }
  }

  .batch-send-btn {
    align-self: flex-end;
  }
}

.terminal-main {
  display: flex;
  flex: 1;
  gap: 12px;
  min-height: 0;
}

.terminal-sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  overflow-y: auto;
  padding: 8px;

  .sidebar-section {
    margin-bottom: 12px;
  }

  .section-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 8px;
    font-size: 11px;
    font-weight: 600;
    color: var(--xp-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .group-label {
    padding: 4px 8px;
    font-size: 11px;
    color: var(--xp-text-muted);
    margin-top: 4px;
  }

  .host-item,
  .cmd-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 10px;
    font-size: 13px;
    color: var(--xp-text-secondary);
    border-radius: var(--xp-radius-sm);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
      color: var(--xp-accent);
    }

    .host-name {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      flex: 1;
    }

    .add-icon {
      margin-left: auto;
      opacity: 0;
      transition: opacity 0.2s;
    }

    &:hover .add-icon {
      opacity: 1;
    }

    &.local {
      border: 1px dashed var(--xp-border);
      margin-bottom: 4px;

      &:hover {
        border-color: var(--xp-accent);
      }
    }
  }

  .cmd-item {
    font-size: 12px;
  }

  .empty-hosts {
    padding: 12px 10px;
    text-align: center;

    .empty-text {
      font-size: 12px;
      color: var(--xp-text-muted);
    }
  }
}

.terminal-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.terminal-tabs {
  display: flex;
  align-items: center;
  gap: 1px;
  padding: 4px 4px 0;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-bottom: none;
  border-radius: var(--xp-radius-sm) var(--xp-radius-sm) 0 0;
  overflow-x: auto;
  flex-shrink: 0;

  &::-webkit-scrollbar {
    height: 0;
  }

  .terminal-tab {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    font-size: 13px;
    color: var(--xp-text-muted);
    cursor: pointer;
    border-bottom: 2px solid transparent;
    border-radius: var(--xp-radius-sm) var(--xp-radius-sm) 0 0;
    transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
    white-space: nowrap;
    user-select: none;

    &:hover {
      color: var(--xp-text-secondary);
      background: rgba(255, 255, 255, 0.03);
    }

    &.active {
      color: var(--xp-accent);
      border-bottom-color: var(--xp-accent);
      background: var(--xp-accent-muted);
      font-weight: 500;
    }

    .remote {
      color: var(--xp-accent-secondary);
    }

    .tab-badge {
      font-size: 10px;
      padding: 1px 5px;
      border-radius: 3px;
      background: rgba(129, 140, 248, 0.15);
      color: var(--xp-accent-secondary);
      font-weight: 600;
    }

    .tab-close {
      margin-left: 4px;
      border-radius: 50%;
      padding: 2px;

      &:hover {
        background: rgba(255, 255, 255, 0.1);
        color: var(--xp-danger);
      }
    }
  }

  .add-tab {
    padding: 8px 10px;
    color: var(--xp-text-muted);

    &:hover {
      color: var(--xp-accent);
    }
  }
}

.terminal-container {
  flex: 1;
  background: var(--xp-terminal-bg);
  border: 1px solid var(--xp-border-light);
  border-top: none;
  border-radius: 0 0 var(--xp-radius-sm) var(--xp-radius-sm);
  overflow: hidden;
  position: relative;
  padding: 4px;
  box-shadow: inset 0 2px 8px rgba(0, 0, 0, 0.15);

  .terminal-instance {
    position: absolute;
    inset: 4px;
    display: none;

    &.active {
      display: block;
    }

    :deep(.xterm) {
      height: 100%;
    }

    :deep(.xterm-screen) {
      height: 100% !important;
    }
  }
}

.sub-view {
  flex: 1;
  min-height: 0;
}

/* ===== 命令面板 ===== */
.command-palette-mask {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(4px);
  display: flex;
  justify-content: center;
  padding-top: 15vh;
}

.command-palette {
  width: 540px;
  max-height: 440px;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-lg);
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  align-self: flex-start;

  :deep(.el-input__wrapper) {
    border-radius: 12px 12px 0 0;
    box-shadow: none !important;
    padding: 8px 16px;
    background: transparent;
  }
}

.palette-results {
  flex: 1;
  overflow-y: auto;
  padding: 4px;
  max-height: 300px;
}

.palette-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 14px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.15s;

  &.active,
  &:hover {
    background: var(--xp-accent-muted);
  }

  .palette-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--xp-text-primary);
  }

  .palette-cmd {
    font-size: 12px;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    color: var(--xp-accent);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.palette-empty {
  padding: 24px;
  text-align: center;
  color: var(--xp-text-muted);
  font-size: 13px;
}

.palette-hint {
  display: flex;
  justify-content: center;
  gap: 16px;
  padding: 8px;
  border-top: 1px solid var(--xp-border-light);
  font-size: 11px;
  color: var(--xp-text-muted);
}

:deep(.xterm-viewport) {
  &::-webkit-scrollbar {
    width: 6px;
  }
  &::-webkit-scrollbar-thumb {
    background: rgba(148, 163, 184, 0.15);
    border-radius: 3px;
  }
}
</style>
