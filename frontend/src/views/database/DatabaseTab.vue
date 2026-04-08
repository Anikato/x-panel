<template>
  <div>
    <!-- Server list -->
    <div class="app-toolbar">
      <el-button type="primary" @click="openCreateServer">{{ t('database.addServer') }}</el-button>
    </div>

    <el-table :data="servers" v-loading="loading" style="width:100%">
      <el-table-column type="expand">
        <template #default="{ row }">
          <div class="instance-panel">
            <div class="instance-toolbar">
              <el-button type="primary" size="small" @click="openCreateInstance(row)">{{ t('database.createDB') }}</el-button>
              <el-button size="small" @click="syncInstances(row)">
                <el-icon><Refresh /></el-icon> {{ t('database.sync') }}
              </el-button>
              <div style="flex:1" />
              <el-text type="info" size="small">{{ t('database.instanceCount', { n: row._instances?.length || 0 }) }}</el-text>
            </div>
            <el-table :data="row._instances || []" v-loading="row._loading" size="small" style="width:100%">
              <el-table-column prop="name" :label="t('database.dbName')" min-width="160" />
              <el-table-column prop="charset" label="Charset" width="120" />
              <el-table-column v-if="dbType === 'postgresql'" prop="owner" label="Owner" width="120" />
              <el-table-column :label="t('commons.createdAt')" width="180">
                <template #default="{ row: inst }">{{ inst.createdAt ? new Date(inst.createdAt).toLocaleString() : '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('commons.actions')" width="320" fixed="right">
                <template #default="{ row: inst }">
                  <el-button link type="primary" size="small" @click="handleBackup(row, inst)">{{ t('database.backup') }}</el-button>
                  <el-button link type="success" size="small" @click="openRestore(row, inst)">{{ t('database.restore') }}</el-button>
                  <el-button link type="primary" size="small" @click="openChangePassword(row, inst)">{{ t('database.changePassword') }}</el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteInstance(row, inst)">{{ t('commons.delete') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
      <el-table-column :label="t('database.from')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.from === 'local' ? 'success' : 'warning'" size="small" effect="plain">{{ row.from === 'local' ? t('database.local') : t('database.remote') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('database.address')" width="200">
        <template #default="{ row }">{{ row.address }}:{{ row.port }}</template>
      </el-table-column>
      <el-table-column prop="username" :label="t('database.username')" width="120" />
      <el-table-column :label="t('commons.actions')" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="testConn(row)">{{ t('database.testConn') }}</el-button>
          <el-button link type="primary" @click="openEditServer(row)">{{ t('commons.edit') }}</el-button>
          <el-button link type="danger" @click="handleDeleteServer(row)">{{ t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Server Create/Edit Drawer -->
    <el-drawer v-model="serverDrawer" :title="editServerMode ? t('commons.edit') : t('database.addServer')" size="500px" destroy-on-close>
      <el-form ref="serverFormRef" :model="serverForm" :rules="serverRules" label-width="100px">
        <el-form-item :label="t('commons.name')" prop="name">
          <el-input v-model="serverForm.name" :placeholder="dbType === 'mysql' ? 'MySQL-Local' : 'PG-Local'" />
        </el-form-item>
        <el-form-item :label="t('database.from')">
          <el-radio-group v-model="serverForm.from" @change="onFromChange">
            <el-radio value="local">{{ t('database.local') }}</el-radio>
            <el-radio value="remote">{{ t('database.remote') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('database.address')">
          <el-input v-model="serverForm.address" :placeholder="serverForm.from === 'local' ? '127.0.0.1' : '192.168.1.100'" :disabled="serverForm.from === 'local'" />
        </el-form-item>
        <el-form-item :label="t('database.port')">
          <el-input-number v-model="serverForm.port" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item :label="t('database.username')">
          <el-input v-model="serverForm.username" :placeholder="dbType === 'mysql' ? 'root' : 'postgres'" />
        </el-form-item>
        <el-form-item :label="t('database.password')">
          <el-input v-model="serverForm.password" type="password" show-password :placeholder="editServerMode ? t('database.passwordNoChange') : ''" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="serverDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button @click="testServerConn" :loading="testing">{{ t('database.testConn') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitServer">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Create Instance Dialog -->
    <el-dialog v-model="instanceCreateDialog" :title="t('database.createDB')" width="460px" destroy-on-close>
      <el-form ref="instFormRef" :model="instForm" :rules="instRules" label-width="100px">
        <el-form-item :label="t('database.dbName')" prop="name">
          <el-input v-model="instForm.name" placeholder="my_database" />
        </el-form-item>
        <el-form-item v-if="dbType === 'mysql'" label="Charset">
          <el-select v-model="instForm.charset" style="width:100%">
            <el-option label="utf8mb4" value="utf8mb4" />
            <el-option label="utf8" value="utf8" />
            <el-option label="latin1" value="latin1" />
            <el-option label="gbk" value="gbk" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="dbType === 'mysql'" :label="t('database.password')">
          <el-input v-model="instForm.password" type="password" show-password :placeholder="t('database.dbUserPasswordHint')" />
        </el-form-item>
        <el-form-item v-if="dbType === 'postgresql'" label="Owner">
          <el-input v-model="instForm.owner" :placeholder="t('database.ownerHint')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="instanceCreateDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitInstance">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Change Password Dialog -->
    <el-dialog v-model="passwordDialog" :title="t('database.changePassword')" width="400px" destroy-on-close>
      <el-form :model="passwordForm" label-width="100px">
        <el-form-item :label="t('database.dbName')">
          <el-input :model-value="passwordForm.dbName" disabled />
        </el-form-item>
        <el-form-item :label="t('database.newPassword')">
          <el-input v-model="passwordForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitChangePassword">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
    <!-- Restore Dialog -->
    <el-dialog v-model="restoreDialog" :title="t('database.restore')" width="500px" destroy-on-close>
      <el-form label-width="100px">
        <el-form-item :label="t('database.dbName')">
          <el-input :model-value="restoreForm.dbName" disabled />
        </el-form-item>
        <el-form-item :label="t('database.backupFile')">
          <el-input v-model="restoreForm.file" :placeholder="t('database.restoreFileHint')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="restoreDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="restoring" @click="submitRestore">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, defineExpose } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { DatabaseServer, DatabaseInstance } from '@/api/interface'
import {
  searchDatabaseServer, createDatabaseServer, updateDatabaseServer, deleteDatabaseServer,
  testDatabaseConnection, searchDatabaseInstance, createDatabaseInstance, deleteDatabaseInstance,
  syncDatabaseInstances, changeInstancePassword, backupDatabaseInstance, restoreDatabaseInstance,
} from '@/api/modules/database'

const props = defineProps<{ dbType: string }>()
const { t } = useI18n()

const loading = ref(false)
const servers = ref<DatabaseServer[]>([])
const submitting = ref(false)
const testing = ref(false)

const loadServers = async () => {
  loading.value = true
  try {
    const res = await searchDatabaseServer({ page: 1, pageSize: 100, type: props.dbType })
    const items = res.data.items || []
    for (const s of items) {
      s._instances = []
      s._loading = false
    }
    servers.value = items
    for (const s of items) {
      loadInstancesForServer(s)
    }
  } finally { loading.value = false }
}

const loadInstancesForServer = async (server: DatabaseServer) => {
  server._loading = true
  try {
    const res = await searchDatabaseInstance({ page: 1, pageSize: 100, serverID: server.id })
    server._instances = res.data.items || []
  } finally { server._loading = false }
}

// Server CRUD
const serverDrawer = ref(false)
const editServerMode = ref(false)
const serverFormRef = ref<FormInstance>()
const defaultServerForm = () => ({
  id: 0, name: '', type: props.dbType, from: 'local',
  address: '127.0.0.1', port: props.dbType === 'mysql' ? 3306 : 5432,
  username: props.dbType === 'mysql' ? 'root' : 'postgres', password: '',
})
const serverForm = reactive(defaultServerForm())
const serverRules: FormRules = {
  name: [{ required: true, trigger: 'blur' }],
}

const onFromChange = (val: string) => {
  if (val === 'local') serverForm.address = '127.0.0.1'
  else serverForm.address = ''
}

const openCreateServer = () => {
  Object.assign(serverForm, defaultServerForm())
  editServerMode.value = false
  serverDrawer.value = true
}

const openEditServer = (row: DatabaseServer) => {
  Object.assign(serverForm, { ...row, password: '' })
  editServerMode.value = true
  serverDrawer.value = true
}

const testServerConn = async () => {
  if (!serverForm.name) {
    ElMessage.warning(t('database.pleaseComplete'))
    return
  }
  if (editServerMode.value && serverForm.id) {
    testing.value = true
    try {
      await testDatabaseConnection({ id: serverForm.id })
      ElMessage.success(t('database.testSuccess'))
    } catch {
      ElMessage.error(t('database.testFail'))
    } finally { testing.value = false }
  } else {
    ElMessage.info(t('database.saveFirst'))
  }
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

const handleDeleteServer = async (row: DatabaseServer) => {
  await ElMessageBox.confirm(t('database.deleteServerConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteDatabaseServer({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadServers()
}

const testConn = async (row: DatabaseServer) => {
  try {
    await testDatabaseConnection({ id: row.id })
    ElMessage.success(t('database.testSuccess'))
  } catch {
    ElMessage.error(t('database.testFail'))
  }
}

// Instances
const instanceCreateDialog = ref(false)
const instFormRef = ref<FormInstance>()
const instForm = reactive({ name: '', charset: 'utf8mb4', password: '', owner: '' })
const instRules: FormRules = { name: [{ required: true, trigger: 'blur' }] }
let currentServer: DatabaseServer | null = null

const openCreateInstance = (server: DatabaseServer) => {
  currentServer = server
  Object.assign(instForm, { name: '', charset: 'utf8mb4', password: '', owner: '' })
  instanceCreateDialog.value = true
}

const submitInstance = async () => {
  if (!instFormRef.value || !currentServer) return
  await instFormRef.value.validate()
  submitting.value = true
  try {
    await createDatabaseInstance({ serverID: currentServer.id, ...instForm })
    ElMessage.success(t('commons.success'))
    instanceCreateDialog.value = false
    await loadInstancesForServer(currentServer)
  } finally { submitting.value = false }
}

const handleDeleteInstance = async (server: DatabaseServer, inst: DatabaseInstance) => {
  await ElMessageBox.confirm(t('database.deleteDBConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteDatabaseInstance({ id: inst.id })
  ElMessage.success(t('commons.success'))
  await loadInstancesForServer(server)
}

const syncInstances = async (server: DatabaseServer) => {
  server._loading = true
  try {
    await syncDatabaseInstances({ id: server.id })
    ElMessage.success(t('commons.success'))
    await loadInstancesForServer(server)
  } finally { server._loading = false }
}

// Change Password
const passwordDialog = ref(false)
const passwordForm = reactive({ id: 0, dbName: '', password: '' })
let passwordServer: DatabaseServer | null = null

const openChangePassword = (server: DatabaseServer, inst: DatabaseInstance) => {
  passwordServer = server
  passwordForm.id = inst.id
  passwordForm.dbName = inst.name
  passwordForm.password = ''
  passwordDialog.value = true
}

const submitChangePassword = async () => {
  if (!passwordForm.password) {
    ElMessage.warning(t('database.newPasswordRequired'))
    return
  }
  submitting.value = true
  try {
    await changeInstancePassword({ id: passwordForm.id, password: passwordForm.password })
    ElMessage.success(t('commons.success'))
    passwordDialog.value = false
  } finally { submitting.value = false }
}

// Restore
const restoreDialog = ref(false)
const restoring = ref(false)
const restoreForm = reactive({ id: 0, dbName: '', file: '' })
let restoreServer: DatabaseServer | null = null

const openRestore = (server: DatabaseServer, inst: DatabaseInstance) => {
  restoreServer = server
  restoreForm.id = inst.id
  restoreForm.dbName = inst.name
  restoreForm.file = ''
  restoreDialog.value = true
}

const submitRestore = async () => {
  if (!restoreForm.file.trim()) {
    ElMessage.warning(t('database.restoreFileRequired'))
    return
  }
  try {
    await ElMessageBox.confirm(t('database.restoreConfirm', { name: restoreForm.dbName }), t('commons.tip'), { type: 'warning' })
  } catch { return }
  restoring.value = true
  try {
    await restoreDatabaseInstance({ id: restoreForm.id, file: restoreForm.file })
    ElMessage.success(t('commons.success'))
    restoreDialog.value = false
  } finally { restoring.value = false }
}

// Backup
const handleBackup = async (server: DatabaseServer, inst: DatabaseInstance) => {
  try {
    await ElMessageBox.confirm(t('database.backupConfirm', { name: inst.name }), t('commons.tip'))
  } catch { return }
  try {
    const res = await backupDatabaseInstance({ id: inst.id })
    ElMessage.success(t('database.backupSuccess', { file: res.data.file }))
  } catch { /* handled by interceptor */ }
}

const refresh = () => loadServers()
defineExpose({ refresh })

onMounted(() => loadServers())
</script>

<style scoped>
.instance-panel {
  padding: 8px 16px;
}
.instance-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
</style>
