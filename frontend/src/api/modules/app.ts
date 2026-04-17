import http from '../http'
import type { PageResult } from '../interface'

export namespace App {
  // 应用商店
  export interface AppSearchReq {
    page: number
    pageSize: number
    name?: string
    type?: string
    tags?: string[]
  }

  export interface AppDTO {
    id: number
    name: string
    key: string
    shortDescZh: string
    shortDescEn: string
    description: string
    icon: string
    type: string
    status: string
    crossVersionUpdate: boolean
    limitNum: number
    website: string
    github: string
    document: string
    recommend: number
    resource: string
    architectures: string[]
    memoryRequired: number
    gpuSupport: boolean
    requiredPanelVersion: string
    tags: string[]
    versions: string[]
    installedCount: number
    createdAt: string
  }

  export interface AppDetailDTO {
    id: number
    appId: number
    version: string
    params: Record<string, any>
    dockerCompose: string
    status: string
    downloadUrl: string
  }

  export interface TagDTO {
    key: string
    name: string
    sort: number
  }

  export interface AppSyncReq {
    force: boolean
  }

  // 应用安装
  export interface AppInstallReq {
    name: string
    appId: number
    appDetailId: number
    params: Record<string, any>
  }

  // 应用导入
  export interface AppImportReq {
    name: string
    backupPath: string
    appKey?: string
    version?: string
  }

  // 导入任务
  export interface AppImportTask {
    id: number
    name: string
    backupPath: string
    appKey: string
    version: string
    status: 'pending' | 'running' | 'success' | 'failed'
    progress: number
    currentStep: string
    message: string
    startedAt?: string
    completedAt?: string
    createdAt: string
  }

  export interface AppInstallSearchReq {
    page: number
    pageSize: number
    name?: string
    type?: string
  }

  export interface AppInstallDTO {
    id: number
    name: string
    appId: number
    appKey: string
    appName: string
    appIcon: string
    version: string
    status: string
    message: string
    containerName: string
    httpPort: number
    httpsPort: number
    webUi: string
    favorite: boolean
    sortOrder: number
    installedAt: string
    canUpdate: boolean
    latestVersion: string
  }

  export interface AppOperateReq {
    installId: number
    operation: 'start' | 'stop' | 'restart'
  }

  export interface AppUninstallReq {
    installId: number
    deleteData: boolean
    forceDelete: boolean
  }

  export interface AppUpdateReq {
    installId: number
    appDetailId: number
  }

  // 应用备份
  export interface AppBackupReq {
    installId: number
    backupName?: string
    description?: string
  }

  export interface AppRestoreReq {
    installId: number
    backupId: number
  }

  export interface AppBackupDTO {
    id: number
    appInstallId: number
    appName: string
    backupName: string
    backupPath: string
    backupType: string
    size: number
    sizeStr: string
    checksum: string
    status: string
    message: string
    createdAt: string
  }
}

// 应用商店 API
export const syncAppStore = (params: App.AppSyncReq) => {
  return http.post('/apps/sync', params)
}

export const searchApps = (params: App.AppSearchReq) => {
  return http.post<PageResult<App.AppDTO>>('/apps/search', params)
}

export const getAppTags = () => {
  return http.get<App.TagDTO[]>('/apps/tags')
}

export const getAppByKey = (key: string) => {
  return http.get<App.AppDTO>(`/apps/${key}`)
}

export const getAppDetail = (appId: number, version: string) => {
  return http.get<App.AppDetailDTO>('/apps/detail', { params: { appId, version } })
}

// 应用安装 API
export const installApp = (params: App.AppInstallReq) => {
  return http.post('/apps/install', params)
}

export const importApp = (params: App.AppImportReq) => {
  return http.post('/apps/import', params)
}

export const getImportProgress = (name: string) => {
  return http.get<App.AppImportTask>(`/apps/import/progress/${name}`)
}

export const getImportTasks = () => {
  return http.get<App.AppImportTask[]>('/apps/import/tasks')
}

export const searchInstalled = (params: App.AppInstallSearchReq) => {
  return http.post<PageResult<App.AppInstallDTO>>('/apps/installed/search', params)
}

export const getInstalled = (id: number) => {
  return http.get<App.AppInstallDTO>(`/apps/installed/${id}`)
}

export const getAppLogs = (id: number, lines?: number) => {
  return http.get<string>(`/apps/installed/${id}/logs`, { params: { lines: lines || 100 } })
}

export const operateApp = (params: App.AppOperateReq) => {
  return http.post('/apps/operate', params)
}

export const uninstallApp = (params: App.AppUninstallReq) => {
  return http.post('/apps/uninstall', params)
}

export const updateApp = (params: App.AppUpdateReq) => {
  return http.post('/apps/update', params)
}

// 应用备份 API
export const backupApp = (params: App.AppBackupReq) => {
  return http.post('/apps/backup', params)
}

export const restoreApp = (params: App.AppRestoreReq) => {
  return http.post('/apps/restore', params)
}

export const searchBackups = (params: App.AppInstallSearchReq) => {
  return http.post<PageResult<App.AppBackupDTO>>('/apps/backups/search', params)
}

export const deleteBackups = (ids: number[]) => {
  return http.post('/apps/backups/del', { ids })
}
