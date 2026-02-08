<template>
  <div class="disk-page">
    <div class="page-header">
      <h3>{{ $t('disk.title') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadDisk" :loading="loading">{{ $t('commons.refresh') }}</el-button>
    </div>

    <el-row :gutter="16">
      <el-col :span="24" v-for="(part, idx) in partitions" :key="idx">
        <el-card shadow="never" class="disk-card">
          <div class="disk-info-row">
            <div class="disk-basic">
              <div class="disk-device">
                <el-icon :size="20"><Coin /></el-icon>
                <span class="device-name">{{ part.device }}</span>
                <el-tag size="small" type="info">{{ part.fsType }}</el-tag>
              </div>
              <div class="disk-mount">{{ $t('disk.mountPoint') }}: {{ part.mountPoint }}</div>
            </div>
            <div class="disk-usage-section">
              <div class="disk-progress">
                <el-progress :percentage="Math.round(part.usedPercent)" :color="progressColor" :stroke-width="18" :text-inside="true" />
              </div>
              <div class="disk-sizes">
                <span>{{ $t('disk.used') }}: {{ formatBytes(part.used) }}</span>
                <span>{{ $t('disk.free') }}: {{ formatBytes(part.free) }}</span>
                <span>{{ $t('disk.total') }}: {{ formatBytes(part.total) }}</span>
              </div>
            </div>
            <div class="disk-inodes" v-if="part.inodesTotal > 0">
              <div class="inodes-label">{{ $t('disk.inodes') }}</div>
              <div class="inodes-detail">
                {{ $t('disk.inodesUsed') }}: {{ formatNumber(part.inodesUsed) }} / {{ formatNumber(part.inodesTotal) }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-empty v-if="!loading && partitions.length === 0" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh, Coin } from '@element-plus/icons-vue'
import { getDiskInfo } from '@/api/modules/disk'

const loading = ref(false)
const partitions = ref<any[]>([])

const loadDisk = async () => {
  loading.value = true
  try {
    const res = await getDiskInfo()
    partitions.value = res.data || []
  } catch { partitions.value = [] }
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

const formatNumber = (n?: number) => {
  if (!n) return '0'
  return n.toLocaleString()
}

onMounted(() => loadDisk())
</script>

<style lang="scss" scoped>
.disk-page { height: 100%; }

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
}

.disk-card {
  margin-bottom: 12px;
}

.disk-info-row {
  display: flex;
  align-items: center;
  gap: 32px;
}

.disk-basic {
  min-width: 200px;

  .disk-device {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;

    .device-name {
      font-weight: 600;
      font-size: 14px;
      color: var(--xp-text-primary);
    }
  }

  .disk-mount {
    font-size: 12px;
    color: var(--xp-text-muted);
    padding-left: 28px;
  }
}

.disk-usage-section {
  flex: 1;

  .disk-progress {
    margin-bottom: 6px;
  }

  .disk-sizes {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: var(--xp-text-secondary);
  }
}

.disk-inodes {
  min-width: 160px;
  text-align: right;

  .inodes-label {
    font-size: 12px;
    color: var(--xp-text-muted);
    margin-bottom: 2px;
  }

  .inodes-detail {
    font-size: 12px;
    color: var(--xp-text-secondary);
  }
}
</style>
