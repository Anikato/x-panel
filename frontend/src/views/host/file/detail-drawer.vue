<template>
  <el-drawer v-model="visible" :title="t('file.fileInfo')" size="380px" direction="rtl">
    <el-descriptions :column="1" border v-if="info">
      <el-descriptions-item :label="t('file.fileName')">
        {{ info.name }}
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.fileType')">
        <span v-if="info.isDir">{{ t('file.directory') }}</span>
        <span v-else-if="info.isSymlink">{{ t('file.symlink') }}</span>
        <span v-else>{{ info.extension ? `.${info.extension}` : t('file.regularFile') }}</span>
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.filePath')">
        <span style="word-break: break-all;">{{ info.path }}</span>
      </el-descriptions-item>
      <el-descriptions-item v-if="info.isSymlink" :label="t('file.linkTarget')">
        {{ info.linkPath || '-' }}
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.fileSize')">
        <template v-if="info.isDir">
          <el-button v-if="!dirSizeLoaded" type="primary" link :loading="dirSizeLoading" @click="calcDirSize">
            {{ t('file.calculate') }}
          </el-button>
          <span v-else>{{ formatSize(dirSize) }}</span>
        </template>
        <span v-else>{{ formatSize(info.size) }}</span>
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.mode')">
        {{ info.mode }} ({{ info.modeNum }})
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.user')">
        {{ info.user }} ({{ t('file.fileUid') }}: {{ info.uid }})
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.group')">
        {{ info.group }} ({{ t('file.fileGid') }}: {{ info.gid }})
      </el-descriptions-item>
      <el-descriptions-item :label="t('file.modTime')">
        {{ formatTime(info.modTime) }}
      </el-descriptions-item>
    </el-descriptions>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getDirSize } from '@/api/modules/file'

const { t } = useI18n()
const visible = ref(false)
const info = ref<any>(null)
const dirSize = ref(0)
const dirSizeLoaded = ref(false)
const dirSizeLoading = ref(false)

const formatSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

const formatTime = (iso: string): string => {
  if (!iso) return '-'
  return new Date(iso).toLocaleString('zh-CN', { hour12: false })
}

const calcDirSize = async () => {
  if (!info.value) return
  dirSizeLoading.value = true
  try {
    const res: any = await getDirSize({ path: info.value.path })
    dirSize.value = res.data?.size || 0
    dirSizeLoaded.value = true
  } catch { /* */ } finally {
    dirSizeLoading.value = false
  }
}

const open = (row: any) => {
  info.value = row
  dirSize.value = 0
  dirSizeLoaded.value = false
  visible.value = true
}

defineExpose({ open })
</script>
