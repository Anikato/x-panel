import type { Router } from 'vue-router'
import { cancelAllPendingRequests } from '@/api/http'
import { getToken } from '@/utils/auth'

export function setupGuard(router: Router) {
  router.beforeEach((_to, _from, next) => {
    cancelAllPendingRequests()

    const token = getToken()

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
