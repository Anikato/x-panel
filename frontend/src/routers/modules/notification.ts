import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/notifications',
    name: 'Notifications',
    component: () => import('@/views/notification/index.vue'),
    meta: { title: 'notification.title', icon: 'Bell', requiresAuth: true },
  },
]

export default routes
