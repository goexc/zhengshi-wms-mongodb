<template>
	<view class="content">

		<view class="header">
			<!-- <u-image :src="userStore.user.avatar"></u-image> -->
			<view class="info">
				<u-avatar src="/static/images/avatar.png" fade-show mode="scaleToFill" size="80"
					shape="circle"></u-avatar>
				<view class="name">
					<u-text :text="userStore.user.name" mode="name" format="encrypt" :line="1" bold size="24"></u-text>
					<u-text class="department" :text="userStore.user.department_name" type="info" :line="1"
						size="15"></u-text>
				</view>
			</view>
			<view class="logout">
				<u-button type="error" plain size="small" icon="../../static/icons/logout.svg"
					@click="handleLogout">退出登录</u-button>
			</view>
		</view>

		<view class="features">
			<u-cell-group :border="false">
				<u-cell size="large" icon="map" title="注册" arrow-direction="right" :border="false" :isLink="true"
					url="/pages/auth/register"></u-cell>
			</u-cell-group>
		</view>
	</view>
</template>


<script setup>
	import {
		onShow
	} from '@dcloudio/uni-app'
	import useUserStore from '@/store/user.js'
	import {
		ref
	} from "vue";

	let userStore = useUserStore()
	let navigationBarHeight = ref(0)
	let height = ref(0)

	//获取导航栏高度
	onShow(() => {
		// let res = uni.getSystemInfoSync()

	})

	//退出登录
	let handleLogout = async () => {
		let res = await uni.$post('/auth/logout', {})
		if (res.code === 200) {
			userStore.logout()
			uni.navigateTo({
				url: '/pages/auth/login'
			})
		} else {
			uni.showToast({
				title: '请重试',
				icon: 'error'
			})
		}
	}
</script>

<style scoped lang="scss">
	.content {
		background-color: #f8f8f8;
		display: flex;
		flex-direction: column;
		// height: calc(100vh - navigationBarHeight);
		height: 100%;
		padding: 10px;
		// flex: 1;

		.header {
			height: 100px;
			// width: 100vw;
			// background-color: green;

			display: flex;
			justify-content: space-between;
			align-items: center;

			.info {
				display: flex;
				justify-content: flex-start;
				align-items: center;

				.name {
					padding: 0 10px;

					.department {
						padding-top: 10px;
					}
				}
			}

			.logout {}

		}

		.features {
			margin: 20px 0;
			padding: 4px;
			border-radius: 10px;
			background-color: #ffffff;
		}

	}
</style>