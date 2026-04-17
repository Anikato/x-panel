<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('haproxy.configHistory') }}</h3>
      <el-button @click="load"><el-icon><Refresh /></el-icon>{{ $t('commons.refresh') }}</el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="list" v-loading="loading" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="operator" :label="$t('haproxy.operator')" width="120" />
        <el-table-column :label="$t('haproxy.success')" width="90">
          <template #default="{ row }">
            <el-tag size="small" :type="row.success ? 'success' : 'danger'">
              {{ row.success ? $t('commons.success') : $t('commons.failed') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" :label="$t('haproxy.message')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="createdAt" :label="$t('commons.createdAt')" width="180">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="$t('commons.actions')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="view(row)">{{ $t('commons.view') }}</el-button>
            <el-button link type="warning" @click="doRollback(row)" :disabled="!row.success">{{ $t('haproxy.rollback') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="total-info" v-if="total">{{ $t('haproxy.totalItems', { count: total }) }}</div>
    </el-card>

    <el-dialog v-model="viewVisible" :title="$t('haproxy.configSnapshot')" width="900px">
      <el-input v-model="viewContent" type="textarea" :rows="24" readonly class="code-view" spellcheck="false" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { listHAProxyConfigVersions, getHAProxyConfigVersion, rollbackHAProxyConfig } from '@/api/modules/haproxy'

const { t } = useI18n()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)

const viewVisible = ref(false)
const viewContent = ref('')

const load = async () => {
  loading.value = true
  try {
    const res = await listHAProxyConfigVersions()
    list.value = res.data || []
    total.value = list.value.length
  } finally {
    loading.value = false
  }
}

const view = async (row: any) => {
  const res = await getHAProxyConfigVersion(row.id)
  viewContent.value = res.data?.content || ''
  viewVisible.value = true
}

const doRollback = async (row: any) => {
  await ElMessageBox.confirm(t('haproxy.rollbackConfirm'), t('commons.warning'), { type: 'warning' })
  await rollbackHAProxyConfig(row.id)
  ElMessage.success(t('commons.operationSuccess'))
  load()
}

const formatTime = (s: string) => {
  if (!s) return '-'
  try { return new Date(s).toLocaleString() } catch { return s }
}

onMounted(() => load())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.total-info { margin-top: 16px; text-align: right; color: #909399; font-size: 13px; }
.code-view :deep(.el-textarea__inner) {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace; font-size: 13px;
}
</style>
