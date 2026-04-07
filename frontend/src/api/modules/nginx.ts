import http from '@/api/http'

// --- Nginx 状态 ---
export const getNginxStatus = () => {
  return http.get('/nginx/status')
}

// --- Nginx 操作（start/stop/reload/reopen/quit）---
export const operateNginx = (operation: string) => {
  return http.post('/nginx/operate', { operation })
}

// --- 配置测试 ---
export const testNginxConfig = () => {
  return http.get('/nginx/config-test')
}

// --- 安装 Nginx ---
export const installNginx = (method: string, version?: string) => {
  return http.post('/nginx/install', { method, version })
}

// --- 安装进度 ---
export const getInstallProgress = () => {
  return http.get('/nginx/install/progress')
}

// --- 卸载 Nginx ---
export const uninstallNginx = (forceCleanup = false, mode?: string) => {
  return http.post('/nginx/uninstall', { forceCleanup, mode })
}

// --- 获取可用的预编译版本列表 ---
export const listNginxVersions = () => {
  return http.get('/nginx/versions')
}

// --- 检查 Nginx 更新 ---
export const checkNginxUpdate = () => {
  return http.get('/nginx/update/check')
}

// --- 升级 Nginx ---
export const upgradeNginx = (version?: string) => {
  return http.post('/nginx/update/upgrade', { version })
}

// --- 设置 Nginx 开机自启 ---
export const setNginxAutoStart = (enable: boolean) => {
  return http.post('/nginx/autostart', { enable })
}
