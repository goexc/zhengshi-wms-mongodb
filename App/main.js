import App from './App'

//导入uviewPlus
import uviewPlus from '@/uni_modules/uview-plus'


import * as Pinia from 'pinia'
import {
	createUnistorage
} from './uni_modules/pinia-plugin-unistorage'

// #ifndef VUE3
import Vue from 'vue'
import './uni.promisify.adaptor'

Vue.config.productionTip = false
App.mpType = 'app'
const app = new Vue({
	...App
})
app.$mount()
// #endif

// #ifdef VUE3
import {
	createSSRApp
} from 'vue'
export function createApp() {
	const app = createSSRApp(App)

	//使用uviewPlus
	app.use(uviewPlus)

	// 状态管理
	const store = Pinia.createPinia()
	// 持久化
	store.use(createUnistorage())
	app.use(store)
	

	return {
		app,
		Pinia, //此处必须将Pinia返回
	}
}
// #endif