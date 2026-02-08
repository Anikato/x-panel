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

// --- 安装 Nginx（从预编译仓库下载）---
export const installNginx = (version: string) => {
  return http.post('/nginx/install', { version })
}

// --- 安装进度 ---
export const getInstallProgress = () => {
  return http.get('/nginx/install/progress')
}

// --- 卸载 Nginx ---
export const uninstallNginx = () => {
  return http.post('/nginx/uninstall')
}

// --- 获取可用的预编译版本列表 ---
export const listNginxVersions = () => {
  return http.get('/nginx/versions')
}
