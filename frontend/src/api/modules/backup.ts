import http from '../http'

export const listBackupAccounts = () => http.get('/backup/accounts')
export const createBackupAccount = (data: any) => http.post('/backup/accounts', data)
export const updateBackupAccount = (data: any) => http.post('/backup/accounts/update', data)
export const deleteBackupAccount = (data: { id: number }) => http.post('/backup/accounts/del', data)

export const createBackup = (data: any) => http.post('/backup', data)
export const searchBackupRecords = (data: any) => http.post('/backup/records/search', data)
export const deleteBackupRecord = (data: { id: number }) => http.post('/backup/records/del', data)
