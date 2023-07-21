<script setup lang="ts">
//带分页的库区下拉菜单
import {onMounted, ref, watch} from "vue";
import {Zone, ZonesRequest} from "@/api/warehouse_zone/types.ts";
import {reqZoneList} from "@/api/warehouse_zone";
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

//查询库区列表
const getZones = async () => {
  zonesForm.value.warehouse_id = props.form.warehouse_id
  let res = await reqZoneList(zonesForm.value)
  if (res.code === 200) {
    zones.value = res.data.list
  } else {
    zones.value = []
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getZones()
})

watch(()=>props.form.warehouse_id, () => {
  console.log('切换仓库（zone）：', props.form.warehouse_id)
  props.form.warehouse_zone_id = ''
  getZones()
})
</script>

<template>
  <el-form-item label="库区列表" prop="warehouse_zone_id">
    <el-select v-model="form.warehouse_zone_id" autocomplete="off" clearable>
      <el-option v-for="(one,idx) in zones"
                 :label="`${ idx+1 }. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>