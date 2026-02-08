import http from '@/api/http'

export const getDiskInfo = () => {
  return http.get('/disk/info')
}
