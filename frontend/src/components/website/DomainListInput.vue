<template>
  <div class="domain-list">
    <el-tag
      v-for="domain in domains"
      :key="domain"
      :closable="!disabled"
      size="small"
      @close="remove(domain)"
    >
      {{ domain }}
    </el-tag>
    <el-input
      v-if="!disabled"
      v-model="draft"
      class="domain-list-input"
      :placeholder="placeholder"
      @keydown.enter.prevent="commit"
      @blur="commit"
    />
    <div class="form-tip">回车添加。可以一次粘贴多个，用逗号、空格或换行分开。</div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { joinDomainList, parseDomainList } from '@/utils/domains'

const props = withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  placeholder: 'www.example.com',
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const draft = ref('')
const domains = computed(() => parseDomainList(props.modelValue || ''))

const commit = () => {
  const next = joinDomainList([...domains.value, ...parseDomainList(draft.value)])
  draft.value = ''
  if (next !== (props.modelValue || '')) {
    emit('update:modelValue', next)
  }
}

const remove = (domain: string) => {
  emit('update:modelValue', joinDomainList(domains.value.filter(item => item !== domain)))
}
</script>

<style scoped>
.domain-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  width: 100%;
  align-items: center;
}
.domain-list-input {
  width: 220px;
}
</style>
