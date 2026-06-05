<template>
  <div>
    <el-tabs v-model="activeTab">
      <el-tab-pane :label="t('backup.accounts')" name="accounts">
        <div class="app-toolbar">
          <el-button type="primary" @click="openCreateAccount">{{ t('backup.addAccount') }}</el-button>
        </div>
        <el-table :data="accounts" v-loading="accountLoading">
          <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
          <el-table-column :label="t('backup.type')" width="120">
            <template #default="{ row }">
              <el-tag :type="typeTagMap[row.type]" size="small" effect="plain">{{ typeLabel(row.type) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="backupPath" :label="t('backup.path')" min-width="240" show-overflow-tooltip />
          <el-table-column :label="t('backup.endpoint')" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">{{ getVarField(row.vars, 'endpoint') || '-' }}</template>
          </el-table-column>
          <el-table-column :label="t('commons.actions')" width="180" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEditAccount(row)">{{ t('commons.edit') }}</el-button>
              <el-button link type="danger" @click="handleDeleteAccount(row)">{{ t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('backup.records')" name="records">
        <div class="app-toolbar">
          <el-button type="primary" @click="backupDialog = true">{{ t('backup.createBackup') }}</el-button>
          <div style="flex:1" />
          <el-select v-model="recordType" clearable :placeholder="t('backup.type')" style="width:140px" @change="loadRecords">
            <el-option :label="t('backup.typeWebsite')" value="website" />
            <el-option :label="t('backup.typeDatabase')" value="database" />
            <el-option :label="t('backup.typeDirectory')" value="directory" />
          </el-select>
        </div>
        <el-table :data="records" v-loading="recordLoading">
          <el-table-column :label="t('backup.type')" width="100">
            <template #default="{ row }">
              <el-tag size="small" effect="plain">{{ row.type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
          <el-table-column prop="fileName" :label="t('backup.fileName')" min-width="280" show-overflow-tooltip />
          <el-table-column :label="t('backup.size')" width="120">
            <template #default="{ row }">{{ formatSize(row.size) }}</template>
          </el-table-column>
          <el-table-column :label="t('backup.path')" min-width="260" show-overflow-tooltip>
            <template #default="{ row }">{{ fullRecordPath(row) || '-' }}</template>
          </el-table-column>
          <el-table-column prop="status" :label="t('backup.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small">{{ row.status === 'Success' ? t('backup.success') : t('backup.failed') }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" :label="t('backup.time')" width="180">
            <template #default="{ row }">{{ new Date(row.createdAt).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column :label="t('commons.actions')" width="150">
            <template #default="{ row }">
              <el-button link type="primary" :disabled="!fullRecordPath(row)" @click="copyRecordPath(row)">{{ t('backup.copyPath') }}</el-button>
              <el-button link type="danger" @click="handleDeleteRecord(row)">{{ t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="app-pagination">
          <el-pagination v-model:current-page="recordPager.page" v-model:page-size="recordPager.pageSize" :total="recordPager.total" layout="total, prev, pager, next" @current-change="loadRecords" />
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- Account drawer -->
    <el-drawer v-model="accountDrawer" :title="editAccountMode ? t('commons.edit') : t('backup.addAccount')" size="520px" destroy-on-close>
      <el-form ref="accountFormRef" :model="accountForm" :rules="accountRules" label-width="120px" class="drawer-form">
        <div class="drawer-section">
          <div class="drawer-section-title">{{ t('backup.basicInfo') }}</div>
          <el-form-item :label="t('commons.name')" prop="name">
            <el-input v-model="accountForm.name" :placeholder="t('backup.accountNameHint')" />
          </el-form-item>
          <el-form-item :label="t('backup.type')" prop="type">
            <el-select v-model="accountForm.type" :disabled="editAccountMode" style="width:100%" @change="onTypeChange">
              <el-option :label="t('backup.typeLocal')" value="local">
                <div class="type-option"><el-icon><FolderOpened /></el-icon><span>{{ t('backup.typeLocal') }}</span><el-text type="info" size="small">{{ t('backup.typeLocalDesc') }}</el-text></div>
              </el-option>
              <el-option label="S3" value="s3">
                <div class="type-option"><el-icon><Cloudy /></el-icon><span>S3 / MinIO</span><el-text type="info" size="small">{{ t('backup.typeS3Desc') }}</el-text></div>
              </el-option>
              <el-option label="SFTP" value="sftp">
                <div class="type-option"><el-icon><Connection /></el-icon><span>SFTP</span><el-text type="info" size="small">{{ t('backup.typeSftpDesc') }}</el-text></div>
              </el-option>
              <el-option label="WebDAV" value="webdav">
                <div class="type-option"><el-icon><Share /></el-icon><span>WebDAV</span><el-text type="info" size="small">{{ t('backup.typeWebdavDesc') }}</el-text></div>
              </el-option>
            </el-select>
          </el-form-item>
        </div>

        <div class="drawer-section">
          <div class="drawer-section-title">{{ t('backup.connectionInfo') }}</div>

          <!-- Local fields -->
          <template v-if="accountForm.type === 'local'">
            <el-form-item v-if="remoteMounts.length > 0" :label="t('backup.remoteMounts')">
              <el-select v-model="selectedMount" :placeholder="t('backup.selectMountHint')" clearable style="width: 100%" @change="onMountSelect">
                <el-option v-for="m in remoteMounts" :key="m.mountPoint" :label="m.mountPoint" :value="m.mountPoint">
                  <div class="mount-option">
                    <span class="mount-path">{{ m.mountPoint }}</span>
                    <span class="mount-info">
                      <el-tag :type="m.fsType.includes('nfs') ? 'warning' : 'success'" size="small" effect="plain">{{ m.fsType.toUpperCase() }}</el-tag>
                      <span class="mount-device">{{ m.device }}</span>
                    </span>
                  </div>
                </el-option>
              </el-select>
              <div class="form-hint">{{ t('backup.mountSelectHint') }}</div>
            </el-form-item>
            <el-form-item :label="t('backup.path')">
              <el-input v-model="accountForm.backupPath" placeholder="/opt/xpanel/backup" />
              <div class="form-hint">{{ t('backup.localPathHint') }}</div>
            </el-form-item>
          </template>

          <!-- S3 fields -->
          <template v-if="accountForm.type === 's3'">
            <el-form-item label="Endpoint" required>
              <el-input v-model="endpointField" placeholder="https://s3.amazonaws.com" />
              <div class="form-hint">{{ t('backup.s3EndpointHint') }}</div>
            </el-form-item>
            <el-form-item label="Region">
              <el-input v-model="regionField" placeholder="us-east-1" />
            </el-form-item>
            <el-form-item label="Bucket" required>
              <el-input v-model="accountForm.bucket" placeholder="my-backup-bucket" />
            </el-form-item>
            <el-form-item label="Access Key" required>
              <el-input v-model="accountForm.accessKey" />
            </el-form-item>
            <el-form-item label="Secret Key" required>
              <el-input v-model="accountForm.credential" type="password" show-password />
            </el-form-item>
            <el-form-item :label="t('backup.path')">
              <el-input v-model="accountForm.backupPath" placeholder="/xpanel-backup" />
              <div class="form-hint">{{ t('backup.s3PathHint') }}</div>
            </el-form-item>
          </template>

          <!-- SFTP fields -->
          <template v-if="accountForm.type === 'sftp'">
            <el-form-item :label="t('backup.sftpAddress')" required>
              <el-input v-model="endpointField" placeholder="192.168.1.100:22" />
            </el-form-item>
            <el-form-item :label="t('backup.sftpUser')" required>
              <el-input v-model="accountForm.accessKey" placeholder="root" />
            </el-form-item>
            <el-form-item :label="t('backup.sftpAuthMode')" required>
              <el-radio-group v-model="sftpAuthMode">
                <el-radio-button value="password">{{ t('backup.sftpAuthPassword') }}</el-radio-button>
                <el-radio-button value="key">{{ t('backup.sftpAuthKey') }}</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="sftpAuthMode === 'password'" :label="t('backup.sftpPassword')" required>
              <el-input v-model="accountForm.credential" type="password" show-password />
            </el-form-item>
            <el-form-item v-else :label="t('backup.sftpPrivateKey')" required>
              <el-input v-model="accountForm.credential" type="textarea" :rows="8" placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
            </el-form-item>
            <el-form-item v-if="sftpAuthMode === 'key'" :label="t('backup.sftpPassPhrase')">
              <el-input v-model="sftpPassPhrase" type="password" show-password />
            </el-form-item>
            <el-form-item :label="t('backup.path')">
              <el-input v-model="accountForm.backupPath" placeholder="/data/backup" />
              <div class="form-hint">{{ t('backup.sftpPathHint') }}</div>
            </el-form-item>
          </template>

          <!-- WebDAV fields -->
          <template v-if="accountForm.type === 'webdav'">
            <el-form-item label="WebDAV URL" required>
              <el-input v-model="endpointField" placeholder="https://dav.example.com/remote.php/dav/files/user/" />
            </el-form-item>
            <el-form-item :label="t('backup.webdavUser')" required>
              <el-input v-model="accountForm.accessKey" />
            </el-form-item>
            <el-form-item :label="t('backup.webdavPassword')" required>
              <el-input v-model="accountForm.credential" type="password" show-password />
            </el-form-item>
            <el-form-item :label="t('backup.path')">
              <el-input v-model="accountForm.backupPath" placeholder="/xpanel-backup" />
            </el-form-item>
          </template>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="accountDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button :loading="testing" @click="testAccount">{{ t('backup.testConn') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAccount">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Backup dialog -->
    <el-dialog v-model="backupDialog" :title="t('backup.createBackup')" width="520px" destroy-on-close>
      <el-form ref="backupFormRef" :model="backupForm" :rules="backupRules" label-width="110px" class="drawer-form">
        <el-form-item :label="t('backup.type')" prop="type">
          <el-radio-group v-model="backupForm.type" class="backup-type-group">
            <el-radio-button value="website">
              <el-icon><Monitor /></el-icon> {{ t('backup.typeWebsite') }}
            </el-radio-button>
            <el-radio-button value="database">
              <el-icon><Coin /></el-icon> {{ t('backup.typeDatabase') }}
            </el-radio-button>
            <el-radio-button value="directory">
              <el-icon><FolderOpened /></el-icon> {{ t('backup.typeDirectory') }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('commons.name')" prop="name">
          <el-input v-model="backupForm.name" :placeholder="backupForm.type === 'website' ? 'example.com' : backupForm.type === 'database' ? 'my_database' : '/data/myapp'" />
        </el-form-item>
        <el-form-item :label="t('backup.account')" prop="accountID">
          <el-select v-model="backupForm.accountID" style="width:100%">
            <el-option v-for="a in accounts" :key="a.id" :label="a.name + ' (' + typeLabel(a.type) + ')'" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="backupForm.type === 'database'" :label="t('backup.dbType')">
          <el-select v-model="backupForm.dbType" style="width:100%">
            <el-option label="MySQL" value="mysql" />
            <el-option label="PostgreSQL" value="postgresql" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="backupForm.type === 'directory'" :label="t('backup.sourceDir')">
          <el-input v-model="backupForm.sourceDir" placeholder="/data/myapp" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="backupDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitBackup">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { FolderOpened, Cloudy, Connection, Share, Monitor, Coin } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import type { BackupAccount, BackupRecord } from '@/api/interface'
import {
  listBackupAccounts, createBackupAccount, updateBackupAccount, deleteBackupAccount,
  testBackupAccount, createBackup, searchBackupRecords, deleteBackupRecord,
} from '@/api/modules/backup'
import { listRemoteMounts } from '@/api/modules/disk'

const { t } = useI18n()
const activeTab = ref('accounts')

const typeTagMap: Record<string, string> = { local: 'success', s3: '', sftp: 'warning', webdav: 'info' }
const typeLabel = (type: string) => {
  const map: Record<string, string> = { local: t('backup.typeLocal'), s3: 'S3 / MinIO', sftp: 'SFTP', webdav: 'WebDAV' }
  return map[type] || type
}
const getVarField = (vars: string, field: string) => {
  try { return JSON.parse(vars || '{}')[field] || '' } catch { return '' }
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
const fullRecordPath = (row: BackupRecord) => {
  if (!row.fileName) return ''
  if (!row.fileDir || row.fileDir === '.') return row.fileName
  return `${row.fileDir.replace(/\/$/, '')}/${row.fileName}`
}

const accountLoading = ref(false)
const accounts = ref<BackupAccount[]>([])
const accountDrawer = ref(false)
const editAccountMode = ref(false)
const submitting = ref(false)
const testing = ref(false)
const accountFormRef = ref<FormInstance>()
const defaultAccountForm = () => ({ id: 0, name: '', type: 'local', bucket: '', accessKey: '', credential: '', backupPath: '/opt/xpanel/backup', vars: '' })
const accountForm = reactive(defaultAccountForm())
const accountRules: FormRules = { name: [{ required: true, trigger: 'blur' }], type: [{ required: true }] }
const endpointField = ref('')
const regionField = ref('')
const sftpAuthMode = ref<'password' | 'key'>('password')
const sftpPassPhrase = ref('')

const remoteMounts = ref<any[]>([])
const selectedMount = ref('')

const loadRemoteMounts = async () => {
  try {
    const res = await listRemoteMounts()
    remoteMounts.value = res.data || []
  } catch { remoteMounts.value = [] }
}

const onMountSelect = (mountPoint: string) => {
  if (mountPoint) {
    accountForm.backupPath = mountPoint.replace(/\/$/, '') + '/xpanel-backup'
  }
}

const onTypeChange = (type: string) => {
  accountForm.accessKey = ''
  accountForm.credential = ''
  accountForm.bucket = ''
  endpointField.value = ''
  regionField.value = ''
  sftpAuthMode.value = 'password'
  sftpPassPhrase.value = ''
  selectedMount.value = ''
  switch (type) {
    case 'local': accountForm.backupPath = '/opt/xpanel/backup'; break
    case 's3': accountForm.backupPath = '/xpanel-backup'; break
    case 'sftp': accountForm.backupPath = '/data/backup'; break
    case 'webdav': accountForm.backupPath = '/xpanel-backup'; break
  }
}

const recordLoading = ref(false)
const records = ref<BackupRecord[]>([])
const recordType = ref('')
const recordPager = reactive({ page: 1, pageSize: 20, total: 0 })

const backupDialog = ref(false)
const backupFormRef = ref<FormInstance>()
const backupForm = reactive({ type: 'website', name: '', accountID: 0 as number, dbType: 'mysql', sourceDir: '' })
const backupRules: FormRules = { type: [{ required: true }], name: [{ required: true, trigger: 'blur' }], accountID: [{ required: true, message: () => t('backup.selectAccount') }] }

const loadAccounts = async () => {
  accountLoading.value = true
  try {
    const res = await listBackupAccounts()
    accounts.value = res.data || []
  } finally { accountLoading.value = false }
}

const openCreateAccount = () => {
  Object.assign(accountForm, defaultAccountForm())
  endpointField.value = ''
  regionField.value = ''
  sftpAuthMode.value = 'password'
  sftpPassPhrase.value = ''
  selectedMount.value = ''
  editAccountMode.value = false
  accountDrawer.value = true
  loadRemoteMounts()
}

const openEditAccount = (row: BackupAccount) => {
  Object.assign(accountForm, { ...row, credential: '' })
  try {
    const v = JSON.parse(row.vars || '{}')
    endpointField.value = v.endpoint || ''
    regionField.value = v.region || ''
    sftpAuthMode.value = v.authMode || 'password'
    sftpPassPhrase.value = v.passPhrase || ''
  } catch {
    endpointField.value = ''
    regionField.value = ''
    sftpAuthMode.value = 'password'
    sftpPassPhrase.value = ''
  }
  selectedMount.value = ''
  editAccountMode.value = true
  accountDrawer.value = true
  if (row.type === 'local') loadRemoteMounts()
}

const buildAccountVars = () => {
  const vars: Record<string, string> = { endpoint: endpointField.value, region: regionField.value }
  if (accountForm.type === 'sftp') {
    vars.authMode = sftpAuthMode.value
    vars.passPhrase = sftpPassPhrase.value
  }
  return JSON.stringify(vars)
}

const testAccount = async () => {
  if (!accountFormRef.value) return
  await accountFormRef.value.validate()
  testing.value = true
  try {
    accountForm.vars = buildAccountVars()
    await testBackupAccount(accountForm)
    ElMessage.success(t('backup.testSuccess'))
  } catch (err: any) {
    ElMessage.error(err?.message || t('backup.testFail'))
  } finally { testing.value = false }
}

const submitAccount = async () => {
  if (!accountFormRef.value) return
  await accountFormRef.value.validate()
  submitting.value = true
  try {
    accountForm.vars = buildAccountVars()
    if (editAccountMode.value) await updateBackupAccount(accountForm)
    else await createBackupAccount(accountForm)
    ElMessage.success(t('commons.success'))
    accountDrawer.value = false
    await loadAccounts()
  } finally { submitting.value = false }
}

const handleDeleteAccount = async (row: BackupAccount) => {
  await ElMessageBox.confirm(t('backup.deleteAccountConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteBackupAccount({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadAccounts()
}

const loadRecords = async () => {
  recordLoading.value = true
  try {
    const res = await searchBackupRecords({ page: recordPager.page, pageSize: recordPager.pageSize, type: recordType.value })
    records.value = res.data.items || []
    recordPager.total = res.data.total
  } finally { recordLoading.value = false }
}

const submitBackup = async () => {
  if (!backupFormRef.value) return
  await backupFormRef.value.validate()
  submitting.value = true
  try {
    await createBackup(backupForm)
    ElMessage.success(t('backup.started'))
    backupDialog.value = false
  } finally { submitting.value = false }
}

const handleDeleteRecord = async (row: BackupRecord) => {
  await ElMessageBox.confirm(t('backup.deleteRecordConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteBackupRecord({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadRecords()
}

const copyRecordPath = async (row: BackupRecord) => {
  const path = fullRecordPath(row)
  if (!path) return
  await navigator.clipboard.writeText(path)
  ElMessage.success(t('backup.pathCopied'))
}

watch(activeTab, (tab) => {
  if (tab === 'records') loadRecords()
})

onMounted(() => loadAccounts())
</script>

<style lang="scss" scoped>
.drawer-form {
  :deep(.el-form-item__label) {
    font-weight: 500;
  }
}

.drawer-section {
  margin-bottom: 24px;
  padding-bottom: 8px;
}

.drawer-section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--xp-text-primary);
  margin-bottom: 16px;
  padding-left: 10px;
  border-left: 3px solid var(--xp-accent);
  line-height: 1;
}

.type-option {
  display: flex;
  align-items: center;
  gap: 8px;

  .el-icon { color: var(--xp-accent); opacity: 0.7; }
}

.form-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.mount-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.mount-path {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
}

.mount-info {
  display: flex;
  align-items: center;
  gap: 6px;
}

.mount-device {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.backup-type-group {
  width: 100%;

  :deep(.el-radio-button) {
    flex: 1;
  }

  :deep(.el-radio-button__inner) {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
  }
}
</style>
