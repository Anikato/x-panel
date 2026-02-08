import http from '@/api/http'

export const getSystemStats = () => {
  return http.get('/monitor/stats')
}
