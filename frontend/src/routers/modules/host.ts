import type { RouteRecordRaw } from 'vue-router'

const hostRoutes: RouteRecordRaw[] = [
  {
    path: '/host/files',
    name: 'FileManager',
    component: () => import('@/views/host/file/index.vue'),
    meta: { title: 'menu.fileManager', icon: 'FolderOpened', requiresAuth: true },
  },
  {
    path: '/host/monitor',
    name: 'Monitor',
    component: () => import('@/views/host/monitor/index.vue'),
    meta: { title: 'menu.monitor', icon: 'DataLine', requiresAuth: true },
  },
  {
    path: '/host/firewall',
    name: 'Firewall',
    component: () => import('@/views/host/firewall/index.vue'),
    meta: { title: 'menu.firewall', icon: 'Shield', requiresAuth: true },
  },
  {
    path: '/host/process',
    name: 'ProcessManage',
    component: () => import('@/views/host/process/index.vue'),
    meta: { title: 'menu.processManage', icon: 'Cpu', requiresAuth: true },
  },
  {
    path: '/host/ssh',
    name: 'SSHManage',
    component: () => import('@/views/host/ssh/index.vue'),
    meta: { title: 'menu.sshManage', icon: 'Key', requiresAuth: true },
  },
  {
    path: '/host/disk',
    name: 'DiskManage',
    component: () => import('@/views/host/disk/index.vue'),
    meta: { title: 'menu.diskManage', icon: 'Coin', requiresAuth: true },
  },
]

export default hostRoutes
