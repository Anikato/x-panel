import http from '../http'

export const listNodes = () => http.get('/nodes')
export const createNode = (data: any) => http.post('/nodes', data)
export const updateNode = (data: any) => http.post('/nodes/update', data)
export const deleteNode = (data: { id: number }) => http.post('/nodes/del', data)
export const testNodeConnection = (data: { id: number }) => http.post('/nodes/test', data)
