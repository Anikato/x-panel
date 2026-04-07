<template>
  <div class="toolbox-nfs-page">
    <div class="page-header">
      <h3>{{ $t('toolbox.nfsTitle') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadAll" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <!-- Not installed -->
    <template v-if="!status.isInstalled">
      <el-card shadow="never" class="install-card">
        <el-empty :description="$t('toolbox.nfsNotInstalled')">
          <template #image>
            <el-icon :size="64" color="var(--xp-text-muted)"><FolderOpened /></el-icon>
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
          <!-- Exports Tab -->
          <el-tab-pane :label="$t('toolbox.nfsExports')" name="exports">
            <div class="tab-toolbar">
              <el-button type="primary" size="small" @click="openExportDialog()">
                {{ $t('commons.create') }}
              </el-button>
            </div>
            <el-table :data="nfsExports" v-loading="exportsLoading" stripe>
              <el-table-column prop="path" :label="$t('toolbox.exportPath')" min-width="200" />
              <el-table-column :label="$t('toolbox.clients')" min-width="350">
                <template #default="{ row }">
                  <div v-for="(client, idx) in row.clients" :key="idx" class="client-item">
                    <el-tag size="small" type="info">{{ client.host }}</el-tag>
                    <span class="client-options">({{ client.options }})</span>
                  </div>
                </template>
              </el-table-column>
              <el-table-column prop="comment" :label="$t('commons.description')" width="200" />
              <el-table-column :label="$t('commons.actions')" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button link type="primary" size="small" @click="openExportDialog(row)">
                    {{ $t('commons.edit') }}
                  </el-button>
                  <el-button link type="danger" size="small" @click="handleDeleteExport(row.path)">
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
            <h4 style="margin: 8px 0;">{{ $t('toolbox.activeExports') }}</h4>
            <el-table :data="activeExportRows" v-loading="connLoading" stripe size="small">
              <el-table-column prop="line" :label="$t('toolbox.exportInfo')" />
            </el-table>
            <h4 style="margin: 16px 0 8px;">{{ $t('toolbox.connectedClients') }}</h4>
            <el-table :data="connData.clients" v-loading="connLoading" stripe size="small">
              <el-table-column prop="hostname" :label="$t('toolbox.clientHost')" min-width="200" />
              <el-table-column prop="dirPath" :label="$t('toolbox.exportPath')" min-width="200" />
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </template>

    <!-- Export Dialog -->
    <el-dialog v-model="exportDialogOpen" :title="exportForm.origPath ? $t('toolbox.editExport') : $t('toolbox.createExport')" width="700px" destroy-on-close>
      <el-form :model="exportForm" :rules="exportRules" ref="exportFormRef" label-width="120px">
        <el-form-item :label="$t('toolbox.exportPath')" prop="path">
          <el-input v-model="exportForm.path" placeholder="/data/nfs-share" />
        </el-form-item>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="exportForm.comment" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.createDir')" v-if="!exportForm.origPath">
          <el-switch v-model="exportForm.createDir" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.clients')">
          <div style="width: 100%;">
            <div v-for="(client, idx) in exportForm.clients" :key="idx" class="client-entry">
              <div class="client-row">
                <el-input v-model="client.host" :placeholder="$t('toolbox.clientHostPlaceholder')" style="width: 200px;" />
                <el-button type="danger" :icon="Delete" circle size="small" @click="exportForm.clients.splice(idx, 1)" style="margin-left: 8px;" />
              </div>
              <div class="options-grid">
                <el-checkbox v-model="client.optRW" @change="rebuildOptions(client)">
                  <el-tooltip :content="$t('toolbox.nfsOptRWDesc')">
                    <span>rw <span class="opt-hint">({{ $t('toolbox.nfsOptRW') }})</span></span>
                  </el-tooltip>
                </el-checkbox>
                <el-checkbox v-model="client.optSync" @change="rebuildOptions(client)">
                  <el-tooltip :content="$t('toolbox.nfsOptSyncDesc')">
                    <span>sync <span class="opt-hint">({{ $t('toolbox.nfsOptSync') }})</span></span>
                  </el-tooltip>
                </el-checkbox>
                <el-checkbox v-model="client.optNoSubtreeCheck" @change="rebuildOptions(client)">
                  <el-tooltip :content="$t('toolbox.nfsOptNoSubtreeCheckDesc')">
                    <span>no_subtree_check</span>
                  </el-tooltip>
                </el-checkbox>
                <el-checkbox v-model="client.optNoRootSquash" @change="rebuildOptions(client)">
                  <el-tooltip :content="$t('toolbox.nfsOptNoRootSquashDesc')">
                    <span>no_root_squash <span class="opt-hint">({{ $t('toolbox.nfsOptNoRootSquash') }})</span></span>
                  </el-tooltip>
                </el-checkbox>
                <el-checkbox v-model="client.optAllSquash" @change="rebuildOptions(client)">
                  <el-tooltip :content="$t('toolbox.nfsOptAllSquashDesc')">
                    <span>all_squash <span class="opt-hint">({{ $t('toolbox.nfsOptAllSquash') }})</span></span>
                  </el-tooltip>
                </el-checkbox>
              </div>
              <div class="options-raw">
                <el-input v-model="client.options" size="small" :placeholder="$t('toolbox.nfsOptRawPlaceholder')">
                  <template #prepend>{{ $t('toolbox.nfsOptRaw') }}</template>
                </el-input>
              </div>
            </div>
            <el-button type="primary" plain size="small" @click="addClient" style="margin-top: 8px;">
              {{ $t('toolbox.addClient') }}
            </el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="exportDialogOpen = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmitExport" :loading="exportSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, FolderOpened, VideoPlay, VideoPause, RefreshRight, Delete } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import {
  getNfsStatus, installNfs, uninstallNfs, operateNfs,
  listNfsExports, createNfsExport, updateNfsExport, deleteNfsExport,
  getNfsConnections,
} from '@/api/modules/toolbox'

const { t } = useI18n()
const loading = ref(false)
const installLoading = ref(false)
const opLoading = ref('')
const activeTab = ref('exports')

const status = ref({ isInstalled: false, isRunning: false, version: '', autoStart: false })

interface ClientEntry {
  host: string
  options: string
  optRW: boolean
  optSync: boolean
  optNoSubtreeCheck: boolean
  optNoRootSquash: boolean
  optAllSquash: boolean
}

function parseClientOptions(options: string): Partial<ClientEntry> {
  const parts = options.split(',').map(s => s.trim())
  return {
    optRW: parts.includes('rw'),
    optSync: parts.includes('sync'),
    optNoSubtreeCheck: parts.includes('no_subtree_check'),
    optNoRootSquash: parts.includes('no_root_squash'),
    optAllSquash: parts.includes('all_squash'),
  }
}

function rebuildOptions(client: ClientEntry) {
  const opts: string[] = []
  opts.push(client.optRW ? 'rw' : 'ro')
  opts.push(client.optSync ? 'sync' : 'async')
  if (client.optNoSubtreeCheck) opts.push('no_subtree_check')
  if (client.optNoRootSquash) opts.push('no_root_squash')
  if (client.optAllSquash) opts.push('all_squash')
  client.options = opts.join(',')
}

function makeClient(host = '*', options = 'rw,sync,no_subtree_check'): ClientEntry {
  const parsed = parseClientOptions(options)
  return { host, options, ...parsed } as ClientEntry
}

const addClient = () => {
  exportForm.clients.push(makeClient())
}

// Exports
const nfsExports = ref<any[]>([])
const exportsLoading = ref(false)
const exportDialogOpen = ref(false)
const exportSubmitting = ref(false)
const exportFormRef = ref<FormInstance>()
const exportForm = reactive({
  origPath: '',
  path: '',
  comment: '',
  createDir: true,
  clients: [makeClient()] as ClientEntry[],
})
const exportRules = reactive<FormRules>({
  path: [{ required: true, message: t('toolbox.pathRequired'), trigger: 'blur' }],
})

// Connections
const connData = ref<{ activeExports: string[]; clients: any[] }>({ activeExports: [], clients: [] })
const connLoading = ref(false)
const activeExportRows = computed(() => (connData.value.activeExports || []).map((line: string) => ({ line })))

const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getNfsStatus()
    if (res.data) status.value = res.data
  } finally {
    loading.value = false
  }
}

const loadExports = async () => {
  exportsLoading.value = true
  try {
    const res = await listNfsExports()
    nfsExports.value = res.data || []
  } finally {
    exportsLoading.value = false
  }
}

const loadConnections = async () => {
  connLoading.value = true
  try {
    const res = await getNfsConnections()
    if (res.data) connData.value = res.data
  } finally {
    connLoading.value = false
  }
}

const loadAll = async () => {
  await loadStatus()
  if (status.value.isInstalled) {
    loadExports()
    loadConnections()
  }
}

const handleInstall = async () => {
  installLoading.value = true
  try {
    await installNfs()
    ElMessage.success(t('commons.operationSuccess'))
    await loadAll()
  } finally {
    installLoading.value = false
  }
}

const handleUninstall = async () => {
  await ElMessageBox.confirm(t('toolbox.uninstallConfirm'), t('commons.warning'), { type: 'warning' })
  try {
    await uninstallNfs()
    ElMessage.success(t('commons.operationSuccess'))
    await loadAll()
  } catch { /* */ }
}

const handleOperate = async (op: string) => {
  opLoading.value = op
  try {
    await operateNfs(op)
    ElMessage.success(t('commons.operationSuccess'))
    await loadStatus()
  } finally {
    opLoading.value = ''
  }
}

// Export CRUD
const openExportDialog = (row?: any) => {
  if (row) {
    exportForm.origPath = row.path
    exportForm.path = row.path
    exportForm.comment = row.comment || ''
    exportForm.createDir = false
    exportForm.clients = (row.clients || []).map((c: any) => makeClient(c.host, c.options))
    if (exportForm.clients.length === 0) {
      exportForm.clients = [makeClient()]
    }
  } else {
    exportForm.origPath = ''
    exportForm.path = ''
    exportForm.comment = ''
    exportForm.createDir = true
    exportForm.clients = [makeClient()]
  }
  exportDialogOpen.value = true
}

const handleSubmitExport = async () => {
  await exportFormRef.value?.validate()
  const validClients = exportForm.clients
    .filter(c => c.host.trim())
    .map(c => ({ host: c.host, options: c.options }))
  if (validClients.length === 0) {
    ElMessage.warning(t('toolbox.clientRequired'))
    return
  }
  exportSubmitting.value = true
  try {
    const payload = { ...exportForm, clients: validClients }
    if (exportForm.origPath) {
      await updateNfsExport(payload)
    } else {
      await createNfsExport(payload)
    }
    ElMessage.success(t('commons.operationSuccess'))
    exportDialogOpen.value = false
    loadExports()
  } finally {
    exportSubmitting.value = false
  }
}

const handleDeleteExport = async (path: string) => {
  await ElMessageBox.confirm(t('toolbox.deleteExportConfirm'), t('commons.warning'), { type: 'warning' })
  await deleteNfsExport(path)
  ElMessage.success(t('commons.deleteSuccess'))
  loadExports()
}

onMounted(() => loadAll())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px;
  h3 { margin: 0; }
}
.install-card { text-align: center; }
.status-row {
  .stat-card {
    text-align: center;
    .stat-title { font-size: 13px; color: var(--xp-text-muted); margin-bottom: 10px; }
    .stat-value { font-size: 14px; font-weight: 600; }
    .mono { font-family: monospace; }
  }
}
.operate-buttons {
  display: flex; align-items: center; justify-content: center; flex-wrap: wrap; gap: 4px;
}
.tab-toolbar { margin-bottom: 12px; display: flex; justify-content: flex-end; }
.client-item {
  display: inline-flex; align-items: center; gap: 4px; margin-right: 8px; margin-bottom: 4px;
  .client-options { font-size: 12px; color: var(--xp-text-muted); }
}
.client-entry {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 10px;
  background: var(--el-fill-color-blank);
}
.client-row { display: flex; align-items: center; margin-bottom: 8px; }
.options-grid {
  display: flex; flex-wrap: wrap; gap: 4px 16px; margin-bottom: 8px;
  .opt-hint { font-size: 11px; color: var(--xp-text-muted); }
}
.options-raw { margin-top: 4px; }
</style>
