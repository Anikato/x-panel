import type { Terminal } from '@xterm/xterm'

// xterm.js 5.5 does not answer XTVERSION or OSC 52. Grok treats an unknown
// terminal with no desktop clipboard as unreachable and only writes
// ~/.grok/last-copy.txt. Identifying as xterm.js makes Grok send OSC 52.
const XTERM_VERSION = '5.5.0'

export function terminalXtversionReply(): string {
  return `\x1bP>|xterm.js(${XTERM_VERSION})\x1b\\`
}

// OSC 52 payload is "Pc;Pd". Pc selects the clipboard; Pd is base64, or "?".
// Returns the text to copy, "" to clear, or null when this sequence is not a
// clipboard write (a query, or a selection Grok is not asking us to store).
export function decodeTerminalOsc52(data: string): string | null {
  const semi = data.indexOf(';')
  const pc = semi < 0 ? '' : data.slice(0, semi)
  const pd = semi < 0 ? data : data.slice(semi + 1)
  if (pd === '?') return null
  if (pc && !pc.includes('c') && !pc.includes('0')) return null
  if (!pd) return ''
  try {
    const bytes = Uint8Array.from(atob(pd), (ch) => ch.charCodeAt(0))
    return new TextDecoder().decode(bytes)
  } catch {
    return null
  }
}

function reply(term: Terminal, data: string) {
  queueMicrotask(() => {
    try { term.input(data, false) } catch { /* terminal already disposed */ }
  })
}

async function writeBrowserClipboard(text: string) {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return
    }
  } catch { /* the page may be in the background; try the older path */ }
  const area = document.createElement('textarea')
  area.value = text
  area.setAttribute('readonly', '')
  area.style.position = 'fixed'
  area.style.left = '-9999px'
  document.body.appendChild(area)
  area.select()
  try { document.execCommand('copy') } catch { /* nothing else can reach the clipboard */ }
  area.remove()
}

export function installTerminalClipboard(term: Terminal) {
  term.parser.registerCsiHandler({ prefix: '>', final: 'q' }, (params) => {
    const raw = params[0]
    const n = Array.isArray(raw) ? raw[0] : raw
    if (n !== undefined && n !== 0) return true
    reply(term, terminalXtversionReply())
    return true
  })
  term.parser.registerOscHandler(52, (data) => {
    const text = decodeTerminalOsc52(data)
    if (text === null && data.endsWith('?')) {
      reply(term, '\x1b]52;c;\x07')
      return true
    }
    if (text) void writeBrowserClipboard(text)
    return true
  })
}
