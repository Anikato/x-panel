<template>
  <div class="home-page">
    <el-row :gutter="16">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <div class="card-header-title">
                <el-icon><Monitor /></el-icon>
                <span>{{ t('home.systemInfo') }}</span>
              </div>
            </div>
          </template>
          <div class="stat-grid">
            <div class="stat-item">
              <div class="stat-label">{{ t('home.panelVersion') }}</div>
              <div class="stat-value">{{ panelVersion }}</div>
            </div>
            <div class="stat-item">
              <div class="stat-label">{{ t('home.runMode') }}</div>
              <div class="stat-value">{{ t('home.standalone') }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" style="margin-top: 16px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <div class="card-header-title">
                <el-icon><Compass /></el-icon>
                <span>{{ t('home.quickEntry') }}</span>
              </div>
            </div>
          </template>
          <div class="quick-grid">
            <div
              v-for="entry in quickEntries"
              :key="entry.path"
              class="quick-item"
              @click="router.push(entry.path)"
            >
              <div class="quick-icon">
                <el-icon :size="22"><component :is="entry.icon" /></el-icon>
              </div>
              <span class="quick-label">{{ entry.title }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getCurrentVersion } from '@/api/modules/upgrade'

const router = useRouter()
const { t } = useI18n()

const panelVersion = ref('...')

const fetchVersion = async () => {
  try {
    const res: any = await getCurrentVersion()
    if (res.data) {
      panelVersion.value = res.data.version === 'dev' ? 'dev' : res.data.version
    }
  } catch {
    panelVersion.value = '-'
  }
}

onMounted(() => fetchVersion())

const quickEntries = [
  { path: '/host/files', title: t('menu.fileManager'), icon: 'FolderOpened' },
  { path: '/terminal', title: t('menu.terminal'), icon: 'Monitor' },
  { path: '/setting', title: t('menu.setting'), icon: 'Setting' },
  { path: '/log/login', title: t('menu.loginLog'), icon: 'Document' },
  { path: '/log/operation', title: t('menu.operationLog'), icon: 'Notebook' },
]
</script>

<style lang="scss" scoped>
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}

.stat-item {
  padding: 16px;
  background: var(--xp-bg-base);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);

  .stat-label {
    font-size: 12px;
    color: var(--xp-text-muted);
    margin-bottom: 6px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .stat-value {
    font-size: 18px;
    font-weight: 600;
    color: var(--xp-text-primary);
  }
}

.quick-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 12px;
}

.quick-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 20px 16px;
  background: var(--xp-bg-base);
  border: 1px solid var(--xp-border-light);
  border-radius: var(--xp-radius);
  cursor: pointer;
  transition: all 0.25s;

  &:hover {
    border-color: rgba(34, 211, 238, 0.2);
    background: var(--xp-accent-muted);
    transform: translateY(-2px);

    .quick-icon {
      color: var(--xp-accent);
      background: rgba(34, 211, 238, 0.12);
    }
  }

  .quick-icon {
    width: 44px;
    height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.04);
    border-radius: 12px;
    color: var(--xp-text-secondary);
    transition: all 0.25s;
  }

  .quick-label {
    font-size: 13px;
    color: var(--xp-text-secondary);
    font-weight: 500;
  }
}
</style>
