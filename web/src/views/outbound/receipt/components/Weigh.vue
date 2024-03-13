<script setup lang="ts">

import {onMounted, ref} from "vue";
import {OutboundOrderMaterial, OutboundOrderWeighRequest} from "@/api/outbound/types.ts";
import {reqOutboundOrderMaterials, reqWeighOutboundOrder} from "@/api/outbound";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import dayjs from "dayjs";

defineOptions({
  'name': 'Weigh'
})
let props = defineProps(['code'])
let emit = defineEmits(['success'])


let materials = ref<OutboundOrderMaterial[]>([])
//查询发货单物料列表
let getMaterials = async (order_code: string) => {
  let res = await reqOutboundOrderMaterials({order_code: order_code})
  if (res.code === 200) {
    materials.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
}

let form = ref<OutboundOrderWeighRequest>({code: props.code, weighing_time: 0, materials: []})

let formRef = ref<FormInstance>()

let rules = ref<FormRules>({
  weighing_time: [
    {
      required: true,
      message: "必填",
      trigger: ["blur", "change"],
    },
    {
      message: '请选择称重日期',
      type: "number",
      min: 1,
      trigger: ['blur', 'change']
    },
    {
      message: '称重日期不能超过当前时间',
      type: "number",
      max: dayjs().unix(),
      trigger: ['blur', 'change']
    }
  ],
  weight: [
      // {required: true, message: '请输入重量', trigger: ['blur','change']},
      {type: 'number', min: 0, message: '请输入≥0的数字', trigger: ['blur','change']},
  ]
})

//提交
let handleSubmit =async () => {
  //1.表单校验
  let valid = await formRef.value?.validate((valid, fields) => {
    if (valid) {

    } else {
      console.log('fields:', fields)
    }
    return
  })

  console.log('valid:', valid)
  if (!valid) {
    return
  }

  materials.value.forEach(item => {
    form.value.materials.push({
      material_id: item.material_id,
      weight: item.weight
    })
  })

  //确认称重,发送请求确认
  let res = await reqWeighOutboundOrder(form.value)
  if(res.code === 200){
    emit('success')
  }else{
    ElMessage.error(res.msg)
  }

}


onMounted(async ()=>{
  await getMaterials(props.code)
})
</script>

<template>
  <div>
    <el-form
    inline
    ref="formRef"
    :rules="rules"
    :model="form"
    size="default"
    label-width="80px"
    >
      <el-form-item label="称重日期" prop="weighing_time">
        <el-date-picker
            v-model.number="form.weighing_time"
            type="date"
            placeholder="请选择称重日期"
            size="default"
            value-format="X"
        />
      </el-form-item>
      <div v-for="(material, idx) in materials" :key="idx">
        <el-form-item label="物料名称">
          <el-input disabled :placeholder="material.name"/>
        </el-form-item>
        <el-form-item label="物料编码">
          <el-input disabled :placeholder="material.model"/>
        </el-form-item>
        <el-form-item label="数量">
          <el-input disabled :placeholder="material.quantity.toString()">
            <template #append>{{ material.unit }}</template>
          </el-input>
        </el-form-item>
        <el-form-item label="重量(kg)" :prop="`materials[${idx}].weight`" :rules="rules.weight">
          <el-input-number
              v-model="material.weight"
              :controls="false"
              :min="0"
          >
            <template #append>Kg</template>
          </el-input-number>
        </el-form-item>
      </div>
      <div>
        <el-form-item >
          <el-button @click="handleSubmit">提交</el-button>
        </el-form-item>
      </div>
    </el-form>
  </div>
</template>

<style scoped lang="scss">

</style>