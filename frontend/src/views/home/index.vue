<template>
  <div class="dashboard">
    <!-- 系统概览：三等分卡片 [资源占用 | 网络 | 系统信息] -->
    <el-card shadow="never" class="dash-card">
      <div class="tri-grid">
        <!-- 左列：资源占用 -->
        <div class="tri-col">
          <div class="col-hd"><el-icon><Odometer /></el-icon><span>{{ t('home.resourceUsage') }}</span></div>
          <div class="res-list">
            <div class="res-item">
              <div class="res-hd"><div class="res-dot cpu-dot"></div><span>CPU</span><span class="res-pct" :class="pctCls(stats.cpu?.usagePercent)">{{ fmtPct(stats.cpu?.usagePercent) }}</span></div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.cpu?.usagePercent, 'cpu')"></div></div>
              <div class="res-foot">{{ stats.cpu?.cores }} {{ t('home.physical') }} / {{ stats.cpu?.logicalCores }} {{ t('home.logical') }}</div>
            </div>
            <div class="res-item">
              <div class="res-hd"><div class="res-dot mem-dot"></div><span>{{ t('home.memory') }}</span><span class="res-pct" :class="pctCls(stats.memory?.usedPercent)">{{ fmtPct(stats.memory?.usedPercent) }}</span></div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.memory?.usedPercent, 'mem')"></div></div>
              <div class="res-foot">{{ formatBytes(stats.memory?.used) }} / {{ formatBytes(stats.memory?.total) }}</div>
              <div class="res-sub" v-if="(stats.memory?.swapTotal ?? 0) > 0">Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }} ({{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%)</div>
            </div>
            <div class="res-item">
              <div class="res-hd"><div class="res-dot load-dot"></div><span>{{ t('home.load') }}</span><span class="res-pct" :class="pctCls(loadPct)">{{ loadPct.toFixed(0) }}%</span></div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(loadPct, 'load')"></div></div>
              <div class="res-foot load-triple"><span>1m: {{ stats.load?.load1?.toFixed(2) || '-' }}</span><span>5m: {{ stats.load?.load5?.toFixed(2) || '-' }}</span><span>15m: {{ stats.load?.load15?.toFixed(2) || '-' }}</span></div>
            </div>
            <template v-for="disk in filteredDisks" :key="disk.mountPoint">
              <div class="res-item disk-item">
                <div class="res-hd"><div class="res-dot disk-dot"></div><span>{{ disk.mountPoint }}</span><span class="res-pct" :class="pctCls(disk.usedPercent)">{{ disk.usedPercent.toFixed(1) }}%</span></div>
                <div class="bar-bg"><div class="bar-fg" :style="barSty(disk.usedPercent, 'disk')"></div></div>
                <div class="res-foot">{{ disk.device }} · {{ disk.fsType }} · {{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</div>
              </div>
            </template>
          </div>
        </div>

        <div class="tri-sep"></div>

        <!-- 中列：网络 -->
        <div class="tri-col">
          <div class="col-hd"><el-icon><Connection /></el-icon><span>{{ t('home.network') }}</span></div>
          <div class="net-list">
            <div class="net-row" v-if="stats.host?.publicIPv4">
              <span class="net-label">{{ t('home.publicIPv4') }}</span>
              <span class="net-val accent">{{ stats.host.publicIPv4 }}<el-icon class="copy-btn" @click="copyText(stats.host.publicIPv4)"><CopyDocument /></el-icon></span>
            </div>
            <div class="net-row" v-if="stats.host?.publicIPv6">
              <span class="net-label">{{ t('home.publicIPv6') }}</span>
              <span class="net-val mono">{{ stats.host.publicIPv6 }}<el-icon class="copy-btn" @click="copyText(stats.host.publicIPv6)"><CopyDocument /></el-icon></span>
            </div>
            <template v-for="iface in stats.host?.interfaces" :key="iface.name">
              <div class="net-row" v-for="ip in iface.ipv4" :key="ip">
                <span class="net-label"><el-tag size="small" :type="iface.status === 'up' ? 'success' : 'info'" effect="plain" round>{{ iface.name }}</el-tag></span>
                <span class="net-val mono">{{ ip }}<el-icon class="copy-btn" @click="copyText(ip.split('/')[0])"><CopyDocument /></el-icon></span>
              </div>
            </template>
            <div class="net-row" v-if="stats.host?.dnsServers?.length">
              <span class="net-label">DNS</span>
              <span class="net-val mono">{{ stats.host.dnsServers.join(', ') }}<el-icon class="copy-btn" @click="copyText(stats.host.dnsServers.join(', '))"><CopyDocument /></el-icon></span>
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

        <div class="tri-sep"></div>

        <!-- 右列：系统信息 -->
        <div class="tri-col">
          <div class="col-hd"><el-icon><Monitor /></el-icon><span>{{ t('home.systemInfo') }}</span></div>
          <div class="sys-list">
            <div class="sys-row" v-for="item in sysInfoItems" :key="item.label">
              <span class="sys-label">{{ item.label }}</span>
              <span class="sys-val" :title="item.value">
                {{ item.value }}
                <el-icon class="copy-btn" @click="copyText(item.value)" v-if="item.value && item.value !== '-'"><CopyDocument /></el-icon>
              </span>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 快速入口 + Top 进程 -->
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
import { ref, computed, watch, onMounted, onUnmounted, markRaw } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useGlobalStore } from '@/store/modules/global'
import { getSystemStats } from '@/api/modules/monitor'
import { ElMessage } from 'element-plus'
import type { SystemStats, HostInfo } from '@/api/interface'
import {
  Monitor, Cpu, Coin, Odometer, Connection,
  Box, Compass, DataLine, CopyDocument,
} from '@element-plus/icons-vue'
import ShieldIcon from '@/components/icons/ShieldIcon.vue'

const router = useRouter()
const { t } = useI18n()
const globalStore = useGlobalStore()
const stats = ref<Partial<SystemStats>>({})
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  try { const r = await getSystemStats(); stats.value = r.data || {} } catch {}
}

const refreshInterval = computed(() => globalStore.dashboardRefreshInterval ?? 5000)

const resetTimer = () => {
  if (timer) { clearInterval(timer); timer = null }
  const ms = refreshInterval.value
  if (ms > 0) timer = setInterval(loadStats, ms)
}

watch(refreshInterval, resetTimer)

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
  { path: '/host/firewall', title: t('menu.firewall'), icon: markRaw(ShieldIcon) },
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

onMounted(() => { loadStats(); resetTimer() })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>

<style lang="scss" scoped>
.dashboard { padding: 0; }
.dash-card {
  margin-bottom: 16px;
  border-left-width: 3px;
}

.card-hd {
  display: flex; align-items: center; gap: 8px;
  font-weight: 600; font-size: 14px; color: var(--xp-text-primary);
  .el-icon { color: var(--xp-accent); opacity: 0.8; }
}

/* ==================== Tri-column grid ==================== */
.tri-grid {
  display: grid;
  grid-template-columns: 1fr auto 1fr auto 1fr;
  gap: 0;
}

.tri-col {
  min-width: 0;
  padding: 0 20px;
  &:first-child { padding-left: 0; }
  &:last-child { padding-right: 0; }
}

.tri-sep {
  width: 1px; align-self: stretch;
  background: var(--xp-border-light);
}

.col-hd {
  display: flex; align-items: center; gap: 8px;
  font-weight: 600; font-size: 13px; color: var(--xp-text-primary);
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--xp-border-light);
  .el-icon { color: var(--xp-accent); opacity: 0.8; }
}

/* ==================== Left: Resources ==================== */
.res-list {
  display: flex; flex-direction: column; gap: 16px;
}

.res-item { /* each resource block */ }

.res-hd {
  display: flex; align-items: center; gap: 8px; margin-bottom: 8px;
  span:first-of-type { font-size: 13px; font-weight: 600; color: var(--xp-text-primary); flex: 1; }
}

.res-dot {
  width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0;
}
.cpu-dot { background: var(--xp-accent); }
.mem-dot { background: #818cf8; }
.load-dot { background: #34d399; }
.disk-dot { background: #60a5fa; }

.res-pct {
  font-size: 18px; font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.c-ok { color: var(--xp-accent); }
.c-warn { color: #f59e0b; }
.c-danger { color: #ef4444; }

.bar-bg {
  width: 100%; height: 6px;
  background: var(--xp-progress-trail, rgba(255,255,255,0.06));
  border-radius: 3px; overflow: hidden; margin-bottom: 6px;
}

.bar-fg {
  height: 100%; border-radius: 3px; min-width: 2px;
  transition: width .8s cubic-bezier(.4,0,.2,1), background .4s ease;
}

.res-foot { font-size: 11px; color: var(--xp-text-secondary); }
.res-sub { font-size: 11px; color: var(--xp-text-muted); margin-top: 2px; }
.load-triple { display: flex; gap: 10px; }

/* ==================== Center: Network ==================== */
.net-list {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 10px;
  align-items: baseline;
}

.net-row {
  display: contents;
}

.net-label {
  font-size: 12px; color: var(--xp-text-muted);
  white-space: nowrap;
}

.net-val {
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
.net-row:hover .copy-btn,
.sys-row:hover .copy-btn { opacity: 1; }

.traffic-tbl {
  width: 100%; border-collapse: collapse; margin-top: 12px;
  font-size: 12px; font-variant-numeric: tabular-nums;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;

  th, td { padding: 3px 0; }
  th { font-weight: 500; color: var(--xp-text-muted); font-size: 11px; }
  th:first-child, td:first-child { text-align: left; }

  .col-up { text-align: right; width: 90px; color: var(--xp-color-up, #34d399); }
  .col-down { text-align: right; width: 90px; color: var(--xp-color-down, #a78bfa); }
  .td-nic { color: var(--xp-text-secondary); font-weight: 500; }
  .tr-total td {
    border-top: 1px solid var(--xp-border-light); padding-top: 5px;
    font-size: 11px; color: var(--xp-text-muted);
  }
}

/* ==================== Right: System Info ==================== */
.sys-list {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 10px;
  align-items: baseline;
}

.sys-row {
  display: contents;
}

.sys-label {
  font-size: 12px; color: var(--xp-text-muted);
  white-space: nowrap; flex-shrink: 0;
}

.sys-val {
  font-size: 13px; color: var(--xp-text-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  display: inline-flex; align-items: center; gap: 4px; min-width: 0;
}

/* ==================== Quick & Process ==================== */
.quick-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px;
}

.quick-item {
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  padding: 16px 8px;
  background: var(--xp-bg-inset); border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius); cursor: pointer;
  transition: border-color .2s, background .2s;

  &:hover {
    border-color: rgba(var(--xp-accent-rgb, 65, 251, 68), 0.25);
    background: rgba(var(--xp-accent-rgb, 65, 251, 68), 0.04);
    .qi-icon { color: var(--xp-accent); }
    .qi-label { color: var(--xp-text-primary); }
  }
}

.qi-icon {
  width: 40px; height: 40px;
  display: flex; align-items: center; justify-content: center;
  background: rgba(255,255,255,.04); border-radius: 12px;
  color: var(--xp-text-secondary); transition: color .2s;
}

.qi-label {
  font-size: 12px; color: var(--xp-text-secondary);
  font-weight: 500; text-align: center; transition: color .2s;
}

.text-danger { color: #ef4444; font-weight: 600; }

/* ==================== Responsive ==================== */
@media (max-width: 1200px) {
  .tri-grid {
    grid-template-columns: 1fr;
    gap: 0;
  }
  .tri-col { padding: 0; }
  .tri-col + .tri-sep { display: none; }
  .tri-sep { display: none; }
  .tri-col + .tri-col {
    margin-top: 18px;
    padding-top: 18px;
    border-top: 1px solid var(--xp-border-light);
  }
}

@media (max-width: 768px) {
  .quick-grid { grid-template-columns: repeat(3, 1fr); }
}

@media (max-width: 480px) {
  .quick-grid { grid-template-columns: repeat(2, 1fr); }
}
</style>
