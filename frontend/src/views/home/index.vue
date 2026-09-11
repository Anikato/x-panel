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
      <article
        v-for="item in gauges"
        :key="item.label"
        class="gauge-card"
        :class="item.tone"
        :style="{ '--gauge-base': item.color }"
      >
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
          <div class="disk-stack">
            <div v-for="disk in filteredDisks" :key="disk.mountPoint" class="res-item">
              <div class="res-hd">
                <div class="res-dot disk-dot"></div>
                <span>{{ disk.mountPoint }}</span>
                <span class="res-pct" :class="pctCls(disk.usedPercent)">{{ disk.usedPercent.toFixed(1) }}%</span>
              </div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(disk.usedPercent, 'disk')"></div></div>
              <div class="disk-pills">
                <button type="button" class="pill" :title="disk.device" @click="copyText(disk.device)">{{ shortDevice(disk.device) }}</button>
                <span class="pill">{{ disk.fsType }}</span>
                <span class="pill">{{ formatBytePair(disk.used, disk.total) }}</span>
                <span v-if="disk.inodesTotal" class="pill" :class="pctCls(disk.inodesPercent)">inode {{ disk.inodesPercent.toFixed(0) }}%</span>
              </div>
            </div>
            <div v-if="(stats.memory?.swapTotal ?? 0) > 0" class="res-item">
              <div class="res-hd">
                <div class="res-dot mem-dot"></div>
                <span>{{ t('home.swap') }}</span>
                <span class="res-pct" :class="pctCls(stats.memory?.swapPercent)">{{ (stats.memory?.swapPercent ?? 0).toFixed(0) }}%</span>
              </div>
              <div class="bar-bg"><div class="bar-fg" :style="barSty(stats.memory?.swapPercent, 'mem')"></div></div>
              <div class="disk-pills">
                <span class="pill">{{ formatBytePair(stats.memory?.swapUsed, stats.memory?.swapTotal) }}</span>
              </div>
            </div>
          </div>
          <div v-if="sensorGroups.length" class="res-item">
            <div class="res-hd"><div class="res-dot temp-dot"></div><span>{{ t('home.sensorTemp') }}</span></div>
            <div class="temp-chips">
              <div v-for="g in sensorGroups" :key="g.id" class="dash-chip" :title="g.keys.join(', ')">
                <em>{{ g.label }}</em>
                <b :class="g.tone">{{ g.temp.toFixed(0) }}°C</b>
                <small v-if="g.hotspot != null">{{ t('home.sensorHotspot', { temp: g.hotspot.toFixed(0) }) }}</small>
              </div>
            </div>
          </div>
          <div v-else class="sensor-empty">{{ noSensorText }}</div>
        </div>
      </article>

      <article class="deck-card net-card">
        <div class="col-hd"><el-icon><Connection /></el-icon><span>{{ t('home.network') }}</span></div>
        <div class="chip-grid rates">
          <div class="dash-chip rate-chip">
            <em>{{ t('home.download') }}</em>
            <b class="col-down">{{ formatSpeed(headlineNic?.speedDown) }}</b>
            <small>{{ headlineNic?.name || t('home.noNetworkData') }}</small>
          </div>
          <div class="dash-chip rate-chip">
            <em>{{ t('home.upload') }}</em>
            <b class="col-up">{{ formatSpeed(headlineNic?.speedUp) }}</b>
            <small>{{ t('home.totalTraffic') }} {{ formatBytes(stats.network?.bytesRecv) }} / {{ formatBytes(stats.network?.bytesSent) }}</small>
          </div>
        </div>
        <div v-if="publicAddrs.length" class="chip-grid addrs">
          <button
            v-for="item in publicAddrs"
            :key="item.label"
            type="button"
            class="dash-chip addr-chip"
            :title="item.value"
            @click="copyText(item.value)"
          >
            <em>{{ item.label }}</em>
            <b>{{ item.value }}</b>
          </button>
        </div>
        <div v-if="addressIfaces.length" class="chip-grid ifaces">
          <button
            v-for="iface in addressIfaces"
            :key="iface.name"
            type="button"
            class="dash-chip addr-chip"
            @click="copyText((iface.ipv4[0] || '').split('/')[0])"
          >
            <em>{{ iface.name }}</em>
            <b>{{ iface.ipv4[0] }}</b>
          </button>
        </div>
        <div v-if="dnsServers.length" class="chip-grid dns">
          <button type="button" class="dash-chip addr-chip" @click="copyText(dnsServers.join(', '))">
            <em>{{ t('home.dns') }}</em>
            <b>{{ dnsServers.join(' · ') }}</b>
          </button>
        </div>
        <div v-if="trafficNics.length" class="traffic-list">
          <div v-for="nic in trafficNics" :key="nic.name" class="traffic-row">
            <span class="td-nic">{{ nic.name }}</span>
            <span class="col-up">{{ formatSpeed(nic.speedUp) }}</span>
            <span class="col-down">{{ formatSpeed(nic.speedDown) }}</span>
          </div>
        </div>
      </article>

      <article class="deck-card sys-card">
        <div class="col-hd"><el-icon><Monitor /></el-icon><span>{{ t('home.systemInfo') }}</span></div>
        <div class="chip-grid sys">
          <button
            v-for="item in sysInfoItems"
            :key="item.label"
            type="button"
            class="dash-chip sys-chip"
            :class="{ wide: item.wide }"
            :title="item.value"
            @click="item.value && item.value !== '-' && copyText(item.value)"
          >
            <em>{{ item.label }}</em>
            <b>{{ item.value }}</b>
          </button>
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

    <article class="deck-card proc-card">
      <div class="col-hd">
        <el-icon><DataLine /></el-icon>
        <span>{{ t('home.topProcess') }}</span>
        <router-link class="card-more" to="/host/process">{{ t('home.viewAllProcesses') }}</router-link>
      </div>
      <div v-if="stats.topProcess?.length" class="proc-list">
        <div v-for="(row, idx) in stats.topProcess" :key="row.pid" class="proc-row">
          <div class="proc-id">
            <b :title="row.name">{{ row.name }}</b>
            <small>#{{ idx + 1 }} · PID {{ row.pid }}</small>
          </div>
          <div class="proc-meter">
            <div class="proc-meter-hd">
              <em>CPU</em>
              <b :class="pctCls(row.cpuPercent)">{{ row.cpuPercent.toFixed(1) }}%</b>
            </div>
            <div class="bar-bg"><div class="bar-fg" :style="barSty(row.cpuPercent, 'cpu')"></div></div>
          </div>
          <div class="proc-meter">
            <div class="proc-meter-hd">
              <em>{{ t('home.memory') }}</em>
              <b :class="pctCls(row.memPercent)">{{ formatBytes(row.memRss) }}</b>
            </div>
            <div class="bar-bg"><div class="bar-fg" :style="barSty(row.memPercent, 'mem')"></div></div>
          </div>
        </div>
      </div>
      <div v-else class="sensor-empty">{{ t('home.waitingForData') }}</div>
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
import { fallbackSensorName, groupSensors, sensorLabelKey } from './sensors'
import { filterAddressIfaces, filterDNS, filterTrafficNics, isNoiseIface } from './network'
import {
  Monitor, Connection, Compass, DataLine, Box,
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
    { label: t('home.hostname'), value: h.hostname || '-', wide: false },
    { label: t('home.kernel'), value: h.kernelVersion || '-', wide: false },
    { label: t('home.cpuModel'), value: stats.value.cpu?.modelName || '-', wide: true },
    { label: t('home.cpuCores'), value: stats.value.cpu ? `${stats.value.cpu.cores} ${t('home.physical')} / ${stats.value.cpu.logicalCores} ${t('home.logical')}` : '-', wide: false },
    { label: t('home.totalMemory'), value: formatBytes(stats.value.memory?.total), wide: false },
    { label: t('home.tcpCongestion'), value: h.tcpCongestion || '-', wide: false },
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

const showStorage = computed(() =>
  filteredDisks.value.length > 0
  || (stats.value.memory?.swapTotal ?? 0) > 0
  || Boolean(stats.value.sensors?.length)
)

const sensorGroups = computed(() =>
  groupSensors(stats.value.sensors || []).map((group) => ({
    ...group,
    label: group.kind === 'other'
      ? fallbackSensorName(group.keys[0] || group.id)
      : t(sensorLabelKey(group.kind)),
    tone: tempCls({
      key: group.keys[0] || group.id,
      temp: Math.max(group.temp, group.hotspot ?? 0),
    }),
  }))
)

const noSensorText = computed(() => {
  const virt = stats.value.host?.virtualization
  if (virt) return t('home.noSensorVm', { virt })
  return t('home.noSensor')
})

const addressIfaces = computed(() => filterAddressIfaces(stats.value.host?.interfaces))
const dnsServers = computed(() => filterDNS(stats.value.host?.dnsServers))
const trafficNics = computed(() => filterTrafficNics(stats.value.netIO))
const publicAddrs = computed(() => {
  const items: { label: string, value: string }[] = []
  if (stats.value.host?.publicIPv4) items.push({ label: t('home.publicIPv4'), value: stats.value.host.publicIPv4 })
  if (stats.value.host?.publicIPv6) items.push({ label: t('home.publicIPv6'), value: stats.value.host.publicIPv6 })
  return items
})

const headlineNic = computed(() => {
  const nics = (stats.value.netIO || []).filter((n) => n.name !== 'lo' && !isNoiseIface(n.name))
  if (!nics.length) return null
  return nics.reduce((best, nic) =>
    ((nic.speedDown || 0) + (nic.speedUp || 0)) > ((best.speedDown || 0) + (best.speedUp || 0)) ? nic : best
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

const formatBytePair = (used?: number, total?: number) => {
  if (!total) return formatBytes(used)
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(Math.floor(Math.log(total) / Math.log(1024)), units.length - 1)
  const div = 1024 ** i
  return `${((used || 0) / div).toFixed(1)} / ${(total / div).toFixed(1)} ${units[i]}`
}

const shortDevice = (dev?: string) => (dev || '-').replace(/^\/dev\//, '')

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
  padding: 6px 10px;
  color: var(--xp-text-primary);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  background: color-mix(in srgb, var(--xp-accent) 8%, var(--xp-bg-inset));
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
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--xp-accent) 16%, transparent);
}

.gauge-card {
  --gauge-color: var(--gauge-base, var(--xp-accent));
  padding: 12px 12px 16px;
  contain: layout paint;
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--gauge-color) 16%, transparent);

  &.c-warn { --gauge-color: var(--xp-warning); }
  &.c-danger { --gauge-color: var(--xp-danger); }

  &::before {
    content: '';
    position: absolute;
    inset: 0;
    pointer-events: none;
  }

  &:nth-child(1)::before {
    background: radial-gradient(130px 90px at 6% -18%, color-mix(in srgb, var(--gauge-color) 14%, transparent), transparent 72%);
  }
  &:nth-child(2)::before {
    background: radial-gradient(120px 80px at 108% 118%, color-mix(in srgb, var(--gauge-color) 12%, transparent), transparent 70%);
  }
  &:nth-child(3)::before {
    background: radial-gradient(90px 70px at 92% 8%, color-mix(in srgb, var(--gauge-color) 11%, transparent), transparent 70%);
  }
  &:nth-child(4)::before {
    background:
      radial-gradient(100px 70px at 12% 110%, color-mix(in srgb, var(--gauge-color) 10%, transparent), transparent 68%),
      radial-gradient(circle at 78% 24%, color-mix(in srgb, var(--gauge-color) 32%, transparent) 0 0.55px, transparent 0.95px);
    background-size: auto, 132px 120px;
  }
}

.overview-grid {
  display: grid;
  align-items: stretch;
  grid-template-columns: minmax(0, 1.08fr) minmax(0, 0.92fr);
  gap: 16px;

  &.has-storage {
    grid-template-columns: minmax(280px, 0.9fr) minmax(0, 1.1fr) minmax(0, 0.95fr);
  }

  > .deck-card {
    min-height: 0;
    height: 100%;
    display: flex;
    flex-direction: column;
  }
}

.col-hd {
  position: relative;
  z-index: 1;
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
  overflow: hidden;
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--xp-info) 16%, transparent);

  &::after {
    content: '';
    position: absolute;
    inset: 0;
    background-image: radial-gradient(circle at 22% 30%, color-mix(in srgb, var(--xp-info) 28%, transparent) 0 0.55px, transparent 0.9px);
    background-size: 140px 120px;
    opacity: 0.7;
    pointer-events: none;
  }
}

.net-card {
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--xp-info) 16%, transparent);

  &::before {
    content: '';
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(color-mix(in srgb, var(--xp-info) 10%, transparent) 1px, transparent 1px),
      linear-gradient(90deg, color-mix(in srgb, var(--xp-info) 10%, transparent) 1px, transparent 1px);
    background-size: 22px 22px;
    mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.22), transparent 70%);
    pointer-events: none;
  }
}

.sys-card {
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--xp-accent) 16%, transparent);

  &::after {
    content: '';
    position: absolute;
    right: -36px;
    bottom: -42px;
    width: 150px;
    height: 150px;
    border: 14px solid color-mix(in srgb, var(--xp-accent) 12%, transparent);
    border-radius: 50%;
    pointer-events: none;
  }
}

.chip-grid {
  position: relative;
  z-index: 1;
  display: grid;
  gap: 8px;
  margin-bottom: 10px;

  &.rates { grid-template-columns: 1fr 1fr; }
  &.addrs,
  &.ifaces { grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); }
  &.dns { grid-template-columns: 1fr; }
  &.sys { grid-template-columns: 1fr 1fr; flex: 1; }
}

.dash-chip {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  min-width: 0;
  padding: 8px 10px;
  color: inherit;
  text-align: left;
  background: color-mix(in srgb, var(--xp-info) 8%, var(--xp-bg-inset));
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);

  em {
    color: var(--xp-text-muted);
    font-size: 11px;
    font-style: normal;
  }

  b {
    color: var(--xp-text-primary);
    font-size: 13px;
    font-weight: 650;
    letter-spacing: -0.02em;
    font-variant-numeric: tabular-nums;
    overflow-wrap: anywhere;
  }

  small {
    color: var(--xp-text-muted);
    font-size: 10px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
  }
}

button.dash-chip {
  cursor: pointer;
  &:hover {
    border-color: color-mix(in srgb, var(--xp-accent) 40%, var(--xp-border-light));
    background: color-mix(in srgb, var(--xp-accent) 10%, var(--xp-bg-inset));
  }
}

.rate-chip b {
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -0.03em;
}

.addr-chip b {
  font-family: var(--xp-font-mono);
  font-size: 12px;
  font-weight: 600;
}

.sys-chip {
  background: color-mix(in srgb, var(--xp-accent) 8%, var(--xp-bg-inset));
  &.wide { grid-column: 1 / -1; }
  b {
    font-size: 12px;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 100%;
  }
  &.wide b { white-space: normal; }
}

.res-list {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
  min-height: 0;
}
.disk-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding-right: 2px;
}
.res-hd {
  display: flex; align-items: center; gap: 8px; margin-bottom: 6px;
  span:first-of-type { flex: 1; color: var(--xp-text-primary); font-size: 13px; font-weight: 600; }
}
.disk-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}
.pill {
  padding: 2px 8px;
  color: var(--xp-text-secondary);
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  background: color-mix(in srgb, var(--xp-info) 8%, var(--xp-bg-inset));
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
  &.c-warn { color: var(--xp-warning); }
  &.c-danger { color: var(--xp-danger); }
}
button.pill {
  font-family: var(--xp-font-mono);
  cursor: pointer;
  &:hover {
    color: var(--xp-text-primary);
    border-color: color-mix(in srgb, var(--xp-accent) 40%, var(--xp-border-light));
  }
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

.temp-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;

  .dash-chip {
    flex-direction: row;
    align-items: baseline;
    gap: 6px;
    padding: 4px 8px;
    b { font-size: 13px; font-weight: 750; }
    small { margin-left: 2px; }
  }
}
.sensor-empty {
  position: relative;
  z-index: 1;
  color: var(--xp-text-muted);
  font-size: 12px;
}

.traffic-list {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 4px;
}
.traffic-row {
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
  padding: 6px 8px;
  font-family: var(--xp-font-mono);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  background: color-mix(in srgb, var(--xp-bg-inset) 70%, transparent);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
}
.td-nic { color: var(--xp-text-secondary); font-weight: 500; }

.col-up { color: var(--xp-color-up, var(--xp-success)); text-align: right; }
.col-down { color: var(--xp-color-down, var(--xp-accent-secondary)); text-align: right; }

.card-more {
  margin-left: auto;
  color: var(--xp-text-muted);
  font-size: 12px;
  font-weight: 500;
  text-decoration: none;
  &:hover { color: var(--xp-accent); }
}

.proc-list {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.proc-row {
  display: grid;
  grid-template-columns: minmax(120px, 0.9fr) minmax(0, 1.15fr) minmax(0, 1.15fr);
  gap: 12px 16px;
  align-items: center;
  padding: 8px 10px;
  background: color-mix(in srgb, var(--xp-bg-inset) 70%, transparent);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius-sm);
}
.proc-id {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
  b {
    color: var(--xp-text-primary);
    font-size: 13px;
    font-weight: 650;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  small {
    color: var(--xp-text-muted);
    font-size: 11px;
    font-variant-numeric: tabular-nums;
  }
}
.proc-meter {
  min-width: 0;
  .bar-bg { margin-bottom: 0; }
}
.proc-meter-hd {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
  em {
    color: var(--xp-text-muted);
    font-size: 11px;
    font-style: normal;
  }
  b {
    font-size: 12px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
  }
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

    :deep(.gauge-chip) { font-size: 14px; }
    :deep(.gauge-sub) { white-space: normal; }
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
  .chip-grid.rates,
  .chip-grid.sys { grid-template-columns: 1fr 1fr; }
  .rate-chip b { font-size: 16px; }
  .quick-chip { min-height: 32px; padding: 0 10px; font-size: 12px; }
  .proc-row { grid-template-columns: 1fr 1fr; gap: 8px 12px; }
  .proc-id { grid-column: 1 / -1; }
}

@media (max-width: 560px) {
  .gauge-deck { gap: 8px; }
  .gauge-card { padding: 8px; }
  .chip-grid.rates,
  .chip-grid.sys { grid-template-columns: 1fr; }
}
</style>
