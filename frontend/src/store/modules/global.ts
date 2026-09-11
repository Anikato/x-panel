import { defineStore } from 'pinia'
import { getSettingInfo } from '@/api/modules/setting'
import { normalizeTerminalCwd } from '@/utils/terminal-cwd'

export type ThemeMode = 'dark' | 'light' | 'auto'
export type BgPreset = 'graphite' | 'abyss' | 'void' | 'tinted' | 'cosmos' | 'warm'
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
    panelName: '',
    theme: 'dark' as ThemeMode,
    accentKey: 'steel',
    accentCustom: '',
    version: '',
    currentNodeID: 0,
    currentNodeName: '',
    serverInfo: null as ServerInfo | null,

    bgPreset: 'graphite' as BgPreset,
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
    showServerClock: true,
    dashboardRefreshInterval: 5000,
    floatTermVisible: false,   // 悬浮终端显隐状态（不持久化，刷新后默认关闭）
    floatTermMinimized: false, // 悬浮终端最小化状态
    terminalTrigger: null as { cwd?: string | null } | null,
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
    setPanelName(name?: string) {
      const nextName = (name || '').trim()
      if (nextName) this.panelName = nextName
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
    openFloatTerminal(cwd?: string | null) {
      this.floatTermVisible = true
      this.floatTermMinimized = false
      this.terminalTrigger = { cwd: normalizeTerminalCwd(cwd) }
    },

    getAppearanceKeys() {
      return {
        bgPreset: this.bgPreset,
        uiFont: this.uiFont,
        uiDensity: this.uiDensity,
        borderRadiusPreset: this.borderRadiusPreset,
        reduceMotion: this.reduceMotion,
        termTheme: this.termTheme,
        termFont: this.termFont,
        termFontSize: this.termFontSize,
        termBgOpacity: this.termBgOpacity,
        cardBorderStyle: this.cardBorderStyle,
        sidebarWidth: this.sidebarWidth,
        accentKey: this.accentKey,
        accentCustom: this.accentCustom,
        theme: this.theme,
        showServerClock: this.showServerClock,
        dashboardRefreshInterval: this.dashboardRefreshInterval,
      }
    },
    async loadPanelNameFromBackend() {
      try {
        const res = await getSettingInfo()
        this.setPanelName(res.data?.panelName)
      } catch {
        /* 网络不可用时保留本地已知面板名称 */
      }
    },
  },
  persist: {
    paths: [
      'isLogin', 'menuCollapse', 'panelName',
      'version', 'currentNodeID', 'currentNodeName',
      'showServerClock', 'dashboardRefreshInterval',
    ],
  },
})
