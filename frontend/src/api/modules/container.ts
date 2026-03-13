import http from '../http'

export const getDockerStatus = () => http.get('/containers/docker/status')

// Container
export const searchContainers = (data: any) => http.post('/containers/search', data)
export const createContainer = (data: any) => http.post('/containers', data)
export const operateContainer = (data: any) => http.post('/containers/operate', data)
export const containerLogs = (data: any) => http.post('/containers/logs', data)
export const removeContainer = (data: { containerID: string }) => http.post('/containers/del', data)

// Image
export const listImages = () => http.get('/containers/image')
export const pullImage = (data: { imageName: string }) => http.post('/containers/image/pull', data)
export const removeImage = (data: { imageID: string }) => http.post('/containers/image/del', data)

// Network
export const listNetworks = () => http.get('/containers/network')
export const createNetwork = (data: any) => http.post('/containers/network', data)
export const removeNetwork = (data: { networkID: string }) => http.post('/containers/network/del', data)

// Volume
export const listVolumes = () => http.get('/containers/volume')
export const createVolume = (data: any) => http.post('/containers/volume', data)
export const removeVolume = (data: { name: string }) => http.post('/containers/volume/del', data)

// Compose
export const listCompose = () => http.get('/containers/compose')
export const createCompose = (data: any) => http.post('/containers/compose', data)
export const operateCompose = (data: any) => http.post('/containers/compose/operate', data)
