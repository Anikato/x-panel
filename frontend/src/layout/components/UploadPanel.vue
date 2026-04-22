<template>
  <Transition name="slide-up">
    <div v-if="uploadStore.queue.length > 0" class="global-upload-panel">
      <div class="upload-panel-hd">
        <span>{{ t('file.upload') }}</span>
        <span class="upload-panel-count">{{ uploadStore.doneCount }}/{{ uploadStore.queue.length }}</span>
        <el-icon v-if="uploadStore.allDone" class="upload-panel-close" @click="uploadStore.clear()"><Close /></el-icon>
      </div>
      <div class="upload-panel-body">
        <div v-for="item in uploadStore.queue" :key="item.id" class="upload-item">
          <span class="upload-item-name" :title="item.name">{{ item.name }}</span>
          <el-progress
            :percentage="item.progress"
            :status="item.error ? 'exception' : item.progress >= 100 ? 'success' : undefined"
            :stroke-width="4"
            :show-text="false"
            style="flex:1;margin:0 8px"
          />
          <span class="upload-item-status">
            <template v-if="item.error">{{ t('file.failed') }}</template>
            <template v-else-if="item.progress >= 100">{{ t('file.done') }}</template>
            <template v-else>{{ item.progress }}%</template>
          </span>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Close } from '@element-plus/icons-vue'
import { useUploadStore } from '@/store/modules/upload'

const { t } = useI18n()
const uploadStore = useUploadStore()
</script>

<style scoped>
.global-upload-panel {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 360px;
  background: var(--xp-bg-card);
  border: 1px solid var(--xp-border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.15);
  z-index: 2048;
  overflow: hidden;
}
.upload-panel-hd {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  font-size: 13px;
  font-weight: 600;
  color: var(--xp-text-primary);
  background: var(--xp-bg-inset);
  border-bottom: 1px solid var(--xp-border);
}
.upload-panel-count {
  flex: 1;
  text-align: right;
  font-weight: 400;
  color: var(--xp-text-muted);
  font-size: 12px;
}
.upload-panel-close {
  cursor: pointer;
  color: var(--xp-text-muted);
  margin-left: 8px;
}
.upload-panel-close:hover {
  color: var(--xp-text-primary);
}
.upload-panel-body {
  max-height: 200px;
  overflow-y: auto;
  padding: 8px 14px;
}
.upload-item {
  display: flex;
  align-items: center;
  padding: 4px 0;
  gap: 4px;
  font-size: 12px;
}
.upload-item-name {
  width: 100px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--xp-text-secondary);
}
.upload-item-status {
  width: 40px;
  text-align: right;
  font-size: 11px;
  color: var(--xp-text-muted);
}

.slide-up-enter-active, .slide-up-leave-active {
  transition: all 0.3s ease;
}
.slide-up-enter-from, .slide-up-leave-to {
  transform: translateY(20px);
  opacity: 0;
}
</style>
