import http from '@/api/http'

// --- 证书 ---
export const searchCertificate = (params: { page: number; pageSize: number; info?: string }) => {
  return http.post('/certificates/search', params)
}

export const createCertificate = (params: {
  primaryDomain: string
  otherDomains?: string
  provider: string
  acmeAccountID?: number
  dnsAccountID?: number
  keyType?: string
  autoRenew?: boolean
  description?: string
  apply?: boolean
}) => {
  return http.post('/certificates', params)
}

export const updateCertificate = (params: {
  id: number
  autoRenew?: boolean
  description?: string
  primaryDomain?: string
  otherDomains?: string
}) => {
  return http.post('/certificates/update', params)
}

export const uploadCertificate = (params: { privateKey: string; certificate: string; description?: string }) => {
  return http.post('/certificates/upload', params)
}

export const deleteCertificate = (id: number) => {
  return http.post('/certificates/del', { id })
}

export const getCertificateDetail = (id: number) => {
  return http.post('/certificates/detail', { id })
}

export const applyCertificate = (id: number) => {
  return http.post('/certificates/apply', { id })
}

export const renewCertificate = (id: number) => {
  return http.post('/certificates/renew', { id })
}

export const getCertificateLog = (id: number) => {
  return http.post('/certificates/log', { id })
}

// --- ACME 账户 ---
export const listAcmeAccount = () => {
  return http.get('/acme-accounts')
}

export const createAcmeAccount = (params: {
  email: string
  type: string
  keyType: string
  caDirURL?: string
}) => {
  return http.post('/acme-accounts', params)
}

export const deleteAcmeAccount = (id: number) => {
  return http.post('/acme-accounts/del', { id })
}

// --- DNS 账户 ---
export const listDnsAccount = () => {
  return http.get('/dns-accounts')
}

export const createDnsAccount = (params: { name: string; type: string; authorization: Record<string, string> }) => {
  return http.post('/dns-accounts', params)
}

export const updateDnsAccount = (params: { id: number; name: string; type: string; authorization: Record<string, string> }) => {
  return http.post('/dns-accounts/update', params)
}

export const deleteDnsAccount = (id: number) => {
  return http.post('/dns-accounts/del', { id })
}

// --- 导入导出 ---
export const exportAccounts = () => {
  return http.get('/ssl/accounts/export')
}

export const importAccounts = (data: any) => {
  return http.post('/ssl/accounts/import', data)
}

// --- SSL 设置 ---
export const getSSLDir = () => {
  return http.get('/ssl/dir')
}

export const updateSSLDir = (dir: string) => {
  return http.post('/ssl/dir', { dir })
}

export const getDnsProviders = () => {
  return http.get('/ssl/dns-providers')
}
