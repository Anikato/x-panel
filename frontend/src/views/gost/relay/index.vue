<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('gost.relayTitle') }}</h3>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>{{ $t('gost.createRelay') }}
      </el-button>
    </div>

    <el-alert type="info" :closable="false" style="margin-bottom: 16px;" show-icon>
      {{ $t('gost.relayDesc') }}
    </el-alert>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe>
        <el-table-column prop="name" :label="$t('gost.name')" min-width="120" />
        <el-table-column prop="listenAddr" :label="$t('gost.listenAddr')" min-width="130">
          <template #default="{ row }">
            <code>{{ row.listenAddr }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="listenerType" :label="$t('gost.listenerType')" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.listenerType?.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="$t('gost.tlsCert')" min-width="140">
          <template #default="{ row }">
            <template v-if="row.listenerType === 'tls' || row.listenerType === 'wss'">
              <el-tag v-if="row.certDomain" size="small" type="success">{{ row.certDomain }}</el-tag>
              <el-tag v-else-if="row.customCertPath" size="small" type="warning">{{ $t('gost.certCustomPath') }}</el-tag>
              <el-tag v-else size="small" type="info">{{ $t('gost.certNone') }}</el-tag>
            </template>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="authUser" :label="$t('gost.authUser')" width="120">
          <template #default="{ row }">
            {{ row.authUser || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="enabled" :label="$t('commons.status')" width="100">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleToggle(row)" size="small" />
          </template>
        </el-table-column>
        <el-table-column :label="$t('commons.actions')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showConnInfo(row)">{{ $t('gost.connInfo') }}</el-button>
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

    <!-- 创建 / 编辑 -->
    <el-dialog v-model="dialogVisible" :title="isEdit ? $t('gost.editRelay') : $t('gost.createRelay')" width="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item :label="$t('gost.name')" prop="name">
          <el-input v-model="form.name" />
        </el-form-item>
        <el-form-item :label="$t('gost.listenAddr')" prop="listenAddr">
          <el-input v-model="form.listenAddr" :placeholder="$t('gost.listenAddrHint')" />
        </el-form-item>
        <el-form-item :label="$t('gost.listenerType')" prop="listenerType">
          <el-select v-model="form.listenerType" style="width: 100%" @change="onListenerTypeChange">
            <el-option label="TCP" value="tcp" />
            <el-option label="TLS" value="tls" />
            <el-option label="WebSocket (WS)" value="ws" />
            <el-option label="WebSocket + TLS (WSS)" value="wss" />
          </el-select>
        </el-form-item>
        <template v-if="showCertFields">
          <el-form-item :label="$t('gost.certSource')">
            <el-radio-group v-model="certMode">
              <el-radio-button value="panel">{{ $t('gost.certFromPanel') }}</el-radio-button>
              <el-radio-button value="custom">{{ $t('gost.certCustomPath') }}</el-radio-button>
              <el-radio-button value="none">{{ $t('gost.certNone') }}</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="certMode === 'panel'" :label="$t('gost.selectCert')">
            <el-select v-model="form.certificateID" style="width: 100%" :placeholder="$t('gost.selectCertHint')" clearable>
              <el-option
                v-for="cert in certList"
                :key="cert.id"
                :label="cert.primaryDomain + (cert.status === 'applied' ? '' : ' (' + cert.status + ')')"
                :value="cert.id"
              />
            </el-select>
          </el-form-item>
          <template v-if="certMode === 'custom'">
            <el-form-item :label="$t('gost.certFilePath')">
              <el-input v-model="form.customCertPath" placeholder="/path/to/fullchain.pem" />
            </el-form-item>
            <el-form-item :label="$t('gost.keyFilePath')">
              <el-input v-model="form.customKeyPath" placeholder="/path/to/privkey.pem" />
            </el-form-item>
          </template>
        </template>
        <el-form-item :label="$t('gost.authUser')">
          <el-input v-model="form.authUser" />
        </el-form-item>
        <el-form-item :label="$t('gost.authPass')">
          <el-input v-model="form.authPass" type="password" show-password :placeholder="isEdit ? $t('gost.authPassHint') : ''" />
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

    <!-- 连接信息 -->
    <el-dialog v-model="connDialogVisible" :title="$t('gost.connInfo')" width="600px">
      <el-descriptions :column="1" border size="small">
        <el-descriptions-item :label="$t('gost.connCommand')">
          <div class="conn-cmd">
            <code>{{ connCommand }}</code>
            <el-button link type="primary" @click="copyCommand" size="small" style="margin-left: 8px;">
              {{ $t('commons.copy') }}
            </el-button>
          </div>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { searchGostService, createGostService, updateGostService, deleteGostService, toggleGostService } from '@/api/modules/gost'
import { searchCertificate } from '@/api/modules/ssl'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const tableData = ref<any[]>([])
const pagination = reactive({ page: 1, pageSize: 20, total: 0 })
const dialogVisible = ref(false)
const connDialogVisible = ref(false)
const isEdit = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const connCommand = ref('')
const certList = ref<any[]>([])
const certMode = ref<'panel' | 'custom' | 'none'>('panel')

const showCertFields = computed(() => form.listenerType === 'tls' || form.listenerType === 'wss')

const form = reactive({
  id: 0,
  name: '',
  type: 'relay_server',
  listenAddr: '',
  listenerType: 'wss',
  authUser: '',
  authPass: '',
  certificateID: 0 as number,
  customCertPath: '',
  customKeyPath: '',
  enableStats: true,
  remark: '',
})

const rules = reactive<FormRules>({
  name: [{ required: true, message: t('gost.nameRequired'), trigger: 'blur' }],
  listenAddr: [{ required: true, message: t('gost.listenAddrRequired'), trigger: 'blur' }],
  listenerType: [{ required: true, trigger: 'change' }],
})

const search = async () => {
  loading.value = true
  try {
    const res = await searchGostService({
      page: pagination.page,
      pageSize: pagination.pageSize,
      type: 'relay_server',
    })
    if (res.data) {
      tableData.value = res.data.items || []
      pagination.total = res.data.total || 0
    }
  } finally {
    loading.value = false
  }
}

const loadCertList = async () => {
  try {
    const res = await searchCertificate({ page: 1, pageSize: 100 })
    if (res.data?.items) {
      certList.value = res.data.items
    }
  } catch { /* ignore */ }
}

const onListenerTypeChange = () => {
  if (showCertFields.value && certList.value.length === 0) {
    loadCertList()
  }
}

const openDialog = (row?: any) => {
  isEdit.value = !!row
  if (row) {
    Object.assign(form, {
      id: row.id,
      name: row.name,
      type: 'relay_server',
      listenAddr: row.listenAddr,
      listenerType: row.listenerType || 'wss',
      authUser: row.authUser || '',
      authPass: '',
      certificateID: row.certificateID || 0,
      customCertPath: row.customCertPath || '',
      customKeyPath: row.customKeyPath || '',
      enableStats: row.enableStats,
      remark: row.remark || '',
    })
    if (row.customCertPath) {
      certMode.value = 'custom'
    } else if (row.certificateID) {
      certMode.value = 'panel'
    } else {
      certMode.value = 'none'
    }
  } else {
    Object.assign(form, {
      id: 0, name: '', type: 'relay_server', listenAddr: ':443',
      listenerType: 'wss', authUser: '', authPass: '', certificateID: 0,
      customCertPath: '', customKeyPath: '', enableStats: true, remark: '',
    })
    certMode.value = 'panel'
  }
  if (showCertFields.value) {
    loadCertList()
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  await formRef.value?.validate()
  submitLoading.value = true
  try {
    const payload = { ...form, targetAddr: '' }
    if (certMode.value === 'panel') {
      payload.customCertPath = ''
      payload.customKeyPath = ''
    } else if (certMode.value === 'custom') {
      payload.certificateID = 0
    } else {
      payload.certificateID = 0
      payload.customCertPath = ''
      payload.customKeyPath = ''
    }
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

const showConnInfo = (row: any) => {
  const port = row.listenAddr?.replace(':', '') || '443'
  const proto = row.listenerType || 'wss'
  const auth = row.authUser ? `${row.authUser}:PASSWORD@` : ''
  connCommand.value = `gost -L :8080 -F "relay+${proto}://${auth}YOUR_SERVER_IP:${port}"`
  connDialogVisible.value = true
}

const copyCommand = () => {
  navigator.clipboard.writeText(connCommand.value).then(() => {
    ElMessage.success(t('commons.copySuccess'))
  }).catch(() => {
    ElMessage.error(t('commons.copyFailed'))
  })
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
.table-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.conn-cmd {
  display: flex;
  align-items: center;
  code {
    background: var(--el-fill-color-light);
    padding: 8px 12px;
    border-radius: 4px;
    font-size: 13px;
    word-break: break-all;
    flex: 1;
  }
}
code {
  background: var(--el-fill-color-light);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
</style>
