<template>
  <!-- 压缩弹窗 -->
  <el-dialog v-model="compressVisible" :title="t('file.compress')" width="640px" destroy-on-close>
    <el-form label-width="100px">
      <el-form-item :label="t('file.compressName')">
        <el-input v-model="compressForm.name" placeholder="archive" />
      </el-form-item>
      <el-form-item :label="t('file.compressType')">
        <el-select v-model="compressForm.type" style="width: 100%">
          <el-option label="tar.gz" value="tar.gz" />
          <el-option label="zip" value="zip" />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('file.targetPath')">
        <el-input v-model="compressForm.dst" />
      </el-form-item>
      <el-form-item :label="t('file.name')">
        <div class="compress-files">
          <el-tag v-for="p in compressForm.paths" :key="p" size="small" class="file-tag">
            {{ p.split('/').pop() }}
          </el-tag>
        </div>
      </el-form-item>
      <el-form-item :label="t('file.excludeRules')">
        <div class="exclude-panel">
          <el-checkbox-group v-model="compressForm.excludePresets" class="exclude-options">
            <el-checkbox label="cache">{{ t('file.excludeCache') }}</el-checkbox>
            <el-checkbox label="logs">{{ t('file.excludeLogs') }}</el-checkbox>
            <el-checkbox label="temp">{{ t('file.excludeTemp') }}</el-checkbox>
            <el-checkbox label="underscore">{{ t('file.excludeUnderscore') }}</el-checkbox>
            <el-checkbox label="hidden">{{ t('file.excludeHidden') }}</el-checkbox>
          </el-checkbox-group>
          <el-input
            v-model="compressForm.customExcludes"
            type="textarea"
            :rows="4"
            :placeholder="t('file.excludePlaceholder')"
          />
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="compressVisible = false">{{ t('commons.cancel') }}</el-button>
      <el-button type="primary" :loading="loading" @click="doCompress">{{ t('commons.confirm') }}</el-button>
    </template>
  </el-dialog>

  <!-- 解压弹窗 -->
  <el-dialog v-model="decompressVisible" :title="t('file.decompress')" width="500px" destroy-on-close>
    <el-form label-width="100px">
      <el-form-item :label="t('file.name')">
        <el-input :model-value="decompressForm.path.split('/').pop()" disabled />
      </el-form-item>
      <el-form-item :label="t('file.decompressTo')">
        <el-input v-model="decompressForm.dst" />
      </el-form-item>
      <el-form-item :label="t('file.decompressOptions')">
        <div class="decompress-options">
          <el-checkbox v-model="decompressForm.extractToSameDir">
            {{ t('file.extractToSameDir') }}
          </el-checkbox>
          <el-radio-group v-model="decompressForm.conflictPolicy">
            <el-radio-button label="overwrite">{{ t('file.decompressConflictOverwrite') }}</el-radio-button>
            <el-radio-button label="skip">{{ t('file.decompressConflictSkip') }}</el-radio-button>
            <el-radio-button label="rename">{{ t('file.decompressConflictRename') }}</el-radio-button>
          </el-radio-group>
        </div>
      </el-form-item>
      <el-form-item :label="t('file.archivePreview')">
        <div class="archive-preview">
          <el-button size="small" :loading="previewLoading" @click="loadArchivePreview">
            {{ t('file.previewArchiveContent') }}
          </el-button>
          <el-alert
            v-if="archivePreview.unsafeEntries.length"
            :title="t('file.unsafeArchivePaths')"
            type="error"
            show-icon
            :closable="false"
          />
          <div v-if="archivePreview.entries.length" class="archive-preview-list">
            <div class="archive-preview-count">
              {{ t('file.archivePreviewCount', { shown: archivePreview.entries.length, total: archivePreview.total }) }}
            </div>
            <div v-for="entry in archivePreview.entries" :key="entry" class="archive-preview-item">
              {{ entry }}
            </div>
          </div>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="decompressVisible = false">{{ t('commons.cancel') }}</el-button>
      <el-button type="primary" :loading="loading" @click="doDecompress">{{ t('commons.confirm') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { compressFile, decompressFile, listArchive } from '@/api/modules/file'

const { t } = useI18n()
const emit = defineEmits(['done'])
const loading = ref(false)

const excludePresetMap: Record<string, string[]> = {
  cache: ['cache*', '*/cache*', '.cache', '*/.cache'],
  logs: ['logs', 'logs/*', '*/logs', '*/logs/*', '*.log'],
  temp: ['tmp*', '*/tmp*', 'temp*', '*/temp*', '*.tmp'],
  underscore: ['_*', '*/_*'],
  hidden: ['.*', '*/.*'],
}

// 压缩
const compressVisible = ref(false)
const compressForm = ref({
  paths: [] as string[],
  dst: '',
  name: '',
  type: 'tar.gz',
  excludePresets: ['cache', 'logs', 'temp'] as string[],
  customExcludes: '',
})

const openCompress = (paths: string[], currentDir: string) => {
  compressForm.value = {
    paths,
    dst: currentDir,
    name: paths.length === 1 ? (paths[0].split('/').pop() || 'archive') : 'archive',
    type: 'tar.gz',
    excludePresets: ['cache', 'logs', 'temp'],
    customExcludes: '',
  }
  compressVisible.value = true
}

const buildExcludeRules = () => {
  const presetRules = compressForm.value.excludePresets.flatMap((key) => excludePresetMap[key] || [])
  const customRules = compressForm.value.customExcludes
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean)
  return Array.from(new Set([...presetRules, ...customRules]))
}

const doCompress = async () => {
  const f = compressForm.value
  if (!f.name) return
  loading.value = true
  try {
    const res: any = await compressFile({
      paths: f.paths,
      dst: f.dst,
      name: f.name + '.' + f.type,
      type: f.type,
      excludes: buildExcludeRules(),
    })
    if (res.data?.taskID) {
      ElMessage.info(t('file.taskStarted'))
    } else {
      ElMessage.success(t('commons.success'))
    }
    compressVisible.value = false
    emit('done')
  } catch { /* */ } finally {
    loading.value = false
  }
}

// 解压
const decompressVisible = ref(false)
const decompressForm = ref({ path: '', dst: '', extractToSameDir: true, conflictPolicy: 'overwrite' })
const previewLoading = ref(false)
const archivePreview = ref({ entries: [] as string[], total: 0, unsafeEntries: [] as string[] })

const openDecompress = (path: string, currentDir: string) => {
  decompressForm.value = { path, dst: currentDir, extractToSameDir: true, conflictPolicy: 'overwrite' }
  archivePreview.value = { entries: [], total: 0, unsafeEntries: [] }
  decompressVisible.value = true
}

const loadArchivePreview = async () => {
  previewLoading.value = true
  try {
    const res: any = await listArchive({ path: decompressForm.value.path })
    archivePreview.value = {
      entries: res.data?.entries || [],
      total: res.data?.total || 0,
      unsafeEntries: res.data?.unsafeEntries || [],
    }
  } catch { /* */ } finally {
    previewLoading.value = false
  }
}

const doDecompress = async () => {
  loading.value = true
  try {
    const res: any = await decompressFile(decompressForm.value)
    if (res.data?.taskID) {
      ElMessage.info(t('file.taskStarted'))
    } else {
      ElMessage.success(t('commons.success'))
    }
    decompressVisible.value = false
    emit('done')
  } catch { /* */ } finally {
    loading.value = false
  }
}

defineExpose({ openCompress, openDecompress })
</script>

<style scoped>
.compress-files {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.file-tag {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.exclude-panel {
  width: 100%;
}
.exclude-options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 2px 12px;
  margin-bottom: 10px;
}
.decompress-options {
  display: grid;
  gap: 10px;
  width: 100%;
}
.archive-preview {
  display: grid;
  gap: 8px;
  width: 100%;
}
.archive-preview-list {
  max-height: 180px;
  overflow: auto;
  border: 1px solid var(--el-border-color);
  border-radius: 6px;
  padding: 8px;
}
.archive-preview-count {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-bottom: 6px;
}
.archive-preview-item {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
