<template>
  <div class="login-container">
    <div class="login-bg-grid"></div>
    <div class="login-bg-glow"></div>

    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <el-icon :size="32"><Monitor /></el-icon>
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
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { initUser } from '@/api/modules/auth'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
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
  background: #050810;
  overflow: hidden;
}

.login-bg-grid {
  position: fixed;
  inset: 0;
  background-image:
    linear-gradient(rgba(34, 211, 238, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(34, 211, 238, 0.03) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: radial-gradient(ellipse 60% 60% at 50% 50%, black 20%, transparent 70%);
}

.login-bg-glow {
  position: fixed;
  inset: 0;
  background:
    radial-gradient(ellipse 40% 50% at 25% 50%, rgba(34, 211, 238, 0.06) 0%, transparent 70%),
    radial-gradient(ellipse 40% 50% at 75% 40%, rgba(129, 140, 248, 0.05) 0%, transparent 70%);
}

.login-card {
  position: relative;
  width: 440px;
  padding: 40px 36px;
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(24px);
  border: 1px solid rgba(34, 211, 238, 0.12);
  border-radius: 20px;
  box-shadow: 0 0 60px rgba(34, 211, 238, 0.06), 0 24px 48px rgba(0, 0, 0, 0.4);
}

.login-header {
  text-align: center;
  margin-bottom: 28px;

  .login-logo {
    width: 56px;
    height: 56px;
    margin: 0 auto 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, rgba(34, 211, 238, 0.15), rgba(129, 140, 248, 0.15));
    border: 1px solid rgba(34, 211, 238, 0.2);
    border-radius: 16px;
    color: var(--xp-accent);
  }

  .login-title {
    font-size: 26px;
    font-weight: 700;
    color: #f1f5f9;
    margin: 0 0 6px;
  }

  .login-desc {
    color: #64748b;
    font-size: 14px;
    margin: 0;
  }
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
  border-radius: var(--xp-radius);
  background: linear-gradient(135deg, #0891b2, #06b6d4);
  border: none;
}

:deep(.el-input__wrapper) {
  border-radius: var(--xp-radius) !important;
  height: 44px;
}

:deep(.el-form-item__label) {
  color: var(--xp-text-secondary);
}
</style>
