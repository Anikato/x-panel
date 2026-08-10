import http from '@/api/http'
import type { NezhaAgentStatus } from '@/views/nezha-agent/state'

export type { NezhaAgentStatus }

/** Allowed lifecycle operations for the bundled Nezha Agent. */
export type NezhaAgentOperation = 'start' | 'stop' | 'restart' | 'enable' | 'disable'

/**
 * Config write payload.
 * clientSecret is optional: omit the field entirely when the user leaves it blank
 * so the backend leaves the existing secret unchanged.
 */
export interface NezhaAgentConfigPayload {
  dashboardUrl: string
  clientSecret?: string
  remoteOperationsEnabled: boolean
  enableAndStart: boolean
}

export const getNezhaAgentStatus = () => {
  return http.get('/nezha-agent/status')
}

export const updateNezhaAgentConfig = (data: NezhaAgentConfigPayload) => {
  return http.put('/nezha-agent/config', data)
}

/** Restore the version-matched bundled binary and systemd unit. No secret body. */
export const installNezhaAgent = () => {
  return http.post('/nezha-agent/install')
}

export const operateNezhaAgent = (operation: NezhaAgentOperation) => {
  return http.post('/nezha-agent/operate', { operation })
}
