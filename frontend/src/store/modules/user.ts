import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    name: '',
  }),
  actions: {
    setToken(token: string) {
      this.token = token
      sessionStorage.setItem('token', token)
    },
    setName(name: string) {
      this.name = name
    },
    logout() {
      this.token = ''
      this.name = ''
      sessionStorage.removeItem('token')
    },
  },
  persist: {
    paths: ['name'],
  },
})
