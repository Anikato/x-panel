<template>
  <div>
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="card-header-title">
            <el-icon><Notebook /></el-icon>
            <span>{{ t('log.operationLog') }}</span>
          </div>
          <el-button type="danger" plain size="small" @click="handleClean">
            <el-icon><Delete /></el-icon>{{ t('log.clean') }}
          </el-button>
        </div>
      </template>

      <el-table :data="tableData" v-loading="loading">
        <el-table-column prop="ip" :label="t('log.ip')" width="150" />
        <el-table-column prop="path" :label="t('log.path')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="method" :label="t('log.method')" width="100">
          <template #default="{ row }">
            <el-tag :type="methodType(row.method)" size="small">{{ row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="t('log.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="latency" :label="t('log.latency')" width="120" />
        <el-table-column prop="createdAt" :label="t('log.time')" width="180" />
      </el-table>

      <div class="table-footer">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
          @size-change="fetchData"
          @current-change="fetchData"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { getOperationLogs, cleanOperationLogs } from '@/api/modules/log'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const tableData = ref<any[]>([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })

const methodType = (m: string): string => ({ GET: 'info', POST: 'success', PUT: 'warning', DELETE: 'danger' }[m] || 'info')

const fetchData = async () => {
  loading.value = true
  try {
    const res: any = await getOperationLogs({ page: pagination.page, pageSize: pagination.pageSize })
    tableData.value = res.data?.items || []
    pagination.total = res.data?.total || 0
  } catch { /* */ } finally { loading.value = false }
}

const handleClean = async () => {
  try {
    await ElMessageBox.confirm(t('log.cleanConfirm'), t('commons.tip'), { type: 'warning' })
    await cleanOperationLogs()
    ElMessage.success(t('commons.success'))
    fetchData()
  } catch { /* cancelled */ }
}

onMounted(() => fetchData())
</script>

<style lang="scss" scoped>
.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
