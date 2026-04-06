import type { RouteRecordRaw } from 'vue-router'

const gostRoutes: RouteRecordRaw[] = [
  {
    path: '/gost/status',
    name: 'GostStatus',
    component: () => import('@/views/gost/status/index.vue'),
    meta: { title: 'gost.status', icon: 'Connection', requiresAuth: true },
  },
  {
    path: '/gost/forward',
    name: 'GostForward',
    component: () => import('@/views/gost/forward/index.vue'),
    meta: { title: 'gost.forward', icon: 'Sort', requiresAuth: true },
  },
  {
    path: '/gost/relay',
    name: 'GostRelay',
    component: () => import('@/views/gost/relay/index.vue'),
    meta: { title: 'gost.relay', icon: 'Share', requiresAuth: true },
  },
  {
    path: '/gost/chain',
    name: 'GostChain',
    component: () => import('@/views/gost/chain/index.vue'),
    meta: { title: 'gost.chain', icon: 'Link', requiresAuth: true },
  },
]

export default gostRoutes
