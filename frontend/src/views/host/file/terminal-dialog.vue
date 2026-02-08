<template>
  <el-drawer
    v-model="visible"
    :title="t('file.terminalTitle') + ' - ' + cwd"
    size="70%"
    direction="btt"
    destroy-on-close
    @close="handleClose"
    class="terminal-drawer"
  >
    <div ref="termEl" class="file-terminal" />
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, nextTick, onBeforeUnmount } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const visible = ref(false)
const cwd = ref('/')
const termEl = ref<HTMLElement>()
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeHandler: (() => void) | null = null

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

function getWsUrl() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = sessionStorage.getItem('token')
  return `${proto}//${location.host}/api/v1/terminal?token=${token}`
}

const open = async (path: string) => {
  cwd.value = path
  visible.value = true
  await nextTick()
  setTimeout(initTerminal, 100)
}

function initTerminal() {
  if (!termEl.value) return

  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    fontSize: 14,
    fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
    theme: terminalTheme,
    scrollback: 5000,
    allowProposedApi: true,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(termEl.value)
  setTimeout(() => fitAddon!.fit(), 50)

  ws = new WebSocket(getWsUrl())

  ws.onopen = () => {
    // Send initial resize
    const resizeData = JSON.stringify({ rows: terminal!.rows, cols: terminal!.cols })
    const msg = new Uint8Array(1 + resizeData.length)
    msg[0] = 1
    for (let i = 0; i < resizeData.length; i++) {
      msg[i + 1] = resizeData.charCodeAt(i)
    }
    ws!.send(msg)

    // cd to the directory
    setTimeout(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(`cd ${JSON.stringify(cwd.value)} && clear\n`)
      }
    }, 300)
  }

  ws.onmessage = (e: MessageEvent) => {
    terminal!.write(e.data)
  }

  ws.onclose = () => {
    terminal?.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n')
  }

  ws.onerror = () => {
    terminal?.write('\r\n\x1b[31m连接错误\x1b[0m\r\n')
  }

  terminal.onData((data: string) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  terminal.onResize(({ rows, cols }) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      const resizeData = JSON.stringify({ rows, cols })
      const msg = new Uint8Array(1 + resizeData.length)
      msg[0] = 1
      for (let i = 0; i < resizeData.length; i++) {
        msg[i + 1] = resizeData.charCodeAt(i)
      }
      ws.send(msg)
    }
  })

  resizeHandler = () => fitAddon!.fit()
  window.addEventListener('resize', resizeHandler)
}

function handleClose() {
  cleanup()
}

function cleanup() {
  if (ws) { ws.close(); ws = null }
  if (terminal) { terminal.dispose(); terminal = null }
  if (fitAddon) { fitAddon = null }
  if (resizeHandler) { window.removeEventListener('resize', resizeHandler); resizeHandler = null }
}

onBeforeUnmount(() => cleanup())

defineExpose({ open })
</script>

<style lang="scss" scoped>
.file-terminal {
  height: calc(50vh - 80px);
  background: #0b0e14;
  border-radius: 4px;
  padding: 8px;

  :deep(.xterm) {
    height: 100%;
  }

  :deep(.xterm-viewport) {
    &::-webkit-scrollbar { width: 6px; }
    &::-webkit-scrollbar-thumb {
      background: rgba(148, 163, 184, 0.15);
      border-radius: 3px;
    }
  }
}
</style>
