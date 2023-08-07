<script setup lang="ts">
//客户列表
import {onMounted, ref} from "vue";
import {Customer, CustomersRequest} from "@/api/customer/types.ts";
import {reqCustomerList} from "@/api/customer";
import {ElMessage} from "element-plus";
//不带带分页的客户下拉菜单
defineOptions({
  name: 'CustomerListItem'
})

defineProps(['form'])

const initCustomersForm = () => {
  return {
    page: 1,
    size: 10,
  }
}
let customers = ref<Customer[]>([])

//查询客户列表
const getCustomers = async () => {
  let res = await reqCustomerList()
  if (res.code === 200) {
    customers.value = res.data.list
  } else {
    customers.value = []
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getCustomers()
})
</script>

<template>
  <el-form-item label="客户" prop="customer_id">
    <el-select v-model="form.customer_id" autocomplete="off" clearable>
      <el-option v-for="(one,idx) in customers"
                 :label="`${idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>