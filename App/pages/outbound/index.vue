<template>
	<view class="content">
		<u-grid :border="false" col="2" @click="clickGrid">
			<u-grid-item class="item" v-for="(item,idx) in features" :key="idx" :customStyle="{padding:20+'rpx'}">
				<u-image :src="`${item.icon}`" width="80" height="80" mode="aspectFill"></u-image>
				<text class="name">{{item.title}}</text>
			</u-grid-item>
		</u-grid>
	</view>
</template>



<script setup>
	import {
		ref
	} from "vue";


	//功能列表
	let features = ref([
		{
			title: '出库单',
			icon: '/static/icons/outbound_order.png',
			page: '/pages/outbound/order'
		},
		{
			title: '敬请期待...',
			icon: '/static/icons/stay_tuned.png',
			page: ''
		},
	])

	let clickGrid = (item) => {
		let page = features.value.find((one, idx) => idx === item).page
		if (page && page.length > 0) {
			uni.navigateTo({
				// uni.switchTab({
				url: page
			})
		}
	}
</script>


<style scoped lang="scss">
	.content {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		
		margin-top: 50px;
	}

</style>
