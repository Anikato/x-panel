import http from '@/api/http'
import { getToken } from '@/utils/auth'

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

export const moveFile = (params: { srcPaths: string[]; dstPath: string; isCopy?: boolean; cover?: boolean; conflictPolicy?: string }) => {
  return http.post('/files/move', params)
}

export const getFileContent = (params: { path: string }) => {
  return http.post('/files/content', params)
}

export const saveFileContent = (params: { path: string; content: string }) => {
  return http.post('/files/save', params, { timeout: 300000 })
}

export const changeFileMode = (params: { path: string; mode: string; sub?: boolean }) => {
  return http.post('/files/mode', params)
}

export const changeFileOwner = (params: { path: string; user: string; group: string; sub?: boolean }) => {
  return http.post('/files/owner', params)
}

export const compressFile = (params: { paths: string[]; dst: string; name: string; type?: string; excludes?: string[] }) => {
  return http.post('/files/compress', params)
}

export const decompressFile = (params: { path: string; dst: string; extractToSameDir?: boolean; conflictPolicy?: string }) => {
  return http.post('/files/decompress', params)
}

export const listArchive = (params: { path: string }) => {
  return http.post<{ entries: string[]; total: number; unsafeEntries: string[] }>('/files/archive/list', params)
}

export const uploadFile = (path: string, file: File, onProgress?: (percent: number) => void) => {
  const formData = new FormData()
  formData.append('path', path)
  formData.append('file', file)
  return http.post('/files/upload', formData, {
    headers: { 'Content-Type': undefined },
    timeout: 0, // 不设超时，大文件上传时间不可预知
    onUploadProgress: onProgress
      ? (e: { loaded: number; total?: number }) => {
          if (e.total) onProgress(Math.round((e.loaded / e.total) * 100))
        }
      : undefined,
  })
}

export const getDownloadUrl = (path: string) => {
  const token = getToken()
  return `/api/v1/files/download?path=${encodeURIComponent(path)}&token=${token}`
}

export const getFileTree = (params: { path: string }) => {
  return http.post('/files/tree', params)
}

export const getDirSize = (params: { path: string }) => {
  return http.post('/files/size', params)
}

export const getUsersAndGroups = () => {
  return http.post<{
    users: { username: string; group: string; uid: string; gid: string; system?: boolean }[]
    groups: string[]
  }>('/files/user/group', {})
}

// ===================== 异步任务 =====================

export const getFileTaskStatus = (taskID: string) => {
  return http.get('/files/task', { params: { id: taskID } })
}

export const listFileTasks = () => {
  return http.get('/files/tasks')
}

export const cancelFileTask = (taskID: string) => {
  return http.post('/files/task/cancel', null, { params: { id: taskID } })
}

/**
 * 轮询文件操作任务直到完成
 */
export const pollFileTask = async (
  taskID: string,
  interval = 2000,
  timeout = 24 * 60 * 60 * 1000,
): Promise<{ status: string; message?: string }> => {
  const start = Date.now()
  while (Date.now() - start < timeout) {
    const res: any = await getFileTaskStatus(taskID)
    const task = res.data
    if (task.status === 'success') return task
    if (task.status === 'failed') throw new Error(task.message || '操作失败')
    await new Promise((r) => setTimeout(r, interval))
  }
  throw new Error('操作超时')
}

// ===================== 冲突检测 =====================

export const checkConflict = (params: { srcPaths: string[]; dstPath: string }) => {
  return http.post('/files/check-conflict', params)
}
