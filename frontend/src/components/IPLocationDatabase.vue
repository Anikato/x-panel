<template>
  <el-button size="small" @click="open">
    {{ $t(info?.loaded === false ? 'ipDatabase.setup' : 'ipDatabase.title') }}
  </el-button>
  <el-dialog v-model="visible" :title="$t('ipDatabase.title')" width="480px" destroy-on-close>
    <div v-loading="loading" class="xp-form-medium">
      <el-descriptions v-if="info" :column="1" border size="small">
        <el-descriptions-item :label="$t('ipDatabase.status')">
          <el-tag :type="info.loaded ? 'success' : 'info'" size="small">
            {{ $t(info.loaded ? 'ipDatabase.loaded' : 'ipDatabase.notLoaded') }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item v-if="info.size" :label="$t('ipDatabase.size')">
          {{ (info.size / 1024 / 1024).toFixed(1) }} MB
        </el-descriptions-item>
        <el-descriptions-item v-if="info.updatedAt" :label="$t('ipDatabase.updated')">
          {{ info.updatedAt }}
        </el-descriptions-item>
      </el-descriptions>
      <p>{{ $t('ipDatabase.hint') }}</p>
    </div>
    <template #footer>
      <el-button @click="visible = false">{{ $t('commons.close') }}</el-button>
      <el-button type="primary" :loading="downloading" :disabled="loading" @click="download">
        {{ $t(info?.loaded ? 'ipDatabase.update' : 'ipDatabase.setup') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { downloadIPDB, getIPDBInfo } from '@/api/modules/toolbox'

const emit = defineEmits<{ updated: [] }>()
const { t } = useI18n()
const info = ref<{ loaded: boolean; size: number; updatedAt: string }>()
const visible = ref(false)
const loading = ref(false)
const downloading = ref(false)

async function loadInfo() {
  if (loading.value) return
  loading.value = true
  try {
    info.value = (await getIPDBInfo()).data
  } catch {
    info.value = undefined
  } finally {
    loading.value = false
  }
}

function open() {
  visible.value = true
  void loadInfo()
}

async function download() {
  downloading.value = true
  try {
    await downloadIPDB()
    ElMessage.success(t('commons.success'))
    await loadInfo()
    emit('updated')
  } catch { /* The HTTP client displays the download error. */ }
  finally { downloading.value = false }
}

onMounted(loadInfo)
</script>
