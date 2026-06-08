import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { initSettings } from './ui/settings'
import router from './router'

initSettings()
createApp(App).use(router).mount('#app')
