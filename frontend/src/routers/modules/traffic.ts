import type { RouteRecordRaw } from 'vue-router'

const trafficRoutes: RouteRecordRaw[] = [
  {
    path: '/traffic',
    name: 'Traffic',
    component: () => import('@/views/traffic/index.vue'),
    meta: { title: 'menu.traffic', icon: 'Odometer', requiresAuth: true },
  },
]

export default trafficRoutes
