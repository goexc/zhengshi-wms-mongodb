<template>
	<view style="background-color: #f8f8f8;">
		<u-list @scrolltolower="handleOrders">
			<u-list-item v-for="(order, index) in orders" :key="index"
				style="background-color: #fff;margin:5px 10px;border-radius: 30upx;">
				<u-cell :border="false" :title="`${order.code}(${order.customer_name}${order.supplier_name})`">
					<template #title>
						<view class="order">
							<view class="order-top p-b-1">
								<u-text :lines="1" bold :text="`${order.customer_name}${order.supplier_name}`"
									size="17px"></u-text>
								<u-tag :text="order.status" type="success">
									<template #icon>
										<u-icon name="checkmark" color="#ffffff"></u-icon>
									</template>
								</u-tag>
							</view>
							<view class="order-content">
								<view class="order-left">
									<u-text class="p-b-1" type="info" decoration="underline" :text="order.code"
										size="18px"></u-text>
									<u-text class="p-b-1" :text="dateFormat(order.receipt_time)"></u-text>
								</view>
								<view class="order-right">
									<u-text class="p-b-1" type="error" :text="`￥${order.total_amount}`" bold
										size="18px"></u-text>
								</view>
							</view>
							<view class="order-bottom">
								<view>
									<up-button type="primary" plain shape="circle" hairline icon="eye"
									@click="handleMaterials(order.code)">查看出库单物料</up-button>
								</view>
							</view>
						</view>
					</template>
				</u-cell>
			</u-list-item>
		</u-list>
		<u-popup :show="show" mode="left" @close="show=false" :safe-area-inset-top="true" closeable
			close-icon-pos="top-right">
			<u-list>
				<u-list-item v-for="(material, idx) in list" :key="idx" class="m-1">
					<view class="p-b-1" @click="handleViewMaterial(material.model)">
						<u-text class="p-b-1" size="large" bold :text="`[${idx+1}] ${material.model}`"></u-text>
						<u-text class="p-b-1" size="34rpx" type="info" bold :text="`${material.name}`"></u-text>
						<u-text size="28rpx" bold decoration="underline" :text="`${material.specification}`"></u-text>
					</view>
					<view class="flex flex-justify-content-end">
						<view>
							<u-text size="36rpx" 
						:text="`${material.quantity}${material.unit}×￥${material.price}=￥${NP.times(material.quantity,material.price)}`"
						type="error"
						style="float:right"
						></u-text>
						</view>
					</view>
				</u-list-item>
			</u-list>
		</u-popup>
	</view>
</template>

<script setup>
	import {
		onMounted,
		reactive,
		ref
	} from "vue";
	import NP from "number-precision";
	import {
		dateFormat
	} from '@/utils/time.js'

	//表单
	let form = reactive({
		page: 1,
		size: 10,
		// status: globalStatus.value,//当前Tab页状态
		type: '',
		code: '',
		supplier_id: '',
		customer_id: '',
		is_pack: -1,
		is_weigh: -1,
	})


	//出库单列表
	let orders = ref([])
	let total = ref(0)

	//操作菜单
	let show = ref(false)
	let title = ref('')
	let list = ref([])

	//出库单列表
	let handleOrders = async () => {
		let res = await uni.$get('/outbound/page', form)
		if (res.code === 200) {
			orders.value.push(...res.data.list)
			total.value = res.data.total
			if (!!res.data.list) {
				form.page++
			} else {
				uni.showToast({
					title: '到底了',
				})
			}
		} else {
			uni.showToast({
				title: res.msg,
				icon: 'error'
			})
		}
	}



	//查看出库单物料
	let handleMaterials = async (order_code) => {
		console.log('查看出库单物料:', order_code)
		let res = await uni.$get('/outbound/materials', {
			order_code: order_code
		})
		if (res.code === 200) {
			list.value = res.data
			title.value = order_code
			show.value = true
		} else {
			uni.showToast({
				title: res.msg,
				icon: 'error'
			})
		}
	}

	//查看物料规格
	let handleViewMaterial = (model)=>{
		if(!!model&model.length>0){
			uni.navigateTo({
				// uni.switchTab({
				url: '/pages/material/list?model='+`${model}`,
			})
		}
		
	}

	onMounted(() => {
		handleOrders()
	})
</script>

<style scoped lang="scss">
	.order {
		display: flex;
		flex-direction: column;

		.order-top {
			display: flex;
			justify-content: space-between;

		}

		.order-content {

			display: flex;
			justify-content: space-between;
			align-items: center;
		}

		.order-bottom {
			display: flex;
			flex-direction: row;
			justify-content: flex-end;
		}
	}
	
</style>