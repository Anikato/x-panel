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
