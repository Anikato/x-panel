<template>
  <div class="dashboard">
    <!-- Card 1: 系统概览 — 左系统信息 右网络 -->
    <el-card shadow="never" class="dash-card">
      <div class="overview-grid">
        <div class="ov-sys">
          <div class="ov-hd"><el-icon><Monitor /></el-icon><span>{{ t('home.systemInfo') }}</span></div>
          <div class="kv-grid">
            <div class="kv-item" v-for="item in sysInfoItems" :key="item.label">
              <span class="kv-k">{{ item.label }}</span>
              <span class="kv-v" :title="item.value">
                {{ item.value }}
                <el-icon class="copy-btn" @click="copyText(item.value)" v-if="item.value && item.value !== '-'"><CopyDocument /></el-icon>
              </span>
            </div>
          </div>
        </div>
        <div class="ov-divider"></div>
        <div class="ov-net">
          <div class="ov-hd"><el-icon><Connection /></el-icon><span>{{ t('home.network') }}</span></div>
          <div class="net-ips">
            <div class="kv-item" v-if="stats.host?.publicIPv4">
              <span class="kv-k">{{ t('home.publicIPv4') }}</span>
              <span class="kv-v accent">{{ stats.host.publicIPv4 }}<el-icon class="copy-btn" @click="copyText(stats.host.publicIPv4)"><CopyDocument /></el-icon></span>
            </div>
            <div class="kv-item" v-if="stats.host?.publicIPv6">
              <span class="kv-k">{{ t('home.publicIPv6') }}</span>
              <span class="kv-v mono">{{ stats.host.publicIPv6 }}<el-icon class="copy-btn" @click="copyText(stats.host.publicIPv6)"><CopyDocument /></el-icon></span>
            </div>
            <template v-for="iface in stats.host?.interfaces" :key="iface.name">
              <div class="kv-item" v-for="ip in iface.ipv4" :key="ip">
                <span class="kv-k"><el-tag size="small" :type="iface.status === 'up' ? 'success' : 'info'" effect="plain" round>{{ iface.name }}</el-tag></span>
                <span class="kv-v mono">{{ ip }}<el-icon class="copy-btn" @click="copyText(ip.split('/')[0])"><CopyDocument /></el-icon></span>
              </div>
            </template>
            <div class="kv-item" v-if="stats.host?.dnsServers?.length">
              <span class="kv-k">DNS</span>
              <span class="kv-v mono">{{ stats.host.dnsServers.join(', ') }}<el-icon class="copy-btn" @click="copyText(stats.host.dnsServers.join(', '))"><CopyDocument /></el-icon></span>
            </div>
          </div>
          <table class="traffic-tbl" v-if="mainNics.length">
            <thead><tr><th></th><th class="col-up">{{ t('home.upload') }}</th><th class="col-down">{{ t('home.download') }}</th></tr></thead>
            <tbody>
              <tr v-for="nic in mainNics" :key="nic.name">
                <td class="td-nic">{{ nic.name }}</td>
                <td class="col-up">{{ formatSpeed(nic.speedUp) }}</td>
                <td class="col-down">{{ formatSpeed(nic.speedDown) }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="tr-total"><td>{{ t('home.totalTraffic') }}</td><td class="col-up">{{ formatBytes(stats.network?.bytesSent) }}</td><td class="col-down">{{ formatBytes(stats.network?.bytesRecv) }}</td></tr>
            </tfoot>
          </table>
        </div>
      </div>
    </el-card>

    <!-- Card 2: 资源 + 磁盘 -->
    <el-card shadow="never" class="dash-card">
      <template #header>
        <div class="card-hd"><el-icon><Odometer /></el-icon><span>{{ t('home.resourceUsage') }}</span></div>
      </template>
      <!-- 资源三格 -->
      <div class="res-row">
        <div class="res-cell">
          <div class="res-hd"><div class="res-dot cpu-dot"></div><span>CPU</span><span class="res-pct" :class="pctCls(stats.cpu?.usagePercent)">{{ fmtPct(stats.cpu?.usagePercent) }}</span></div>
          <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.cpu?.usagePercent, 'cpu')"></div></div>
          <div class="res-foot">{{ stats.cpu?.cores }} {{ t('home.physical') }} / {{ stats.cpu?.logicalCores }} {{ t('home.logical') }}</div>
        </div>
        <div class="res-sep"></div>
        <div class="res-cell">
          <div class="res-hd"><div class="res-dot mem-dot"></div><span>{{ t('home.memory') }}</span><span class="res-pct" :class="pctCls(stats.memory?.usedPercent)">{{ fmtPct(stats.memory?.usedPercent) }}</span></div>
          <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.memory?.usedPercent, 'mem')"></div></div>
          <div class="res-foot">{{ formatBytes(stats.memory?.used) }} / {{ formatBytes(stats.memory?.total) }}</div>
          <div class="res-sub" v-if="(stats.memory?.swapTotal ?? 0) > 0">Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }} ({{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%)</div>
        </div>
        <div class="res-sep"></div>
        <div class="res-cell">
          <div class="res-hd"><div class="res-dot load-dot"></div><span>{{ t('home.load') }}</span><span class="res-pct" :class="pctCls(loadPct)">{{ loadPct.toFixed(0) }}%</span></div>
          <div class="bar-bg"><div class="bar-fg" :style="barSty(loadPct, 'load')"></div></div>
          <div class="res-foot load-triple"><span>1m: {{ stats.load?.load1?.toFixed(2) || '-' }}</span><span>5m: {{ stats.load?.load5?.toFixed(2) || '-' }}</span><span>15m: {{ stats.load?.load15?.toFixed(2) || '-' }}</span></div>
        </div>
      </div>
      <!-- 磁盘 -->
      <div class="disk-area" v-if="filteredDisks.length">
        <div class="disk-divider"></div>
        <div class="disk-rows">
          <div class="dk-row" v-for="disk in filteredDisks" :key="disk.mountPoint">
            <span class="dk-mount" :title="disk.mountPoint">{{ disk.mountPoint }}</span>
            <span class="dk-dev">{{ disk.device }} · {{ disk.fsType }}</span>
            <div class="dk-bar"><div class="bar-bg"><div class="bar-fg" :style="barSty(disk.usedPercent, 'disk')"></div></div></div>
            <span class="dk-pct" :class="pctCls(disk.usedPercent)">{{ disk.usedPercent.toFixed(1) }}%</span>
            <span class="dk-size">{{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- Card 3: 快速入口 + Top 进程 -->
    <el-row :gutter="16">
      <el-col :xs="24" :lg="10">
        <el-card shadow="never" class="dash-card">
          <template #header>
            <div class="card-hd"><el-icon><Compass /></el-icon><span>{{ t('home.quickEntry') }}</span></div>
          </template>
          <div class="quick-grid">
            <div v-for="entry in quickEntries" :key="entry.path" class="quick-item" @click="router.push(entry.path)">
              <div class="qi-icon"><el-icon :size="20"><component :is="entry.icon" /></el-icon></div>
              <span class="qi-label">{{ entry.title }}</span>
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
            <el-table-column label="CPU %" width="90" align="right">
              <template #default="{ row }"><span :class="row.cpuPercent > 50 ? 'text-danger' : ''">{{ row.cpuPercent.toFixed(1) }}%</span></template>
            </el-table-column>
            <el-table-column :label="t('home.memoryUsage')" width="90" align="right">
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
const stats = ref<Partial<SystemStats>>({})
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  try { const r = await getSystemStats(); stats.value = r.data || {} } catch {}
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

const sysInfoItems = computed(() => {
  const h = stats.value.host ?? ({} as Partial<HostInfo>)
  return [
    { label: t('home.hostname'), value: h.hostname || '-' },
    { label: t('home.os'), value: `${h.platform || ''} ${h.platformVersion || ''}`.trim() || '-' },
    { label: t('home.kernel'), value: h.kernelVersion || '-' },
    { label: t('home.arch'), value: h.kernelArch || '-' },
    { label: t('home.uptime'), value: formatUptime(stats.value.uptime) },
    { label: t('home.timezone'), value: h.timezone || '-' },
    { label: t('home.virtualization'), value: h.virtualization || '-' },
    { label: t('home.cpuModel'), value: stats.value.cpu?.modelName || '-' },
    { label: t('home.cpuCores'), value: stats.value.cpu ? `${stats.value.cpu.cores} ${t('home.physical')} / ${stats.value.cpu.logicalCores} ${t('home.logical')}` : '-' },
    { label: t('home.totalMemory'), value: formatBytes(stats.value.memory?.total) },
  ]
})

const loadPct = computed(() => {
  const c = stats.value.cpu?.logicalCores || 1
  return Math.min(((stats.value.load?.load1 || 0) / c) * 100, 100)
})

const mainNics = computed(() => (stats.value.netIO || []).filter(n => n.name !== 'lo').slice(0, 6))

const ignoreMounts = new Set(['/boot', '/boot/efi', '/boot/firmware'])
const ignorePfx = ['/snap/', '/run/']
const ignoreFs = new Set(['squashfs', 'tmpfs', 'devtmpfs', 'overlay'])

const filteredDisks = computed(() =>
  (stats.value.disks || []).filter(d =>
    !ignoreMounts.has(d.mountPoint) &&
    !ignoreFs.has(d.fsType) &&
    !ignorePfx.some(p => d.mountPoint.startsWith(p)) &&
    d.total >= 100 * 1024 * 1024
  )
)

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
  try { await navigator.clipboard.writeText(text); ElMessage.success(t('commons.copy') + ' ✓') }
  catch { ElMessage.error('Copy failed') }
}

const accentColor = () => getComputedStyle(document.documentElement).getPropertyValue('--xp-accent').trim() || '#22d3ee'
const palette: Record<string, string> = { cpu: '', mem: '#818cf8', load: '#34d399', disk: '#60a5fa' }

const barColor = (pct: number, type: string) => {
  if (pct >= 90) return '#ef4444'
  if (pct >= 70) return '#f59e0b'
  if (!palette.cpu) palette.cpu = accentColor()
  return palette[type] || palette.cpu
}

const barSty = (pct?: number, type = 'cpu') => {
  const v = Math.min(pct || 0, 100); const c = barColor(v, type)
  return { width: `${v}%`, background: `linear-gradient(90deg, ${c}cc, ${c})`, boxShadow: `0 0 6px ${c}33` }
}

const pctCls = (pct?: number) => (pct || 0) >= 90 ? 'c-danger' : (pct || 0) >= 70 ? 'c-warn' : 'c-ok'
const fmtPct = (v?: number) => `${(v ?? 0).toFixed(1)}%`

const formatBytes = (b?: number) => {
  if (!b || b === 0) return '0 B'
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(b) / Math.log(1024))
  return (b / 1024 ** i).toFixed(1) + ' ' + u[i]
}

const formatSpeed = (s?: number) => {
  if (!s || s < 0) return '0 B/s'
  if (s < 1024) return s.toFixed(0) + ' B/s'
  if (s < 1048576) return (s / 1024).toFixed(1) + ' KB/s'
  return (s / 1048576).toFixed(2) + ' MB/s'
}

onMounted(() => { loadStats(); timer = setInterval(loadStats, 5000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<style lang="scss" scoped>
.dashboard { padding: 0; }
.dash-card {
  margin-bottom: 16px;
  background: linear-gradient(135deg, var(--xp-bg-card) 0%, rgba(var(--xp-accent-rgb, 34, 211, 238), 0.04) 100%);
  border: 1px solid var(--xp-border-light);
  transition: border-color 0.2s;
  &:hover { border-color: rgba(var(--xp-accent-rgb, 34, 211, 238), 0.2); }
}

.card-hd {
  display: flex; align-items: center; gap: 8px;
  font-weight: 600; font-size: 14px; color: var(--xp-text-primary);
}

/* ==================== Card 1: Overview ==================== */
.overview-grid {
  display: flex; gap: 0;
}

.ov-sys { flex: 1; min-width: 0; padding-right: 24px; }
.ov-net { flex: 0 0 380px; min-width: 0; padding-left: 24px; }

.ov-divider {
  width: 1px; align-self: stretch;
  background: var(--xp-border-light);
}

.ov-hd {
  display: flex; align-items: center; gap: 6px;
  font-weight: 600; font-size: 13px; color: var(--xp-text-primary);
  margin-bottom: 14px;
}

/* Key-Value */
.kv-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 280px));
  gap: 8px 20px;
}

.kv-item {
  display: flex; align-items: baseline; gap: 8px; min-width: 0;
}

.kv-k {
  font-size: 12px; color: var(--xp-text-muted);
  white-space: nowrap; flex-shrink: 0; min-width: 50px;
}

.kv-v {
  font-size: 13px; color: var(--xp-text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  display: inline-flex; align-items: center; gap: 4px; min-width: 0;
  &.accent { color: var(--xp-accent); font-weight: 600; }
  &.mono { font-family: 'JetBrains Mono', 'Fira Code', monospace; font-size: 12px; }
}

.copy-btn {
  opacity: 0; cursor: pointer; flex-shrink: 0; transition: opacity .15s;
  color: var(--xp-text-muted);
  &:hover { color: var(--xp-accent); }
}
.kv-item:hover .copy-btn { opacity: 1; }

/* Network IPs */
.net-ips { display: flex; flex-direction: column; gap: 5px; }

/* Traffic table */
.traffic-tbl {
  width: 100%; border-collapse: collapse; margin-top: 12px;
  font-size: 12px; font-variant-numeric: tabular-nums;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;

  th, td { padding: 3px 0; }
  th { font-weight: 500; color: var(--xp-text-muted); font-size: 11px; }
  th:first-child, td:first-child { text-align: left; }

  .col-up { text-align: right; width: 100px; color: var(--xp-color-up, #34d399); }
  .col-down { text-align: right; width: 100px; color: var(--xp-color-down, #a78bfa); }
  .td-nic { color: var(--xp-text-secondary); font-weight: 500; }
  .tr-total td {
    border-top: 1px solid var(--xp-border-light); padding-top: 5px;
    font-size: 11px; color: var(--xp-text-muted);
  }
}

/* ==================== Card 2: Resources + Disk ==================== */
.res-row {
  display: flex; align-items: flex-start;
}

.res-cell { flex: 1; min-width: 0; padding: 0 20px; }
.res-cell:first-child { padding-left: 0; }
.res-cell:last-child { padding-right: 0; }

.res-sep {
  width: 1px; align-self: stretch; min-height: 60px;
  background: var(--xp-border-light);
}

.res-hd {
  display: flex; align-items: center; gap: 8px; margin-bottom: 10px;
  span:first-of-type { font-size: 13px; font-weight: 600; color: var(--xp-text-primary); flex: 1; }
}

.res-dot {
  width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
}
.cpu-dot { background: var(--xp-accent); }
.mem-dot { background: #818cf8; }
.load-dot { background: #34d399; }

.res-pct {
  font-size: 20px; font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.c-ok { color: var(--xp-accent); }
.c-warn { color: #f59e0b; }
.c-danger { color: #ef4444; }

.bar-bg {
  width: 100%; height: 5px;
  background: var(--xp-progress-trail, rgba(255,255,255,0.06));
  border-radius: 3px; overflow: hidden; margin-bottom: 8px;
}

.bar-fg {
  height: 100%; border-radius: 3px; min-width: 2px;
  transition: width .8s cubic-bezier(.4,0,.2,1), background .4s ease;
}

.res-foot { font-size: 12px; color: var(--xp-text-secondary); }
.res-sub { font-size: 11px; color: var(--xp-text-muted); margin-top: 2px; }
.load-triple { display: flex; gap: 12px; }

/* Disk area */
.disk-divider {
  height: 1px; background: var(--xp-border-light); margin: 18px 0 14px;
}

.disk-rows { display: flex; flex-direction: column; gap: 10px; }

.dk-row {
  display: grid;
  grid-template-columns: 100px 140px 1fr 60px 130px;
  align-items: center; gap: 12px;
}

.dk-mount {
  font-size: 13px; font-weight: 600; color: var(--xp-text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

.dk-dev {
  font-size: 11px; color: var(--xp-text-muted);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

.dk-bar { min-width: 0; .bar-bg { margin-bottom: 0; } }

.dk-pct {
  font-size: 14px; font-weight: 700; text-align: right;
  font-variant-numeric: tabular-nums;
}

.dk-size {
  font-size: 12px; color: var(--xp-text-secondary); text-align: right;
  font-variant-numeric: tabular-nums; white-space: nowrap;
}

/* ==================== Card 3: Quick & Process ==================== */
.quick-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px;
}

.quick-item {
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  padding: 14px 8px;
  background: var(--xp-bg-inset); border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius); cursor: pointer;
  transition: all .25s cubic-bezier(.4,0,.2,1);

  &:hover {
    border-color: var(--xp-accent-muted); background: var(--xp-accent-muted);
    transform: translateY(-2px); box-shadow: 0 4px 14px rgba(0,0,0,.1);
    .qi-icon { color: var(--xp-accent); transform: scale(1.1); }
    .qi-label { color: var(--xp-accent); }
  }
}

.qi-icon {
  width: 36px; height: 36px;
  display: flex; align-items: center; justify-content: center;
  background: rgba(255,255,255,.04); border-radius: 10px;
  color: var(--xp-text-secondary); transition: all .25s;
}

.qi-label {
  font-size: 12px; color: var(--xp-text-secondary);
  font-weight: 500; text-align: center; transition: color .2s;
}

.text-danger { color: #ef4444; font-weight: 600; }

/* ==================== Responsive ==================== */
@media (max-width: 1200px) {
  .overview-grid { flex-direction: column; }
  .ov-sys { padding-right: 0; padding-bottom: 16px; }
  .ov-net { flex: none; padding-left: 0; padding-top: 16px; }
  .ov-divider { width: 100%; height: 1px; }
  .dk-row { grid-template-columns: 80px 110px 1fr 50px 110px; gap: 8px; }
}

@media (max-width: 768px) {
  .kv-grid { grid-template-columns: 1fr; }
  .res-row { flex-direction: column; gap: 20px; }
  .res-cell { padding: 0; }
  .res-sep { width: 100%; height: 1px; min-height: 0; }
  .dk-row { grid-template-columns: 1fr 1fr; gap: 4px 8px; }
  .dk-bar { grid-column: 1 / -1; }
  .dk-mount { grid-column: 1; }
  .dk-dev { display: none; }
  .dk-pct { text-align: left; }
  .dk-size { text-align: left; }
  .quick-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 480px) {
  .quick-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
