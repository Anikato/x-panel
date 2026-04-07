<template>
  <div class="fail2ban-page">
    <!-- 未安装 -->
    <el-card v-if="!status.isInstalled" shadow="never">
      <el-empty :description="$t('toolbox.fail2ban.notInstalled')">
        <el-button type="primary" @click="handleInstall" :loading="installLoading">{{ $t('toolbox.fail2ban.install') }}</el-button>
      </el-empty>
    </el-card>

    <!-- 已安装 -->
    <template v-if="status.isInstalled">
      <!-- 状态栏 -->
      <el-card shadow="never" style="margin-bottom: 16px">
        <div class="status-bar">
          <div class="status-item">
            <span class="status-label">{{ $t('toolbox.fail2ban.serviceStatus') }}</span>
            <el-tag :type="status.isRunning ? 'success' : 'danger'" size="small">
              {{ status.isRunning ? $t('toolbox.fail2ban.running') : $t('toolbox.fail2ban.stopped') }}
            </el-tag>
          </div>
          <div class="status-item" v-if="status.version">
            <span class="status-label">{{ $t('toolbox.fail2ban.version') }}</span>
            <span>{{ status.version }}</span>
          </div>
          <div class="status-item">
            <span class="status-label">{{ $t('toolbox.fail2ban.totalBanned') }}</span>
            <el-tag type="warning" size="small">{{ totalBanned }}</el-tag>
          </div>
          <div class="status-actions">
            <el-button-group size="small">
              <el-button @click="handleOperate('start')" :disabled="status.isRunning">{{ $t('toolbox.fail2ban.start') }}</el-button>
              <el-button @click="handleOperate('stop')" :disabled="!status.isRunning">{{ $t('toolbox.fail2ban.stop') }}</el-button>
              <el-button @click="handleOperate('restart')">{{ $t('toolbox.fail2ban.restart') }}</el-button>
            </el-button-group>
            <div class="autostart-toggle">
              <span class="status-label">{{ $t('toolbox.fail2ban.autoStart') }}</span>
              <el-switch v-model="status.autoStart" @change="handleAutoStart" size="small" />
            </div>
          </div>
        </div>
      </el-card>

      <!-- Tabs -->
      <el-card shadow="never">
        <el-tabs v-model="activeTab">
          <!-- SSH 防护 -->
          <el-tab-pane :label="$t('toolbox.fail2ban.sshProtection')" name="ssh">
            <el-form :model="sshForm" label-width="140px" style="max-width: 600px; margin-top: 16px;">
              <el-form-item :label="$t('toolbox.fail2ban.enabled')">
                <el-switch v-model="sshForm.enabled" />
              </el-form-item>
              <el-form-item :label="$t('toolbox.fail2ban.sshPort')">
                <el-input v-model="sshForm.port" :placeholder="$t('toolbox.fail2ban.sshPortHint')" style="width: 200px" />
              </el-form-item>
              <el-form-item :label="$t('toolbox.fail2ban.maxRetry')">
                <el-input-number v-model="sshForm.maxRetry" :min="1" :max="100" />
                <span class="form-hint">{{ $t('toolbox.fail2ban.maxRetryHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('toolbox.fail2ban.findTime')">
                <el-input v-model="sshForm.findTime" style="width: 200px" />
                <span class="form-hint">{{ $t('toolbox.fail2ban.findTimeHint') }}</span>
              </el-form-item>
              <el-form-item :label="$t('toolbox.fail2ban.banTime')">
                <el-input v-model="sshForm.banTime" style="width: 200px" />
                <span class="form-hint">{{ $t('toolbox.fail2ban.banTimeHint') }}</span>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="handleSaveSSH" :loading="sshSaving">{{ $t('commons.save') }}</el-button>
              </el-form-item>
            </el-form>
          </el-tab-pane>

          <!-- 封禁列表 -->
          <el-tab-pane :label="$t('toolbox.fail2ban.bannedList')" name="banned">
            <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center;">
              <el-input v-model="banFilter" :placeholder="$t('toolbox.fail2ban.filterIP')" clearable style="width: 240px" size="small" />
              <el-button size="small" :icon="Refresh" @click="loadBanned" :loading="bannedLoading">{{ $t('commons.refresh') }}</el-button>
            </div>
            <el-table :data="filteredBanned" v-loading="bannedLoading" stripe size="small">
              <el-table-column prop="ip" label="IP" min-width="180" />
              <el-table-column prop="jail" label="Jail" width="140" />
              <el-table-column :label="$t('commons.actions')" width="100" align="center">
                <template #default="{ row }">
                  <el-popconfirm :title="$t('toolbox.fail2ban.unbanConfirm', { ip: row.ip })" @confirm="handleUnban(row)">
                    <template #reference>
                      <el-button type="warning" text size="small">{{ $t('toolbox.fail2ban.unban') }}</el-button>
                    </template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
            <div v-if="!bannedLoading && bannedList.length === 0" style="text-align: center; padding: 20px; color: var(--xp-text-muted);">
              {{ $t('toolbox.fail2ban.noBanned') }}
            </div>
          </el-tab-pane>

          <!-- Jail 管理 -->
          <el-tab-pane :label="$t('toolbox.fail2ban.jailManage')" name="jails">
            <el-button size="small" :icon="Refresh" @click="loadJails" :loading="jailsLoading" style="margin-bottom: 12px">{{ $t('commons.refresh') }}</el-button>
            <el-table :data="jails" v-loading="jailsLoading" stripe size="small">
              <el-table-column prop="name" :label="$t('toolbox.fail2ban.jailName')" width="140" />
              <el-table-column :label="$t('toolbox.fail2ban.enabled')" width="80" align="center">
                <template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'" size="small">{{ row.enabled ? $t('toolbox.fail2ban.yes') : $t('toolbox.fail2ban.no') }}</el-tag></template>
              </el-table-column>
              <el-table-column prop="port" :label="$t('toolbox.fail2ban.port')" width="100" />
              <el-table-column prop="maxRetry" :label="$t('toolbox.fail2ban.maxRetry')" width="80" />
              <el-table-column prop="findTime" :label="$t('toolbox.fail2ban.findTime')" width="100" />
              <el-table-column prop="banTime" :label="$t('toolbox.fail2ban.banTime')" width="100" />
              <el-table-column :label="$t('toolbox.fail2ban.currentBanned')" width="100" align="center">
                <template #default="{ row }"><el-tag v-if="row.bannedCount > 0" type="danger" size="small">{{ row.bannedCount }}</el-tag><span v-else>0</span></template>
              </el-table-column>
              <el-table-column :label="$t('commons.actions')" width="80" align="center">
                <template #default="{ row }">
                  <el-button text size="small" @click="openEditJail(row)">{{ $t('commons.edit') }}</el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>

          <!-- 日志 -->
          <el-tab-pane :label="$t('toolbox.fail2ban.logs')" name="logs">
            <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center;">
              <el-select v-model="logLines" size="small" style="width: 120px">
                <el-option :label="'100 ' + $t('toolbox.fail2ban.lines')" :value="100" />
                <el-option :label="'200 ' + $t('toolbox.fail2ban.lines')" :value="200" />
                <el-option :label="'500 ' + $t('toolbox.fail2ban.lines')" :value="500" />
                <el-option :label="'1000 ' + $t('toolbox.fail2ban.lines')" :value="1000" />
              </el-select>
              <el-button size="small" :icon="Refresh" @click="loadLogs" :loading="logsLoading">{{ $t('commons.refresh') }}</el-button>
            </div>
            <div class="log-viewer">
              <pre>{{ logContent || $t('toolbox.fail2ban.noLogs') }}</pre>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </template>

    <!-- 编辑 Jail 对话框 -->
    <el-dialog v-model="editJailVisible" :title="$t('toolbox.fail2ban.editJail')" width="500px" :close-on-click-modal="false">
      <el-form :model="editJailForm" label-width="120px">
        <el-form-item :label="$t('toolbox.fail2ban.jailName')">
          <el-input :model-value="editJailForm.name" disabled />
        </el-form-item>
        <el-form-item :label="$t('toolbox.fail2ban.enabled')">
          <el-switch v-model="editJailForm.enabled" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.fail2ban.port')">
          <el-input v-model="editJailForm.port" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.fail2ban.maxRetry')">
          <el-input-number v-model="editJailForm.maxRetry" :min="1" :max="100" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.fail2ban.findTime')">
          <el-input v-model="editJailForm.findTime" />
        </el-form-item>
        <el-form-item :label="$t('toolbox.fail2ban.banTime')">
          <el-input v-model="editJailForm.banTime" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editJailVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleSaveJail" :loading="jailSaving">{{ $t('commons.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import {
  getFail2banStatus, installFail2ban, operateFail2ban,
  listFail2banJails, updateFail2banJail, setFail2banSSH,
  listFail2banBanned, unbanFail2banIP, getFail2banLogs,
} from '@/api/modules/toolbox'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const activeTab = ref('ssh')
const installLoading = ref(false)

const status = reactive({ isInstalled: false, isRunning: false, version: '', autoStart: false })

const loadStatus = async () => {
  try {
    const res = await getFail2banStatus()
    Object.assign(status, res.data || {})
  } catch {}
}

const handleInstall = async () => {
  installLoading.value = true
  try {
    await installFail2ban()
    ElMessage.success(t('commons.success'))
    await loadStatus()
    if (status.isInstalled) {
      loadJails()
      loadBanned()
    }
  } catch {}
  finally { installLoading.value = false }
}

const handleOperate = async (op: string) => {
  try {
    await operateFail2ban(op)
    ElMessage.success(t('commons.success'))
    await loadStatus()
  } catch {}
}

const handleAutoStart = async (val: boolean) => {
  try {
    await operateFail2ban(val ? 'enable' : 'disable')
    ElMessage.success(t('commons.success'))
    await loadStatus()
  } catch {}
}

// SSH 防护
const sshForm = reactive({ enabled: true, port: 'ssh', maxRetry: 5, findTime: '600', banTime: '3600' })
const sshSaving = ref(false)

const loadSSHFromJails = (jailList: any[]) => {
  const sshd = jailList.find((j: any) => j.name === 'sshd')
  if (sshd) {
    sshForm.enabled = sshd.enabled
    sshForm.port = sshd.port || 'ssh'
    sshForm.maxRetry = sshd.maxRetry || 5
    sshForm.findTime = sshd.findTime || '600'
    sshForm.banTime = sshd.banTime || '3600'
  }
}

const handleSaveSSH = async () => {
  sshSaving.value = true
  try {
    await setFail2banSSH(sshForm)
    ElMessage.success(t('commons.saveSuccess'))
    loadJails()
  } catch {}
  finally { sshSaving.value = false }
}

// Jails
const jails = ref<any[]>([])
const jailsLoading = ref(false)

const loadJails = async () => {
  jailsLoading.value = true
  try {
    const res = await listFail2banJails()
    jails.value = res.data || []
    loadSSHFromJails(jails.value)
  } catch { jails.value = [] }
  finally { jailsLoading.value = false }
}

const editJailVisible = ref(false)
const editJailForm = reactive({ name: '', enabled: true, port: '', maxRetry: 5, findTime: '', banTime: '', action: '' })
const jailSaving = ref(false)

const openEditJail = (row: any) => {
  Object.assign(editJailForm, {
    name: row.name, enabled: row.enabled, port: row.port || '',
    maxRetry: row.maxRetry || 5, findTime: row.findTime || '', banTime: row.banTime || '', action: row.action || '',
  })
  editJailVisible.value = true
}

const handleSaveJail = async () => {
  jailSaving.value = true
  try {
    await updateFail2banJail(editJailForm)
    ElMessage.success(t('commons.saveSuccess'))
    editJailVisible.value = false
    loadJails()
  } catch {}
  finally { jailSaving.value = false }
}

// Banned
const bannedList = ref<any[]>([])
const bannedLoading = ref(false)
const banFilter = ref('')

const totalBanned = computed(() => bannedList.value.length)

const filteredBanned = computed(() => {
  if (!banFilter.value) return bannedList.value
  return bannedList.value.filter((b: any) => b.ip.includes(banFilter.value) || b.jail.includes(banFilter.value))
})

const loadBanned = async () => {
  bannedLoading.value = true
  try {
    const res = await listFail2banBanned()
    bannedList.value = res.data || []
  } catch { bannedList.value = [] }
  finally { bannedLoading.value = false }
}

const handleUnban = async (row: any) => {
  try {
    await unbanFail2banIP(row.ip, row.jail)
    ElMessage.success(t('commons.success'))
    loadBanned()
  } catch {}
}

// Logs
const logContent = ref('')
const logLines = ref(200)
const logsLoading = ref(false)

const loadLogs = async () => {
  logsLoading.value = true
  try {
    const res = await getFail2banLogs(logLines.value)
    logContent.value = res.data || ''
  } catch { logContent.value = '' }
  finally { logsLoading.value = false }
}

onMounted(async () => {
  await loadStatus()
  if (status.isInstalled) {
    loadJails()
    loadBanned()
  }
})
</script>

<style lang="scss" scoped>
.status-bar {
  display: flex; align-items: center; gap: 24px; flex-wrap: wrap;
}
.status-item {
  display: flex; align-items: center; gap: 8px;
}
.status-label {
  font-size: 13px; color: var(--xp-text-muted);
}
.status-actions {
  margin-left: auto; display: flex; align-items: center; gap: 16px;
}
.autostart-toggle {
  display: flex; align-items: center; gap: 8px;
}
.form-hint {
  margin-left: 8px; font-size: 12px; color: var(--xp-text-muted);
}
.log-viewer {
  background: var(--xp-bg-inset); border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius); padding: 12px; max-height: 500px; overflow: auto;

  pre {
    margin: 0; font-size: 12px; line-height: 1.5;
    font-family: 'JetBrains Mono', 'Fira Code', monospace;
    color: var(--xp-text-primary); white-space: pre-wrap; word-break: break-all;
  }
}
</style>
