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
            <el-icon v-else-if="task.status === 'cancelled'" color="var(--el-color-warning)"><CircleClose /></el-icon>
            <el-icon v-else color="var(--el-color-danger)"><CircleClose /></el-icon>
          </div>
          <div class="task-info">
            <div class="task-name">{{ task.name || taskTypeLabel(task.type) }}</div>

            <!-- 运行中：显示进度条 + 速度 + ETA -->
            <template v-if="task.status === 'running'">
              <template v-if="task.bytesTotal > 0">
                <el-progress
                  :percentage="task.progress"
                  :stroke-width="4"
                  :show-text="false"
                  class="task-progress"
                />
                <div class="task-stats">
                  <span>{{ formatBytes(task.bytesDone) }} / {{ formatBytes(task.bytesTotal) }}</span>
                  <span v-if="task.speed > 0">· {{ formatBytes(task.speed) }}/s</span>
                  <span v-if="task.speed > 0 && task.bytesTotal > task.bytesDone">
                    · 约{{ formatEta(task.bytesTotal - task.bytesDone, task.speed) }}
                  </span>
                </div>
                <div v-if="task.currentFile" class="task-current-file">{{ task.currentFile }}</div>
              </template>
              <template v-else>
                <div class="task-stats">
                  <span v-if="task.bytesDone > 0">已下载 {{ formatBytes(task.bytesDone) }}</span>
                  <span v-if="task.speed > 0">· {{ formatBytes(task.speed) }}/s</span>
                  <span>用时 {{ formatDuration(task.startTime) }}</span>
                </div>
                <div v-if="task.currentFile" class="task-current-file">{{ task.currentFile }}</div>
              </template>
            </template>

            <!-- 失败 -->
            <template v-else-if="task.status === 'failed'">
              <span class="task-error">{{ task.message }}</span>
            </template>

            <!-- 已取消 -->
            <template v-else-if="task.status === 'cancelled'">
              <span class="task-time">已取消</span>
            </template>

            <!-- 成功 -->
            <template v-else>
              <span class="task-time">{{ formatEndDuration(task.startTime, task.endTime) }}</span>
            </template>
          </div>
          <div v-if="task.status === 'running' && task.type === 'download'" class="task-actions">
            <el-button link size="small" :loading="cancellingIds.has(task.id)" @click.stop="cancelTask(task)">
              取消
            </el-button>
          </div>
        </div>
        <div v-if="tasks.length === 0" class="task-empty">{{ t('file.taskEmpty') }}</div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Loading, CircleCheck, CircleClose, ArrowDown } from '@element-plus/icons-vue'
import { cancelFileTask, listFileTasks } from '@/api/modules/file'

const { t } = useI18n()

interface FileTask {
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

const tasks = ref<FileTask[]>([])
const isCollapsed = ref(false)
const cancellingIds = ref(new Set<string>())
let pollTimer: ReturnType<typeof setInterval> | null = null

const runningCount = computed(() => tasks.value.filter(t => t.status === 'running').length)
const finishedTasks = computed(() => tasks.value.filter(t => t.status !== 'running'))

function taskTypeLabel(type: string): string {
  const map: Record<string, string> = {
    move: '文件操作',
    compress: '压缩',
    decompress: '解压',
    download: '远程下载',
  }
  return map[type] || type
}

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatEta(remainingBytes: number, speed: number): string {
  if (speed <= 0) return '...'
  const secs = Math.ceil(remainingBytes / speed)
  if (secs < 60) return `${secs} 秒`
  const mins = Math.floor(secs / 60)
  const s = secs % 60
  if (mins < 60) return `${mins} 分 ${s} 秒`
  const hours = Math.floor(mins / 60)
  return `${hours} 小时 ${mins % 60} 分`
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

async function cancelTask(task: FileTask) {
  if (cancellingIds.value.has(task.id)) return
  cancellingIds.value = new Set([...cancellingIds.value, task.id])
  try {
    await cancelFileTask(task.id)
    ElMessage.success('已发送取消请求')
    await fetchTasks()
  } catch {
    // 全局请求拦截器会处理错误提示
  } finally {
    const next = new Set(cancellingIds.value)
    next.delete(task.id)
    cancellingIds.value = next
  }
}

function refresh() {
  fetchTasks()
}

// 动态调整轮询频率：运行中 1s，空闲 30s
function startPolling() {
  if (pollTimer) clearInterval(pollTimer)
  const interval = runningCount.value > 0 ? 1000 : 30000
  pollTimer = setInterval(() => {
    fetchTasks()
  }, interval)
}

watch(runningCount, () => {
  startPolling()
})

onMounted(() => {
  fetchTasks()
  startPolling()
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
  width: 400px;
  max-height: 420px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.15);
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
  max-height: 360px;
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

.task-actions {
  flex-shrink: 0;
  margin-top: -2px;
}

.task-name {
  font-size: 13px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.task-progress {
  margin: 4px 0;
}

.task-stats {
  display: flex;
  gap: 6px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
  flex-wrap: wrap;
}

.task-current-file {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}

.task-time {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.task-error {
  font-size: 12px;
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
