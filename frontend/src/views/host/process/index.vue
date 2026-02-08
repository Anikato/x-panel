<template>
  <div class="process-page">
    <div class="page-header">
      <h3>{{ $t('process.title') }}</h3>
      <div class="header-actions">
        <el-radio-group v-model="activeTab" size="small">
          <el-radio-button value="process">{{ $t('process.title') }}</el-radio-button>
          <el-radio-button value="network">{{ $t('process.network') }}</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <!-- 进程列表 -->
    <template v-if="activeTab === 'process'">
      <div class="toolbar">
        <el-input v-model="searchName" :placeholder="$t('process.name')" prefix-icon="Search" size="small" clearable class="search-input" @input="loadProcesses" />
        <el-input v-model="searchUser" :placeholder="$t('process.user')" prefix-icon="User" size="small" clearable class="search-input" @input="loadProcesses" />
        <el-select v-model="searchStatus" :placeholder="$t('process.status')" size="small" clearable @change="loadProcesses" style="width: 120px">
          <el-option label="运行中" value="running" />
          <el-option label="睡眠" value="sleeping" />
          <el-option label="已停止" value="stopped" />
          <el-option label="僵尸" value="zombie" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="loadProcesses" :loading="loading">{{ $t('commons.refresh') }}</el-button>
      </div>

      <el-table :data="processes" size="small" v-loading="loading" max-height="600" default-sort="{ prop: 'cpuPercent', order: 'descending' }">
        <el-table-column prop="pid" label="PID" width="80" sortable />
        <el-table-column prop="name" :label="$t('process.name')" min-width="140" show-overflow-tooltip />
        <el-table-column prop="username" :label="$t('process.user')" width="100" show-overflow-tooltip />
        <el-table-column prop="cpuPercent" :label="$t('process.cpu')" width="100" sortable>
          <template #default="{ row }">
            <span :class="{ 'text-danger': row.cpuPercent > 80 }">{{ row.cpuPercent?.toFixed(1) }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="memPercent" :label="$t('process.mem')" width="100" sortable>
          <template #default="{ row }">
            <span :class="{ 'text-danger': row.memPercent > 80 }">{{ row.memPercent?.toFixed(1) }}%</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('process.memRss')" width="100">
          <template #default="{ row }">{{ formatBytes(row.memRSS) }}</template>
        </el-table-column>
        <el-table-column prop="status" :label="$t('process.status')" width="90">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="numThreads" :label="$t('process.threads')" width="70" />
        <el-table-column prop="cmdLine" :label="$t('process.command')" min-width="200" show-overflow-tooltip />
        <el-table-column :label="$t('commons.actions')" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" size="small" @click="handleKill(row)">{{ $t('process.kill') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="process-count">{{ $t('monitor.total') }}: {{ processes.length }}</div>
    </template>

    <!-- 网络连接 -->
    <template v-if="activeTab === 'network'">
      <div class="toolbar">
        <el-button size="small" :icon="Refresh" @click="loadConnections" :loading="connLoading">{{ $t('commons.refresh') }}</el-button>
      </div>
      <el-table :data="connections" size="small" v-loading="connLoading" max-height="600">
        <el-table-column prop="protocol" :label="$t('process.protocol')" width="80" />
        <el-table-column :label="$t('process.localAddr')" min-width="180">
          <template #default="{ row }">{{ row.localAddr }}:{{ row.localPort }}</template>
        </el-table-column>
        <el-table-column :label="$t('process.remoteAddr')" min-width="180">
          <template #default="{ row }">
            {{ row.remoteAddr ? `${row.remoteAddr}:${row.remotePort}` : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="$t('process.connStatus')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'LISTEN' ? 'success' : 'info'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="pid" label="PID" width="80" />
        <el-table-column prop="name" :label="$t('process.name')" min-width="120" show-overflow-tooltip />
      </el-table>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { searchProcess, stopProcess, getConnections } from '@/api/modules/process'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const activeTab = ref('process')
const loading = ref(false)
const processes = ref<any[]>([])
const searchName = ref('')
const searchUser = ref('')
const searchStatus = ref('')

const connLoading = ref(false)
const connections = ref<any[]>([])

const loadProcesses = async () => {
  loading.value = true
  try {
    const res = await searchProcess({
      name: searchName.value,
      username: searchUser.value,
      status: searchStatus.value,
      sortBy: 'cpu',
    })
    processes.value = res.data || []
  } catch { processes.value = [] }
  finally { loading.value = false }
}

const loadConnections = async () => {
  connLoading.value = true
  try {
    const res = await getConnections()
    connections.value = res.data || []
  } catch { connections.value = [] }
  finally { connLoading.value = false }
}

const handleKill = async (row: any) => {
  await ElMessageBox.confirm(
    t('process.killConfirm', { pid: row.pid, name: row.name }),
    t('commons.tip'),
    { type: 'warning' },
  )
  try {
    await stopProcess({ pid: row.pid, signal: 'kill' })
    ElMessage.success(t('commons.success'))
    loadProcesses()
  } catch { /* handled */ }
}

const statusType = (s: string) => {
  const map: Record<string, string> = { running: 'success', sleeping: 'info', stopped: 'warning', zombie: 'danger' }
  return (map[s] || 'info') as any
}

const statusLabel = (s: string) => {
  const map: Record<string, string> = { running: '运行', sleeping: '睡眠', stopped: '停止', zombie: '僵尸' }
  return map[s] || s
}

const formatBytes = (bytes?: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

watch(activeTab, (val) => {
  if (val === 'network' && connections.value.length === 0) loadConnections()
})

onMounted(() => loadProcesses())
</script>

<style lang="scss" scoped>
.process-page {
  height: 100%;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;

  .search-input { width: 180px; }
}

.process-count {
  margin-top: 8px;
  font-size: 12px;
  color: var(--xp-text-muted);
  text-align: right;
}

.text-danger { color: var(--xp-danger); }
</style>
