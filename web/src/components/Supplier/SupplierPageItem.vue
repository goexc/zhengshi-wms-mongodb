<script setup lang="ts">
//带分页的供应商下拉菜单
defineOptions({
  name: 'SupplierPageItem'
})
//供应商列表
import {onMounted, ref} from "vue";
import {Supplier, SuppliersRequest} from "@/api/supplier/types.ts";
import {reqSuppliers} from "@/api/supplier";
import {ElMessage} from "element-plus";

defineProps(['form'])

const initSuppliersForm = () => {
  return {
    page: 1,
    size: 10,
  }
}
let suppliers = ref<Supplier[]>([])
let suppliersForm = ref<SuppliersRequest>(initSuppliersForm())
let suppliersTotal = ref<number>(0)

//查询供应商列表
const getSuppliers = async () => {
  let res = await reqSuppliers(suppliersForm.value)
  if (res.code === 200) {
    suppliers.value = res.data.list
    suppliersTotal.value = res.data.total
  } else {
    suppliers.value = []
    suppliersTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getSuppliers()
})
</script>

<template>
  <el-form-item label="供应商" prop="supplier_id">
    <el-select v-model="form.supplier_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="suppliersForm.size"
          v-model:current-page="suppliersForm.page"
          :total="suppliersTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getSuppliers"
      />
      <el-option v-for="(one,idx) in suppliers"
                 :label="`${suppliersForm.size * (suppliersForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>