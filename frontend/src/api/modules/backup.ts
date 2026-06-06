import http from '../http'
import type { SearchReq, BackupAccount, BackupForm } from '../interface'
import { getToken } from '@/utils/auth'

export const listBackupAccounts = () => http.get('/backup/accounts')
export const createBackupAccount = (data: Omit<BackupAccount, 'id'>) => http.post('/backup/accounts', data)
export const updateBackupAccount = (data: BackupAccount) => http.post('/backup/accounts/update', data)
export const testBackupAccount = (data: BackupAccount) => http.post('/backup/accounts/test', data)
export const deleteBackupAccount = (data: { id: number }) => http.post('/backup/accounts/del', data)

export const createBackup = (data: BackupForm) => http.post('/backup', data)
export const searchBackupRecords = (data: SearchReq & { type?: string; name?: string; status?: string; accountID?: number }) =>
  http.post('/backup/records/search', data)
export const deleteBackupRecord = (data: { id: number }) => http.post('/backup/records/del', data)

export interface BackupStorageReq {
  accountID: number
  prefix?: string
  path?: string
  content?: string
}

export interface BackupStorageObject {
  name: string
  path: string
}

export const listStorageObjects = (data: BackupStorageReq) => http.post('/backup/storage/list', data)
export const readStorageObject = (data: BackupStorageReq) => http.post('/backup/storage/read', data)
export const saveStorageObject = (data: BackupStorageReq) => http.post('/backup/storage/save', data)
export const deleteStorageObject = (data: BackupStorageReq) => http.post('/backup/storage/delete', data)

export const uploadStorageObject = (data: { accountID: number; prefix?: string; path?: string; file: File }) => {
  const form = new FormData()
  form.append('accountID', String(data.accountID))
  form.append('prefix', data.prefix || '')
  form.append('path', data.path || '')
  form.append('file', data.file)
  return http.post('/backup/storage/upload', form, { headers: { 'Content-Type': 'multipart/form-data' } })
}

export const downloadStorageObject = async (data: BackupStorageReq) => {
  const res = await fetch('/api/v1/backup/storage/download', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`,
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.blob()
}
