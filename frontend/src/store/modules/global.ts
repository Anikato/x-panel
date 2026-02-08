import { defineStore } from 'pinia'

export const useGlobalStore = defineStore('global', {
  state: () => ({
    isLogin: false,
    loading: false,
    menuCollapse: false,
    panelName: 'X-Panel',
    theme: 'light' as 'light' | 'dark',
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
  },
  persist: true,
})
