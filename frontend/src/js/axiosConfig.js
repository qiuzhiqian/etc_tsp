import axios from 'axios';
import VueAxios from 'vue-axios'
import router from './router';

// axios 配置
axios.defaults.timeout = 8000;

export default {
    install(Vue) {
        Vue.use(VueAxios, axios)
        // http request 拦截器
        axios.interceptors.request.use(
            config => {
                if (sessionStorage.token) { //判断token是否存在
                    config.headers.Authorization = sessionStorage.token;  //将token设置成请求头
                }
                return config;
            },
            err => {
                return Promise.reject(err);
            }
        );

        // http response 拦截器
        axios.interceptors.response.use(
            response => {
                if (response.data.errno === 999) {
                    router.replace('/');
                    window.console.log("token过期");
                }
                return response;
            },
            error => {
                return Promise.reject(error);
            }
        );
    }
}