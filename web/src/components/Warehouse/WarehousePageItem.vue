<script setup lang="ts">
//带分页的仓库下拉菜单
defineOptions({
  name: 'WarehousePageItem'
})
//仓库列表
import {onMounted, ref} from "vue";
import {Warehouse, WarehousesRequest} from "@/api/warehouse/types.ts";
import {reqWarehouses} from "@/api/warehouse";
import {ElMessage} from "element-plus";

defineProps(['form'])

const initWarehousesForm = () => {
  return {
    page: 1,
    size: 10,
  }
}
let warehouses = ref<Warehouse[]>([])
let warehousesForm = ref<WarehousesRequest>(initWarehousesForm())
let warehousesTotal = ref<number>(0)

//查询仓库列表
const getWarehouses = async () => {
  let res = await reqWarehouses(warehousesForm.value)
  if (res.code === 200) {
    warehouses.value = res.data.list
    warehousesTotal.value = res.data.total
  } else {
    warehouses.value = []
    warehousesTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getWarehouses()
})
</script>

<template>
  <el-form-item label="仓库列表" prop="warehouse_id">
    <el-select v-model="form.warehouse_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="warehousesForm.size"
          v-model:current-page="warehousesForm.page"
          :total="warehousesTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getWarehouses"
      />
      <el-option v-for="(one,idx) in warehouses"
                 :label="`${warehousesForm.size * (warehousesForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>