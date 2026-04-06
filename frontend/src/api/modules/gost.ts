import http from '@/api/http'

// --- GOST 状态 / 安装 ---
export const getGostStatus = () => {
  return http.get('/gost/status')
}

export const installGost = (version?: string) => {
  return http.post('/gost/install', { version })
}

export const getGostInstallProgress = () => {
  return http.get('/gost/install/progress')
}

export const uninstallGost = () => {
  return http.post('/gost/uninstall')
}

export const operateGost = (operation: string) => {
  return http.post('/gost/operate', { operation })
}

export const checkGostUpdate = () => {
  return http.get('/gost/check-update')
}

export const upgradeGost = (version: string) => {
  return http.post('/gost/upgrade', { version })
}

// --- GOST Service (端口转发 / 中继服务) ---
export const searchGostService = (params: object) => {
  return http.post('/gost/services/search', params)
}

export const createGostService = (params: object) => {
  return http.post('/gost/services', params)
}

export const updateGostService = (params: object) => {
  return http.post('/gost/services/update', params)
}

export const deleteGostService = (id: number) => {
  return http.post('/gost/services/del', { id })
}

export const toggleGostService = (id: number, enabled: boolean) => {
  return http.post('/gost/services/toggle', { id, enabled })
}

// --- GOST Chain (转发链) ---
export const searchGostChain = (params: object) => {
  return http.post('/gost/chains/search', params)
}

export const createGostChain = (params: object) => {
  return http.post('/gost/chains', params)
}

export const updateGostChain = (params: object) => {
  return http.post('/gost/chains/update', params)
}

export const deleteGostChain = (id: number) => {
  return http.post('/gost/chains/del', { id })
}

export const syncGost = () => {
  return http.post('/gost/sync')
}
