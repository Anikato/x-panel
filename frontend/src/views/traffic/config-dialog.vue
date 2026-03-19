<template>
  <el-dialog
    v-model="open"
    :title="isEdit ? $t('traffic.editConfig') : $t('traffic.addConfig')"
    width="520px"
    :close-on-click-modal="false"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
      <el-form-item :label="$t('traffic.interface')" prop="interfaceName">
        <el-select
          v-model="form.interfaceName"
          :placeholder="$t('traffic.selectInterface')"
          :disabled="isEdit"
          style="width: 100%"
        >
          <el-option
            v-for="iface in interfaces"
            :key="iface.name"
            :label="formatIfaceLabel(iface)"
            :value="iface.name"
          />
        </el-select>
      </el-form-item>

      <el-form-item :label="$t('traffic.monthlyLimit')" prop="monthlyLimit">
        <div style="display: flex; gap: 8px; width: 100%">
          <el-input-number
            v-model="limitValue"
            :min="0"
            :precision="1"
            :step="1"
            controls-position="right"
            style="flex: 1"
          />
          <el-select v-model="limitUnit" style="width: 100px">
            <el-option label="GB" value="GB" />
            <el-option label="TB" value="TB" />
          </el-select>
        </div>
        <div class="form-hint">{{ $t('traffic.monthlyLimitHint') }}</div>
      </el-form-item>

      <el-form-item :label="$t('traffic.resetDay')" prop="resetDay">
        <el-input-number
          v-model="form.resetDay"
          :min="1"
          :max="28"
          controls-position="right"
          style="width: 100%"
        />
        <div class="form-hint">{{ $t('traffic.resetDayHint') }}</div>
      </el-form-item>

      <el-form-item :label="$t('traffic.enabled')">
        <el-switch v-model="form.enabled" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="open = false">{{ $t('commons.cancel') }}</el-button>
      <el-button type="primary" @click="onSubmit" :loading="submitting">
        {{ $t('commons.confirm') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { trafficApi } from '@/api/modules/traffic'
import type { InterfaceInfo, TrafficConfig } from '@/api/modules/traffic'

const emit = defineEmits<{ refresh: [] }>()
const { t } = useI18n()

const open = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formRef = ref<FormInstance>()
const interfaces = ref<InterfaceInfo[]>([])

const form = reactive({
  interfaceName: '',
  monthlyLimit: 0,
  resetDay: 1,
  enabled: true,
})

const limitValue = ref(0)
const limitUnit = ref<'GB' | 'TB'>('GB')

const rules = reactive<FormRules>({
  interfaceName: [{ required: true, message: t('traffic.interfaceRequired'), trigger: 'change' }],
  resetDay: [{ required: true, message: t('traffic.resetDayRequired'), trigger: 'change' }],
})

const formatIfaceLabel = (iface: InterfaceInfo) => {
  const ips = iface.ipv4?.join(', ') || ''
  return ips ? `${iface.name} (${ips})` : iface.name
}

const acceptParams = async (config?: TrafficConfig) => {
  isEdit.value = !!config
  try {
    const res: any = await trafficApi.listInterfaces()
    interfaces.value = res.data || []
  } catch { /* handled */ }

  if (config) {
    form.interfaceName = config.interfaceName
    form.resetDay = config.resetDay
    form.enabled = config.enabled
    if (config.monthlyLimit >= 1024 * 1024 * 1024 * 1024) {
      limitValue.value = +(config.monthlyLimit / (1024 * 1024 * 1024 * 1024)).toFixed(1)
      limitUnit.value = 'TB'
    } else {
      limitValue.value = +(config.monthlyLimit / (1024 * 1024 * 1024)).toFixed(1)
      limitUnit.value = 'GB'
    }
  } else {
    form.interfaceName = ''
    form.resetDay = 1
    form.enabled = true
    limitValue.value = 0
    limitUnit.value = 'GB'
  }
  open.value = true
}

const onSubmit = async () => {
  await formRef.value?.validate()
  submitting.value = true

  const multiplier = limitUnit.value === 'TB' ? 1024 * 1024 * 1024 * 1024 : 1024 * 1024 * 1024
  form.monthlyLimit = Math.round(limitValue.value * multiplier)

  try {
    await trafficApi.createConfig({
      interfaceName: form.interfaceName,
      monthlyLimit: form.monthlyLimit,
      resetDay: form.resetDay,
      enabled: form.enabled,
    })
    ElMessage.success(t('commons.success'))
    emit('refresh')
    open.value = false
  } finally {
    submitting.value = false
  }
}

defineExpose({ acceptParams })
</script>

<style lang="scss" scoped>
.form-hint {
  font-size: 12px;
  color: var(--xp-text-muted);
  line-height: 1.4;
  margin-top: 4px;
}
</style>
