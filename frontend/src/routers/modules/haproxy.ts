import type { RouteRecordRaw } from 'vue-router'

const haproxyRoutes: RouteRecordRaw[] = [
  {
    path: '/haproxy/status',
    name: 'HAProxyStatus',
    component: () => import('@/views/haproxy/status/index.vue'),
    meta: { title: 'haproxy.status', icon: 'Aim', requiresAuth: true },
  },
  {
    path: '/haproxy/http-lb',
    name: 'HAProxyHTTPLB',
    component: () => import('@/views/haproxy/http-lb/index.vue'),
    meta: { title: 'haproxy.httpLB', icon: 'Aim', requiresAuth: true },
  },
  {
    path: '/haproxy/tcp-lb',
    name: 'HAProxyTCPLB',
    component: () => import('@/views/haproxy/tcp-lb/index.vue'),
    meta: { title: 'haproxy.tcpLB', icon: 'Aim', requiresAuth: true },
  },
  {
    path: '/haproxy/backends',
    name: 'HAProxyBackends',
    component: () => import('@/views/haproxy/backends/index.vue'),
    meta: { title: 'haproxy.backends', icon: 'Aim', requiresAuth: true },
  },
  {
    path: '/haproxy/stats',
    name: 'HAProxyStats',
    component: () => import('@/views/haproxy/stats/index.vue'),
    meta: { title: 'haproxy.stats', icon: 'Aim', requiresAuth: true },
  },
  {
    path: '/haproxy/config',
    name: 'HAProxyConfig',
    component: () => import('@/views/haproxy/config/index.vue'),
    meta: { title: 'haproxy.config', icon: 'Aim', requiresAuth: true },
  },
  {
    path: '/haproxy/history',
    name: 'HAProxyHistory',
    component: () => import('@/views/haproxy/history/index.vue'),
    meta: { title: 'haproxy.history', icon: 'Aim', requiresAuth: true },
  },
]

export default haproxyRoutes
