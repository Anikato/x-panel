import type { RouteRecordRaw } from 'vue-router'

const databaseRoutes: RouteRecordRaw[] = [
  {
    path: '/database',
    name: 'Database',
    component: () => import('@/views/database/index.vue'),
    meta: { title: 'menu.database', requiresAuth: true },
  },
]

export default databaseRoutes
