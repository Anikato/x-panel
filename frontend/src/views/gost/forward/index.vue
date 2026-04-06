<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('gost.forwardTitle') }}</h3>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>{{ $t('gost.createForward') }}
      </el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="name" :label="$t('gost.name')" min-width="120" />
        <el-table-column prop="type" :label="$t('gost.type')" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.type === 'tcp_udp_forward'" type="warning" size="small">TCP+UDP</el-tag>
            <el-tag v-else :type="row.type === 'tcp_forward' ? 'primary' : 'success'" size="small">
              {{ row.type === 'tcp_forward' ? 'TCP' : 'UDP' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="listenAddr" :label="$t('gost.listenAddr')" min-width="130">
          <template #default="{ row }">
            <code>{{ row.listenAddr }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="targetAddr" :label="$t('gost.targetAddr')" min-width="160">
          <template #default="{ row }">
            <code>{{ row.targetAddr }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="chainName" :label="$t('gost.chain')" width="140">
          <template #default="{ row }">
            <el-tag v-if="row.chainName" size="small" type="warning">{{ row.chainName }}</el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('gost.traffic')" min-width="160">
          <template #default="{ row }">
            <template v-if="row.enableStats && row.enabled">
              <span class="traffic-text">↑ {{ formatBytes(row.inputBytes) }}</span>
              <span class="traffic-text" style="margin-left: 8px;">↓ {{ formatBytes(row.outputBytes) }}</span>
            </template>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" :label="$t('commons.status')" width="100">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleToggle(row)" size="small" />
          </template>
        </el-table-column>
        <el-table-column :label="$t('commons.actions')" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDialog(row)">{{ $t('commons.edit') }}</el-button>
            <el-button link type="danger" @click="handleDelete(row)">{{ $t('commons.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer" v-if="pagination.total > pagination.pageSize">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="pagination.total"
          layout="total, prev, pager, next"
          @current-change="search"
        />
      </div>
    </el-card>

    <!-- 创建 / 编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('gost.editForward') : $t('gost.createForward')" width="560px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="$t('gost.name')" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="$t('gost.type')" prop="type">
          <el-radio-group v-model="form.type">
            <el-radio value="tcp_forward">TCP</el-radio>
            <el-radio value="udp_forward">UDP</el-radio>
            <el-radio value="tcp_udp_forward">TCP+UDP</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('gost.listenAddr')" prop="listenAddr">
          <el-input v-model="form.listenAddr" :placeholder="$t('gost.listenAddrHint')" />
        </el-form-item>
        <el-form-item :label="$t('gost.targetAddr')" prop="targetAddr">
          <el-input v-model="form.targetAddr" :placeholder="$t('gost.targetAddrHint')" />
        </el-form-item>
        <el-form-item :label="$t('gost.chainRef')">
          <el-select v-model="form.chainID" clearable style="width: 100%">
            <el-option :label="$t('gost.chainRefNone')" :value="0" />
            <el-option v-for="c in chainOptions" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('gost.enableStats')">
          <el-switch v-model="form.enableStats" />
        </el-form-item>
        <el-form-item :label="$t('gost.remark')">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitLoading">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { searchGostService, createGostService, updateGostService, deleteGostService, toggleGostService, searchGostChain } from '@/api/modules/gost'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const tableData = ref<any[]>([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const dialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const chainOptions = ref<any[]>([])

const form = reactive({
  id: 0,
  name: '',
  type: 'tcp_forward',
  listenAddr: '',
  targetAddr: '',
  listenerType: 'tcp',
  chainID: 0,
  enableStats: true,
  remark: '',
})

const rules = reactive<FormRules>({
  name: [{ required: true, message: t('gost.nameRequired'), trigger: 'blur' }],
  type: [{ required: true, trigger: 'change' }],
  listenAddr: [{ required: true, message: t('gost.listenAddrRequired'), trigger: 'blur' }],
  targetAddr: [{ required: true, message: t('gost.targetAddrRequired'), trigger: 'blur' }],
})

const search = async () => {
  loading.value = true
  try {
    const res = await searchGostService({
      page: pagination.page,
      pageSize: pagination.pageSize,
      type: 'tcp_forward,udp_forward,tcp_udp_forward',
    })
    if (res.data) {
      tableData.value = res.data.items || []
      pagination.total = res.data.total || 0
    }
  } finally {
    loading.value = false
  }
}

const loadChains = async () => {
  try {
    const res = await searchGostChain({ page: 1, pageSize: 100 })
    if (res.data) {
      chainOptions.value = res.data.items || []
    }
  } catch { /* ignore */ }
}

const openDialog = (row?: any) => {
  isEdit.value = !!row
  if (row) {
    Object.assign(form, {
      id: row.id,
      name: row.name,
      type: row.type,
      listenAddr: row.listenAddr,
      targetAddr: row.targetAddr,
      listenerType: row.listenerType || 'tcp',
      chainID: row.chainID || 0,
      enableStats: row.enableStats,
      remark: row.remark || '',
    })
  } else {
    Object.assign(form, {
      id: 0, name: '', type: 'tcp_forward', listenAddr: '', targetAddr: '',
      listenerType: 'tcp', chainID: 0, enableStats: true, remark: '',
    })
  }
  loadChains()
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  submitLoading.value = true
  try {
    const payload = { ...form }
    if (isEdit.value) {
      await updateGostService(payload)
    } else {
      await createGostService(payload)
    }
    ElMessage.success(t('commons.operationSuccess'))
    dialogVisible.value = false
    await search()
  } finally {
    submitLoading.value = false
  }
}

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm(t('gost.deleteConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteGostService(row.id)
  ElMessage.success(t('commons.deleteSuccess'))
  await search()
}

const handleToggle = async (row: any) => {
  try {
    await toggleGostService(row.id, row.enabled)
    ElMessage.success(t('commons.operationSuccess'))
  } catch {
    row.enabled = !row.enabled
  }
}

const formatBytes = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i]
}

onMounted(() => search())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.text-muted { color: var(--xp-text-muted); }
.traffic-text { font-size: 12px; font-family: monospace; }
.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
code {
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
</style>
