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

// ====== Fail2ban ======

export const getFail2banStatus = () => http.get('/toolbox/fail2ban/status')
export const installFail2ban = () => http.post('/toolbox/fail2ban/install')
export const uninstallFail2ban = () => http.post('/toolbox/fail2ban/uninstall')
export const operateFail2ban = (operation: string) => http.post('/toolbox/fail2ban/operate', { operation })

export const listFail2banJails = () => http.get('/toolbox/fail2ban/jails')
export const updateFail2banJail = (params: object) => http.post('/toolbox/fail2ban/jails/update', params)
export const setFail2banSSH = (params: object) => http.post('/toolbox/fail2ban/jails/ssh', params)

export const listFail2banBanned = () => http.get('/toolbox/fail2ban/banned')
export const banFail2banIP = (ip: string, jail: string = 'sshd') => http.post('/toolbox/fail2ban/ban', { ip, jail })
export const unbanFail2banIP = (ip: string, jail: string) => http.post('/toolbox/fail2ban/unban', { ip, jail })

export const getFail2banLogs = (lines = 200) => http.get(`/toolbox/fail2ban/logs?lines=${lines}`)

// ====== IP Location ======

export const lookupIP = (ip: string) => http.get(`/toolbox/ip/lookup?ip=${ip}`)
export const lookupIPBatch = (ips: string[]) => http.post('/toolbox/ip/lookup/batch', { ips })
export const getIPDBInfo = () => http.get('/toolbox/ip/db/info')
export const downloadIPDB = () => http.post('/toolbox/ip/db/download')

// ====== Systemd Service Manager ======

export const listSystemdServices = (showAll = false) => http.get(`/toolbox/services?all=${showAll}`)
export const getSystemdServiceDetail = (name: string) => http.get(`/toolbox/services/detail?name=${name}`)
export const createSystemdService = (params: object) => http.post('/toolbox/services/create', params)
export const updateSystemdService = (params: object) => http.post('/toolbox/services/update', params)
export const deleteSystemdService = (name: string) => http.post('/toolbox/services/delete', { name })
export const operateSystemdService = (name: string, operation: string) => http.post('/toolbox/services/operate', { name, operation })
export const getSystemdServiceLogs = (name: string, lines = 100) => http.get(`/toolbox/services/logs?name=${name}&lines=${lines}`)
