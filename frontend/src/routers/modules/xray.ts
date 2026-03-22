import type { RouteRecordRaw } from 'vue-router'

const xrayRoutes: RouteRecordRaw[] = [
  {
    path: '/xray',
    name: 'XrayManage',
    component: () => import('@/views/xray/index.vue'),
    meta: { title: 'xray.title', icon: 'Connection', requiresAuth: true },
  },
]

export default xrayRoutes
