<template>
  <div v-if="tasks.length > 0" class="task-panel" :class="{ collapsed: isCollapsed }">
    <!-- 标题栏 -->
    <div class="task-panel-header" @click="isCollapsed = !isCollapsed">
      <div class="header-left">
        <el-icon class="spin-icon" v-if="runningCount > 0"><Loading /></el-icon>
        <el-icon v-else><CircleCheck /></el-icon>
        <span class="header-title">
          {{ runningCount > 0
            ? t('file.taskRunning', { count: runningCount })
            : t('file.taskAllDone')
          }}
        </span>
      </div>
      <div class="header-right">
        <el-button v-if="!isCollapsed && finishedTasks.length > 0" link size="small" @click.stop="clearFinished">
          {{ t('file.taskClear') }}
        </el-button>
        <el-icon :class="{ 'chevron-up': !isCollapsed }"><ArrowDown /></el-icon>
      </div>
    </div>

    <!-- 任务列表 -->
    <transition name="slide">
      <div v-show="!isCollapsed" class="task-panel-body">
        <div
          v-for="task in tasks"
          :key="task.id"
          class="task-item"
          :class="'task-' + task.status"
        >
          <div class="task-icon">
            <el-icon v-if="task.status === 'running'" class="spin-icon"><Loading /></el-icon>
            <el-icon v-else-if="task.status === 'success'" color="var(--el-color-success)"><CircleCheck /></el-icon>
            <el-icon v-else color="var(--el-color-danger)"><CircleClose /></el-icon>
          </div>
          <div class="task-info">
            <div class="task-name">{{ task.name || taskTypeLabel(task.type) }}</div>
            <div class="task-meta">
              <span v-if="task.status === 'running'" class="task-time">{{ formatDuration(task.startTime) }}</span>
              <span v-else-if="task.status === 'failed'" class="task-error">{{ task.message }}</span>
              <span v-else class="task-time">{{ formatEndDuration(task.startTime, task.endTime) }}</span>
            </div>
          </div>
        </div>
        <div v-if="tasks.length === 0" class="task-empty">{{ t('file.taskEmpty') }}</div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loading, CircleCheck, CircleClose, ArrowDown } from '@element-plus/icons-vue'
import { listFileTasks } from '@/api/modules/file'

const { t } = useI18n()

interface FileTask {
  id: string
  name: string
  type: string
  status: string
  message?: string
  startTime: number
  endTime?: number
}

const tasks = ref<FileTask[]>([])
const isCollapsed = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null

const runningCount = computed(() => tasks.value.filter(t => t.status === 'running').length)
const finishedTasks = computed(() => tasks.value.filter(t => t.status !== 'running'))

function taskTypeLabel(type: string): string {
  const map: Record<string, string> = {
    move: '文件操作',
    compress: '压缩',
    decompress: '解压',
  }
  return map[type] || type
}

function formatDuration(startTime: number): string {
  const seconds = Math.floor(Date.now() / 1000 - startTime)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (minutes < 60) return `${minutes}m ${secs}s`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

function formatEndDuration(startTime: number, endTime?: number): string {
  if (!endTime) return ''
  const seconds = endTime - startTime
  if (seconds < 1) return '< 1s'
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ${seconds % 60}s`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

async function fetchTasks() {
  try {
    const res: any = await listFileTasks()
    tasks.value = res.data || []
  } catch { /* ignore */ }
}

function clearFinished() {
  tasks.value = tasks.value.filter(t => t.status === 'running')
}

// 暴露给父组件：手动触发刷新
function refresh() {
  fetchTasks()
}

onMounted(() => {
  fetchTasks()
  // 有运行中任务时 2 秒轮询，否则 10 秒
  pollTimer = setInterval(() => {
    fetchTasks()
  }, runningCount.value > 0 ? 2000 : 10000)
})

onBeforeUnmount(() => {
  if (pollTimer) clearInterval(pollTimer)
})

defineExpose({ refresh })
</script>

<style scoped>
.task-panel {
  position: fixed;
  bottom: 16px;
  right: 16px;
  width: 380px;
  max-height: 400px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.12);
  z-index: 2000;
  overflow: hidden;
  transition: all 0.3s ease;
}

.task-panel.collapsed {
  max-height: 44px;
}

.task-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  cursor: pointer;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-extra-light);
  user-select: none;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-right .el-icon {
  transition: transform 0.3s;
}

.chevron-up {
  transform: rotate(180deg);
}

.task-panel-body {
  max-height: 340px;
  overflow-y: auto;
  padding: 4px 0;
}

.task-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  transition: background 0.2s;
}

.task-item:hover {
  background: var(--el-fill-color-lighter);
}

.task-icon {
  flex-shrink: 0;
  margin-top: 2px;
  font-size: 16px;
}

.task-info {
  flex: 1;
  min-width: 0;
}

.task-name {
  font-size: 13px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}

.task-error {
  color: var(--el-color-danger);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-empty {
  text-align: center;
  padding: 20px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

/* slide transition */
.slide-enter-active,
.slide-leave-active {
  transition: max-height 0.3s ease, opacity 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  max-height: 0;
  opacity: 0;
}
</style>
