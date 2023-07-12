<script setup lang="ts">
//引入仓库、库区、货架、货位接口方法
import {onMounted} from "vue";
import useCategoryStore from "@/store/module/category";
import {WarehouseRackListRequest} from "@/api/warehouse_rack/types.ts";
import {WarehouseZoneListRequest} from "@/api/warehouse_zone/types.ts";

//场景属性scene:true，查看货位列表；false，添加货位
defineProps(['scene'])

//仓库管理
let categoryStore = useCategoryStore();

//组件挂载完毕
onMounted(() => {
  categoryStore.warehouse_id = ''
  categoryStore.warehouse_zone_id = ''
  categoryStore.warehouse_rack_id = ''
  //获取仓库列表
  getWarehouseList()
})


//获取仓库列表
const getWarehouseList = async () => {
  //数据全部清空
  categoryStore.$reset()
  /*
  //将下级id清空
  categoryStore.warehouse_id = ''
  categoryStore.warehouse_zone_id = ''
  categoryStore.warehouse_rack_id = ''
  //清空下级列表
  categoryStore.warehouses = []
  categoryStore.warehouse_zones = []
  categoryStore.warehouse_racks = []
  categoryStore.warehouse_bins = []
*/
  await categoryStore.getWarehouseList()
}

//仓库变化
const handleWarehouseChange = () => {
  //获取仓库的库区列表
  let req = <WarehouseZoneListRequest>({
    warehouse_id: categoryStore.warehouse_id
  })
  categoryStore.getWarehouseZoneList(req)
  //将下级id清空
  categoryStore.warehouse_zone_id = ''
  categoryStore.warehouse_rack_id = ''
  //清空下级列表
  categoryStore.warehouse_zones = []
  categoryStore.warehouse_racks = []
  categoryStore.warehouse_bins = []
}

//库区变化
const handleWarehouseZoneChange = () => {
  //获取货架列表
  let req = <WarehouseRackListRequest>({
    warehouse_zone_id: categoryStore.warehouse_zone_id
  })
  categoryStore.getWarehouseRackList(req)
  //将下级id清空
  categoryStore.warehouse_rack_id = ''
  //清空下级列表
  categoryStore.warehouse_racks = []
  categoryStore.warehouse_bins = []
}

</script>

<template>
  <el-card>
    <!--  三级组件  -->
    <el-form :inline="true">
      <el-form-item label="仓库">
        <el-select
            v-model="categoryStore.warehouse_id"
            @change="handleWarehouseChange"
            clearable
            :disabled="!scene"
        >
          <el-option v-for="(warehouse, index) in categoryStore.warehouses" :key="index" :label="warehouse.name"
                     :value="warehouse.id"></el-option>
        </el-select>
      </el-form-item>
      <el-form-item label="库区">
        <el-select
            v-model="categoryStore.warehouse_zone_id"
            @change="handleWarehouseZoneChange"
            clearable
            :disabled="!scene"
        >
          <el-option v-for="(warehouse_zone, index) in categoryStore.warehouse_zones" :key="index"
                     :label="warehouse_zone.name" :value="warehouse_zone.id"></el-option>
        </el-select>

      </el-form-item>
      <el-form-item label="货架">
        <el-select
            v-model="categoryStore.warehouse_rack_id"
            clearable
            :disabled="!scene"
        >

          <el-option v-for="(warehouse_rack, index) in categoryStore.warehouse_racks" :key="index"
                     :label="warehouse_rack.name" :value="warehouse_rack.id"></el-option>
        </el-select>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<style scoped>

</style>