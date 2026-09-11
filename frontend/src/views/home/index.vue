<template>
  <div class="dashboard xp-page-shell">
    <div class="xp-hero overview-hero">
      <div>
        <div class="xp-hero-eyebrow">{{ t('home.overview') }}</div>
        <h2>{{ stats.host?.hostname || globalStore.panelName || 'X-Panel' }}</h2>
        <p>{{ systemSummary }}</p>
        <div class="hero-chips">
          <span v-for="chip in hostChips" :key="chip.label" class="hero-chip">
            <em>{{ chip.label }}</em>{{ chip.value }}
          </span>
        </div>
      </div>
      <div class="hero-aside">
        <div class="xp-hero-meta">{{ t('home.refreshEvery', { seconds: Math.round(refreshInterval / 1000) }) }}</div>
        <router-link class="history-link" to="/host/monitor">{{ t('home.viewHistory') }}</router-link>
      </div>
    </div>

    <div class="gauge-deck">
      <article v-for="item in gauges" :key="item.label" class="gauge-card" :class="item.tone">
        <GaugeMeter
          :label="item.label"
          :percent="item.pct"
          :sub="item.sub"
          :tone="item.tone"
          :value="item.value"
          :color="item.color"
        />
      </article>
    </div>

    <div class="overview-grid" :class="{ 'has-storage': showStorage }">
      <article v-if="showStorage" class="deck-card storage-card">
        <div class="col-hd"><el-icon><Box /></el-icon><span>{{ t('home.storageAndSensors') }}</span></div>
        <div class="res-list">
          <div v-if="(stats.memory?.swapTotal ?? 0) > 0" class="res-item">
            <div class="res-hd">
              <div class="res-dot mem-dot"></div>
              <span>{{ t('home.swap') }}</span>
              <span class="res-pct" :class="pctCls(stats.memory?.swapPercent)">{{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%</span>
            </div>
            <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.memory?.swapPercent, 'mem')"></div></div>
            <div class="res-foot">{{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }}</div>
          </div>
          <div v-for="disk in extraDisks" :key="disk.mountPoint" class="res-item">
            <div class="res-hd">
              <div class="res-dot disk-dot"></div>
              <span>{{ disk.mountPoint }}</span>
              <span class="res-pct" :class="pctCls(disk.usedPercent)">{{ disk.usedPercent.toFixed(1) }}%</span>
            </div>
            <div class="bar-bg"><div class="bar-fg" :style="barSty(disk.usedPercent, 'disk')"></div></div>
            <div class="res-foot">{{ disk.device }} · {{ disk.fsType }} · {{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}</div>
          </div>
          <div v-if="stats.sensors?.length" class="res-item">
            <div class="res-hd"><div class="res-dot temp-dot"></div><span>{{ t('home.sensorTemp') }}</span></div>
            <div class="temp-grid">
              <div v-for="s in stats.sensors" :key="s.key" class="temp-cell" :title="s.key">
                <span class="temp-name">{{ fmtSensorName(s.key) }}</span>
                <span class="temp-val" :class="tempCls(s)">{{ s.temp.toFixed(0) }}°C</span>
              </div>
            </div>
          </div>
        </div>
      </article>

      <article class="deck-card net-card">
        <div class="col-hd"><el-icon><Connection /></el-icon><span>{{ t('home.network') }}</span></div>
        <div class="net-hero">
          <div class="net-rate">
            <em>{{ t('home.download') }}</em>
            <strong class="col-down">{{ formatSpeed(headlineNic?.speedDown) }}</strong>
            <small>{{ headlineNic?.name || t('home.noNetworkData') }}</small>
          </div>
          <div class="net-rate">
            <em>{{ t('home.upload') }}</em>
            <strong class="col-up">{{ formatSpeed(headlineNic?.speedUp) }}</strong>
            <small>{{ t('home.totalTraffic') }} {{ formatBytes(stats.network?.bytesRecv) }} / {{ formatBytes(stats.network?.bytesSent) }}</small>
          </div>
        </div>
        <div v-if="stats.host?.publicIPv4" class="addr-plate" @click="copyText(stats.host.publicIPv4)">
          <span>{{ t('home.publicIPv4') }}</span>
          <strong>{{ stats.host.publicIPv4 }}</strong>
          <el-icon class="copy-btn visible"><CopyDocument /></el-icon>
        </div>
        <div class="net-list">
          <div v-if="stats.host?.publicIPv6" class="net-row">
            <span class="net-label">{{ t('home.publicIPv6') }}</span>
            <span class="net-val mono">{{ stats.host.publicIPv6 }}<el-icon class="copy-btn" @click="copyText(stats.host.publicIPv6)"><CopyDocument /></el-icon></span>
          </div>
          <template v-for="iface in stats.host?.interfaces" :key="iface.name">
            <div v-for="ip in iface.ipv4" :key="ip" class="net-row">
              <span class="net-label"><el-tag size="small" :type="iface.status === 'up' ? 'success' : 'info'" effect="plain">{{ iface.name }}</el-tag></span>
              <span class="net-val mono">{{ ip }}<el-icon class="copy-btn" @click="copyText(ip.split('/')[0])"><CopyDocument /></el-icon></span>
            </div>
          </template>
          <div v-if="stats.host?.dnsServers?.length" class="net-row">
            <span class="net-label">DNS</span>
            <span class="net-val mono">{{ stats.host.dnsServers.join(', ') }}<el-icon class="copy-btn" @click="copyText(stats.host.dnsServers.join(', '))"><CopyDocument /></el-icon></span>
          </div>
        </div>
        <table v-if="mainNics.length" class="traffic-tbl">
          <thead><tr><th></th><th class="col-up">{{ t('home.upload') }}</th><th class="col-down">{{ t('home.download') }}</th></tr></thead>
          <tbody>
            <tr v-for="nic in mainNics" :key="nic.name">
              <td class="td-nic">{{ nic.name }}</td>
              <td class="col-up">{{ formatSpeed(nic.speedUp) }}</td>
              <td class="col-down">{{ formatSpeed(nic.speedDown) }}</td>
            </tr>
          </tbody>
        </table>
      </article>

      <article class="deck-card sys-card">
        <div class="col-hd"><el-icon><Monitor /></el-icon><span>{{ t('home.systemInfo') }}</span></div>
        <div class="sys-list">
          <div v-for="item in sysInfoItems" :key="item.label" class="sys-row">
            <span class="sys-label">{{ item.label }}</span>
            <span class="sys-val" :title="item.value">
              {{ item.value }}
              <el-icon v-if="item.value && item.value !== '-'" class="copy-btn" @click="copyText(item.value)"><CopyDocument /></el-icon>
            </span>
          </div>
        </div>
      </article>
    </div>

    <article class="deck-card">
      <div class="col-hd"><el-icon><Compass /></el-icon><span>{{ t('home.quickEntry') }}</span></div>
      <div class="quick-rail">
        <button
          v-for="entry in quickEntries"
          :key="entry.path"
          type="button"
          class="quick-chip"
          @click="router.push(entry.path)"
        >
          <el-icon :size="16"><component :is="entry.icon" /></el-icon>
          <span>{{ entry.title }}</span>
        </button>
      </div>
    </article>

    <article class="deck-card">
      <div class="col-hd"><el-icon><DataLine /></el-icon><span>{{ t('home.topProcess') }}</span></div>
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
    </article>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, markRaw } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useGlobalStore } from '@/store/modules/global'
import { getSystemStats } from '@/api/modules/monitor'
import { ElMessage } from 'element-plus'
import type { SystemStats, HostInfo, SensorTemp } from '@/api/interface'
import {
  Monitor, Connection, Compass, DataLine, CopyDocument, Box,
} from '@element-plus/icons-vue'
import ShieldIcon from '@/components/icons/ShieldIcon.vue'
import { chartTokens, onAppearanceChange } from '@/theme'
import GaugeMeter from './GaugeMeter.vue'

const router = useRouter()
const { t } = useI18n()
const globalStore = useGlobalStore()
const stats = ref<Partial<SystemStats>>({})
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  try { const r = await getSystemStats(); stats.value = r.data || {} } catch {}
}

const refreshInterval = computed(() => globalStore.dashboardRefreshInterval ?? 5000)

const systemSummary = computed(() => {
  const h = stats.value.host ?? ({} as Partial<HostInfo>)
  const os = `${h.platform || ''} ${h.platformVersion || ''}`.trim()
  return [os, h.kernelArch, h.virtualization].filter(Boolean).join(' · ') || t('home.waitingForData')
})

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
    { label: t('home.kernel'), value: h.kernelVersion || '-' },
    { label: t('home.cpuModel'), value: stats.value.cpu?.modelName || '-' },
    { label: t('home.cpuCores'), value: stats.value.cpu ? `${stats.value.cpu.cores} ${t('home.physical')} / ${stats.value.cpu.logicalCores} ${t('home.logical')}` : '-' },
    { label: t('home.totalMemory'), value: formatBytes(stats.value.memory?.total) },
    { label: t('home.tcpCongestion'), value: h.tcpCongestion || '-' },
  ]
})

const loadPct = computed(() => {
  const c = stats.value.cpu?.logicalCores || 1
  return Math.min(((stats.value.load?.load1 || 0) / c) * 100, 100)
})

const hostChips = computed(() => {
  const h = stats.value.host ?? ({} as Partial<HostInfo>)
  return [
    { label: t('home.uptime'), value: formatUptime(stats.value.uptime) },
    { label: t('home.arch'), value: h.kernelArch || '-' },
    { label: t('home.virtualization'), value: h.virtualization || t('home.physicalMachine') },
    { label: t('home.timezone'), value: h.timezone || '-' },
  ]
})

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

const primaryDisk = computed(() => {
  const disks = filteredDisks.value
  return disks.find((d) => d.mountPoint === '/') || disks.slice().sort((a, b) => b.total - a.total)[0] || null
})

const extraDisks = computed(() => {
  const primary = primaryDisk.value
  return filteredDisks.value.filter((d) => d.mountPoint !== primary?.mountPoint)
})

const showStorage = computed(() =>
  extraDisks.value.length > 0
  || (stats.value.memory?.swapTotal ?? 0) > 0
  || Boolean(stats.value.sensors?.length)
)

const mainNics = computed(() => (stats.value.netIO || []).filter(n => n.name !== 'lo').slice(0, 6))

const headlineNic = computed(() => {
  const nics = mainNics.value
  if (!nics.length) return null
  return nics.reduce((best, nic) =>
    (nic.speedDown + nic.speedUp) > (best.speedDown + best.speedUp) ? nic : best
  )
})

const gauges = computed(() => {
  themeTick.value
  const palette = metricPalette()
  const cpu = stats.value.cpu?.usagePercent ?? 0
  const mem = stats.value.memory?.usedPercent ?? 0
  const load = loadPct.value
  const disk = primaryDisk.value
  return [
    {
      label: 'CPU',
      pct: cpu,
      value: fmtPct(cpu),
      sub: stats.value.cpu ? `${stats.value.cpu.logicalCores} ${t('home.logical')}` : '-',
      tone: pctCls(cpu),
      color: palette.cpu,
    },
    {
      label: t('home.memory'),
      pct: mem,
      value: fmtPct(mem),
      sub: `${formatBytes(stats.value.memory?.used)} / ${formatBytes(stats.value.memory?.total)}`,
      tone: pctCls(mem),
      color: palette.mem,
    },
    {
      label: t('home.load'),
      pct: load,
      value: fmtPct(load),
      sub: `1m ${stats.value.load?.load1?.toFixed(2) || '-'} · 5m ${stats.value.load?.load5?.toFixed(2) || '-'} · 15m ${stats.value.load?.load15?.toFixed(2) || '-'}`,
      tone: pctCls(load),
      color: palette.load,
    },
    {
      label: disk?.mountPoint || t('home.diskUsage'),
      pct: disk?.usedPercent ?? 0,
      value: disk ? `${disk.usedPercent.toFixed(1)}%` : '—',
      sub: disk ? `${formatBytes(disk.used)} / ${formatBytes(disk.total)}` : '-',
      tone: pctCls(disk?.usedPercent),
      color: palette.disk,
    },
  ]
})

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
  catch { ElMessage.error(t('commons.copyFailed')) }
}

const themeTick = ref(0)
const metricPalette = () => {
  const tokens = chartTokens()
  return { cpu: tokens.accent, mem: tokens.secondary, load: tokens.success, disk: tokens.info }
}

const barColor = (pct: number, type: string) => {
  const tokens = chartTokens()
  if (pct >= 90) return tokens.danger
  if (pct >= 70) return tokens.warning
  const palette = metricPalette()
  return palette[type as keyof typeof palette] || tokens.accent
}

const barSty = (pct?: number, type = 'cpu') => {
  themeTick.value
  const v = Math.min(pct || 0, 100); const c = barColor(v, type)
  return { width: `${v}%`, background: `linear-gradient(90deg, ${c}cc, ${c})` }
}

const pctCls = (pct?: number) => (pct || 0) >= 90 ? 'c-danger' : (pct || 0) >= 70 ? 'c-warn' : 'c-ok'
const fmtPct = (v?: number) => `${(v ?? 0).toFixed(1)}%`

const fmtSensorName = (key: string) => key.replace(/_/g, ' ')
const tempCls = (s: SensorTemp) => {
  const high = s.high && s.high > 0 ? s.high : 80
  const warn = Math.min(high - 15, 65)
  if (s.temp >= high) return 'c-danger'
  if (s.temp >= warn) return 'c-warn'
  return 'c-ok'
}

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

let stopAppearance: (() => void) | undefined
onMounted(() => {
  loadStats()
  resetTimer()
  stopAppearance = onAppearanceChange(() => { themeTick.value++ })
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
  stopAppearance?.()
})
</script>

<style lang="scss" scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.overview-hero {
  align-items: flex-start;
  margin-bottom: 0;
}

.hero-aside {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 8px;
}

.history-link {
  color: var(--xp-accent);
  font-size: 12px;
  text-decoration: none;
  &:hover { text-decoration: underline; }
}

.hero-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.hero-chip {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  padding: 4px 10px;
  color: var(--xp-text-primary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);

  em {
    color: var(--xp-text-muted);
    font-style: normal;
    font-size: 11px;
  }
}

.gauge-deck {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.gauge-card,
.deck-card {
  position: relative;
  overflow: hidden;
  padding: 16px 18px;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
}

.gauge-card {
  padding: 12px 12px 16px;
  contain: layout paint;
}

.overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.08fr) minmax(0, 0.92fr);
  gap: 16px;

  &.has-storage {
    grid-template-columns: minmax(280px, 0.9fr) minmax(0, 1.1fr) minmax(0, 0.95fr);
  }
}

.col-hd {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  padding-bottom: 10px;
  color: var(--xp-text-primary);
  font-size: 13px;
  font-weight: 600;
  border-bottom: 1px solid var(--xp-border-light);
  .el-icon { color: var(--xp-accent); opacity: 0.85; }
}

.storage-card {
  background:
    linear-gradient(180deg, color-mix(in srgb, var(--xp-info) 7%, var(--xp-bg-surface)), var(--xp-bg-surface));
}

.net-card {
  background:
    radial-gradient(420px 180px at 108% -8%, color-mix(in srgb, var(--xp-info) 22%, transparent), transparent 58%),
    linear-gradient(165deg, color-mix(in srgb, var(--xp-info) 8%, var(--xp-bg-surface)), var(--xp-bg-surface));

  &::before {
    content: '';
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(color-mix(in srgb, var(--xp-text-primary) 7%, transparent) 1px, transparent 1px),
      linear-gradient(90deg, color-mix(in srgb, var(--xp-text-primary) 7%, transparent) 1px, transparent 1px);
    background-size: 22px 22px;
    mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.28), transparent 72%);
    pointer-events: none;
  }
}

.sys-card {
  background:
    radial-gradient(280px 160px at -8% 112%, color-mix(in srgb, var(--xp-accent) 20%, transparent), transparent 60%),
    linear-gradient(155deg, color-mix(in srgb, var(--xp-accent) 8%, var(--xp-bg-surface)), var(--xp-bg-surface));

  &::after {
    content: '';
    position: absolute;
    right: -36px;
    bottom: -42px;
    width: 150px;
    height: 150px;
    border: 16px solid color-mix(in srgb, var(--xp-accent) 14%, transparent);
    border-radius: 50%;
    pointer-events: none;
  }
}

.net-hero {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 14px;
}

.net-rate {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--xp-bg-inset) 82%, transparent);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);

  em {
    color: var(--xp-text-muted);
    font-size: 11px;
    font-style: normal;
  }

  strong {
    font-size: 20px;
    font-weight: 800;
    letter-spacing: -0.03em;
    font-variant-numeric: tabular-nums;
    line-height: 1.1;
  }

  small {
    color: var(--xp-text-muted);
    font-size: 11px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.addr-plate {
  position: relative;
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin-bottom: 14px;
  padding: 10px 12px;
  color: var(--xp-text-primary);
  background: color-mix(in srgb, var(--xp-accent) 10%, var(--xp-bg-inset));
  border: 1px solid color-mix(in srgb, var(--xp-accent) 28%, var(--xp-border-light));
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  span {
    color: var(--xp-text-muted);
    font-size: 11px;
    white-space: nowrap;
  }

  strong {
    flex: 1;
    min-width: 0;
    font-family: var(--xp-font-mono);
    font-size: 16px;
    font-weight: 700;
    letter-spacing: 0.02em;
  }
}

.res-list { display: flex; flex-direction: column; gap: 14px; }
.res-hd {
  display: flex; align-items: center; gap: 8px; margin-bottom: 8px;
  span:first-of-type { flex: 1; color: var(--xp-text-primary); font-size: 13px; font-weight: 600; }
}
.res-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.mem-dot { background: var(--xp-accent-secondary); }
.disk-dot { background: var(--xp-info); }
.temp-dot { background: var(--xp-warning); }
.res-pct { font-size: 15px; font-weight: 700; font-variant-numeric: tabular-nums; }
.c-ok { color: var(--xp-success); }
.c-warn { color: var(--xp-warning); }
.c-danger { color: var(--xp-danger); }
.bar-bg {
  width: 100%; height: 5px; margin-bottom: 6px;
  background: color-mix(in srgb, var(--xp-text-primary) 8%, transparent);
  border-radius: 99px; overflow: hidden;
}
.bar-fg { height: 100%; min-width: 2px; border-radius: 99px; transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1); }
.res-foot { font-size: 11px; color: var(--xp-text-secondary); }

.temp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(128px, 1fr));
  gap: 4px 12px;
}
.temp-cell { display: flex; align-items: baseline; justify-content: space-between; gap: 6px; min-width: 0; }
.temp-name { color: var(--xp-text-secondary); font-size: 11px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.temp-val { flex-shrink: 0; font-size: 13px; font-weight: 700; font-variant-numeric: tabular-nums; }

.net-list {
  position: relative;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 6px 10px;
  align-items: baseline;
}
.net-row { display: contents; }
.net-label { color: var(--xp-text-muted); font-size: 12px; white-space: nowrap; }
.net-val {
  display: inline-flex; align-items: center; gap: 4px; min-width: 0;
  color: var(--xp-text-primary); font-size: 13px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  &.mono { font-family: var(--xp-font-mono); font-size: 12px; }
}

.copy-btn {
  flex-shrink: 0; color: var(--xp-text-muted); opacity: 0; cursor: pointer;
  transition: opacity 0.15s, color 0.15s;
  &:hover { color: var(--xp-accent); }
  &.visible { opacity: 0.7; }
}
.net-row:hover .copy-btn,
.sys-row:hover .copy-btn,
.addr-plate:hover .copy-btn { opacity: 1; }

.traffic-tbl {
  position: relative;
  width: 100%;
  margin-top: 12px;
  border-collapse: collapse;
  font-family: var(--xp-font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  th, td { padding: 3px 0; }
  th { color: var(--xp-text-muted); font-size: 11px; font-weight: 500; }
  th:first-child, td:first-child { text-align: left; }
  .td-nic { color: var(--xp-text-secondary); font-weight: 500; }
}

.col-up { text-align: right; color: var(--xp-color-up, var(--xp-success)); }
.col-down { text-align: right; color: var(--xp-color-down, var(--xp-accent-secondary)); }

.sys-list {
  position: relative;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px 12px;
  align-items: baseline;
}
.sys-row { display: contents; }
.sys-label { color: var(--xp-text-muted); font-size: 12px; white-space: nowrap; }
.sys-val {
  display: inline-flex; align-items: center; gap: 4px; min-width: 0;
  color: var(--xp-text-primary); font-size: 13px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

.quick-rail { display: flex; flex-wrap: wrap; gap: 8px; }
.quick-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: var(--xp-control-height);
  padding: 0 12px;
  color: var(--xp-text-secondary);
  background: var(--xp-bg-inset);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;
  &:hover {
    color: var(--xp-text-primary);
    background: var(--xp-accent-muted);
    border-color: var(--xp-accent);
  }
}

.text-danger { color: var(--xp-danger); font-weight: 600; }

@media (max-width: 1440px) {
  .gauge-deck {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .gauge-card {
    padding: 10px 12px;
    --gauge-stroke: 7;

    :deep(.gauge-meter) {
      flex-direction: row;
      align-items: center;
      gap: 8px 12px;
    }

    :deep(.gauge-well) {
      width: 42%;
      max-width: 128px;
      margin: 0;
      flex-shrink: 0;
    }

    :deep(.gauge-caption) {
      align-items: flex-start;
      margin-top: 0;
      text-align: left;
      min-width: 0;
    }

    :deep(.gauge-sub) { white-space: normal; }
    :deep(.gauge-readout strong) { font-size: 18px; }
  }
}

@media (max-width: 1200px) {
  .overview-grid,
  .overview-grid.has-storage {
    grid-template-columns: 1fr;
  }

  .storage-card { grid-column: auto; }
}

@media (max-width: 800px) {
  .dashboard { gap: 12px; }
  .hero-aside { align-items: flex-start; }
  .hero-chips { gap: 6px; }
  .hero-chip { padding: 3px 8px; font-size: 11px; }
  .net-hero { grid-template-columns: 1fr 1fr; gap: 8px; }
  .net-rate strong { font-size: 16px; }
  .addr-plate {
    flex-wrap: wrap;
    strong { font-size: 13px; overflow-wrap: anywhere; }
  }
  .sys-label { white-space: normal; }
  .traffic-tbl { display: block; overflow-x: auto; }
  .quick-chip { min-height: 32px; padding: 0 10px; font-size: 12px; }
}

@media (max-width: 560px) {
  .gauge-deck { gap: 8px; }
  .gauge-card { padding: 8px; }
  .net-hero { grid-template-columns: 1fr; }
}
</style>
