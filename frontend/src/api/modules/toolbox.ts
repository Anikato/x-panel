import http from '@/api/http'

// ====== Samba ======

export const getSambaStatus = () => http.get('/toolbox/samba/status')
export const installSamba = () => http.post('/toolbox/samba/install')
export const uninstallSamba = () => http.post('/toolbox/samba/uninstall')
export const operateSamba = (operation: string) => http.post('/toolbox/samba/operate', { operation })

export const listSambaShares = () => http.get('/toolbox/samba/shares')
export const createSambaShare = (params: object) => http.post('/toolbox/samba/shares/create', params)
export const updateSambaShare = (params: object) => http.post('/toolbox/samba/shares/update', params)
export const deleteSambaShare = (name: string) => http.post('/toolbox/samba/shares/del', { name })

export const listSambaUsers = () => http.get('/toolbox/samba/users')
export const createSambaUser = (params: { username: string; password: string }) =>
  http.post('/toolbox/samba/users/create', params)
export const deleteSambaUser = (username: string) => http.post('/toolbox/samba/users/del', { username })
export const updateSambaPassword = (params: { username: string; password: string }) =>
  http.post('/toolbox/samba/users/password', params)
export const toggleSambaUser = (username: string, enabled: boolean) =>
  http.post('/toolbox/samba/users/toggle', { username, enabled })

export const getSambaGlobalConfig = () => http.get('/toolbox/samba/config')
export const updateSambaGlobalConfig = (params: object) => http.post('/toolbox/samba/config/update', params)

export const getSambaConnections = () => http.get('/toolbox/samba/connections')

// ====== NFS ======

export const getNfsStatus = () => http.get('/toolbox/nfs/status')
export const installNfs = () => http.post('/toolbox/nfs/install')
export const uninstallNfs = () => http.post('/toolbox/nfs/uninstall')
export const operateNfs = (operation: string) => http.post('/toolbox/nfs/operate', { operation })

export const listNfsExports = () => http.get('/toolbox/nfs/exports')
export const createNfsExport = (params: object) => http.post('/toolbox/nfs/exports/create', params)
export const updateNfsExport = (params: object) => http.post('/toolbox/nfs/exports/update', params)
export const deleteNfsExport = (path: string) => http.post('/toolbox/nfs/exports/del', { path })

export const getNfsConnections = () => http.get('/toolbox/nfs/connections')
