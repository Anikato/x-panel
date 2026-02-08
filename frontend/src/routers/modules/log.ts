import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/log/login',
    name: 'LoginLog',
    component: () => import('@/views/log/login/index.vue'),
    meta: { title: 'menu.loginLog', icon: 'Document' },
  },
  {
    path: '/log/operation',
    name: 'OperationLog',
    component: () => import('@/views/log/operation/index.vue'),
    meta: { title: 'menu.operationLog', icon: 'Notebook' },
  },
]

export default routes
