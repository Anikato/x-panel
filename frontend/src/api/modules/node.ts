import http from '../http'
import type { NodeForm } from '../interface'

export const listNodes = () => http.get('/nodes')
export const createNode = (data: NodeForm) => http.post('/nodes', data)
export const updateNode = (data: NodeForm) => http.post('/nodes/update', data)
export const deleteNode = (data: { id: number }) => http.post('/nodes/del', data)
export const testNodeConnection = (data: { id: number }) => http.post('/nodes/test', data)
export const testSSH = (data: {
  sshHost: string
  sshPort: number
  sshUser: string
  sshPassword: string
}) => http.post('/nodes/ssh-test', data)
export const agentAction = (data: { id: number; action: string }) =>
  http.post('/nodes/agent-action', data, { timeout: 300000 })
