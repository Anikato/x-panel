import http from '@/api/http'

export const getFirewallBase = () => {
  return http.get('/firewall/base')
}

export const operateFirewall = (operation: string) => {
  return http.post('/firewall/operate', { operation })
}

export const searchPortRules = (params: { page: number; pageSize: number; info?: string; strategy?: string }) => {
  return http.post('/firewall/port/search', params)
}

export const createPortRule = (params: { port: string; protocol: string; strategy: string; from?: string }) => {
  return http.post('/firewall/port', params)
}

export const deletePortRule = (params: { port: string; protocol: string; strategy: string; from?: string }) => {
  return http.post('/firewall/port/del', params)
}

export const getIPRules = () => {
  return http.get('/firewall/ip')
}

export const createIPRule = (params: { address: string; strategy: string }) => {
  return http.post('/firewall/ip', params)
}

export const deleteIPRule = (params: { address: string; strategy: string }) => {
  return http.post('/firewall/ip/del', params)
}
