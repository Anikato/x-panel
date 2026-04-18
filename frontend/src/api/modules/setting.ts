import http from '../http'

/** 获取面板设置 */
export const getSettingInfo = () => {
  return http.get('/settings')
}

/** 更新设置项 */
export const updateSetting = (data: { key: string; value: string }) => {
  return http.post('/settings/update', data)
}

/** 更新面板端口 */
export const updatePort = (data: { port: string }) => {
  return http.post('/settings/port/update', data)
}

/** 测试代理连通性 */
export const testProxy = (data: { address: string }) => {
  return http.post('/settings/proxy/test', data)
}

/** 重启服务器 */
export const rebootServer = () => {
  return http.post('/settings/reboot')
}

/** 关闭服务器 */
export const shutdownServer = () => {
  return http.post('/settings/shutdown')
}

/** 重启面板 */
export const restartPanel = () => {
  return http.post('/settings/restart-panel')
}

/** 获取面板 HTTPS 配置 */
export const getPanelSSL = () => {
  return http.get('/settings/panel-ssl')
}

/** 将面板 HTTPS 切换为证书管理中的指定证书 */
export const updatePanelSSL = (data: { certificateId: number }) => {
  return http.post('/settings/panel-ssl/update', data)
}
