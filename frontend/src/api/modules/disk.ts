import http from '@/api/http'

export const getDiskInfo = () => {
  return http.get('/disk/info')
}

export const listRemoteMounts = () => {
  return http.get('/disk/remote')
}

export const mountRemote = (data: {
  protocol: string
  server: string
  sharePath: string
  mountPoint: string
  username?: string
  password?: string
  options?: string
}) => {
  return http.post('/disk/remote/mount', data)
}

export const unmountRemote = (data: { mountPoint: string }) => {
  return http.post('/disk/remote/unmount', data)
}
