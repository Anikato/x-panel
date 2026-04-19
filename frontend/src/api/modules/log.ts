import http from '../http'

interface PageParams {
  page: number
  pageSize: number
}

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
