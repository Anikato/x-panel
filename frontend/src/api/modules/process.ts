import http from '@/api/http'

export const searchProcess = (params: {
  pid?: number
  name?: string
  username?: string
  status?: string
  sortBy?: string
  sortDesc?: boolean
}) => {
  return http.post('/process/search', params)
}

export const stopProcess = (params: { pid: number; signal?: string }) => {
  return http.post('/process/stop', params)
}

export const getConnections = () => {
  return http.get('/process/connections')
}
