<template>
  <div class="website-page">
    <div class="page-header">
      <h3>{{ $t('website.title') }}</h3>
      <el-button size="small" type="primary" @click="openCreateDialog">
        <el-icon><Plus /></el-icon>
        {{ $t('website.create') }}
      </el-button>
    </div>

    <div class="filter-bar">
      <el-input v-model="searchInfo" :placeholder="$t('commons.search')" prefix-icon="Search" size="small" clearable class="search-input" @input="loadWebsites" />
      <el-select v-model="filterType" size="small" clearable :placeholder="$t('website.type')" @change="loadWebsites">
        <el-option :label="$t('website.typeStatic')" value="static" />
        <el-option :label="$t('website.typeProxy')" value="reverse_proxy" />
      </el-select>
      <el-select v-model="filterStatus" size="small" clearable :placeholder="$t('website.status')" @change="loadWebsites">
        <el-option :label="$t('website.running')" value="running" />
        <el-option :label="$t('website.stopped')" value="stopped" />
      </el-select>
    </div>

    <el-table :data="websites" style="width: 100%" v-loading="loading">
      <el-table-column prop="primaryDomain" :label="$t('website.domain')" min-width="200">
        <template #default="{ row }">
          <div class="domain-cell">
            <el-link type="primary" @click="goConfig(row.id)">{{ row.primaryDomain }}</el-link>
            <el-tag v-if="row.sslEnable" type="success" size="small" effect="plain" class="ssl-badge">SSL</el-tag>
            <span v-if="row.domains" class="domain-extra">+{{ row.domains.split(',').length }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.type')" width="120">
        <template #default="{ row }">
          <el-tag :type="row.type === 'static' ? 'info' : 'warning'" size="small">
            {{ row.type === 'static' ? $t('website.typeStatic') : $t('website.typeProxy') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('website.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'running' ? 'success' : 'danger'" size="small" effect="dark" round>
            {{ row.status === 'running' ? $t('website.running') : $t('website.stopped') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="remark" :label="$t('commons.description')" min-width="120" show-overflow-tooltip />
      <el-table-column :label="$t('commons.actions')" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click="goConfig(row.id)">
            {{ $t('commons.edit') }}
          </el-button>
          <el-button v-if="row.status === 'stopped'" link type="success" size="small" @click="handleEnable(row)">
            {{ $t('website.enable') }}
          </el-button>
          <el-button v-else link type="warning" size="small" @click="handleDisable(row)">
            {{ $t('website.disable') }}
          </el-button>
          <el-button link type="danger" size="small" @click="handleDelete(row)">
            {{ $t('commons.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination v-if="total > 0" class="mt-pagination" :current-page="page" :page-size="pageSize" :total="total" layout="total, prev, pager, next" @current-change="(p: number) => { page = p; loadWebsites() }" />

    <!-- 创建网站对话框 -->
    <el-dialog v-model="createDialogVisible" :title="$t('website.create')" width="540px" destroy-on-close>
      <el-form :model="createForm" label-width="100px">
        <el-form-item :label="$t('website.type')">
          <el-radio-group v-model="createForm.type">
            <el-radio value="static">{{ $t('website.typeStatic') }}</el-radio>
            <el-radio value="reverse_proxy">{{ $t('website.typeProxy') }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="$t('website.domain')">
          <el-input v-model="createForm.primaryDomain" placeholder="example.com" />
        </el-form-item>
        <el-form-item :label="$t('website.otherDomains')">
          <el-input v-model="createForm.domains" placeholder="www.example.com" />
          <div class="form-tip">{{ $t('website.otherDomainsHint') }}</div>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'static'" :label="$t('website.siteDir')">
          <el-input v-model="createForm.siteDir" :placeholder="`/var/www/${createForm.primaryDomain || 'example.com'}`" />
          <div class="form-tip">{{ $t('website.siteDirHint') }}</div>
        </el-form-item>
        <el-form-item v-if="createForm.type === 'reverse_proxy'" :label="$t('website.proxyPass')">
          <el-input v-model="createForm.proxyPass" placeholder="http://127.0.0.1:8080" />
          <div class="form-tip">{{ $t('website.proxyPassHint') }}</div>
        </el-form-item>
        <el-form-item :label="$t('commons.description')">
          <el-input v-model="createForm.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialogVisible = false">{{ $t('commons.cancel') }}</el-button>
        <el-button type="primary" @click="handleCreate" :loading="createLoading">{{ $t('commons.confirm') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { searchWebsite, createWebsite, deleteWebsite, enableWebsite, disableWebsite } from '@/api/modules/website'

const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const websites = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const searchInfo = ref('')
const filterType = ref('')
const filterStatus = ref('')

const createDialogVisible = ref(false)
const createLoading = ref(false)
const createForm = ref({
  primaryDomain: '',
  domains: '',
  type: 'static',
  remark: '',
  siteDir: '',
  proxyPass: '',
})

const loadWebsites = async () => {
  loading.value = true
  try {
    const res = await searchWebsite({
      page: page.value,
      pageSize: pageSize.value,
      info: searchInfo.value,
      type: filterType.value,
      status: filterStatus.value,
    })
    websites.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch { websites.value = [] }
  finally { loading.value = false }
}

const openCreateDialog = () => {
  createForm.value = { primaryDomain: '', domains: '', type: 'static', remark: '', siteDir: '', proxyPass: '' }
  createDialogVisible.value = true
}

const handleCreate = async () => {
  if (!createForm.value.primaryDomain) { ElMessage.warning('请输入域名'); return }
  if (createForm.value.type === 'reverse_proxy' && !createForm.value.proxyPass) { ElMessage.warning('请输入代理地址'); return }
  createLoading.value = true
  try {
    await createWebsite(createForm.value)
    ElMessage.success(t('commons.success'))
    createDialogVisible.value = false
    loadWebsites()
  } catch {}
  finally { createLoading.value = false }
}

const handleEnable = async (row: any) => {
  try {
    await ElMessageBox.confirm(t('website.enableConfirm'), t('commons.tip'), { type: 'info' })
    await enableWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const handleDisable = async (row: any) => {
  try {
    await ElMessageBox.confirm(t('website.disableConfirm'), t('commons.tip'), { type: 'warning' })
    await disableWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(t('website.deleteConfirm'), t('commons.tip'), { type: 'error' })
    await deleteWebsite(row.id)
    ElMessage.success(t('commons.success'))
    loadWebsites()
  } catch {}
}

const goConfig = (id: number) => {
  router.push(`/website/websites/${id}`)
}

onMounted(() => loadWebsites())
</script>

<style lang="scss" scoped>
.website-page {
  height: 100%;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;

  h3 {
    margin: 0;
    font-size: 16px;
    color: var(--xp-text-primary);
  }
}

.filter-bar {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;

  .search-input {
    width: 240px;
  }
}

.mt-pagination {
  margin-top: 16px;
  justify-content: flex-end;
}

.domain-cell {
  display: flex;
  align-items: center;
  gap: 6px;

  .ssl-badge {
    font-size: 10px;
    padding: 0 4px;
    height: 18px;
    line-height: 18px;
  }

  .domain-extra {
    font-size: 11px;
    padding: 1px 5px;
    border-radius: 3px;
    background: rgba(34, 211, 238, 0.12);
    color: var(--xp-accent);
  }
}

.form-tip {
  font-size: 12px;
  color: var(--xp-text-muted);
  margin-top: 4px;
}
</style>
