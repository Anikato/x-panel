import http from '../http'

/** 获取面板设置 */
export const getSettingInfo = () => {
  return http.get('/settings')
}

/** 更新设置项 */
export const updateSetting = (data: { key: string; value: string }) => {
  return http.post('/settings/update', data)
}
