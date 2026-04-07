import { defineStore } from 'pinia'

export type ThemeMode = 'dark' | 'light' | 'auto'

export interface ServerInfo {
  hostname: string
  platform: string
  platformVersion: string
  kernelArch: string
  virtualization: string
  uptime: number
}

export const useGlobalStore = defineStore('global', {
  state: () => ({
    isLogin: false,
    loading: false,
    menuCollapse: false,
    panelName: 'X-Panel',
    theme: 'dark' as ThemeMode,
    accentKey: 'neon',
    accentCustom: '',
    version: '',
    currentNodeID: 0,
    currentNodeName: '',
    serverInfo: null as ServerInfo | null,
  }),
  actions: {
    setLogin(status: boolean) {
      this.isLogin = status
    },
    setLoading(status: boolean) {
      this.loading = status
    },
    toggleMenuCollapse() {
      this.menuCollapse = !this.menuCollapse
    },
    setPanelName(name: string) {
      this.panelName = name
    },
    setVersion(ver: string) {
      this.version = ver
    },
    setCurrentNode(id: number, name: string) {
      this.currentNodeID = id
      this.currentNodeName = name
    },
    setTheme(mode: ThemeMode) {
      this.theme = mode
    },
    cycleTheme() {
      const order: ThemeMode[] = ['dark', 'light', 'auto']
      const idx = order.indexOf(this.theme)
      this.theme = order[(idx + 1) % order.length]
    },
    setAccent(key: string, customHex?: string) {
      this.accentKey = key
      this.accentCustom = customHex || ''
    },
    setServerInfo(info: ServerInfo) {
      this.serverInfo = info
    },
  },
  persist: {
    pick: ['isLogin', 'menuCollapse', 'panelName', 'theme', 'accentKey', 'accentCustom', 'version', 'currentNodeID', 'currentNodeName'],
  },
})
