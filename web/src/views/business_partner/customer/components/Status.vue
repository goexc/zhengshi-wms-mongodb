<script setup lang="ts">

import {nextTick, reactive, ref} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {reqChangeCustomerStatus} from "@/api/customer";
import {CustomerStatusRequest} from "@/api/customer/types.ts";
import {CustomerStatus} from "@/enums/customer.ts";
defineOptions({
  name: "Status"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['customer'])
const emit = defineEmits(['success', 'cancel'])

const formRef = ref<FormInstance>()
const form = ref<CustomerStatusRequest>({id: props.customer.id, status: props.customer.status})
const rules = reactive<FormRules>({
  status: [
    {
      required: true,
      message: "请选择指定的客户状态",
      type: "enum",
      enum: CustomerStatus,
      trigger: ["blur", "change"],
    },
  ],
})

//修改客户状态
const handleSubmit = async () => {
  let valid = await formRef.value?.validate((isValid, fields)=>{
    if (isValid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  if(!valid){
    return
  }

  let res = await reqChangeCustomerStatus(form.value)
  if(res.code === 200){
    await nextTick(() => {
      formRef.value?.clearValidate()
    })
    ElMessage.success(res.msg)
    emit('success')
  }else{
    ElMessage.error(res.msg)
  }
}
//关闭表单
const cancel = () => {
  nextTick(() => {
    formRef.value?.clearValidate()
  })
  emit('cancel')
}
</script>

<template>
<el-form
    label-width="100px"
    style="width:360px"
    :model="form"
    ref="formRef"
    :rules="rules"
>

  <el-form-item label="客户状态" prop="status">
    <el-select filterable v-model.trim="form.status" clearable placeholder="请选择客户状态">
      <el-option v-for="(item,idx) in CustomerStatus" :key="idx" :label="`${idx+1}.${item}`" :value="item"></el-option>
    </el-select>
  </el-form-item>
  <el-form-item>
    <el-button plain size="default" @click="cancel">取消</el-button>
    <el-button type="primary" plain size="default" @click="handleSubmit">提交</el-button>
  </el-form-item>
</el-form>
</template>

<style scoped lang="scss">

</style>