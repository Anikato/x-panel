import assert from 'node:assert/strict'
import test from 'node:test'
import {
  NAV_MODULES,
  collectReachableDestinations,
  findLocalNavItem,
  resolveModule,
  resolveModuleDestination,
  resolveRememberedPath,
  searchNavigation,
} from './registry.ts'
import { rememberModulePath } from './memory.ts'

const REQUIRED_PATHS = [
  '/home',
  '/host/files',
  '/terminal',
  '/website/websites',
  '/website/nginx',
  '/website/ssl',
  '/container',
  '/database',
  '/haproxy/status',
  '/haproxy/http-lb',
  '/haproxy/tcp-lb',
  '/haproxy/backends',
  '/haproxy/stats',
  '/haproxy/config',
  '/haproxy/history',
  '/gost/status',
  '/gost/forward',
  '/gost/relay',
  '/gost/chain',
  '/traffic',
  '/host/monitor',
  '/cronjob',
  '/backup',
  '/host/process',
  '/host/disk',
  '/host/users',
  '/host/system',
  '/toolbox/services',
  '/toolbox/samba',
  '/toolbox/nfs',
  '/host/firewall',
  '/host/ssh',
  '/toolbox/fail2ban',
  '/nezha-agent',
  '/log/system',
  '/log/login',
  '/log/operation',
  '/setting',
]

test('every existing operational page is reachable from the navigation registry', () => {
  const reachable = collectReachableDestinations().map((item) => item.path)
  for (const path of REQUIRED_PATHS) {
    assert.ok(reachable.includes(path), `missing nav destination for ${path}`)
  }
})

test('sidebar modules do not expand HAProxy and GOST technical pages', () => {
  const sidebarIds = NAV_MODULES.filter((mod) => mod.placement === 'main' || mod.placement === 'footer').map((mod) => mod.id)
  assert.ok(sidebarIds.includes('network'))
  assert.ok(!sidebarIds.includes('haproxy'))
  assert.ok(!sidebarIds.includes('gost'))
  assert.equal(NAV_MODULES.find((mod) => mod.id === 'network')?.partitions?.length, 2)
})

test('website detail belongs to website module and keeps that module active', () => {
  const mod = resolveModule('/website/websites/42')
  assert.equal(mod?.id, 'website')
  assert.equal(resolveModule('/website/websites')?.id, 'website')
  assert.equal(resolveModule('/website/nginx')?.id, 'website')
  assert.equal(resolveModule('/website/ssl')?.id, 'certs')
})

test('node management stays hidden from primary navigation', () => {
  assert.equal(resolveModule('/node'), undefined)
  const reachable = collectReachableDestinations().map((item) => item.path)
  assert.ok(!reachable.includes('/node'))
})

test('search finds modules by title key and keywords', () => {
  const hits = searchNavigation('文件')
  assert.ok(hits.some((hit) => hit.path === '/host/files'))
  const certHits = searchNavigation('证书')
  assert.ok(certHits.some((hit) => hit.path === '/website/ssl'))
  const fail2banHits = searchNavigation('fail2ban')
  assert.ok(fail2banHits.some((hit) => hit.path === '/toolbox/fail2ban'))
})

test('invalid remembered subpage falls back to the module default', () => {
  const network = resolveModule('/haproxy/status')
  assert.ok(network)
  const fallback = resolveRememberedPath(network, '/haproxy/not-a-page')
  assert.equal(fallback.path, network.defaultPath)
})

test('remembered HAProxy page is reused when still valid', () => {
  const network = resolveModule('/haproxy/http-lb')
  assert.ok(network)
  const remembered = resolveRememberedPath(network, '/haproxy/http-lb')
  assert.equal(remembered.path, '/haproxy/http-lb')
})

test('memory store ignores expired paths and keeps valid ones', () => {
  const store = new Map<string, string>()
  rememberModulePath('network', '/haproxy/stats', store)
  assert.equal(store.get('network'), '/haproxy/stats')
  rememberModulePath('network', '/does-not-exist', store)
  assert.equal(store.get('network'), '/haproxy/stats')
})

test('backup records query is a distinct destination from accounts', () => {
  const records = findLocalNavItem('/backup', { tab: 'records' })
  const accounts = findLocalNavItem('/backup', { tab: 'accounts' })
  assert.equal(records?.id, 'backup-records')
  assert.equal(accounts?.id, 'backup-accounts')
})

test('resolveModuleDestination uses last valid subpage', () => {
  const store = new Map<string, string>([['host', '/host/disk']])
  const dest = resolveModuleDestination('host', store)
  assert.ok(dest)
  assert.equal(dest.path, '/host/disk')
  const missing = resolveModuleDestination('missing-module', store)
  assert.equal(missing, undefined)
})
