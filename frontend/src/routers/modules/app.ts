import type { RouteRecordRaw } from 'vue-router'

const appRoutes: RouteRecordRaw[] = [
  {
    path: '/app/store',
    name: 'AppStore',
    component: () => import('@/views/app/store/index.vue'),
    meta: { title: 'app.store', icon: 'ShoppingCart', requiresAuth: true },
  },
  {
    path: '/app/installed',
    name: 'AppInstalled',
    component: () => import('@/views/app/installed/index.vue'),
    meta: { title: 'app.installed', icon: 'Box', requiresAuth: true },
  },
  {
    path: '/app/backups',
    name: 'AppBackups',
    component: () => import('@/views/app/backups/index.vue'),
    meta: { title: 'app.backups', icon: 'FolderChecked', requiresAuth: true },
  },
]

export default appRoutes
