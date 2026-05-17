import type { Router } from 'vue-router'
import { cancelAllPendingRequests } from '@/api/http'
import { getToken } from '@/utils/auth'

export function setupGuard(router: Router) {
  router.beforeEach((to, _from, next) => {
    cancelAllPendingRequests()

    const token = getToken()

    if (token && to.meta.guestOnly) {
      next({ path: '/home', replace: true })
      return
    }

    if (to.meta.requiresAuth === false) {
      next()
      return
    }

    if (!token) {
      next({ path: '/login', query: { redirect: to.fullPath } })
      return
    }

    next()
  })
}
