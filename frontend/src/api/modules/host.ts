import http from '@/api/http'

// --- 主机管理 ---
export const searchHost = (params: { page: number; pageSize: number; info?: string; groupID?: number }) => {
  return http.post('/hosts/search', params)
}

export const createHost = (params: {
  groupID?: number
  name: string
  addr: string
  port: number
  user: string
  authMode: string
  password?: string
  privateKey?: string
  passPhrase?: string
  description?: string
}) => {
  return http.post('/hosts', params)
}

export const updateHost = (params: {
  id: number
  groupID?: number
  name: string
  addr: string
  port: number
  user: string
  authMode: string
  password?: string
  privateKey?: string
  passPhrase?: string
  description?: string
}) => {
  return http.post('/hosts/update', params)
}

export const deleteHost = (id: number) => {
  return http.post('/hosts/del', { id })
}

export const getHostTree = () => {
  return http.get('/hosts/tree')
}

export const testHost = (id: number) => {
  return http.get(`/hosts/test?id=${id}`)
}

export const testHostConn = (params: {
  name: string
  addr: string
  port: number
  user: string
  authMode: string
  password?: string
  privateKey?: string
  passPhrase?: string
}) => {
  return http.post('/hosts/test-conn', params)
}

// --- 快速命令 ---
export const searchCommand = (params: { page: number; pageSize: number; info?: string; groupID?: number }) => {
  return http.post('/commands/search', params)
}

export const createCommand = (params: { groupID?: number; name: string; command: string }) => {
  return http.post('/commands', params)
}

export const updateCommand = (params: { id: number; groupID?: number; name: string; command: string }) => {
  return http.post('/commands/update', params)
}

export const deleteCommand = (id: number) => {
  return http.post('/commands/del', { id })
}

export const getCommandTree = () => {
  return http.get('/commands/tree')
}

// --- 分组 ---
export const getGroupList = (type: string) => {
  return http.get(`/groups?type=${type}`)
}

export const createGroup = (params: { name: string; type: string }) => {
  return http.post('/groups', params)
}

export const updateGroup = (params: { id: number; name: string }) => {
  return http.post('/groups/update', params)
}

export const deleteGroup = (id: number) => {
  return http.post('/groups/del', { id })
}
