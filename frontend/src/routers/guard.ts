import type { Router } from 'vue-router'

export function setupGuard(router: Router) {
  router.beforeEach((_to, _from, next) => {
    const token = sessionStorage.getItem('token')

    // 不需要认证的页面直接放行
    if (_to.meta.requiresAuth === false) {
      next()
      return
    }

    // 需要认证但没有 token → 跳转登录
    if (!token) {
      next({ path: '/login', query: { redirect: _to.fullPath } })
      return
    }

    next()
  })
}
