import Vue from 'vue'
import App from './App.vue'
import BaiduMap from 'vue-baidu-map'
import ViewUI from 'view-design';
import 'view-design/dist/styles/iview.css';

Vue.config.productionTip = false

Vue.use(ViewUI);

Vue.use(BaiduMap, {
  // ak 是在百度地图开发者平台申请的密钥 详见 http://lbsyun.baidu.com/apiconsole/key */
  ak: 'UuWGDj774aFbBhXgtSFMgvKISROhfcAy'
})

new Vue({
  render: h => h(App),
}).$mount('#app')
