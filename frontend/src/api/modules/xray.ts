import http from '@/api/http'

// ============================================================
// 传输方式子配置
// ============================================================

export interface XrayRawSettings {
  headerType: 'none' | 'http'
  acceptProxyProtocol: boolean
}

export interface XrayWSSettings {
  path: string
  host: string
  acceptProxyProtocol: boolean
}

export interface XrayGRPCSettings {
  serviceName: string
  multiMode: boolean
  idleTimeout: number
  healthCheckTimeout: number
  permitWithoutStream: boolean
  initialWindowsSize: number
}

export interface XrayXHTTPSettings {
  host: string
  path: string
  mode: 'auto' | 'packet-up' | 'stream-up' | 'stream-one'
  noSSEHeader: boolean
  xPaddingBytes: string
  scStreamUpServerSecs: string
  scMaxBufferedPosts: number
}

export interface XrayHTTPUpgradeSettings {
  path: string
  host: string
  acceptProxyProtocol: boolean
}

// ============================================================
// 安全方式子配置
// ============================================================

export interface XrayTLSSettings {
  serverName: string
  certFile: string
  keyFile: string
  alpn: string[]
  fingerprint: string
  minVersion: string
  rejectUnknownSni: boolean
}

export interface XrayRealitySettings {
  privateKey: string
  publicKey: string
  shortIds: string[]
  serverNames: string[]
  dest: string
  fingerprint: string
  spiderX: string
  xver: number
  show: boolean
}

// ============================================================
// 节点
// ============================================================

export interface XrayNode {
  id: number
  name: string
  protocol: 'vless' | 'vmess' | 'trojan' | 'shadowsocks'
  listenAddr: string
  port: number
  network: 'raw' | 'ws' | 'grpc' | 'xhttp' | 'httpupgrade'
  security: 'none' | 'tls' | 'reality'
  flow: string
  sniffEnabled: boolean
  sniffDestOverride: string[]
  rawSettings?: XrayRawSettings
  wsSettings?: XrayWSSettings
  grpcSettings?: XrayGRPCSettings
  xhttpSettings?: XrayXHTTPSettings
  httpUpgradeSettings?: XrayHTTPUpgradeSettings
  tlsSettings?: XrayTLSSettings
  realitySettings?: XrayRealitySettings
  remark: string
  enabled: boolean
  userCount: number
  createdAt: string
}

export interface XrayNodeForm {
  id?: number
  name: string
  protocol: string
  listenAddr: string
  port: number | null
  network: string
  security: string
  flow: string
  sniffEnabled: boolean
  sniffDestOverride: string[]
  rawSettings: XrayRawSettings
  wsSettings: XrayWSSettings
  grpcSettings: XrayGRPCSettings
  xhttpSettings: XrayXHTTPSettings
  httpUpgradeSettings: XrayHTTPUpgradeSettings
  tlsSettings: XrayTLSSettings
  realitySettings: XrayRealitySettings
  remark: string
  enabled: boolean
}

// ============================================================
// 用户
// ============================================================

export interface XrayUser {
  id: number
  nodeId: number
  nodeName: string
  name: string
  uuid: string
  email: string
  flow: string
  level: number
  expireAt: string | null
  enabled: boolean
  remark: string
  uploadTotal: number
  downloadTotal: number
  createdAt: string
}

export interface XrayUserForm {
  id?: number
  nodeId: number
  name: string
  uuid: string
  flow: string
  level: number
  expireAt: string | null
  enabled: boolean
  remark: string
}

export interface XrayUserSearch {
  nodeId?: number
  page: number
  pageSize: number
}

// ============================================================
// 状态 & 工具
// ============================================================

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

// ============================================================
// API 调用
// ============================================================

export const getXrayStatus = () => http.get<XrayStatus>('/xray/status')
export const startXrayInstall = () => http.post('/xray/install', {})
export const getXrayInstallLog = () => http.get<XrayInstallStatus>('/xray/install/log')

export const listXrayNodes = () => http.get<XrayNode[]>('/xray/nodes')
export const createXrayNode = (data: object) => http.post('/xray/nodes', data)
export const updateXrayNode = (data: object) => http.post('/xray/nodes/update', data)
export const deleteXrayNode = (id: number) => http.post('/xray/nodes/del', { id })
export const toggleXrayNode = (id: number) => http.post('/xray/nodes/toggle', { id })

export const searchXrayUsers = (data: XrayUserSearch) =>
  http.post<{ total: number; items: XrayUser[] }>('/xray/users/search', data)
export const createXrayUser = (data: object) => http.post('/xray/users', data)
export const updateXrayUser = (data: object) => http.post('/xray/users/update', data)
export const deleteXrayUser = (id: number) => http.post('/xray/users/del', { id })

export const generateRealityKeys = () => http.get<XrayRealityKeys>('/xray/reality/keys')
export const getXrayShareLink = (id: number) =>
  http.post<{ link: string }>('/xray/users/share-link', { id })
export const getXrayTrafficHistory = (id: number) =>
  http.post<XrayTrafficDaily[]>('/xray/users/traffic-history', { id })
