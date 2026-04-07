<template>
  <div class="disk-page">
    <div class="page-header">
      <h3>{{ $t('disk.title') }}</h3>
      <el-button size="small" :icon="Refresh" @click="loadAll" :loading="loading">{{ $t('commons.refresh') }}</el-button>
    </div>

    <!-- 本地磁盘 -->
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

    <!-- 远程挂载 -->
    <div class="page-header" style="margin-top: 20px;">
      <h3>{{ $t('disk.remoteMount') }}</h3>
      <el-button size="small" type="primary" :icon="Plus" @click="showMountDialog = true">
        {{ $t('disk.addMount') }}
      </el-button>
    </div>

    <el-table :data="remoteMounts" v-loading="remoteLoading" v-if="remoteMounts.length > 0">
      <el-table-column prop="device" :label="$t('disk.remoteSource')" min-width="200" show-overflow-tooltip />
      <el-table-column prop="mountPoint" :label="$t('disk.mountPoint')" min-width="140" />
      <el-table-column prop="fsType" :label="$t('disk.fsType')" width="80" />
      <el-table-column :label="$t('disk.usage')" width="200">
        <template #default="{ row }">
          <template v-if="row.total > 0">
            <el-progress :percentage="Math.round(row.percent)" :color="progressColor" :stroke-width="12" :text-inside="true" />
            <span class="remote-size">{{ formatBytes(row.used) }} / {{ formatBytes(row.total) }}</span>
          </template>
          <span v-else class="text-muted">-</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('disk.persist')" width="90" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.inFstab" type="success" size="small">{{ $t('disk.fstabYes') }}</el-tag>
          <el-tag v-else type="info" size="small">{{ $t('disk.fstabNo') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="options" :label="$t('disk.mountOptions')" min-width="200" show-overflow-tooltip />
      <el-table-column :label="$t('commons.actions')" width="100" align="center">
        <template #default="{ row }">
          <el-popconfirm :title="$t('disk.unmountConfirm')" @confirm="handleUnmount(row)">
            <template #reference>
              <el-button type="danger" text size="small">{{ $t('disk.unmount') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-empty v-else-if="!remoteLoading" :description="$t('disk.noRemoteMount')" :image-size="60" />

    <!-- 挂载对话框 -->
    <el-dialog v-model="showMountDialog" :title="$t('disk.addMount')" width="560px" :close-on-click-modal="false" @close="resetForm">
      <el-form ref="mountFormRef" :model="mountForm" :rules="mountRules" label-width="100px">
        <el-form-item :label="$t('disk.protocol')" prop="protocol">
          <el-select v-model="mountForm.protocol" style="width: 100%" @change="onProtocolChange">
            <el-option label="NFS" value="nfs" />
            <el-option label="SMB / CIFS" value="cifs" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('disk.server')" prop="server">
          <el-input v-model="mountForm.server" placeholder="192.168.1.100" />
        </el-form-item>
        <el-form-item :label="$t('disk.sharePath')" prop="sharePath">
          <el-input v-model="mountForm.sharePath" :placeholder="mountForm.protocol === 'nfs' ? '/data/share' : 'share_name'" />
        </el-form-item>
        <el-form-item :label="$t('disk.mountPoint')" prop="mountPoint">
          <el-input v-model="mountForm.mountPoint" placeholder="/mnt/remote" />
        </el-form-item>
        <template v-if="mountForm.protocol === 'cifs'">
          <el-form-item :label="$t('disk.username')">
            <el-input v-model="mountForm.username" :placeholder="$t('disk.usernameHint')" />
          </el-form-item>
          <el-form-item :label="$t('disk.password')">
            <el-input v-model="mountForm.password" type="password" show-password :placeholder="$t('disk.passwordHint')" />
          </el-form-item>
        </template>

        <el-form-item :label="$t('disk.networkPreset')">
          <el-radio-group v-model="mountForm.preset" @change="onPresetChange">
            <el-radio-button value="default">
              <el-tooltip :content="$t('disk.presetDefaultTip')" placement="top">
                <span>{{ $t('disk.presetDefault') }}</span>
              </el-tooltip>
            </el-radio-button>
            <el-radio-button value="unstable">
              <el-tooltip :content="$t('disk.presetUnstableTip')" placement="top">
                <span>{{ $t('disk.presetUnstable') }}</span>
              </el-tooltip>
            </el-radio-button>
            <el-radio-button value="lan">
              <el-tooltip :content="$t('disk.presetLanTip')" placement="top">
                <span>{{ $t('disk.presetLan') }}</span>
              </el-tooltip>
            </el-radio-button>
            <el-radio-button value="custom">
              <span>{{ $t('disk.presetCustom') }}</span>
            </el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-form-item v-if="mountForm.preset !== 'custom'" :label="$t('disk.previewOptions')">
          <el-input :model-value="previewOptions" disabled />
        </el-form-item>

        <el-form-item v-if="mountForm.preset === 'custom'" :label="$t('disk.mountOptions')">
          <el-input v-model="mountForm.options" :placeholder="$t('disk.mountOptionsHint')" />
        </el-form-item>

        <el-form-item :label="$t('disk.persist')">
          <el-switch v-model="mountForm.persist" />
          <span class="form-hint">{{ $t('disk.persistHint') }}</span>
        </el-form-item>

        <el-alert
          v-if="mountForm.preset === 'unstable'"
          :title="$t('disk.unstableNetworkTip')"
          type="info"
          :closable="false"
          show-icon
          style="margin-bottom: 8px"
        />
      </el-form>
      <template #footer>
        <el-button @click="showMountDialog = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleMount" :loading="mounting">{{ $t('disk.mount') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh, Coin, Plus } from '@element-plus/icons-vue'
import { getDiskInfo, listRemoteMounts, mountRemote, unmountRemote } from '@/api/modules/disk'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { FormInstance, FormRules } from 'element-plus'
import type { PartitionInfo, RemoteMountInfo } from '@/api/interface'

const { t } = useI18n()
const loading = ref(false)
const remoteLoading = ref(false)
const partitions = ref<PartitionInfo[]>([])
const remoteMounts = ref<RemoteMountInfo[]>([])
const showMountDialog = ref(false)
const mounting = ref(false)
const mountFormRef = ref<FormInstance>()

const nfsPresets: Record<string, string> = {
  default: 'rw,soft,timeo=30,retrans=3',
  unstable: 'rw,soft,timeo=10,retrans=2,actimeo=60,noatime',
  lan: 'rw,hard,timeo=600,retrans=5,rsize=1048576,wsize=1048576',
}

const cifsPresets: Record<string, string> = {
  default: 'rw,soft,echo_interval=10,actimeo=30',
  unstable: 'rw,soft,echo_interval=5,actimeo=30,cache=loose,nobrl,noserverino',
  lan: 'rw,hard,cache=strict,rsize=4194304,wsize=4194304',
}

const mountForm = reactive({
  protocol: 'nfs',
  server: '',
  sharePath: '',
  mountPoint: '',
  username: '',
  password: '',
  options: '',
  preset: 'default',
  persist: false,
})

const previewOptions = computed(() => {
  const presets = mountForm.protocol === 'nfs' ? nfsPresets : cifsPresets
  return presets[mountForm.preset] || presets['default']
})

const onProtocolChange = () => {
  mountForm.preset = 'default'
  mountForm.options = ''
}

const onPresetChange = (val: string) => {
  if (val !== 'custom') {
    mountForm.options = ''
  }
}

const resetForm = () => {
  Object.assign(mountForm, {
    protocol: 'nfs', server: '', sharePath: '', mountPoint: '',
    username: '', password: '', options: '', preset: 'default', persist: false,
  })
}

const mountRules = reactive<FormRules>({
  protocol: [{ required: true, trigger: 'change' }],
  server: [{ required: true, message: t('disk.serverRequired'), trigger: 'blur' }],
  sharePath: [{ required: true, message: t('disk.sharePathRequired'), trigger: 'blur' }],
  mountPoint: [{ required: true, message: t('disk.mountPointRequired'), trigger: 'blur' }],
})

const loadDisk = async () => {
  loading.value = true
  try {
    const res = await getDiskInfo()
    partitions.value = res.data || []
  } catch { partitions.value = [] }
  finally { loading.value = false }
}

const loadRemote = async () => {
  remoteLoading.value = true
  try {
    const res = await listRemoteMounts()
    remoteMounts.value = res.data || []
  } catch { remoteMounts.value = [] }
  finally { remoteLoading.value = false }
}

const loadAll = () => {
  loadDisk()
  loadRemote()
}

const handleMount = async () => {
  await mountFormRef.value?.validate()
  mounting.value = true
  try {
    await mountRemote({
      protocol: mountForm.protocol,
      server: mountForm.server,
      sharePath: mountForm.sharePath,
      mountPoint: mountForm.mountPoint,
      username: mountForm.username || undefined,
      password: mountForm.password || undefined,
      options: mountForm.preset === 'custom' ? mountForm.options : undefined,
      preset: mountForm.preset !== 'custom' ? mountForm.preset : undefined,
      persist: mountForm.persist,
    })
    ElMessage.success(t('commons.success'))
    showMountDialog.value = false
    resetForm()
    loadAll()
  } catch { /* handled by interceptor */ }
  finally { mounting.value = false }
}

const handleUnmount = async (row: RemoteMountInfo) => {
  try {
    await unmountRemote({ mountPoint: row.mountPoint, removeFstab: row.inFstab })
    ElMessage.success(t('commons.success'))
    loadAll()
  } catch { /* handled */ }
}

const progressColor = (percentage: number) => {
  if (percentage < 50) return getComputedStyle(document.documentElement).getPropertyValue('--xp-accent').trim() || '#22d3ee'
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

onMounted(() => loadAll())
</script>

<style lang="scss" scoped>
.disk-page { height: 100%; }

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

.remote-size {
  font-size: 11px;
  color: var(--xp-text-muted);
}

.text-muted {
  color: var(--xp-text-muted);
}

.form-hint {
  margin-left: 8px;
  font-size: 12px;
  color: var(--xp-text-muted);
}
</style>
