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
      <el-tab-pane label="公钥管理" name="keys">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>authorized_keys</span>
              <el-button type="primary" size="small" @click="showAddKeyDialog = true">添加公钥</el-button>
            </div>
          </template>
          <el-table :data="authorizedKeys" v-loading="keysLoading" size="small">
            <el-table-column prop="keyType" label="类型" width="120" />
            <el-table-column prop="name" label="备注" min-width="200">
              <template #default="{ row }">{{ row.name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="fingerprint" label="指纹" width="180">
              <template #default="{ row }">
                <code class="fingerprint-text">{{ row.fingerprint }}...</code>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="100" fixed="right">
              <template #default="{ row }">
                <el-button link type="danger" size="small" @click="handleDeleteKey(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!keysLoading && authorizedKeys.length === 0" description="暂无公钥" />
        </el-card>

        <el-dialog v-model="showAddKeyDialog" title="添加 SSH 公钥" width="560px" destroy-on-close>
          <el-form label-width="80px">
            <el-form-item label="公钥内容">
              <el-input v-model="newKeyContent" type="textarea" :rows="5" placeholder="粘贴 SSH 公钥，如 ssh-rsa AAAA... user@host" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="newKeyName" placeholder="可选，用于标识此公钥" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="showAddKeyDialog = false">取消</el-button>
            <el-button type="primary" :loading="addingKey" @click="handleAddKey">确认添加</el-button>
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
            <el-option label="全部" value="all" />
            <el-option :label="$t('sshManage.success')" value="success" />
            <el-option :label="$t('sshManage.failed')" value="failed" />
          </el-select>
          <el-button size="small" :icon="Refresh" @click="loadSSHLog">{{ $t('commons.refresh') }}</el-button>
        </div>
        <el-table :data="sshLogs" size="small" v-loading="logLoading" max-height="500">
          <el-table-column prop="date" :label="$t('log.time')" width="180" />
          <el-table-column prop="status" :label="$t('log.status')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">{{ row.status === 'success' ? '成功' : '失败' }}</el-tag>
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
  if (!newKeyContent.value.trim()) {
    ElMessage.warning('请输入公钥内容')
    return
  }
  addingKey.value = true
  try {
    await addAuthorizedKey({ key: newKeyContent.value, name: newKeyName.value })
    ElMessage.success('公钥添加成功')
    showAddKeyDialog.value = false
    newKeyContent.value = ''
    newKeyName.value = ''
    loadAuthorizedKeys()
  } catch { /* handled */ }
  finally { addingKey.value = false }
}

const handleDeleteKey = async (row: AuthorizedKey) => {
  await ElMessageBox.confirm('确认删除此公钥？', t('commons.tip'), { type: 'warning' })
  try {
    await deleteAuthorizedKey(row.fingerprint)
    ElMessage.success(t('commons.success'))
    loadAuthorizedKeys()
  } catch { /* handled */ }
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
  if (!content.trim()) { ElMessage.warning('配置内容不能为空'); return }
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
</style>
