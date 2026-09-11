<template>
  <div class="log-meta-banner">
    <el-alert type="info" :closable="false" show-icon>
      {{ $t('nginx.logScopeActiveOnly') }}
    </el-alert>
    <el-alert v-if="view === 'updating'" type="info" :closable="false" show-icon>
      {{ $t('nginx.logUpdating') }}
    </el-alert>
    <el-alert v-if="view === 'stale'" type="warning" :closable="false" show-icon>
      {{ $t('nginx.logUpdateFailedKeepLast') }}
    </el-alert>
    <el-alert v-if="view === 'error'" type="error" :closable="false" show-icon>
      {{ error || $t('nginx.logAnalyzeFailed') }}
    </el-alert>
    <el-alert v-if="meta?.partial" type="warning" :closable="false" show-icon>
      {{ partialText }}
    </el-alert>
    <el-alert v-if="meta?.sharedLog" type="warning" :closable="false" show-icon>
      {{ $t('nginx.logSharedUnattributed') }}
    </el-alert>
    <div v-if="meta" class="meta-line">
      <span v-if="observedText">{{ $t('nginx.logObservedRange') }}: {{ observedText }}</span>
      <span v-if="meta.generatedAt">{{ $t('nginx.logGeneratedAt') }}: {{ formatTime(meta.generatedAt) }}</span>
      <span v-if="meta.invalidLines">{{ $t('nginx.logInvalidLines', { count: meta.invalidLines }) }}</span>
    </div>
    <el-collapse v-if="hasDetails" class="meta-details">
      <el-collapse-item :title="$t('nginx.logDetails')" name="details">
        <div v-if="reasonLabels.length">{{ reasonLabels.join('；') }}</div>
        <div v-for="item in meta?.failedFiles || []" :key="'f-'+item.path">{{ $t('nginx.logFailedFiles') }}: {{ item.path }} ({{ reasonLabel(item.reason) }})</div>
        <div v-for="item in meta?.skippedFiles || []" :key="'s-'+item.path">{{ $t('nginx.logSkippedFiles') }}: {{ item.path }} ({{ reasonLabel(item.reason) }})</div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { NginxLogAnalysisMeta } from '@/api/modules/website'
import type { AnalysisViewKind } from './log-analysis-request'

const props = defineProps<{
  meta?: NginxLogAnalysisMeta | null
  view: AnalysisViewKind
  error?: string | null
}>()

const { t } = useI18n()

const formatTime = (value?: string) => {
  if (!value) return ''
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}

const observedText = computed(() => {
  if (!props.meta?.observedFrom || !props.meta?.observedTo) return ''
  return `${formatTime(props.meta.observedFrom)} ~ ${formatTime(props.meta.observedTo)}`
})

const reasonLabel = (reason: string) => {
  const key = `nginx.reason_${reason}`
  const translated = t(key)
  return translated === key ? reason : translated
}

const reasonLabels = computed(() => (props.meta?.reasons || []).map(reasonLabel))

const partialText = computed(() => {
  const reasons = props.meta?.reasons || []
  if (reasons.includes('line_limit')) {
    return t('nginx.logPartialLineLimit', { lines: props.meta?.scannedLines || 0 })
  }
  return t('nginx.logPartialGeneric')
})

const hasDetails = computed(() => {
  const meta = props.meta
  if (!meta) return false
  return (meta.reasons && meta.reasons.length > 0) || (meta.failedFiles && meta.failedFiles.length > 0) || (meta.skippedFiles && meta.skippedFiles.length > 0)
})
</script>

<style scoped lang="scss">
.log-meta-banner {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 12px;
}
.meta-line {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 20px;
  font-size: 12px;
  color: var(--xp-text-muted);
}
.meta-details {
  --el-collapse-header-height: 36px;
}
</style>
