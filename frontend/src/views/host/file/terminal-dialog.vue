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
    <div ref="termEl" class="file-terminal" @click="focusTerminal" />
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
let resizeObserver: ResizeObserver | null = null

import { getTermThemeByKey, getTermFontByKey, applyBgOpacity } from '@/utils/terminal-theme'
import { useGlobalStore } from '@/store/modules/global'

const globalStore = useGlobalStore()

function getWsUrl() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = sessionStorage.getItem('token')
  return `${proto}//${location.host}/api/v1/terminal?token=${token}`
}

function sendResize(wsConn: WebSocket, rows: number, cols: number) {
  const resizeData = JSON.stringify({ rows, cols })
  const msg = new Uint8Array(1 + resizeData.length)
  msg[0] = 1
  for (let i = 0; i < resizeData.length; i++) {
    msg[i + 1] = resizeData.charCodeAt(i)
  }
  wsConn.send(msg)
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
    fontSize: globalStore.termFontSize,
    fontFamily: getTermFontByKey(globalStore.termFont),
    theme: applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity),
    scrollback: 5000,
    allowProposedApi: true,
  })

  terminal.attachCustomKeyEventHandler((event: KeyboardEvent) => {
    if (event.ctrlKey && event.shiftKey && ['c', 'v'].includes(event.key.toLowerCase())) {
      return false
    }
    if (event.key === 'F11' || event.key === 'F12') return false
    return true
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(termEl.value)
  setTimeout(() => {
    try { fitAddon!.fit() } catch { /* */ }
    terminal!.focus()
  }, 100)

  resizeObserver = new ResizeObserver(() => {
    if (fitAddon) { try { fitAddon.fit() } catch { /* */ } }
  })
  resizeObserver.observe(termEl.value)

  ws = new WebSocket(getWsUrl())
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    sendResize(ws!, terminal!.rows, terminal!.cols)
    terminal!.focus()
    setTimeout(() => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(`cd ${JSON.stringify(cwd.value)} && clear\n`)
      }
    }, 300)
  }

  ws.onmessage = (e: MessageEvent) => {
    if (e.data instanceof ArrayBuffer) {
      terminal!.write(new Uint8Array(e.data))
    } else {
      terminal!.write(e.data)
    }
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
      sendResize(ws, rows, cols)
    }
  })
}

function focusTerminal() {
  terminal?.focus()
}

function handleClose() {
  cleanup()
}

function cleanup() {
  if (ws) { ws.close(); ws = null }
  if (terminal) { terminal.dispose(); terminal = null }
  if (fitAddon) { fitAddon = null }
  if (resizeObserver) { resizeObserver.disconnect(); resizeObserver = null }
}

onBeforeUnmount(() => cleanup())

defineExpose({ open })
</script>

<style lang="scss" scoped>
.file-terminal {
  height: calc(50vh - 80px);
  background: var(--xp-terminal-bg);
  border-radius: 4px;

  :deep(.xterm) {
    height: 100%;
    padding: 4px;
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
