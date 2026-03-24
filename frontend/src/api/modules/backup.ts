import http from '../http'
import type { SearchReq, BackupAccount, BackupForm } from '../interface'

export const listBackupAccounts = () => http.get('/backup/accounts')
export const createBackupAccount = (data: Omit<BackupAccount, 'id'>) => http.post('/backup/accounts', data)
export const updateBackupAccount = (data: BackupAccount) => http.post('/backup/accounts/update', data)
export const deleteBackupAccount = (data: { id: number }) => http.post('/backup/accounts/del', data)

export const createBackup = (data: BackupForm) => http.post('/backup', data)
export const searchBackupRecords = (data: SearchReq & { type?: string }) =>
  http.post('/backup/records/search', data)
export const deleteBackupRecord = (data: { id: number }) => http.post('/backup/records/del', data)
