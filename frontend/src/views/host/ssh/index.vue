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
              <el-switch v-model="passwordAuth" @change="handleUpdate('PasswordAuthentication', passwordAuth ? 'yes' : 'no')" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.pubkeyAuth')">
              <el-switch v-model="pubkeyAuth" @change="handleUpdate('PubkeyAuthentication', pubkeyAuth ? 'yes' : 'no')" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.useDNS')">
              <el-switch v-model="useDNS" @change="handleUpdate('UseDNS', useDNS ? 'yes' : 'no')" />
            </el-form-item>
            <el-form-item :label="$t('sshManage.autoStart')">
              <el-switch v-model="sshInfo.autoStart" @change="handleOperate(sshInfo.autoStart ? 'enable' : 'disable')" />
            </el-form-item>
          </el-form>
          <el-empty v-else :description="sshInfo.message || $t('sshManage.notInstalled')" />
        </el-card>
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
import { ref, computed, onMounted, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getSSHInfo, operateSSH, updateSSHConfig, searchSSHLog } from '@/api/modules/ssh-manage'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const activeTab = ref('config')
const loading = ref(false)

const sshInfo = ref<any>({})
const passwordAuth = computed({
  get: () => sshInfo.value.passwordAuthentication === 'yes',
  set: () => {},
})
const pubkeyAuth = computed({
  get: () => sshInfo.value.pubkeyAuthentication === 'yes',
  set: () => {},
})
const useDNS = computed({
  get: () => sshInfo.value.useDNS === 'yes',
  set: () => {},
})

// 日志
const logLoading = ref(false)
const sshLogs = ref<any[]>([])
const logTotal = ref(0)
const logPage = ref(1)
const logPageSize = ref(20)
const logStatus = ref('all')

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

watch(activeTab, (val) => {
  if (val === 'log' && sshLogs.value.length === 0) loadSSHLog()
})

onMounted(() => loadSSH())
</script>

<style lang="scss" scoped>
.ssh-page { height: 100%; }

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
}

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
</style>
