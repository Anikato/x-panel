import type { RouteRecordRaw } from 'vue-router'

const containerRoutes: RouteRecordRaw[] = [
  {
    path: '/container',
    name: 'Container',
    component: () => import('@/views/container/index.vue'),
    meta: { title: 'menu.container', requiresAuth: true },
  },
]

export default containerRoutes
