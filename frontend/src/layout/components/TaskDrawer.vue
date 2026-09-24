<template>
  <el-drawer
    v-model="visible"
    :title="t('header.tasks')"
    size="420px"
    append-to-body
    :destroy-on-close="false"
  >
    <div class="task-drawer">
      <section>
        <div class="task-section-title">
          <span>{{ t('file.upload') }}</span>
          <span class="task-count">{{ uploadStore.doneCount }}/{{ uploadStore.queue.length }}</span>
        </div>
        <div v-if="uploadStore.queue.length === 0" class="task-empty">{{ t('header.noUploadTasks') }}</div>
        <div v-for="item in uploadStore.queue" :key="item.id" class="task-row xp-inset">
          <div class="task-name" :title="item.name">{{ item.name }}</div>
          <el-progress
            :percentage="item.progress"
            :status="item.error ? 'exception' : item.progress >= 100 ? 'success' : undefined"
            :stroke-width="4"
            :show-text="false"
          />
          <div class="task-meta">
            <span v-if="item.error" class="is-error">{{ item.errorMessage || t('file.uploadFailed') }}</span>
            <template v-else>
              <span>{{ item.progress }}%</span>
              <span v-if="item.bytesTotal > 0"> · {{ formatBytes(item.bytesDone) }} / {{ formatBytes(item.bytesTotal) }}</span>
              <span v-if="item.progress < 100 && item.speed > 0"> · {{ formatBytes(item.speed) }}/s</span>
              <span v-if="item.progress < 100 && item.speed > 0 && item.bytesTotal > item.bytesDone">
                · 约{{ formatEta(item.bytesTotal - item.bytesDone, item.speed) }}
              </span>
            </template>
          </div>
        </div>
      </section>

      <section>
        <div class="task-section-title">
          <span>{{ t('header.fileTasks') }}</span>
          <el-button v-if="fileTaskStore.finishedTasks.length" link type="primary" @click="fileTaskStore.clearFinished()">
            {{ t('commons.delete') }}
          </el-button>
        </div>
        <div v-if="fileTaskStore.tasks.length === 0" class="task-empty">{{ t('header.noFileTasks') }}</div>
        <div v-for="task in fileTaskStore.tasks" :key="task.id" class="task-row xp-inset">
          <div class="task-name">{{ task.name }}</div>
          <el-progress
            v-if="task.status === 'running' && task.bytesTotal > 0"
            :percentage="task.progress"
            :stroke-width="4"
            :show-text="false"
          />
          <el-progress
            v-else-if="task.status === 'running'"
            :percentage="100"
            :indeterminate="true"
            :stroke-width="4"
            :show-text="false"
          />
          <div class="task-meta">
            <span v-if="task.status === 'failed'" class="is-error">{{ task.message }}</span>
            <span v-else-if="task.status === 'cancelled'">已取消</span>
            <template v-else-if="task.status === 'running' && task.bytesTotal > 0">
              <span>{{ task.progress }}% · {{ formatBytes(task.bytesDone) }} / {{ formatBytes(task.bytesTotal) }}</span>
              <span v-if="task.speed > 0"> · {{ formatBytes(task.speed) }}/s</span>
              <span v-if="task.speed > 0 && task.bytesTotal > task.bytesDone">
                · 约{{ formatEta(task.bytesTotal - task.bytesDone, task.speed) }}
              </span>
            </template>
            <template v-else-if="task.status === 'running'">
              <span>处理中</span>
              <span v-if="task.bytesDone > 0"> · {{ formatBytes(task.bytesDone) }}</span>
              <span v-if="task.speed > 0"> · {{ formatBytes(task.speed) }}/s</span>
            </template>
            <span v-else>{{ t('header.taskDone') }}</span>
          </div>
          <div v-if="task.status === 'running' && task.currentFile" class="task-current">{{ task.currentFile }}</div>
        </div>
      </section>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useUploadStore } from '@/store/modules/upload'
import { useFileTaskStore } from '@/store/modules/fileTask'

const { t } = useI18n()
const uploadStore = useUploadStore()
const fileTaskStore = useFileTaskStore()

const visible = defineModel<boolean>({ default: false })

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B'
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
  if (mins < 60) return `${mins} 分 ${secs % 60} 秒`
  return `${Math.floor(mins / 60)} 小时 ${mins % 60} 分`
}

const activeCount = computed(() => {
  const uploads = uploadStore.queue.filter((item) => !item.error && item.progress < 100).length
  return uploads + fileTaskStore.runningCount
})

defineExpose({ activeCount })
</script>

<style scoped lang="scss">
.task-drawer {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.task-section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  color: var(--xp-text-primary);
  font-size: 13px;
  font-weight: 650;
}

.task-count,
.task-meta {
  color: var(--xp-text-muted);
  font-size: 12px;
}

.task-empty {
  padding: 16px 0;
  color: var(--xp-text-muted);
  font-size: 13px;
}

.task-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 12px;
  padding: 10px 12px;
  border-radius: var(--xp-radius-sm);
}

.task-name,
.task-current {
  overflow: hidden;
  color: var(--xp-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-current {
  color: var(--xp-text-muted);
  font-size: 12px;
}

.is-error {
  color: var(--xp-danger);
}
</style>
