<template>
  <div class="dashboard">
    <!-- Row 1: 系统信息 + 网络 -->
    <el-row :gutter="16" class="dash-row">
      <el-col :xs="24" :lg="14">
        <el-card shadow="never" class="dash-card">
          <template #header>
            <div class="card-hd"><el-icon><Monitor /></el-icon><span>{{ t('home.systemInfo') }}</span></div>
          </template>
          <div class="kv-grid">
            <div class="kv-item" v-for="item in sysInfoItems" :key="item.label">
              <span class="kv-label">{{ item.label }}</span>
              <span class="kv-value" :title="item.value">
                {{ item.value }}
                <el-icon class="copy-btn" @click="copyText(item.value)" v-if="item.value && item.value !== '-'"><CopyDocument /></el-icon>
              </span>
            </div>
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="10">
        <el-card shadow="never" class="dash-card">
          <template #header>
            <div class="card-hd"><el-icon><Connection /></el-icon><span>{{ t('home.network') }}</span></div>
          </template>
          <!-- IP 信息 -->
          <div class="net-ips">
            <div class="kv-item" v-if="stats.host?.publicIPv4">
              <span class="kv-label">{{ t('home.publicIPv4') }}</span>
              <span class="kv-value accent">
                {{ stats.host.publicIPv4 }}
                <el-icon class="copy-btn" @click="copyText(stats.host.publicIPv4)"><CopyDocument /></el-icon>
              </span>
            </div>
            <div class="kv-item" v-if="stats.host?.publicIPv6">
              <span class="kv-label">{{ t('home.publicIPv6') }}</span>
              <span class="kv-value mono-sm">
                {{ stats.host.publicIPv6 }}
                <el-icon class="copy-btn" @click="copyText(stats.host.publicIPv6)"><CopyDocument /></el-icon>
              </span>
            </div>
            <template v-for="iface in stats.host?.interfaces" :key="iface.name">
              <div class="kv-item" v-for="ip in iface.ipv4" :key="ip">
                <span class="kv-label">
                  <el-tag size="small" :type="iface.status === 'up' ? 'success' : 'info'" effect="plain" round>{{ iface.name }}</el-tag>
                </span>
                <span class="kv-value mono-sm">
                  {{ ip }}
                  <el-icon class="copy-btn" @click="copyText(ip.split('/')[0])"><CopyDocument /></el-icon>
                </span>
              </div>
            </template>
            <div class="kv-item" v-if="stats.host?.dnsServers?.length">
              <span class="kv-label">DNS</span>
              <span class="kv-value mono-sm">
                {{ stats.host.dnsServers.join(', ') }}
                <el-icon class="copy-btn" @click="copyText(stats.host.dnsServers.join(', '))"><CopyDocument /></el-icon>
              </span>
            </div>
          </div>
          <!-- 实时流量 -->
          <div class="net-traffic" v-if="mainNics.length">
            <div class="traffic-sep"></div>
            <table class="traffic-table">
              <thead>
                <tr>
                  <th></th>
                  <th class="th-up">{{ t('home.upload') }}</th>
                  <th class="th-down">{{ t('home.download') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="nic in mainNics" :key="nic.name">
                  <td class="td-nic">{{ nic.name }}</td>
                  <td class="td-up">{{ formatSpeed(nic.speedUp) }}</td>
                  <td class="td-down">{{ formatSpeed(nic.speedDown) }}</td>
                </tr>
              </tbody>
              <tfoot>
                <tr class="traffic-total">
                  <td>{{ t('home.totalTraffic') }}</td>
                  <td class="td-up">{{ formatBytes(stats.network?.bytesSent) }}</td>
                  <td class="td-down">{{ formatBytes(stats.network?.bytesRecv) }}</td>
                </tr>
              </tfoot>
            </table>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Row 2: 资源占用 (single card) -->
    <el-card shadow="never" class="dash-card resource-card">
      <template #header>
        <div class="card-hd"><el-icon><Cpu /></el-icon><span>{{ t('home.resourceUsage') }}</span></div>
      </template>
      <div class="res-grid">
        <!-- CPU -->
        <div class="res-item">
          <div class="res-top">
            <div class="res-icon cpu-bg"><el-icon :size="16"><Cpu /></el-icon></div>
            <span class="res-label">CPU</span>
            <span class="res-pct" :class="pctClass(stats.cpu?.usagePercent)">{{ (stats.cpu?.usagePercent ?? 0).toFixed(1) }}%</span>
          </div>
          <div class="bar-wrap"><div class="bar-fill" :style="barStyle(stats.cpu?.usagePercent, 'cpu')"></div></div>
          <div class="res-desc">{{ stats.cpu?.cores }} {{ t('home.physical') }} / {{ stats.cpu?.logicalCores }} {{ t('home.logical') }}</div>
        </div>
        <!-- Memory -->
        <div class="res-item">
          <div class="res-top">
            <div class="res-icon mem-bg"><el-icon :size="16"><Coin /></el-icon></div>
            <span class="res-label">{{ t('home.memory') }}</span>
            <span class="res-pct" :class="pctClass(stats.memory?.usedPercent)">{{ (stats.memory?.usedPercent ?? 0).toFixed(1) }}%</span>
          </div>
          <div class="bar-wrap"><div class="bar-fill" :style="barStyle(stats.memory?.usedPercent, 'mem')"></div></div>
          <div class="res-desc">{{ formatBytes(stats.memory?.used) }} / {{ formatBytes(stats.memory?.total) }}</div>
          <div class="res-sub" v-if="(stats.memory?.swapTotal ?? 0) > 0">
            Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }} ({{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%)
          </div>
        </div>
        <!-- Load -->
        <div class="res-item">
          <div class="res-top">
            <div class="res-icon load-bg"><el-icon :size="16"><Odometer /></el-icon></div>
            <span class="res-label">{{ t('home.load') }}</span>
            <span class="res-pct" :class="pctClass(loadPercent)">{{ loadPercent.toFixed(0) }}%</span>
          </div>
          <div class="bar-wrap"><div class="bar-fill" :style="barStyle(loadPercent, 'load')"></div></div>
          <div class="res-desc load-vals">
            <span>1m: {{ stats.load?.load1?.toFixed(2) || '-' }}</span>
            <span>5m: {{ stats.load?.load5?.toFixed(2) || '-' }}</span>
            <span>15m: {{ stats.load?.load15?.toFixed(2) || '-' }}</span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Row 3: 磁盘使用 (single card) -->
    <el-card shadow="never" class="dash-card" v-if="filteredDisks.length">
      <template #header>
        <div class="card-hd"><el-icon><Box /></el-icon><span>{{ t('home.diskUsage') }}</span></div>
      </template>
      <div class="disk-list">
        <div class="disk-row" v-for="disk in filteredDisks" :key="disk.mountPoint">
          <div class="disk-meta">
            <span class="disk-mount" :title="disk.mountPoint">{{ disk.mountPoint }}</span>
            <span class="disk-fs">{{ disk.device }} · {{ disk.fsType }}</span>
          </div>
          <div class="disk-bar-area">
            <div class="bar-wrap"><div class="bar-fill" :style="barStyle(disk.usedPercent, 'disk')"></div></div>
          </div>
          <div class="disk-nums">
            <span class="disk-pct" :class="pctClass(disk.usedPercent)">{{ disk.usedPercent.toFixed(1) }}%</span>
            <span class="disk-size">{{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Row 4: 快速入口 + Top 进程 -->
    <el-row :gutter="16" class="dash-row">
      <el-col :xs="24" :lg="10">
        <el-card shadow="never" class="dash-card">
          <template #header>
            <div class="card-hd"><el-icon><Compass /></el-icon><span>{{ t('home.quickEntry') }}</span></div>
          </template>
          <div class="quick-grid">
            <div v-for="entry in quickEntries" :key="entry.path" class="quick-item" @click="router.push(entry.path)">
              <div class="quick-icon"><el-icon :size="20"><component :is="entry.icon" /></el-icon></div>
              <span class="quick-label">{{ entry.title }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="14">
        <el-card shadow="never" class="dash-card">
          <template #header>
            <div class="card-hd"><el-icon><DataLine /></el-icon><span>{{ t('home.topProcess') }}</span></div>
          </template>
          <el-table :data="stats.topProcess || []" size="small" :show-header="true" stripe>
            <el-table-column prop="pid" label="PID" width="70" />
            <el-table-column prop="name" :label="t('home.processName')" min-width="140" show-overflow-tooltip />
            <el-table-column label="CPU %" width="100" align="right">
              <template #default="{ row }">
                <span :class="row.cpuPercent > 50 ? 'text-danger' : ''">{{ row.cpuPercent.toFixed(1) }}%</span>
              </template>
            </el-table-column>
            <el-table-column :label="t('home.memoryUsage')" width="100" align="right">
              <template #default="{ row }">{{ formatBytes(row.memRss) }}</template>
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
import { ElMessage } from 'element-plus'
import type { SystemStats, HostInfo } from '@/api/interface'
import {
  Monitor, Cpu, Coin, Odometer, Connection,
  Box, Compass, DataLine, CopyDocument,
} from '@element-plus/icons-vue'

const router = useRouter()
const { t } = useI18n()
const loading = ref(false)
const stats = ref<Partial<SystemStats>>({})
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  loading.value = true
  try {
    const res = await getSystemStats()
    stats.value = res.data || {}
  } catch { /* handled */ }
  finally { loading.value = false }
}

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
  return (stats.value.netIO || []).filter((n) => n.name !== 'lo').slice(0, 6)
})

const ignoredMounts = new Set(['/boot', '/boot/efi', '/boot/firmware'])
const ignoredPrefixes = ['/snap/', '/run/']
const ignoredFs = new Set(['squashfs', 'tmpfs', 'devtmpfs', 'overlay'])

const filteredDisks = computed(() => {
  return (stats.value.disks || []).filter((d) => {
    if (ignoredMounts.has(d.mountPoint)) return false
    if (ignoredFs.has(d.fsType)) return false
    if (ignoredPrefixes.some(p => d.mountPoint.startsWith(p))) return false
    if (d.total < 100 * 1024 * 1024) return false
    return true
  })
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

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('commons.copy') + ' ✓')
  } catch { ElMessage.error('Copy failed') }
}

const getCS = (v: string) => getComputedStyle(document.documentElement).getPropertyValue(v).trim()
const THEME = { danger: '#ef4444', warning: '#f59e0b', cpu: '', mem: '#818cf8', load: '#34d399', disk: '#60a5fa' }

const getBarColor = (pct: number, type: string): string => {
  if (pct >= 90) return THEME.danger
  if (pct >= 70) return THEME.warning
  if (!THEME.cpu) THEME.cpu = getCS('--xp-accent') || '#22d3ee'
  return (THEME as Record<string, string>)[type] || THEME.cpu
}

const barStyle = (pct?: number, type = 'cpu') => {
  const v = Math.min(pct || 0, 100)
  const c = getBarColor(v, type)
  return { width: `${v}%`, background: `linear-gradient(90deg, ${c}dd, ${c})`, boxShadow: `0 0 8px ${c}40` }
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

onMounted(() => {
  loadStats()
  timer = setInterval(loadStats, 5000)
})
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<style lang="scss" scoped>
.dashboard { padding: 0; }
.dash-row { margin-bottom: 16px; }
.dash-card { margin-bottom: 16px; }

/* ===== Card Header ===== */
.card-hd {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
  color: var(--xp-text-primary);
}

/* ===== Key-Value Grid (System Info / Network IPs) ===== */
.kv-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 10px 24px;
}

.kv-item {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
}

.kv-label {
  font-size: 12px;
  color: var(--xp-text-muted);
  white-space: nowrap;
  flex-shrink: 0;
  min-width: 56px;
}

.kv-value {
  font-size: 13px;
  color: var(--xp-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;

  &.accent { color: var(--xp-accent); font-weight: 600; }
  &.mono-sm { font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 12px; }
}

.copy-btn {
  opacity: 0;
  cursor: pointer;
  flex-shrink: 0;
  transition: opacity 0.15s;
  color: var(--xp-text-muted);
  &:hover { color: var(--xp-accent); }
}
.kv-item:hover .copy-btn { opacity: 1; }

/* ===== Network Card ===== */
.net-ips { display: flex; flex-direction: column; gap: 6px; }

.traffic-sep {
  height: 1px;
  background: var(--xp-border-light);
  margin: 12px 0;
}

.traffic-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;

  th, td { padding: 4px 0; }
  th { font-weight: 500; color: var(--xp-text-muted); text-align: right; font-size: 11px; }
  th:first-child, td:first-child { text-align: left; }

  .th-up, .td-up { color: var(--xp-color-up, #34d399); width: 110px; text-align: right; }
  .th-down, .td-down { color: var(--xp-color-down, #a78bfa); width: 110px; text-align: right; }
  .td-nic { color: var(--xp-text-secondary); font-weight: 500; }

  .traffic-total {
    td { border-top: 1px solid var(--xp-border-light); padding-top: 6px; font-size: 11px; color: var(--xp-text-muted); }
  }
}

/* ===== Resource Card ===== */
.resource-card { margin-bottom: 16px; }

.res-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 24px;
}

.res-item {
  min-width: 0;
}

.res-top {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.res-icon {
  width: 30px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  flex-shrink: 0;
}

.cpu-bg { background: var(--xp-accent-muted); color: var(--xp-accent); }
.mem-bg { background: rgba(129, 140, 248, 0.12); color: var(--xp-accent-secondary, #818cf8); }
.load-bg { background: rgba(52, 211, 153, 0.12); color: var(--xp-success, #34d399); }

.res-label { font-size: 13px; font-weight: 600; color: var(--xp-text-primary); flex: 1; }

.res-pct {
  font-size: 20px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.pct-normal { color: var(--xp-accent); }
.pct-warning { color: var(--xp-warning, #f59e0b); }
.pct-danger { color: var(--xp-danger, #ef4444); }

.bar-wrap {
  width: 100%;
  height: 5px;
  background: var(--xp-progress-trail, rgba(255,255,255,0.06));
  border-radius: 3px;
  overflow: hidden;
  margin-bottom: 8px;
}

.bar-fill {
  height: 100%;
  border-radius: 3px;
  transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1), background 0.4s ease;
  min-width: 2px;
}

.res-desc { font-size: 12px; color: var(--xp-text-secondary); }
.res-sub { font-size: 11px; color: var(--xp-text-muted); margin-top: 2px; }
.load-vals { display: flex; gap: 14px; }

/* ===== Disk Card ===== */
.disk-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.disk-row {
  display: grid;
  grid-template-columns: 180px 1fr 140px;
  align-items: center;
  gap: 16px;
}

.disk-meta {
  min-width: 0;
  overflow: hidden;
}

.disk-mount {
  font-size: 13px;
  font-weight: 600;
  color: var(--xp-text-primary);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-fs {
  font-size: 11px;
  color: var(--xp-text-muted);
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disk-bar-area {
  min-width: 0;
  .bar-wrap { margin-bottom: 0; }
}

.disk-nums {
  text-align: right;
  white-space: nowrap;
}

.disk-pct {
  font-size: 15px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  display: block;
}

.disk-size {
  font-size: 11px;
  color: var(--xp-text-secondary);
}

/* ===== Quick Entry ===== */
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
  padding: 14px 8px;
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);

  &:hover {
    border-color: var(--xp-accent-muted);
    background: var(--xp-accent-muted);
    transform: translateY(-2px);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.1);

    .quick-icon { color: var(--xp-accent); transform: scale(1.1); }
    .quick-label { color: var(--xp-accent); }
  }
}

.quick-icon {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(255, 255, 255, 0.04);
  border-radius: 10px;
  color: var(--xp-text-secondary);
  transition: all 0.3s;
}

.quick-label {
  font-size: 12px;
  color: var(--xp-text-secondary);
  font-weight: 500;
  text-align: center;
  transition: color 0.2s;
}

.text-danger { color: var(--xp-danger, #ef4444); font-weight: 600; }

/* ===== Responsive ===== */
@media (max-width: 1200px) {
  .res-grid { grid-template-columns: repeat(3, 1fr); gap: 16px; }
  .disk-row { grid-template-columns: 140px 1fr 120px; gap: 12px; }
}

@media (max-width: 768px) {
  .kv-grid { grid-template-columns: 1fr; }
  .res-grid { grid-template-columns: 1fr; gap: 20px; }
  .disk-row { grid-template-columns: 1fr; gap: 6px; }
  .disk-nums { text-align: left; display: flex; align-items: baseline; gap: 8px; }
  .disk-pct { display: inline; }
  .quick-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 480px) {
  .quick-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
