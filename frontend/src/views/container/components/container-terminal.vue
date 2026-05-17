<template>
  <el-drawer
    v-model="visible"
    :title="title"
    size="70%"
    destroy-on-close
    append-to-body
    @closed="cleanup"
  >
    <div class="container-terminal">
      <div class="terminal-toolbar">
        <el-form class="terminal-form" inline label-position="top">
          <el-form-item :label="t('container.user')">
            <el-input v-model="form.user" :placeholder="t('container.userPlaceholder')" clearable />
          </el-form-item>
          <el-form-item :label="t('container.command')">
            <div class="command-control">
              <el-checkbox v-model="form.customCommand">{{ t('container.customCommand') }}</el-checkbox>
              <el-select v-if="!form.customCommand" v-model="form.command" style="width: 180px">
                <el-option v-for="item in commandOptions" :key="item" :label="item" :value="item" />
              </el-select>
              <el-input v-else v-model="form.command" :placeholder="t('container.commandPlaceholder')" style="width: 220px" />
            </div>
          </el-form-item>
        </el-form>
        <el-button type="primary" :disabled="connected" @click="connect">
          {{ connected ? t('container.connected') : t('container.connect') }}
        </el-button>
      </div>

      <div ref="terminalRef" class="terminal-surface" @click="focusTerminal" />
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, nextTick, reactive, ref, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import type { Container } from '@/api/interface'
import { getToken } from '@/utils/auth'
import { getTermThemeByKey, getTermFontByKey, applyBgOpacity } from '@/utils/terminal-theme'
import { useGlobalStore } from '@/store/modules/global'
import { createTerminalHistoryController } from '@/utils/terminal-history'

const { t } = useI18n()
const globalStore = useGlobalStore()

const visible = ref(false)
const connected = ref(false)
const container = ref<Container | null>(null)
const terminalRef = ref<HTMLElement | null>(null)
const form = reactive({
  user: '',
  command: '/bin/sh',
  customCommand: false,
})

const commandOptions = ['/bin/sh', '/bin/bash', '/bin/ash', '/bin/zsh']
const title = computed(() => container.value ? `${t('container.terminal')} - ${container.value.name}` : t('container.terminal'))

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let resizeObserver: ResizeObserver | null = null
let terminalListeners: Array<{ dispose: () => void }> = []
let history = createTerminalHistoryController()

const open = async (row: Container) => {
  container.value = row
  form.user = ''
  form.command = '/bin/sh'
  form.customCommand = false
  visible.value = true
  connected.value = false
  await nextTick()
  setTimeout(initTerminal, 100)
}

const initTerminal = () => {
  if (!terminalRef.value || terminal) return

  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'bar',
    fontSize: globalStore.termFontSize,
    fontFamily: getTermFontByKey(globalStore.termFont),
    theme: applyBgOpacity(getTermThemeByKey(globalStore.termTheme), globalStore.termBgOpacity),
    scrollback: 5000,
    allowProposedApi: true,
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(terminalRef.value)

  resizeObserver = new ResizeObserver(() => {
    if (!fitAddon) return
    try {
      fitAddon.fit()
      sendResize()
    } catch { /* ignore */ }
  })
  resizeObserver.observe(terminalRef.value)

  setTimeout(() => {
    try { fitAddon?.fit() } catch { /* ignore */ }
    terminal?.focus()
  }, 100)
}

const connect = async () => {
  if (!container.value) return
  if (!terminal) initTerminal()
  await nextTick()
  if (!terminal) return

  cleanupConnection()
  terminal?.clear()
  terminal?.write(`\x1b[36m${t('container.connecting')}\x1b[0m\r\n`)

  ws = new WebSocket(buildWsURL())
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    connected.value = true
    sendResize()
    terminal?.focus()
  }

  ws.onmessage = (event: MessageEvent) => {
    if (event.data instanceof ArrayBuffer) {
      terminal?.write(new Uint8Array(event.data))
    } else {
      terminal?.write(event.data)
    }
  }

  ws.onclose = () => {
    connected.value = false
    terminal?.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n')
  }

  ws.onerror = () => {
    terminal?.write('\r\n\x1b[31m连接错误\x1b[0m\r\n')
  }

  clearTerminalListeners()
  terminalListeners.push(terminal!.onData((data: string) => {
    if (!ws || ws.readyState !== WebSocket.OPEN || !terminal) return
    const inAlternateBuffer = terminal.buffer.active.type === 'alternate'
    if (!history.handleData(data, (payload) => ws?.send(payload), { inAlternateBuffer })) {
      ws.send(data)
    }
  }))

  terminalListeners.push(terminal!.onResize(() => sendResize()))
}

const buildWsURL = () => {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const params = new URLSearchParams({
    token: getToken(),
    containerID: container.value?.id || '',
    command: form.command || '/bin/sh',
  })
  if (form.user) params.set('user', form.user)
  return `${proto}//${location.host}/api/v1/terminal?${params.toString()}`
}

const sendResize = () => {
  if (!ws || ws.readyState !== WebSocket.OPEN || !terminal) return
  const resizeData = JSON.stringify({ rows: terminal.rows, cols: terminal.cols })
  const msg = new Uint8Array(1 + resizeData.length)
  msg[0] = 1
  for (let i = 0; i < resizeData.length; i++) msg[i + 1] = resizeData.charCodeAt(i)
  ws.send(msg)
}

const focusTerminal = () => {
  terminal?.focus()
}

const cleanupConnection = () => {
  clearTerminalListeners()
  if (ws) {
    ws.close()
    ws = null
  }
  connected.value = false
  history = createTerminalHistoryController()
}

const clearTerminalListeners = () => {
  terminalListeners.forEach(listener => listener.dispose())
  terminalListeners = []
}

const cleanup = () => {
  cleanupConnection()
  resizeObserver?.disconnect()
  resizeObserver = null
  terminal?.dispose()
  terminal = null
  fitAddon = null
}

onBeforeUnmount(() => cleanup())

defineExpose({ open })
</script>

<style scoped lang="scss">
.container-terminal {
  height: calc(100vh - 96px);
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.terminal-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius);
  background: var(--xp-bg-card);
}

.terminal-form {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;

  :deep(.el-form-item) {
    margin: 0;
  }
}

.command-control {
  display: flex;
  align-items: center;
  gap: 10px;
}

.terminal-surface {
  flex: 1;
  min-height: 420px;
  padding: 10px;
  overflow: hidden;
  border-radius: var(--xp-radius);
  border: 1px solid var(--xp-border);
  background: var(--xp-terminal-bg);
}
</style>
