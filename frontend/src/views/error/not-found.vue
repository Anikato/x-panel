<template>
  <div class="xp-empty-page">
    <el-card class="xp-empty-card">
      <div class="xp-empty-code">404</div>
      <h2>{{ t('error.notFoundTitle') }}</h2>
      <p>{{ t('error.notFoundDesc') }}</p>
      <div class="xp-empty-actions">
        <el-button type="primary" @click="router.push('/home')">
          {{ t('error.backHome') }}
        </el-button>
        <el-button @click="router.back()">
          {{ t('error.backPrevious') }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
</script>
