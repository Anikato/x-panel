import { createI18n } from 'vue-i18n'
import type { App } from 'vue'
import zh from './zh'

// X-Panel 当前产品策略为中文单语言；所有用户可见文本仍统一走 i18n key，便于集中维护。
const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  fallbackLocale: 'zh',
  messages: { zh },
})

export function setupI18n(app: App) {
  app.use(i18n)
}

export default i18n
