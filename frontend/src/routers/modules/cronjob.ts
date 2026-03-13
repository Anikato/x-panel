import type { RouteRecordRaw } from 'vue-router'

const cronjobRoutes: RouteRecordRaw[] = [
  {
    path: '/cronjob',
    name: 'Cronjob',
    component: () => import('@/views/cronjob/index.vue'),
    meta: { title: 'menu.cronjob', requiresAuth: true },
  },
]

export default cronjobRoutes
