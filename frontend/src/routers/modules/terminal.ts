import type { RouteRecordRaw } from 'vue-router'

const terminalRoutes: RouteRecordRaw[] = [
  {
    path: '/terminal',
    name: 'Terminal',
    component: () => import('@/views/terminal/index.vue'),
    meta: { title: 'menu.terminal', icon: 'Monitor', requiresAuth: true },
  },
]

export default terminalRoutes
