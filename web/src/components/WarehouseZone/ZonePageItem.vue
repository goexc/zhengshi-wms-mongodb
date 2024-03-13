<script setup lang="ts">
//带分页的库区下拉菜单
import {onMounted, ref, watch} from "vue";
import {Zone, ZonesRequest} from "@/api/warehouse_zone/types.ts";
import {reqZones} from "@/api/warehouse_zone";
import {ElMessage} from "element-plus";

defineOptions({
  name: 'ZonePageItem'
})

let props = defineProps(['form'])

//库区列表
const initZonesForm = () => {
  return {
    page: 1,
    size: 10,
    warehouse_id: '',
  }
}
let zones = ref<Zone[]>([])
let zonesForm = ref<ZonesRequest>(initZonesForm())
let zonesTotal = ref<number>(0)

//查询库区列表
const getZones = async () => {
  zonesForm.value.warehouse_id = props.form.warehouse_id
  let res = await reqZones(zonesForm.value)
  if (res.code === 200) {
    zones.value = res.data.list
    zonesTotal.value = res.data.total
  } else {
    zones.value = []
    zonesTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getZones()
})

watch(()=>props.form.warehouse_id, () => {
  props.form.warehouse_zone_id = ''
  getZones()
})
</script>

<template>
  <el-form-item label="库区列表" prop="warehouse_zone_id">
    <el-select v-model.trim="form.warehouse_zone_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="zonesForm.size"
          v-model:current-page="zonesForm.page"
          :total="zonesTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getZones"
      />
      <el-option v-for="(one,idx) in zones"
                 :label="`${zonesForm.size * (zonesForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>