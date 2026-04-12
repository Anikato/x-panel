<template>
  <div class="monitor-page">
    <div class="page-header">
      <h3>{{ $t('monitor.title') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadStats" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <!-- 三列概览：资源占用 | 网络 | 进程 Top -->
    <el-card shadow="never" class="dash-card">
      <div class="tri-grid">
        <!-- 左列：资源占用 -->
        <div class="tri-col">
          <div class="col-hd"><el-icon><Odometer /></el-icon><span>{{ $t('home.resourceUsage') }}</span></div>
          <div class="res-list">
            <div class="res-item">
              <div class="res-hd"><div class="res-dot cpu-dot"></div><span>CPU</span><span class="res-pct" :class="pctCls(stats.cpu?.usagePercent)">{{ fmtPct(stats.cpu?.usagePercent) }}</span></div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.cpu?.usagePercent, 'cpu')"></div></div>
              <div class="res-foot">{{ stats.cpu?.cores }} {{ $t('home.physical') }} / {{ stats.cpu?.logicalCores }} {{ $t('home.logical') }}</div>
            </div>
            <div class="res-item">
              <div class="res-hd"><div class="res-dot mem-dot"></div><span>{{ $t('home.memory') }}</span><span class="res-pct" :class="pctCls(stats.memory?.usedPercent)">{{ fmtPct(stats.memory?.usedPercent) }}</span></div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.memory?.usedPercent, 'mem')"></div></div>
              <div class="res-foot">{{ formatBytes(stats.memory?.used) }} / {{ formatBytes(stats.memory?.total) }}</div>
              <div class="res-sub" v-if="(stats.memory?.swapTotal ?? 0) > 0">Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }} ({{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%)</div>
            </div>
            <div class="res-item">
              <div class="res-hd"><div class="res-dot load-dot"></div><span>{{ $t('home.load') }}</span><span class="res-pct" :class="pctCls(loadPct)">{{ loadPct.toFixed(0) }}%</span></div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(loadPct, 'load')"></div></div>
              <div class="res-foot load-triple"><span>1m: {{ stats.load?.load1?.toFixed(2) || '-' }}</span><span>5m: {{ stats.load?.load5?.toFixed(2) || '-' }}</span><span>15m: {{ stats.load?.load15?.toFixed(2) || '-' }}</span></div>
            </div>
            <template v-for="disk in filteredDisks" :key="disk.mountPoint">
              <div class="res-item">
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
          <div class="col-hd"><el-icon><Connection /></el-icon><span>{{ $t('monitor.network') }}</span></div>
          <div class="net-list">
            <div class="net-row" v-if="stats.host?.publicIPv4">
              <span class="net-label">{{ $t('home.publicIPv4') }}</span>
              <span class="net-val accent">{{ stats.host.publicIPv4 }}<el-icon class="copy-btn" @click="copyText(stats.host.publicIPv4)"><CopyDocument /></el-icon></span>
            </div>
            <template v-for="iface in stats.host?.interfaces" :key="iface.name">
              <div class="net-row" v-for="ip in iface.ipv4" :key="ip">
                <span class="net-label"><el-tag size="small" :type="iface.status === 'up' ? 'success' : 'info'" effect="plain" round>{{ iface.name }}</el-tag></span>
                <span class="net-val mono">{{ ip }}</span>
              </div>
            </template>
            <div class="net-row" v-if="stats.host?.dnsServers?.length">
              <span class="net-label">DNS</span>
              <span class="net-val mono">{{ stats.host.dnsServers.join(', ') }}</span>
            </div>
          </div>
          <table class="traffic-tbl" v-if="mainNics.length">
            <thead><tr><th></th><th class="col-up">{{ $t('home.upload') }}</th><th class="col-down">{{ $t('home.download') }}</th></tr></thead>
            <tbody>
              <tr v-for="nic in mainNics" :key="nic.name">
                <td class="td-nic">{{ nic.name }}</td>
                <td class="col-up">{{ formatSpeed(nic.speedUp) }}</td>
                <td class="col-down">{{ formatSpeed(nic.speedDown) }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr class="tr-total"><td>{{ $t('home.totalTraffic') }}</td><td class="col-up">{{ formatBytes(stats.network?.bytesSent) }}</td><td class="col-down">{{ formatBytes(stats.network?.bytesRecv) }}</td></tr>
            </tfoot>
          </table>
        </div>

        <div class="tri-sep"></div>

        <!-- 右列：Top 进程 -->
        <div class="tri-col">
          <div class="col-hd"><el-icon><DataLine /></el-icon><span>{{ $t('home.topProcess') }}</span></div>
          <el-table :data="stats.topProcess || []" size="small" stripe max-height="400">
            <el-table-column prop="pid" label="PID" width="60" />
            <el-table-column prop="name" :label="$t('home.processName')" min-width="100" show-overflow-tooltip />
            <el-table-column label="CPU" width="70" align="right">
              <template #default="{ row }"><span :class="row.cpuPercent > 50 ? 'text-danger' : ''">{{ row.cpuPercent.toFixed(1) }}%</span></template>
            </el-table-column>
            <el-table-column :label="$t('home.memoryUsage')" width="80" align="right">
              <template #default="{ row }">{{ formatBytes(row.memRss) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </el-card>

    <!-- 磁盘详情表 -->
    <el-card shadow="never" class="dash-card">
      <template #header>
        <div class="card-hd"><el-icon><Box /></el-icon><span>{{ $t('home.diskUsage') }}</span></div>
      </template>
      <el-table :data="stats.disks || []" size="small" stripe>
        <el-table-column prop="mountPoint" :label="$t('disk.mountPoint')" min-width="100" show-overflow-tooltip />
        <el-table-column prop="device" :label="$t('disk.device')" min-width="100" show-overflow-tooltip />
        <el-table-column prop="fsType" :label="$t('disk.fsType')" width="70" />
        <el-table-column :label="$t('monitor.usage')" min-width="160">
          <template #default="{ row }">
            <div class="bar-bg"><div class="bar-fg" :style="barSty(row.usedPercent, 'disk')"></div></div>
          </template>
        </el-table-column>
        <el-table-column :label="$t('monitor.used')" width="80" align="right">
          <template #default="{ row }">
            <span class="res-pct-sm" :class="pctCls(row.usedPercent)">{{ Math.round(row.usedPercent) }}%</span>
          </template>
        </el-table-column>
        <el-table-column label="" width="150" align="right">
          <template #default="{ row }">
            {{ formatBytes(row.used) }} / {{ formatBytes(row.total) }}
          </template>
        </el-table-column>
        <el-table-column label="Inode" width="70" align="right">
          <template #default="{ row }">
            <span v-if="row.inodesTotal">{{ Math.round(row.inodesPercent) }}%</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh, CopyDocument, Odometer, Connection, DataLine, Box } from '@element-plus/icons-vue'
import { getSystemStats } from '@/api/modules/monitor'
import { ElMessage } from 'element-plus'
import type { SystemStats } from '@/api/interface'

const { t } = useI18n()
const loading = ref(false)
const stats = ref<Partial<SystemStats>>({})
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  loading.value = true
  try {
    const res = await getSystemStats()
    stats.value = res.data || {}
  } catch { /* */ }
  finally { loading.value = false }
}

const loadPct = computed(() => {
  const c = stats.value.cpu?.logicalCores || 1
  return Math.min(((stats.value.load?.load1 || 0) / c) * 100, 100)
})

const mainNics = computed(() => (stats.value.netIO || []).filter(n => n.name !== 'lo').slice(0, 8))

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

const copyText = async (text: string) => {
  if (!text) return
  try { await navigator.clipboard.writeText(text); ElMessage.success(t('commons.copy') + ' ✓') }
  catch { /* */ }
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
.monitor-page { height: 100%; }

.page-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 16px;
  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
}

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

/* ==================== Resources ==================== */
.res-list { display: flex; flex-direction: column; gap: 16px; }

.res-hd {
  display: flex; align-items: center; gap: 8px; margin-bottom: 8px;
  span:first-of-type { font-size: 13px; font-weight: 600; color: var(--xp-text-primary); flex: 1; }
}

.res-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.cpu-dot { background: var(--xp-accent); }
.mem-dot { background: #818cf8; }
.load-dot { background: #34d399; }
.disk-dot { background: #60a5fa; }

.res-pct { font-size: 18px; font-weight: 700; font-variant-numeric: tabular-nums; }
.res-pct-sm { font-size: 13px; font-weight: 700; font-variant-numeric: tabular-nums; }

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

/* ==================== Network ==================== */
.net-list {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 10px;
  align-items: baseline;
}
.net-row { display: contents; }
.net-label { font-size: 12px; color: var(--xp-text-muted); white-space: nowrap; }
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
.net-row:hover .copy-btn { opacity: 1; }

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
  .tr-total td { border-top: 1px solid var(--xp-border-light); padding-top: 5px; font-size: 11px; color: var(--xp-text-muted); }
}

.text-danger { color: #ef4444; font-weight: 600; }

/* ==================== Responsive ==================== */
@media (max-width: 1200px) {
  .tri-grid { grid-template-columns: 1fr; gap: 0; }
  .tri-col { padding: 0; }
  .tri-sep { display: none; }
  .tri-col + .tri-col { margin-top: 18px; padding-top: 18px; border-top: 1px solid var(--xp-border-light); }
}
</style>
