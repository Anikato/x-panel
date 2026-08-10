<template>
  <div class="nezha-agent-page" v-loading="loading && !status">
    <div class="page-header">
      <div class="page-header-text">
        <h3>{{ $t('nezhaAgent.title') }}</h3>
        <p class="page-desc">{{ $t('nezhaAgent.pageDesc') }}</p>
      </div>
      <el-button size="small" :icon="Refresh" :loading="loading" @click="loadStatus">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <template v-if="status && view">
      <!-- Alerts -->
      <div class="alerts-block">
        <el-alert
          v-if="!status.componentAvailable"
          type="error"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.status.componentMissing')"
          :description="$t('nezhaAgent.alert.componentMissing')"
        />
        <el-alert
          v-if="status.componentAvailable && !status.configured"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.status.pending')"
          :description="$t('nezhaAgent.alert.configMissing')"
        />
        <el-alert
          v-if="status.configured && !status.configHealthy"
          type="error"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.status.corrupt')"
        >
          <template #default>
            <div>{{ $t('nezhaAgent.alert.configCorrupt') }}</div>
            <div v-if="status.configError" class="alert-detail mono">
              {{ $t('nezhaAgent.alert.configError') }}: {{ status.configError }}
            </div>
          </template>
        </el-alert>
        <el-alert
          v-if="status.conflicts && status.conflicts.length > 0"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.status.conflict')"
        >
          <template #default>
            <div>{{ $t('nezhaAgent.alert.conflicts') }}</div>
            <ul class="conflict-list">
              <li v-for="(c, i) in status.conflicts" :key="i" class="mono">
                {{ $t('nezhaAgent.alert.conflictItem', { kind: c.kind, detail: c.detail, message: c.message }) }}
              </li>
            </ul>
          </template>
        </el-alert>
        <el-alert
          v-if="status.serviceError"
          type="error"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.alert.serviceError')"
          :description="status.serviceError"
        />
        <el-alert
          v-if="status.permissionsWarning"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.alert.permissionsWarning')"
          :description="status.permissionsWarning"
        />
        <el-alert
          v-if="status.drift"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.alert.drift')"
        />
        <el-alert
          v-if="status.configured && !status.tls"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.alert.tlsOff')"
        />
        <el-alert
          v-if="status.configured && status.insecureTls"
          type="warning"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.alert.insecureTls')"
        />
        <el-alert
          v-if="status.configured && !status.remoteOperationsEnabled"
          type="info"
          :closable="false"
          show-icon
          :title="$t('nezhaAgent.alert.remoteOpsOff')"
        />
      </div>

      <!-- Status cards -->
      <el-row :gutter="12" class="stat-row">
        <el-col :xs="12" :sm="12" :md="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nezhaAgent.runtimeStatus') }}</div>
            <div class="stat-value">
              <el-tag :type="statusTagType" effect="dark" round size="small">
                {{ $t(view.statusLabelKey) }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="12" :sm="12" :md="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nezhaAgent.autoStart') }}</div>
            <div class="stat-value">
              <el-tag :type="status.enabled ? 'success' : 'info'" effect="dark" round size="small">
                {{ status.enabled ? $t('nezhaAgent.enabled') : $t('nezhaAgent.disabled') }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
        <el-col :xs="12" :sm="12" :md="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nezhaAgent.version') }}</div>
            <div class="stat-value mono">{{ status.version || $t('nezhaAgent.empty') }}</div>
          </el-card>
        </el-col>
        <el-col :xs="12" :sm="12" :md="6">
          <el-card shadow="never" class="stat-card">
            <div class="stat-title">{{ $t('nezhaAgent.configHealth') }}</div>
            <div class="stat-value">
              <el-tag :type="configHealthTagType" effect="dark" round size="small">
                {{ configHealthLabel }}
              </el-tag>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- Details + Operations -->
      <el-row :gutter="12" class="detail-row">
        <el-col :xs="24" :md="14">
          <el-card shadow="never">
            <template #header>
              <span>{{ $t('nezhaAgent.details') }}</span>
            </template>
            <el-descriptions :column="1" border size="small" class="detail-desc">
              <el-descriptions-item :label="$t('nezhaAgent.uuid')">
                <span class="mono">{{ status.uuid || $t('nezhaAgent.empty') }}</span>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.dashboard')">
                <span class="mono">{{ status.dashboardUrl || $t('nezhaAgent.empty') }}</span>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.server')">
                <span class="mono">{{ status.server || $t('nezhaAgent.empty') }}</span>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.tls')">
                <el-tag :type="status.tls ? 'success' : 'danger'" size="small">
                  {{ status.tls ? $t('nezhaAgent.yes') : $t('nezhaAgent.no') }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.insecureTls')">
                <el-tag :type="status.insecureTls ? 'warning' : 'info'" size="small">
                  {{ status.insecureTls ? $t('nezhaAgent.yes') : $t('nezhaAgent.no') }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.secret')">
                <el-tag :type="status.secretConfigured ? 'success' : 'warning'" size="small">
                  {{ status.secretConfigured ? $t('nezhaAgent.secretConfigured') : $t('nezhaAgent.secretNotConfigured') }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.remoteOperations')">
                <el-tag :type="status.remoteOperationsEnabled ? 'success' : 'info'" size="small">
                  {{ status.remoteOperationsEnabled ? $t('nezhaAgent.enabled') : $t('nezhaAgent.disabled') }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item :label="$t('nezhaAgent.serviceState')">
                <span class="mono">{{ status.serviceState || $t('nezhaAgent.empty') }}</span>
              </el-descriptions-item>
            </el-descriptions>
          </el-card>
        </el-col>

        <el-col :xs="24" :md="10">
          <el-card shadow="never">
            <template #header>
              <span>{{ $t('nezhaAgent.operations') }}</span>
            </template>
            <div class="operate-buttons">
              <el-button
                v-if="!status.componentAvailable"
                type="primary"
                size="small"
                :loading="configSaving"
                @click="openInstall"
              >
                {{ $t('nezhaAgent.installAndConfigure') }}
              </el-button>
              <el-button
                v-if="view.canConfigure"
                type="primary"
                size="small"
                @click="openConfig"
              >
                {{ status.configured ? $t('nezhaAgent.editConfig') : $t('nezhaAgent.configure') }}
              </el-button>
              <el-button
                v-if="view.canStart"
                type="success"
                size="small"
                :loading="operateLoading === 'start'"
                @click="handleOperate('start')"
              >
                <el-icon><VideoPlay /></el-icon>
                {{ $t('nezhaAgent.start') }}
              </el-button>
              <el-button
                v-if="view.canStop"
                type="danger"
                size="small"
                :loading="operateLoading === 'stop'"
                :title="$t('nezhaAgent.stopHint')"
                @click="handleStop"
              >
                <el-icon><VideoPause /></el-icon>
                {{ $t(temporaryStop.labelKey) }}
              </el-button>
              <el-button
                v-if="view.canRestart"
                type="primary"
                size="small"
                :loading="operateLoading === 'restart'"
                @click="handleOperate('restart')"
              >
                <el-icon><RefreshRight /></el-icon>
                {{ $t('nezhaAgent.restart') }}
              </el-button>
              <el-button
                v-if="view.canEnable"
                type="success"
                plain
                size="small"
                :loading="operateLoading === 'enable'"
                @click="handleOperate('enable')"
              >
                {{ $t('nezhaAgent.enable') }}
              </el-button>
              <el-button
                v-if="view.canDisable"
                type="danger"
                plain
                size="small"
                :loading="operateLoading === 'disable'"
                @click="handleCompleteDisable"
              >
                {{ $t(completeDisable.labelKey) }}
              </el-button>
              <el-button
                v-if="view.canViewLogs"
                size="small"
                @click="openLogs"
              >
                {{ $t('nezhaAgent.logs') }}
              </el-button>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- Config dialog -->
    <el-dialog
      v-model="configVisible"
      :title="configDialogTitle"
      width="520px"
      :close-on-click-modal="false"
      @closed="resetConfigForm"
    >
      <el-form ref="formRef" :model="form" :rules="formRules" label-width="130px" size="default">
        <el-form-item :label="$t('nezhaAgent.form.dashboardUrl')" prop="dashboardUrl">
          <el-input
            v-model="form.dashboardUrl"
            :placeholder="$t('nezhaAgent.form.dashboardUrlPlaceholder')"
            clearable
          />
          <div class="form-hint">{{ $t('nezhaAgent.form.dashboardUrlHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('nezhaAgent.form.clientSecret')" prop="clientSecret">
          <el-input
            v-model="form.clientSecret"
            type="password"
            show-password
            autocomplete="new-password"
            :placeholder="$t('nezhaAgent.form.clientSecretPlaceholder')"
          />
          <div class="form-hint">{{ $t('nezhaAgent.form.clientSecretHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('nezhaAgent.form.remoteOperationsEnabled')">
          <el-switch v-model="form.remoteOperationsEnabled" />
          <div class="form-hint">{{ $t('nezhaAgent.form.remoteOperationsHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('nezhaAgent.form.enableAndStart')">
          <el-switch v-model="form.enableAndStart" />
          <div class="form-hint">{{ $t('nezhaAgent.form.enableAndStartHint') }}</div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="configVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="configSaving" @click="submitConfig">
          {{ form.enableAndStart ? $t('nezhaAgent.form.submitAndStart') : $t('nezhaAgent.form.submit') }}
        </el-button>
      </template>
    </el-dialog>

    <!-- Logs dialog — fixed unit name, plain-text pre (no v-html) -->
    <el-dialog
      v-model="logVisible"
      :title="$t('nezhaAgent.logsTitle')"
      width="860px"
    >
      <div class="log-toolbar">
        <el-select v-model="logLines" size="small" style="width: 120px" @change="loadLogs">
          <el-option :label="`100 ${$t('nezhaAgent.logLines')}`" :value="100" />
          <el-option :label="`200 ${$t('nezhaAgent.logLines')}`" :value="200" />
          <el-option :label="`500 ${$t('nezhaAgent.logLines')}`" :value="500" />
          <el-option :label="`1000 ${$t('nezhaAgent.logLines')}`" :value="1000" />
        </el-select>
        <el-button size="small" :icon="Refresh" :loading="logLoading" @click="loadLogs">
          {{ $t('commons.refresh') }}
        </el-button>
      </div>
      <div class="log-viewer" v-loading="logLoading">
        <pre>{{ logContent || $t('nezhaAgent.noLogs') }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Refresh, VideoPlay, VideoPause, RefreshRight } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { useI18n } from 'vue-i18n'
import {
  getNezhaAgentStatus,
  updateNezhaAgentConfig,
  installNezhaAgent,
  operateNezhaAgent,
  type NezhaAgentConfigPayload,
  type NezhaAgentOperation,
} from '@/api/modules/nezha-agent'
import { getSystemdServiceLogs } from '@/api/modules/toolbox'
import {
  completeDisable,
  createNezhaAgentForm,
  deriveNezhaAgentView,
  temporaryStop,
  type NezhaAgentForm,
  type NezhaAgentStatus,
} from './state'

/** Fixed bundled unit name — never accept user-supplied unit names. */
const NEZHA_AGENT_UNIT = 'xpanel-nezha-agent'

const { t } = useI18n()

const loading = ref(false)
const status = ref<NezhaAgentStatus | null>(null)
const operateLoading = ref('')
const configVisible = ref(false)
const configSaving = ref(false)
const installMode = ref(false)
const formRef = ref<FormInstance>()
const form = ref<NezhaAgentForm>({
  dashboardUrl: '',
  clientSecret: '',
  remoteOperationsEnabled: true,
  enableAndStart: true,
})
/** Snapshot of remote ops when the dialog opened — used for disable confirmation. */
const originalRemoteOps = ref(false)

const logVisible = ref(false)
const logContent = ref('')
const logLines = ref(200)
const logLoading = ref(false)

const view = computed(() => (status.value ? deriveNezhaAgentView(status.value) : null))

const statusTagType = computed(() => {
  if (!view.value) return 'info'
  switch (view.value.kind) {
    case 'running':
      return 'success'
    case 'stopped':
    case 'pending':
      return 'info'
    case 'component-missing':
    case 'corrupt':
    case 'service-error':
      return 'danger'
    case 'conflict':
      return 'warning'
    default:
      return 'info'
  }
})

const configHealthTagType = computed(() => {
  if (!status.value) return 'info'
  if (!status.value.configured) return 'info'
  return status.value.configHealthy ? 'success' : 'danger'
})

const configHealthLabel = computed(() => {
  if (!status.value) return t('nezhaAgent.empty')
  if (!status.value.configured) return t('nezhaAgent.configMissing')
  return status.value.configHealthy ? t('nezhaAgent.configHealthy') : t('nezhaAgent.configUnhealthy')
})

const configDialogTitle = computed(() => {
  if (installMode.value) return t('nezhaAgent.installAndConfigure')
  if (status.value?.configured) return t('nezhaAgent.editConfig')
  return t('nezhaAgent.configureAndStart')
})

const formRules = computed<FormRules>(() => ({
  dashboardUrl: [
    {
      required: true,
      message: () => t('nezhaAgent.form.dashboardRequired'),
      trigger: 'blur',
    },
  ],
  clientSecret: [
    {
      validator: (_rule, value: string, callback) => {
        // First-time setup (secretConfigured=false): secret is required.
        // Existing secret: empty means leave unchanged.
        if (!status.value?.secretConfigured) {
          if (!value || !String(value).trim()) {
            callback(new Error(t('nezhaAgent.form.clientSecretRequired')))
            return
          }
        }
        callback()
      },
      trigger: 'blur',
    },
  ],
}))

const loadStatus = async () => {
  loading.value = true
  try {
    const res = await getNezhaAgentStatus()
    if (res.data) {
      status.value = res.data as NezhaAgentStatus
    }
  } catch {
    /* interceptor surfaces error */
  } finally {
    loading.value = false
  }
}

const openConfig = () => {
  if (!status.value) return
  form.value = createNezhaAgentForm(status.value)
  originalRemoteOps.value = status.value.remoteOperationsEnabled
  configVisible.value = true
}

const openInstall = () => {
  if (!status.value) return
  installMode.value = true
  form.value = createNezhaAgentForm(status.value)
  form.value.enableAndStart = true
  originalRemoteOps.value = status.value.remoteOperationsEnabled
  configVisible.value = true
}

const resetConfigForm = () => {
  installMode.value = false
  form.value = {
    dashboardUrl: '',
    clientSecret: '',
    remoteOperationsEnabled: true,
    enableAndStart: false,
  }
  formRef.value?.clearValidate()
}

const submitConfig = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  // Confirm before turning remote ops off when it was previously enabled.
  if (originalRemoteOps.value && !form.value.remoteOperationsEnabled) {
    try {
      await ElMessageBox.confirm(
        t('nezhaAgent.form.remoteOpsDisableConfirm'),
        t('commons.warning'),
        { type: 'warning', confirmButtonText: t('commons.confirm'), cancelButtonText: t('commons.cancel') },
      )
    } catch {
      return
    }
  }

  // Build payload: omit clientSecret entirely when blank (leave unchanged).
  // Emptiness uses trim(); non-empty secrets are sent as the original value
  // (do not trim/alter the real key — leading/trailing spaces may be significant).
  const payload: NezhaAgentConfigPayload = {
    dashboardUrl: form.value.dashboardUrl.trim(),
    remoteOperationsEnabled: form.value.remoteOperationsEnabled,
    enableAndStart: form.value.enableAndStart,
  }
  if (form.value.clientSecret.trim()) {
    payload.clientSecret = form.value.clientSecret
  }

  configSaving.value = true
  let installedForSubmit = false
  try {
    if (installMode.value) {
      // Install first: AgentSecret is not sent unless component repair succeeds.
      await installNezhaAgent()
      installedForSubmit = true
    }
    await updateNezhaAgentConfig(payload)
    ElMessage.success(t('commons.operationSuccess'))
    // Clear secret after success before closing.
    form.value.clientSecret = ''
    configVisible.value = false
  } catch {
    // If configuration/start fails after repair, retain assets but leave the unit stopped.
    if (installedForSubmit) {
      try {
        await operateNezhaAgent('stop')
      } catch {
        /* original request error is already surfaced by the interceptor */
      }
    }
  } finally {
    configSaving.value = false
    // Ensure secret never lingers in form state after a submit attempt.
    form.value.clientSecret = ''
    await loadStatus()
  }
}

const handleOperate = async (operation: NezhaAgentOperation) => {
  operateLoading.value = operation
  try {
    await operateNezhaAgent(operation)
    ElMessage.success(t('commons.operationSuccess'))
    setTimeout(() => loadStatus(), 800)
  } catch {
    /* interceptor */
  } finally {
    operateLoading.value = ''
  }
}

/** Runtime-only stop — does not change boot enablement. */
const handleStop = async () => {
  await handleOperate(temporaryStop.operation)
}

/** Complete disable (disable --now): retains binary, config.yml, and UUID. */
const handleCompleteDisable = async () => {
  try {
    await ElMessageBox.confirm(
      t(completeDisable.retainNoteKey),
      t(completeDisable.labelKey),
      { type: 'warning', confirmButtonText: t('commons.confirm'), cancelButtonText: t('commons.cancel') },
    )
  } catch {
    return
  }
  await handleOperate(completeDisable.operation)
}

const openLogs = () => {
  logContent.value = ''
  logVisible.value = true
  loadLogs()
}

const loadLogs = async () => {
  logLoading.value = true
  try {
    // Always the fixed bundled unit — never accept user unit names.
    const res = await getSystemdServiceLogs(NEZHA_AGENT_UNIT, logLines.value)
    logContent.value = (res.data as string) || ''
  } catch {
    logContent.value = ''
  } finally {
    logLoading.value = false
  }
}

onMounted(() => {
  loadStatus()
})
</script>

<style lang="scss" scoped>
.nezha-agent-page {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;

  h3 {
    margin: 0 0 4px;
    font-size: 18px;
    color: var(--xp-text-primary);
  }

  .page-desc {
    margin: 0;
    font-size: 13px;
    color: var(--xp-text-muted);
    max-width: 560px;
  }
}

.alerts-block {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;

  :deep(.el-alert) {
    align-items: flex-start;
  }
}

.alert-detail {
  margin-top: 6px;
  font-size: 12px;
  color: var(--xp-text-secondary, var(--xp-text-muted));
}

.conflict-list {
  margin: 6px 0 0;
  padding-left: 18px;
  font-size: 12px;
  line-height: 1.6;
}

.stat-row {
  margin-bottom: 12px;

  .el-col {
    margin-bottom: 12px;
  }
}

.stat-card {
  text-align: center;
  height: 100%;

  .stat-title {
    font-size: 12px;
    color: var(--xp-text-muted);
    margin-bottom: 10px;
  }

  .stat-value {
    font-size: 14px;
    font-weight: 600;
    min-height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
}

.detail-row {
  .el-col {
    margin-bottom: 12px;
  }
}

.detail-desc {
  :deep(.el-descriptions__label) {
    width: 120px;
  }
}

.operate-buttons {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  word-break: break-all;
}

.form-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--xp-text-muted);
  line-height: 1.4;
  width: 100%;
}

.log-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.log-viewer {
  background: var(--xp-bg-inset, var(--el-fill-color-darker));
  border: 1px solid var(--xp-border-light, var(--el-border-color-lighter));
  border-radius: var(--xp-radius, 4px);
  padding: 12px;
  max-height: 520px;
  overflow: auto;

  pre {
    margin: 0;
    font-size: 12px;
    line-height: 1.5;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    color: var(--xp-text-primary);
    white-space: pre-wrap;
    word-break: break-all;
  }
}
</style>
