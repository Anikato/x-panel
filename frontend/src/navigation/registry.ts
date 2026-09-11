export type NavPlacement = 'main' | 'footer'
export type NavGroupId = 'workspace' | 'apps' | 'ops'
export type LocalNavKind = 'none' | 'tabs' | 'side'

export interface NavLocation {
  path: string
  query?: Record<string, string>
}

export interface LocalNavItem extends NavLocation {
  id: string
  titleKey: string
  keywords?: string[]
  children?: LocalNavItem[]
}

export interface NavPartition {
  id: string
  titleKey: string
  defaultPath: string
  items: LocalNavItem[]
}

export interface NavModule {
  id: string
  titleKey: string
  icon: string
  group?: NavGroupId
  placement: NavPlacement
  defaultPath: string
  keywords: string[]
  localNav: LocalNavKind
  items?: LocalNavItem[]
  partitions?: NavPartition[]
  match?: (path: string) => boolean
}

export interface NavHit extends NavLocation {
  id: string
  moduleId: string
  titleKey: string
  keywords: string[]
}

export const NAV_GROUPS: { id: NavGroupId; titleKey: string }[] = [
  { id: 'workspace', titleKey: 'nav.groupWorkspace' },
  { id: 'apps', titleKey: 'nav.groupApps' },
  { id: 'ops', titleKey: 'nav.groupOps' },
]

export const NAV_MODULES: NavModule[] = [
  {
    id: 'overview',
    titleKey: 'nav.overview',
    icon: 'HomeFilled',
    group: 'workspace',
    placement: 'main',
    defaultPath: '/home',
    keywords: ['概览', '首页', '仪表盘'],
    localNav: 'none',
  },
  {
    id: 'files',
    titleKey: 'nav.files',
    icon: 'FolderOpened',
    group: 'workspace',
    placement: 'main',
    defaultPath: '/host/files',
    keywords: ['文件', '文件管理', 'file'],
    localNav: 'none',
  },
  {
    id: 'terminal',
    titleKey: 'nav.terminal',
    icon: 'Monitor',
    group: 'workspace',
    placement: 'main',
    defaultPath: '/terminal',
    keywords: ['终端', 'SSH', '命令行'],
    localNav: 'none',
  },
  {
    id: 'website',
    titleKey: 'nav.website',
    icon: 'ChromeFilled',
    group: 'apps',
    placement: 'main',
    defaultPath: '/website/websites',
    keywords: ['网站', '站点', 'nginx 站点'],
    localNav: 'tabs',
    items: [
      { id: 'website-list', titleKey: 'nav.websiteList', path: '/website/websites', keywords: ['网站列表', '站点'] },
      { id: 'website-nginx', titleKey: 'nav.nginx', path: '/website/nginx', keywords: ['nginx', '配置'] },
    ],
    match: (path) => /^\/website\/websites(\/|$)/.test(path) || path === '/website/nginx',
  },
  {
    id: 'network',
    titleKey: 'nav.network',
    icon: 'Connection',
    group: 'apps',
    placement: 'main',
    defaultPath: '/haproxy/status',
    keywords: ['网络服务', '负载均衡', '代理', 'haproxy', 'gost'],
    localNav: 'side',
    partitions: [
      {
        id: 'haproxy',
        titleKey: 'nav.networkLoadBalance',
        defaultPath: '/haproxy/status',
        items: [
          { id: 'haproxy-status', titleKey: 'menu.haproxyStatus', path: '/haproxy/status', keywords: ['haproxy', '概览'] },
          { id: 'haproxy-http', titleKey: 'menu.haproxyHTTPLB', path: '/haproxy/http-lb', keywords: ['http', '负载'] },
          { id: 'haproxy-tcp', titleKey: 'menu.haproxyTCPLB', path: '/haproxy/tcp-lb', keywords: ['tcp'] },
          { id: 'haproxy-backends', titleKey: 'menu.haproxyBackends', path: '/haproxy/backends', keywords: ['后端池'] },
          { id: 'haproxy-stats', titleKey: 'menu.haproxyStats', path: '/haproxy/stats', keywords: ['实时统计'] },
          { id: 'haproxy-config', titleKey: 'menu.haproxyConfig', path: '/haproxy/config', keywords: ['配置'] },
          { id: 'haproxy-history', titleKey: 'menu.haproxyHistory', path: '/haproxy/history', keywords: ['历史', '回滚'] },
        ],
      },
      {
        id: 'gost',
        titleKey: 'nav.networkProxy',
        defaultPath: '/gost/status',
        items: [
          { id: 'gost-status', titleKey: 'menu.gostStatus', path: '/gost/status', keywords: ['gost', '概览'] },
          { id: 'gost-forward', titleKey: 'menu.gostForward', path: '/gost/forward', keywords: ['转发'] },
          { id: 'gost-relay', titleKey: 'menu.gostRelay', path: '/gost/relay', keywords: ['转发器', '中继'] },
          { id: 'gost-chain', titleKey: 'menu.gostChain', path: '/gost/chain', keywords: ['代理链', '转发链'] },
        ],
      },
    ],
    match: (path) => path.startsWith('/haproxy/') || path.startsWith('/gost/'),
  },
  {
    id: 'container',
    titleKey: 'nav.container',
    icon: 'Box',
    group: 'apps',
    placement: 'main',
    defaultPath: '/container',
    keywords: ['容器', 'docker', 'compose', '镜像'],
    localNav: 'none',
  },
  {
    id: 'certs',
    titleKey: 'nav.certs',
    icon: 'Lock',
    group: 'apps',
    placement: 'main',
    defaultPath: '/website/ssl',
    keywords: ['证书', 'ssl', 'https', 'acme'],
    localNav: 'none',
    match: (path) => path === '/website/ssl' || path.startsWith('/website/ssl/'),
  },
  {
    id: 'database',
    titleKey: 'nav.database',
    icon: 'Coin',
    group: 'apps',
    placement: 'main',
    defaultPath: '/database',
    keywords: ['数据库', 'mysql', 'mariadb', 'postgres'],
    localNav: 'none',
  },
  {
    id: 'automation',
    titleKey: 'nav.automation',
    icon: 'Timer',
    group: 'ops',
    placement: 'main',
    defaultPath: '/cronjob',
    keywords: ['自动化', '备份', '计划任务', 'cron'],
    localNav: 'tabs',
    items: [
      { id: 'cronjob', titleKey: 'nav.cronjob', path: '/cronjob', keywords: ['计划任务', '定时'] },
      { id: 'backup-records', titleKey: 'nav.backupRecords', path: '/backup', query: { tab: 'records' }, keywords: ['备份记录'] },
      { id: 'backup-accounts', titleKey: 'nav.backupAccounts', path: '/backup', query: { tab: 'accounts' }, keywords: ['存储账户'] },
      { id: 'backup-storage', titleKey: 'nav.backupObjects', path: '/backup', query: { tab: 'storage' }, keywords: ['对象文件'] },
    ],
    match: (path) => path === '/cronjob' || path === '/backup',
  },
  {
    id: 'monitor',
    titleKey: 'nav.monitor',
    icon: 'DataLine',
    group: 'ops',
    placement: 'main',
    defaultPath: '/host/monitor',
    keywords: ['监控', '流量', '资源'],
    localNav: 'tabs',
    items: [
      { id: 'monitor-resource', titleKey: 'nav.resourceMonitor', path: '/host/monitor', keywords: ['资源监控'] },
      { id: 'monitor-traffic', titleKey: 'nav.traffic', path: '/traffic', keywords: ['网络流量'] },
    ],
  },
  {
    id: 'host',
    titleKey: 'nav.host',
    icon: 'Platform',
    group: 'ops',
    placement: 'main',
    defaultPath: '/toolbox/services',
    keywords: ['主机系统', 'systemd', '磁盘', '用户'],
    localNav: 'side',
    items: [
      { id: 'host-services', titleKey: 'nav.systemServices', path: '/toolbox/services', keywords: ['系统服务', 'systemd'] },
      { id: 'host-process', titleKey: 'nav.processes', path: '/host/process', keywords: ['进程'] },
      { id: 'host-disk', titleKey: 'nav.disks', path: '/host/disk', keywords: ['磁盘', '存储'] },
      {
        id: 'host-share',
        titleKey: 'nav.fileShare',
        path: '/toolbox/samba',
        keywords: ['文件共享'],
        children: [
          { id: 'host-samba', titleKey: 'menu.toolboxSamba', path: '/toolbox/samba', keywords: ['samba'] },
          { id: 'host-nfs', titleKey: 'menu.toolboxNfs', path: '/toolbox/nfs', keywords: ['nfs'] },
        ],
      },
      { id: 'host-users', titleKey: 'nav.systemUsers', path: '/host/users', keywords: ['系统用户', 'linux'] },
      { id: 'host-system', titleKey: 'nav.hostConfig', path: '/host/system', keywords: ['主机配置'] },
      { id: 'host-agent', titleKey: 'nav.cloudAgent', path: '/nezha-agent', keywords: ['云控', 'agent'] },
    ],
  },
  {
    id: 'security',
    titleKey: 'nav.security',
    icon: 'Shield',
    group: 'ops',
    placement: 'main',
    defaultPath: '/host/firewall',
    keywords: ['安全', '防火墙', 'ssh', 'fail2ban'],
    localNav: 'tabs',
    items: [
      { id: 'security-firewall', titleKey: 'nav.firewall', path: '/host/firewall', keywords: ['防火墙'] },
      { id: 'security-ssh', titleKey: 'nav.sshService', path: '/host/ssh', keywords: ['ssh 服务'] },
      { id: 'security-fail2ban', titleKey: 'nav.fail2ban', path: '/toolbox/fail2ban', keywords: ['fail2ban', '封禁'] },
    ],
  },
  {
    id: 'logs',
    titleKey: 'nav.panelLogs',
    icon: 'Document',
    placement: 'footer',
    defaultPath: '/log/system',
    keywords: ['面板日志', '运行日志', '登录日志', '操作日志'],
    localNav: 'tabs',
    items: [
      { id: 'log-system', titleKey: 'nav.runLog', path: '/log/system', keywords: ['运行日志'] },
      { id: 'log-login', titleKey: 'nav.loginLog', path: '/log/login', keywords: ['登录日志'] },
      { id: 'log-operation', titleKey: 'nav.operationLog', path: '/log/operation', keywords: ['操作日志'] },
    ],
  },
  {
    id: 'settings',
    titleKey: 'nav.settings',
    icon: 'Setting',
    placement: 'footer',
    defaultPath: '/setting',
    keywords: ['面板设置', '外观', '账号'],
    localNav: 'none',
  },
]

export const SIDEBAR_MAIN_MODULES = NAV_MODULES.filter((mod) => mod.placement === 'main')
export const SIDEBAR_FOOTER_MODULES = NAV_MODULES.filter((mod) => mod.placement === 'footer')

function queryEquals(a?: Record<string, string>, b?: Record<string, string>) {
  const left = a || {}
  const right = b || {}
  const keys = new Set([...Object.keys(left), ...Object.keys(right)])
  for (const key of keys) {
    if ((left[key] || '') !== (right[key] || '')) return false
  }
  return true
}

function flattenItems(moduleId: string, items: LocalNavItem[] = []): NavHit[] {
  const hits: NavHit[] = []
  for (const item of items) {
    hits.push({
      id: item.id,
      moduleId,
      titleKey: item.titleKey,
      path: item.path,
      query: item.query,
      keywords: item.keywords || [],
    })
    if (item.children?.length) hits.push(...flattenItems(moduleId, item.children))
  }
  return hits
}

export function collectModuleHits(mod: NavModule): NavHit[] {
  const hits: NavHit[] = [{
    id: mod.id,
    moduleId: mod.id,
    titleKey: mod.titleKey,
    path: mod.defaultPath,
    keywords: mod.keywords,
  }]
  if (mod.items) hits.push(...flattenItems(mod.id, mod.items))
  if (mod.partitions) {
    for (const partition of mod.partitions) {
      hits.push(...flattenItems(mod.id, partition.items))
    }
  }
  return hits
}

export function collectReachableDestinations(): NavHit[] {
  const seen = new Set<string>()
  const result: NavHit[] = []
  for (const mod of NAV_MODULES) {
    for (const hit of collectModuleHits(mod)) {
      const key = `${hit.path}?${new URLSearchParams(hit.query || {}).toString()}`
      if (seen.has(key)) continue
      seen.add(key)
      result.push(hit)
    }
  }
  return result
}

export function locationMatches(target: NavLocation, path: string, query: Record<string, string> = {}) {
  if (target.path !== path) return false
  if (!target.query) return true
  return Object.entries(target.query).every(([key, value]) => query[key] === value)
}

export function findLocalNavItem(path: string, query: Record<string, string> = {}): LocalNavItem | undefined {
  const items: LocalNavItem[] = []
  for (const mod of NAV_MODULES) {
    if (mod.items) items.push(...mod.items)
    if (mod.partitions) {
      for (const partition of mod.partitions) items.push(...partition.items)
    }
  }
  const flat: LocalNavItem[] = []
  const walk = (list: LocalNavItem[]) => {
    for (const item of list) {
      flat.push(item)
      if (item.children) walk(item.children)
    }
  }
  walk(items)
  const queried = flat.find((item) => item.query && locationMatches(item, path, query))
  if (queried) return queried
  if (path === '/backup' && !query.tab) {
    return flat.find((item) => item.id === 'backup-records')
  }
  return flat.find((item) => !item.query && item.path === path)
}

export function resolveModule(path: string): NavModule | undefined {
  const scored = NAV_MODULES
    .map((mod) => {
      const hits = collectModuleHits(mod)
      const exact = hits.some((hit) => hit.path === path)
      const matched = mod.match?.(path) || exact
      const specificity = exact ? 2 : matched ? 1 : 0
      return { mod, specificity }
    })
    .filter((entry) => entry.specificity > 0)
    .sort((a, b) => b.specificity - a.specificity)
  return scored[0]?.mod
}

export function parseStoredLocation(value: string): NavLocation {
  const url = new URL(value, 'http://local.invalid')
  const query = Object.fromEntries(url.searchParams.entries())
  return {
    path: url.pathname,
    query: Object.keys(query).length ? query : undefined,
  }
}

export function serializeLocation(loc: NavLocation) {
  const qs = loc.query && Object.keys(loc.query).length
    ? `?${new URLSearchParams(loc.query).toString()}`
    : ''
  return `${loc.path}${qs}`
}

export function moduleContainsPath(mod: NavModule, path: string) {
  const loc = parseStoredLocation(path)
  return collectModuleHits(mod).some((hit) => hit.path === loc.path)
}

export function resolveRememberedPath(mod: NavModule, path?: string | null): NavLocation {
  if (path && moduleContainsPath(mod, path)) {
    const loc = parseStoredLocation(path)
    if (loc.path === '/backup' && !loc.query?.tab) loc.query = { tab: 'records' }
    return loc
  }
  return { path: mod.defaultPath }
}

export function resolveModuleDestination(moduleId: string, store: Map<string, string>): NavLocation | undefined {
  const mod = NAV_MODULES.find((item) => item.id === moduleId)
  if (!mod) return undefined
  return resolveRememberedPath(mod, store.get(moduleId))
}

export function searchNavigation(keyword: string): NavHit[] {
  const q = keyword.trim().toLowerCase()
  if (!q) return []
  const seen = new Set<string>()
  const hits: NavHit[] = []
  for (const hit of collectReachableDestinations()) {
    const haystack = [hit.id, hit.titleKey, hit.path, ...hit.keywords].join(' ').toLowerCase()
    if (!haystack.includes(q)) continue
    const key = `${hit.path}?${new URLSearchParams(hit.query || {}).toString()}`
    if (seen.has(key)) continue
    seen.add(key)
    hits.push(hit)
  }
  return hits
}

export function hideLocalNav(mod: NavModule | undefined, path: string) {
  if (!mod || mod.localNav === 'none') return true
  if (mod.id === 'website' && /^\/website\/websites\/.+/.test(path)) return true
  return false
}

export function resolveNetworkPartition(path: string) {
  const network = NAV_MODULES.find((mod) => mod.id === 'network')
  if (!network?.partitions) return undefined
  return network.partitions.find((partition) => partition.items.some((item) => item.path === path))
    || network.partitions.find((partition) => path.startsWith(`/${partition.id}/`))
    || network.partitions[0]
}
