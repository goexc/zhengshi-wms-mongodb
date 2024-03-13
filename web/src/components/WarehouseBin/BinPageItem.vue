<script setup lang="ts">
//带分页的货位下拉菜单
import {onMounted, ref, watch} from "vue";
import {WarehouseBin, WarehouseBinsRequest} from "@/api/warehouse_bin/types.ts";
import {reqWarehouseBins} from "@/api/warehouse_bin";
import {ElMessage} from "element-plus";

defineOptions({
  name: 'BinPageItem'
})

let props = defineProps(['form'])

//货位列表
const initBinsForm = () => {
  return {
    page: 1,
    size: 10,
    warehouse_id: '',
    warehouse_zone_id: '',
    warehouse_rack_id: '',
  }
}
let bins = ref<WarehouseBin[]>([])
let binsForm = ref<WarehouseBinsRequest>(initBinsForm())
let binsTotal = ref<number>(0)

//查询货位列表
const getBins = async () => {
  binsForm.value.warehouse_id = props.form.warehouse_id
  binsForm.value.warehouse_zone_id = props.form.warehouse_zone_id
  binsForm.value.warehouse_rack_id = props.form.warehouse_rack_id
  let res = await reqWarehouseBins(binsForm.value)
  if (res.code === 200) {
    bins.value = res.data.list
    binsTotal.value = res.data.total
  } else {
    bins.value = []
    binsTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getBins()
})

watch(()=>props.form.warehouse_id, () => {
  props.form.warehouse_bin_id = ''
  getBins()
})
watch(()=>props.form.warehouse_zone_id, () => {
  console.log('切换库区：', props.form.warehouse_zone_id)
  props.form.warehouse_bin_id = ''
  getBins()
})
</script>

<template>
  <el-form-item label="货位列表" prop="warehouse_bin_id">
    <el-select v-model.trim="form.warehouse_bin_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="binsForm.size"
          v-model:current-page="binsForm.page"
          :total="binsTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getBins"
      />
      <el-option v-for="(one,idx) in bins"
                 :label="`${binsForm.size * (binsForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>