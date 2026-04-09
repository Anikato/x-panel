// ======================== API 响应 ========================

export interface ResData<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageResult<T> {
  total: number
  items: T[]
}

export interface SearchReq {
  page: number
  pageSize: number
}

// ======================== Auth ========================

export interface LoginPayload {
  name: string
  password: string
  captchaID?: string
  captcha?: string
}

export interface LoginResult {
  token: string
  name: string
  needCaptcha?: boolean
}

export interface CaptchaResult {
  captchaID: string
  imageData: string
}

export interface LoginSetting {
  panelName: string
}

// ======================== Setting ========================

export interface SettingInfo {
  panelName: string
  sessionTimeout: string
  securityEntrance: string
  githubToken: string
  serverPort: string
  userName: string
  agentToken: string
  autoUpgrade: string
}

export interface VersionInfo {
  version: string
  commitHash: string
  buildTime: string
  goVersion: string
}

export interface UpgradeInfo {
  hasUpdate: boolean
  latestVersion: string
  publishDate: string
  releaseNote: string
  downloadUrl: string
  checksumUrl: string
}

// ======================== File Manager ========================

export interface FileInfo {
  name: string
  path: string
  isDir: boolean
  isSymlink: boolean
  size: number
  mode: string
  user: string
  group: string
  modTime: string
}

export interface FileListResult {
  items: FileInfo[]
}

export interface DirSizeResult {
  size: number
}

// ======================== Database ========================

export interface DatabaseServer {
  id: number
  name: string
  type: string
  from: string
  address: string
  port: number
  username: string
  password: string
  _instances?: DatabaseInstance[]
  _loading?: boolean
}

export interface DatabaseInstance {
  id: number
  name: string
  charset: string
  owner: string
  createdAt: string
}

export interface DatabaseServerForm {
  id: number
  name: string
  type: string
  from: string
  address: string
  port: number
  username: string
  password: string
}

export interface DatabaseInstanceForm {
  serverID: number
  name: string
  charset: string
  password: string
  owner: string
}

// ======================== Container ========================

export interface Container {
  id: string
  name: string
  image: string
  state: string
  status: string
  ports: string
  ipAddress: string
  cpuPercent: number
  memUsage: number
  memLimit: number
  memPercent: number
  runTime: string
}

export interface DockerStatus {
  isExist: boolean
  isActive: boolean
  version: string
}

export interface ContainerImage {
  id: string
  tags: string[]
  size: number
}

export interface ContainerNetwork {
  id: string
  name: string
  driver: string
  subnet: string
  gateway: string
}

export interface ContainerVolume {
  name: string
  driver: string
  mountPoint: string
}

export interface ContainerCreateForm {
  name: string
  image: string
  restartPolicy: string
  env: string[]
  cmd: string[]
}

export interface ContainerSearchReq extends SearchReq {
  name?: string
}

export interface ContainerOperateReq {
  containerID: string
  operation: string
}

export interface NetworkCreateForm {
  name: string
  driver: string
  subnet: string
  gateway: string
}

export interface VolumeCreateForm {
  name: string
  driver: string
}

export interface ComposeItem {
  name: string
  status: string
  path: string
}

// ======================== Cronjob ========================

export interface Cronjob {
  id: number
  name: string
  type: string
  spec: string
  status: string
  script: string
  url: string
  website: string
  dbType: string
  dbName: string
  sourceDir: string
  targetAccountID: number
  retainCopies: number
  exclusionRules: string
  compressFormat: string
  encryptPassword: string
}

/** 与后端 dto.CronjobCreate 对齐（创建不传 id/status） */
export interface CronjobCreateForm {
  name: string
  type: string
  spec: string
  script: string
  url: string
  website: string
  dbType: string
  dbName: string
  sourceDir: string
  targetAccountID: number
  retainCopies: number
  exclusionRules: string
  compressFormat: string
  encryptPassword: string
}

/** 与后端 dto.CronjobUpdate 对齐 */
export interface CronjobUpdateForm extends CronjobCreateForm {
  id: number
}

export interface CronjobRecord {
  startTime: string
  duration: number
  status: string
  message: string
}

// ======================== Node ========================

export interface NodeItem {
  id: number
  name: string
  sshHost: string
  sshPort: number
  sshUser: string
  sshPassword: string
  address: string
  status: string
  os: string
  hostname: string
  groupID: number
  _actionLoading?: boolean
}

export interface NodeForm {
  id: number
  name: string
  sshHost: string
  sshPort: number
  sshUser: string
  sshPassword: string
  panelPort: string
  agentToken: string
  groupID: number
}

export interface AgentActionResult {
  output: string
  success: boolean
}

// ======================== Backup ========================

export interface BackupAccount {
  id: number
  name: string
  type: string
  backupPath: string
  accessKey: string
  credential: string
  bucket: string
  vars: string
}

export interface BackupRecord {
  id: number
  type: string
  name: string
  fileName: string
  status: string
  createdAt: string
}

export interface BackupForm {
  type: string
  name: string
  accountID: number
  dbType: string
  sourceDir: string
}

// ======================== SSL ========================

export interface Certificate {
  id: number
  primaryDomain: string
  domains: string
  status: string
  provider: string
  type: string
  expireDate: string
  startDate: string
  autoRenew: boolean
  message: string
  pem: string
  privateKey: string
  filePath: string
  description: string
}

export interface AcmeAccount {
  id: number
  email: string
  type: string
  keyType: string
  url: string
}

export interface DnsAccount {
  id: number
  name: string
  type: string
  authorization: Record<string, string>
}

export interface DnsProvider {
  value: string
  label: string
  fields: string
}

// ======================== Cert Sync ========================

export interface CertSource {
  id: number
  name: string
  serverAddr: string
  syncInterval: number
  postSyncCommand: string
  enabled: boolean
  lastSyncAt: string | null
  lastSyncStatus: string
  lastSyncMessage: string
  createdAt: string
}

export interface CertSyncLog {
  id: number
  sourceID: number
  sourceName: string
  domain: string
  status: string
  message: string
  certificateID: number
  createdAt: string
}

export interface CertServerSetting {
  enabled: boolean
  token: string
}

// ======================== Website ========================

export interface Website {
  id: number
  primaryDomain: string
  domains: string
  type: string
  status: string
  sslEnable: boolean
  remark: string
  siteDir: string
  proxyPass: string
}

export interface ConfFile {
  name: string
}

// ======================== Nginx ========================

export interface NginxStatus {
  isInstalled: boolean
  isRunning: boolean
  version: string
  pid: number
  configOK: boolean
  installDir: string
  startedAt: string
  autoStart: boolean
  systemMode: boolean
  hasBothInstalled: boolean
  websiteCount: number
}

export interface NginxVersion {
  version: string
  publishedAt: string
}

export interface NginxTestResult {
  success: boolean
  output: string
}

export interface NginxInstallProgress {
  phase: string
  message: string
  percent: number
}

// ======================== Process ========================

export interface ProcessInfo {
  pid: number
  name: string
  username: string
  cpuPercent: number
  memPercent: number
  memRSS: number
  memRss: number
  status: string
  numThreads: number
  cmdLine: string
}

export interface NetworkConn {
  protocol: string
  localAddr: string
  localPort: number
  remoteAddr: string
  remotePort: number
  status: string
  pid: number
  name: string
}

// ======================== Log ========================

export interface OperationLog {
  id: number
  method: string
  path: string
  ip: string
  status: string
  latency: string
  message: string
  createdAt: string
}

export interface LoginLog {
  id: number
  ip: string
  address: string
  status: string
  message: string
  createdAt: string
}

// ======================== Monitor / Home ========================

export interface HostInfo {
  hostname: string
  platform: string
  platformVersion: string
  kernelVersion: string
  kernelArch: string
  virtualization: string
  timezone: string
  publicIPv4: string
  publicIPv6: string
  interfaces: NetInterface[]
  dnsServers: string[]
}

export interface NetInterface {
  name: string
  ipv4: string[]
  status: string
}

export interface CpuInfo {
  modelName: string
  cores: number
  logicalCores: number
  usagePercent: number
}

export interface MemoryInfo {
  total: number
  used: number
  usedPercent: number
  swapTotal: number
  swapUsed: number
  swapPercent: number
}

export interface LoadInfo {
  load1: number
  load5: number
  load15: number
}

export interface DiskInfo {
  device: string
  mountPoint: string
  fsType: string
  total: number
  used: number
  usedPercent: number
  inodesTotal: number
  inodesUsed: number
  inodesPercent: number
}

export interface NetIOInfo {
  name: string
  speedUp: number
  speedDown: number
}

export interface TopProcess {
  pid: number
  name: string
  cpuPercent: number
  memRss: number
}

export interface NetworkTotal {
  bytesSent: number
  bytesRecv: number
}

export interface SystemStats {
  host: HostInfo
  cpu: CpuInfo
  memory: MemoryInfo
  load: LoadInfo
  disks: DiskInfo[]
  network: NetworkTotal
  netIO: NetIOInfo[]
  topProcess: TopProcess[]
  uptime: number
}

// ======================== SSH ========================

export interface SSHInfo {
  isExist: boolean
  isActive: boolean
  message: string
  port: string
  listenAddress: string
  passwordAuthentication: string
  pubkeyAuthentication: string
  permitRootLogin: string
  useDNS: string
  autoStart: boolean
}

export interface SSHLogEntry {
  date: string
  status: string
  user: string
  ip: string
  port: string
  message: string
}

export interface AuthorizedKey {
  keyType: string
  key: string
  name: string
  fingerprint: string
}

// ======================== Disk (Detail) ========================

export interface DiskDetail {
  device: string
  model: string
  size: number
  type: string
  partitions: PartitionInfo[]
}

export interface PartitionInfo {
  device: string
  mountPoint: string
  fsType: string
  total: number
  used: number
  free: number
  usedPercent: number
  inodesTotal: number
  inodesUsed: number
  inodesFree: number
}

export interface RemoteMountInfo {
  device: string
  mountPoint: string
  fsType: string
  options: string
  total: number
  used: number
  free: number
  percent: number
  inFstab: boolean
}

// ======================== Traffic ========================

export interface TrafficConfig {
  id: number
  name: string
  interface: string
  port: number
  protocol: string
  enabled: boolean
}

// ======================== Host Tree / Command ========================

export interface HostTreeGroup {
  label: string
  children: HostTreeItem[]
}

export interface HostTreeItem {
  id: number
  label: string
}

export interface CommandItem {
  id: number
  name: string
  command: string
}

export interface CommandGroup {
  label: string
  children: CommandItem[]
}

// ======================== GOST ========================

export interface GostStatus {
  isInstalled: boolean
  isRunning: boolean
  version: string
  apiReady: boolean
}

export interface GostInstallProgress {
  phase: string
  message: string
  percent: number
}

export interface GostCheckUpdateResp {
  currentVersion: string
  latestVersion: string
  hasUpdate: boolean
  releaseURL: string
}

export interface GostServiceInfo {
  id: number
  name: string
  type: string
  listenAddr: string
  targetAddr: string
  listenerType: string
  authUser: string
  chainID: number
  chainName: string
  certificateID: number
  certDomain: string
  customCertPath: string
  customKeyPath: string
  enableStats: boolean
  enabled: boolean
  remark: string
}

export interface GostChainInfo {
  id: number
  name: string
  hops: string
  hopCount: number
  refCount: number
  remark: string
}
