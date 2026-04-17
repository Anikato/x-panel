import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import Layout from '@/layout/index.vue'
import homeRoutes from './modules/home'
import websiteRoutes from './modules/website'
import hostRoutes from './modules/host'
import terminalRoutes from './modules/terminal'
import backupRoutes from './modules/backup'
import containerRoutes from './modules/container'
import cronjobRoutes from './modules/cronjob'
import databaseRoutes from './modules/database'
import logRoutes from './modules/log'
import nodeRoutes from './modules/node'
import settingRoutes from './modules/setting'
import trafficRoutes from './modules/traffic'
import gostRoutes from './modules/gost'
import haproxyRoutes from './modules/haproxy'
import toolboxRoutes from './modules/toolbox'
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
      ...backupRoutes,
      ...containerRoutes,
      ...cronjobRoutes,
      ...databaseRoutes,
      ...logRoutes,
      ...trafficRoutes,
      ...gostRoutes,
      ...haproxyRoutes,
      ...toolboxRoutes,
      ...nodeRoutes,
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
