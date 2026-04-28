import { defineStore } from 'pinia'
import { removeToken, setToken } from '@/utils/auth'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    name: '',
  }),
  actions: {
    setToken(token: string, remember = false) {
      this.token = token
      setToken(token, remember)
    },
    setName(name: string) {
      this.name = name
    },
    logout() {
      this.token = ''
      this.name = ''
      removeToken()
    },
  },
  persist: {
    paths: ['name'],
  },
})
