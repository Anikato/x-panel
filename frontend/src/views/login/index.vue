<template>
  <div class="login-container">
    <div class="login-bg-grid"></div>
    <div class="login-bg-glow"></div>

    <div class="login-card">
      <div class="login-header">
        <div class="login-logo">
          <el-icon :size="32"><Monitor /></el-icon>
        </div>
        <h1 class="login-title">{{ panelName }}</h1>
        <p class="login-desc">{{ t('login.title') }}</p>
      </div>
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        size="large"
        @keyup.enter="handleLogin"
      >
        <el-form-item prop="name">
          <el-input
            v-model="form.name"
            :prefix-icon="User"
            :placeholder="t('login.namePlaceholder')"
          />
        </el-form-item>
        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :prefix-icon="Lock"
            :placeholder="t('login.passwordPlaceholder')"
          />
        </el-form-item>
        <el-form-item v-if="needCaptcha" prop="captcha">
          <div class="captcha-row">
            <el-input
              v-model="form.captcha"
              :placeholder="t('login.captchaPlaceholder')"
              class="captcha-input"
            />
            <img
              v-if="captchaImage"
              :src="captchaImage"
              class="captcha-image"
              @click="loadCaptcha"
            />
          </div>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" class="login-btn" :loading="loading" @click="handleLogin">
            {{ t('login.login') }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { login, checkIsInit, getLoginSetting, getCaptcha } from '@/api/modules/auth'
import { useUserStore } from '@/store/modules/user'
import { useGlobalStore } from '@/store/modules/global'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()
const globalStore = useGlobalStore()
const { t } = useI18n()

const formRef = ref<FormInstance>()
const loading = ref(false)
const panelName = ref('X-Panel')

const needCaptcha = ref(false)
const captchaImage = ref('')
const captchaID = ref('')

const form = reactive({ name: '', password: '', captcha: '' })

const rules: FormRules = {
  name: [{ required: true, message: () => t('login.nameRequired'), trigger: 'blur' }],
  password: [{ required: true, message: () => t('login.passwordRequired'), trigger: 'blur' }],
}

onMounted(async () => {
  try {
    const initRes = await checkIsInit()
    if (!initRes.data) { router.push('/init'); return }
    const settingRes = await getLoginSetting()
    if (settingRes.data?.panelName) panelName.value = settingRes.data.panelName
  } catch { /* backend not ready */ }
})

const loadCaptcha = async () => {
  try {
    const res = await getCaptcha()
    captchaID.value = res.data.captchaID
    captchaImage.value = res.data.imageData
  } catch { /* ignore */ }
}

const handleLogin = async () => {
  if (!formRef.value) return
  await formRef.value.validate()
  loading.value = true
  try {
    const payload: Record<string, string> = { name: form.name, password: form.password }
    if (needCaptcha.value) {
      payload.captchaID = captchaID.value
      payload.captcha = form.captcha
    }
    const res = await login(payload)
    if (res.data.needCaptcha && !res.data.token) {
      needCaptcha.value = true
      form.captcha = ''
      await loadCaptcha()
      ElMessage.warning(t('login.captchaRequired'))
      return
    }
    if (!res.data.token) {
      ElMessage.error(t('login.loginFailed'))
      if (needCaptcha.value) {
        form.captcha = ''
        await loadCaptcha()
      }
      return
    }
    userStore.setToken(res.data.token)
    userStore.setName(res.data.name)
    globalStore.setLogin(true)
    ElMessage.success(t('commons.success'))
    router.push((route.query.redirect as string) || '/home')
  } catch {
    if (needCaptcha.value) {
      form.captcha = ''
      await loadCaptcha()
    }
  } finally { loading.value = false }
}
</script>

<style lang="scss" scoped>
.login-container {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--xp-bg-auth);
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
    radial-gradient(ellipse 40% 50% at 75% 40%, rgba(129, 140, 248, 0.05) 0%, transparent 70%);
  animation: glowPulse 10s ease-in-out infinite alternate;
}

@keyframes glowPulse {
  0% { opacity: 0.6; }
  100% { opacity: 1; }
}

.login-card {
  position: relative;
  width: 400px;
  padding: 44px 36px;
  background: rgba(17, 24, 39, 0.7);
  backdrop-filter: blur(24px);
  border: 1px solid var(--xp-accent-muted);
  border-radius: 20px;
  box-shadow:
    var(--xp-accent-glow),
    0 24px 48px rgba(0, 0, 0, 0.4);
}

.login-header {
  text-align: center;
  margin-bottom: 36px;

  .login-logo {
    width: 56px;
    height: 56px;
    margin: 0 auto 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, var(--xp-accent-muted), rgba(129, 140, 248, 0.15));
    border: 1px solid var(--xp-accent-muted);
    border-radius: 16px;
    color: var(--xp-accent);
  }

  .login-title {
    font-size: 26px;
    font-weight: 700;
    color: var(--xp-text-primary);
    margin: 0 0 6px;
    letter-spacing: -0.5px;
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
  border-radius: var(--xp-radius);
  background: var(--xp-btn-primary-gradient);
  border: none;
  letter-spacing: 0.5px;
  transition: all 0.3s;

  &:hover {
    background: var(--xp-btn-primary-gradient-hover);
    box-shadow: var(--xp-accent-glow);
  }
}

.captcha-row {
  display: flex;
  width: 100%;
  gap: 12px;
  align-items: center;

  .captcha-input {
    flex: 1;
  }

  .captcha-image {
    height: 44px;
    border-radius: var(--xp-radius);
    cursor: pointer;
    border: 1px solid var(--xp-accent-muted);
  }
}

:deep(.el-input__wrapper) {
  border-radius: var(--xp-radius) !important;
  height: 44px;
}
</style>
