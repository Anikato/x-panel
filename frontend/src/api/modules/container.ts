import http from '../http'
import type {
  ContainerSearchReq,
  ContainerCreateForm,
  ContainerOperateReq,
  NetworkCreateForm,
  VolumeCreateForm,
} from '../interface'

export const getDockerStatus = () => http.get('/containers/docker/status')
export const installDocker = () => http.post('/containers/docker/install')
export const getDockerInstallLog = () => http.get('/containers/docker/install/log')

export const searchContainers = (data: ContainerSearchReq) => http.post('/containers/search', data)
export const createContainer = (data: ContainerCreateForm) => http.post('/containers', data)
export const operateContainer = (data: ContainerOperateReq) => http.post('/containers/operate', data)
export const containerLogs = (data: { containerID: string; tail: string }) => http.post('/containers/logs', data)
export const removeContainer = (data: { containerID: string }) => http.post('/containers/del', data)

export const listImages = () => http.get('/containers/image')
export const pullImage = (data: { imageName: string }) => http.post('/containers/image/pull', data)
export const removeImage = (data: { imageID: string }) => http.post('/containers/image/del', data)

export const listNetworks = () => http.get('/containers/network')
export const createNetwork = (data: NetworkCreateForm) => http.post('/containers/network', data)
export const removeNetwork = (data: { networkID: string }) => http.post('/containers/network/del', data)

export const listVolumes = () => http.get('/containers/volume')
export const createVolume = (data: VolumeCreateForm) => http.post('/containers/volume', data)
export const removeVolume = (data: { name: string }) => http.post('/containers/volume/del', data)

export const listCompose = () => http.get('/containers/compose')
export const createCompose = (data: { name: string; path: string; content: string }) =>
  http.post('/containers/compose', data)
export const operateCompose = (data: { name: string; operation: string }) =>
  http.post('/containers/compose/operate', data)

// 新增功能
export const inspectDocker = (data: { type: string; id: string }) => http.post('/containers/inspect', data)
export const pruneDocker = (data: { pruneType: string; withAll?: boolean }) => http.post('/containers/prune', data)
export const renameContainer = (data: { containerID: string; newName: string }) => http.post('/containers/rename', data)
export const cleanContainerLog = (data: { containerID: string }) => http.post('/containers/log/clean', data)
export const commitContainer = (data: { containerID: string; newImageName: string; comment?: string; pause?: boolean }) =>
  http.post('/containers/commit', data)
export const loadDockerMirrors = () => http.get('/containers/docker/mirrors')
export const updateDockerMirrors = (data: { mirrors: string[] }) => http.post('/containers/docker/mirrors', data)
