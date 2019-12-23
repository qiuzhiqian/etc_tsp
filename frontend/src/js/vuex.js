import Vue from 'vue';
import Vuex from "vuex"

Vue.use(Vuex)

const store = new Vuex.Store({
    state: {
        count: 0,
        mapKey: ""
    },
    mutations: {
        increment(state) {
            state.count++
        },
        setMapkey(state, val) {
            window.console.log("val:", val)
            state.mapKey = val
        }
    }
})

export default store