<template>
  <div>
    <div class="app-toolbar">
      <el-button type="primary" @click="openCreateServer">{{ t('database.addServer') }}</el-button>
      <div style="flex:1" />
      <el-select v-model="searchType" clearable :placeholder="t('database.type')" style="width:140px;margin-right:10px" @change="loadServers">
        <el-option label="MySQL" value="mysql" />
        <el-option label="PostgreSQL" value="postgresql" />
      </el-select>
    </div>

    <el-table :data="servers" v-loading="loading" style="width:100%">
      <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
      <el-table-column prop="type" :label="t('database.type')" width="120" />
      <el-table-column prop="address" :label="t('database.address')" width="160" />
      <el-table-column prop="port" :label="t('database.port')" width="80" />
      <el-table-column :label="t('commons.actions')" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openInstances(row)">{{ t('database.databases') }}</el-button>
          <el-button link type="primary" @click="testConn(row)">{{ t('database.testConn') }}</el-button>
          <el-button link type="primary" @click="openEditServer(row)">{{ t('commons.edit') }}</el-button>
          <el-button link type="danger" @click="handleDeleteServer(row)">{{ t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="app-pagination">
      <el-pagination v-model:current-page="pager.page" v-model:page-size="pager.pageSize" :total="pager.total" layout="total, sizes, prev, pager, next" :page-sizes="[20,50]" @size-change="loadServers" @current-change="loadServers" />
    </div>

    <!-- Server Create/Edit -->
    <el-drawer v-model="serverDrawer" :title="editServerMode ? t('commons.edit') : t('database.addServer')" size="480px" destroy-on-close>
      <el-form ref="serverFormRef" :model="serverForm" :rules="serverRules" label-width="100px">
        <el-form-item :label="t('commons.name')" prop="name"><el-input v-model="serverForm.name" /></el-form-item>
        <el-form-item :label="t('database.type')" prop="type">
          <el-select v-model="serverForm.type" :disabled="editServerMode" style="width:100%">
            <el-option label="MySQL" value="mysql" /><el-option label="PostgreSQL" value="postgresql" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('database.from')" prop="from">
          <el-radio-group v-model="serverForm.from">
            <el-radio value="local">{{ t('database.local') }}</el-radio>
            <el-radio value="remote">{{ t('database.remote') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('database.address')"><el-input v-model="serverForm.address" placeholder="127.0.0.1" /></el-form-item>
        <el-form-item :label="t('database.port')"><el-input-number v-model="serverForm.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item :label="t('database.username')"><el-input v-model="serverForm.username" /></el-form-item>
        <el-form-item :label="t('database.password')"><el-input v-model="serverForm.password" type="password" show-password /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serverDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitServer">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Instances drawer -->
    <el-drawer v-model="instanceDrawer" :title="currentServerName + ' - ' + t('database.databases')" size="640px" destroy-on-close>
      <div style="margin-bottom:12px;display:flex;gap:10px">
        <el-button type="primary" size="small" @click="openCreateInstance">{{ t('database.createDB') }}</el-button>
        <el-button size="small" @click="syncInstances">{{ t('database.sync') }}</el-button>
      </div>
      <el-table :data="instances" v-loading="instanceLoading">
        <el-table-column prop="name" :label="t('commons.name')" min-width="160" />
        <el-table-column prop="charset" label="Charset" width="120" />
        <el-table-column :label="t('commons.actions')" width="120">
          <template #default="{ row }">
            <el-button link type="danger" @click="handleDeleteInstance(row)">{{ t('commons.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="app-pagination">
        <el-pagination v-model:current-page="instPager.page" v-model:page-size="instPager.pageSize" :total="instPager.total" layout="total, prev, pager, next" @current-change="loadInstances" />
      </div>

      <!-- Create Instance -->
      <el-dialog v-model="instanceCreateDialog" :title="t('database.createDB')" width="420px" destroy-on-close append-to-body>
        <el-form ref="instFormRef" :model="instForm" :rules="instRules" label-width="90px">
          <el-form-item :label="t('commons.name')" prop="name"><el-input v-model="instForm.name" /></el-form-item>
          <el-form-item label="Charset"><el-input v-model="instForm.charset" placeholder="utf8mb4" /></el-form-item>
          <el-form-item v-if="currentServerType === 'mysql'" :label="t('database.password')"><el-input v-model="instForm.password" type="password" show-password :placeholder="t('database.dbUserPasswordHint')" /></el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="instanceCreateDialog = false">{{ t('commons.cancel') }}</el-button>
          <el-button type="primary" :loading="submitting" @click="submitInstance">{{ t('commons.confirm') }}</el-button>
        </template>
      </el-dialog>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  searchDatabaseServer, createDatabaseServer, updateDatabaseServer, deleteDatabaseServer,
  testDatabaseConnection, searchDatabaseInstance, createDatabaseInstance, deleteDatabaseInstance, syncDatabaseInstances,
} from '@/api/modules/database'

const { t } = useI18n()

const loading = ref(false)
const servers = ref<any[]>([])
const searchType = ref('')
const pager = reactive({ page: 1, pageSize: 20, total: 0 })

const serverDrawer = ref(false)
const editServerMode = ref(false)
const submitting = ref(false)
const serverFormRef = ref<FormInstance>()
const defaultServerForm = () => ({ id: 0, name: '', type: 'mysql', from: 'local', address: '127.0.0.1', port: 3306, username: 'root', password: '' })
const serverForm = reactive(defaultServerForm())
const serverRules: FormRules = {
  name: [{ required: true, trigger: 'blur' }],
  type: [{ required: true }],
  from: [{ required: true }],
}

const instanceDrawer = ref(false)
const instanceLoading = ref(false)
const instances = ref<any[]>([])
const instPager = reactive({ page: 1, pageSize: 20, total: 0 })
let currentServerID = 0
const currentServerName = ref('')
const currentServerType = ref('')

const instanceCreateDialog = ref(false)
const instFormRef = ref<FormInstance>()
const instForm = reactive({ name: '', charset: 'utf8mb4', password: '', owner: '' })
const instRules: FormRules = { name: [{ required: true, trigger: 'blur' }] }

const loadServers = async () => {
  loading.value = true
  try {
    const res: any = await searchDatabaseServer({ page: pager.page, pageSize: pager.pageSize, type: searchType.value })
    servers.value = res.data.items || []
    pager.total = res.data.total
  } finally { loading.value = false }
}

const openCreateServer = () => {
  Object.assign(serverForm, defaultServerForm())
  editServerMode.value = false
  serverDrawer.value = true
}

const openEditServer = (row: any) => {
  Object.assign(serverForm, { ...row, password: '' })
  editServerMode.value = true
  serverDrawer.value = true
}

const submitServer = async () => {
  if (!serverFormRef.value) return
  await serverFormRef.value.validate()
  submitting.value = true
  try {
    if (editServerMode.value) await updateDatabaseServer(serverForm)
    else await createDatabaseServer(serverForm)
    ElMessage.success(t('commons.success'))
    serverDrawer.value = false
    await loadServers()
  } finally { submitting.value = false }
}

const handleDeleteServer = async (row: any) => {
  await ElMessageBox.confirm(t('database.deleteServerConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteDatabaseServer({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadServers()
}

const testConn = async (row: any) => {
  try {
    await testDatabaseConnection({ id: row.id })
    ElMessage.success(t('database.testSuccess'))
  } catch {
    ElMessage.error(t('database.testFail'))
  }
}

const openInstances = (row: any) => {
  currentServerID = row.id
  currentServerName.value = row.name
  currentServerType.value = row.type
  instPager.page = 1
  instanceDrawer.value = true
  loadInstances()
}

const loadInstances = async () => {
  instanceLoading.value = true
  try {
    const res: any = await searchDatabaseInstance({ page: instPager.page, pageSize: instPager.pageSize, serverID: currentServerID })
    instances.value = res.data.items || []
    instPager.total = res.data.total
  } finally { instanceLoading.value = false }
}

const openCreateInstance = () => {
  Object.assign(instForm, { name: '', charset: 'utf8mb4', password: '', owner: '' })
  instanceCreateDialog.value = true
}

const submitInstance = async () => {
  if (!instFormRef.value) return
  await instFormRef.value.validate()
  submitting.value = true
  try {
    await createDatabaseInstance({ serverID: currentServerID, ...instForm })
    ElMessage.success(t('commons.success'))
    instanceCreateDialog.value = false
    await loadInstances()
  } finally { submitting.value = false }
}

const handleDeleteInstance = async (row: any) => {
  await ElMessageBox.confirm(t('database.deleteDBConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteDatabaseInstance({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadInstances()
}

const syncInstances = async () => {
  await syncDatabaseInstances({ id: currentServerID })
  ElMessage.success(t('commons.success'))
  await loadInstances()
}

onMounted(() => loadServers())
</script>

<style scoped>
.app-toolbar { display: flex; align-items: center; margin-bottom: 16px; }
.app-pagination { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
