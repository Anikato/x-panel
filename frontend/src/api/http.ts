import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/routers'
import { getToken, removeToken } from '@/utils/auth'
import i18n from '@/i18n'

const t = (key: string) => String(i18n.global.t(key))

// 使用唯一请求 ID（自增）作为 key，避免同名接口互相覆盖 AbortController
let _reqIdCounter = 0
const pendingRequests = new Map<string, AbortController>()

// 不参与路由跳转取消的白名单接口（关键初始化请求）
const SKIP_CANCEL_URLS = ['/settings']

export function cancelAllPendingRequests() {
  pendingRequests.forEach((controller, key) => {
    // 白名单接口不被批量取消
    if (SKIP_CANCEL_URLS.some(url => key.includes(url))) return
    controller.abort()
  })
  // 只删除非白名单的 key
  for (const key of pendingRequests.keys()) {
    if (!SKIP_CANCEL_URLS.some(url => key.includes(url))) {
      pendingRequests.delete(key)
    }
  }
}

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
})

http.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    try {
      const globalRaw = localStorage.getItem('global')
      if (globalRaw) {
        const globalState = JSON.parse(globalRaw)
        if (globalState.currentNodeID) {
          config.headers['X-Node-ID'] = String(globalState.currentNodeID)
        }
      }
    } catch { /* ignore */ }

    const controller = new AbortController()
    config.signal = controller.signal
    // 用唯一 ID 作 key，避免同名接口覆盖彼此的 controller
    const reqId = `${config.method}:${config.url}:${++_reqIdCounter}`
    ;(config as any).__reqId = reqId
    pendingRequests.set(reqId, controller)

    return config
  },
  (error) => Promise.reject(error),
)

http.interceptors.response.use(
  (response) => {
    const reqId = (response.config as any).__reqId
    if (reqId) pendingRequests.delete(reqId)

    const res = response.data
    if (res.code === 0) {
      return res
    }
    ElMessage.error(res.message || t('commons.requestFailed'))
    return Promise.reject(new Error(res.message || t('commons.requestFailed')))
  },
  (error) => {
    if (error.config) {
      const reqId = (error.config as any).__reqId
      if (reqId) pendingRequests.delete(reqId)
    }

    if (axios.isCancel(error)) {
      return Promise.reject(error)
    }

    if (error.response) {
      const { status, data } = error.response
      if (status === 401) {
        removeToken()
        router.push('/login')
        ElMessage.error(t('commons.loginExpired'))
      } else {
        ElMessage.error(data?.message || t('commons.serverError'))
      }
    } else if (error.code !== 'ERR_CANCELED') {
      ElMessage.error(t('commons.networkError'))
    }
    return Promise.reject(error)
  },
)

export default http
