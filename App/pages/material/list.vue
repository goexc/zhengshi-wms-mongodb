<template>
	<!-- 建议放在外层 -->
	<u-sticky class="sticky" :offset-top="0" bg-color="#f4f4f4">
		<u-search class="search" v-model.trim="form.model" @change="handleSearch" :show-action="false"
			placeholder="请填写物料型号" :clearabled="true" height="80rpx" :searchIconSize="26" bg-color="white"></u-search>
	</u-sticky>
	<u-list class="list" @scrolltolower="handleScrolltolower" :pagingEnabled="true" :enableBackToTop="true"
		:preLoadScreen="1.5">
		<view class="content">
			<view class="material" v-for="(one, index) in list" :key="index">
				<!-- <up-image v-if="!one.image" class="avatar" :src="material_image" mode="scaleToFill" width="80" hei
					height="80" shape="square" radius="10"></up-image> -->
				<u-album v-if="!one.image" :urls="[material_image]" singleSize="80" :previewFullImage="false"></u-album>

				<u-album v-else-if="one.image.endsWith('.svg')" :urls="[oss_api+one.image]" class="avatar"
					mode="scaleToFill" singleSize="80" shape="square" radius="10"></u-album>
				<u-album v-else class="avatar" :urls="[oss_api+one.image]" mode="scaleToFill" singleSize="80"
					shape="square" radius="10"></u-album>


				<!-- <u-avatar v-else-if="one.image.endsWith('.svg')" class="avatar" :src="`${oss_api+one.image}`"
					mode="scaleToFill" size="80" shape="square" radius="10"></u-avatar>
				<u-avatar v-else class="avatar" :src="`${oss_api+one.image}_148x148`" mode="scaleToFill" size="80"
					shape="square" radius="10"></u-avatar> -->

				<view class="info">
					<view class="name">
						<u-text :text="`${one.model}`" :line="1" bold size="16"></u-text>
						<u-text :text="`${one.name}`" :line="1" bold size="16"></u-text>
					</view>
					<view class="spec"><u-text :text="one.specification" size="14" type="info" bold
							decoration="underline"></u-text></view>
					<view class="price" style="display: flex;justify-content: flex-start;width: 240px;"
						v-for="(price, idx) in one.prices?.sort((a,b)=>b.since-a.since)" :key="idx">
						<!-- <u-text :text="price.price" bold size="20" type="error" mode="price"></u-text> -->
						<u-text :text="`￥${price.price}`" bold size="20" type="error"></u-text>
						<u-text :text="price.since" mode="date" size="12" type="info"></u-text>
						<!-- <u-text :text="price.company_name" size="12" type="info" mode="name" format="encrypt"></u-text> -->
					</view>
				</view>
			</view>
		</view>
	</u-list>
</template>

<script setup>
	import {
		onMounted,
		reactive,
		ref
	} from "vue";
	import {
		onLoad
	} from '@dcloudio/uni-app'
	import {
		oss_api,
		material_image
	} from '@/config/index.js'



	//分页参数
	let form = ref({
		page: 1,
		size: 15,
		name: '',
		category_id: '',
		material: '', //材质
		specification: '', //规格
		model: '', //型号
		surface_treatment: '', //表面处理
		strength_grade: '', //强度等级
	})

	//物料列表
	let list = ref([])
	let handleScrolltolower = async () => {
		await materialsPageRequest()
	}

	//搜索框防抖定时器
	let debounceTimer = null

	//搜索
	let handleSearch = async () => {
		!!debounceTimer?clearTimeout(debounceTimer):''

		debounceTimer = setTimeout(async() => {
			form.value.page = 1
			form.value.size = 15
			list.value = []
			await materialsPageRequest()
		}, 500)
	}


	//请求物料分页
	let materialsPageRequest = async () => {
		let res = await uni.$get('/material', form.value)
		if (res.code === 200) {
			if (res.data.list && res.data.list.length > 0) {
				//第一页不用push
				if (form.value.page === 1) {
					list.value = res.data.list
				} else {
					list.value.push(...res.data.list)
				}

				form.value.page++
			} else {
				uni.showToast({
					title: '到底了'
				})
			}
		} else {
			uni.showToast({
				title: res.msg,
				icon: 'error'
			})
		}
	}

	onMounted(async () => {
		// await materialsPageRequest()
	})

	onLoad(async (option) => {
		form.value.model = option.hasOwnProperty('model') ? option.model : ''
		if (form.value.model.length === 0) {
			await materialsPageRequest()
		}
	})
</script>

<style scoped lang="scss">
	.sticky {
		padding: 10px 10px 0 10px;

		.search {}

	}

	.list {

		background-color: #f4f4f4;

		.content {
			height: auto;
			background-color: #f4f4f4;
			margin: 0;
			padding: 0 10px;


			.material {
				display: flex;
				// align-items: center;
				justify-content: flex-start;

				// margin: 0 10px 10px 10px;
				margin: 10px 0;
				padding: 10px;
				// height: 150px;
				border-radius: 10px;
				background-color: #ffffff;

				.avatar {}

				.spec {
					padding: 10px 0;
				}

				.info {
					padding: 0 10px;
				}

			}
		}
	}
</style>