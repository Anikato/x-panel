import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import Layout from '@/layout/index.vue'
import homeRoutes from './modules/home'
import websiteRoutes from './modules/website'
import hostRoutes from './modules/host'
import terminalRoutes from './modules/terminal'
import logRoutes from './modules/log'
import settingRoutes from './modules/setting'
import { setupGuard } from './guard'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/init',
    name: 'Init',
    component: () => import('@/views/init/index.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: Layout,
    redirect: '/home',
    children: [
      ...homeRoutes,
      ...websiteRoutes,
      ...hostRoutes,
      ...terminalRoutes,
      ...logRoutes,
      ...settingRoutes,
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/home',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

setupGuard(router)

export default router
