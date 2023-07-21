<script setup lang="ts">

import {nextTick, reactive, ref} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {reqChangeRackStatus} from "@/api/warehouse_rack";
import {RackStatusRequest} from "@/api/warehouse_rack/types.ts";
import {RackStatus} from "@/enums/rack.ts";
defineOptions({
  name: "Status"
})

//获取父组件传递过来的全部路由数组
const props = defineProps(['rack'])
const emit = defineEmits(['success', 'cancel'])

const formRef = ref<FormInstance>()
const form = ref<RackStatusRequest>({id: props.rack.id, name: props.rack.name, status: props.rack.status})
const rules = reactive<FormRules>({
  status: [
    {
      required: true,
      message: "请选择指定的货架状态",
      type: "enum",
      enum: RackStatus,
      trigger: ["blur", "change"],
    },
  ],
})

//修改货架状态
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

  let res = await reqChangeRackStatus(form.value)
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

  <el-form-item label="货架状态" prop="status">
    <el-select v-model="form.status" clearable placeholder="请选择货架状态">
      <el-option v-for="(item,idx) in RackStatus" :key="idx" :label="`${idx+1}.${item}`" :value="item"></el-option>
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