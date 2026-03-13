import type { RouteRecordRaw } from 'vue-router'

const nodeRoutes: RouteRecordRaw[] = [
  {
    path: '/node',
    name: 'Node',
    component: () => import('@/views/node/index.vue'),
    meta: { title: 'menu.node', requiresAuth: true },
  },
]

export default nodeRoutes
