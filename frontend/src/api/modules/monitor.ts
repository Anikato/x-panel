import http from '@/api/http'

export const getSystemStats = () => {
  return http.get('/monitor/stats')
}

export const loadMonitorHistory = (data: { param: string; io?: string; network?: string; startTime: string; endTime: string }) => {
  return http.post('/monitor/history', data)
}

export const getMonitorSetting = () => {
  return http.get('/monitor/setting')
}

export const updateMonitorSetting = (data: { key: string; value: string }) => {
  return http.post('/monitor/setting/update', data)
}

export const cleanMonitorData = () => {
  return http.post('/monitor/history/clean')
}

export const getIOOptions = () => {
  return http.get('/monitor/io-options')
}

export const getNetworkOptions = () => {
  return http.get('/monitor/network-options')
}
