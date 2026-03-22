import http from '@/api/http'

export interface XrayNode {
  id: number
  name: string
  protocol: 'vless' | 'vmess' | 'trojan'
  port: number
  transport: 'tcp' | 'ws' | 'grpc'
  security: 'none' | 'tls' | 'reality'
  domain: string
  realityPublicKey: string
  realityShortIds: string
  realityServerNames: string
  path: string
  serviceName: string
  remark: string
  enabled: boolean
  userCount: number
  createdAt: string
}

export interface XrayNodeCreate {
  name: string
  protocol: string
  port: number
  transport: string
  security: string
  domain?: string
  tlsCert?: string
  tlsKey?: string
  realityPrivateKey?: string
  realityPublicKey?: string
  realityShortIds?: string
  realityServerNames?: string
  path?: string
  serviceName?: string
  remark?: string
}

export interface XrayNodeUpdate {
  id: number
  name: string
  transport: string
  security: string
  domain?: string
  tlsCert?: string
  tlsKey?: string
  realityPrivateKey?: string
  realityPublicKey?: string
  realityShortIds?: string
  realityServerNames?: string
  path?: string
  serviceName?: string
  remark?: string
  enabled: boolean
}

export interface XrayUser {
  id: number
  nodeId: number
  nodeName: string
  name: string
  uuid: string
  email: string
  level: number
  expireAt: string | null
  enabled: boolean
  remark: string
  uploadTotal: number
  downloadTotal: number
  createdAt: string
}

export interface XrayUserCreate {
  nodeId: number
  name: string
  uuid?: string
  level?: number
  expireAt?: string | null
  remark?: string
}

export interface XrayUserUpdate {
  id: number
  name: string
  level: number
  expireAt?: string | null
  enabled: boolean
  remark?: string
}

export interface XrayUserSearch {
  nodeId?: number
  page: number
  pageSize: number
}

export interface XrayStatus {
  installed: boolean
  running: boolean
  version: string
  configPath: string
  binPath: string
}

export interface XrayInstallStatus {
  running: boolean
  log: string
}

export interface XrayTrafficDaily {
  date: string
  upload: number
  download: number
}

export interface XrayRealityKeys {
  privateKey: string
  publicKey: string
}

// ==================== API 调用 ====================

export const getXrayStatus = () => http.get<XrayStatus>('/xray/status')

export const startXrayInstall = () => http.post('/xray/install', {})

export const getXrayInstallLog = () => http.get<XrayInstallStatus>('/xray/install/log')

export const listXrayNodes = () => http.get<XrayNode[]>('/xray/nodes')

export const createXrayNode = (data: XrayNodeCreate) => http.post('/xray/nodes', data)

export const updateXrayNode = (data: XrayNodeUpdate) => http.post('/xray/nodes/update', data)

export const deleteXrayNode = (id: number) => http.post('/xray/nodes/del', { id })

export const toggleXrayNode = (id: number) => http.post('/xray/nodes/toggle', { id })

export const searchXrayUsers = (data: XrayUserSearch) =>
  http.post<{ total: number; items: XrayUser[] }>('/xray/users/search', data)

export const createXrayUser = (data: XrayUserCreate) => http.post('/xray/users', data)

export const updateXrayUser = (data: XrayUserUpdate) => http.post('/xray/users/update', data)

export const deleteXrayUser = (id: number) => http.post('/xray/users/del', { id })

export const generateRealityKeys = () => http.get<XrayRealityKeys>('/xray/reality/keys')

export const getXrayShareLink = (id: number) =>
  http.post<{ link: string }>('/xray/users/share-link', { id })

export const getXrayTrafficHistory = (id: number) =>
  http.post<XrayTrafficDaily[]>('/xray/users/traffic-history', { id })
