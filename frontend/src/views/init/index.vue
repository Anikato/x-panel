<template>
  <div class="login-container">
    <div class="login-bg-grid"></div>
    <div class="login-bg-glow"></div>
    <button
      type="button"
      class="login-mode-toggle"
      :aria-label="isDark ? t('header.themeLight') : t('header.themeDark')"
      @click="toggleLoginMode"
    >
      <el-icon :size="16">
        <Sunny v-if="isDark" />
        <Moon v-else />
      </el-icon>
      <span>{{ isDark ? t('header.themeLight') : t('header.themeDark') }}</span>
    </button>

    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <XPanelLogo />
        </div>
        <h1 class="login-title">X-Panel</h1>
        <p class="login-desc">{{ t('init.desc') }}</p>
      </div>
      <el-form ref="formRef" :model="form" :rules="rules" size="large" label-position="top">
        <el-form-item :label="t('init.name')" prop="name">
          <el-input v-model="form.name" :prefix-icon="User" />
        </el-form-item>
        <el-form-item :label="t('init.password')" prop="password">
          <el-input v-model="form.password" type="password" show-password :prefix-icon="Lock" />
        </el-form-item>
        <el-form-item :label="t('init.confirmPassword')" prop="confirmPassword">
          <el-input v-model="form.confirmPassword" type="password" show-password :prefix-icon="Lock" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" class="login-btn" :loading="loading" @click="handleInit">
            {{ t('init.submit') }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock, Moon, Sunny } from '@element-plus/icons-vue'
import XPanelLogo from '@/components/brand/XPanelLogo.vue'
import { useAppearanceStore } from '@/store/modules/appearance'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { checkIsInit, initUser } from '@/api/modules/auth'
import { useI18n } from 'vue-i18n'
import { getToken } from '@/utils/auth'

const router = useRouter()
const { t } = useI18n()
const appearanceStore = useAppearanceStore()
const isDark = computed(() => appearanceStore.resolved.colorMode === 'dark')
const toggleLoginMode = () => {
  appearanceStore.applyImmediate({ mode: isDark.value ? 'light' : 'dark' })
}
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({ name: '', password: '', confirmPassword: '' })

const validateConfirm = (_rule: unknown, value: string, callback: (err?: Error) => void) => {
  if (value !== form.password) callback(new Error(t('init.passwordMismatch')))
  else callback()
}

const rules: FormRules = {
  name: [{ required: true, message: () => t('login.nameRequired'), trigger: 'blur' }],
  password: [
    { required: true, message: () => t('login.passwordRequired'), trigger: 'blur' },
    { min: 6, message: () => t('init.passwordMinLength'), trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: () => t('init.confirmRequired'), trigger: 'blur' },
    { validator: validateConfirm, trigger: 'blur' },
  ],
}

onMounted(async () => {
  try {
    const res = await checkIsInit()
    if (res.data) {
      router.replace(getToken() ? '/home' : '/login')
    }
  } catch { /* backend not ready */ }
})

const handleInit = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  loading.value = true
  try {
    await initUser({ name: form.name, password: form.password })
    ElMessage.success(t('init.success'))
    router.push('/login')
  } catch { /* interceptor */ } finally { loading.value = false }
}
</script>

<style lang="scss" scoped>
.login-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: var(--xp-bg-auth);
  overflow: hidden;
}

.login-bg-grid {
  position: fixed;
  inset: 0;
  background-image:
    linear-gradient(var(--xp-accent-muted) 1px, transparent 1px),
    linear-gradient(90deg, var(--xp-accent-muted) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: radial-gradient(ellipse 60% 60% at 50% 50%, black 20%, transparent 70%);
  opacity: 0.3;
}

.login-bg-glow {
  position: fixed;
  inset: 0;
  background:
    radial-gradient(ellipse 40% 50% at 25% 50%, var(--xp-accent-muted) 0%, transparent 70%),
    radial-gradient(ellipse 40% 50% at 75% 40%, var(--xp-accent-muted) 0%, transparent 70%);
}

.login-card {
  position: relative;
  width: 440px;
  padding: 40px 36px;
  background: color-mix(in srgb, var(--xp-bg-surface) 88%, transparent);
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-lg);
  box-shadow: var(--xp-shadow-overlay);
}

.login-header {
  text-align: center;
  margin-bottom: 28px;

  .login-logo {
    width: 64px;
    height: 64px;
    margin: 0 auto 16px;
    padding: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--xp-accent-muted);
    border: 1px solid var(--xp-border);
    border-radius: var(--xp-radius-sm);
  }

  .login-title {
    font-size: 26px;
    font-weight: 700;
    color: var(--xp-text-primary);
    margin: 0 0 6px;
  }

  .login-desc {
    color: var(--xp-text-muted);
    font-size: 14px;
    margin: 0;
  }
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  border-radius: var(--xp-radius-sm);
  background: var(--xp-btn-primary-gradient);
  border: none;
}

.login-mode-toggle {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 12px;
  color: var(--xp-text-primary);
  font-size: 13px;
  background: var(--xp-bg-surface);
  border: 1px solid var(--xp-border);
  border-radius: var(--xp-radius-sm);
  cursor: pointer;

  &:hover {
    border-color: var(--xp-accent);
    color: var(--xp-accent);
  }
}

:deep(.el-input__wrapper) {
  border-radius: var(--xp-radius-sm) !important;
  height: 44px;
}

:deep(.el-form-item__label) {
  color: var(--xp-text-secondary);
}
</style>
