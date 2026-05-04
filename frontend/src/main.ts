import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import { initSettings } from './ui/settings'
import router from './router'
import { i18n } from './i18n'
import "flag-icons/css/flag-icons.min.css";

initSettings()
createApp(App).use(router).use(i18n).mount('#app')
