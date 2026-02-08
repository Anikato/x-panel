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
        <el-popover placement="bottom" :width="280" trigger="click">
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
        <div class="terminal-container">
          <div
            v-for="tab in tabs"
            :key="tab.id"
            :ref="(el: any) => setTermRef(tab.id, el as HTMLElement)"
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { getHostTree, getCommandTree } from '@/api/modules/host'
import { ElMessage } from 'element-plus'
import HostManage from './host/index.vue'
import CommandManage from './command/index.vue'

interface TermTab {
  id: string
  title: string
  hostId?: number
  terminal?: Terminal
  fitAddon?: FitAddon
  ws?: WebSocket
  _resizeHandler?: () => void
}

const currentView = ref<'terminal' | 'hosts' | 'commands'>('terminal')
const tabs = ref<TermTab[]>([])
const activeTab = ref('')
const termRefs: Record<string, HTMLElement | null> = {}
let tabCounter = 0
const batchCommand = ref('')
const hostTree = ref<any[]>([])
const commandList = ref<any[]>([])

const setTermRef = (id: string, el: HTMLElement | null) => {
  if (el) termRefs[id] = el
}

const getWsUrl = (hostId?: number) => {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = sessionStorage.getItem('token')
  let url = `${proto}//${location.host}/api/v1/terminal?token=${token}`
  if (hostId) url += `&id=${hostId}`
  return url
}

const terminalTheme = {
  background: '#0b0e14',
  foreground: '#e6edf3',
  cursor: '#22d3ee',
  cursorAccent: '#0b0e14',
  selectionBackground: 'rgba(34, 211, 238, 0.2)',
  black: '#0b0e14',
  red: '#f87171',
  green: '#4ade80',
  yellow: '#fbbf24',
  blue: '#60a5fa',
  magenta: '#c084fc',
  cyan: '#22d3ee',
  white: '#e6edf3',
  brightBlack: '#475569',
  brightRed: '#fca5a5',
  brightGreen: '#86efac',
  brightYellow: '#fde68a',
  brightBlue: '#93c5fd',
  brightMagenta: '#d8b4fe',
  brightCyan: '#67e8f9',
  brightWhite: '#f8fafc',
}

const createTerminal = async (tab: TermTab) => {
  await nextTick()
  const el = termRefs[tab.id]
  if (!el) return

  const terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    fontSize: 14,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
    theme: terminalTheme,
    scrollback: 5000,
    allowProposedApi: true,
  })

  const fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(el)
  setTimeout(() => fitAddon.fit(), 50)

  tab.terminal = terminal
  tab.fitAddon = fitAddon

  const ws = new WebSocket(getWsUrl(tab.hostId))
  tab.ws = ws

  ws.onopen = () => {
    const resizeData = JSON.stringify({ rows: terminal.rows, cols: terminal.cols })
    const msg = new Uint8Array(1 + resizeData.length)
    msg[0] = 1
    for (let i = 0; i < resizeData.length; i++) {
      msg[i + 1] = resizeData.charCodeAt(i)
    }
    ws.send(msg)
  }

  ws.onmessage = (e: MessageEvent) => {
    terminal.write(e.data)
  }

  ws.onclose = () => {
    terminal.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n')
  }

  ws.onerror = () => {
    terminal.write('\r\n\x1b[31m连接错误\x1b[0m\r\n')
  }

  terminal.onData((data: string) => {
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  terminal.onResize(({ rows, cols }) => {
    if (ws.readyState === WebSocket.OPEN) {
      const resizeData = JSON.stringify({ rows, cols })
      const msg = new Uint8Array(1 + resizeData.length)
      msg[0] = 1
      for (let i = 0; i < resizeData.length; i++) {
        msg[i + 1] = resizeData.charCodeAt(i)
      }
      ws.send(msg)
    }
  })

  const handleResize = () => {
    if (activeTab.value === tab.id) {
      fitAddon.fit()
    }
  }
  window.addEventListener('resize', handleResize)
  tab._resizeHandler = handleResize
}

const addLocalTab = async () => {
  tabCounter++
  const tab: TermTab = {
    id: `term-${tabCounter}`,
    title: `本地终端 ${tabCounter}`,
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
  if (tab._resizeHandler) window.removeEventListener('resize', tab._resizeHandler)
  tabs.value.splice(idx, 1)
  if (activeTab.value === tab.id && tabs.value.length > 0) {
    activeTab.value = tabs.value[Math.min(idx, tabs.value.length - 1)].id
  }
}

const sendBatchCommand = () => {
  if (!batchCommand.value.trim()) return
  const cmd = batchCommand.value + '\n'
  for (const tab of tabs.value) {
    if (tab.ws && tab.ws.readyState === WebSocket.OPEN) {
      tab.ws.send(cmd)
    }
  }
  batchCommand.value = ''
  ElMessage.success('命令已发送到所有终端')
}

const executeCommand = (cmd: string) => {
  const tab = tabs.value.find((t) => t.id === activeTab.value)
  if (tab?.ws && tab.ws.readyState === WebSocket.OPEN) {
    tab.ws.send(cmd + '\n')
  } else {
    ElMessage.warning('当前没有活跃的终端连接')
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
    const tree = res.data || []
    const flat: any[] = []
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
      tab?.terminal?.focus()
    })
  }
})

onMounted(() => {
  loadHostTree()
  loadCommands()
  addLocalTab()
})

onBeforeUnmount(() => {
  for (const tab of tabs.value) {
    if (tab.ws) tab.ws.close()
    if (tab.terminal) tab.terminal.dispose()
    if (tab._resizeHandler) window.removeEventListener('resize', tab._resizeHandler)
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
      background: rgba(34, 211, 238, 0.12);
      border-color: var(--xp-accent);
      color: var(--xp-accent);
      box-shadow: -1px 0 0 0 var(--xp-accent);
    }
  }
}

.batch-input-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;

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
      background: rgba(34, 211, 238, 0.08);
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
  gap: 2px;
  padding: 0 4px;
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
    padding: 8px 14px;
    font-size: 13px;
    color: var(--xp-text-muted);
    cursor: pointer;
    border-bottom: 2px solid transparent;
    transition: all 0.2s;
    white-space: nowrap;
    user-select: none;

    &:hover {
      color: var(--xp-text-secondary);
      background: rgba(255, 255, 255, 0.03);
    }

    &.active {
      color: var(--xp-accent);
      border-bottom-color: var(--xp-accent);
      background: rgba(34, 211, 238, 0.05);
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
  background: #0b0e14;
  border: 1px solid var(--xp-border-light);
  border-top: none;
  border-radius: 0 0 var(--xp-radius-sm) var(--xp-radius-sm);
  overflow: hidden;
  position: relative;

  .terminal-instance {
    position: absolute;
    inset: 0;
    padding: 8px;
    display: none;

    &.active {
      display: block;
    }
  }
}

.sub-view {
  flex: 1;
  min-height: 0;
}

:deep(.xterm) {
  height: 100%;
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
