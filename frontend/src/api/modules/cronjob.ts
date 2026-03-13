import http from '../http'

export const createCronjob = (data: any) => http.post('/cronjobs', data)
export const updateCronjob = (data: any) => http.post('/cronjobs/update', data)
export const deleteCronjob = (data: { id: number }) => http.post('/cronjobs/del', data)
export const searchCronjob = (data: any) => http.post('/cronjobs/search', data)
export const getCronjob = (data: { id: number }) => http.post('/cronjobs/detail', data)
export const updateCronjobStatus = (data: { id: number; status: string }) => http.post('/cronjobs/status', data)
export const handleOnceCronjob = (data: { id: number }) => http.post('/cronjobs/handle-once', data)
export const searchCronjobRecords = (data: any) => http.post('/cronjobs/records', data)
