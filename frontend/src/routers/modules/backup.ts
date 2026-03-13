import type { RouteRecordRaw } from 'vue-router'

const backupRoutes: RouteRecordRaw[] = [
  {
    path: '/backup',
    name: 'Backup',
    component: () => import('@/views/backup/index.vue'),
    meta: { title: 'menu.backup', requiresAuth: true },
  },
]

export default backupRoutes
