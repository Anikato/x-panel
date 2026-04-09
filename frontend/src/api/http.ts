import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '@/routers'

const pendingRequests = new Map<string, AbortController>()

function getRequestKey(config: any): string {
  return `${config.method}:${config.url}`
}

export function cancelAllPendingRequests() {
  pendingRequests.forEach((controller) => {
    controller.abort()
  })
  pendingRequests.clear()
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
    const token = sessionStorage.getItem('token')
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
    const key = getRequestKey(config)
    pendingRequests.set(key, controller)

    return config
  },
  (error) => Promise.reject(error),
)

http.interceptors.response.use(
  (response) => {
    const key = getRequestKey(response.config)
    pendingRequests.delete(key)

    const res = response.data
    if (res.code === 0) {
      return res
    }
    ElMessage.error(res.message || '请求失败')
    return Promise.reject(new Error(res.message || '请求失败'))
  },
  (error) => {
    if (error.config) {
      const key = getRequestKey(error.config)
      pendingRequests.delete(key)
    }

    if (axios.isCancel(error)) {
      return Promise.reject(error)
    }

    if (error.response) {
      const { status, data } = error.response
      if (status === 401) {
        sessionStorage.removeItem('token')
        router.push('/login')
        ElMessage.error('登录已过期，请重新登录')
      } else {
        ElMessage.error(data?.message || '服务器错误')
      }
    } else if (error.code !== 'ERR_CANCELED') {
      ElMessage.error('网络连接失败')
    }
    return Promise.reject(error)
  },
)

export default http
