<template>
  <div>
    <el-row :gutter="16">
      <!-- 主机名 -->
      <el-col :xs="24" :sm="12">
        <el-card shadow="never" style="margin-bottom: 16px;">
          <template #header>{{ $t('systemSetting.hostname') }}</template>
          <el-form label-width="110px">
            <el-form-item :label="$t('systemSetting.hostnameCurrent')">
              <span style="font-weight: bold;">{{ systemInfo.hostname || '-' }}</span>
            </el-form-item>
            <el-form-item :label="$t('systemSetting.hostnameNew')">
              <div style="display: flex; gap: 8px; width: 100%;">
                <el-input v-model="newHostname" :placeholder="systemInfo.hostname" style="flex: 1;" />
                <el-button type="primary" @click="handleSetHostname" :loading="hostnameSaving" :disabled="!newHostname">
                  {{ $t('commons.save') }}
                </el-button>
              </div>
            </el-form-item>
            <el-form-item>
              <el-text type="info" size="small">{{ $t('systemSetting.hostnameHint') }}</el-text>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 时区 -->
      <el-col :xs="24" :sm="12">
        <el-card shadow="never" style="margin-bottom: 16px;">
          <template #header>{{ $t('systemSetting.timezone') }}</template>
          <el-form label-width="110px">
            <el-form-item :label="$t('systemSetting.timezoneCurrent')">
              <span style="font-weight: bold;">{{ systemInfo.timezone || '-' }}</span>
            </el-form-item>
            <el-form-item :label="$t('systemSetting.timezoneSelect')">
              <div style="display: flex; gap: 8px; width: 100%;">
                <el-select v-model="newTimezone" filterable :placeholder="systemInfo.timezone" style="flex: 1;">
                  <el-option v-for="tz in timezones" :key="tz" :label="tz" :value="tz" />
                </el-select>
                <el-button type="primary" @click="handleSetTimezone" :loading="timezoneSaving" :disabled="!newTimezone">
                  {{ $t('commons.save') }}
                </el-button>
              </div>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- DNS -->
    <el-card shadow="never" style="margin-bottom: 16px;">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between;">
          <span>{{ $t('systemSetting.dns') }}</span>
          <div>
            <el-dropdown trigger="click" @command="addPresetDNS" style="margin-right: 8px;">
              <el-button size="small">{{ $t('systemSetting.dnsPresets') }}<el-icon class="el-icon--right"><ArrowDown /></el-icon></el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item v-for="p in dnsPresets" :key="p.value" :command="p.value">
                    {{ p.label }} ({{ p.value }})
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
            <el-button size="small" @click="addDNS">{{ $t('systemSetting.dnsAdd') }}</el-button>
          </div>
        </div>
      </template>
      <div style="margin-bottom: 8px;">
        <el-text type="info" size="small">{{ $t('systemSetting.dnsHint') }}</el-text>
      </div>
      <div v-for="(dns, idx) in dnsServers" :key="idx" style="display: flex; gap: 8px; margin-bottom: 8px;">
        <el-input v-model="dnsServers[idx]" placeholder="8.8.8.8" style="flex: 1;" />
        <el-button type="danger" link @click="dnsServers.splice(idx, 1)">
          <el-icon><Delete /></el-icon>
        </el-button>
      </div>
      <el-button type="primary" @click="handleSaveDNS" :loading="dnsSaving" style="margin-top: 8px;">
        {{ $t('commons.save') }}
      </el-button>
    </el-card>

    <!-- Swap -->
    <el-card shadow="never">
      <template #header>{{ $t('systemSetting.swap') }}</template>
      <el-descriptions :column="2" border v-if="swapInfo.file">
        <el-descriptions-item :label="$t('systemSetting.swapStatus')">
          <el-tag :type="swapInfo.enabled ? 'success' : 'info'">
            {{ swapInfo.enabled ? $t('systemSetting.swapEnabled') : $t('systemSetting.swapDisabled') }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('systemSetting.swapFile')">{{ swapInfo.file }}</el-descriptions-item>
        <el-descriptions-item :label="$t('systemSetting.swapTotal')">{{ formatSize(swapInfo.total) }}</el-descriptions-item>
        <el-descriptions-item :label="$t('systemSetting.swapUsed')">{{ formatSize(swapInfo.used) }}</el-descriptions-item>
      </el-descriptions>
      <el-empty v-else :description="$t('systemSetting.noSwapFile')" :image-size="60" />

      <div style="margin-top: 16px; display: flex; gap: 8px;">
        <template v-if="swapInfo.file">
          <el-button v-if="swapInfo.enabled" type="warning" @click="handleSwapOp('off')" :loading="swapOperating">
            {{ $t('systemSetting.swapOff') }}
          </el-button>
          <el-button v-else type="success" @click="handleSwapOp('on')" :loading="swapOperating">
            {{ $t('systemSetting.swapOn') }}
          </el-button>
          <el-button type="danger" @click="handleDeleteSwap" :loading="swapOperating">
            {{ $t('systemSetting.swapDelete') }}
          </el-button>
        </template>
        <el-button type="primary" @click="showCreateSwap = true">
          {{ $t('systemSetting.swapCreate') }}
        </el-button>
      </div>
    </el-card>

    <!-- Swap 创建对话框 -->
    <el-dialog v-model="showCreateSwap" :title="$t('systemSetting.swapCreate')" width="420px" destroy-on-close>
      <el-form label-width="110px">
        <el-form-item :label="$t('systemSetting.swapSizeMB')">
          <el-input-number v-model="swapSizeMB" :min="64" :max="65536" :step="256" style="width: 100%;" />
        </el-form-item>
        <el-form-item>
          <el-text type="info" size="small">{{ $t('systemSetting.swapSizeHint') }}</el-text>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateSwap = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateSwap" :loading="swapCreating">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, ArrowDown } from '@element-plus/icons-vue'
import {
  getSystemInfo,
  setHostname,
  setTimezone,
  listTimezones,
  getDNS,
  setDNS,
  getSwapInfo,
  createSwap,
  deleteSwap,
  swapOperate,
} from '@/api/modules/host'

const { t } = useI18n()

const systemInfo = reactive({ hostname: '', timezone: '' })
const newHostname = ref('')
const newTimezone = ref('')
const timezones = ref<string[]>([])
const dnsServers = ref<string[]>([])
const swapInfo = reactive({ total: 0, used: 0, file: '', enabled: false })

const hostnameSaving = ref(false)
const timezoneSaving = ref(false)
const dnsSaving = ref(false)
const swapOperating = ref(false)
const swapCreating = ref(false)
const showCreateSwap = ref(false)
const swapSizeMB = ref(1024)

const dnsPresets = [
  { label: 'Google', value: '8.8.8.8' },
  { label: 'Google 备', value: '8.8.4.4' },
  { label: 'Cloudflare', value: '1.1.1.1' },
  { label: '阿里 DNS', value: '223.5.5.5' },
  { label: '腾讯 DNS', value: '119.29.29.29' },
  { label: '114 DNS', value: '114.114.114.114' },
]

const formatSize = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let val = bytes
  while (val >= 1024 && i < units.length - 1) {
    val /= 1024
    i++
  }
  return val.toFixed(1) + ' ' + units[i]
}

const loadAll = async () => {
  try {
    const res = await getSystemInfo()
    const data = res.data
    systemInfo.hostname = data.hostname
    systemInfo.timezone = data.timezone
    dnsServers.value = data.dns || []
    Object.assign(swapInfo, data.swap || {})
  } catch {}
}

const loadTimezones = async () => {
  try {
    const res = await listTimezones()
    timezones.value = res.data || []
  } catch {}
}

const handleSetHostname = async () => {
  hostnameSaving.value = true
  try {
    await setHostname(newHostname.value)
    ElMessage.success(t('commons.success'))
    systemInfo.hostname = newHostname.value
    newHostname.value = ''
  } finally {
    hostnameSaving.value = false
  }
}

const handleSetTimezone = async () => {
  timezoneSaving.value = true
  try {
    await setTimezone(newTimezone.value)
    ElMessage.success(t('commons.success'))
    systemInfo.timezone = newTimezone.value
    newTimezone.value = ''
  } finally {
    timezoneSaving.value = false
  }
}

const addDNS = () => {
  dnsServers.value.push('')
}

const addPresetDNS = (val: string) => {
  if (!dnsServers.value.includes(val)) {
    dnsServers.value.push(val)
  }
}

const handleSaveDNS = async () => {
  dnsSaving.value = true
  try {
    await setDNS(dnsServers.value.filter(s => s.trim()))
    ElMessage.success(t('commons.saveSuccess'))
  } finally {
    dnsSaving.value = false
  }
}

const loadSwap = async () => {
  try {
    const res = await getSwapInfo()
    Object.assign(swapInfo, res.data || {})
  } catch {}
}

const handleSwapOp = async (op: string) => {
  swapOperating.value = true
  try {
    await swapOperate(op)
    ElMessage.success(t('commons.success'))
    await loadSwap()
  } finally {
    swapOperating.value = false
  }
}

const handleCreateSwap = async () => {
  swapCreating.value = true
  try {
    await createSwap(swapSizeMB.value)
    ElMessage.success(t('commons.success'))
    showCreateSwap.value = false
    await loadSwap()
  } finally {
    swapCreating.value = false
  }
}

const handleDeleteSwap = async () => {
  await ElMessageBox.confirm(t('systemSetting.swapDeleteConfirm'), t('commons.warning'), {
    confirmButtonText: t('commons.confirm'),
    cancelButtonText: t('commons.cancel'),
    type: 'warning',
  })
  swapOperating.value = true
  try {
    await deleteSwap()
    ElMessage.success(t('commons.success'))
    await loadSwap()
  } finally {
    swapOperating.value = false
  }
}

onMounted(() => {
  loadAll()
  loadTimezones()
})
</script>
