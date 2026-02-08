<template>
  <div class="firewall-page">
    <div class="page-header">
      <h3>{{ $t('firewall.title') }}</h3>
      <div class="header-actions">
        <el-tag :type="baseInfo.isActive ? 'success' : 'danger'" size="default">
          {{ baseInfo.isExist ? (baseInfo.isActive ? $t('firewall.enabled') : $t('firewall.disabled')) : $t('firewall.notInstalled') }}
        </el-tag>
        <template v-if="baseInfo.isExist">
          <el-button size="small" type="success" plain @click="handleOperate('enable')" v-if="!baseInfo.isActive">{{ $t('firewall.enable') }}</el-button>
          <el-button size="small" type="danger" plain @click="handleOperate('disable')" v-if="baseInfo.isActive">{{ $t('firewall.disable') }}</el-button>
          <el-button size="small" type="warning" plain @click="handleOperate('reload')" v-if="baseInfo.isActive">{{ $t('firewall.reload') }}</el-button>
        </template>
      </div>
    </div>

    <el-tabs v-model="activeTab" v-if="baseInfo.isExist">
      <!-- 端口规则 -->
      <el-tab-pane :label="$t('firewall.portRules')" name="port">
        <div class="toolbar">
          <el-input v-model="portSearch" :placeholder="$t('commons.search')" prefix-icon="Search" size="small" clearable class="search-input" @input="loadPortRules" />
          <el-button size="small" type="primary" @click="portDialogVisible = true">
            <el-icon><Plus /></el-icon>
            {{ $t('firewall.addRule') }}
          </el-button>
        </div>
        <el-table :data="portRules" size="small" v-loading="portLoading">
          <el-table-column prop="port" :label="$t('firewall.port')" width="140" />
          <el-table-column prop="protocol" :label="$t('firewall.protocol')" width="100">
            <template #default="{ row }">
              <el-tag size="small">{{ row.protocol }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="strategy" :label="$t('firewall.strategy')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.strategy === 'allow' ? 'success' : 'danger'" size="small">
                {{ row.strategy === 'allow' ? $t('firewall.allow') : $t('firewall.deny') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="from" :label="$t('firewall.from')" min-width="150" />
          <el-table-column :label="$t('commons.actions')" width="80" fixed="right">
            <template #default="{ row }">
              <el-button link type="danger" size="small" @click="handleDeletePort(row)">{{ $t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination v-if="portTotal > 0" class="mt-12" :current-page="portPage" :page-size="portPageSize" :total="portTotal" layout="total, prev, pager, next" @current-change="(p: number) => { portPage = p; loadPortRules() }" />
      </el-tab-pane>

      <!-- IP 规则 -->
      <el-tab-pane :label="$t('firewall.ipRules')" name="ip">
        <div class="toolbar">
          <el-button size="small" type="primary" @click="ipDialogVisible = true">
            <el-icon><Plus /></el-icon>
            {{ $t('firewall.addRule') }}
          </el-button>
        </div>
        <el-table :data="ipRules" size="small" v-loading="ipLoading">
          <el-table-column prop="address" :label="$t('firewall.address')" min-width="200" />
          <el-table-column prop="strategy" :label="$t('firewall.strategy')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.strategy === 'allow' ? 'success' : 'danger'" size="small">
                {{ row.strategy === 'allow' ? $t('firewall.allow') : $t('firewall.deny') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$t('commons.actions')" width="80">
            <template #default="{ row }">
              <el-button link type="danger" size="small" @click="handleDeleteIP(row)">{{ $t('commons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-empty v-else-if="!loading" :description="$t('firewall.notInstalled') + ' (ufw)'" />

    <!-- 端口规则对话框 -->
    <el-dialog v-model="portDialogVisible" :title="$t('firewall.addRule')" width="460px" destroy-on-close>
      <el-form :model="portForm" label-width="80px">
        <el-form-item :label="$t('firewall.port')">
          <el-input v-model="portForm.port" placeholder="80 或 8000:8100" />
        </el-form-item>
        <el-form-item :label="$t('firewall.protocol')">
          <el-select v-model="portForm.protocol" style="width: 100%">
            <el-option label="TCP" value="tcp" />
            <el-option label="UDP" value="udp" />
            <el-option label="TCP/UDP" value="tcp/udp" />
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('firewall.strategy')">
          <el-radio-group v-model="portForm.strategy">
            <el-radio value="allow">{{ $t('firewall.allow') }}</el-radio>
            <el-radio value="deny">{{ $t('firewall.deny') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('firewall.from')">
          <el-input v-model="portForm.from" :placeholder="$t('firewall.anywhere')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="portDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreatePort" :loading="portSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>

    <!-- IP 规则对话框 -->
    <el-dialog v-model="ipDialogVisible" :title="$t('firewall.addRule')" width="400px" destroy-on-close>
      <el-form :model="ipForm" label-width="80px">
        <el-form-item :label="$t('firewall.address')">
          <el-input v-model="ipForm.address" placeholder="192.168.1.100" />
        </el-form-item>
        <el-form-item :label="$t('firewall.strategy')">
          <el-radio-group v-model="ipForm.strategy">
            <el-radio value="allow">{{ $t('firewall.allow') }}</el-radio>
            <el-radio value="deny">{{ $t('firewall.deny') }}</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ipDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreateIP" :loading="ipSubmitting">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { getFirewallBase, operateFirewall, searchPortRules, createPortRule, deletePortRule, getIPRules, createIPRule, deleteIPRule } from '@/api/modules/firewall'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)
const activeTab = ref('port')
const baseInfo = ref<any>({})

// 端口规则
const portLoading = ref(false)
const portRules = ref<any[]>([])
const portTotal = ref(0)
const portPage = ref(1)
const portPageSize = ref(20)
const portSearch = ref('')
const portDialogVisible = ref(false)
const portSubmitting = ref(false)
const portForm = ref({ port: '', protocol: 'tcp', strategy: 'allow', from: '' })

// IP 规则
const ipLoading = ref(false)
const ipRules = ref<any[]>([])
const ipDialogVisible = ref(false)
const ipSubmitting = ref(false)
const ipForm = ref({ address: '', strategy: 'deny' })

const loadBase = async () => {
  loading.value = true
  try {
    const res = await getFirewallBase()
    baseInfo.value = res.data || {}
  } catch { /* handled */ }
  finally { loading.value = false }
}

const handleOperate = async (op: string) => {
  try {
    await operateFirewall(op)
    ElMessage.success(t('commons.success'))
    setTimeout(loadBase, 500)
  } catch { /* handled */ }
}

const loadPortRules = async () => {
  portLoading.value = true
  try {
    const res = await searchPortRules({ page: portPage.value, pageSize: portPageSize.value, info: portSearch.value })
    portRules.value = res.data?.items || []
    portTotal.value = res.data?.total || 0
  } catch { portRules.value = [] }
  finally { portLoading.value = false }
}

const handleCreatePort = async () => {
  if (!portForm.value.port) { ElMessage.warning('请输入端口'); return }
  portSubmitting.value = true
  try {
    await createPortRule(portForm.value)
    ElMessage.success(t('commons.success'))
    portDialogVisible.value = false
    loadPortRules()
  } catch { /* handled */ }
  finally { portSubmitting.value = false }
}

const handleDeletePort = async (row: any) => {
  await ElMessageBox.confirm(t('firewall.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  try {
    await deletePortRule(row)
    ElMessage.success(t('commons.success'))
    loadPortRules()
  } catch { /* handled */ }
}

const loadIPRules = async () => {
  ipLoading.value = true
  try {
    const res = await getIPRules()
    ipRules.value = res.data || []
  } catch { ipRules.value = [] }
  finally { ipLoading.value = false }
}

const handleCreateIP = async () => {
  if (!ipForm.value.address) { ElMessage.warning('请输入 IP 地址'); return }
  ipSubmitting.value = true
  try {
    await createIPRule(ipForm.value)
    ElMessage.success(t('commons.success'))
    ipDialogVisible.value = false
    loadIPRules()
  } catch { /* handled */ }
  finally { ipSubmitting.value = false }
}

const handleDeleteIP = async (row: any) => {
  await ElMessageBox.confirm(t('firewall.deleteConfirm'), t('commons.tip'), { type: 'warning' })
  try {
    await deleteIPRule(row)
    ElMessage.success(t('commons.success'))
    loadIPRules()
  } catch { /* handled */ }
}

watch(activeTab, (val) => {
  if (val === 'ip' && ipRules.value.length === 0) loadIPRules()
})

onMounted(async () => {
  await loadBase()
  if (baseInfo.value.isExist) {
    loadPortRules()
  }
})
</script>

<style lang="scss" scoped>
.firewall-page { height: 100%; }

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  h3 { margin: 0; font-size: 16px; color: var(--xp-text-primary); }
  .header-actions { display: flex; align-items: center; gap: 8px; }
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  .search-input { width: 240px; }
}

.mt-12 { margin-top: 12px; }
</style>
