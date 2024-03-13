<script setup lang="ts">
//带分页的承运商下拉菜单
defineOptions({
  name: 'CarrierPageItem'
})
//承运商列表
import {onMounted, ref} from "vue";
import {Carrier, CarriersRequest} from "@/api/carrier/types.ts";
import {reqCarriers} from "@/api/carrier";
import {ElMessage} from "element-plus";

defineProps(['form'])

const initCarriersForm = () => {
  return {
    page: 1,
    size: 10,
    name: '',
    code: '',
    manager: '',
    contact: '',
    email: '',
    level: 1,
  }
}
let carriers = ref<Carrier[]>([])
let carriersForm = ref<CarriersRequest>(initCarriersForm())
let carriersTotal = ref<number>(0)

//查询承运商列表
const getCarriers = async () => {
  let res = await reqCarriers(carriersForm.value)
  if (res.code === 200) {
    carriers.value = res.data.list
    carriersTotal.value = res.data.total
  } else {
    carriers.value = []
    carriersTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getCarriers()
})
</script>

<template>
  <el-form-item label="承运商" prop="carrier_id">
    <el-select v-model.trim="form.carrier_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="carriersForm.size"
          v-model:current-page="carriersForm.page"
          :total="carriersTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getCarriers"
      />
      <el-option v-for="(one,idx) in carriers"
                 :label="`${carriersForm.size * (carriersForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>