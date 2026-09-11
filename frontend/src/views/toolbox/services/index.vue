<template>
  <div class="services-page">
    <div class="app-toolbar">
      <el-input
        v-model="searchKey"
        :placeholder="$t('toolbox.services.search')"
        prefix-icon="Search"
        clearable
        class="search-input"
      />
      <el-checkbox v-model="showAll" @change="loadServices">{{ $t('toolbox.services.showAll') }}</el-checkbox>
      <el-checkbox v-model="showSystemOnly">{{ $t('toolbox.services.showSystemOnly') }}</el-checkbox>
      <div class="toolbar-spacer" />
      <el-button :icon="Refresh" :loading="loading" @click="loadServices">{{ $t('commons.refresh') }}</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">{{ $t('toolbox.services.createService') }}</el-button>
    </div>

    <el-table
      :data="filteredServices"
      v-loading="loading"
      stripe
      style="width: 100%"
      :default-sort="{ prop: 'name', order: 'ascending' }"
    >
      <el-table-column prop="name" :label="$t('toolbox.services.name')" min-width="240" sortable>
        <template #default="{ row }">
          <div class="svc-cell">
            <div class="svc-title">
              <button type="button" class="svc-name" @click="openDetail(row.name)">{{ row.name }}</button>
              <el-tag v-if="row.isPanel" type="success" size="small" effect="plain">{{ $t('toolbox.services.panelCreated') }}</el-tag>
            </div>
            <div v-if="row.description" class="svc-desc" :title="row.description">{{ row.description }}</div>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('toolbox.services.status')" width="110">
        <template #default="{ row }">
          <el-tag :type="stateType(row.activeState)" size="small" :title="row.subState">
            {{ stateLabel(row.activeState) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('toolbox.services.memory')" width="100">
        <template #default="{ row }">
          <span class="svc-metric">{{ row.memoryCurrent || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('toolbox.services.restarts')" width="72" align="right">
        <template #default="{ row }">
          <span class="svc-metric" :class="{ 'is-warn': row.restartCount > 0 }">{{ row.restartCount || 0 }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('toolbox.services.autoStart')" width="88" align="center">
        <template #default="{ row }">
          <el-switch v-model="row.enabled" size="small" @change="(v: boolean) => handleToggleEnabled(row, v)" />
        </template>
      </el-table-column>
      <el-table-column :label="$t('commons.actions')" width="200" fixed="right">
        <template #default="{ row }">
          <div class="svc-actions">
            <el-button
              v-if="row.activeState !== 'active'"
              link
              type="primary"
              @click="handleOp(row.name, 'start')"
            >{{ $t('toolbox.services.start') }}</el-button>
            <el-button
              v-else
              link
              type="primary"
              @click="handleOp(row.name, 'stop')"
            >{{ $t('toolbox.services.stop') }}</el-button>
            <el-button link type="primary" @click="handleOp(row.name, 'restart')">{{ $t('toolbox.services.restart') }}</el-button>
            <el-dropdown trigger="click" @command="(cmd: string) => handleMore(cmd, row)">
              <el-button link type="primary">
                {{ $t('toolbox.services.more') }}
                <el-icon class="el-icon--right"><ArrowDown /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="logs">{{ $t('toolbox.services.logs') }}</el-dropdown-item>
                  <el-dropdown-item command="edit">{{ $t('toolbox.services.edit') }}</el-dropdown-item>
                  <el-dropdown-item command="unit">Unit</el-dropdown-item>
                  <el-dropdown-item v-if="row.isPanel" command="delete" divided>
                    <span class="svc-danger">{{ $t('commons.delete') }}</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </el-table-column>
      <template #empty>
        <el-empty :description="$t('commons.noData')" />
      </template>
    </el-table>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="editVisible" :title="isEdit ? $t('toolbox.services.editService') : $t('toolbox.services.createService')" width="600px" :close-on-click-modal="false">
      <el-form :model="editForm" label-width="120px" :rules="formRules" ref="formRef">
        <el-form-item :label="$t('toolbox.services.serviceName')" prop="name">
          <el-input v-model="editForm.name" :disabled="isEdit" :placeholder="$t('toolbox.services.serviceNameHint')">
            <template #prepend v-if="!isEdit">xp-</template>
          </el-input>
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.description')">
          <el-input v-model="editForm.description" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.execStart')" prop="execStart">
          <el-input v-model="editForm.execStart" :placeholder="$t('toolbox.services.execStartHint')" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.workingDir')">
          <el-input v-model="editForm.workingDir" :placeholder="$t('toolbox.services.workingDirHint')" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.user')">
          <el-input v-model="editForm.user" :placeholder="$t('toolbox.services.userHint')" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.restartPolicy')">
          <el-select v-model="editForm.restart" style="width: 100%">
            <el-option :label="$t('toolbox.services.restartOnFailure')" value="on-failure" />
            <el-option :label="$t('toolbox.services.restartAlways')" value="always" />
            <el-option :label="$t('toolbox.services.restartOnAbnormal')" value="on-abnormal" />
            <el-option :label="$t('toolbox.services.restartNo')" value="no" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.restartSec')">
          <el-input-number v-model="editForm.restartSec" :min="0" :max="3600" />
          <span class="form-hint">{{ $t('toolbox.services.restartSecHint') }}</span>
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.environment')">
          <el-input v-model="editForm.environment" type="textarea" :rows="2" :placeholder="$t('toolbox.services.envHint')" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.services.afterTarget')">
          <el-select v-model="editForm.afterTarget" filterable allow-create default-first-option style="width: 100%"
            :placeholder="$t('toolbox.services.afterTargetHint')">
            <el-option label="network.target — 网络就绪后启动" value="network.target" />
            <el-option label="network-online.target — 网络完全连通后启动" value="network-online.target" />
            <el-option label="multi-user.target — 多用户模式就绪后启动" value="multi-user.target" />
            <el-option label="syslog.target — 系统日志就绪后启动" value="syslog.target" />
            <el-option label="mysql.service — MySQL 启动后" value="mysql.service" />
            <el-option label="postgresql.service — PostgreSQL 启动后" value="postgresql.service" />
            <el-option label="redis.service — Redis 启动后" value="redis.service" />
            <el-option label="nginx.service — Nginx 启动后" value="nginx.service" />
            <el-option label="docker.service — Docker 启动后" value="docker.service" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="!isEdit" :label="$t('toolbox.services.autoStart')">
          <el-switch v-model="editForm.autoStart" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailVisible" :title="$t('toolbox.services.serviceDetail')" width="700px">
      <el-descriptions :column="2" border size="small" v-if="detailData">
        <el-descriptions-item :label="$t('toolbox.services.name')">{{ detailData.name }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.status')">
          <el-tag :type="stateType(detailData.activeState)" size="small">{{ detailData.activeState }} ({{ detailData.subState }})</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.description')" :span="2">{{ detailData.description }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.execStart')" :span="2">{{ detailData.execStart }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.workingDir')">{{ detailData.workingDir || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.user')">{{ detailData.user || 'root' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.restartPolicy')">{{ detailData.restart }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.restartSec')">{{ detailData.restartSec || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="'PID'">{{ detailData.mainPID || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.memory')">{{ detailData.memoryCurrent || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="'CPU'">{{ detailData.cpuUsage || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.startedAt')">{{ detailData.startedAt || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('toolbox.services.unitFile')" :span="2">{{ detailData.unitFile }}</el-descriptions-item>
      </el-descriptions>
      <div v-if="detailData?.unitContent" class="detail-unit">
        <div class="detail-unit-title">{{ $t('toolbox.services.unitContent') }}</div>
        <div class="log-viewer"><pre>{{ detailData.unitContent }}</pre></div>
      </div>
      <template #footer>
        <el-button type="primary" size="small" @click="openEditFromDetail">{{ $t('toolbox.services.edit') }}</el-button>
        <el-button size="small" @click="openUnitEditor(detailData!.name); detailVisible=false">Unit</el-button>
        <el-popconfirm v-if="detailData?.isPanel" :title="$t('toolbox.services.deleteConfirm')" @confirm="handleDelete(detailData!.name)">
          <template #reference><el-button type="danger" size="small">{{ $t('commons.delete') }}</el-button></template>
        </el-popconfirm>
        <el-button @click="detailVisible = false">{{ $t('commons.close') }}</el-button>
      </template>
    </el-dialog>

    <!-- Unit 源码编辑对话框 -->
    <el-dialog v-model="unitVisible" :title="$t('toolbox.services.unitEditorTitle', { name: unitServiceName })" width="760px" :close-on-click-modal="false">
      <div class="unit-hint">{{ $t('toolbox.services.unitEditorHint') }}</div>
      <el-input v-model="unitContent" type="textarea" :rows="22" class="unit-editor" />
      <template #footer>
        <el-button @click="unitVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="unitSaving" @click="handleSaveUnit">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 日志对话框 -->
    <el-dialog v-model="logVisible" :title="$t('toolbox.services.serviceLogs') + ' - ' + logServiceName" width="860px" @closed="stopLogPoll">
      <div class="log-toolbar">
        <el-select v-model="logLines" style="width:120px" @change="loadLogs">
          <el-option :label="$t('toolbox.services.logLineCount', { n: 100 })" :value="100" />
          <el-option :label="$t('toolbox.services.logLineCount', { n: 200 })" :value="200" />
          <el-option :label="$t('toolbox.services.logLineCount', { n: 500 })" :value="500" />
          <el-option :label="$t('toolbox.services.logLineCount', { n: 1000 })" :value="1000" />
        </el-select>
        <el-switch v-model="logAutoRefresh" :active-text="$t('toolbox.services.autoRefresh')" @change="toggleLogPoll" />
        <el-button :icon="Refresh" :loading="logLoading" @click="loadLogs">{{ $t('commons.refresh') }}</el-button>
      </div>
      <div class="log-viewer log-viewer-tall" ref="logViewerRef">
        <pre v-html="colorizedLog || $t('toolbox.services.noLogs')" />
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ArrowDown, Plus, Refresh } from '@element-plus/icons-vue'
import {
  listSystemdServices, getSystemdServiceDetail,
  createSystemdService, updateSystemdService, deleteSystemdService,
  operateSystemdService, getSystemdServiceLogs,
  getServiceUnitContent, saveServiceUnitContent,
} from '@/api/modules/toolbox'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const services = ref<any[]>([])
const searchKey = ref('')
const showAll = ref(false)
const showSystemOnly = ref(false)

const filteredServices = computed(() => {
  let list = services.value
  if (showSystemOnly.value) {
    list = list.filter(s => !s.isPanel)
  }
  if (searchKey.value) {
    const key = searchKey.value.toLowerCase()
    list = list.filter(s => s.name.toLowerCase().includes(key) || (s.description || '').toLowerCase().includes(key))
  }
  return list
})

const stateType = (state: string) => {
  if (state === 'active') return 'success'
  if (state === 'failed') return 'danger'
  if (state === 'inactive') return 'info'
  return 'warning'
}

const stateLabel = (state: string) => {
  if (state === 'active') return t('toolbox.services.running')
  if (state === 'failed') return t('commons.failed')
  if (state === 'inactive') return t('toolbox.services.stopped')
  if (state === 'activating') return t('toolbox.services.activating')
  if (state === 'deactivating') return t('toolbox.services.deactivating')
  return state || '-'
}

const loadServices = async () => {
  loading.value = true
  try {
    const res = await listSystemdServices(showAll.value)
    services.value = res.data || []
  } catch { services.value = [] }
  finally { loading.value = false }
}

const handleOp = async (name: string, op: string) => {
  try {
    await operateSystemdService(name, op)
    ElMessage.success(t('commons.success'))
    loadServices()
  } catch {}
}

const handleToggleEnabled = async (row: any, val: boolean) => {
  try {
    await operateSystemdService(row.name, val ? 'enable' : 'disable')
    ElMessage.success(t('commons.success'))
  } catch { row.enabled = !val }
}

// Create/Edit
const editVisible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const formRef = ref()
const editForm = reactive({
  name: '', description: '', type: 'simple',
  execStart: '', execStartPre: '', execStopPost: '',
  workingDir: '', user: '',
  restart: 'on-failure', restartSec: 5,
  environment: '', afterTarget: 'network.target',
  stdOutput: 'journal', stdError: 'journal',
  timeoutStart: 0, timeoutStop: 0,
  autoStart: true,
})

const formRules = {
  name: [{ required: true, message: t('toolbox.services.nameRequired'), trigger: 'blur' }],
  execStart: [{ required: true, message: t('toolbox.services.execStartRequired'), trigger: 'blur' }],
}

const resetForm = () => Object.assign(editForm, {
  name: '', description: '', type: 'simple',
  execStart: '', execStartPre: '', execStopPost: '',
  workingDir: '', user: '',
  restart: 'on-failure', restartSec: 5,
  environment: '', afterTarget: 'network.target',
  stdOutput: 'journal', stdError: 'journal',
  timeoutStart: 0, timeoutStop: 0,
  autoStart: true,
})

const openCreate = () => {
  isEdit.value = false
  resetForm()
  editVisible.value = true
}

const fillFormFromDetail = (d: any) => {
  Object.assign(editForm, {
    name: d.name,
    description: d.description || '',
    type: d.type || 'simple',
    execStart: d.execStart || '',
    execStartPre: d.execStartPre || '',
    execStopPost: d.execStopPost || '',
    workingDir: d.workingDir || '',
    user: d.user || '',
    restart: d.restart || 'on-failure',
    restartSec: typeof d.restartSec === 'number' ? d.restartSec : 5,
    environment: d.environment || '',
    afterTarget: d.afterTarget || 'network.target',
    stdOutput: d.stdOutput || 'journal',
    stdError: d.stdError || 'journal',
    timeoutStart: d.timeoutStart || 0,
    timeoutStop: d.timeoutStop || 0,
    autoStart: true,
  })
}

const openEdit = async (name: string) => {
  try {
    const res = await getSystemdServiceDetail(name)
    isEdit.value = true
    fillFormFromDetail(res.data)
    editVisible.value = true
  } catch {}
}

const openEditFromDetail = () => {
  if (!detailData.value) return
  isEdit.value = true
  fillFormFromDetail(detailData.value)
  detailVisible.value = false
  editVisible.value = true
}

const handleSave = async () => {
  saving.value = true
  try {
    if (isEdit.value) {
      await updateSystemdService(editForm)
    } else {
      await createSystemdService(editForm)
    }
    ElMessage.success(t('commons.saveSuccess'))
    editVisible.value = false
    loadServices()
  } catch {}
  finally { saving.value = false }
}

// Detail
const detailVisible = ref(false)
const detailData = ref<any>(null)

const openDetail = async (name: string) => {
  try {
    const res = await getSystemdServiceDetail(name)
    detailData.value = res.data
    detailVisible.value = true
  } catch {}
}

const handleDelete = async (name: string) => {
  try {
    await deleteSystemdService(name)
    ElMessage.success(t('commons.success'))
    detailVisible.value = false
    loadServices()
  } catch {}
}

const handleMore = (cmd: string, row: { name: string; isPanel?: boolean }) => {
  if (cmd === 'logs') openLogs(row.name)
  else if (cmd === 'edit') openEdit(row.name)
  else if (cmd === 'unit') openUnitEditor(row.name)
  else if (cmd === 'delete' && row.isPanel) confirmDelete(row.name)
}

const confirmDelete = async (name: string) => {
  try {
    await ElMessageBox.confirm(t('toolbox.services.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  } catch {
    return
  }
  await handleDelete(name)
}

// Unit 源码编辑
const unitVisible = ref(false)
const unitServiceName = ref('')
const unitContent = ref('')
const unitSaving = ref(false)

const openUnitEditor = async (name: string) => {
  unitServiceName.value = name
  unitContent.value = ''
  try {
    const res = await getServiceUnitContent(name)
    unitContent.value = res.data || ''
  } catch {}
  unitVisible.value = true
}

const handleSaveUnit = async () => {
  unitSaving.value = true
  try {
    await saveServiceUnitContent(unitServiceName.value, unitContent.value)
    ElMessage.success(t('commons.saveSuccess'))
    unitVisible.value = false
    loadServices()
  } catch {}
  finally { unitSaving.value = false }
}

// Logs
const logVisible = ref(false)
const logServiceName = ref('')
const logContent = ref('')
const logLines = ref(100)
const logLoading = ref(false)
const logAutoRefresh = ref(false)
const logViewerRef = ref<HTMLElement>()
let logPollTimer: ReturnType<typeof setInterval> | null = null

const colorizedLog = computed(() => {
  if (!logContent.value) return ''
  return logContent.value
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/(\bERROR\b|\berror\b|\bERR\b|\bFailed\b|\bfailed\b)/g, '<span class="log-err">$1</span>')
    .replace(/(\bWARN\b|\bwarning\b|\bWARNING\b)/g, '<span class="log-warn">$1</span>')
    .replace(/(\bINFO\b|\binfo\b|\bStarted\b|\bactive\b)/g, '<span class="log-info">$1</span>')
    .replace(/(\bDEBUG\b|\bdebug\b)/g, '<span class="log-debug">$1</span>')
})

const openLogs = (name: string) => {
  logServiceName.value = name
  logContent.value = ''
  logAutoRefresh.value = false
  logVisible.value = true
  loadLogs()
}

const loadLogs = async () => {
  logLoading.value = true
  try {
    const res = await getSystemdServiceLogs(logServiceName.value, logLines.value)
    logContent.value = res.data || ''
    await nextTick()
    if (logViewerRef.value) logViewerRef.value.scrollTop = logViewerRef.value.scrollHeight
  } catch { logContent.value = '' }
  finally { logLoading.value = false }
}

const toggleLogPoll = (val: boolean) => {
  if (val) {
    logPollTimer = setInterval(loadLogs, 3000)
  } else {
    stopLogPoll()
  }
}

const stopLogPoll = () => {
  if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null }
  logAutoRefresh.value = false
}

onMounted(() => loadServices())
onUnmounted(() => stopLogPoll())
</script>

<style lang="scss" scoped>
.services-page {
  min-height: 100%;
}

.app-toolbar {
  flex-wrap: wrap;
}

.search-input {
  width: 260px;
}

.toolbar-spacer {
  flex: 1;
}

.svc-cell {
  min-width: 0;
  padding: 4px 0;
}

.svc-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.svc-name {
  padding: 0;
  border: 0;
  background: none;
  color: var(--xp-accent);
  font: inherit;
  font-weight: 650;
  cursor: pointer;
  text-align: left;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &:hover {
    text-decoration: underline;
  }
}

.svc-desc {
  margin-top: 2px;
  color: var(--xp-text-muted);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.svc-metric {
  color: var(--xp-text-secondary);
  font-variant-numeric: tabular-nums;

  &.is-warn {
    color: var(--xp-warning);
    font-weight: 650;
  }
}

.svc-actions {
  display: inline-flex;
  align-items: center;
  flex-wrap: nowrap;
  gap: 2px;
}

.svc-danger {
  color: var(--xp-danger);
}

.form-hint {
  margin-left: 8px;
  font-size: 12px;
  color: var(--xp-text-muted);
}

.unit-hint {
  margin-bottom: 8px;
  color: var(--xp-text-muted);
  font-size: 12px;
}

.unit-editor {
  :deep(.el-textarea__inner) {
    font-family: var(--xp-font-mono);
    font-size: 12px;
  }
}

.log-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.log-viewer {
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  padding: 12px;
  overflow: auto;

  pre {
    margin: 0;
    color: var(--xp-text-primary);
    font-family: var(--xp-font-mono);
    font-size: 12px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-all;
  }

  :deep(.log-err) { color: var(--xp-danger); }
  :deep(.log-warn) { color: var(--xp-warning); }
  :deep(.log-info) { color: var(--xp-success); }
  :deep(.log-debug) { color: var(--xp-text-muted); }
}

.log-viewer-tall {
  max-height: 520px;
}

.detail-unit {
  margin-top: 16px;
}

.detail-unit-title {
  margin-bottom: 8px;
  color: var(--xp-text-primary);
  font-size: 13px;
  font-weight: 600;
}
</style>
