<template>
  <div class="sidebar" :class="{ 'is-collapse': globalStore.menuCollapse }">
    <div class="sidebar-logo">
      <div class="logo-icon">
        <el-icon :size="24"><Monitor /></el-icon>
      </div>
      <transition name="fade-text">
        <span v-if="!globalStore.menuCollapse" class="logo-text">
          {{ globalStore.panelName }}
        </span>
      </transition>
    </div>

    <el-scrollbar class="sidebar-menu-scroll">
      <el-menu
        :default-active="activeMenu"
        :collapse="globalStore.menuCollapse"
        :collapse-transition="false"
        router
      >
        <template v-for="item in menuList" :key="item.path">
          <el-menu-item v-if="!item.children" :index="item.path">
            <el-icon><component :is="item.icon" /></el-icon>
            <template #title>{{ item.title }}</template>
          </el-menu-item>

          <el-sub-menu v-else :index="item.path">
            <template #title>
              <el-icon><component :is="item.icon" /></el-icon>
              <span>{{ item.title }}</span>
            </template>
            <el-menu-item
              v-for="child in item.children"
              :key="child.path"
              :index="child.path"
            >
              {{ child.title }}
            </el-menu-item>
          </el-sub-menu>
        </template>
      </el-menu>
    </el-scrollbar>

    <div class="sidebar-footer">
      <div class="sidebar-version" v-if="!globalStore.menuCollapse">
        {{ globalStore.version || '...' }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useGlobalStore } from '@/store/modules/global'
import { getCurrentVersion } from '@/api/modules/upgrade'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const globalStore = useGlobalStore()
const { t } = useI18n()

const activeMenu = computed(() => route.path)

// 如果 global store 中还没有版本号，就去获取
onMounted(async () => {
  if (!globalStore.version) {
    try {
      const res: any = await getCurrentVersion()
      if (res.data) {
        globalStore.setVersion(res.data.version === 'dev' ? 'dev' : res.data.version)
      }
    } catch { /* ignore */ }
  }
})

const menuList = computed(() => [
  { path: '/home', title: t('menu.home'), icon: 'HomeFilled' },
  {
    path: '/website',
    title: t('menu.website'),
    icon: 'ChromeFilled',
    children: [
      { path: '/website/nginx', title: t('menu.nginx') },
      { path: '/website/ssl', title: t('menu.ssl') },
    ],
  },
  {
    path: '/host',
    title: t('menu.host'),
    icon: 'Platform',
    children: [
      { path: '/host/files', title: t('menu.fileManager') },
      { path: '/host/monitor', title: t('menu.monitor') },
      { path: '/host/firewall', title: t('menu.firewall') },
      { path: '/host/process', title: t('menu.processManage') },
      { path: '/host/ssh', title: t('menu.sshManage') },
      { path: '/host/disk', title: t('menu.diskManage') },
    ],
  },
  { path: '/terminal', title: t('menu.terminal'), icon: 'Monitor' },
  {
    path: '/log',
    title: t('menu.log'),
    icon: 'Document',
    children: [
      { path: '/log/login', title: t('menu.loginLog') },
      { path: '/log/operation', title: t('menu.operationLog') },
    ],
  },
  { path: '/setting', title: t('menu.setting'), icon: 'Setting' },
])
</script>

<style lang="scss" scoped>
.sidebar {
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  width: var(--xp-sidebar-width);
  background: var(--xp-bg-sidebar);
  border-right: 1px solid var(--xp-border-light);
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  z-index: 1001;
  display: flex;
  flex-direction: column;

  &.is-collapse {
    width: var(--xp-sidebar-collapse-width);
  }
}

.sidebar-logo {
  height: var(--xp-header-height);
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 12px;
  border-bottom: 1px solid var(--xp-border-light);
  flex-shrink: 0;

  .logo-icon {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, rgba(34, 211, 238, 0.15), rgba(129, 140, 248, 0.15));
    border-radius: 10px;
    color: var(--xp-accent);
    flex-shrink: 0;
  }

  .logo-text {
    color: var(--xp-text-primary);
    font-size: 17px;
    font-weight: 700;
    white-space: nowrap;
    letter-spacing: -0.3px;
  }
}

.sidebar-menu-scroll {
  flex: 1;
  overflow: hidden;
}

.sidebar-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--xp-border-light);

  .sidebar-version {
    font-size: 11px;
    color: var(--xp-text-muted);
    text-align: center;
    letter-spacing: 0.5px;
  }
}

// Menu 样式
:deep(.el-menu) {
  border-right: none;
  background: transparent;
  padding: 8px;

  .el-menu-item,
  .el-sub-menu__title {
    color: var(--xp-text-secondary);
    border-radius: var(--xp-radius-sm);
    margin: 2px 0;
    height: 42px;
    line-height: 42px;
    font-size: 14px;

    &:hover {
      background: rgba(34, 211, 238, 0.06);
      color: var(--xp-text-primary);
    }

    .el-icon {
      font-size: 18px;
    }
  }

  .el-menu-item.is-active {
    background: linear-gradient(90deg, rgba(34, 211, 238, 0.12), rgba(34, 211, 238, 0.04));
    color: var(--xp-accent);
    font-weight: 500;
    position: relative;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 8px;
      bottom: 8px;
      width: 3px;
      background: var(--xp-accent);
      border-radius: 0 3px 3px 0;
    }
  }

  .el-sub-menu .el-menu-item {
    padding-left: 52px !important;
    font-size: 13px;
    height: 38px;
    line-height: 38px;
  }
}

.fade-text-enter-active,
.fade-text-leave-active {
  transition: opacity 0.2s;
}
.fade-text-enter-from,
.fade-text-leave-to {
  opacity: 0;
}
</style>
