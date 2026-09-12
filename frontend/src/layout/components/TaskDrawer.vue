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
            <span v-else>{{ item.progress }}%</span>
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
          <div class="task-meta">
            <span v-if="task.status === 'failed'" class="is-error">{{ task.message }}</span>
            <span v-else-if="task.status === 'running'">{{ t('header.taskRunning') }}</span>
            <span v-else>{{ t('header.taskDone') }}</span>
          </div>
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

.task-name {
  overflow: hidden;
  color: var(--xp-text-primary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.is-error {
  color: var(--xp-danger);
}
</style>
