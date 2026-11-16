import http from '../http'

interface PageParams {
  page: number
  pageSize: number
}

export type PanelAccessKind =
  | 'entrance_blocked'
  | 'entrance_matched'
  | 'page_opened'
  | 'login_failed'
  | 'captcha_blocked'
  | 'login_succeeded'

export interface PanelAccessLog {
  id: number
  ip: string
  kind: PanelAccessKind
  hits: number
  firstSeen: string
  lastSeen: string
}

export const getPanelAccessLogs = (params: PageParams & { ip?: string; kind?: PanelAccessKind }) =>
  http.get<{ items: PanelAccessLog[]; total: number }>('/logs/panel-access', { params })

/** 登录日志分页查询 */
export const getLoginLogs = (data: PageParams) => {
  return http.post('/logs/login', data)
}

/** 操作日志分页查询 */
export const getOperationLogs = (data: PageParams) => {
  return http.post('/logs/operation', data)
}

/** 清空登录日志 */
export const cleanLoginLogs = () => {
  return http.post('/logs/login/clean')
}

/** 清空操作日志 */
export const cleanOperationLogs = () => {
  return http.post('/logs/operation/clean')
}

/** 获取面板系统日志 */
export const getSystemLog = (lines: number = 100, level: string = '', keyword: string = '') => {
  return http.get<string>('/logs/system', { params: { lines, level, keyword } })
}

/** 清空面板系统日志 */
export const cleanSystemLog = () => {
  return http.post('/logs/system/clean')
}
