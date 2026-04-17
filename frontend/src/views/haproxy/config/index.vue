<template>
  <div>
    <div class="page-header">
      <h3>{{ $t('haproxy.rawConfig') }}</h3>
      <div>
        <el-tag type="warning" v-if="!readonly">{{ $t('haproxy.configWarning') }}</el-tag>
      </div>
    </div>

    <el-card shadow="never">
      <div class="toolbar">
        <el-radio-group v-model="mode" size="small" @change="onModeChange">
          <el-radio-button value="preview">{{ $t('haproxy.previewMode') }}</el-radio-button>
          <el-radio-button value="active">{{ $t('haproxy.activeMode') }}</el-radio-button>
          <el-radio-button value="custom">{{ $t('haproxy.customMode') }}</el-radio-button>
        </el-radio-group>
        <div class="actions">
          <el-button @click="load"><el-icon><Refresh /></el-icon>{{ $t('commons.refresh') }}</el-button>
          <el-button @click="doValidate" :loading="validating">{{ $t('haproxy.validate') }}</el-button>
          <el-button type="primary" v-if="mode === 'custom'" @click="doSave" :loading="saving">{{ $t('haproxy.saveAndReload') }}</el-button>
        </div>
      </div>

      <el-alert v-if="validateMsg" :title="validateMsg" :type="validateOk ? 'success' : 'error'" show-icon style="margin-bottom: 12px;" closable @close="validateMsg = ''" />

      <div class="editor">
        <el-input v-model="content" type="textarea" :rows="28" :readonly="mode !== 'custom'" class="code-editor" spellcheck="false" />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { getHAProxyRawConfig, previewHAProxyConfig, saveHAProxyRawConfig, testHAProxyConfig } from '@/api/modules/haproxy'

const { t } = useI18n()
const mode = ref<'preview' | 'active' | 'custom'>('preview')
const content = ref('')
const validating = ref(false)
const saving = ref(false)
const validateMsg = ref('')
const validateOk = ref(false)

const readonly = computed(() => mode.value !== 'custom')

const load = async () => {
  try {
    if (mode.value === 'preview') {
      const res = await previewHAProxyConfig()
      content.value = res.data?.content || ''
    } else {
      const res = await getHAProxyRawConfig()
      content.value = res.data?.content || ''
    }
    validateMsg.value = ''
  } catch {}
}

const onModeChange = () => load()

const doValidate = async () => {
  validating.value = true
  try {
    const res = await testHAProxyConfig({ content: content.value })
    validateOk.value = !!res.data?.valid
    validateMsg.value = res.data?.valid
      ? t('haproxy.validateOK')
      : `${t('haproxy.validateFail')}: ${res.data?.output || ''}`
  } finally {
    validating.value = false
  }
}

const doSave = async () => {
  await ElMessageBox.confirm(t('haproxy.saveRawConfirm'), t('commons.warning'), { type: 'warning' })
  saving.value = true
  try {
    await saveHAProxyRawConfig({ content: content.value })
    ElMessage.success(t('haproxy.saveAndReloadSuccess'))
  } finally {
    saving.value = false
  }
}

onMounted(() => load())
</script>

<style lang="scss" scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 16px;
  h3 { margin: 0; }
}
.toolbar {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: 12px;
  .actions > * + * { margin-left: 8px; }
}
.code-editor :deep(.el-textarea__inner) {
  font-family: 'Menlo', 'Monaco', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.5;
}
</style>
