<template>
  <div class="header">
    <div class="header-left">
      <div class="collapse-btn" @click="globalStore.toggleMenuCollapse">
        <el-icon :size="18">
          <Fold v-if="!globalStore.menuCollapse" />
          <Expand v-else />
        </el-icon>
      </div>
      <el-breadcrumb separator="/">
        <el-breadcrumb-item v-for="item in breadcrumbs" :key="item.path">
          {{ item.title }}
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>
    <div class="header-right">
      <el-select
        v-model="currentNode"
        size="small"
        style="width: 160px; margin-right: 12px"
        @change="onNodeChange"
      >
        <el-option :label="t('node.local')" :value="0" />
        <el-option v-for="n in nodes" :key="n.id" :label="n.name" :value="n.id" />
      </el-select>
      <el-popover trigger="click" :width="180" placement="bottom-end">
        <template #reference>
          <div class="accent-btn" :title="t('setting.accentColor')">
            <div class="accent-dot" :style="{ background: currentAccent }"></div>
          </div>
        </template>
        <div class="accent-picker">
          <span class="accent-picker-title">{{ t('setting.accentColor') }}</span>
          <div class="accent-options">
            <div
              v-for="color in accentColors"
              :key="color.key"
              class="accent-option"
              :class="{ active: globalStore.accentColor === color.key }"
              :style="{ background: color.value }"
              @click="globalStore.setAccentColor(color.key as any)"
            ></div>
          </div>
        </div>
      </el-popover>
      <el-dropdown @command="handleCommand" trigger="click">
        <div class="user-dropdown">
          <div class="user-avatar">
            <el-icon :size="14"><UserFilled /></el-icon>
          </div>
          <span class="username">{{ userStore.name || 'admin' }}</span>
          <el-icon :size="12" class="arrow"><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="password">
              <el-icon><Lock /></el-icon>{{ t('header.changePassword') }}
            </el-dropdown-item>
            <el-dropdown-item command="logout" divided>
              <el-icon><SwitchButton /></el-icon>{{ t('header.logout') }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessageBox, ElMessage } from 'element-plus'
import { useGlobalStore } from '@/store/modules/global'
import { useUserStore } from '@/store/modules/user'
import { logout as logoutApi } from '@/api/modules/auth'
import { listNodes } from '@/api/modules/node'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const router = useRouter()
const globalStore = useGlobalStore()
const userStore = useUserStore()
const { t } = useI18n()

const nodes = ref<any[]>([])
const currentNode = ref(globalStore.currentNodeID || 0)

const accentColors = [
  { key: 'cyan', value: '#22d3ee' },
  { key: 'blue', value: '#3b82f6' },
  { key: 'purple', value: '#8b5cf6' },
  { key: 'green', value: '#10b981' },
  { key: 'orange', value: '#f59e0b' },
  { key: 'pink', value: '#ec4899' },
]
const currentAccent = computed(() => {
  return accentColors.find(c => c.key === globalStore.accentColor)?.value || '#22d3ee'
})

const loadNodes = async () => {
  try {
    const res: any = await listNodes()
    nodes.value = res.data || []
  } catch { /* ignore */ }
}

const onNodeChange = (val: number) => {
  const node = nodes.value.find((n: any) => n.id === val)
  globalStore.setCurrentNode(val, node ? node.name : '')
  window.location.reload()
}

onMounted(() => loadNodes())

const breadcrumbs = computed(() => {
  return route.matched
    .filter((item) => item.meta?.title)
    .map((item) => ({
      path: item.path,
      title: t(item.meta.title as string),
    }))
})

const handleCommand = async (command: string) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm(t('header.logoutConfirm'), t('commons.tip'), {
        type: 'warning',
        confirmButtonText: t('commons.confirm'),
        cancelButtonText: t('commons.cancel'),
      })
      await logoutApi()
      userStore.logout()
      globalStore.setLogin(false)
      router.push('/login')
    } catch {
      // cancelled
    }
  } else if (command === 'password') {
    router.push('/setting')
  }
}
</script>

<style lang="scss" scoped>
.header {
  height: var(--xp-header-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  background: var(--xp-bg-header);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--xp-border-light);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;

  .collapse-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--xp-radius-sm);
    color: var(--xp-text-secondary);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
      color: var(--xp-accent);
    }
  }
}

.header-right {
  .user-dropdown {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    padding: 4px 10px;
    border-radius: var(--xp-radius-sm);
    transition: all 0.2s;

    &:hover {
      background: var(--xp-accent-muted);
    }

    .user-avatar {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: linear-gradient(135deg, var(--xp-accent), var(--xp-accent-secondary));
      border-radius: 50%;
      color: #0b0e14;
    }

    .username {
      font-size: 13px;
      color: var(--xp-text-secondary);
      max-width: 100px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .arrow {
      color: var(--xp-text-muted);
    }
  }
}

.accent-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--xp-radius-sm);
  cursor: pointer;
  transition: all 0.2s;
  margin-right: 8px;

  &:hover { background: var(--xp-accent-muted); }

  .accent-dot {
    width: 16px;
    height: 16px;
    border-radius: 50%;
    box-shadow: 0 0 8px rgba(0,0,0,0.3);
  }
}

.accent-picker {
  .accent-picker-title {
    font-size: 12px;
    color: var(--xp-text-muted);
    display: block;
    margin-bottom: 8px;
  }
  .accent-options {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .accent-option {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    cursor: pointer;
    transition: all 0.2s;
    border: 2px solid transparent;

    &:hover { transform: scale(1.15); }
    &.active {
      border-color: var(--xp-text-primary);
      box-shadow: 0 0 0 2px var(--xp-bg-surface), 0 0 12px currentColor;
    }
  }
}
</style>
