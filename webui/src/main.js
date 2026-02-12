import {createApp, reactive} from 'vue'
import App from './App.vue'
import router from './router'
import api from './services/api.js';
import ErrorMsg from './components/ErrorMsg.vue'
import LoadingSpinner from './components/LoadingSpinner.vue'

import './assets/dashboard.css'
import './assets/main.css'

const app = createApp(App)
app.config.globalProperties.$axios = api;
app.component("ErrorMsg", ErrorMsg);
app.component("LoadingSpinner", LoadingSpinner);
app.use(router)
app.mount('#app')
