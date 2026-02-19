import http from '@/api/http'

// --- 网站管理 ---
export const searchWebsite = (params: { page: number; pageSize: number; info?: string; type?: string; status?: string }) => {
  return http.post('/websites/search', params)
}

export const createWebsite = (params: { primaryDomain: string; domains?: string; type: string; remark?: string; siteDir?: string; proxyPass?: string }) => {
  return http.post('/websites', params)
}

export const updateWebsite = (params: any) => {
  return http.post('/websites/update', params)
}

export const deleteWebsite = (id: number) => {
  return http.post('/websites/del', { id })
}

export const getWebsiteDetail = (id: number) => {
  return http.post('/websites/detail', { id })
}

export const enableWebsite = (id: number) => {
  return http.post('/websites/enable', { id })
}

export const disableWebsite = (id: number) => {
  return http.post('/websites/disable', { id })
}

export const getWebsiteNginxConfig = (id: number) => {
  return http.post('/websites/nginx-config', { id })
}

export const getWebsiteLog = (params: { id: number; type: string; tail?: number }) => {
  return http.post('/websites/log', params)
}

// --- Nginx 配置文件管理 ---
export const getNginxMainConf = () => {
  return http.get('/nginx/conf')
}

export const saveNginxMainConf = (content: string) => {
  return http.post('/nginx/conf', { content })
}

export const listNginxConfFiles = () => {
  return http.get('/nginx/conf-files')
}

export const getNginxConfFile = (name: string) => {
  return http.post('/nginx/conf-file', { name })
}

export const saveNginxConfFile = (filePath: string, content: string) => {
  return http.post('/nginx/conf-file/save', { filePath, content })
}
