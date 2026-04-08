<template>
  <div class="process-page">
    <div class="page-header">
      <h3>{{ $t('process.title') }}</h3>
      <div class="header-actions">
        <el-radio-group v-model="activeTab" size="small">
          <el-radio-button value="process">{{ $t('process.title') }}</el-radio-button>
          <el-radio-button value="listen">{{ $t('process.listenPorts') }}</el-radio-button>
          <el-radio-button value="connections">{{ $t('process.activeConns') }}</el-radio-button>
          <el-radio-button value="network">{{ $t('process.allConns') }}</el-radio-button>
        </el-radio-group>
      </div>
    </div>

    <!-- 进程列表 -->
    <template v-if="activeTab === 'process'">
      <div class="toolbar">
        <el-input v-model="searchName" :placeholder="$t('process.name')" prefix-icon="Search" size="small" clearable class="search-input" @input="debouncedLoadProcesses" />
        <el-input v-model="searchUser" :placeholder="$t('process.user')" prefix-icon="User" size="small" clearable class="search-input" @input="debouncedLoadProcesses" />
        <el-select v-model="searchStatus" :placeholder="$t('process.status')" size="small" clearable @change="loadProcesses" style="width: 120px">
          <el-option :label="$t('process.running')" value="running" />
          <el-option :label="$t('process.sleeping')" value="sleeping" />
          <el-option :label="$t('process.stopped')" value="stopped" />
          <el-option :label="$t('process.zombie')" value="zombie" />
        </el-select>
        <el-divider direction="vertical" />
        <el-checkbox v-model="autoRefresh" size="small">{{ $t('process.autoRefresh') }}</el-checkbox>
        <el-select v-if="autoRefresh" v-model="refreshInterval" size="small" style="width: 90px" @change="resetAutoRefresh">
          <el-option label="3s" :value="3000" />
          <el-option label="5s" :value="5000" />
          <el-option label="10s" :value="10000" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="loadProcesses" :loading="loading">{{ $t('commons.refresh') }}</el-button>
      </div>

      <el-table :data="processes" size="small" v-loading="loading" max-height="600" :default-sort="{ prop: 'cpuPercent', order: 'descending' }">
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

    <!-- 监听端口 -->
    <template v-if="activeTab === 'listen'">
      <div class="toolbar">
        <el-input v-model="listenSearch" :placeholder="$t('process.searchPortOrProcess')" prefix-icon="Search" size="small" clearable class="search-input" style="width: 260px" />
        <el-divider direction="vertical" />
        <el-checkbox v-model="connAutoRefresh" size="small">{{ $t('process.autoRefresh') }}</el-checkbox>
        <el-select v-if="connAutoRefresh" v-model="connRefreshInterval" size="small" style="width: 90px" @change="resetConnAutoRefresh">
          <el-option label="3s" :value="3000" />
          <el-option label="5s" :value="5000" />
          <el-option label="10s" :value="10000" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="loadConnections" :loading="connLoading">{{ $t('commons.refresh') }}</el-button>
      </div>
      <div class="stat-cards">
        <div class="stat-card">
          <span class="stat-num accent">{{ listenPorts.length }}</span>
          <span class="stat-label">{{ $t('process.listenPorts') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-num">{{ new Set(listenPorts.map(c => c.name)).size }}</span>
          <span class="stat-label">{{ $t('process.services') }}</span>
        </div>
      </div>
      <el-table :data="filteredListenPorts" size="small" v-loading="connLoading" max-height="520" :default-sort="{ prop: 'localPort', order: 'ascending' }">
        <el-table-column prop="localPort" :label="$t('process.port')" width="100" sortable />
        <el-table-column prop="protocol" :label="$t('process.protocol')" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.protocol === 'tcp6' || row.protocol === 'udp6' ? 'warning' : 'info'" effect="plain">{{ row.protocol }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="localAddr" :label="$t('process.listenAddr')" min-width="160">
          <template #default="{ row }">{{ row.localAddr }}:{{ row.localPort }}</template>
        </el-table-column>
        <el-table-column prop="name" :label="$t('process.processName')" min-width="140" show-overflow-tooltip />
        <el-table-column prop="pid" label="PID" width="80" />
        <el-table-column :label="$t('process.connCount')" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="portConnCount(row.localPort) > 0" size="small" type="success">{{ portConnCount(row.localPort) }}</el-tag>
            <span v-else class="text-muted">0</span>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 活跃连接（按远程 IP 聚合） -->
    <template v-if="activeTab === 'connections'">
      <div class="toolbar">
        <el-input v-model="activeConnSearch" :placeholder="$t('process.searchIPOrPort')" prefix-icon="Search" size="small" clearable class="search-input" style="width: 260px" />
        <el-divider direction="vertical" />
        <el-checkbox v-model="connAutoRefresh" size="small">{{ $t('process.autoRefresh') }}</el-checkbox>
        <el-select v-if="connAutoRefresh" v-model="connRefreshInterval" size="small" style="width: 90px" @change="resetConnAutoRefresh">
          <el-option label="3s" :value="3000" />
          <el-option label="5s" :value="5000" />
          <el-option label="10s" :value="10000" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="loadConnections" :loading="connLoading">{{ $t('commons.refresh') }}</el-button>
      </div>
      <div class="stat-cards">
        <div class="stat-card">
          <span class="stat-num accent">{{ establishedConns.length }}</span>
          <span class="stat-label">{{ $t('process.activeConns') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-num">{{ Object.keys(groupedByIP).length }}</span>
          <span class="stat-label">{{ $t('process.uniqueIPs') }}</span>
        </div>
      </div>
      <el-table :data="filteredGroupedConns" size="small" v-loading="connLoading" max-height="520" :default-sort="{ prop: 'count', order: 'descending' }">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="expand-detail">
              <el-table :data="row.connections" size="small" :show-header="true">
                <el-table-column prop="localPort" :label="$t('process.localPort')" width="100" />
                <el-table-column prop="remotePort" :label="$t('process.remotePort')" width="100" />
                <el-table-column prop="name" :label="$t('process.processName')" min-width="120" show-overflow-tooltip />
                <el-table-column prop="status" :label="$t('process.connStatus')" width="120">
                  <template #default="{ row: sub }">
                    <el-tag size="small" :type="connStatusType(sub.status)">{{ sub.status }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="pid" label="PID" width="80" />
              </el-table>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="remoteAddr" :label="$t('process.remoteIP')" min-width="160" />
        <el-table-column prop="count" :label="$t('process.connCount')" width="110" sortable />
        <el-table-column :label="$t('process.targetPorts')" min-width="200">
          <template #default="{ row }">
            <el-tag v-for="port in row.ports.slice(0, 8)" :key="port" size="small" effect="plain" style="margin: 0 4px 2px 0">{{ port }}</el-tag>
            <span v-if="row.ports.length > 8" class="text-muted">+{{ row.ports.length - 8 }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('process.services')" min-width="160">
          <template #default="{ row }">
            <span>{{ row.processes.join(', ') }}</span>
          </template>
        </el-table-column>
      </el-table>
    </template>

    <!-- 全部网络连接 -->
    <template v-if="activeTab === 'network'">
      <div class="toolbar">
        <el-input v-model="connSearch" :placeholder="$t('process.searchAddrOrProcess')" prefix-icon="Search" size="small" clearable class="search-input" style="width: 220px" />
        <el-select v-model="connStatusFilter" :placeholder="$t('process.connStatus')" size="small" clearable style="width: 140px">
          <el-option v-for="s in connStatusOptions" :key="s" :label="s" :value="s" />
        </el-select>
        <el-select v-model="connProtoFilter" :placeholder="$t('process.protocol')" size="small" clearable style="width: 100px">
          <el-option label="TCP" value="tcp" />
          <el-option label="TCP6" value="tcp6" />
          <el-option label="UDP" value="udp" />
          <el-option label="UDP6" value="udp6" />
        </el-select>
        <el-divider direction="vertical" />
        <el-checkbox v-model="connAutoRefresh" size="small">{{ $t('process.autoRefresh') }}</el-checkbox>
        <el-select v-if="connAutoRefresh" v-model="connRefreshInterval" size="small" style="width: 90px" @change="resetConnAutoRefresh">
          <el-option label="3s" :value="3000" />
          <el-option label="5s" :value="5000" />
          <el-option label="10s" :value="10000" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="loadConnections" :loading="connLoading">{{ $t('commons.refresh') }}</el-button>
      </div>
      <div class="stat-cards">
        <div class="stat-card">
          <span class="stat-num accent">{{ connections.length }}</span>
          <span class="stat-label">{{ $t('monitor.total') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-num" style="color: var(--xp-success)">{{ listenPorts.length }}</span>
          <span class="stat-label">LISTEN</span>
        </div>
        <div class="stat-card">
          <span class="stat-num" style="color: var(--xp-info)">{{ establishedConns.length }}</span>
          <span class="stat-label">ESTABLISHED</span>
        </div>
        <div class="stat-card">
          <span class="stat-num" style="color: var(--xp-warning)">{{ connections.filter(c => c.status === 'TIME_WAIT').length }}</span>
          <span class="stat-label">TIME_WAIT</span>
        </div>
      </div>
      <el-table :data="filteredConnections" size="small" v-loading="connLoading" max-height="480">
        <el-table-column prop="protocol" :label="$t('process.protocol')" width="80" />
        <el-table-column :label="$t('process.localAddr')" min-width="180">
          <template #default="{ row }">{{ row.localAddr }}:{{ row.localPort }}</template>
        </el-table-column>
        <el-table-column :label="$t('process.remoteAddr')" min-width="180">
          <template #default="{ row }">
            {{ row.remoteAddr ? `${row.remoteAddr}:${row.remotePort}` : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" :label="$t('process.connStatus')" width="130">
          <template #default="{ row }">
            <el-tag size="small" :type="connStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="pid" label="PID" width="80" />
        <el-table-column prop="name" :label="$t('process.name')" min-width="120" show-overflow-tooltip />
      </el-table>
      <div class="process-count">{{ $t('process.filtered') }}: {{ filteredConnections.length }} / {{ connections.length }}</div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { searchProcess, stopProcess, getConnections } from '@/api/modules/process'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { ProcessInfo, NetworkConn } from '@/api/interface'

const { t } = useI18n()
const activeTab = ref('process')
const loading = ref(false)
const processes = ref<ProcessInfo[]>([])
const searchName = ref('')
const searchUser = ref('')
const searchStatus = ref('')

const autoRefresh = ref(false)
const refreshInterval = ref(5000)
let processTimer: ReturnType<typeof setInterval> | null = null

const connLoading = ref(false)
const connections = ref<NetworkConn[]>([])
const connAutoRefresh = ref(false)
const connRefreshInterval = ref(5000)
let connTimer: ReturnType<typeof setInterval> | null = null

const connSearch = ref('')
const connStatusFilter = ref('')
const connProtoFilter = ref('')
const listenSearch = ref('')
const activeConnSearch = ref('')

let debounceTimer: ReturnType<typeof setTimeout> | null = null
const debouncedLoadProcesses = () => {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(loadProcesses, 300)
}

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

const listenPorts = computed(() =>
  connections.value.filter(c => c.status === 'LISTEN')
)

const establishedConns = computed(() =>
  connections.value.filter(c => c.status === 'ESTABLISHED')
)

const filteredListenPorts = computed(() => {
  const q = listenSearch.value.toLowerCase().trim()
  if (!q) return listenPorts.value
  return listenPorts.value.filter(c =>
    String(c.localPort).includes(q) ||
    c.name?.toLowerCase().includes(q) ||
    c.localAddr?.toLowerCase().includes(q) ||
    String(c.pid).includes(q)
  )
})

const portConnCount = (port: number) =>
  establishedConns.value.filter(c => c.localPort === port).length

const groupedByIP = computed(() => {
  const map: Record<string, { remoteAddr: string; count: number; ports: number[]; processes: string[]; connections: NetworkConn[] }> = {}
  for (const c of establishedConns.value) {
    if (!c.remoteAddr) continue
    if (!map[c.remoteAddr]) {
      map[c.remoteAddr] = { remoteAddr: c.remoteAddr, count: 0, ports: [], processes: [], connections: [] }
    }
    const g = map[c.remoteAddr]
    g.count++
    g.connections.push(c)
    if (!g.ports.includes(c.localPort)) g.ports.push(c.localPort)
    if (c.name && !g.processes.includes(c.name)) g.processes.push(c.name)
  }
  return map
})

const filteredGroupedConns = computed(() => {
  const arr = Object.values(groupedByIP.value)
  const q = activeConnSearch.value.toLowerCase().trim()
  if (!q) return arr
  return arr.filter(g =>
    g.remoteAddr.includes(q) ||
    g.ports.some(p => String(p).includes(q)) ||
    g.processes.some(p => p.toLowerCase().includes(q))
  )
})

const connStatusOptions = computed(() => {
  const set = new Set(connections.value.map(c => c.status).filter(Boolean))
  return Array.from(set).sort()
})

const filteredConnections = computed(() => {
  let data = connections.value
  if (connStatusFilter.value) {
    data = data.filter(c => c.status === connStatusFilter.value)
  }
  if (connProtoFilter.value) {
    data = data.filter(c => c.protocol === connProtoFilter.value)
  }
  const q = connSearch.value.toLowerCase().trim()
  if (q) {
    data = data.filter(c =>
      c.localAddr?.toLowerCase().includes(q) ||
      String(c.localPort).includes(q) ||
      c.remoteAddr?.toLowerCase().includes(q) ||
      String(c.remotePort).includes(q) ||
      c.name?.toLowerCase().includes(q) ||
      String(c.pid).includes(q)
    )
  }
  return data
})

const handleKill = async (row: ProcessInfo) => {
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

const connStatusType = (s: string) => {
  if (s === 'LISTEN') return 'success'
  if (s === 'ESTABLISHED') return 'primary'
  if (s === 'TIME_WAIT' || s === 'CLOSE_WAIT') return 'warning'
  if (s === 'SYN_SENT' || s === 'SYN_RECV') return 'info'
  return 'info'
}

const statusType = (s: string) => {
  const map: Record<string, string> = { running: 'success', sleeping: 'info', stopped: 'warning', zombie: 'danger' }
  return (map[s] || 'info') as '' | 'success' | 'info' | 'warning' | 'danger'
}

const statusLabel = (s: string) => {
  const map: Record<string, () => string> = {
    running: () => t('process.running'),
    sleeping: () => t('process.sleeping'),
    stopped: () => t('process.stopped'),
    zombie: () => t('process.zombie'),
  }
  return map[s] ? map[s]() : s
}

const formatBytes = (bytes?: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

const resetAutoRefresh = () => {
  if (processTimer) { clearInterval(processTimer); processTimer = null }
  if (autoRefresh.value) {
    processTimer = setInterval(loadProcesses, refreshInterval.value)
  }
}

const resetConnAutoRefresh = () => {
  if (connTimer) { clearInterval(connTimer); connTimer = null }
  if (connAutoRefresh.value) {
    connTimer = setInterval(loadConnections, connRefreshInterval.value)
  }
}

watch(autoRefresh, resetAutoRefresh)
watch(connAutoRefresh, resetConnAutoRefresh)

watch(activeTab, (val) => {
  if (['listen', 'connections', 'network'].includes(val) && connections.value.length === 0) {
    loadConnections()
  }
})

onMounted(() => loadProcesses())

onUnmounted(() => {
  if (processTimer) clearInterval(processTimer)
  if (connTimer) clearInterval(connTimer)
  if (debounceTimer) clearTimeout(debounceTimer)
})
</script>

<style lang="scss" scoped>
.process-page {
  height: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  .search-input { width: 180px; }
}

.stat-cards {
  display: flex;
  gap: 12px;
  margin-bottom: 14px;
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 20px;
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  min-width: 100px;
}

.stat-num {
  font-size: 22px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--xp-text-primary);
  &.accent { color: var(--xp-accent); }
}

.stat-label {
  font-size: 12px;
  color: var(--xp-text-muted);
  margin-top: 2px;
}

.process-count {
  margin-top: 8px;
  font-size: 12px;
  color: var(--xp-text-muted);
  text-align: right;
}

.text-danger { color: var(--xp-danger); }
.text-muted { color: var(--xp-text-muted); font-size: 12px; }

.expand-detail {
  padding: 8px 16px 8px 48px;
}
</style>
