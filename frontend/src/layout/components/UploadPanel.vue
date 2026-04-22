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
          <!-- 文件名 -->
          <div class="upload-item-name" :title="item.name">{{ item.name }}</div>
          <!-- 进度条 -->
          <el-progress
            :percentage="item.progress"
            :status="item.error ? 'exception' : item.progress >= 100 ? 'success' : undefined"
            :stroke-width="3"
            :show-text="false"
            class="upload-item-bar"
          />
          <!-- 状态行 -->
          <div class="upload-item-meta">
            <template v-if="item.error">
              <span class="meta-error">上传失败</span>
            </template>
            <template v-else-if="item.progress >= 100">
              <span class="meta-done">✓ {{ formatBytes(item.bytesTotal) }}</span>
            </template>
            <template v-else>
              <span class="meta-bytes">{{ formatBytes(item.bytesDone) }} / {{ formatBytes(item.bytesTotal) }}</span>
              <span v-if="item.speed > 0" class="meta-speed">{{ formatBytes(item.speed) }}/s</span>
              <span v-if="item.speed > 0 && item.bytesTotal > item.bytesDone" class="meta-eta">
                · 约{{ formatEta(item.bytesTotal - item.bytesDone, item.speed) }}
              </span>
            </template>
          </div>
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
  return `${Math.floor(mins / 60)} 小时 ${mins % 60} 分`
}
</script>

<style scoped>
.global-upload-panel {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 380px;
  background: var(--xp-bg-card);
  border: 1px solid var(--xp-border);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.18);
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
  margin-left: 4px;
  transition: color 0.2s;
}
.upload-panel-close:hover { color: var(--xp-text-primary); }

.upload-panel-body {
  max-height: 240px;
  overflow-y: auto;
  padding: 8px 0;
}

.upload-item {
  padding: 6px 14px;
}

.upload-item-name {
  font-size: 12px;
  color: var(--xp-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 4px;
}

.upload-item-bar {
  margin-bottom: 3px;
}

.upload-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--xp-text-muted);
}

.meta-bytes { color: var(--xp-text-muted); }
.meta-speed { color: var(--xp-accent); font-weight: 500; }
.meta-eta   { color: var(--xp-text-muted); }
.meta-done  { color: var(--el-color-success); }
.meta-error { color: var(--el-color-danger); }

.slide-up-enter-active, .slide-up-leave-active {
  transition: all 0.3s ease;
}
.slide-up-enter-from, .slide-up-leave-to {
  transform: translateY(20px);
  opacity: 0;
}
</style>
