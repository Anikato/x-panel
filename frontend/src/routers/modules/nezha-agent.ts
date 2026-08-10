import type { RouteRecordRaw } from 'vue-router'

const nezhaAgentRoutes: RouteRecordRaw[] = [
  {
    path: '/nezha-agent',
    name: 'NezhaAgent',
    component: () => import('@/views/nezha-agent/index.vue'),
    meta: { title: 'menu.nezhaAgent', requiresAuth: true },
  },
]

export default nezhaAgentRoutes
