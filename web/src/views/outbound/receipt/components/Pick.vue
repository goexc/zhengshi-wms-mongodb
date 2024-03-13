<script setup lang="ts">
import {reactive, ref} from "vue";
import { OutboundOrderPickRequest} from "@/api/outbound/types.ts";
import {ElMessage, FormRules} from "element-plus";
import {reqPickOutboundOrder} from "@/api/outbound";
import dayjs from "dayjs";

defineOptions({
  name: 'Pick'
})

const props= defineProps(['code'])
const emit = defineEmits(['success', 'cancel'])

const form = ref<OutboundOrderPickRequest>({
  code: props.code,
  picking_time: 0,
})

const formRef = ref()
const rules = reactive<FormRules>({
  picking_time: [
    {
      required: true,
      message: "必填",
      trigger: ["blur", "change"],
    },
    {
      message: '请选择拣货日期',
      type: "number",
      min: 1,
      trigger: ['blur', 'change']
    },
    {
      message: '拣货日期不能超过当前时间',
      type: "number",
      max: dayjs().unix(),
      trigger: ['blur', 'change']
    }
  ]
})


//确认拣货
let pick = async () => {
  //参数校验
  let valid = await formRef.value?.validate((isValid:boolean) => {
    if (!isValid) {
    }
    return
  })

  if (!valid) {
    return
  }

  //确认拣货,发送请求确认
  let res = await reqPickOutboundOrder({code:form.value.code,picking_time: form.value.picking_time})
  if (res.code === 200) {
    ElMessage.success(res.msg)
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}
</script>

<template>
  <el-form
    :model="form"
    ref="formRef"
    :rules="rules"
  >
    <el-form-item label="拣货日期" prop="picking_time">
      <el-date-picker
          v-model.number="form.picking_time"
          type="date"
          placeholder="请选择拣货日期"
          size="default"
          value-format="X"
      />
    </el-form-item>
    <el-form-item>
      <el-button @click="emit('cancel')">取消</el-button>
      <el-button type="primary" @click="pick">确认</el-button>
    </el-form-item>
  </el-form>
</template>

<style scoped lang="scss">

</style>