import http from '../http'
import type { SearchReq, CronjobCreateForm, CronjobUpdateForm } from '../interface'

export const createCronjob = (data: CronjobCreateForm) => http.post('/cronjobs', data)
export const updateCronjob = (data: CronjobUpdateForm) => http.post('/cronjobs/update', data)
export const deleteCronjob = (data: { id: number }) => http.post('/cronjobs/del', data)
export const searchCronjob = (data: SearchReq & { type?: string; info?: string }) =>
  http.post('/cronjobs/search', data)
export const getCronjob = (data: { id: number }) => http.post('/cronjobs/detail', data)
export const updateCronjobStatus = (data: { id: number; status: string }) =>
  http.post('/cronjobs/status', data)
export const handleOnceCronjob = (data: { id: number }) => http.post('/cronjobs/handle-once', data)
export const searchCronjobRecords = (data: SearchReq & { cronjobID: number }) =>
  http.post('/cronjobs/records', data)
