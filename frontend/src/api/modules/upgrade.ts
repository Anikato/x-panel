import http from '../http'

/** 获取当前版本信息 */
export const getCurrentVersion = () => {
  return http.get('/upgrade/current')
}

/** 检查更新 */
export const checkUpdate = (data?: { releaseUrl?: string }) => {
  return http.post('/upgrade/check', data || {})
}

/** 执行升级 */
export const doUpgrade = (data: { version: string; downloadUrl: string; checksumUrl?: string }) => {
  return http.post('/upgrade/do', data)
}

/** 获取升级日志 */
export const getUpgradeLog = () => {
  return http.get('/upgrade/log')
}
