import { defineStore } from 'pinia'

export type ThemeMode = 'dark' | 'light' | 'auto'
export type BgPreset = 'abyss' | 'void' | 'tinted' | 'cosmos' | 'warm'
export type UiFont = 'system' | 'inter' | 'noto' | 'lxgw'
export type UiDensity = 'compact' | 'default' | 'comfortable'
export type BorderRadiusPreset = 'sharp' | 'default' | 'rounded'
export type CardBorderStyle = 'accent-left' | 'full' | 'shadow-only'
export type SidebarWidthPreset = 'narrow' | 'default' | 'wide'

export interface ServerInfo {
  hostname: string
  platform: string
  platformVersion: string
  kernelArch: string
  virtualization: string
  uptime: number
  timezone: string
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

    bgPreset: 'abyss' as BgPreset,
    uiFont: 'system' as UiFont,
    uiDensity: 'default' as UiDensity,
    borderRadiusPreset: 'default' as BorderRadiusPreset,
    reduceMotion: false,
    termTheme: 'default' as string,
    termFont: 'jetbrains' as string,
    termFontSize: 14,
    termBgOpacity: 1.0,
    cardBorderStyle: 'accent-left' as CardBorderStyle,
    sidebarWidth: 'default' as SidebarWidthPreset,
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
    pick: [
      'isLogin', 'menuCollapse', 'panelName', 'theme', 'accentKey', 'accentCustom',
      'version', 'currentNodeID', 'currentNodeName',
      'bgPreset', 'uiFont', 'uiDensity', 'borderRadiusPreset', 'reduceMotion',
      'termTheme', 'termFont', 'termFontSize', 'termBgOpacity',
      'cardBorderStyle', 'sidebarWidth',
    ],
  },
})
