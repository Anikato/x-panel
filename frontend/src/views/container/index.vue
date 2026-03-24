<template>
  <div>
    <div v-if="dockerChecking" v-loading="true" style="height: 200px" />
    <el-empty v-else-if="!dockerAvailable" description="Docker 未安装或未启动">
      <template #default>
        <p style="color: var(--xp-text-muted); margin-bottom: 16px">请先安装并启动 Docker 服务后刷新页面</p>
        <el-button type="primary" @click="checkDocker">重新检测</el-button>
      </template>
    </el-empty>
    <el-tabs v-else v-model="activeTab">
      <el-tab-pane :label="t('container.containers')" name="containers">
        <div class="app-toolbar">
          <el-button type="primary" @click="createContainerDrawer = true">{{ t('commons.create') }}</el-button>
          <div style="flex:1" />
          <el-input v-model="containerName" :placeholder="t('commons.search')" style="width:200px" clearable @clear="loadContainers" @keyup.enter="loadContainers" />
        </div>
        <el-table :data="containers" v-loading="containerLoading">
          <el-table-column prop="name" :label="t('commons.name')" min-width="160" />
          <el-table-column prop="image" :label="t('container.image')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="state" :label="t('container.state')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.state === 'running' ? 'success' : 'info'" size="small">{{ row.state }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="status" :label="t('container.status')" width="180" show-overflow-tooltip />
          <el-table-column :label="t('commons.actions')" width="280" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="operate(row, 'start')" :disabled="row.state === 'running'">{{ t('container.start') }}</el-button>
              <el-button link type="primary" @click="operate(row, 'stop')" :disabled="row.state !== 'running'">{{ t('container.stop') }}</el-button>
              <el-button link type="primary" @click="operate(row, 'restart')">{{ t('container.restart') }}</el-button>
              <el-button link type="primary" @click="viewLogs(row)">{{ t('container.logs') }}</el-button>
              <el-button link type="danger" @click="handleRemoveContainer(row)">{{ t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="app-pagination">
          <el-pagination v-model:current-page="containerPager.page" v-model:page-size="containerPager.pageSize" :total="containerPager.total" layout="total, sizes, prev, pager, next" :page-sizes="[20,50]" @size-change="loadContainers" @current-change="loadContainers" />
        </div>
      </el-tab-pane>

      <el-tab-pane :label="t('container.images')" name="images">
        <div class="app-toolbar">
          <el-button type="primary" @click="pullDrawer = true">{{ t('container.pullImage') }}</el-button>
        </div>
        <el-table :data="images" v-loading="imageLoading">
          <el-table-column prop="id" label="ID" width="140" />
          <el-table-column :label="t('container.tags')" min-width="240">
            <template #default="{ row }">{{ (row.tags || []).join(', ') || '-' }}</template>
          </el-table-column>
          <el-table-column :label="t('container.size')" width="120">
            <template #default="{ row }">{{ formatSize(row.size) }}</template>
          </el-table-column>
          <el-table-column :label="t('commons.actions')" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="handleRemoveImage(row)">{{ t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('container.networks')" name="networks">
        <div class="app-toolbar">
          <el-button type="primary" @click="networkCreateDialog = true">{{ t('commons.create') }}</el-button>
        </div>
        <el-table :data="networks" v-loading="networkLoading">
          <el-table-column prop="name" :label="t('commons.name')" min-width="160" />
          <el-table-column prop="driver" label="Driver" width="120" />
          <el-table-column prop="subnet" label="Subnet" width="160" />
          <el-table-column prop="gateway" label="Gateway" width="160" />
          <el-table-column :label="t('commons.actions')" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="handleRemoveNetwork(row)">{{ t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane :label="t('container.volumes')" name="volumes">
        <div class="app-toolbar">
          <el-button type="primary" @click="volumeCreateDialog = true">{{ t('commons.create') }}</el-button>
        </div>
        <el-table :data="volumes" v-loading="volumeLoading">
          <el-table-column prop="name" :label="t('commons.name')" min-width="200" />
          <el-table-column prop="driver" label="Driver" width="120" />
          <el-table-column prop="mountPoint" :label="t('container.mountPoint')" min-width="240" show-overflow-tooltip />
          <el-table-column :label="t('commons.actions')" width="100">
            <template #default="{ row }">
              <el-button link type="danger" @click="handleRemoveVolume(row)">{{ t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- Create Container Drawer -->
    <el-drawer v-model="createContainerDrawer" :title="t('container.createContainer')" size="560px" destroy-on-close>
      <el-form ref="containerFormRef" :model="containerForm" label-width="100px">
        <el-form-item :label="t('commons.name')" prop="name"><el-input v-model="containerForm.name" /></el-form-item>
        <el-form-item :label="t('container.image')" prop="image"><el-input v-model="containerForm.image" /></el-form-item>
        <el-form-item :label="t('container.restartPolicy')">
          <el-select v-model="containerForm.restartPolicy" style="width:100%">
            <el-option label="no" value="" /><el-option label="always" value="always" /><el-option label="unless-stopped" value="unless-stopped" /><el-option label="on-failure" value="on-failure" />
          </el-select>
        </el-form-item>
        <el-form-item label="CMD"><el-input v-model="cmdStr" placeholder="e.g. /bin/sh -c 'echo hello'" /></el-form-item>
        <el-form-item label="ENV"><el-input v-model="envStr" type="textarea" :rows="3" placeholder="KEY=VALUE (one per line)" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createContainerDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="submitContainer">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-drawer>

    <!-- Pull Image -->
    <el-dialog v-model="pullDrawer" :title="t('container.pullImage')" width="420px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item :label="t('container.image')"><el-input v-model="pullImageName" placeholder="nginx:latest" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pullDrawer = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="pulling" @click="handlePullImage">{{ t('container.pullImage') }}</el-button>
      </template>
    </el-dialog>

    <!-- Network Create -->
    <el-dialog v-model="networkCreateDialog" :title="t('container.createNetwork')" width="420px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item :label="t('commons.name')"><el-input v-model="networkForm.name" /></el-form-item>
        <el-form-item label="Driver"><el-input v-model="networkForm.driver" placeholder="bridge" /></el-form-item>
        <el-form-item label="Subnet"><el-input v-model="networkForm.subnet" placeholder="172.20.0.0/16" /></el-form-item>
        <el-form-item label="Gateway"><el-input v-model="networkForm.gateway" placeholder="172.20.0.1" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="networkCreateDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreateNetwork">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Volume Create -->
    <el-dialog v-model="volumeCreateDialog" :title="t('container.createVolume')" width="420px" destroy-on-close>
      <el-form label-width="80px">
        <el-form-item :label="t('commons.name')"><el-input v-model="volumeForm.name" /></el-form-item>
        <el-form-item label="Driver"><el-input v-model="volumeForm.driver" placeholder="local" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="volumeCreateDialog = false">{{ t('commons.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreateVolume">{{ t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- Logs Drawer -->
    <el-drawer v-model="logsDrawer" :title="t('container.logs')" size="640px" destroy-on-close>
      <pre class="log-content">{{ logContent }}</pre>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import type { Container, ContainerImage, ContainerNetwork, ContainerVolume } from '@/api/interface'
import {
  getDockerStatus,
  searchContainers, createContainer, operateContainer, containerLogs, removeContainer,
  listImages, pullImage, removeImage,
  listNetworks, createNetwork, removeNetwork,
  listVolumes, createVolume, removeVolume,
} from '@/api/modules/container'

const { t } = useI18n()
const activeTab = ref('containers')

const containerLoading = ref(false)
const containers = ref<Container[]>([])
const containerName = ref('')
const containerPager = reactive({ page: 1, pageSize: 20, total: 0 })

const imageLoading = ref(false)
const images = ref<ContainerImage[]>([])

const networkLoading = ref(false)
const networks = ref<ContainerNetwork[]>([])

const volumeLoading = ref(false)
const volumes = ref<ContainerVolume[]>([])

const submitting = ref(false)
const createContainerDrawer = ref(false)
const containerForm = reactive({ name: '', image: '', restartPolicy: '', env: [] as string[], cmd: [] as string[] })
const cmdStr = ref('')
const envStr = ref('')

const pullDrawer = ref(false)
const pullImageName = ref('')
const pulling = ref(false)

const networkCreateDialog = ref(false)
const networkForm = reactive({ name: '', driver: 'bridge', subnet: '', gateway: '' })

const volumeCreateDialog = ref(false)
const volumeForm = reactive({ name: '', driver: 'local' })

const dockerAvailable = ref(true)
const dockerChecking = ref(true)

const logsDrawer = ref(false)
const logContent = ref('')

const formatSize = (bytes: number) => {
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const loadContainers = async () => {
  containerLoading.value = true
  try {
    const res = await searchContainers({ page: containerPager.page, pageSize: containerPager.pageSize, name: containerName.value })
    containers.value = res.data.items || []
    containerPager.total = res.data.total
  } finally { containerLoading.value = false }
}

const loadImages = async () => {
  imageLoading.value = true
  try {
    const res = await listImages()
    images.value = res.data || []
  } finally { imageLoading.value = false }
}

const loadNetworks = async () => {
  networkLoading.value = true
  try {
    const res = await listNetworks()
    networks.value = res.data || []
  } finally { networkLoading.value = false }
}

const loadVolumes = async () => {
  volumeLoading.value = true
  try {
    const res = await listVolumes()
    volumes.value = res.data || []
  } finally { volumeLoading.value = false }
}

watch(activeTab, (tab) => {
  if (tab === 'containers') loadContainers()
  else if (tab === 'images') loadImages()
  else if (tab === 'networks') loadNetworks()
  else if (tab === 'volumes') loadVolumes()
})

const operate = async (row: Container, op: string) => {
  await operateContainer({ containerID: row.id, operation: op })
  ElMessage.success(t('commons.success'))
  await loadContainers()
}

const viewLogs = async (row: Container) => {
  const res = await containerLogs({ containerID: row.id, tail: '200' })
  logContent.value = res.data || ''
  logsDrawer.value = true
}

const handleRemoveContainer = async (row: Container) => {
  await ElMessageBox.confirm(t('container.deleteContainerConfirm'), t('commons.tip'), { type: 'warning' })
  await removeContainer({ containerID: row.id })
  ElMessage.success(t('commons.success'))
  await loadContainers()
}

const submitContainer = async () => {
  submitting.value = true
  try {
    const payload = {
      ...containerForm,
      cmd: cmdStr.value ? cmdStr.value.split(/\s+/) : [],
      env: envStr.value ? envStr.value.split('\n').filter(Boolean) : [],
    }
    await createContainer(payload)
    ElMessage.success(t('commons.success'))
    createContainerDrawer.value = false
    await loadContainers()
  } finally { submitting.value = false }
}

const handlePullImage = async () => {
  pulling.value = true
  try {
    await pullImage({ imageName: pullImageName.value })
    ElMessage.success(t('commons.success'))
    pullDrawer.value = false
    await loadImages()
  } finally { pulling.value = false }
}

const handleRemoveImage = async (row: ContainerImage) => {
  await ElMessageBox.confirm(t('container.deleteImageConfirm'), t('commons.tip'), { type: 'warning' })
  await removeImage({ imageID: row.id })
  ElMessage.success(t('commons.success'))
  await loadImages()
}

const handleCreateNetwork = async () => {
  submitting.value = true
  try {
    await createNetwork(networkForm)
    ElMessage.success(t('commons.success'))
    networkCreateDialog.value = false
    await loadNetworks()
  } finally { submitting.value = false }
}

const handleRemoveNetwork = async (row: ContainerNetwork) => {
  await ElMessageBox.confirm(t('container.deleteNetworkConfirm'), t('commons.tip'), { type: 'warning' })
  await removeNetwork({ networkID: row.id })
  ElMessage.success(t('commons.success'))
  await loadNetworks()
}

const handleCreateVolume = async () => {
  submitting.value = true
  try {
    await createVolume(volumeForm)
    ElMessage.success(t('commons.success'))
    volumeCreateDialog.value = false
    await loadVolumes()
  } finally { submitting.value = false }
}

const handleRemoveVolume = async (row: ContainerVolume) => {
  await ElMessageBox.confirm(t('container.deleteVolumeConfirm'), t('commons.tip'), { type: 'warning' })
  await removeVolume({ name: row.name })
  ElMessage.success(t('commons.success'))
  await loadVolumes()
}

const checkDocker = async () => {
  dockerChecking.value = true
  try {
    const res = await getDockerStatus()
    dockerAvailable.value = res.data?.available === true
  } catch {
    dockerAvailable.value = false
  } finally {
    dockerChecking.value = false
  }
}

onMounted(async () => {
  await checkDocker()
  if (dockerAvailable.value) loadContainers()
})
</script>

<style scoped>
.log-content {
  background: var(--xp-bg-inset);
  color: var(--xp-text-primary);
  padding: 16px;
  border-radius: 8px;
  font-family: var(--xp-font-mono);
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 70vh;
  overflow-y: auto;
}
</style>
