import http from '../http'

export interface BackgroundTerminal {
  id: string
  name: string
  cwd: string
  startedAt: string
  status: 'running' | 'exited'
  attached: boolean
}

export const listTerminalSessions = () =>
  http.get<never, { data: BackgroundTerminal[] }>('/terminal/sessions', { skipErrorMessage: true })

export const renameTerminalSession = (id: string, name: string) =>
  http.post('/terminal/sessions/rename', { id, name })

export const closeTerminalSession = (id: string) => http.post('/terminal/sessions/del', { id })

export const setTerminalSessionLock = (id: string, locked: boolean, name: string) =>
  http.post('/terminal/sessions/lock', { id, locked, name })
