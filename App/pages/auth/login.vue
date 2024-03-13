<template>
	<view class="content">
		<view class="logo">
			<u-image :src="logo" shape="circle" :lazy-load="true" width="80px" height="80px"></u-image>
		</view>
		<view class="text">
			<u--input v-model="loginForm.name" prefixIcon="account" border="bottom" clearable
				placeholder="请填写账号"></u--input>
		</view>
		<view class="text">
			<u--input v-model="loginForm.password" prefixIcon="lock" type="password" border="bottom" clearable
				placeholder="请填写密码"></u--input>
		</view>
		<view class="btns">
			<u-button type="warning" shape="circle" :plain="false" text="立即登录" :loading="loading" loading-text="正在登录…"
				:disabled="loading" @click="handleLogin"></u-button>
		</view>
	</view>
</template>

<script setup>
	import {
		reactive,
		ref
	} from "vue";
	import useUserStore from '@/store/user.js'


	let userStore = useUserStore()

	//logo图片
	let logo = ref('../../static/logo.png')

	//登录数据
	let loginForm = reactive({
		name: '系统管理员',
		password: '111111',
	})

	//是否显示登录
	let show = ref(true)
	//正在加载
	let loading = ref(false)

	//登录
	let handleLogin = async () => {
		console.log('form:', loginForm)
		if (loginForm.name.length < 1) {
			uni.showToast({
				title: '请填写账号'
			})
			return
		}

		if (loginForm.password.length < 1) {
			uni.showToast({
				title: '请填写密码'
			})
			return
		}

		loading.value = true
		await loginRequest()
		loading.value = false
	}

	//登录请求
	let loginRequest = async () => {
		let res = await uni.$post('/auth/login', {
			name: loginForm.name,
			password: loginForm.password
		})
		if (res.code === 200) {
			userStore.setUser(res.data)
			userStore.user.mobile = res.data.mobile
			userStore.user.token = res.data.token
			userStore.user.exp = res.data.exp



			uni.showToast({
				icon: 'success',
				title: '登录成功'
			})

			uni.switchTab({
				url: '/pages/index/index'
			})
		} else {
			uni.showToast({
				icon: 'error',
				title: res.msg
			})
		}
	}
</script>

<style scoped lang="scss">
	.content {
		width: 100%;
		height: 100vh;
		padding: 100px 10px;
		box-sizing: border-box;


		// background: url("@/static/images/bg-1.png") no-repeat center center;
		// background-size: cover;


		.logo {
			display: flex;
			justify-content: center;

			padding: 20px 0;
		}

		.text {
			padding: 10px 28px;
		}

		.btns {
			padding: 20px 40px;
		}
	}
</style>