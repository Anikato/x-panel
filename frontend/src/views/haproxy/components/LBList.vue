<template>
  <div>
    <div class="page-header">
      <h3>{{ mode === 'http' ? $t('haproxy.httpLB') : $t('haproxy.tcpLB') }}</h3>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>{{ $t('haproxy.createLB') }}
      </el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="name" :label="$t('haproxy.name')" min-width="140" />
        <el-table-column :label="$t('haproxy.bindAddr')" min-width="160">
          <template #default="{ row }">
            <code>{{ row.bindAddr }}:{{ row.bindPort }}</code>
          </template>
        </el-table-column>
        <el-table-column v-if="mode === 'http'" :label="$t('haproxy.enableSSL')" width="90">
          <template #default="{ row }">
            <el-tag v-if="row.enableSSL" type="success" size="small">SSL</el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column v-if="mode === 'http'" :label="$t('haproxy.certDomain')" min-width="140">
          <template #default="{ row }">
            <span v-if="row.certDomain">{{ row.certDomain }}</span>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('haproxy.defaultBackend')" min-width="140">
          <template #default="{ row }">
            <el-tag v-if="row.defaultBackend" size="small">{{ row.defaultBackend }}</el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="remark" :label="$t('haproxy.remark')" min-width="140" show-overflow-tooltip />
        <el-table-column :label="$t('commons.status')" width="90">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleToggle(row)" size="small" />
          </template>
        </el-table-column>
        <el-table-column :label="$t('commons.actions')" width="210" fixed="right">
          <template #default="{ row }">
            <el-button v-if="mode === 'http'" link type="primary" @click="openACL(row)">ACL</el-button>
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

    <!-- 编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('haproxy.editLB') : $t('haproxy.createLB')" width="640px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item :label="$t('haproxy.name')" prop="name">
          <el-input v-model="form.name" :disabled="isEdit" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.bindAddr')">
          <el-input v-model="form.bindAddr" placeholder="0.0.0.0" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.bindPort')" prop="bindPort">
          <el-input-number v-model="form.bindPort" :min="1" :max="65535" style="width: 100%;" />
        </el-form-item>
        <template v-if="mode === 'http'">
          <el-form-item :label="$t('haproxy.enableSSL')">
            <el-switch v-model="form.enableSSL" />
          </el-form-item>
          <el-form-item v-if="form.enableSSL" :label="$t('haproxy.selectCertificate')" prop="certificateID">
            <el-select v-model="form.certificateID" filterable style="width: 100%;">
              <el-option v-for="c in certList" :key="c.id" :label="c.primaryDomain || c.domains" :value="c.id" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.enableSSL" :label="$t('haproxy.sslRedirect')">
            <el-switch v-model="form.sslRedirect" />
          </el-form-item>
          <el-form-item :label="$t('haproxy.xForwardedFor')">
            <el-switch v-model="form.xForwardedFor" />
          </el-form-item>
        </template>
        <el-form-item :label="$t('haproxy.defaultBackend')">
          <el-select v-model="form.defaultBackendID" clearable style="width: 100%;">
            <el-option v-for="b in filteredBackends" :key="b.id" :label="b.name" :value="b.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('haproxy.maxConn')">
          <el-input-number v-model="form.maxConn" :min="0" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.timeoutClient')">
          <el-input-number v-model="form.timeoutClient" :min="1" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.remark')">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- ACL 对话框（仅 HTTP） -->
    <ACLDialog v-if="aclLB" v-model="aclVisible" :lb="aclLB" :backends="backendList" @closed="aclLB = null" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  searchHAProxyLB, createHAProxyLB, updateHAProxyLB, deleteHAProxyLB, toggleHAProxyLB,
  listHAProxyBackend, listCertificatesForHAProxy,
} from '@/api/modules/haproxy'
import ACLDialog from './ACLDialog.vue'

const props = defineProps<{ mode: 'http' | 'tcp' }>()
const { t } = useI18n()

const loading = ref(false)
const tableData = ref<any[]>([])
const pagination = ref({ page: 1, pageSize: 20, total: 0 })
const dialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const backendList = ref<any[]>([])
const certList = ref<any[]>([])
const aclVisible = ref(false)
const aclLB = ref<any>(null)

const defaultForm = () => ({
  id: 0, name: '', mode: props.mode,
  bindAddr: '0.0.0.0', bindPort: props.mode === 'http' ? 80 : 3306,
  enableSSL: false, certificateID: 0, sslRedirect: false,
  defaultBackendID: 0, xForwardedFor: true,
  maxConn: 2000, timeoutConnect: 5, timeoutClient: 30, timeoutServer: 30,
  remark: '',
})
const form = ref<any>(defaultForm())

const rules = {
  name: [{ required: true, message: t('haproxy.name'), trigger: 'blur' }],
  bindPort: [{ required: true, message: t('haproxy.bindPort'), trigger: 'blur' }],
  certificateID: [{ required: true, message: t('haproxy.selectCertificate'), trigger: 'change' }],
}

const filteredBackends = computed(() => backendList.value.filter((b) => b.mode === props.mode))

const search = async () => {
  loading.value = true
  try {
    const res = await searchHAProxyLB({
      page: pagination.value.page, pageSize: pagination.value.pageSize,
      info: '', mode: props.mode,
    })
    tableData.value = res.data?.items || []
    pagination.value.total = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const loadBackends = async () => {
  const res = await listHAProxyBackend()
  backendList.value = res.data || []
}
const loadCerts = async () => {
  const res = await listCertificatesForHAProxy()
  certList.value = res.data?.items || []
}

const openDialog = (row?: any) => {
  isEdit.value = !!row
  form.value = row ? { ...defaultForm(), ...row, mode: props.mode } : defaultForm()
  dialogVisible.value = true
}

const handleSubmit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    const payload = { ...form.value, mode: props.mode }
    if (isEdit.value) {
      await updateHAProxyLB(payload)
    } else {
      await createHAProxyLB(payload)
    }
    ElMessage.success(t('commons.operationSuccess'))
    dialogVisible.value = false
    search()
  } finally {
    submitting.value = false
  }
}

const handleToggle = async (row: any) => {
  try {
    await toggleHAProxyLB({ id: row.id, enabled: row.enabled })
    ElMessage.success(t('commons.operationSuccess'))
  } catch {
    row.enabled = !row.enabled
  }
}

const handleDelete = async (row: any) => {
  await ElMessageBox.confirm(t('haproxy.deleteLBConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteHAProxyLB(row.id)
  ElMessage.success(t('commons.deleteSuccess'))
  search()
}

const openACL = (row: any) => {
  aclLB.value = row
  aclVisible.value = true
}

onMounted(async () => {
  await Promise.all([loadBackends(), loadCerts()])
  search()
})
</script>

<style lang="scss" scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.table-footer {
  margin-top: 16px; display: flex; justify-content: flex-end;
}
.text-muted { color: var(--xp-text-muted); }
</style>
