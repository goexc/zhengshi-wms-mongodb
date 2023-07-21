<script setup lang="ts">
//带分页的货架下拉菜单
import {onMounted, ref, watch} from "vue";
import {Rack, RacksRequest} from "@/api/warehouse_rack/types.ts";
import {reqRacks} from "@/api/warehouse_rack";
import {ElMessage} from "element-plus";

defineOptions({
  name: 'RackPageItem'
})

let props = defineProps(['form'])

//货架列表
const initRacksForm = () => {
  return {
    page: 1,
    size: 10,
    warehouse_id: '',
    warehouse_zone_id: '',
  }
}
let racks = ref<Rack[]>([])
let racksForm = ref<RacksRequest>(initRacksForm())
let racksTotal = ref<number>(0)

//查询货架列表
const getRacks = async () => {
  racksForm.value.warehouse_id = props.form.warehouse_id
  racksForm.value.warehouse_zone_id = props.form.warehouse_zone_id
  let res = await reqRacks(racksForm.value)
  if (res.code === 200) {
    racks.value = res.data.list
    racksTotal.value = res.data.total
  } else {
    racks.value = []
    racksTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getRacks()
})

watch(()=>props.form.warehouse_id, () => {
  props.form.warehouse_rack_id = ''
  getRacks()
})
watch(()=>props.form.warehouse_zone_id, () => {
  console.log('切换库区：', props.form.warehouse_zone_id)
  props.form.warehouse_rack_id = ''
  getRacks()
})
</script>

<template>
  <el-form-item label="货架列表" prop="warehouse_rack_id">
    <el-select v-model="form.warehouse_rack_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="racksForm.size"
          v-model:current-page="racksForm.page"
          :total="racksTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getRacks"
      />
      <el-option v-for="(one,idx) in racks"
                 :label="`${racksForm.size * (racksForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>