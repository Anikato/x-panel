<template>
  <el-dialog v-model="visible" :title="t('file.changePermission')" width="480px" destroy-on-close>
    <el-form label-width="90px">
      <el-form-item :label="t('file.path')">
        <el-input :model-value="form.path" disabled />
      </el-form-item>
      <el-form-item :label="t('file.owner')">
        <el-checkbox-group v-model="ownerPerms" @change="updateCode">
          <el-checkbox label="r">{{ t('file.read') }}</el-checkbox>
          <el-checkbox label="w">{{ t('file.write') }}</el-checkbox>
          <el-checkbox label="x">{{ t('file.execute') }}</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item :label="t('file.groupPerm')">
        <el-checkbox-group v-model="groupPerms" @change="updateCode">
          <el-checkbox label="r">{{ t('file.read') }}</el-checkbox>
          <el-checkbox label="w">{{ t('file.write') }}</el-checkbox>
          <el-checkbox label="x">{{ t('file.execute') }}</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item :label="t('file.otherPerm')">
        <el-checkbox-group v-model="otherPerms" @change="updateCode">
          <el-checkbox label="r">{{ t('file.read') }}</el-checkbox>
          <el-checkbox label="w">{{ t('file.write') }}</el-checkbox>
          <el-checkbox label="x">{{ t('file.execute') }}</el-checkbox>
        </el-checkbox-group>
      </el-form-item>
      <el-form-item :label="t('file.permissionCode')">
        <el-input v-model="modeStr" style="width: 120px" maxlength="4" @input="parseCode" />
      </el-form-item>
      <el-form-item>
        <el-checkbox v-model="recursive">{{ t('file.recursive') }}</el-checkbox>
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">{{ t('commons.cancel') }}</el-button>
      <el-button type="primary" :loading="loading" @click="doChange">{{ t('commons.confirm') }}</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { changeFileMode } from '@/api/modules/file'

const { t } = useI18n()
const emit = defineEmits(['done'])
const visible = ref(false)
const loading = ref(false)
const form = ref({ path: '' })
const modeStr = ref('0644')
const recursive = ref(false)
const ownerPerms = ref<string[]>([])
const groupPerms = ref<string[]>([])
const otherPerms = ref<string[]>([])

function permToNum(perms: string[]): number {
  let n = 0
  if (perms.includes('r')) n += 4
  if (perms.includes('w')) n += 2
  if (perms.includes('x')) n += 1
  return n
}

function numToPerms(n: number): string[] {
  const p: string[] = []
  if (n & 4) p.push('r')
  if (n & 2) p.push('w')
  if (n & 1) p.push('x')
  return p
}

function updateCode() {
  const o = permToNum(ownerPerms.value)
  const g = permToNum(groupPerms.value)
  const t = permToNum(otherPerms.value)
  modeStr.value = `0${o}${g}${t}`
}

function parseCode(val: string) {
  const clean = val.replace(/[^0-7]/g, '')
  const digits = clean.replace(/^0+/, '').padStart(3, '0').slice(-3)
  ownerPerms.value = numToPerms(parseInt(digits[0]))
  groupPerms.value = numToPerms(parseInt(digits[1]))
  otherPerms.value = numToPerms(parseInt(digits[2]))
}

function parseModeString(mode: string) {
  // Parse "-rwxr-xr--" style strings
  if (mode.length >= 10) {
    ownerPerms.value = []
    groupPerms.value = []
    otherPerms.value = []
    if (mode[1] === 'r') ownerPerms.value.push('r')
    if (mode[2] === 'w') ownerPerms.value.push('w')
    if (mode[3] === 'x' || mode[3] === 's') ownerPerms.value.push('x')
    if (mode[4] === 'r') groupPerms.value.push('r')
    if (mode[5] === 'w') groupPerms.value.push('w')
    if (mode[6] === 'x' || mode[6] === 's') groupPerms.value.push('x')
    if (mode[7] === 'r') otherPerms.value.push('r')
    if (mode[8] === 'w') otherPerms.value.push('w')
    if (mode[9] === 'x' || mode[9] === 't') otherPerms.value.push('x')
    updateCode()
  }
}

const open = (path: string, mode: string) => {
  form.value.path = path
  recursive.value = false
  parseModeString(mode)
  visible.value = true
}

const doChange = async () => {
  loading.value = true
  try {
    await changeFileMode({ path: form.value.path, mode: modeStr.value.replace(/^0+/, '') || '0', sub: recursive.value })
    ElMessage.success(t('commons.success'))
    visible.value = false
    emit('done')
  } catch { /* */ } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>
