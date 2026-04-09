import http from '@/api/http'

// --- 证书源管理 ---
export const listCertSources = () => {
  return http.get('/cert-sources')
}

export const createCertSource = (params: {
  name: string
  serverAddr: string
  token: string
  syncInterval: number
  postSyncCommand: string
  enabled: boolean
}) => {
  return http.post('/cert-sources', params)
}

export const updateCertSource = (params: {
  id: number
  name: string
  serverAddr: string
  token?: string
  syncInterval: number
  postSyncCommand: string
  enabled: boolean
}) => {
  return http.post('/cert-sources/update', params)
}

export const deleteCertSource = (id: number) => {
  return http.post('/cert-sources/del', { id })
}

export const syncCertSource = (id: number) => {
  return http.post('/cert-sources/sync', { id })
}

export const testCertSource = (id: number) => {
  return http.post('/cert-sources/test', { id })
}

// --- 同步日志 ---
export const searchSyncLogs = (params: { page: number; pageSize: number; sourceID?: number }) => {
  return http.post('/cert-sync/logs', params)
}

// --- 证书服务端设置 ---
export const getCertServerSetting = () => {
  return http.get('/cert-server/setting')
}

export const updateCertServerSetting = (params: { enabled: boolean; token: string }) => {
  return http.post('/cert-server/setting', params)
}
