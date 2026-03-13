import http from '../http'

export const createDatabaseServer = (data: any) => http.post('/databases/servers', data)
export const updateDatabaseServer = (data: any) => http.post('/databases/servers/update', data)
export const deleteDatabaseServer = (data: { id: number }) => http.post('/databases/servers/del', data)
export const searchDatabaseServer = (data: any) => http.post('/databases/servers/search', data)
export const testDatabaseConnection = (data: { id: number }) => http.post('/databases/servers/test', data)

export const createDatabaseInstance = (data: any) => http.post('/databases/instances', data)
export const deleteDatabaseInstance = (data: { id: number }) => http.post('/databases/instances/del', data)
export const searchDatabaseInstance = (data: any) => http.post('/databases/instances/search', data)
export const syncDatabaseInstances = (data: { id: number }) => http.post('/databases/instances/sync', data)
