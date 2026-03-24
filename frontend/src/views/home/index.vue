<template>
  <div class="dashboard">
    <!-- 顶部系统信息 -->
    <div class="dashboard-header">
      <div class="system-identity">
        <div class="system-logo">
          <el-icon :size="28"><Monitor /></el-icon>
        </div>
        <div class="system-meta">
          <h2 class="hostname">{{ stats.host?.hostname || '...' }}</h2>
          <div class="system-tags">
            <el-tag size="small" effect="dark" round>{{ panelVersion }}</el-tag>
            <el-tag size="small" effect="plain" round type="info">
              {{ stats.host?.platform }} {{ stats.host?.platformVersion }}
            </el-tag>
            <el-tag size="small" effect="plain" round type="info">
              {{ stats.host?.kernelArch }}
            </el-tag>
            <el-tag v-if="stats.host?.virtualization" size="small" effect="plain" round type="warning">
              {{ stats.host?.virtualization }}
            </el-tag>
          </div>
        </div>
      </div>
      <div class="header-right">
        <div class="uptime-display">
          <el-icon><Clock /></el-icon>
          <span>{{ t('home.uptime') }}: {{ formatUptime(stats.uptime) }}</span>
        </div>
        <el-button-group size="small">
          <el-button type="warning" plain @click="handleRestartPanel">
            <el-icon><RefreshRight /></el-icon>{{ t('home.restartPanel') }}
          </el-button>
          <el-button type="danger" plain @click="handleRebootServer">
            <el-icon><SwitchButton /></el-icon>{{ t('home.rebootServer') }}
          </el-button>
        </el-button-group>
        <el-button text :icon="Refresh" @click="loadStats" :loading="loading" circle />
      </div>
    </div>

    <!-- 系统信息 + 网络信息 合并 -->
    <el-row :gutter="16" class="info-row">
      <el-col :xs="24" :lg="14">
        <el-card shadow="never" class="info-card">
          <template #header>
            <div class="card-header-row">
              <el-icon><Monitor /></el-icon>
              <span>{{ t('home.systemInfo') }}</span>
            </div>
          </template>
          <div class="sys-info-grid">
            <div class="sys-info-item" v-for="item in sysInfoItems" :key="item.label">
              <span class="sys-info-label">{{ item.label }}</span>
              <span class="sys-info-value">
                {{ item.value }}
                <el-icon class="copy-btn" @click="copyText(item.value)" v-if="item.value && item.value !== '-'"><CopyDocument /></el-icon>
              </span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="10">
        <el-card shadow="never" class="info-card" v-if="stats.host?.interfaces?.length || stats.host?.publicIPv4">
          <template #header>
            <div class="card-header-row">
              <el-icon><Connection /></el-icon>
              <span>{{ t('home.networkInfo') }}</span>
            </div>
          </template>
          <div class="net-info-list">
            <div class="net-info-row" v-if="stats.host?.publicIPv4">
              <span class="net-label">{{ t('home.publicIPv4') }}</span>
              <span class="net-value highlight">
                {{ stats.host.publicIPv4 }}
                <el-icon class="copy-btn" @click="copyText(stats.host.publicIPv4)"><CopyDocument /></el-icon>
              </span>
            </div>
            <div class="net-info-row" v-if="stats.host?.publicIPv6">
              <span class="net-label">{{ t('home.publicIPv6') }}</span>
              <span class="net-value">
                {{ stats.host.publicIPv6 }}
                <el-icon class="copy-btn" @click="copyText(stats.host.publicIPv6)"><CopyDocument /></el-icon>
              </span>
            </div>
            <template v-for="iface in stats.host?.interfaces" :key="iface.name">
              <div class="net-info-row" v-for="ip in iface.ipv4" :key="ip">
                <span class="net-label">
                  <el-tag size="small" :type="iface.status === 'up' ? 'success' : 'info'" effect="plain" round>{{ iface.name }}</el-tag>
                </span>
                <span class="net-value">
                  {{ ip }}
                  <el-icon class="copy-btn" @click="copyText(ip.split('/')[0])"><CopyDocument /></el-icon>
                </span>
              </div>
            </template>
            <div class="net-info-row" v-if="stats.host?.dnsServers?.length">
              <span class="net-label">DNS</span>
              <span class="net-value">
                {{ stats.host.dnsServers.join(', ') }}
                <el-icon class="copy-btn" @click="copyText(stats.host.dnsServers.join(', '))"><CopyDocument /></el-icon>
              </span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 资源占用 -->
    <div class="resource-section">
      <h3 class="section-title">{{ t('home.resourceUsage') }}</h3>
      <el-row :gutter="16">
        <!-- CPU -->
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="resource-card">
            <div class="resource-header">
              <div class="resource-icon cpu-icon">
                <el-icon :size="18"><Cpu /></el-icon>
              </div>
              <span class="resource-name">CPU</span>
              <span class="resource-pct" :class="pctClass(stats.cpu?.usagePercent)">
                {{ (stats.cpu?.usagePercent ?? 0).toFixed(1) }}%
              </span>
            </div>
            <div class="resource-bar-wrapper">
              <div class="resource-bar" :style="barStyle(stats.cpu?.usagePercent, 'cpu')"></div>
            </div>
            <div class="resource-detail">
              {{ stats.cpu?.cores }} {{ t('home.physical') }} / {{ stats.cpu?.logicalCores }} {{ t('home.logical') }}
            </div>
          </div>
        </el-col>
        <!-- 内存 -->
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="resource-card">
            <div class="resource-header">
              <div class="resource-icon mem-icon">
                <el-icon :size="18"><Coin /></el-icon>
              </div>
              <span class="resource-name">{{ t('home.memory') }}</span>
              <span class="resource-pct" :class="pctClass(stats.memory?.usedPercent)">
                {{ (stats.memory?.usedPercent ?? 0).toFixed(1) }}%
              </span>
            </div>
            <div class="resource-bar-wrapper">
              <div class="resource-bar" :style="barStyle(stats.memory?.usedPercent, 'mem')"></div>
            </div>
            <div class="resource-detail">
              {{ formatBytes(stats.memory?.used) }} / {{ formatBytes(stats.memory?.total) }}
            </div>
            <div class="resource-sub" v-if="(stats.memory?.swapTotal ?? 0) > 0">
              Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }}
              ({{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%)
            </div>
          </div>
        </el-col>
        <!-- 负载 -->
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="resource-card">
            <div class="resource-header">
              <div class="resource-icon load-icon">
                <el-icon :size="18"><Odometer /></el-icon>
              </div>
              <span class="resource-name">{{ t('home.load') }}</span>
              <span class="resource-pct" :class="pctClass(loadPercent)">
                {{ loadPercent.toFixed(0) }}%
              </span>
            </div>
            <div class="resource-bar-wrapper">
              <div class="resource-bar" :style="barStyle(loadPercent, 'load')"></div>
            </div>
            <div class="resource-detail load-detail">
              <span>1m: {{ stats.load?.load1?.toFixed(2) || '-' }}</span>
              <span>5m: {{ stats.load?.load5?.toFixed(2) || '-' }}</span>
              <span>15m: {{ stats.load?.load15?.toFixed(2) || '-' }}</span>
            </div>
          </div>
        </el-col>
        <!-- 网络 -->
        <el-col :xs="24" :sm="12" :lg="6">
          <div class="resource-card">
            <div class="resource-header">
              <div class="resource-icon net-icon">
                <el-icon :size="18"><Connection /></el-icon>
              </div>
              <span class="resource-name">{{ t('home.network') }}</span>
            </div>
            <div class="net-stats">
              <div class="net-row" v-for="nic in mainNics" :key="nic.name">
                <span class="net-nic">{{ nic.name }}</span>
                <span class="net-speed-up">↑ {{ formatSpeed(nic.speedUp) }}</span>
                <span class="net-speed-down">↓ {{ formatSpeed(nic.speedDown) }}</span>
              </div>
            </div>
            <div class="resource-detail">
              {{ t('home.totalTraffic') }}: ↑ {{ formatBytes(stats.network?.bytesSent) }}  ↓ {{ formatBytes(stats.network?.bytesRecv) }}
            </div>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- 磁盘使用 -->
    <div class="disk-section" v-if="stats.disks?.length">
      <h3 class="section-title">{{ t('home.diskUsage') }}</h3>
      <el-row :gutter="16">
        <el-col :xs="24" :sm="12" :lg="8" v-for="disk in stats.disks" :key="disk.mountPoint">
          <div class="disk-card">
            <div class="disk-header">
              <div class="disk-icon">
                <el-icon :size="16"><Box /></el-icon>
              </div>
              <span class="disk-mount">{{ disk.mountPoint }}</span>
              <span class="disk-pct" :class="pctClass(disk.usedPercent)">
                {{ disk.usedPercent.toFixed(1) }}%
              </span>
            </div>
            <div class="resource-bar-wrapper">
              <div class="resource-bar" :style="barStyle(disk.usedPercent, 'disk')"></div>
            </div>
            <div class="disk-detail">
              <span>{{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</span>
              <span class="disk-fs">{{ disk.device }} · {{ disk.fsType }}</span>
            </div>
            <div class="disk-inode" v-if="disk.inodesTotal > 0">
              Inode: {{ disk.inodesPercent.toFixed(0) }}%
              ({{ formatNumber(disk.inodesUsed) }} / {{ formatNumber(disk.inodesTotal) }})
            </div>
          </div>
        </el-col>
      </el-row>
    </div>

    <!-- 快速入口 + Top 进程 -->
    <el-row :gutter="16" class="bottom-section">
      <el-col :xs="24" :lg="10">
        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="section-card-header">
              <el-icon><Compass /></el-icon>
              <span>{{ t('home.quickEntry') }}</span>
            </div>
          </template>
          <div class="quick-grid">
            <div
              v-for="entry in quickEntries"
              :key="entry.path"
              class="quick-item"
              @click="router.push(entry.path)"
            >
              <div class="quick-icon">
                <el-icon :size="20"><component :is="entry.icon" /></el-icon>
              </div>
              <span class="quick-label">{{ entry.title }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="14">
        <el-card shadow="never" class="section-card">
          <template #header>
            <div class="section-card-header">
              <el-icon><DataLine /></el-icon>
              <span>{{ t('home.topProcess') }}</span>
            </div>
          </template>
          <el-table :data="stats.topProcess || []" size="small" :show-header="true" stripe>
            <el-table-column prop="pid" label="PID" width="70" />
            <el-table-column prop="name" :label="t('home.processName')" min-width="140" show-overflow-tooltip />
            <el-table-column label="CPU %" width="100" align="right">
              <template #default="{ row }">
                <span :class="row.cpuPercent > 50 ? 'text-danger' : ''">
                  {{ row.cpuPercent.toFixed(1) }}%
                </span>
              </template>
            </el-table-column>
            <el-table-column :label="t('home.memoryUsage')" width="100" align="right">
              <template #default="{ row }">
                {{ formatBytes(row.memRss) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getSystemStats } from '@/api/modules/monitor'
import { getCurrentVersion } from '@/api/modules/upgrade'
import { rebootServer, shutdownServer, restartPanel } from '@/api/modules/setting'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { SystemStats, HostInfo } from '@/api/interface'
import {
  Monitor, Clock, Refresh, Cpu, Coin, Odometer, Connection,
  Box, Compass, DataLine, CopyDocument, SwitchButton, RefreshRight,
} from '@element-plus/icons-vue'

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const stats = ref<Partial<SystemStats>>({})
const panelVersion = ref('...')
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  loading.value = true
  try {
    const res = await getSystemStats()
    stats.value = res.data || {}
  } catch { /* handled by interceptor */ }
  finally { loading.value = false }
}

const fetchVersion = async () => {
  try {
    const res = await getCurrentVersion()
    if (res.data) {
      panelVersion.value = res.data.version === 'dev' ? 'dev' : res.data.version
    }
  } catch {
    panelVersion.value = '-'
  }
}

// 系统信息项（带复制按钮）
const sysInfoItems = computed(() => {
  const h = stats.value.host ?? ({} as Partial<HostInfo>)
  return [
    { label: t('home.hostname'), value: h.hostname || '-' },
    { label: t('home.os'), value: `${h.platform || ''} ${h.platformVersion || ''}`.trim() || '-' },
    { label: t('home.kernel'), value: h.kernelVersion || '-' },
    { label: t('home.arch'), value: h.kernelArch || '-' },
    { label: t('home.timezone'), value: h.timezone || '-' },
    { label: t('home.virtualization'), value: h.virtualization || '-' },
    { label: t('home.cpuModel'), value: stats.value.cpu?.modelName || '-' },
    { label: t('home.cpuCores'), value: stats.value.cpu ? `${stats.value.cpu.cores} ${t('home.physical')} / ${stats.value.cpu.logicalCores} ${t('home.logical')}` : '-' },
    { label: t('home.totalMemory'), value: formatBytes(stats.value.memory?.total) },
  ]
})

const loadPercent = computed(() => {
  const cores = stats.value.cpu?.logicalCores || 1
  const load1 = stats.value.load?.load1 || 0
  return Math.min((load1 / cores) * 100, 100)
})

const mainNics = computed(() => {
  return (stats.value.netIO || []).filter((n) => n.name !== 'lo').slice(0, 4)
})

const quickEntries = computed(() => [
  { path: '/host/files', title: t('menu.fileManager'), icon: 'FolderOpened' },
  { path: '/terminal', title: t('menu.terminal'), icon: 'Monitor' },
  { path: '/website/nginx', title: t('menu.nginx'), icon: 'Platform' },
  { path: '/website/ssl', title: t('menu.ssl'), icon: 'Lock' },
  { path: '/host/firewall', title: t('menu.firewall'), icon: 'Shield' },
  { path: '/host/process', title: t('menu.processManage'), icon: 'DataAnalysis' },
  { path: '/setting', title: t('menu.setting'), icon: 'Setting' },
  { path: '/log/operation', title: t('menu.operationLog'), icon: 'Notebook' },
])

const handleRebootServer = async () => {
  await ElMessageBox.confirm(t('home.rebootConfirm'), t('commons.tip'), { type: 'warning', confirmButtonText: t('home.rebootServer') })
  await rebootServer()
  ElMessage.success(t('home.rebootSuccess'))
}

const handleShutdownServer = async () => {
  await ElMessageBox.confirm(t('home.shutdownConfirm'), t('commons.tip'), { type: 'error', confirmButtonText: t('home.shutdownServer') })
  await shutdownServer()
  ElMessage.success(t('home.shutdownSuccess'))
}

const handleRestartPanel = async () => {
  await ElMessageBox.confirm(t('home.restartPanelConfirm'), t('commons.tip'), { type: 'warning' })
  await restartPanel()
  ElMessage.success(t('home.restartPanelSuccess'))
}

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('commons.copy') + ' ✓')
  } catch {
    ElMessage.error('Copy failed')
  }
}

const getCS = (v: string) => getComputedStyle(document.documentElement).getPropertyValue(v).trim()
const THEME_COLORS = {
  danger: '#ef4444', warning: '#f59e0b',
  cpu: '', mem: '#818cf8', load: '#34d399', disk: '#60a5fa', net: '#a78bfa',
}

const getBarColor = (pct: number, type: string): string => {
  if (pct >= 90) return THEME_COLORS.danger
  if (pct >= 70) return THEME_COLORS.warning
  if (!THEME_COLORS.cpu) THEME_COLORS.cpu = getCS('--xp-accent') || '#22d3ee'
  return (THEME_COLORS as Record<string, string>)[type] || THEME_COLORS.cpu
}

const barStyle = (pct?: number, type = 'cpu') => {
  const v = Math.min(pct || 0, 100)
  const color = getBarColor(v, type)
  return {
    width: `${v}%`,
    background: `linear-gradient(90deg, ${color}dd, ${color})`,
    boxShadow: `0 0 8px ${color}40`,
  }
}

const pctClass = (pct?: number) => {
  const v = pct || 0
  if (v >= 90) return 'pct-danger'
  if (v >= 70) return 'pct-warning'
  return 'pct-normal'
}

const formatBytes = (bytes?: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

const formatSpeed = (bytesPerSec?: number) => {
  if (!bytesPerSec || bytesPerSec < 0) return '0 B/s'
  if (bytesPerSec < 1024) return bytesPerSec.toFixed(0) + ' B/s'
  if (bytesPerSec < 1024 * 1024) return (bytesPerSec / 1024).toFixed(1) + ' KB/s'
  return (bytesPerSec / 1024 / 1024).toFixed(2) + ' MB/s'
}

const formatUptime = (seconds?: number) => {
  if (!seconds) return '-'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const parts = []
  if (d > 0) parts.push(`${d} ${t('monitor.days')}`)
  if (h > 0) parts.push(`${h} ${t('monitor.hours')}`)
  parts.push(`${m} ${t('monitor.minutes')}`)
  return parts.join(' ')
}

const formatNumber = (n?: number) => {
  if (!n) return '0'
  return n.toLocaleString()
}

onMounted(() => {
  fetchVersion()
  loadStats()
  timer = setInterval(loadStats, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style lang="scss" scoped>
.dashboard {
  padding: 0;
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px 24px;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-lg);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.15);
  transition: box-shadow 0.3s;

  &:hover {
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  }
}

.system-identity {
  display: flex;
  align-items: center;
  gap: 16px;
}

.system-logo {
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--xp-accent-muted);
  border-radius: 12px;
  color: var(--xp-accent);
}

.system-meta {
  .hostname {
    margin: 0 0 6px 0;
    font-size: 20px;
    font-weight: 700;
    color: var(--xp-text-primary);
    letter-spacing: 0.3px;
  }
  .system-tags {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.uptime-display {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--xp-accent);
  font-weight: 500;
  background: var(--xp-accent-muted);
  padding: 6px 14px;
  border-radius: 20px;
}

/* ===== 信息行 ===== */
.info-row {
  margin-bottom: 20px;
}

.info-card {
  height: 100%;
  margin-bottom: 16px;
}

.card-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: var(--xp-text-primary);
}

.sys-info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px 32px;
}

.sys-info-item {
  display: flex;
  gap: 8px;
  align-items: baseline;
}

.sys-info-label {
  font-size: 12px;
  color: var(--xp-text-muted);
  white-space: nowrap;
  min-width: 70px;
}

.sys-info-value {
  font-size: 13px;
  color: var(--xp-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 4px;
}

.sys-info-item:hover :deep(.copy-btn),
.net-info-row:hover :deep(.copy-btn) {
  opacity: 1;
}

/* ===== 网络信息 ===== */
.net-info-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.net-info-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 0;
}

.net-label {
  font-size: 12px;
  color: var(--xp-text-muted);
  min-width: 80px;
  white-space: nowrap;
}

.net-value {
  font-size: 13px;
  color: var(--xp-text-primary);
  display: flex;
  align-items: center;
  gap: 4px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 12px;

  &.highlight {
    color: var(--xp-accent);
    font-weight: 600;
  }
}

/* ===== 区域标题 ===== */
.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--xp-text-primary);
  margin: 0 0 14px 0;
  padding-left: 10px;
  border-left: 3px solid var(--xp-accent);
}

/* ===== 资源占用 ===== */
.resource-section {
  margin-bottom: 20px;
}

.resource-card {
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  padding: 18px 20px;
  margin-bottom: 16px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  &:hover {
    border-color: var(--xp-accent-muted);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
    transform: translateY(-2px);
  }
}

.resource-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.resource-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}

.cpu-icon { background: var(--xp-accent-muted); color: var(--xp-accent); }
.mem-icon { background: rgba(129, 140, 248, 0.12); color: var(--xp-accent-secondary); }
.load-icon { background: rgba(52, 211, 153, 0.12); color: var(--xp-success); }
.net-icon { background: rgba(167, 139, 250, 0.12); color: var(--xp-color-down); }

.resource-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--xp-text-primary);
  flex: 1;
}

.resource-pct {
  font-size: 18px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.pct-normal { color: var(--xp-accent); }
.pct-warning { color: var(--xp-warning); }
.pct-danger { color: var(--xp-danger); }

.resource-bar-wrapper {
  width: 100%;
  height: 6px;
  background: var(--xp-progress-trail);
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 10px;
}

.resource-bar {
  height: 100%;
  border-radius: 3px;
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1), background 0.4s ease;
  min-width: 2px;
}

.resource-detail {
  font-size: 12px;
  color: var(--xp-text-secondary);
}

.resource-sub {
  font-size: 11px;
  color: var(--xp-text-muted);
  margin-top: 2px;
}

.load-detail {
  display: flex;
  gap: 16px;
}

/* ===== 网络速率 ===== */
.net-stats { margin-bottom: 8px; }

.net-row {
  display: flex;
  align-items: center;
  padding: 3px 0;
  font-size: 12px;
  gap: 12px;
}

.net-nic {
  color: var(--xp-text-secondary);
  font-weight: 500;
  min-width: 60px;
}

.net-speed-up { color: var(--xp-color-up); font-variant-numeric: tabular-nums; }
.net-speed-down { color: var(--xp-color-down); font-variant-numeric: tabular-nums; }

/* ===== 磁盘 ===== */
.disk-section { margin-bottom: 20px; }

.disk-card {
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  padding: 16px 18px;
  margin-bottom: 16px;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  &:hover {
    border-color: var(--xp-accent-muted);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.1);
    transform: translateY(-1px);
  }
}

.disk-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.disk-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  background: rgba(96, 165, 250, 0.12);
  color: var(--xp-info);
}

.disk-mount {
  font-size: 13px;
  font-weight: 600;
  color: var(--xp-text-primary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-pct {
  font-size: 15px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.disk-detail {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: var(--xp-text-secondary);
}

.disk-fs { color: var(--xp-text-muted); }

.disk-inode {
  font-size: 11px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}

/* ===== 底部 ===== */
.bottom-section { margin-bottom: 20px; }
.section-card { margin-bottom: 16px; }

.section-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--xp-text-primary);
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 16px 8px;
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  &:hover {
    border-color: var(--xp-accent-muted);
    background: var(--xp-accent-muted);
    transform: translateY(-3px);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.1);

    .quick-icon {
      color: var(--xp-accent);
      background: var(--xp-accent-muted);
      transform: scale(1.1);
    }

    .quick-label {
      color: var(--xp-accent);
    }
  }

  .quick-icon {
    width: 38px;
    height: 38px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.04);
    border-radius: 10px;
    color: var(--xp-text-secondary);
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .quick-label {
    font-size: 12px;
    color: var(--xp-text-secondary);
    font-weight: 500;
    text-align: center;
    transition: color 0.2s;
  }
}

.text-danger {
  color: var(--xp-danger);
  font-weight: 600;
}

@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }
  .sys-info-grid { grid-template-columns: repeat(2, 1fr); }
  .net-info-list { grid-template-columns: 1fr; }
  .quick-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 480px) {
  .sys-info-grid { grid-template-columns: 1fr; }
  .quick-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
