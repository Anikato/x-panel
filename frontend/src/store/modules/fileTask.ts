import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface FileTask {
  id: string
  name: string
  type: string
  status: string
  message?: string
  startTime: number
  endTime?: number
  progress: number
  bytesDone: number
  bytesTotal: number
  speed: number
  currentFile: string
}

export const useFileTaskStore = defineStore('fileTask', () => {
  const tasks = ref<FileTask[]>([])
  let pollTimer: ReturnType<typeof setInterval> | null = null
  let apiModule: any = null

  const runningCount = computed(() => tasks.value.filter(t => t.status === 'running').length)
  const finishedTasks = computed(() => tasks.value.filter(t => t.status !== 'running'))

  async function fetchTasks() {
    try {
      if (!apiModule) {
        apiModule = await import('@/api/modules/file')
      }
      const res: any = await apiModule.listFileTasks()
      tasks.value = res.data || []
    } catch { /* ignore */ }
  }

  function clearFinished() {
    tasks.value = tasks.value.filter(t => t.status === 'running')
  }

  function startPolling() {
    if (pollTimer) clearInterval(pollTimer)
    // 运行中 1s，空闲 30s
    const interval = runningCount.value > 0 ? 1000 : 30000
    pollTimer = setInterval(async () => {
      await fetchTasks()
      // 频率自适应：任务状态变化时重新设定间隔
      startPolling()
    }, interval)
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  // 初始化：全局调用一次
  function init() {
    fetchTasks()
    startPolling()
  }

  return { tasks, runningCount, finishedTasks, fetchTasks, clearFinished, init, stopPolling }
})
