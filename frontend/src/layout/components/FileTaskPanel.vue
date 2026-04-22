<template>
  <Transition name="slide-up">
    <div v-if="store.tasks.length > 0" class="global-task-panel" :class="{ collapsed: isCollapsed }">
      <!-- 标题栏 -->
      <div class="task-panel-hd" @click="isCollapsed = !isCollapsed">
        <div class="hd-left">
          <el-icon class="spin-icon" v-if="store.runningCount > 0"><Loading /></el-icon>
          <el-icon v-else><CircleCheck /></el-icon>
          <span class="hd-title">
            {{ store.runningCount > 0 ? `${store.runningCount} 个任务执行中` : '所有任务已完成' }}
          </span>
        </div>
        <div class="hd-right">
          <el-button v-if="!isCollapsed && store.finishedTasks.length > 0" link size="small" @click.stop="store.clearFinished()">
            清除
          </el-button>
          <el-icon :class="{ 'rotate-180': !isCollapsed }"><ArrowDown /></el-icon>
        </div>
      </div>

      <!-- 任务列表 -->
      <Transition name="slide">
        <div v-show="!isCollapsed" class="task-panel-body">
          <div
            v-for="task in store.tasks"
            :key="task.id"
            class="task-item"
            :class="'task-' + task.status"
          >
            <div class="task-icon">
              <el-icon v-if="task.status === 'running'" class="spin-icon"><Loading /></el-icon>
              <el-icon v-else-if="task.status === 'success'" style="color:var(--el-color-success)"><CircleCheck /></el-icon>
              <el-icon v-else style="color:var(--el-color-danger)"><CircleClose /></el-icon>
            </div>
            <div class="task-info">
              <div class="task-name">{{ task.name }}</div>

              <!-- 运行中且有进度信息 -->
              <template v-if="task.status === 'running' && task.bytesTotal > 0">
                <el-progress :percentage="task.progress" :stroke-width="3" :show-text="false" class="task-bar" />
                <div class="task-stats">
                  <span>{{ formatBytes(task.bytesDone) }} / {{ formatBytes(task.bytesTotal) }}</span>
                  <span v-if="task.speed > 0" class="stat-speed">· {{ formatBytes(task.speed) }}/s</span>
                  <span v-if="task.speed > 0 && task.bytesTotal > task.bytesDone" class="stat-eta">
                    · 约{{ formatEta(task.bytesTotal - task.bytesDone, task.speed) }}
                  </span>
                </div>
                <div v-if="task.currentFile" class="task-file">{{ task.currentFile }}</div>
              </template>

              <!-- 运行中无进度（下载/压缩等） -->
              <template v-else-if="task.status === 'running'">
                <span class="task-meta">{{ formatDuration(task.startTime) }}</span>
              </template>

              <!-- 失败 -->
              <template v-else-if="task.status === 'failed'">
                <span class="task-error">{{ task.message }}</span>
              </template>

              <!-- 成功 -->
              <template v-else>
                <span class="task-meta">{{ formatEndDuration(task.startTime, task.endTime) }}</span>
              </template>
            </div>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Loading, CircleCheck, CircleClose, ArrowDown } from '@element-plus/icons-vue'
import { useFileTaskStore } from '@/store/modules/fileTask'

const store = useFileTaskStore()
const isCollapsed = ref(false)

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 ** 3) return `${(bytes / 1024 ** 2).toFixed(1)} MB`
  return `${(bytes / 1024 ** 3).toFixed(2)} GB`
}

function formatEta(rem: number, speed: number): string {
  const s = Math.ceil(rem / speed)
  if (s < 60) return `${s} 秒`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} 分 ${s % 60} 秒`
  return `${Math.floor(m / 60)} 小时 ${m % 60} 分`
}

function formatDuration(startTime: number): string {
  const s = Math.floor(Date.now() / 1000 - startTime)
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m ${s % 60}s`
}

function formatEndDuration(startTime: number, endTime?: number): string {
  if (!endTime) return ''
  const s = endTime - startTime
  if (s < 1) return '< 1s'
  if (s < 60) return `${s}s`
  return `${Math.floor(s / 60)}m ${s % 60}s`
}
</script>

<style scoped>
.global-task-panel {
  position: fixed;
  bottom: 24px;
  left: 24px;
  width: 400px;
  max-height: 420px;
  background: var(--xp-bg-card);
  border: 1px solid var(--xp-border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
  z-index: 2048;
  overflow: hidden;
  transition: max-height 0.3s ease;
}

.global-task-panel.collapsed {
  max-height: 44px;
}

.task-panel-hd {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  cursor: pointer;
  background: var(--xp-bg-inset);
  border-bottom: 1px solid var(--xp-border);
  user-select: none;
}

.hd-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.hd-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--xp-text-primary);
}

.hd-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.rotate-180 {
  transform: rotate(180deg);
  transition: transform 0.3s;
}

.task-panel-body {
  max-height: 370px;
  overflow-y: auto;
  padding: 4px 0;
}

.task-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 9px 14px;
  transition: background 0.2s;
}

.task-item:hover {
  background: var(--xp-bg-hover);
}

.task-icon {
  flex-shrink: 0;
  margin-top: 2px;
  font-size: 15px;
}

.task-info {
  flex: 1;
  min-width: 0;
}

.task-name {
  font-size: 13px;
  color: var(--xp-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 3px;
}

.task-bar { margin: 3px 0; }

.task-stats {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--xp-text-muted);
  flex-wrap: wrap;
}

.stat-speed { color: var(--xp-accent); font-weight: 500; }

.task-file {
  font-size: 11px;
  color: var(--xp-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-top: 2px;
}

.task-meta {
  font-size: 12px;
  color: var(--xp-text-muted);
}

.task-error {
  font-size: 12px;
  color: var(--el-color-danger);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.slide-enter-active, .slide-leave-active {
  transition: max-height 0.3s ease, opacity 0.2s ease;
}
.slide-enter-from, .slide-leave-to { max-height: 0; opacity: 0; }

.slide-up-enter-active, .slide-up-leave-active { transition: all 0.3s ease; }
.slide-up-enter-from, .slide-up-leave-to { transform: translateY(20px); opacity: 0; }
</style>
