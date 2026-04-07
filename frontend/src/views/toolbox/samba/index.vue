<template>
  <div class="toolbox-samba-page">
    <div class="page-header">
      <h3>{{ $t('toolbox.sambaTitle') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadAll" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <!-- Not installed -->
    <template v-if="!status.isInstalled">
      <el-card shadow="never" class="install-card">
        <el-empty :description="$t('toolbox.sambaNotInstalled')">
          <template #image>
            <el-icon :size="64" color="var(--xp-text-muted)"><Share /></el-icon>
          </template>
          <el-button type="primary" @click="handleInstall" :loading="installLoading">
            {{ $t('toolbox.install') }}
          </el-button>
        </el-empty>
      </el-card>
    </template>

    <!-- Installed -->
    <template v-if="status.isInstalled">
      <!-- Status bar -->
      <el-row :gutter="16" class="status-row">
        <el-col :span="4">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('commons.status') }}</div>
            <div class="stat-value">
              <el-tag :type="status.isRunning ? 'success' : 'danger'" effect="dark" round>
                {{ status.isRunning ? $t('toolbox.running') : $t('toolbox.stopped') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="5">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('toolbox.version') }}</div>
            <div class="stat-value mono">{{ status.version || '-' }}</div>
          </el-card>
        </el-col>
        <el-col :span="4">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('toolbox.autoStart') }}</div>
            <div class="stat-value">
              <el-tag :type="status.autoStart ? 'success' : 'info'" effect="dark" round>
                {{ status.autoStart ? $t('commons.enabled') : $t('commons.disabled') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :span="11">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('commons.operate') }}</div>
            <div class="stat-value operate-buttons">
              <el-button type="success" size="small" :disabled="status.isRunning" @click="handleOperate('start')" :loading="opLoading === 'start'">
                <el-icon><VideoPlay /></el-icon>
              </el-button>
              <el-button type="danger" size="small" :disabled="!status.isRunning" @click="handleOperate('stop')" :loading="opLoading === 'stop'">
                <el-icon><VideoPause /></el-icon>
              </el-button>
              <el-button type="primary" size="small" :disabled="!status.isRunning" @click="handleOperate('restart')" :loading="opLoading === 'restart'">
                <el-icon><RefreshRight /></el-icon>
              </el-button>
              <el-divider direction="vertical" />
              <el-button size="small" :type="status.autoStart ? 'warning' : 'success'" plain @click="handleOperate(status.autoStart ? 'disable' : 'enable')">
                {{ status.autoStart ? $t('toolbox.disableAutoStart') : $t('toolbox.enableAutoStart') }}
              </el-button>
              <el-button type="danger" size="small" plain @click="handleUninstall">
                {{ $t('toolbox.uninstall') }}
              </el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- Tabs -->
      <el-card shadow="never" style="margin-top: 16px;">
        <el-tabs v-model="activeTab">
          <!-- Shares Tab -->
          <el-tab-pane :label="$t('toolbox.sambaShares')" name="shares">
            <div class="tab-toolbar">
              <el-button type="primary" size="small" @click="openShareDialog()">
                {{ $t('commons.create') }}
              </el-button>
            </div>
            <el-table :data="shares" v-loading="sharesLoading" stripe>
              <el-table-column prop="name" :label="$t('commons.name')" width="150" />
              <el-table-column prop="path" :label="$t('toolbox.path')" min-width="200" />
              <el-table-column prop="comment" :label="$t('commons.description')" min-width="150" />
              <el-table-column :label="$t('toolbox.writable')" width="80" align="center">
                <template #default="{ row }">
                  <el-tag :type="row.writable ? 'success' : 'info'" size="small">
                    {{ row.writable ? $t('toolbox.yes') : $t('toolbox.no') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column :label="$t('toolbox.guestOK')" width="80" align="center">
                <template #default="{ row }">
                  <el-tag :type="row.guestOK ? 'warning' : 'info'" size="small">
                    {{ row.guestOK ? $t('toolbox.yes') : $t('toolbox.no') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="validUsers" :label="$t('toolbox.validUsers')" width="150" />
              <el-table-column :label="$t('commons.actions')" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="openShareDialog(row)">
                    {{ $t('commons.edit') }}
                  </el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteShare(row.name)">
                    {{ $t('commons.delete') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- Users Tab -->
          <el-tab-pane :label="$t('toolbox.sambaUsers')" name="users">
            <div class="tab-toolbar">
              <el-button type="primary" size="small" @click="openUserDialog()">
                {{ $t('commons.create') }}
              </el-button>
            </div>
            <el-table :data="users" v-loading="usersLoading" stripe>
              <el-table-column prop="username" :label="$t('toolbox.username')" width="200" />
              <el-table-column prop="flags" :label="$t('toolbox.accountFlags')" min-width="200" />
              <el-table-column :label="$t('commons.actions')" width="250" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="openPasswordDialog(row.username)">
                    {{ $t('toolbox.changePassword') }}
                  </el-button>
                  <el-button link :type="row.flags?.includes('D') ? 'success' : 'warning'" size="small" @click="handleToggleUser(row)">
                    {{ row.flags?.includes('D') ? $t('commons.enable') : $t('toolbox.disable') }}
                  </el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteUser(row.username)">
                    {{ $t('commons.delete') }}
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- Connections Tab -->
          <el-tab-pane :label="$t('toolbox.connections')" name="connections">
            <div class="tab-toolbar">
              <el-button size="small" :icon="Refresh" @click="loadConnections" :loading="connLoading">
                {{ $t('commons.refresh') }}
              </el-button>
            </div>
            <h4 style="margin: 8px 0;">{{ $t('toolbox.activeProcesses') }}</h4>
            <el-table :data="connections.processes" v-loading="connLoading" stripe size="small">
              <el-table-column prop="pid" label="PID" width="80" />
              <el-table-column prop="username" :label="$t('toolbox.username')" width="120" />
              <el-table-column prop="group" :label="$t('toolbox.group')" width="120" />
              <el-table-column prop="machine" :label="$t('toolbox.machine')" min-width="150" />
              <el-table-column prop="protocol" :label="$t('toolbox.protocol')" width="100" />
              <el-table-column prop="encryption" :label="$t('toolbox.encryption')" width="120" />
            </el-table>
            <h4 style="margin: 16px 0 8px;">{{ $t('toolbox.activeShares') }}</h4>
            <el-table :data="connections.shares" v-loading="connLoading" stripe size="small">
              <el-table-column prop="service" :label="$t('toolbox.shareName')" width="150" />
              <el-table-column prop="pid" label="PID" width="80" />
              <el-table-column prop="machine" :label="$t('toolbox.machine')" min-width="150" />
              <el-table-column prop="connectedAt" :label="$t('toolbox.connectedAt')" min-width="180" />
            </el-table>
          </el-tab-pane>

          <!-- Global Config Tab -->
          <el-tab-pane :label="$t('toolbox.globalConfig')" name="config">
            <el-form :model="globalConfig" label-width="160px" style="max-width: 600px; margin-top: 8px;" v-loading="configLoading">
              <el-form-item label="Workgroup">
                <el-input v-model="globalConfig.workgroup" placeholder="WORKGROUP" />
              </el-form-item>
              <el-form-item :label="$t('toolbox.serverName')">
                <el-input v-model="globalConfig.serverName" placeholder="Samba Server" />
              </el-form-item>
              <el-form-item label="Security">
                <el-select v-model="globalConfig.security" placeholder="user" style="width: 100%">
                  <el-option label="user" value="user" />
                  <el-option label="share" value="share" />
                </el-select>
              </el-form-item>
              <el-form-item label="Map to Guest">
                <el-select v-model="globalConfig.mapToGuest" style="width: 100%">
                  <el-option label="Never" value="Never" />
                  <el-option label="Bad User" value="Bad User" />
                  <el-option label="Bad Password" value="Bad Password" />
                </el-select>
              </el-form-item>
              <el-form-item :label="$t('toolbox.interfaces')">
                <el-input v-model="globalConfig.interfaces" :placeholder="$t('toolbox.interfacesPlaceholder')" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="handleSaveConfig" :loading="configSaving">
                  {{ $t('commons.save') }}
                </el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </template>

    <!-- Share Dialog -->
    <el-dialog v-model="shareDialogOpen" :title="shareForm.origName ? $t('toolbox.editShare') : $t('toolbox.createShare')" width="560px" destroy-on-close>
      <el-form :model="shareForm" :rules="shareRules" ref="shareFormRef" label-width="120px">
        <el-form-item :label="$t('commons.name')" prop="name">
          <el-input v-model="shareForm.name" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.path')" prop="path">
          <el-input v-model="shareForm.path" placeholder="/data/share" />
        </el-form-item>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="shareForm.comment" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.writable')">
          <el-switch v-model="shareForm.writable" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.guestOK')">
          <el-switch v-model="shareForm.guestOK" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.validUsers')">
          <el-input v-model="shareForm.validUsers" :placeholder="$t('toolbox.validUsersPlaceholder')" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.createDir')" v-if="!shareForm.origName">
          <el-switch v-model="shareForm.createDir" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="shareDialogOpen = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmitShare" :loading="shareSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- User Dialog -->
    <el-dialog v-model="userDialogOpen" :title="$t('toolbox.createUser')" width="460px" destroy-on-close>
      <el-form :model="userForm" :rules="userRules" ref="userFormRef" label-width="100px">
        <el-form-item :label="$t('toolbox.username')" prop="username">
          <el-input v-model="userForm.username" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.password')" prop="password">
          <el-input v-model="userForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialogOpen = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmitUser" :loading="userSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Password Dialog -->
    <el-dialog v-model="passwordDialogOpen" :title="$t('toolbox.changePassword')" width="460px" destroy-on-close>
      <el-form :model="passwordForm" :rules="passwordRules" ref="passwordFormRef" label-width="100px">
        <el-form-item :label="$t('toolbox.username')">
          <el-input v-model="passwordForm.username" disabled />
        </el-form-item>
        <el-form-item :label="$t('toolbox.newPassword')" prop="password">
          <el-input v-model="passwordForm.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialogOpen = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmitPassword" :loading="passwordSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Share, VideoPlay, VideoPause, RefreshRight } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import {
  getSambaStatus, installSamba, uninstallSamba, operateSamba,
  listSambaShares, createSambaShare, updateSambaShare, deleteSambaShare,
  listSambaUsers, createSambaUser, deleteSambaUser, updateSambaPassword, toggleSambaUser,
  getSambaGlobalConfig, updateSambaGlobalConfig, getSambaConnections,
} from '@/api/modules/toolbox'

const { t } = useI18n()
const loading = ref(false)
const installLoading = ref(false)
const opLoading = ref('')
const activeTab = ref('shares')

const status = ref({ isInstalled: false, isRunning: false, version: '', autoStart: false })

// Shares
const shares = ref<any[]>([])
const sharesLoading = ref(false)
const shareDialogOpen = ref(false)
const shareSubmitting = ref(false)
const shareFormRef = ref<FormInstance>()
const shareForm = reactive({
  origName: '',
  name: '',
  path: '',
  comment: '',
  writable: true,
  guestOK: false,
  validUsers: '',
  createDir: true,
})
const shareRules = reactive<FormRules>({
  name: [{ required: true, message: t('toolbox.nameRequired'), trigger: 'blur' }],
  path: [{ required: true, message: t('toolbox.pathRequired'), trigger: 'blur' }],
})

// Users
const users = ref<any[]>([])
const usersLoading = ref(false)
const userDialogOpen = ref(false)
const userSubmitting = ref(false)
const userFormRef = ref<FormInstance>()
const userForm = reactive({ username: '', password: '' })
const userRules = reactive<FormRules>({
  username: [{ required: true, message: t('toolbox.usernameRequired'), trigger: 'blur' }],
  password: [{ required: true, message: t('toolbox.passwordRequired'), trigger: 'blur' }],
})

// Password
const passwordDialogOpen = ref(false)
const passwordSubmitting = ref(false)
const passwordFormRef = ref<FormInstance>()
const passwordForm = reactive({ username: '', password: '' })
const passwordRules = reactive<FormRules>({
  password: [{ required: true, message: t('toolbox.passwordRequired'), trigger: 'blur' }],
})

// Connections
const connections = ref<{ processes: any[]; shares: any[] }>({ processes: [], shares: [] })
const connLoading = ref(false)

// Global Config
const globalConfig = reactive({
  workgroup: '', serverName: '', security: '', mapToGuest: '', logLevel: '', maxLogSize: '', interfaces: '',
})
const configLoading = ref(false)
const configSaving = ref(false)

const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getSambaStatus()
    if (res.data) status.value = res.data
  } finally {
    loading.value = false
  }
}

const loadShares = async () => {
  sharesLoading.value = true
  try {
    const res = await listSambaShares()
    shares.value = res.data || []
  } finally {
    sharesLoading.value = false
  }
}

const loadUsers = async () => {
  usersLoading.value = true
  try {
    const res = await listSambaUsers()
    users.value = res.data || []
  } finally {
    usersLoading.value = false
  }
}

const loadConnections = async () => {
  connLoading.value = true
  try {
    const res = await getSambaConnections()
    if (res.data) connections.value = res.data
  } finally {
    connLoading.value = false
  }
}

const loadConfig = async () => {
  configLoading.value = true
  try {
    const res = await getSambaGlobalConfig()
    if (res.data) Object.assign(globalConfig, res.data)
  } finally {
    configLoading.value = false
  }
}

const loadAll = async () => {
  await loadStatus()
  if (status.value.isInstalled) {
    loadShares()
    loadUsers()
    loadConnections()
    loadConfig()
  }
}

const handleInstall = async () => {
  installLoading.value = true
  try {
    await installSamba()
    ElMessage.success(t('commons.operationSuccess'))
    await loadAll()
  } finally {
    installLoading.value = false
  }
}

const handleUninstall = async () => {
  await ElMessageBox.confirm(t('toolbox.uninstallConfirm'), t('commons.warning'), { type: 'warning' })
  try {
    await uninstallSamba()
    ElMessage.success(t('commons.operationSuccess'))
    await loadAll()
  } catch { /* */ }
}

const handleOperate = async (op: string) => {
  opLoading.value = op
  try {
    await operateSamba(op)
    ElMessage.success(t('commons.operationSuccess'))
    setTimeout(() => loadStatus(), 1000)
  } finally {
    opLoading.value = ''
  }
}

// Share CRUD
const openShareDialog = (row?: any) => {
  if (row) {
    Object.assign(shareForm, { ...row, origName: row.name, createDir: false })
  } else {
    Object.assign(shareForm, { origName: '', name: '', path: '', comment: '', writable: true, guestOK: false, validUsers: '', createDir: true })
  }
  shareDialogOpen.value = true
}

const handleSubmitShare = async () => {
  await shareFormRef.value?.validate()
  shareSubmitting.value = true
  try {
    if (shareForm.origName) {
      await updateSambaShare(shareForm)
    } else {
      await createSambaShare(shareForm)
    }
    ElMessage.success(t('commons.operationSuccess'))
    shareDialogOpen.value = false
    loadShares()
  } finally {
    shareSubmitting.value = false
  }
}

const handleDeleteShare = async (name: string) => {
  await ElMessageBox.confirm(t('toolbox.deleteShareConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteSambaShare(name)
  ElMessage.success(t('commons.deleteSuccess'))
  loadShares()
}

// User CRUD
const openUserDialog = () => {
  userForm.username = ''
  userForm.password = ''
  userDialogOpen.value = true
}

const handleSubmitUser = async () => {
  await userFormRef.value?.validate()
  userSubmitting.value = true
  try {
    await createSambaUser(userForm)
    ElMessage.success(t('commons.operationSuccess'))
    userDialogOpen.value = false
    loadUsers()
  } finally {
    userSubmitting.value = false
  }
}

const handleDeleteUser = async (username: string) => {
  await ElMessageBox.confirm(t('toolbox.deleteUserConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteSambaUser(username)
  ElMessage.success(t('commons.deleteSuccess'))
  loadUsers()
}

const handleToggleUser = async (row: any) => {
  const enabled = row.flags?.includes('D')
  await toggleSambaUser(row.username, enabled)
  ElMessage.success(t('commons.operationSuccess'))
  loadUsers()
}

// Password
const openPasswordDialog = (username: string) => {
  passwordForm.username = username
  passwordForm.password = ''
  passwordDialogOpen.value = true
}

const handleSubmitPassword = async () => {
  await passwordFormRef.value?.validate()
  passwordSubmitting.value = true
  try {
    await updateSambaPassword(passwordForm)
    ElMessage.success(t('commons.operationSuccess'))
    passwordDialogOpen.value = false
  } finally {
    passwordSubmitting.value = false
  }
}

// Config
const handleSaveConfig = async () => {
  configSaving.value = true
  try {
    await updateSambaGlobalConfig(globalConfig)
    ElMessage.success(t('commons.saveSuccess'))
  } finally {
    configSaving.value = false
  }
}

onMounted(() => loadAll())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.install-card {
  text-align: center;
}
.status-row {
  .stat-card {
    text-align: center;
    .stat-title {
      font-size: 13px;
      color: var(--xp-text-muted);
      margin-bottom: 10px;
    }
    .stat-value {
      font-size: 14px;
      font-weight: 600;
    }
    .mono { font-family: monospace; }
  }
}
.operate-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-wrap: wrap;
  gap: 4px;
}
.tab-toolbar {
  margin-bottom: 12px;
  display: flex;
  justify-content: flex-end;
}
</style>
