<template>
  <div class="dashboard">
    <!-- 顶部系统信息 + 版本 -->
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
          </div>
        </div>
      </div>
      <div class="header-right">
        <div class="uptime-display">
          <el-icon><Clock /></el-icon>
          <span>{{ t('home.uptime') }}: {{ formatUptime(stats.uptime) }}</span>
        </div>
        <el-button text :icon="Refresh" @click="loadStats" :loading="loading" circle />
      </div>
    </div>

    <!-- 系统详情卡片 -->
    <el-card shadow="never" class="sys-info-card">
      <div class="sys-info-grid">
        <div class="sys-info-item">
          <span class="sys-info-label">{{ t('home.os') }}</span>
          <span class="sys-info-value">{{ stats.host?.platform }} {{ stats.host?.platformVersion }}</span>
        </div>
        <div class="sys-info-item">
          <span class="sys-info-label">{{ t('home.kernel') }}</span>
          <span class="sys-info-value">{{ stats.host?.kernelVersion || '-' }}</span>
        </div>
        <div class="sys-info-item">
          <span class="sys-info-label">{{ t('home.arch') }}</span>
          <span class="sys-info-value">{{ stats.host?.kernelArch || '-' }}</span>
        </div>
        <div class="sys-info-item">
          <span class="sys-info-label">{{ t('home.cpuModel') }}</span>
          <span class="sys-info-value">{{ stats.cpu?.modelName || '-' }}</span>
        </div>
        <div class="sys-info-item">
          <span class="sys-info-label">{{ t('home.cpuCores') }}</span>
          <span class="sys-info-value">{{ stats.cpu?.cores }} {{ t('home.physical') }} / {{ stats.cpu?.logicalCores }} {{ t('home.logical') }}</span>
        </div>
        <div class="sys-info-item">
          <span class="sys-info-label">{{ t('home.totalMemory') }}</span>
          <span class="sys-info-value">{{ formatBytes(stats.memory?.total) }}</span>
        </div>
      </div>
    </el-card>

    <!-- 资源占用 — 进度条风格 -->
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
            <div class="resource-sub" v-if="stats.memory?.swapTotal > 0">
              Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }}
              ({{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%)
            </div>
          </div>
        </el-col>

        <!-- 系统负载 -->
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
import {
  Monitor, Clock, Refresh, Cpu, Coin, Odometer, Connection,
  Box, Compass, DataLine, FolderOpened, Setting, Document, Notebook,
} from '@element-plus/icons-vue'

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const stats = ref<any>({})
const panelVersion = ref('...')
let timer: ReturnType<typeof setInterval> | null = null

// 获取系统状态
const loadStats = async () => {
  loading.value = true
  try {
    const res = await getSystemStats()
    stats.value = res.data || {}
  } catch { /* handled by interceptor */ }
  finally { loading.value = false }
}

// 获取面板版本
const fetchVersion = async () => {
  try {
    const res: any = await getCurrentVersion()
    if (res.data) {
      panelVersion.value = res.data.version === 'dev' ? 'dev' : res.data.version
    }
  } catch {
    panelVersion.value = '-'
  }
}

// 负载百分比（load1 / 逻辑核心数 * 100）
const loadPercent = computed(() => {
  const cores = stats.value.cpu?.logicalCores || 1
  const load1 = stats.value.load?.load1 || 0
  return Math.min((load1 / cores) * 100, 100)
})

// 主网卡（过滤 lo）
const mainNics = computed(() => {
  return (stats.value.netIO || []).filter((n: any) => n.name !== 'lo').slice(0, 4)
})

// 快速入口
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

// 进度条颜色
const getBarColor = (pct: number, type: string): string => {
  if (pct >= 90) return '#ef4444'
  if (pct >= 70) return '#f59e0b'
  const colors: Record<string, string> = {
    cpu: '#22d3ee',
    mem: '#818cf8',
    load: '#34d399',
    disk: '#60a5fa',
    net: '#a78bfa',
  }
  return colors[type] || '#22d3ee'
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

// 工具函数
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

/* ===== 顶部区域 ===== */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px 24px;
  background: linear-gradient(135deg, var(--xp-bg-surface) 0%, var(--xp-bg-elevated) 100%);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-lg);
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

/* ===== 系统信息卡片 ===== */
.sys-info-card {
  margin-bottom: 20px;
}

.sys-info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px 32px;
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

/* ===== 资源占用卡片 ===== */
.resource-section {
  margin-bottom: 20px;
}

.resource-card {
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  padding: 18px 20px;
  margin-bottom: 16px;
  transition: all 0.25s;

  &:hover {
    border-color: rgba(34, 211, 238, 0.15);
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
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

.cpu-icon {
  background: rgba(34, 211, 238, 0.12);
  color: #22d3ee;
}

.mem-icon {
  background: rgba(129, 140, 248, 0.12);
  color: #818cf8;
}

.load-icon {
  background: rgba(52, 211, 153, 0.12);
  color: #34d399;
}

.net-icon {
  background: rgba(167, 139, 250, 0.12);
  color: #a78bfa;
}

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
.pct-warning { color: #f59e0b; }
.pct-danger { color: #ef4444; }

/* ===== 进度条 ===== */
.resource-bar-wrapper {
  width: 100%;
  height: 8px;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 4px;
  overflow: hidden;
  margin-bottom: 8px;
}

.resource-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 0.6s ease, background 0.4s ease;
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

/* ===== 网络 ===== */
.net-stats {
  margin-bottom: 8px;
}

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

.net-speed-up {
  color: #22d3ee;
  font-variant-numeric: tabular-nums;
}

.net-speed-down {
  color: #a78bfa;
  font-variant-numeric: tabular-nums;
}

/* ===== 磁盘 ===== */
.disk-section {
  margin-bottom: 20px;
}

.disk-card {
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  padding: 16px 18px;
  margin-bottom: 16px;
  transition: all 0.25s;

  &:hover {
    border-color: rgba(96, 165, 250, 0.15);
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
  color: #60a5fa;
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

.disk-fs {
  color: var(--xp-text-muted);
}

.disk-inode {
  font-size: 11px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}

/* ===== 底部区域 ===== */
.bottom-section {
  margin-bottom: 20px;
}

.section-card {
  margin-bottom: 16px;
}

.section-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--xp-text-primary);
}

/* ===== 快速入口 ===== */
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
  background: var(--xp-bg-base);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;
  transition: all 0.25s;

  &:hover {
    border-color: rgba(34, 211, 238, 0.2);
    background: var(--xp-accent-muted);
    transform: translateY(-2px);

    .quick-icon {
      color: var(--xp-accent);
      background: rgba(34, 211, 238, 0.12);
    }
  }

  .quick-icon {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.04);
    border-radius: 8px;
    color: var(--xp-text-secondary);
    transition: all 0.25s;
  }

  .quick-label {
    font-size: 12px;
    color: var(--xp-text-secondary);
    font-weight: 500;
    text-align: center;
  }
}

/* ===== 表格 ===== */
.text-danger {
  color: #ef4444;
  font-weight: 600;
}

/* ===== 响应式 ===== */
@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .sys-info-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .quick-grid {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (max-width: 480px) {
  .sys-info-grid {
    grid-template-columns: 1fr;
  }

  .quick-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
