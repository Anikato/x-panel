<template>
  <!-- 压缩弹窗 -->
  <el-dialog v-model="compressVisible" :title="t('file.compress')" width="500px" destroy-on-close>
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
import { compressFile, decompressFile } from '@/api/modules/file'

const { t } = useI18n()
const emit = defineEmits(['done'])
const loading = ref(false)

// 压缩
const compressVisible = ref(false)
const compressForm = ref({ paths: [] as string[], dst: '', name: '', type: 'tar.gz' })

const openCompress = (paths: string[], currentDir: string) => {
  compressForm.value = {
    paths,
    dst: currentDir,
    name: paths.length === 1 ? (paths[0].split('/').pop() || 'archive') : 'archive',
    type: 'tar.gz',
  }
  compressVisible.value = true
}

const doCompress = async () => {
  const f = compressForm.value
  if (!f.name) return
  loading.value = true
  try {
    await compressFile({ paths: f.paths, dst: f.dst, name: f.name + '.' + f.type, type: f.type })
    ElMessage.success(t('commons.success'))
    compressVisible.value = false
    emit('done')
  } catch { /* */ } finally {
    loading.value = false
  }
}

// 解压
const decompressVisible = ref(false)
const decompressForm = ref({ path: '', dst: '' })

const openDecompress = (path: string, currentDir: string) => {
  decompressForm.value = { path, dst: currentDir }
  decompressVisible.value = true
}

const doDecompress = async () => {
  loading.value = true
  try {
    await decompressFile(decompressForm.value)
    ElMessage.success(t('commons.success'))
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
</style>
