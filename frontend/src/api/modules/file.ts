import http from '@/api/http'

export const listFiles = (params: { path: string; showHidden?: boolean; search?: string; containSub?: boolean; sortBy?: string; sortOrder?: string }) => {
  return http.post('/files/search', params)
}

export const createFile = (params: { path: string; isDir: boolean; mode?: string }) => {
  return http.post('/files', params)
}

export const deleteFile = (params: { path: string }) => {
  return http.post('/files/del', params)
}

export const batchDeleteFile = (params: { paths: string[] }) => {
  return http.post('/files/batch-del', params)
}

export const renameFile = (params: { oldName: string; newName: string }) => {
  return http.post('/files/rename', params)
}

export const moveFile = (params: { srcPaths: string[]; dstPath: string; isCopy?: boolean; cover?: boolean }) => {
  return http.post('/files/move', params)
}

export const getFileContent = (params: { path: string }) => {
  return http.post('/files/content', params)
}

export const saveFileContent = (params: { path: string; content: string }) => {
  return http.post('/files/save', params)
}

export const changeFileMode = (params: { path: string; mode: string; sub?: boolean }) => {
  return http.post('/files/mode', params)
}

export const changeFileOwner = (params: { path: string; user: string; group: string; sub?: boolean }) => {
  return http.post('/files/owner', params)
}

export const compressFile = (params: { paths: string[]; dst: string; name: string; type?: string }) => {
  return http.post('/files/compress', params)
}

export const decompressFile = (params: { path: string; dst: string }) => {
  return http.post('/files/decompress', params)
}

export const uploadFile = (path: string, file: File) => {
  const formData = new FormData()
  formData.append('path', path)
  formData.append('file', file)
  return http.post('/files/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 600000, // 10 min for uploads
  })
}

export const getDownloadUrl = (path: string) => {
  const token = sessionStorage.getItem('token')
  return `/api/v1/files/download?path=${encodeURIComponent(path)}&token=${token}`
}

export const getFileTree = (params: { path: string }) => {
  return http.post('/files/tree', params)
}

export const getDirSize = (params: { path: string }) => {
  return http.post('/files/size', params)
}

export const getUsersAndGroups = () => {
  return http.post('/files/user/group', {})
}
