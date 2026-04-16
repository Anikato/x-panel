import http from '@/api/http'

export const getDiskInfo = () => {
  return http.get('/disk/info')
}

export const listRemoteMounts = () => {
  return http.get('/disk/remote')
}

export const browseShares = (data: {
  protocol: string
  server: string
  username?: string
  password?: string
}) => {
  return http.post('/disk/remote/browse-shares', data)
}

export const installShareDeps = (data: { package: string }) => {
  return http.post('/disk/install-share-deps', data)
}

export const mountRemote = (data: {
  protocol: string
  server: string
  sharePath: string
  mountPoint: string
  username?: string
  password?: string
  options?: string
  preset?: string
  persist?: boolean
}) => {
  return http.post('/disk/remote/mount', data)
}

export const unmountRemote = (data: { mountPoint: string; removeFstab?: boolean }) => {
  return http.post('/disk/remote/unmount', data)
}

export const listBlockDevices = () => {
  return http.get('/disk/block-devices')
}

export const mountLocal = (data: { device: string; mountPoint: string; fsType?: string; persist?: boolean }) => {
  return http.post('/disk/local/mount', data)
}

export const unmountLocal = (data: { mountPoint: string; removeFstab?: boolean }) => {
  return http.post('/disk/local/unmount', data)
}
