<template>
  <div class="nic-page xp-page-shell">
    <div class="app-toolbar">
      <p class="page-sub">{{ t('nic.subtitle') }}</p>
      <span class="toolbar-spacer" />
      <el-button size="small" :icon="Refresh" :loading="loading" @click="loadNics">{{ t('commons.refresh') }}</el-button>
    </div>

    <el-table :data="nics" v-loading="loading" size="small" stripe :empty-text="t('nic.empty')">
      <el-table-column :label="t('nic.name')" min-width="140">
        <template #default="{ row }">
          <div class="nic-name">
            <span class="status-dot" :class="row.connected ? 'online' : 'offline'" />
            <b>{{ row.name }}</b>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="t('nic.kind')" width="110">
        <template #default="{ row }">
          <el-tag size="small" type="info">{{ kindLabel(row.kind) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('nic.link')" width="110">
        <template #default="{ row }">
          <el-tag size="small" :type="row.connected ? 'success' : 'info'">
            {{ row.connected ? t('nic.connected') : t('nic.disconnected') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('nic.speed')" min-width="150">
        <template #default="{ row }">
          <span v-if="formatLinkSpeed(row.speedMbps)">
            {{ formatLinkSpeed(row.speedMbps) }}
            <span v-if="duplexLabel(row.duplex)" class="duplex">· {{ duplexLabel(row.duplex) }}</span>
          </span>
          <span v-else class="text-muted">{{ speedFallback(row.speedState) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('nic.ipv4')" min-width="160">
        <template #default="{ row }">
          <button
            v-if="row.ipv4?.[0]"
            type="button"
            class="copy-chip"
            :title="row.ipv4.join(', ')"
            @click="copyText(row.ipv4[0].split('/')[0])"
          >
            {{ row.ipv4[0] }}
          </button>
          <span v-else class="text-muted">-</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('nic.mac')" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">
          <button v-if="row.mac" type="button" class="copy-chip" @click="copyText(row.mac)">{{ row.mac }}</button>
          <span v-else class="text-muted">-</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { listNics } from '@/api/modules/monitor'
import type { NetInterface } from '@/api/interface'
import { filterInventoryNics, formatLinkSpeed } from '@/views/home/network'

const { t } = useI18n()
const loading = ref(false)
const nics = ref<NetInterface[]>([])
let timer: ReturnType<typeof setInterval> | null = null

const kindLabel = (kind?: string) => {
  const keys: Record<string, string> = {
    ethernet: 'nic.kindEthernet',
    wifi: 'nic.kindWifi',
    bridge: 'nic.kindBridge',
    bond: 'nic.kindBond',
    vlan: 'nic.kindVlan',
    virtual: 'nic.kindVirtual',
  }
  return kind && keys[kind] ? t(keys[kind]) : kind || '-'
}

const speedFallback = (state?: string) => {
  if (state === 'unreported') return t('nic.unreported')
  return t('nic.unnegotiated')
}

const duplexLabel = (duplex?: string) => {
  if (duplex === 'full') return t('nic.duplexFull')
  if (duplex === 'half') return t('nic.duplexHalf')
  return ''
}

const loadNics = async () => {
  loading.value = nics.value.length === 0
  try {
    const res = await listNics()
    nics.value = filterInventoryNics(res.data || [])
  } catch {
    /* interceptor */
  } finally {
    loading.value = false
  }
}

const copyText = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('commons.copySuccess'))
  } catch {
    ElMessage.error(t('commons.copyFailed'))
  }
}

onMounted(() => {
  loadNics()
  timer = setInterval(loadNics, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style lang="scss" scoped>
.nic-page {
  height: 100%;
}

.page-sub {
  margin: 0;
  color: var(--xp-text-muted);
  font-size: 13px;
}

.nic-name {
  display: inline-flex;
  align-items: center;
  gap: 8px;

  b {
    font-weight: 600;
  }
}

.duplex {
  color: var(--xp-text-muted);
}

.copy-chip {
  padding: 0;
  color: var(--xp-text-primary);
  background: none;
  border: 0;
  cursor: pointer;
  font: inherit;

  &:hover {
    color: var(--xp-accent);
  }
}

.text-muted {
  color: var(--xp-text-muted);
}
</style>
