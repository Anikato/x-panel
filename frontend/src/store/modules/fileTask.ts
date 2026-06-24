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
  let initialized = false

  // 已被用户清除的已完成任务 ID 集合，防止轮询时被后端数据重新覆盖回来
  const clearedIds = new Set<string>()

  const runningCount = computed(() => tasks.value.filter(t => t.status === 'running').length)
  const finishedTasks = computed(() => tasks.value.filter(t => t.status !== 'running'))

  async function fetchTasks() {
    try {
      if (!apiModule) {
        apiModule = await import('@/api/modules/file')
      }
      const res: any = await apiModule.listFileTasks()
      const all: FileTask[] = res.data || []

      // 如果某个已清除的任务重新变成 running（说明是新任务），从清除集合中移除
      for (const task of all) {
        if (task.status === 'running' && clearedIds.has(task.id)) {
          clearedIds.delete(task.id)
        }
      }

      // 过滤掉已被用户清除的已完成任务
      tasks.value = all.filter(t => !(clearedIds.has(t.id) && t.status !== 'running'))
    } catch { /* ignore */ }
  }

  function clearFinished() {
    // 记录当前所有已完成任务的 ID，防止下次轮询再拉回来
    for (const t of tasks.value) {
      if (t.status !== 'running') {
        clearedIds.add(t.id)
      }
    }
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
    initialized = false
  }

  // 初始化：全局调用一次
  function init() {
    if (initialized) return
    initialized = true
    fetchTasks()
    startPolling()
  }

  return { tasks, runningCount, finishedTasks, fetchTasks, clearFinished, init, stopPolling }
})
