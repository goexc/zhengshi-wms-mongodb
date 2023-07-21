<script setup lang="ts">
//仓库列表
import {onMounted, ref} from "vue";
import {Warehouse} from "@/api/warehouse/types.ts";
import {reqWarehouseList} from "@/api/warehouse";
import {ElMessage} from "element-plus";
//不带分页的仓库下拉菜单
defineOptions({
  name: 'WarehouseListItem'
})


defineProps(['form'])

let warehouses = ref<Warehouse[]>([])

//查询仓库列表
const getWarehouses = async () => {
  let res = await reqWarehouseList()
  if (res.code === 200) {
    warehouses.value = res.data.list
  } else {
    warehouses.value = []
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
      <el-option v-for="(one,idx) in warehouses"
                 :label="`${ idx+1 }. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>