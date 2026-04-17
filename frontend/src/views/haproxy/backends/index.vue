<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('haproxy.backends') }}</h3>
      <el-button type="primary" @click="openBackendDialog()">
        <el-icon><Plus /></el-icon>{{ $t('haproxy.createBackend') }}
      </el-button>
    </div>

    <el-card shadow="never">
      <el-table :data="tableData" v-loading="loading" stripe @expand-change="onExpandChange">
        <el-table-column type="expand">
          <template #default="{ row }">
            <BackendServers :backend="row" :servers="serversByID[row.id] || []" @refresh="onServersRefresh(row.id)" />
          </template>
        </el-table-column>
        <el-table-column prop="name" :label="$t('haproxy.name')" min-width="140" />
        <el-table-column prop="mode" :label="$t('haproxy.mode')" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.mode === 'http' ? 'primary' : 'success'">{{ row.mode.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="balance" :label="$t('haproxy.balance')" width="130" />
        <el-table-column :label="$t('haproxy.healthType')" width="110">
          <template #default="{ row }"><code>{{ row.healthType }}</code></template>
        </el-table-column>
        <el-table-column prop="serverCount" :label="$t('haproxy.serverCount')" width="90" />
        <el-table-column prop="refCount" :label="$t('haproxy.refCount')" width="80" />
        <el-table-column prop="remark" :label="$t('haproxy.remark')" min-width="140" show-overflow-tooltip />
        <el-table-column :label="$t('commons.actions')" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openBackendDialog(row)">{{ $t('commons.edit') }}</el-button>
            <el-button link type="primary" @click="openAddServer(row)">{{ $t('haproxy.addServer') }}</el-button>
            <el-button link type="danger" @click="handleDeleteBackend(row)">{{ $t('commons.delete') }}</el-button>
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

    <!-- Backend 表单 -->
    <el-dialog v-model="backendDialog" :title="isEditBE ? $t('haproxy.editBackend') : $t('haproxy.createBackend')" width="680px" destroy-on-close>
      <el-form :model="beForm" label-width="130px">
        <el-form-item :label="$t('haproxy.name')">
          <el-input v-model="beForm.name" :disabled="isEditBE" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.mode')">
          <el-radio-group v-model="beForm.mode">
            <el-radio value="http">HTTP</el-radio>
            <el-radio value="tcp">TCP</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('haproxy.balance')">
          <el-select v-model="beForm.balance" style="width: 100%;">
            <el-option value="roundrobin" :label="$t('haproxy.balanceRoundrobin')" />
            <el-option value="leastconn" :label="$t('haproxy.balanceLeastconn')" />
            <el-option value="source" :label="$t('haproxy.balanceSource')" />
            <el-option value="uri" :label="$t('haproxy.balanceURI')" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="beForm.mode === 'http'" :label="$t('haproxy.stickyType')">
          <el-select v-model="beForm.stickyType" style="width: 100%;">
            <el-option value="" :label="$t('haproxy.stickyNone')" />
            <el-option value="cookie" :label="$t('haproxy.stickyCookie')" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="beForm.stickyType === 'cookie'" :label="$t('haproxy.stickyName')">
          <el-input v-model="beForm.stickyName" placeholder="SRVID" />
        </el-form-item>

        <el-divider content-position="left">{{ $t('haproxy.healthType') }}</el-divider>
        <el-form-item :label="$t('haproxy.healthType')">
          <el-select v-model="beForm.healthType" style="width: 100%;">
            <el-option value="tcp" :label="$t('haproxy.healthTCP')" />
            <el-option value="http" :label="$t('haproxy.healthHTTP')" />
            <el-option value="mysql" :label="$t('haproxy.healthMySQL')" />
            <el-option value="pgsql" :label="$t('haproxy.healthPgSQL')" />
            <el-option value="redis" :label="$t('haproxy.healthRedis')" />
            <el-option value="ssl-hello" :label="$t('haproxy.healthSSLHello')" />
            <el-option value="none" :label="$t('haproxy.healthNone')" />
          </el-select>
        </el-form-item>
        <template v-if="beForm.healthType === 'http'">
          <el-form-item :label="$t('haproxy.healthMethod')">
            <el-select v-model="beForm.healthMethod" style="width: 100%;">
              <el-option value="GET" label="GET" />
              <el-option value="HEAD" label="HEAD" />
              <el-option value="POST" label="POST" />
            </el-select>
          </el-form-item>
          <el-form-item :label="$t('haproxy.healthPath')">
            <el-input v-model="beForm.healthPath" placeholder="/health" />
          </el-form-item>
          <el-form-item :label="$t('haproxy.healthHost')">
            <el-input v-model="beForm.healthHost" placeholder="api.example.com" />
          </el-form-item>
          <el-form-item :label="$t('haproxy.healthExpect')">
            <el-input v-model="beForm.healthExpect" :placeholder="$t('haproxy.healthExpectHint')" />
          </el-form-item>
        </template>
        <el-form-item v-if="beForm.healthType !== 'none'" :label="$t('haproxy.healthInter')">
          <el-input-number v-model="beForm.healthInter" :min="500" :step="500" style="width: 100%;" />
        </el-form-item>
        <el-form-item v-if="beForm.healthType !== 'none'" :label="$t('haproxy.healthRise')">
          <el-input-number v-model="beForm.healthRise" :min="1" :max="20" style="width: 100%;" />
        </el-form-item>
        <el-form-item v-if="beForm.healthType !== 'none'" :label="$t('haproxy.healthFall')">
          <el-input-number v-model="beForm.healthFall" :min="1" :max="20" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.remark')">
          <el-input v-model="beForm.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="backendDialog = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="submitBackend" :loading="submitting">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- Server 表单 -->
    <el-dialog v-model="serverDialog" :title="serverForm.id ? $t('haproxy.editServer') : $t('haproxy.addServer')" width="520px" destroy-on-close>
      <el-form :model="serverForm" label-width="120px">
        <el-form-item :label="$t('haproxy.serverName')">
          <el-input v-model="serverForm.name" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.serverAddress')">
          <el-input v-model="serverForm.address" placeholder="192.168.1.1" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.serverPort')">
          <el-input-number v-model="serverForm.port" :min="1" :max="65535" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.serverWeight')">
          <el-input-number v-model="serverForm.weight" :min="0" :max="256" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.serverMaxConn')">
          <el-input-number v-model="serverForm.maxConn" :min="0" style="width: 100%;" />
        </el-form-item>
        <el-form-item :label="$t('haproxy.serverBackup')"><el-switch v-model="serverForm.backup" /></el-form-item>
        <el-form-item :label="$t('haproxy.serverSSL')"><el-switch v-model="serverForm.ssl" /></el-form-item>
        <el-form-item v-if="serverForm.ssl" :label="$t('haproxy.serverSSLVerify')"><el-switch v-model="serverForm.sslVerify" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serverDialog = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="submitServer" :loading="submitting">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  searchHAProxyBackend, createHAProxyBackend, updateHAProxyBackend, deleteHAProxyBackend,
  getHAProxyBackend, createHAProxyServer, updateHAProxyServer, deleteHAProxyServer,
} from '@/api/modules/haproxy'
import BackendServers from '../components/BackendServers.vue'

const { t } = useI18n()
const loading = ref(false)
const tableData = ref<any[]>([])
const pagination = ref({ page: 1, pageSize: 20, total: 0 })

const backendDialog = ref(false)
const isEditBE = ref(false)
const submitting = ref(false)
const defaultBE = () => ({
  id: 0, name: '', mode: 'http', balance: 'roundrobin',
  stickyType: '', stickyName: '',
  healthType: 'tcp', healthPath: '/', healthMethod: 'GET',
  healthHost: '', healthExpect: '',
  healthInter: 2000, healthRise: 2, healthFall: 3,
  remark: '',
})
const beForm = ref<any>(defaultBE())

const serverDialog = ref(false)
const defaultServer = () => ({ id: 0, backendID: 0, name: '', address: '', port: 80, weight: 100, maxConn: 0, backup: false, disabled: false, ssl: false, sslVerify: false })
const serverForm = ref<any>(defaultServer())

const serversByID = reactive<Record<number, any[]>>({})

const search = async () => {
  loading.value = true
  try {
    const res = await searchHAProxyBackend({
      page: pagination.value.page, pageSize: pagination.value.pageSize, info: '', mode: '',
    })
    tableData.value = res.data?.items || []
    pagination.value.total = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const openBackendDialog = (row?: any) => {
  isEditBE.value = !!row
  beForm.value = row ? { ...defaultBE(), ...row } : defaultBE()
  backendDialog.value = true
}

const submitBackend = async () => {
  submitting.value = true
  try {
    if (isEditBE.value) {
      await updateHAProxyBackend(beForm.value)
    } else {
      await createHAProxyBackend(beForm.value)
    }
    ElMessage.success(t('commons.operationSuccess'))
    backendDialog.value = false
    search()
  } finally {
    submitting.value = false
  }
}

const handleDeleteBackend = async (row: any) => {
  await ElMessageBox.confirm(t('haproxy.deleteBackendConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteHAProxyBackend(row.id)
  ElMessage.success(t('commons.deleteSuccess'))
  search()
}

const onExpandChange = async (row: any, expanded: any[]) => {
  const isExpanded = Array.isArray(expanded) ? expanded.includes(row) : !!expanded
  if (isExpanded && !serversByID[row.id]) {
    const res = await getHAProxyBackend(row.id)
    serversByID[row.id] = res.data?.servers || []
  }
}

const onServersRefresh = async (beID: number) => {
  const res = await getHAProxyBackend(beID)
  serversByID[beID] = res.data?.servers || []
  search()
}

const openAddServer = (be: any) => {
  serverForm.value = { ...defaultServer(), backendID: be.id }
  serverDialog.value = true
}

const submitServer = async () => {
  submitting.value = true
  try {
    if (serverForm.value.id) {
      await updateHAProxyServer(serverForm.value)
    } else {
      await createHAProxyServer(serverForm.value)
    }
    ElMessage.success(t('commons.operationSuccess'))
    serverDialog.value = false
    onServersRefresh(serverForm.value.backendID)
  } finally {
    submitting.value = false
  }
}

defineExpose({ openEditServer: (s: any) => { serverForm.value = { ...s }; serverDialog.value = true } })

onMounted(() => search())
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
</style>
