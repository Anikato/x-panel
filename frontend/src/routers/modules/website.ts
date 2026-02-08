import type { RouteRecordRaw } from 'vue-router'

const websiteRoutes: RouteRecordRaw[] = [
  {
    path: '/website/nginx',
    name: 'NginxManage',
    component: () => import('@/views/website/nginx/index.vue'),
    meta: { title: 'nginx.title', icon: 'Connection', requiresAuth: true },
  },
  {
    path: '/website/ssl',
    name: 'SSLManage',
    component: () => import('@/views/website/ssl/index.vue'),
    meta: { title: 'ssl.title', icon: 'Lock', requiresAuth: true },
  },
]

export default websiteRoutes
