import Vue from 'vue'
import App from './App.vue'
import ViewUI from 'view-design';
import 'view-design/dist/styles/iview.css';
import utils from './js/utils.js'
import router from './js/router.js'

import axiosConfig from './js/axiosConfig.js'
import store from './js/vuex.js'

Vue.config.productionTip = false
Vue.config.devtools = true;

Vue.use(ViewUI);
Vue.use(utils)
Vue.use(axiosConfig)

new Vue({
  router,
  store,
  render: h => h(App),
}).$mount('#app')
