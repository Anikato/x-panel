<template>
  <el-dialog v-model="visible" :title="t('file.changePermission')" width="480px" destroy-on-close>
    <el-form label-width="90px" v-loading="loadingUsers">
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
      <el-divider />
      <el-form-item :label="t('file.ownerUser')">
        <el-select
          v-model="form.user"
          filterable
          allow-create
          default-first-option
          clearable
          style="width: 100%"
          :placeholder="t('file.ownerUser') + ' / UID'"
        >
          <el-option
            v-for="u in users"
            :key="u.uid"
            :label="`${u.username} (${u.uid}) · ${u.group}${u.system ? ' · system' : ''}`"
            :value="u.username"
          />
        </el-select>
      </el-form-item>
      <el-form-item :label="t('file.ownerGroup')">
        <el-select
          v-model="form.group"
          filterable
          allow-create
          default-first-option
          clearable
          style="width: 100%"
          :placeholder="t('file.ownerGroup') + ' / GID'"
        >
          <el-option v-for="g in groups" :key="g" :label="g" :value="g" />
        </el-select>
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
import { changeFileMode, changeFileOwner, getUsersAndGroups } from '@/api/modules/file'

const { t } = useI18n()
const emit = defineEmits(['done'])
const visible = ref(false)
const loading = ref(false)
const loadingUsers = ref(false)
const form = ref({ path: '', user: '', group: '' })
const modeStr = ref('0644')
const recursive = ref(true)
const ownerPerms = ref<string[]>([])
const groupPerms = ref<string[]>([])
const otherPerms = ref<string[]>([])
const users = ref<{ username: string; group: string; uid: string; gid: string; system?: boolean }[]>([])
const groups = ref<string[]>([])

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

const loadUsersAndGroups = async () => {
  loadingUsers.value = true
  try {
    const res = await getUsersAndGroups()
    users.value = res.data?.users || []
    groups.value = res.data?.groups || []
  } catch { /* */ } finally {
    loadingUsers.value = false
  }
}

const open = (path: string, mode: string, currentUser = '', currentGroup = '') => {
  form.value = { path, user: currentUser, group: currentGroup }
  recursive.value = true
  parseModeString(mode)
  visible.value = true
  loadUsersAndGroups()
}

const doChange = async () => {
  loading.value = true
  try {
    await changeFileMode({ path: form.value.path, mode: modeStr.value.replace(/^0+/, '') || '0', sub: recursive.value })
    if (form.value.user && form.value.group) {
      await changeFileOwner({
        path: form.value.path,
        user: form.value.user,
        group: form.value.group,
        sub: recursive.value,
      })
    }
    ElMessage.success(t('commons.success'))
    visible.value = false
    emit('done')
  } catch { /* */ } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>
