<script setup>
	import {
		onLaunch,
		onShow,
		onHide
	} from '@dcloudio/uni-app'

	import useUserStore from '@/store/user.js'

	// 导入request
	import './utils/request.js'

	let userStore = useUserStore()


	onLaunch(() => {
		// console.log('App onLaunch')
	})
	onShow(() => {
		console.log('App onShow')
		if (userStore.user.exp < (new Date().getTime() / 1000)) {
			console.log('登录状态过期')
			userStore.logout()
			uni.navigateTo({
				url: '/pages/auth/login',
			})
		}
	})
	onHide(() => {
		// console.log('App Hide')
	})
</script>


<style lang="scss">
	// 引入uviewPlus基础样式
	/* 注意要写在第一行，同时给style标签加入lang="scss"属性 */
	@import "@/uni_modules/uview-plus/index.scss";

	//引入模板的全局样式
	@import "@/static/styles/index.css"
</style>