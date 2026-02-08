import http from '../http'

/** 检查面板是否已初始化 */
export const checkIsInit = () => {
  return http.get('/auth/is-init')
}

/** 获取登录页设置（面板名称、主题等） */
export const getLoginSetting = () => {
  return http.get('/auth/setting')
}

/** 初始化管理员用户 */
export const initUser = (data: { name: string; password: string }) => {
  return http.post('/auth/init', data)
}

/** 用户登录 */
export const login = (data: { name: string; password: string }) => {
  return http.post('/auth/login', data)
}

/** 退出登录 */
export const logout = () => {
  return http.post('/auth/logout')
}

/** 修改密码 */
export const updatePassword = (data: { oldPassword: string; newPassword: string }) => {
  return http.post('/auth/password', data)
}
