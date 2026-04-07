import type { RouteRecordRaw } from 'vue-router'

const toolboxRoutes: RouteRecordRaw[] = [
  {
    path: '/toolbox/samba',
    name: 'ToolboxSamba',
    component: () => import('@/views/toolbox/samba/index.vue'),
    meta: { title: 'toolbox.samba', icon: 'Share', requiresAuth: true },
  },
  {
    path: '/toolbox/nfs',
    name: 'ToolboxNfs',
    component: () => import('@/views/toolbox/nfs/index.vue'),
    meta: { title: 'toolbox.nfs', icon: 'FolderOpened', requiresAuth: true },
  },
]

export default toolboxRoutes
