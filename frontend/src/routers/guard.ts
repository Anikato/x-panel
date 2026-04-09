import type { Router } from 'vue-router'
import { cancelAllPendingRequests } from '@/api/http'

export function setupGuard(router: Router) {
  router.beforeEach((_to, _from, next) => {
    cancelAllPendingRequests()

    const token = sessionStorage.getItem('token')

    if (_to.meta.requiresAuth === false) {
      next()
      return
    }

    if (!token) {
      next({ path: '/login', query: { redirect: _to.fullPath } })
      return
    }

    next()
  })
}
