import { createI18n } from 'vue-i18n'
import type { App } from 'vue'
import zh from './zh'

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
