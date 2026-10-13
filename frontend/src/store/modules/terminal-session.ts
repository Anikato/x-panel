import { defineStore } from 'pinia'
import {
  closeTerminalSession,
  listTerminalSessions,
  setTerminalSessionLock,
  type BackgroundTerminal,
} from '@/api/modules/terminal'

export type { BackgroundTerminal }

// A slower list response must not overwrite a newer refresh or a terminate.
let listSeq = 0

export const useTerminalSessionStore = defineStore('terminalSession', {
  state: () => ({
    items: [] as BackgroundTerminal[],
    surface: 'float' as 'float' | 'page',
  }),
  getters: {
    runningCount: (state) => state.items.filter((item) => item.status === 'running').length,
  },
  actions: {
    async refresh() {
      const seq = ++listSeq
      const res = await listTerminalSessions()
      if (seq !== listSeq) return
      this.items = res.data || []
    },
    async terminate(id: string) {
      await closeTerminalSession(id)
      listSeq++
      this.items = this.items.filter((item) => item.id !== id)
    },
    async setLocked(id: string, locked: boolean, name: string) {
      await setTerminalSessionLock(id, locked, name)
      if (!locked) {
        listSeq++
        this.items = this.items.filter((item) => item.id !== id)
      }
    },
  },
})
