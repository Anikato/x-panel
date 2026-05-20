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
              <el-table-column v-if="dbType === 'mysql'" prop="username" :label="t('database.username')" width="140" />
              <el-table-column v-if="dbType === 'mysql'" prop="permission" :label="t('database.permission')" width="140" />
              <el-table-column v-if="dbType === 'postgresql'" prop="username" :label="t('database.username')" width="140" />
              <el-table-column v-if="dbType === 'postgresql'" prop="owner" label="Owner" width="120" />
              <el-table-column v-if="dbType === 'postgresql'" :label="t('database.superUser')" width="110">
                <template #default="{ row: inst }">
                  <el-tag :type="inst.superUser ? 'danger' : 'info'" size="small" effect="plain">
                    {{ inst.superUser ? t('database.superUser') : t('database.normalUser') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="t('database.backupOverview')" min-width="260">
                <template #default="{ row: inst }">
                  <div v-if="inst._backupLoading" class="backup-overview muted">{{ t('database.loadingBackups') }}</div>
                  <div v-else-if="inst._backupTotal" class="backup-overview">
                    <el-tag size="small" effect="plain">{{ t('database.backupCount', { n: inst._backupTotal }) }}</el-tag>
                    <span>{{ formatSize(inst._latestBackup?.size || 0) }}</span>
                    <span>{{ formatTime(inst._latestBackup?.createdAt) }}</span>
                  </div>
                  <el-text v-else type="info" size="small">{{ t('database.noBackups') }}</el-text>
                </template>
              </el-table-column>
              <el-table-column :label="t('commons.createdAt')" width="180">
                <template #default="{ row: inst }">{{ inst.createdAt ? new Date(inst.createdAt).toLocaleString() : '-' }}</template>
              </el-table-column>
              <el-table-column :label="t('commons.actions')" width="460" fixed="right">
                <template #default="{ row: inst }">
                  <el-button link type="primary" size="small" @click="handleBackup(row, inst)">{{ t('database.backup') }}</el-button>
                  <el-button link type="primary" size="small" @click="openBackupHistory(row, inst)">{{ t('database.backupHistory') }}</el-button>
                  <el-button link type="success" size="small" @click="openRestore(row, inst)">{{ t('database.restore') }}</el-button>
                  <el-button link type="primary" size="small" @click="openChangePassword(row, inst)">{{ t('database.changePassword') }}</el-button>
                  <el-button v-if="dbType === 'postgresql'" link type="primary" size="small" @click="handleChangePrivileges(row, inst)">{{ t('database.changePrivileges') }}</el-button>
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
        <el-form-item v-if="dbType === 'mysql'" :label="t('database.username')">
          <el-input v-model="instForm.username" :placeholder="t('database.mysqlUsernameHint')" />
        </el-form-item>
        <el-form-item v-if="dbType === 'mysql'" :label="t('database.password')">
          <el-input v-model="instForm.password" type="password" show-password :placeholder="t('database.mysqlPasswordHint')" />
        </el-form-item>
        <el-form-item v-if="dbType === 'mysql'" :label="t('database.permission')">
          <el-select
            v-model="instForm.permission"
            filterable
            allow-create
            default-first-option
            style="width:100%"
            :placeholder="t('database.permissionHint')"
          >
            <el-option label="%" value="%" />
            <el-option label="localhost" value="localhost" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="dbType === 'postgresql'" :label="t('database.username')">
          <el-input v-model="instForm.username" :placeholder="t('database.pgUsernameHint')" />
        </el-form-item>
        <el-form-item v-if="dbType === 'postgresql'" :label="t('database.password')">
          <el-input v-model="instForm.password" type="password" show-password :placeholder="t('database.pgPasswordHint')" />
        </el-form-item>
        <el-form-item v-if="dbType === 'postgresql'" :label="t('database.superUser')">
          <el-switch v-model="instForm.superUser" />
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
          <div class="restore-file-row">
            <el-input v-model="restoreForm.file" :placeholder="t('database.restoreFileHint')" />
            <el-upload
              :auto-upload="false"
              :show-file-list="false"
              accept=".sql,.gz,.zip,.tar,.dump"
              :on-change="handleRestoreFileChange"
            >
              <el-button :icon="Upload" :loading="restoreUploading">
                {{ restoreUploading ? `${restoreUploadProgress}%` : t('database.uploadRestoreFile') }}
              </el-button>
            </el-upload>
          </div>
        </el-form-item>
        <el-form-item :label="t('database.backupRecord')">
          <el-select
            v-model="restoreForm.backupRecordID"
            clearable
            filterable
            :placeholder="t('database.backupRecordHint')"
            style="width:100%"
          >
            <el-option
              v-for="record in backupRecords"
              :key="record.id"
              :label="formatBackupRecord(record)"
              :value="record.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="restoreDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="restoring" @click="submitRestore">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Backup History Dialog -->
    <el-dialog v-model="backupHistoryDialog" :title="t('database.backupHistoryTitle', { name: backupHistoryForm.dbName || '-' })" width="920px" destroy-on-close>
      <el-table :data="historyRecords" v-loading="historyLoading" size="small">
        <el-table-column prop="fileName" :label="t('backup.fileName')" min-width="240" show-overflow-tooltip />
        <el-table-column :label="t('backup.size')" width="110">
          <template #default="{ row }">{{ formatSize(row.size) }}</template>
        </el-table-column>
        <el-table-column :label="t('database.backupSource')" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ backupRecordSource(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('backup.path')" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ fullRecordPath(row) || '-' }}</template>
        </el-table-column>
        <el-table-column prop="status" :label="t('backup.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small">
              {{ row.status === 'Success' ? t('backup.success') : t('backup.failed') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('backup.time')" width="170">
          <template #default="{ row }">{{ formatTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('commons.actions')" width="190" fixed="right">
          <template #default="{ row }">
            <el-button link type="success" size="small" :disabled="row.status !== 'Success'" @click="restoreFromHistory(row)">{{ t('database.restore') }}</el-button>
            <el-button link type="primary" size="small" :disabled="!fullRecordPath(row)" @click="copyRecordPath(row)">{{ t('backup.copyPath') }}</el-button>
            <el-button link type="danger" size="small" @click="deleteHistoryRecord(row)">{{ t('commons.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="app-pagination">
        <el-pagination
          v-model:current-page="historyPager.page"
          v-model:page-size="historyPager.pageSize"
          :total="historyPager.total"
          layout="total, prev, pager, next"
          @current-change="loadBackupHistory"
        />
      </div>
      <template #footer>
        <el-button @click="backupHistoryDialog = false">{{ t('commons.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, defineExpose } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Upload } from '@element-plus/icons-vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { UploadFile } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { DatabaseServer, DatabaseInstance, BackupAccount } from '@/api/interface'
import type { BackupRecord } from '@/api/interface'
import { useFileTaskStore } from '@/store/modules/fileTask'
import {
  searchDatabaseServer, createDatabaseServer, updateDatabaseServer, deleteDatabaseServer,
  testDatabaseConnection, searchDatabaseInstance, createDatabaseInstance, deleteDatabaseInstance,
  syncDatabaseInstances, changeInstancePassword, changeInstancePrivileges, backupDatabaseInstance, restoreDatabaseInstance,
  uploadDatabaseRestoreFile,
} from '@/api/modules/database'
import { deleteBackupRecord, listBackupAccounts, searchBackupRecords } from '@/api/modules/backup'

const props = defineProps<{ dbType: string }>()
const { t } = useI18n()
const fileTaskStore = useFileTaskStore()

const loading = ref(false)
const servers = ref<DatabaseServer[]>([])
const submitting = ref(false)
const testing = ref(false)
const backupAccounts = ref<BackupAccount[]>([])

const loadServers = async () => {
  loading.value = true
  try {
    loadBackupAccounts()
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
    const instances = res.data.items || []
    server._instances = instances
    for (const inst of instances) {
      loadBackupSummary(inst)
    }
  } finally { server._loading = false }
}

const loadBackupAccounts = async () => {
  try {
    const res = await listBackupAccounts()
    backupAccounts.value = res.data || []
  } catch {
    backupAccounts.value = []
  }
}

const loadBackupSummary = async (inst: DatabaseInstance) => {
  inst._backupLoading = true
  try {
    const res = await searchBackupRecords({
      page: 1,
      pageSize: 1,
      type: 'database',
      name: inst.name,
      status: 'Success',
    })
    inst._backupTotal = res.data.total || 0
    inst._latestBackup = res.data.items?.[0]
  } catch {
    inst._backupTotal = 0
    inst._latestBackup = undefined
  } finally {
    inst._backupLoading = false
  }
}

const formatSize = (size: number) => {
  if (!size) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = size
  let idx = 0
  while (value >= 1024 && idx < units.length - 1) {
    value /= 1024
    idx++
  }
  return `${value.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`
}

const formatTime = (time?: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString()
}

const fullRecordPath = (row: BackupRecord) => {
  if (!row.fileName) return ''
  if (!row.fileDir || row.fileDir === '.') return row.fileName
  return `${row.fileDir.replace(/\/$/, '')}/${row.fileName}`
}

const backupRecordSource = (row: BackupRecord) => {
  if (!row.accountID) return t('database.serverDisk')
  const account = backupAccounts.value.find(item => item.id === row.accountID)
  return account ? `${account.name} (${account.type})` : `#${row.accountID}`
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
const instForm = reactive({ name: '', charset: 'utf8mb4', password: '', owner: '', username: '', permission: '%', superUser: false })
const instRules: FormRules = { name: [{ required: true, trigger: 'blur' }] }
let currentServer: DatabaseServer | null = null

const openCreateInstance = (server: DatabaseServer) => {
  currentServer = server
  Object.assign(instForm, { name: '', charset: 'utf8mb4', password: '', owner: '', username: '', permission: '%', superUser: false })
  instanceCreateDialog.value = true
}

const submitInstance = async () => {
  if (!instFormRef.value || !currentServer) return
  await instFormRef.value.validate()
  if (props.dbType === 'postgresql' && !instForm.password) {
    ElMessage.warning(t('database.pgPasswordRequired'))
    return
  }
  if (props.dbType === 'mysql' && !instForm.password) {
    ElMessage.warning(t('database.mysqlPasswordRequired'))
    return
  }
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

const handleChangePrivileges = async (server: DatabaseServer, inst: DatabaseInstance) => {
  const nextSuperUser = !inst.superUser
  const role = nextSuperUser ? t('database.superUser') : t('database.normalUser')
  const username = inst.username || inst.owner || inst.name
  await ElMessageBox.confirm(t('database.privilegesConfirm', { name: username, role }), t('commons.tip'), { type: 'warning' })
  await changeInstancePrivileges({ id: inst.id, superUser: nextSuperUser })
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
const restoreUploading = ref(false)
const restoreUploadProgress = ref(0)
const restoreForm = reactive({ id: 0, dbName: '', file: '', backupRecordID: undefined as number | undefined })
const backupRecords = ref<BackupRecord[]>([])
let restoreServer: DatabaseServer | null = null

const openRestore = async (server: DatabaseServer, inst: DatabaseInstance) => {
  restoreServer = server
  restoreForm.id = inst.id
  restoreForm.dbName = inst.name
  restoreForm.file = ''
  restoreForm.backupRecordID = undefined
  restoreUploadProgress.value = 0
  backupRecords.value = []
  restoreDialog.value = true
  try {
    const res = await searchBackupRecords({
      page: 1,
      pageSize: 50,
      type: 'database',
      name: inst.name,
      status: 'Success',
    })
    backupRecords.value = res.data.items || []
  } catch {
    backupRecords.value = []
  }
}

const handleRestoreFileChange = async (uploadFile: UploadFile) => {
  const raw = uploadFile.raw
  if (!raw) return
  restoreUploading.value = true
  restoreUploadProgress.value = 0
  try {
    const res = await uploadDatabaseRestoreFile(raw, (percent) => {
      restoreUploadProgress.value = percent
    })
    restoreForm.file = res.data.file
    restoreForm.backupRecordID = undefined
    ElMessage.success(t('database.uploadRestoreSuccess'))
  } finally {
    restoreUploading.value = false
  }
}

const submitRestore = async () => {
  if (!restoreForm.file.trim() && !restoreForm.backupRecordID) {
    ElMessage.warning(t('database.restoreFileRequired'))
    return false
  }
  try {
    await ElMessageBox.confirm(t('database.restoreConfirm', { name: restoreForm.dbName }), t('commons.tip'), { type: 'warning' })
  } catch { return false }
  restoring.value = true
  try {
    const res = await restoreDatabaseInstance({
      id: restoreForm.id,
      file: restoreForm.file.trim() || undefined,
      backupRecordID: restoreForm.backupRecordID,
    })
    if (res.data?.taskID) {
      ElMessage.info(t('database.restoreTaskStarted'))
      fileTaskStore.fetchTasks()
    } else {
      ElMessage.success(t('commons.success'))
    }
    restoreDialog.value = false
    return true
  } finally { restoring.value = false }
}

const formatBackupRecord = (record: BackupRecord) => {
  const time = record.createdAt ? new Date(record.createdAt).toLocaleString() : '-'
  return `${record.fileName} (${time})`
}

// Backup
const handleBackup = async (server: DatabaseServer, inst: DatabaseInstance) => {
  try {
    await ElMessageBox.confirm(t('database.backupConfirm', { name: inst.name }), t('commons.tip'))
  } catch { return }
  try {
    const res = await backupDatabaseInstance({ id: inst.id })
    if (res.data?.taskID) {
      ElMessage.info(t('database.backupTaskStarted'))
      fileTaskStore.fetchTasks()
      setTimeout(() => loadBackupSummary(inst), 3000)
    } else {
      ElMessage.success(t('database.backupSuccess', { file: res.data.file }))
      await loadBackupSummary(inst)
    }
  } catch { /* handled by interceptor */ }
}

// Backup history
const backupHistoryDialog = ref(false)
const historyLoading = ref(false)
const historyRecords = ref<BackupRecord[]>([])
const historyPager = reactive({ page: 1, pageSize: 10, total: 0 })
const backupHistoryForm = reactive({ id: 0, dbName: '' })
let backupHistoryInstance: DatabaseInstance | null = null

const openBackupHistory = async (server: DatabaseServer, inst: DatabaseInstance) => {
  restoreServer = server
  backupHistoryInstance = inst
  backupHistoryForm.id = inst.id
  backupHistoryForm.dbName = inst.name
  historyPager.page = 1
  backupHistoryDialog.value = true
  await loadBackupHistory()
}

const loadBackupHistory = async () => {
  if (!backupHistoryForm.dbName) return
  historyLoading.value = true
  try {
    const res = await searchBackupRecords({
      page: historyPager.page,
      pageSize: historyPager.pageSize,
      type: 'database',
      name: backupHistoryForm.dbName,
    })
    historyRecords.value = res.data.items || []
    historyPager.total = res.data.total || 0
  } finally {
    historyLoading.value = false
  }
}

const restoreFromHistory = async (record: BackupRecord) => {
  restoreForm.id = backupHistoryForm.id
  restoreForm.dbName = backupHistoryForm.dbName
  restoreForm.file = ''
  restoreForm.backupRecordID = record.id
  const ok = await submitRestore()
  if (ok) backupHistoryDialog.value = false
}

const deleteHistoryRecord = async (row: BackupRecord) => {
  await ElMessageBox.confirm(t('backup.deleteRecordConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteBackupRecord({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadBackupHistory()
  if (backupHistoryInstance) await loadBackupSummary(backupHistoryInstance)
}

const copyRecordPath = async (row: BackupRecord) => {
  const path = fullRecordPath(row)
  if (!path) return
  await navigator.clipboard.writeText(path)
  ElMessage.success(t('backup.pathCopied'))
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
.restore-file-row {
  display: flex;
  width: 100%;
  gap: 8px;
}
.restore-file-row .el-input {
  flex: 1;
}
</style>
