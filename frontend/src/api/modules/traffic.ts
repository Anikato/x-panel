import http from '@/api/http'

export interface TrafficConfig {
  id: number
  interfaceName: string
  monthlyLimit: number
  resetDay: number
  enabled: boolean
}

export interface TrafficConfigCreate {
  interfaceName: string
  monthlyLimit: number
  resetDay: number
  enabled: boolean
}

export interface InterfaceInfo {
  name: string
  ipv4: string[]
  ipv6: string[]
  mac: string
  status: string
}

export interface TrafficStatsRequest {
  interfaceName: string
  startTime: string
  endTime: string
  groupBy: 'hour' | 'day'
}

export interface TrafficStatsItem {
  timestamp: string
  bytesSent: number
  bytesRecv: number
}

export interface TrafficStatsResponse {
  interfaceName: string
  items: TrafficStatsItem[]
  totalSent: number
  totalRecv: number
}

export interface TrafficSummaryItem {
  interfaceName: string
  monthlyLimit: number
  resetDay: number
  periodStart: string
  periodEnd: string
  totalSent: number
  totalRecv: number
  totalUsed: number
  usedPercent: number
  enabled: boolean
}

export const trafficApi = {
  listConfigs: () => http.get<{ data: TrafficConfig[] }>('/traffic/configs'),
  createConfig: (params: TrafficConfigCreate) => http.post('/traffic/configs', params),
  deleteConfig: (interfaceName: string) => http.post('/traffic/configs/del', { interfaceName }),
  listInterfaces: () => http.get<{ data: InterfaceInfo[] }>('/traffic/interfaces'),
  getStats: (params: TrafficStatsRequest) => http.post<{ data: TrafficStatsResponse }>('/traffic/stats', params),
  getSummary: () => http.get<{ data: TrafficSummaryItem[] }>('/traffic/summary'),
}
