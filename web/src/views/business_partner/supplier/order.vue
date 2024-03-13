<script setup lang="ts">
import {onMounted} from "vue";

// let initSupplierOrdersRequest = () => {
//   return <SupplierOrdersRequest>{
//     page: 1,
//     size: 15,
//     supplier_id: '',
//     order_code: '',
//     start_date: 0,
//     end_date: 0,
//   }
// }

//表单
// let form = ref<SupplierOrdersRequest>(initSupplierOrdersRequest())


//查询供应商订单
let getOrders = async ()=>{

}

onMounted(async ()=>{
  await getOrders()
})
</script>

<template>
<div>采购订单管理</div>
</template>

<style scoped lang="scss">

</style>