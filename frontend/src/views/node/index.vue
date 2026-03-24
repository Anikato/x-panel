<template>
  <div>
    <div class="app-toolbar">
      <el-button type="primary" @click="openCreate">{{ t('node.addNode') }}</el-button>
    </div>
    <el-table :data="nodes" v-loading="loading">
      <el-table-column prop="name" :label="t('commons.name')" min-width="120" />
      <el-table-column :label="t('node.sshInfo')" min-width="200">
        <template #default="{ row }">
          <span>{{ row.sshUser }}@{{ row.sshHost }}:{{ row.sshPort || 22 }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('node.panelAddr')" min-width="180">
        <template #default="{ row }">{{ row.address || '-' }}</template>
      </el-table-column>
      <el-table-column :label="t('node.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">{{ row.status === 'online' ? t('node.online') : t('node.offline') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="os" label="OS" width="120" show-overflow-tooltip />
      <el-table-column prop="hostname" :label="t('node.hostname')" width="120" show-overflow-tooltip />
      <el-table-column :label="t('commons.actions')" width="340" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="handleAgentAction(row, 'install')" :loading="row._actionLoading">{{ t('node.install') }}</el-button>
          <el-button link type="primary" @click="handleAgentAction(row, 'update')" :loading="row._actionLoading">{{ t('node.update') }}</el-button>
          <el-button link type="primary" @click="testPanelConn(row)">{{ t('node.testConn') }}</el-button>
          <el-button link type="primary" @click="openEdit(row)">{{ t('commons.edit') }}</el-button>
          <el-button link type="warning" @click="handleAgentAction(row, 'uninstall')" :loading="row._actionLoading">{{ t('node.uninstall') }}</el-button>
          <el-button link type="danger" @click="handleDelete(row)">{{ t('commons.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Create / Edit drawer -->
    <el-drawer v-model="drawerVisible" :title="editMode ? t('commons.edit') : t('node.addNode')" size="520px" destroy-on-close>
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-divider content-position="left">{{ t('node.basicInfo') }}</el-divider>
        <el-form-item :label="t('commons.name')" prop="name">
          <el-input v-model="form.name" :placeholder="t('node.namePlaceholder')" />
        </el-form-item>

        <el-divider content-position="left">{{ t('node.sshConfig') }}</el-divider>
        <el-form-item :label="t('node.sshHost')" prop="sshHost">
          <el-input v-model="form.sshHost" placeholder="192.168.1.100" />
        </el-form-item>
        <el-form-item :label="t('node.sshPort')">
          <el-input-number v-model="form.sshPort" :min="1" :max="65535" />
        </el-form-item>
        <el-form-item :label="t('node.sshUser')" prop="sshUser">
          <el-input v-model="form.sshUser" placeholder="root" />
        </el-form-item>
        <el-form-item :label="t('node.sshPassword')" prop="sshPassword">
          <el-input v-model="form.sshPassword" type="password" show-password :placeholder="editMode ? t('node.passwordNoChange') : ''" />
        </el-form-item>
        <el-form-item>
          <el-button @click="handleTestSSH" :loading="sshTesting">
            {{ t('node.testSSH') }}
          </el-button>
          <el-tag v-if="sshTestResult === 'success'" type="success" size="small" style="margin-left:8px">{{ t('node.sshSuccess') }}</el-tag>
          <el-tag v-if="sshTestResult === 'fail'" type="danger" size="small" style="margin-left:8px">{{ t('node.sshFail') }}</el-tag>
        </el-form-item>

        <el-divider content-position="left">{{ t('node.agentConfig') }}</el-divider>
        <el-form-item :label="t('node.panelPort')">
          <el-input v-model="form.panelPort" placeholder="7777" />
          <div class="form-hint">{{ t('node.panelPortHint') }}</div>
        </el-form-item>
        <el-form-item :label="t('node.agentToken')">
          <div style="display:flex;gap:8px;width:100%">
            <el-input v-model="form.agentToken" type="password" show-password style="flex:1" />
            <el-button @click="generateToken" size="default">{{ t('node.generate') }}</el-button>
          </div>
          <div class="form-hint">{{ t('node.agentTokenHint') }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="drawerVisible = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submit">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Agent Action Output dialog -->
    <el-dialog v-model="outputDialog" :title="outputTitle" width="700px" destroy-on-close>
      <el-input type="textarea" :model-value="actionOutput" :rows="20" readonly class="output-textarea" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { NodeItem } from '@/api/interface'
import { listNodes, createNode, updateNode, deleteNode, testNodeConnection, testSSH, agentAction } from '@/api/modules/node'

const { t } = useI18n()
const loading = ref(false)
const nodes = ref<NodeItem[]>([])

const drawerVisible = ref(false)
const editMode = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const sshTesting = ref(false)
const sshTestResult = ref('')

const defaultForm = () => ({
  id: 0, name: '', sshHost: '', sshPort: 22, sshUser: 'root', sshPassword: '',
  panelPort: '7777', agentToken: '', groupID: 0,
})
const form = reactive(defaultForm())
const rules: FormRules = {
  name: [{ required: true, trigger: 'blur' }],
  sshHost: [{ required: true, trigger: 'blur' }],
  sshUser: [{ required: true, trigger: 'blur' }],
  sshPassword: [{ required: true, trigger: 'blur' }],
}

const outputDialog = ref(false)
const outputTitle = ref('')
const actionOutput = ref('')

const generateToken = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let token = ''
  for (let i = 0; i < 32; i++) token += chars.charAt(Math.floor(Math.random() * chars.length))
  form.agentToken = token
}

const load = async () => {
  loading.value = true
  try {
    const res = await listNodes()
    const items = (res.data || []).map((n: NodeItem) => ({ ...n, _actionLoading: false }))
    nodes.value = items
  } finally { loading.value = false }
}

const openCreate = () => {
  Object.assign(form, defaultForm())
  sshTestResult.value = ''
  generateToken()
  editMode.value = false
  drawerVisible.value = true
}

const openEdit = (row: NodeItem) => {
  Object.assign(form, {
    id: row.id, name: row.name,
    sshHost: row.sshHost, sshPort: row.sshPort || 22, sshUser: row.sshUser, sshPassword: '',
    panelPort: row.address ? row.address.split(':').pop() : '7777',
    agentToken: '', groupID: row.groupID,
  })
  sshTestResult.value = ''
  editMode.value = true

  if (editMode.value) {
    rules.sshPassword = []
  }
  drawerVisible.value = true
}

const handleTestSSH = async () => {
  if (!form.sshHost || !form.sshUser || !form.sshPassword) {
    ElMessage.warning(t('node.fillSSHFirst'))
    return
  }
  sshTesting.value = true
  sshTestResult.value = ''
  try {
    await testSSH({ sshHost: form.sshHost, sshPort: form.sshPort, sshUser: form.sshUser, sshPassword: form.sshPassword })
    sshTestResult.value = 'success'
  } catch {
    sshTestResult.value = 'fail'
  } finally { sshTesting.value = false }
}

const submit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  submitting.value = true
  try {
    if (editMode.value) await updateNode(form)
    else await createNode(form)
    ElMessage.success(t('commons.success'))
    drawerVisible.value = false
    await load()
  } finally { submitting.value = false }
}

const handleDelete = async (row: NodeItem) => {
  await ElMessageBox.confirm(t('node.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  await deleteNode({ id: row.id })
  ElMessage.success(t('commons.success'))
  await load()
}

const testPanelConn = async (row: NodeItem) => {
  try {
    await testNodeConnection({ id: row.id })
    ElMessage.success(t('node.testSuccess'))
  } catch {
    ElMessage.error(t('node.testFail'))
  }
}

const handleAgentAction = async (row: NodeItem, action: string) => {
  const actionLabels: Record<string, string> = {
    install: t('node.install'), uninstall: t('node.uninstall'), update: t('node.update'),
  }
  try {
    await ElMessageBox.confirm(
      t('node.agentActionConfirm', { action: actionLabels[action], name: row.name }),
      t('commons.tip'),
      { type: action === 'uninstall' ? 'warning' : 'info' },
    )
  } catch { return }

  row._actionLoading = true
  outputTitle.value = `${actionLabels[action]} - ${row.name}`
  actionOutput.value = t('node.executing') + '...\n'
  outputDialog.value = true

  try {
    const res = await agentAction({ id: row.id, action })
    actionOutput.value = res.data.output || ''
    if (res.data.success) {
      ElMessage.success(t('node.actionSuccess'))
    } else {
      ElMessage.warning(t('node.actionFailed'))
    }
    await load()
  } catch {
    actionOutput.value += '\n' + t('node.actionFailed')
  } finally { row._actionLoading = false }
}

onMounted(() => load())
</script>

<style scoped>
.form-hint { margin-top: 4px; font-size: 12px; color: var(--el-text-color-secondary); }
.output-textarea :deep(.el-textarea__inner) {
  font-family: var(--xp-font-mono);
  font-size: 12px;
  background: var(--xp-bg-inset);
  color: var(--xp-text-primary);
  line-height: 1.5;
}
</style>
