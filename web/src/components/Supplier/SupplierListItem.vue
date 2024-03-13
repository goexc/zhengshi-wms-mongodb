<script setup lang="ts">
//不带带分页的供应商下拉菜单
defineOptions({
  name: 'SupplierListItem'
})
//供应商列表
import {onMounted, ref} from "vue";
import {Supplier, SuppliersRequest} from "@/api/supplier/types.ts";
import {reqSupplierList} from "@/api/supplier";
import {ElMessage} from "element-plus";

defineProps(['form'])

const initSuppliersForm = () => {
  return {
    page: 1,
    size: 10,
    name: '',
    code: '',
    manager: '',
    contact: '',
    email: '',
    level: 0,
  }
}
let suppliers = ref<Supplier[]>([])
let suppliersForm = ref<SuppliersRequest>(initSuppliersForm())

//查询供应商列表
const getSuppliers = async () => {
  // let res = await reqSupplierList(suppliersForm.value)
  let res = await reqSupplierList()
  if (res.code === 200) {
    suppliers.value = res.data.list
  } else {
    suppliers.value = []
    ElMessage.error(res.msg)
  }
}

onMounted(() => {
  getSuppliers()
})
</script>

<template>
  <el-form-item label="供应商" prop="supplier_id">
    <el-select v-model.trim="form.supplier_id" autocomplete="off" clearable>
      <el-option v-for="(one,idx) in suppliers"
                 :label="`${suppliersForm.size * (suppliersForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>