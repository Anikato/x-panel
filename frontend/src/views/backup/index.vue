<template>
  <div>
    <el-tabs v-model="activeTab">
      <el-tab-pane :label="t('backup.accounts')" name="accounts">
        <div class="app-toolbar">
          <el-button type="primary" @click="openCreateAccount">{{ t('backup.addAccount') }}</el-button>
        </div>
        <el-table :data="accounts" v-loading="accountLoading">
          <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
          <el-table-column prop="type" :label="t('backup.type')" width="120" />
          <el-table-column prop="backupPath" :label="t('backup.path')" min-width="200" show-overflow-tooltip />
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
            <el-option label="Website" value="website" />
            <el-option label="Database" value="database" />
            <el-option label="Directory" value="directory" />
          </el-select>
        </div>
        <el-table :data="records" v-loading="recordLoading">
          <el-table-column prop="type" :label="t('backup.type')" width="100" />
          <el-table-column prop="name" :label="t('commons.name')" min-width="140" />
          <el-table-column prop="fileName" :label="t('backup.fileName')" min-width="240" show-overflow-tooltip />
          <el-table-column prop="status" :label="t('backup.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'Success' ? 'success' : 'danger'" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="createdAt" :label="t('backup.time')" width="180">
            <template #default="{ row }">{{ new Date(row.createdAt).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column :label="t('commons.actions')" width="100">
            <template #default="{ row }">
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
    <el-drawer v-model="accountDrawer" :title="editAccountMode ? t('commons.edit') : t('backup.addAccount')" size="480px" destroy-on-close>
      <el-form ref="accountFormRef" :model="accountForm" :rules="accountRules" label-width="100px">
        <el-form-item :label="t('commons.name')" prop="name"><el-input v-model="accountForm.name" /></el-form-item>
        <el-form-item :label="t('backup.type')" prop="type">
          <el-select v-model="accountForm.type" :disabled="editAccountMode" style="width:100%">
            <el-option label="Local" value="local" /><el-option label="S3" value="s3" /><el-option label="SFTP" value="sftp" /><el-option label="WebDAV" value="webdav" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="accountForm.type === 's3'" label="Bucket"><el-input v-model="accountForm.bucket" /></el-form-item>
        <el-form-item v-if="accountForm.type !== 'local'" :label="t('backup.accessKey')"><el-input v-model="accountForm.accessKey" /></el-form-item>
        <el-form-item v-if="accountForm.type !== 'local'" :label="t('backup.credential')"><el-input v-model="accountForm.credential" type="password" show-password /></el-form-item>
        <el-form-item :label="t('backup.path')"><el-input v-model="accountForm.backupPath" placeholder="/opt/xpanel/backup" /></el-form-item>
        <el-form-item v-if="['s3','sftp','webdav'].includes(accountForm.type)" label="Endpoint"><el-input v-model="endpointField" placeholder="https://s3.example.com" /></el-form-item>
        <el-form-item v-if="accountForm.type === 's3'" label="Region"><el-input v-model="regionField" placeholder="us-east-1" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAccount">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Backup dialog -->
    <el-dialog v-model="backupDialog" :title="t('backup.createBackup')" width="460px" destroy-on-close>
      <el-form ref="backupFormRef" :model="backupForm" :rules="backupRules" label-width="100px">
        <el-form-item :label="t('backup.type')" prop="type">
          <el-select v-model="backupForm.type" style="width:100%">
            <el-option label="Website" value="website" /><el-option label="Database" value="database" /><el-option label="Directory" value="directory" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('commons.name')" prop="name"><el-input v-model="backupForm.name" /></el-form-item>
        <el-form-item :label="t('backup.account')" prop="accountID">
          <el-select v-model="backupForm.accountID" style="width:100%">
            <el-option v-for="a in accounts" :key="a.id" :label="a.name + ' (' + a.type + ')'" :value="a.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="backupForm.type === 'database'" :label="t('backup.dbType')">
          <el-select v-model="backupForm.dbType" style="width:100%">
            <el-option label="MySQL" value="mysql" /><el-option label="PostgreSQL" value="postgresql" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="backupForm.type === 'directory'" :label="t('backup.sourceDir')"><el-input v-model="backupForm.sourceDir" /></el-form-item>
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
import { useI18n } from 'vue-i18n'
import {
  listBackupAccounts, createBackupAccount, updateBackupAccount, deleteBackupAccount,
  createBackup, searchBackupRecords, deleteBackupRecord,
} from '@/api/modules/backup'

const { t } = useI18n()
const activeTab = ref('accounts')

const accountLoading = ref(false)
const accounts = ref<any[]>([])
const accountDrawer = ref(false)
const editAccountMode = ref(false)
const submitting = ref(false)
const accountFormRef = ref<FormInstance>()
const defaultAccountForm = () => ({ id: 0, name: '', type: 'local', bucket: '', accessKey: '', credential: '', backupPath: '', vars: '' })
const accountForm = reactive(defaultAccountForm())
const accountRules: FormRules = { name: [{ required: true, trigger: 'blur' }], type: [{ required: true }] }
const endpointField = ref('')
const regionField = ref('')

const recordLoading = ref(false)
const records = ref<any[]>([])
const recordType = ref('')
const recordPager = reactive({ page: 1, pageSize: 20, total: 0 })

const backupDialog = ref(false)
const backupFormRef = ref<FormInstance>()
const backupForm = reactive({ type: 'website', name: '', accountID: 0 as number, dbType: 'mysql', sourceDir: '' })
const backupRules: FormRules = { type: [{ required: true }], name: [{ required: true, trigger: 'blur' }], accountID: [{ required: true }] }

const loadAccounts = async () => {
  accountLoading.value = true
  try {
    const res: any = await listBackupAccounts()
    accounts.value = res.data || []
  } finally { accountLoading.value = false }
}

const openCreateAccount = () => {
  Object.assign(accountForm, defaultAccountForm())
  endpointField.value = ''
  regionField.value = ''
  editAccountMode.value = false
  accountDrawer.value = true
}

const openEditAccount = (row: any) => {
  Object.assign(accountForm, { ...row, credential: '' })
  try {
    const v = JSON.parse(row.vars || '{}')
    endpointField.value = v.endpoint || ''
    regionField.value = v.region || ''
  } catch { endpointField.value = ''; regionField.value = '' }
  editAccountMode.value = true
  accountDrawer.value = true
}

const submitAccount = async () => {
  if (!accountFormRef.value) return
  await accountFormRef.value.validate()
  submitting.value = true
  try {
    accountForm.vars = JSON.stringify({ endpoint: endpointField.value, region: regionField.value })
    if (editAccountMode.value) await updateBackupAccount(accountForm)
    else await createBackupAccount(accountForm)
    ElMessage.success(t('commons.success'))
    accountDrawer.value = false
    await loadAccounts()
  } finally { submitting.value = false }
}

const handleDeleteAccount = async (row: any) => {
  await ElMessageBox.confirm(t('backup.deleteAccountConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteBackupAccount({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadAccounts()
}

const loadRecords = async () => {
  recordLoading.value = true
  try {
    const res: any = await searchBackupRecords({ page: recordPager.page, pageSize: recordPager.pageSize, type: recordType.value })
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

const handleDeleteRecord = async (row: any) => {
  await ElMessageBox.confirm(t('backup.deleteRecordConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteBackupRecord({ id: row.id })
  ElMessage.success(t('commons.success'))
  await loadRecords()
}

watch(activeTab, (tab) => {
  if (tab === 'records') loadRecords()
})

onMounted(() => loadAccounts())
</script>

<style scoped>
.app-toolbar { display: flex; align-items: center; margin-bottom: 16px; }
.app-pagination { display: flex; justify-content: flex-end; margin-top: 16px; }
</style>
