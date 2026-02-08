<template>
  <div class="monitor-page">
    <div class="page-header">
      <h3>{{ $t('monitor.title') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadStats" :loading="loading">
        {{ $t('commons.refresh') }}
      </el-button>
    </div>

    <!-- 系统信息 -->
    <el-card shadow="never" class="host-info-card" v-if="stats.host">
      <div class="host-info-grid">
        <div class="info-item">
          <span class="info-label">主机名</span>
          <span class="info-value">{{ stats.host.hostname || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">操作系统</span>
          <span class="info-value">{{ stats.host.platform }} {{ stats.host.platformVersion }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">内核版本</span>
          <span class="info-value">{{ stats.host.kernelVersion || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">系统架构</span>
          <span class="info-value">{{ stats.host.kernelArch || '-' }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">运行时间</span>
          <span class="info-value uptime-highlight">{{ formatUptime(stats.uptime) }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">CPU</span>
          <span class="info-value">{{ stats.cpu?.modelName }} ({{ stats.cpu?.logicalCores }}核)</span>
        </div>
      </div>
    </el-card>

    <!-- 概览卡片 -->
    <el-row :gutter="16" class="overview-row">
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-title">CPU</div>
          <el-progress type="dashboard" :percentage="Math.round(stats.cpu?.usagePercent || 0)" :color="progressColor" :width="100" />
          <div class="stat-detail">{{ stats.cpu?.cores }}物理核 / {{ stats.cpu?.logicalCores }}逻辑核</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-title">{{ $t('monitor.memory') }}</div>
          <el-progress type="dashboard" :percentage="Math.round(stats.memory?.usedPercent || 0)" :color="progressColor" :width="100" />
          <div class="stat-detail">{{ formatBytes(stats.memory?.used) }} / {{ formatBytes(stats.memory?.total) }}</div>
          <div class="stat-sub" v-if="stats.memory?.swapTotal > 0">
            Swap: {{ formatBytes(stats.memory?.swapUsed) }} / {{ formatBytes(stats.memory?.swapTotal) }}
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-title">{{ $t('monitor.load') }}</div>
          <div class="load-values">
            <div class="load-item">
              <span class="load-num">{{ stats.load?.load1?.toFixed(2) || '-' }}</span>
              <span class="load-label">1 min</span>
            </div>
            <div class="load-item">
              <span class="load-num">{{ stats.load?.load5?.toFixed(2) || '-' }}</span>
              <span class="load-label">5 min</span>
            </div>
            <div class="load-item">
              <span class="load-num">{{ stats.load?.load15?.toFixed(2) || '-' }}</span>
              <span class="load-label">15 min</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never" class="stat-card">
          <div class="stat-title">{{ $t('monitor.network') }}</div>
          <div class="net-speed" v-if="stats.netIO?.length">
            <div v-for="nic in stats.netIO" :key="nic.name" class="nic-item">
              <span class="nic-name">{{ nic.name }}</span>
              <span class="nic-speed">
                <span class="up">↑ {{ formatSpeed(nic.speedUp) }}</span>
                <span class="down">↓ {{ formatSpeed(nic.speedDown) }}</span>
              </span>
            </div>
          </div>
          <div class="stat-detail">
            累计 ↑ {{ formatBytes(stats.network?.bytesSent) }}  ↓ {{ formatBytes(stats.network?.bytesRecv) }}
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Top 进程 + 磁盘 并排 -->
    <el-row :gutter="16">
      <el-col :span="10">
        <el-card shadow="never" class="section-card">
          <template #header>
            <span>Top 进程 (CPU)</span>
          </template>
          <el-table :data="stats.topProcess || []" size="small">
            <el-table-column prop="pid" label="PID" width="70" />
            <el-table-column prop="name" label="进程名" min-width="120" show-overflow-tooltip />
            <el-table-column label="CPU %" width="90" align="right">
              <template #default="{ row }">
                <span :class="row.cpuPercent > 50 ? 'text-danger' : ''">{{ row.cpuPercent.toFixed(1) }}%</span>
              </template>
            </el-table-column>
            <el-table-column label="内存" width="90" align="right">
              <template #default="{ row }">
                {{ formatBytes(row.memRss) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="14">
        <el-card shadow="never" class="section-card">
          <template #header>
            <span>{{ $t('monitor.disk') }}{{ $t('monitor.usage') }}</span>
          </template>
          <el-table :data="stats.disks || []" size="small">
            <el-table-column prop="mountPoint" :label="$t('disk.mountPoint')" min-width="100" show-overflow-tooltip />
            <el-table-column prop="device" :label="$t('disk.device')" min-width="100" show-overflow-tooltip />
            <el-table-column prop="fsType" :label="$t('disk.fsType')" width="70" />
            <el-table-column :label="$t('monitor.usage')" min-width="160">
              <template #default="{ row }">
                <el-progress :percentage="Math.round(row.usedPercent)" :color="progressColor" :stroke-width="14" :text-inside="true" />
              </template>
            </el-table-column>
            <el-table-column :label="$t('monitor.used')" width="130" align="right">
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
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getSystemStats } from '@/api/modules/monitor'

const loading = ref(false)
const stats = ref<any>({})
let timer: ReturnType<typeof setInterval> | null = null

const loadStats = async () => {
  loading.value = true
  try {
    const res = await getSystemStats()
    stats.value = res.data || {}
  } catch { /* handled by interceptor */ }
  finally { loading.value = false }
}

const progressColor = (percentage: number) => {
  if (percentage < 50) return '#22d3ee'
  if (percentage < 80) return '#f59e0b'
  return '#ef4444'
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
  if (d > 0) parts.push(`${d} 天`)
  if (h > 0) parts.push(`${h} 时`)
  parts.push(`${m} 分`)
  return parts.join(' ')
}

onMounted(() => {
  loadStats()
  timer = setInterval(loadStats, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style lang="scss" scoped>
.monitor-page {
  height: 100%;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 {
    margin: 0;
    font-size: 16px;
    color: var(--xp-text-primary);
  }
}

/* 系统信息卡片 */
.host-info-card {
  margin-bottom: 16px;

  .host-info-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px 24px;
  }

  .info-item {
    display: flex;
    align-items: baseline;
    gap: 8px;
  }

  .info-label {
    font-size: 12px;
    color: var(--xp-text-muted);
    white-space: nowrap;
    min-width: 60px;
  }

  .info-value {
    font-size: 13px;
    color: var(--xp-text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .uptime-highlight {
    color: #22d3ee;
    font-weight: 600;
  }
}

.overview-row {
  margin-bottom: 16px;
}

.stat-card {
  text-align: center;
  min-height: 200px;

  .stat-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--xp-text-secondary);
    margin-bottom: 12px;
  }

  .stat-detail {
    font-size: 12px;
    color: var(--xp-text-secondary);
    margin-top: 8px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .stat-sub {
    font-size: 11px;
    color: var(--xp-text-muted);
    margin-top: 4px;
  }
}

.load-values {
  display: flex;
  justify-content: center;
  gap: 24px;
  padding: 20px 0;

  .load-item {
    text-align: center;
  }

  .load-num {
    display: block;
    font-size: 24px;
    font-weight: 600;
    color: var(--xp-text-primary);
  }

  .load-label {
    font-size: 11px;
    color: var(--xp-text-muted);
  }
}

/* 网络速率 */
.net-speed {
  padding: 8px 0;

  .nic-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 4px 12px;
    font-size: 12px;
  }

  .nic-name {
    color: var(--xp-text-secondary);
    font-weight: 500;
  }

  .nic-speed {
    display: flex;
    gap: 12px;

    .up {
      color: #22d3ee;
    }

    .down {
      color: #a78bfa;
    }
  }
}

.section-card {
  margin-bottom: 16px;
}

.text-danger {
  color: #ef4444;
  font-weight: 600;
}
</style>
