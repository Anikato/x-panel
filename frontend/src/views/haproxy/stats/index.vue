<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('haproxy.stats') }}</h3>
      <div>
        <el-switch v-model="autoRefresh" :active-text="$t('haproxy.autoRefresh')" @change="toggleAutoRefresh" />
        <el-button style="margin-left: 12px;" @click="loadAll">
          <el-icon><Refresh /></el-icon>{{ $t('commons.refresh') }}
        </el-button>
        <el-button type="warning" @click="clearAll" :disabled="!hasData">{{ $t('haproxy.clearCounters') }}</el-button>
      </div>
    </div>

    <!-- 未安装提示 -->
    <el-alert v-if="notInstalledError" type="warning" :closable="false" show-icon style="margin-bottom: 16px;">
      <template #title>
        {{ $t('haproxy.notInstalled') }}
      </template>
      <template #default>
        {{ $t('haproxy.pleaseInstallFirst') }}
        <el-button type="primary" size="small" style="margin-left: 12px;" @click="goToStatus">
          {{ $t('haproxy.goToInstall') }}
        </el-button>
      </template>
    </el-alert>

    <el-row :gutter="16" v-loading="loading">
      <el-col :span="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-label">{{ $t('haproxy.currentConn') }}</div>
          <div class="metric-value">{{ infoMap.CurrConns || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-label">{{ $t('haproxy.totalConn') }}</div>
          <div class="metric-value">{{ infoMap.CumConns || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-label">{{ $t('haproxy.currentRate') }}</div>
          <div class="metric-value">{{ infoMap.ConnRate || '-' }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="metric-card">
          <div class="metric-label">{{ $t('haproxy.uptime') }}</div>
          <div class="metric-value">{{ infoMap.Uptime || '-' }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" style="margin-top: 16px;">
      <template #header><span>{{ $t('haproxy.frontends') }}</span></template>
      <el-table :data="stats.frontends || []" stripe size="small">
        <el-table-column prop="name" :label="$t('haproxy.proxyName')" min-width="140" />
        <el-table-column :label="$t('haproxy.svStatus')" width="100">
          <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="curConns" :label="$t('haproxy.scur')" width="80" />
        <el-table-column prop="maxConns" :label="$t('haproxy.smax')" width="80" />
        <el-table-column prop="totalConns" :label="$t('haproxy.stot')" width="100" />
        <el-table-column :label="$t('haproxy.bin')" width="110"><template #default="{ row }">{{ formatBytes(row.bytesIn) }}</template></el-table-column>
        <el-table-column :label="$t('haproxy.bout')" width="110"><template #default="{ row }">{{ formatBytes(row.bytesOut) }}</template></el-table-column>
        <el-table-column prop="reqRate" :label="$t('haproxy.reqRate')" width="100" />
        <el-table-column prop="totalReq" :label="$t('haproxy.totalReq')" width="120" />
      </el-table>
    </el-card>

    <el-card shadow="never" style="margin-top: 16px;">
      <template #header><span>{{ $t('haproxy.backends') }}</span></template>
      <el-table :data="stats.backends || []" stripe size="small">
        <el-table-column prop="name" :label="$t('haproxy.proxyName')" min-width="140" />
        <el-table-column :label="$t('haproxy.svStatus')" width="100">
          <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="actServers" :label="$t('haproxy.actServers')" width="80" />
        <el-table-column prop="bckServers" :label="$t('haproxy.bckServers')" width="80" />
        <el-table-column prop="totalServers" :label="$t('haproxy.totalServers')" width="90" />
        <el-table-column prop="curConns" :label="$t('haproxy.scur')" width="80" />
        <el-table-column prop="totalConns" :label="$t('haproxy.stot')" width="100" />
        <el-table-column :label="$t('haproxy.bin')" width="110"><template #default="{ row }">{{ formatBytes(row.bytesIn) }}</template></el-table-column>
        <el-table-column :label="$t('haproxy.bout')" width="110"><template #default="{ row }">{{ formatBytes(row.bytesOut) }}</template></el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" style="margin-top: 16px;">
      <template #header><span>{{ $t('haproxy.serversRuntime') }}</span></template>
      <el-table :data="stats.servers || []" stripe size="small">
        <el-table-column prop="backend" :label="$t('haproxy.proxyName')" min-width="130" />
        <el-table-column prop="name" :label="$t('haproxy.svName')" min-width="130" />
        <el-table-column :label="$t('haproxy.svStatus')" width="110">
          <template #default="{ row }"><el-tag size="small" :type="statusType(row.status)">{{ row.status }}</el-tag></template>
        </el-table-column>
        <el-table-column prop="weight" :label="$t('haproxy.serverWeight')" width="80" />
        <el-table-column prop="curConns" :label="$t('haproxy.scur')" width="80" />
        <el-table-column prop="totalConns" :label="$t('haproxy.stot')" width="100" />
        <el-table-column :label="$t('haproxy.bin')" width="110"><template #default="{ row }">{{ formatBytes(row.bytesIn) }}</template></el-table-column>
        <el-table-column :label="$t('haproxy.bout')" width="110"><template #default="{ row }">{{ formatBytes(row.bytesOut) }}</template></el-table-column>
        <el-table-column prop="checkStatus" :label="$t('haproxy.checkStatus')" min-width="120" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { getHAProxyStats, getHAProxyRuntimeInfo, clearHAProxyCounters } from '@/api/modules/haproxy'

const { t } = useI18n()
const router = useRouter()
const loading = ref(false)
const stats = ref<any>({ frontends: [], backends: [], servers: [] })
const infoRaw = ref('')
const autoRefresh = ref(false)
const notInstalledError = ref(false)
let timer: any = null

const hasData = computed(() => {
  return stats.value.frontends?.length > 0 || stats.value.backends?.length > 0 || stats.value.servers?.length > 0
})

const infoMap = computed(() => {
  const m: Record<string, string> = {}
  ;(infoRaw.value || '').split('\n').forEach((line) => {
    const idx = line.indexOf(':')
    if (idx > 0) {
      m[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
    }
  })
  return m
})

const statusType = (status: string) => {
  if (!status) return 'info'
  const s = status.toUpperCase()
  if (s.startsWith('UP')) return 'success'
  if (s.startsWith('DOWN')) return 'danger'
  if (s === 'OPEN') return 'primary'
  if (s.startsWith('NOLB') || s.startsWith('MAINT')) return 'warning'
  return 'info'
}

const formatBytes = (n: number) => {
  if (!n || isNaN(n)) return '0'
  const units = ['B', 'K', 'M', 'G', 'T']
  let i = 0
  let v = Number(n)
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return v.toFixed(i === 0 ? 0 : 2) + units[i]
}

const loadAll = async () => {
  loading.value = true
  notInstalledError.value = false
  try {
    const [s, i] = await Promise.all([getHAProxyStats(), getHAProxyRuntimeInfo()])
    stats.value = s.data || { frontends: [], backends: [], servers: [] }
    infoRaw.value = i.data?.raw || ''
  } catch (err: any) {
    // 检查是否是未安装错误
    if (err?.message?.includes('未安装') || err?.message?.includes('not installed')) {
      notInstalledError.value = true
      // 停止自动刷新
      if (autoRefresh.value) {
        autoRefresh.value = false
        toggleAutoRefresh(false)
      }
    }
  } finally {
    loading.value = false
  }
}

const clearAll = async () => {
  await ElMessageBox.confirm(t('haproxy.clearCountersConfirm'), t('commons.warning'), { type: 'warning' })
  try {
    await clearHAProxyCounters()
    ElMessage.success(t('commons.operationSuccess'))
    loadAll()
  } catch (err: any) {
    if (err?.message?.includes('未安装') || err?.message?.includes('not installed')) {
      notInstalledError.value = true
    }
  }
}

const toggleAutoRefresh = (v: boolean) => {
  if (v) {
    timer = setInterval(loadAll, 3000)
  } else if (timer) {
    clearInterval(timer); timer = null
  }
}

const goToStatus = () => {
  router.push('/haproxy/status')
}

onMounted(() => loadAll())
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<style lang="scss" scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.metric-card {
  .metric-label { color: #909399; font-size: 13px; }
  .metric-value { font-size: 24px; font-weight: 600; margin-top: 8px; color: #303133; }
}
</style>
