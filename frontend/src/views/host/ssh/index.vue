<template>
  <div class="ssh-page">
    <div class="page-header">
      <h3>{{ $t('sshManage.title') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadSSH" :loading="loading">{{ $t('commons.refresh') }}</el-button>
    </div>

    <el-tabs v-model="activeTab">
      <!-- SSH 配置 -->
      <el-tab-pane :label="$t('sshManage.title')" name="config">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>{{ $t('sshManage.status') }}</span>
              <div class="header-actions">
                <el-tag :type="sshInfo.isActive ? 'success' : 'danger'" size="small">
                  {{ sshInfo.isExist ? (sshInfo.isActive ? $t('sshManage.active') : $t('sshManage.inactive')) : $t('sshManage.notInstalled') }}
                </el-tag>
                <template v-if="sshInfo.isExist">
                  <el-button size="small" type="success" plain @click="handleOperate('start')" :disabled="sshInfo.isActive">{{ $t('sshManage.start') }}</el-button>
                  <el-button size="small" type="danger" plain @click="handleOperate('stop')" :disabled="!sshInfo.isActive">{{ $t('sshManage.stop') }}</el-button>
                  <el-button size="small" type="warning" plain @click="handleOperate('restart')" :disabled="!sshInfo.isActive">{{ $t('sshManage.restart') }}</el-button>
                </template>
              </div>
            </div>
          </template>

          <el-form v-if="sshInfo.isExist" label-width="140px" class="ssh-form">
            <el-form-item :label="$t('sshManage.port')">
              <el-input v-model="sshInfo.port" style="width: 200px" />
              <el-button type="primary" link @click="handleUpdate('Port', sshInfo.port)" class="ml-12">{{ $t('commons.save') }}</el-button>
            </el-form-item>
            <el-form-item :label="$t('sshManage.listenAddr')">
              <el-input v-model="sshInfo.listenAddress" style="width: 200px" />
              <el-button type="primary" link @click="handleUpdate('ListenAddress', sshInfo.listenAddress)" class="ml-12">{{ $t('commons.save') }}</el-button>
            </el-form-item>
            <el-form-item :label="$t('sshManage.permitRoot')">
              <el-select v-model="sshInfo.permitRootLogin" style="width: 200px" @change="handleUpdate('PermitRootLogin', sshInfo.permitRootLogin)">
                <el-option label="yes" value="yes" />
                <el-option label="no" value="no" />
                <el-option label="prohibit-password" value="prohibit-password" />
                <el-option label="without-password" value="without-password" />
              </el-select>
            </el-form-item>
            <el-form-item :label="$t('sshManage.passwordAuth')">
              <el-switch :model-value="sshInfo.passwordAuthentication === 'yes'" @change="(v: boolean) => handleUpdate('PasswordAuthentication', v ? 'yes' : 'no')" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.pubkeyAuth')">
              <el-switch :model-value="sshInfo.pubkeyAuthentication === 'yes'" @change="(v: boolean) => handleUpdate('PubkeyAuthentication', v ? 'yes' : 'no')" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.useDNS')">
              <el-switch :model-value="sshInfo.useDNS === 'yes'" @change="(v: boolean) => handleUpdate('UseDNS', v ? 'yes' : 'no')" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.autoStart')">
              <el-switch v-model="sshInfo.autoStart" @change="handleOperate(sshInfo.autoStart ? 'enable' : 'disable')" />
            </el-form-item>
          </el-form>
          <el-empty v-else :description="sshInfo.message || $t('sshManage.notInstalled')" />
        </el-card>
      </el-tab-pane>

      <!-- 公钥管理 -->
      <el-tab-pane label="authorized_keys" name="keys">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>authorized_keys</span>
              <el-button type="primary" size="small" @click="openAddKeyDialog">{{ $t('commons.create') }}</el-button>
            </div>
          </template>
          <el-table :data="authorizedKeys" v-loading="keysLoading" size="small">
            <el-table-column prop="keyType" :label="$t('sshManage.keyType')" width="120" />
            <el-table-column prop="name" :label="$t('commons.name')" min-width="200">
              <template #default="{ row }">{{ row.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="fingerprint" :label="$t('sshManage.keyFingerprint')" width="180">
              <template #default="{ row }">
                <code class="fingerprint-text">{{ row.fingerprint }}...</code>
              </template>
            </el-table-column>
            <el-table-column :label="$t('commons.actions')" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" size="small" @click="handleDeleteKey(row)">{{ $t('commons.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <el-dialog v-model="showAddKeyDialog" :title="$t('sshManage.addAuthorizedKey')" width="600px" destroy-on-close>
          <el-radio-group v-model="addKeyMode" style="margin-bottom: 16px; width: 100%;">
            <el-radio-button value="paste">{{ $t('sshManage.pasteKey') }}</el-radio-button>
            <el-radio-button value="upload">{{ $t('sshManage.uploadFile') }}</el-radio-button>
            <el-radio-button value="select" :disabled="sshKeys.length === 0">{{ $t('sshManage.selectExistingKey') }}</el-radio-button>
          </el-radio-group>

          <el-form label-width="80px">
            <!-- 粘贴公钥 -->
            <template v-if="addKeyMode === 'paste'">
              <el-form-item :label="$t('sshManage.publicKey')">
                <el-input v-model="newKeyContent" type="textarea" :rows="5" placeholder="ssh-rsa AAAA... user@host" />
              </el-form-item>
            </template>

            <!-- 上传公钥文件 -->
            <template v-if="addKeyMode === 'upload'">
              <el-form-item :label="$t('sshManage.pubKeyFile')">
                <div class="upload-area">
                  <el-upload
                    :auto-upload="false"
                    :show-file-list="false"
                    :on-change="handleFileSelect"
                    accept=".pub,.pem,.txt"
                  >
                    <el-button size="small" type="primary" plain>{{ $t('sshManage.selectFile') }}</el-button>
                  </el-upload>
                  <span v-if="uploadFileName" class="upload-filename">{{ uploadFileName }}</span>
                </div>
              </el-form-item>
              <el-form-item v-if="newKeyContent" :label="$t('sshManage.preview')">
                <el-input :model-value="newKeyContent" type="textarea" :rows="3" readonly />
              </el-form-item>
            </template>

            <!-- 选择已有密钥 -->
            <template v-if="addKeyMode === 'select'">
              <el-form-item :label="$t('sshManage.selectKey')">
                <el-select v-model="selectedKeyName" style="width: 100%" :placeholder="$t('sshManage.selectExistingKey')" @change="handleSelectKey">
                  <el-option v-for="key in sshKeys" :key="key.name" :label="`${key.name} (${key.keyType} ${key.bits || ''})`" :value="key.name" />
                </el-select>
              </el-form-item>
              <el-form-item v-if="newKeyContent" :label="$t('sshManage.preview')">
                <el-input :model-value="newKeyContent" type="textarea" :rows="3" readonly />
              </el-form-item>
            </template>

            <el-form-item :label="$t('commons.name')">
              <el-input v-model="newKeyName" :placeholder="$t('sshManage.keyCommentHint')" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="showAddKeyDialog = false">{{ $t('commons.cancel') }}</el-button>
            <el-button type="primary" :loading="addingKey" @click="handleAddKey" :disabled="!newKeyContent.trim()">{{ $t('commons.confirm') }}</el-button>
          </template>
        </el-dialog>
      </el-tab-pane>

      <!-- 私钥管理 -->
      <el-tab-pane :label="$t('sshManage.keyManage')" name="privateKeys">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>{{ $t('sshManage.keyManage') }}</span>
              <div style="display: flex; gap: 8px;">
                <el-button type="primary" size="small" @click="showGenerateDialog = true">{{ $t('sshManage.generateKey') }}</el-button>
                <el-button size="small" @click="showImportDialog = true">{{ $t('sshManage.importKey') }}</el-button>
              </div>
            </div>
          </template>
          <el-table :data="sshKeys" v-loading="sshKeysLoading" size="small">
            <el-table-column prop="name" :label="$t('sshManage.keyName')" min-width="140" />
            <el-table-column prop="keyType" :label="$t('sshManage.keyType')" width="120" />
            <el-table-column prop="bits" :label="$t('sshManage.keyBits')" width="80">
              <template #default="{ row }">{{ row.bits || '-' }}</template>
            </el-table-column>
            <el-table-column prop="fingerprint" :label="$t('sshManage.keyFingerprint')" min-width="240">
              <template #default="{ row }">
                <code class="fingerprint-text">{{ row.fingerprint }}</code>
              </template>
            </el-table-column>
            <el-table-column :label="$t('commons.actions')" width="260" fixed="right">
              <template #default="{ row }">
                <el-button link type="success" size="small" @click="handleDeployKey(row)">{{ $t('sshManage.deployKey') }}</el-button>
                <el-button link type="primary" size="small" @click="handleCopyPubKey(row)">{{ $t('sshManage.copyPublicKey') }}</el-button>
                <el-button link type="warning" size="small" @click="handleViewPrivateKey(row)">{{ $t('sshManage.viewPrivateKey') }}</el-button>
                <el-button link type="danger" size="small" @click="handleDeleteSSHKey(row)">{{ $t('commons.delete') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <!-- 生成密钥对 -->
        <el-dialog v-model="showGenerateDialog" :title="$t('sshManage.generateKey')" width="480px" destroy-on-close>
          <el-form label-width="80px">
            <el-form-item :label="$t('sshManage.keyName')">
              <el-input v-model="genForm.name" placeholder="my-key" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.keyBits')">
              <el-select v-model="genForm.bits" style="width: 100%;">
                <el-option label="2048" :value="2048" />
                <el-option label="4096 (推荐)" :value="4096" />
              </el-select>
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="showGenerateDialog = false">{{ $t('commons.cancel') }}</el-button>
            <el-button type="primary" :loading="generating" @click="handleGenerate">{{ $t('commons.confirm') }}</el-button>
          </template>
        </el-dialog>

        <!-- 生成结果 -->
        <el-dialog v-model="showGenResult" :title="$t('sshManage.privateKeyContent')" width="600px">
          <el-alert type="warning" :closable="false" style="margin-bottom: 12px;">
            {{ $t('sshManage.privateKeyHint') }}
          </el-alert>
          <el-input :model-value="genResultKey" type="textarea" :rows="12" readonly />
          <template #footer>
            <el-button @click="handleCopyText(genResultKey)">{{ $t('commons.copy') }}</el-button>
            <el-button type="primary" @click="showGenResult = false">{{ $t('commons.confirm') }}</el-button>
          </template>
        </el-dialog>

        <!-- 导入私钥 -->
        <el-dialog v-model="showImportDialog" :title="$t('sshManage.importKey')" width="560px" destroy-on-close>
          <el-form label-width="80px">
            <el-form-item :label="$t('sshManage.keyName')">
              <el-input v-model="importForm.name" placeholder="my-imported-key" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.privateKey')">
              <el-input v-model="importForm.privateKey" type="textarea" :rows="8" :placeholder="$t('sshManage.importPrivateKey')" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="showImportDialog = false">{{ $t('commons.cancel') }}</el-button>
            <el-button type="primary" :loading="importing" @click="handleImport">{{ $t('commons.confirm') }}</el-button>
          </template>
        </el-dialog>

        <!-- 查看私钥 -->
        <el-dialog v-model="showPrivateKeyDialog" :title="$t('sshManage.privateKeyContent')" width="600px">
          <el-input :model-value="viewPrivateKey" type="textarea" :rows="12" readonly />
          <template #footer>
            <el-button @click="handleCopyText(viewPrivateKey)">{{ $t('commons.copy') }}</el-button>
            <el-button type="primary" @click="showPrivateKeyDialog = false">{{ $t('commons.confirm') }}</el-button>
          </template>
        </el-dialog>
      </el-tab-pane>

      <!-- sshd_config 编辑 -->
      <el-tab-pane :label="$t('sshManage.sshdConfig')" name="sshdConfig">
        <div class="sshd-editor-section">
          <div class="sshd-toolbar">
            <span class="sshd-file-label">/etc/ssh/sshd_config</span>
            <div class="sshd-actions">
              <el-button size="small" @click="loadSSHDConfig" :loading="sshdLoading">{{ $t('commons.refresh') }}</el-button>
              <el-button size="small" type="primary" @click="handleSaveSSHDConfig" :loading="sshdSaving">{{ $t('commons.save') }}</el-button>
            </div>
          </div>
          <div ref="sshdEditorRef" class="sshd-editor-container" />
          <div class="sshd-hint">{{ $t('sshManage.sshdConfigHint') }}</div>
        </div>
      </el-tab-pane>

      <!-- SSH 登录日志 -->
      <el-tab-pane :label="$t('sshManage.loginLog')" name="log">
        <div class="toolbar">
          <el-select v-model="logStatus" size="small" style="width: 120px" @change="loadSSHLog">
            <el-option :label="$t('commons.search')" value="all" />
            <el-option :label="$t('sshManage.success')" value="success" />
            <el-option :label="$t('sshManage.failed')" value="failed" />
          </el-select>
          <el-button size="small" :icon="Refresh" @click="loadSSHLog">{{ $t('commons.refresh') }}</el-button>
        </div>
        <el-table :data="sshLogs" size="small" v-loading="logLoading" max-height="500">
          <el-table-column prop="date" :label="$t('log.time')" width="180" />
          <el-table-column prop="status" :label="$t('log.status')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
                {{ row.status === 'success' ? $t('sshManage.success') : $t('sshManage.failed') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="user" :label="$t('process.user')" width="100" />
          <el-table-column prop="ip" :label="$t('log.ip')" width="150" />
          <el-table-column prop="port" :label="$t('sshManage.port')" width="80" />
        </el-table>
        <el-pagination v-if="logTotal > 0" class="mt-12" :current-page="logPage" :page-size="logPageSize" :total="logTotal" layout="total, prev, pager, next" @current-change="(p: number) => { logPage = p; loadSSHLog() }" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import {
  getSSHInfo, operateSSH, updateSSHConfig, searchSSHLog,
  getSSHDConfig, saveSSHDConfig,
  listAuthorizedKeys, addAuthorizedKey, deleteAuthorizedKey,
  listSSHKeys, getSSHPrivateKey, generateSSHKey, importSSHKey, deleteSSHKey,
} from '@/api/modules/ssh-manage'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import * as monaco from 'monaco-editor'
import type { SSHInfo, SSHLogEntry, AuthorizedKey } from '@/api/interface'

const { t } = useI18n()
const activeTab = ref('config')
const loading = ref(false)

const sshInfo = ref<SSHInfo>({} as SSHInfo)

const logLoading = ref(false)
const sshLogs = ref<SSHLogEntry[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logPageSize = ref(20)
const logStatus = ref('all')

// 公钥管理
const keysLoading = ref(false)
const authorizedKeys = ref<AuthorizedKey[]>([])
const showAddKeyDialog = ref(false)
const newKeyContent = ref('')
const newKeyName = ref('')
const addingKey = ref(false)
const addKeyMode = ref<'paste' | 'upload' | 'select'>('paste')
const uploadFileName = ref('')
const selectedKeyName = ref('')

// 私钥管理
const sshKeysLoading = ref(false)
const sshKeys = ref<any[]>([])
const showGenerateDialog = ref(false)
const showImportDialog = ref(false)
const showGenResult = ref(false)
const showPrivateKeyDialog = ref(false)
const genResultKey = ref('')
const viewPrivateKey = ref('')
const generating = ref(false)
const importing = ref(false)
const genForm = ref({ name: '', bits: 4096 })
const importForm = ref({ name: '', privateKey: '' })

const loadSSH = async () => {
  loading.value = true
  try {
    const res = await getSSHInfo()
    sshInfo.value = res.data || {}
  } catch { /* handled */ }
  finally { loading.value = false }
}

const handleOperate = async (op: string) => {
  try {
    await operateSSH(op)
    ElMessage.success(t('commons.success'))
    setTimeout(loadSSH, 1000)
  } catch { /* handled */ }
}

const handleUpdate = async (key: string, value: string) => {
  await ElMessageBox.confirm(t('sshManage.updateConfirm'), t('commons.tip'), { type: 'warning' })
  try {
    await updateSSHConfig(key, value)
    ElMessage.success(t('commons.success'))
    loadSSH()
  } catch { /* handled */ }
}

const loadSSHLog = async () => {
  logLoading.value = true
  try {
    const res = await searchSSHLog({ page: logPage.value, pageSize: logPageSize.value, status: logStatus.value })
    sshLogs.value = res.data?.items || []
    logTotal.value = res.data?.total || 0
  } catch { sshLogs.value = [] }
  finally { logLoading.value = false }
}

const openAddKeyDialog = () => {
  addKeyMode.value = 'paste'
  newKeyContent.value = ''
  newKeyName.value = ''
  uploadFileName.value = ''
  selectedKeyName.value = ''
  if (sshKeys.value.length === 0) loadSSHKeys()
  showAddKeyDialog.value = true
}

const handleFileSelect = (file: any) => {
  const reader = new FileReader()
  reader.onload = (e) => {
    newKeyContent.value = (e.target?.result as string || '').trim()
    uploadFileName.value = file.name
  }
  reader.readAsText(file.raw)
}

const handleSelectKey = () => {
  const key = sshKeys.value.find(k => k.name === selectedKeyName.value)
  if (key?.publicKey) {
    newKeyContent.value = key.publicKey
    if (!newKeyName.value) newKeyName.value = key.name
  }
}

const handleDeployKey = async (row: any) => {
  if (!row.publicKey) {
    ElMessage.warning(t('sshManage.noPublicKey'))
    return
  }
  await ElMessageBox.confirm(
    t('sshManage.deployKeyConfirm', { name: row.name }),
    t('commons.tip'),
    { type: 'info' }
  )
  try {
    await addAuthorizedKey({ key: row.publicKey, name: row.name })
    ElMessage.success(t('commons.success'))
    loadAuthorizedKeys()
  } catch { /* handled */ }
}

// 公钥管理
const loadAuthorizedKeys = async () => {
  keysLoading.value = true
  try {
    const res = await listAuthorizedKeys()
    authorizedKeys.value = res.data || []
  } catch { authorizedKeys.value = [] }
  finally { keysLoading.value = false }
}

const handleAddKey = async () => {
  if (!newKeyContent.value.trim()) return
  addingKey.value = true
  try {
    await addAuthorizedKey({ key: newKeyContent.value, name: newKeyName.value })
    ElMessage.success(t('commons.success'))
    showAddKeyDialog.value = false
    newKeyContent.value = ''
    newKeyName.value = ''
    loadAuthorizedKeys()
  } catch { /* handled */ }
  finally { addingKey.value = false }
}

const handleDeleteKey = async (row: AuthorizedKey) => {
  await ElMessageBox.confirm(t('commons.tip'), { type: 'warning' })
  try {
    await deleteAuthorizedKey(row.fingerprint)
    ElMessage.success(t('commons.success'))
    loadAuthorizedKeys()
  } catch { /* handled */ }
}

// 私钥管理
const loadSSHKeys = async () => {
  sshKeysLoading.value = true
  try {
    const res = await listSSHKeys()
    sshKeys.value = res.data || []
  } catch { sshKeys.value = [] }
  finally { sshKeysLoading.value = false }
}

const handleGenerate = async () => {
  if (!genForm.value.name.trim()) {
    ElMessage.warning(t('sshManage.keyNameRequired'))
    return
  }
  generating.value = true
  try {
    const res = await generateSSHKey(genForm.value)
    genResultKey.value = res.data?.privateKey || ''
    showGenerateDialog.value = false
    showGenResult.value = true
    genForm.value = { name: '', bits: 4096 }
    loadSSHKeys()
  } catch { /* handled */ }
  finally { generating.value = false }
}

const handleImport = async () => {
  if (!importForm.value.name.trim() || !importForm.value.privateKey.trim()) return
  importing.value = true
  try {
    await importSSHKey(importForm.value)
    ElMessage.success(t('commons.success'))
    showImportDialog.value = false
    importForm.value = { name: '', privateKey: '' }
    loadSSHKeys()
  } catch { /* handled */ }
  finally { importing.value = false }
}

const handleCopyPubKey = (row: any) => {
  navigator.clipboard.writeText(row.publicKey || '').then(() => {
    ElMessage.success(t('commons.copySuccess'))
  }).catch(() => ElMessage.error(t('commons.copyFailed')))
}

const handleViewPrivateKey = async (row: any) => {
  try {
    const res = await getSSHPrivateKey(row.name)
    viewPrivateKey.value = res.data || ''
    showPrivateKeyDialog.value = true
  } catch { /* handled */ }
}

const handleDeleteSSHKey = async (row: any) => {
  await ElMessageBox.confirm(
    t('sshManage.deleteKeyConfirm', { name: row.name }),
    t('commons.tip'),
    { type: 'warning' }
  )
  try {
    await deleteSSHKey(row.name)
    ElMessage.success(t('commons.success'))
    loadSSHKeys()
  } catch { /* handled */ }
}

const handleCopyText = (text: string) => {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(t('commons.copySuccess'))
  }).catch(() => ElMessage.error(t('commons.copyFailed')))
}

// sshd_config editor
const sshdEditorRef = ref<HTMLElement>()
const sshdLoading = ref(false)
const sshdSaving = ref(false)
let sshdEditor: monaco.editor.IStandaloneCodeEditor | null = null

const loadSSHDConfig = async () => {
  sshdLoading.value = true
  try {
    const res = await getSSHDConfig()
    const content = res.data || ''
    await nextTick()
    if (sshdEditor) {
      sshdEditor.setValue(content)
    } else if (sshdEditorRef.value) {
      sshdEditor = monaco.editor.create(sshdEditorRef.value, {
        value: content,
        language: 'plaintext',
        theme: 'vs-dark',
        fontSize: 13,
        fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', monospace",
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        lineNumbers: 'on',
        automaticLayout: true,
        tabSize: 4,
        wordWrap: 'on',
      })
    }
  } catch { /* handled */ }
  finally { sshdLoading.value = false }
}

const handleSaveSSHDConfig = async () => {
  if (!sshdEditor) return
  const content = sshdEditor.getValue()
  if (!content.trim()) return
  sshdSaving.value = true
  try {
    await saveSSHDConfig(content)
    ElMessage.success(t('commons.success'))
  } catch { /* handled */ }
  finally { sshdSaving.value = false }
}

watch(activeTab, (val) => {
  if (val === 'log' && sshLogs.value.length === 0) loadSSHLog()
  if (val === 'sshdConfig' && !sshdEditor) loadSSHDConfig()
  if (val === 'keys' && authorizedKeys.value.length === 0) loadAuthorizedKeys()
  if (val === 'privateKeys' && sshKeys.value.length === 0) loadSSHKeys()
})

onMounted(() => loadSSH())

onBeforeUnmount(() => {
  if (sshdEditor) { sshdEditor.dispose(); sshdEditor = null }
})
</script>

<style lang="scss" scoped>
.ssh-page { height: 100%; }

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  .header-actions { display: flex; align-items: center; gap: 8px; }
}

.ssh-form {
  max-width: 600px;
  .ml-12 { margin-left: 12px; }
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.mt-12 { margin-top: 12px; }

.sshd-editor-section {
  display: flex;
  flex-direction: column;
  height: 500px;
}

.sshd-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;

  .sshd-file-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--xp-accent);
    font-family: 'Fira Code', 'Consolas', monospace;
  }

  .sshd-actions {
    display: flex;
    gap: 8px;
  }
}

.sshd-editor-container {
  flex: 1;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--xp-border-light);
}

.sshd-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--xp-text-muted);
}

.fingerprint-text {
  font-size: 12px;
  font-family: 'JetBrains Mono', monospace;
  color: var(--xp-text-muted);
}

.upload-area {
  display: flex;
  align-items: center;
  gap: 12px;
}

.upload-filename {
  font-size: 13px;
  color: var(--xp-text-secondary);
  font-family: 'JetBrains Mono', monospace;
}
</style>
