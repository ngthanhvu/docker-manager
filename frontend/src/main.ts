import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { initSettings } from './ui/settings'
import { initAuth } from './ui/auth'
import router from './router'

initSettings()
initAuth()
createApp(App).use(router).mount('#app')
