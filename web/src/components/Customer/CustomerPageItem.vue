<script setup lang="ts">
//带分页的客户下拉菜单
defineOptions({
  name: 'CustomerPageItem'
})
//客户列表
import {onMounted, ref} from "vue";
import {Customer, CustomersRequest} from "@/api/customer/types.ts";
import {reqCustomers} from "@/api/customer";
import {ElMessage} from "element-plus";

defineProps(['form'])

const initCustomersForm = () => {
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
let customers = ref<Customer[]>([])
let customersForm = ref<CustomersRequest>(initCustomersForm())
let customersTotal = ref<number>(0)

//查询客户列表
const getCustomers = async () => {
  let res = await reqCustomers(customersForm.value)
  if (res.code === 200) {
    customers.value = res.data.list
    customersTotal.value = res.data.total
  } else {
    customers.value = []
    customersTotal.value = 0
    ElMessage.error(res.msg)
  }
}

onMounted(()=>{
  getCustomers()
})
</script>

<template>
  <el-form-item label="客户" prop="customer_id">
    <el-select v-model.trim="form.customer_id" autocomplete="off" clearable>
      <el-pagination
          v-model:page-size="customersForm.size"
          v-model:current-page="customersForm.page"
          :total="customersTotal"
          style="width: 100%"
          layout="prev, pager, next"
          @current-change="getCustomers"
      />
      <el-option v-for="(one,idx) in customers"
                 :label="`${customersForm.size * (customersForm.page-1) + idx+1}. ${one.name}`"
                 :value="one.id" :key="idx"/>
    </el-select>
  </el-form-item>
</template>

<style scoped lang="scss">

</style>