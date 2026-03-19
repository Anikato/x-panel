import { defineStore } from 'pinia'

export type AccentColor = 'cyan' | 'blue' | 'purple' | 'green' | 'orange' | 'pink'

const accentPresets: Record<AccentColor, { primary: string; hover: string; muted: string; glow: string; secondary: string }> = {
  cyan:   { primary: '#22d3ee', hover: '#06b6d4', muted: 'rgba(34,211,238,0.15)',  glow: '0 0 20px rgba(34,211,238,0.2)',  secondary: '#818cf8' },
  blue:   { primary: '#3b82f6', hover: '#2563eb', muted: 'rgba(59,130,246,0.15)',  glow: '0 0 20px rgba(59,130,246,0.2)',  secondary: '#8b5cf6' },
  purple: { primary: '#8b5cf6', hover: '#7c3aed', muted: 'rgba(139,92,246,0.15)',  glow: '0 0 20px rgba(139,92,246,0.2)',  secondary: '#ec4899' },
  green:  { primary: '#10b981', hover: '#059669', muted: 'rgba(16,185,129,0.15)',  glow: '0 0 20px rgba(16,185,129,0.2)',  secondary: '#3b82f6' },
  orange: { primary: '#f59e0b', hover: '#d97706', muted: 'rgba(245,158,11,0.15)',  glow: '0 0 20px rgba(245,158,11,0.2)',  secondary: '#ef4444' },
  pink:   { primary: '#ec4899', hover: '#db2777', muted: 'rgba(236,72,153,0.15)',  glow: '0 0 20px rgba(236,72,153,0.2)',  secondary: '#8b5cf6' },
}

export const useGlobalStore = defineStore('global', {
  state: () => ({
    isLogin: false,
    loading: false,
    menuCollapse: false,
    panelName: 'X-Panel',
    theme: 'light' as 'light' | 'dark',
    version: '',
    currentNodeID: 0,
    currentNodeName: '',
    accentColor: 'cyan' as AccentColor,
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
    setAccentColor(color: AccentColor) {
      this.accentColor = color
      this.applyAccentColor()
    },
    applyAccentColor() {
      const preset = accentPresets[this.accentColor] || accentPresets.cyan
      const root = document.documentElement
      root.style.setProperty('--xp-accent', preset.primary)
      root.style.setProperty('--xp-accent-hover', preset.hover)
      root.style.setProperty('--xp-accent-muted', preset.muted)
      root.style.setProperty('--xp-accent-glow', preset.glow)
      root.style.setProperty('--xp-accent-secondary', preset.secondary)
      // Also update Element Plus primary
      root.style.setProperty('--el-color-primary', preset.primary)
      root.style.setProperty('--el-color-primary-dark-2', preset.hover)
    },
  },
  persist: true,
})
