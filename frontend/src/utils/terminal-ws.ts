import { normalizeTerminalCwd } from './terminal-cwd.ts'

export interface TerminalWsUrlOptions {
  protocol: string
  host: string
  token: string
  hostId?: number
  cwd?: string | null
}

export function buildTerminalWsUrl(options: TerminalWsUrlOptions): string {
  const proto = options.protocol === 'https:' ? 'wss:' : 'ws:'
  const url = new URL(`${proto}//${options.host}/api/v1/terminal`)
  url.searchParams.set('token', options.token)

  if (options.hostId) {
    url.searchParams.set('id', String(options.hostId))
  } else if (options.cwd) {
    url.searchParams.set('cwd', normalizeTerminalCwd(options.cwd))
  }

  return url.toString()
}
