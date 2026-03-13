import http from '@/api/http'

export const getSSHInfo = () => {
  return http.get('/ssh/info')
}

export const operateSSH = (operation: string) => {
  return http.post('/ssh/operate', { operation })
}

export const updateSSHConfig = (key: string, value: string) => {
  return http.post('/ssh/update', { key, value })
}

export const searchSSHLog = (params: { page: number; pageSize: number; status?: string; info?: string }) => {
  return http.post('/ssh/log', params)
}

export const getSSHDConfig = () => {
  return http.get('/ssh/sshd-config')
}

export const saveSSHDConfig = (content: string) => {
  return http.post('/ssh/sshd-config', { content })
}
