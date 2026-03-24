import http from '../http'
import type { SearchReq, DatabaseServerForm, DatabaseInstanceForm } from '../interface'

export const createDatabaseServer = (data: DatabaseServerForm) => http.post('/databases/servers', data)
export const updateDatabaseServer = (data: DatabaseServerForm) => http.post('/databases/servers/update', data)
export const deleteDatabaseServer = (data: { id: number }) => http.post('/databases/servers/del', data)
export const searchDatabaseServer = (data: SearchReq & { type: string }) =>
  http.post('/databases/servers/search', data)
export const testDatabaseConnection = (data: { id: number }) => http.post('/databases/servers/test', data)

export const createDatabaseInstance = (data: DatabaseInstanceForm) => http.post('/databases/instances', data)
export const deleteDatabaseInstance = (data: { id: number }) => http.post('/databases/instances/del', data)
export const searchDatabaseInstance = (data: SearchReq & { serverID: number }) =>
  http.post('/databases/instances/search', data)
export const syncDatabaseInstances = (data: { id: number }) => http.post('/databases/instances/sync', data)
export const changeInstancePassword = (data: { id: number; password: string }) =>
  http.post('/databases/instances/password', data)
export const backupDatabaseInstance = (data: { id: number }) => http.post('/databases/instances/backup', data)
