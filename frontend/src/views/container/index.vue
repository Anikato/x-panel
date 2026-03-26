<template>
  <div>
    <div v-if="dockerChecking" v-loading="true" style="height: 200px" />

    <!-- Docker Not Installed -->
    <el-card v-else-if="!dockerStatus.isExist" class="docker-install-card">
      <div class="docker-install-content">
        <el-icon :size="64" color="var(--xp-text-muted)"><box /></el-icon>
        <h2>{{ t('container.dockerNotInstalled') }}</h2>
        <p>{{ t('container.dockerNotInstalledDesc') }}</p>
        <el-button type="primary" size="large" :loading="installing" @click="handleInstallDocker">
          <el-icon v-if="!installing"><download /></el-icon>
          {{ installing ? t('container.installing') : t('container.installDocker') }}
        </el-button>
        <el-button @click="checkDocker">{{ t('container.recheck') }}</el-button>
        <div v-if="installLog" class="install-log">
          <div class="install-log-header">{{ t('container.installLog') }}</div>
          <pre class="log-content">{{ installLog }}</pre>
        </div>
      </div>
    </el-card>

    <!-- Docker Installed but Not Running -->
    <el-card v-else-if="!dockerStatus.isActive" class="docker-install-card">
      <div class="docker-install-content">
        <el-icon :size="64" color="var(--el-color-warning)"><warning-filled /></el-icon>
        <h2>{{ t('container.dockerNotRunning') }}</h2>
        <p>{{ t('container.dockerNotRunningDesc') }}</p>
        <el-button type="primary" @click="checkDocker">{{ t('container.recheck') }}</el-button>
      </div>
    </el-card>

    <!-- Docker Available -->
    <template v-else>
      <div class="docker-info-bar">
        <span>Docker {{ dockerStatus.version }}</span>
      </div>
      <el-tabs v-model="activeTab">
        <el-tab-pane :label="t('container.containers')" name="containers">
          <div class="app-toolbar">
            <el-button type="primary" @click="createContainerDrawer = true">{{ t('commons.create') }}</el-button>
            <div style="flex:1" />
            <el-input v-model="containerName" :placeholder="t('commons.search')" style="width:200px" clearable @clear="loadContainers" @keyup.enter="loadContainers" />
          </div>
          <el-table :data="containers" v-loading="containerLoading" :row-class-name="containerRowClass">
            <el-table-column type="selection" width="40" />
            <el-table-column prop="name" :label="t('commons.name')" min-width="160" show-overflow-tooltip />
            <el-table-column prop="image" :label="t('container.image')" min-width="160" show-overflow-tooltip />
            <el-table-column prop="state" :label="t('container.state')" width="90">
              <template #default="{ row }">
                <el-tag :type="stateTagType(row.state)" size="small">{{ row.state }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('container.resource')" min-width="150">
              <template #default="{ row }">
                <template v-if="row.state === 'running'">
                  <div class="resource-cell">
                    <span>CPU: {{ row.cpuPercent.toFixed(2) }}%</span>
                    <span>{{ t('container.mem') }}: {{ formatBytes(row.memUsage) }}</span>
                  </div>
                </template>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="ipAddress" :label="t('container.ip')" width="130">
              <template #default="{ row }">{{ row.ipAddress || '-' }}</template>
            </el-table-column>
            <el-table-column :label="t('container.ports')" min-width="200">
              <template #default="{ row }">
                <div v-if="row.ports" class="port-tags">
                  <el-tag v-for="(port, idx) in splitPorts(row.ports)" :key="idx" size="small" type="info" class="port-tag">{{ port }}</el-tag>
                </div>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>
            <el-table-column prop="status" :label="t('container.runTime')" width="130" show-overflow-tooltip />
            <el-table-column :label="t('commons.actions')" width="230" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="operate(row, 'start')" :disabled="row.state === 'running'">{{ t('container.start') }}</el-button>
                <el-button link type="primary" @click="operate(row, 'stop')" :disabled="row.state !== 'running'">{{ t('container.stop') }}</el-button>
                <el-button link type="primary" @click="operate(row, 'restart')">{{ t('container.restart') }}</el-button>
                <el-dropdown trigger="click">
                  <el-button link type="primary">{{ t('container.more') }}<el-icon class="el-icon--right"><arrow-down /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item @click="viewLogs(row)">{{ t('container.logs') }}</el-dropdown-item>
                      <el-dropdown-item divided @click="handleRemoveContainer(row)">
                        <span style="color: var(--el-color-danger)">{{ t('commons.delete') }}</span>
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </template>
            </el-table-column>
          </el-table>
          <div class="app-pagination">
            <el-pagination v-model:current-page="containerPager.page" v-model:page-size="containerPager.pageSize" :total="containerPager.total" layout="total, sizes, prev, pager, next" :page-sizes="[20,50,100]" @size-change="loadContainers" @current-change="loadContainers" />
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
    </template>

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
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Box, Download, WarningFilled, ArrowDown } from '@element-plus/icons-vue'
import type { Container, ContainerImage, ContainerNetwork, ContainerVolume, DockerStatus } from '@/api/interface'
import {
  getDockerStatus,
  installDocker, getDockerInstallLog,
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

const dockerStatus = reactive<DockerStatus>({ isExist: false, isActive: false, version: '' })
const dockerChecking = ref(true)
const installing = ref(false)
const installLog = ref('')
let installLogTimer: ReturnType<typeof setInterval> | null = null

const logsDrawer = ref(false)
const logContent = ref('')

const formatSize = (bytes: number) => {
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
}

const stateTagType = (state: string) => {
  switch (state) {
    case 'running': return 'success'
    case 'exited': return 'danger'
    case 'paused': return 'warning'
    case 'created': return 'info'
    default: return 'info'
  }
}

const containerRowClass = ({ row }: { row: Container }) => {
  return row.state === 'running' ? '' : 'row-stopped'
}

const splitPorts = (ports: string) => {
  if (!ports) return []
  return ports.split(', ').filter(Boolean)
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

const handleInstallDocker = async () => {
  await ElMessageBox.confirm(t('container.installDockerConfirm'), t('commons.tip'), { type: 'info' })
  installing.value = true
  installLog.value = ''
  try {
    await installDocker()
    ElMessage.success(t('container.installStarted'))
    startInstallLogPolling()
  } catch {
    installing.value = false
  }
}

const startInstallLogPolling = () => {
  installLogTimer = setInterval(async () => {
    try {
      const res = await getDockerInstallLog()
      installLog.value = res.data?.log || ''
      if (!res.data?.running) {
        stopInstallLogPolling()
        installing.value = false
        await checkDocker()
        if (dockerStatus.isActive) {
          ElMessage.success(t('container.installSuccess'))
        }
      }
    } catch {
      // ignore polling errors
    }
  }, 2000)
}

const stopInstallLogPolling = () => {
  if (installLogTimer) {
    clearInterval(installLogTimer)
    installLogTimer = null
  }
}

const checkDocker = async () => {
  dockerChecking.value = true
  try {
    const res = await getDockerStatus()
    Object.assign(dockerStatus, res.data || {})
  } catch {
    dockerStatus.isExist = false
    dockerStatus.isActive = false
    dockerStatus.version = ''
  } finally {
    dockerChecking.value = false
  }
}

onMounted(async () => {
  await checkDocker()
  if (dockerStatus.isActive) loadContainers()
})

onUnmounted(() => {
  stopInstallLogPolling()
})
</script>

<style scoped>
.docker-install-card {
  max-width: 700px;
  margin: 40px auto;
}
.docker-install-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 32px 0;
  text-align: center;
}
.docker-install-content h2 {
  margin: 0;
  font-size: 20px;
  color: var(--xp-text-primary);
}
.docker-install-content p {
  color: var(--xp-text-muted);
  margin: 0;
  max-width: 460px;
}
.install-log {
  width: 100%;
  margin-top: 16px;
  text-align: left;
}
.install-log-header {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
  color: var(--xp-text-secondary);
}
.docker-info-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 0;
  font-size: 13px;
  color: var(--xp-text-muted);
}
.resource-cell {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 12px;
  line-height: 1.4;
}
.port-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.port-tag {
  font-family: var(--xp-font-mono);
  font-size: 11px;
}
.text-muted {
  color: var(--xp-text-muted);
}
.log-content {
  background: var(--xp-bg-inset);
  color: var(--xp-text-primary);
  padding: 16px;
  border-radius: 8px;
  font-family: var(--xp-font-mono);
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 400px;
  overflow-y: auto;
}
:deep(.row-stopped) {
  opacity: 0.7;
}
</style>
