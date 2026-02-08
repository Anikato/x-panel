import { createApp } from 'vue'
import App from './App.vue'
import router from './routers'
import { setupStore } from './store'
import { setupI18n } from './i18n'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import './assets/styles/index.scss'

const app = createApp(App)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

setupStore(app)
setupI18n(app)
app.use(ElementPlus, { locale: zhCn })
app.use(router)
app.mount('#app')
