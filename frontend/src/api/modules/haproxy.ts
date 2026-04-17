import http from '@/api/http'

// --- 状态 / 安装 / 升级 ---
export const getHAProxyStatus = () => http.get('/haproxy/status')
export const installHAProxy = () => http.post('/haproxy/install', {})
export const getHAProxyInstallProgress = () => http.get('/haproxy/install/progress')
export const uninstallHAProxy = () => http.post('/haproxy/uninstall')
export const operateHAProxy = (operation: string) => http.post('/haproxy/operate', { operation })
export const checkHAProxyUpdate = () => http.get('/haproxy/check-update')
export const upgradeHAProxy = () => http.post('/haproxy/upgrade', {})

// --- LB ---
export const searchHAProxyLB = (params: object) => http.post('/haproxy/lbs/search', params)
export const listHAProxyLB = () => http.get('/haproxy/lbs')
export const createHAProxyLB = (params: object) => http.post('/haproxy/lbs', params)
export const updateHAProxyLB = (params: object) => http.post('/haproxy/lbs/update', params)
export const deleteHAProxyLB = (id: number) => http.post('/haproxy/lbs/del', { id })
export const toggleHAProxyLB = (params: { id: number; enabled: boolean }) =>
  http.post('/haproxy/lbs/toggle', params)

// --- Backend ---
export const searchHAProxyBackend = (params: object) => http.post('/haproxy/backends/search', params)
export const listHAProxyBackend = () => http.get('/haproxy/backends')
export const getHAProxyBackend = (id: number) => http.post('/haproxy/backends/detail', { id })
export const createHAProxyBackend = (params: object) => http.post('/haproxy/backends', params)
export const updateHAProxyBackend = (params: object) => http.post('/haproxy/backends/update', params)
export const deleteHAProxyBackend = (id: number) => http.post('/haproxy/backends/del', { id })

// --- Server ---
export const createHAProxyServer = (params: object) => http.post('/haproxy/servers', params)
export const updateHAProxyServer = (params: object) => http.post('/haproxy/servers/update', params)
export const deleteHAProxyServer = (id: number) => http.post('/haproxy/servers/del', { id })
// 运行时：toggle 走 admin socket，立即生效。disable 为 true 即下线
export const toggleHAProxyServerLive = (params: { id: number; disable: boolean }) =>
  http.post('/haproxy/servers/toggle-live', params)
export const setHAProxyServerWeightLive = (params: { id: number; weight: number }) =>
  http.post('/haproxy/servers/weight-live', params)

// --- ACL ---
export const listHAProxyACL = (lbID: number) => http.post('/haproxy/acls/search', { lbID })
export const createHAProxyACL = (params: object) => http.post('/haproxy/acls', params)
export const updateHAProxyACL = (params: object) => http.post('/haproxy/acls/update', params)
export const deleteHAProxyACL = (id: number) => http.post('/haproxy/acls/del', { id })

// --- Stats / Runtime ---
export const getHAProxyStats = () => http.get('/haproxy/stats')
export const getHAProxyRuntimeInfo = () => http.get('/haproxy/runtime-info')
export const clearHAProxyCounters = () => http.post('/haproxy/counters/clear')

// --- Raw Config / 历史 ---
export const getHAProxyRawConfig = () => http.get('/haproxy/config/raw')
export const previewHAProxyConfig = () => http.get('/haproxy/config/preview')
export const saveHAProxyRawConfig = (params: { content: string }) =>
  http.post('/haproxy/config/raw', params)
export const testHAProxyConfig = (params: { content: string }) =>
  http.post('/haproxy/config/test', params)
export const rebuildHAProxyConfig = () => http.post('/haproxy/config/rebuild')
export const listHAProxyConfigVersions = () => http.get('/haproxy/config/versions')
export const getHAProxyConfigVersion = (id: number) =>
  http.post('/haproxy/config/versions/detail', { id })
export const rollbackHAProxyConfig = (id: number) =>
  http.post('/haproxy/config/rollback', { id })

// --- 证书（HTTP LB 选择证书时使用） ---
export const listCertificatesForHAProxy = () =>
  http.post('/certificates/search', { page: 1, pageSize: 100 })
