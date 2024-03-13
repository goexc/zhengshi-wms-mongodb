<script setup lang="ts">
/*出库操作*/

import CarrierPageItem from "@/components/Carrier/CarrierPageItem.vue";
import {onMounted, reactive, ref} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {OutboundOrderDepartureRequest, OutboundOrderMaterial} from "@/api/outbound/types.ts";
import {reqDepartureOutboundOrder, reqOutboundOrderMaterials} from "@/api/outbound";
import NP from "number-precision";
import dayjs from "dayjs";

defineOptions({
  name: 'Departure'
})
let props = defineProps(['code'])

let emit = defineEmits(['success'])
let rules = reactive<FormRules>({
  departure_time: [
  {
    required: true,
    message: "必填",
    trigger: ["blur", "change"],
  },
  {
    message: '请选择出库时间',
    type: "number",
    min: 1,
    trigger: ['blur', 'change']
  },
  {
    message: '出库时间不能超过当前时间',
    type: "number",
    max: dayjs().unix(),
    trigger: ['blur', 'change']
  }
],
  has_carrier: [
    {required: true, type: 'boolean', message: '请选择是否承运', trigger: ['blur', 'change']},
  ],
  carrier_id: [
    {required: true, message: '请选择承运商', trigger: ['blur', 'change']},
  ],
  carrier_cost: [
    {required: true, type: 'number', message: '请填写运费', trigger: ['blur', 'change']},
    {type: 'number', min: 0, message: '运费必须≥0', trigger: ['blur', 'change']},
  ],
  other_cost: [
    {required: true, type: 'number', message: '请填写其他费用', trigger: ['blur', 'change']},
    {type: 'number', min: 0, message: '其他费用必须≥0', trigger: ['blur', 'change']},
  ],
})
let formRef = ref<FormInstance>()
let form = ref<OutboundOrderDepartureRequest>({
  code: props.code,
  departure_time: 0,
  has_carrier: true,
  carrier_id: '',
  carrier_cost: 0,
  other_cost: 0
})

let materials = ref<OutboundOrderMaterial[]>([])

//查询发货单物料列表
let getMaterials = async (order_code: string) => {
  let res = await reqOutboundOrderMaterials({order_code: order_code})
  if (res.code === 200) {
    materials.value = res.data
    materials.value.every(item => item.returned_quantity = 0)
  } else {
    ElMessage.error(res.msg)
  }
}

//是否承运
let handleChange = () => {
  if(!form.value.has_carrier){
    form.value.carrier_id = ''
    form.value.carrier_cost = 0
  }
}

//提交
let handleSubmit = async () => {
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


  //2.提交
  let res = await reqDepartureOutboundOrder(form.value)
  if (res.code === 200) {
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(async () => {
  await getMaterials(props.code)
})
</script>

<template>
  <div>
    <el-table
        class="m-b-2"
        :data="materials"
        size="small"
        border>
      <el-table-column label="序号" prop="index" width="80px"/>
      <el-table-column label="物料名称" prop="name"/>
      <el-table-column label="物料规格" prop="model"/>
      <el-table-column label="数量" prop="quantity" align="center">
        <template #default="{row}">
          <el-text type="primary" size="small">{{ row.quantity }}</el-text>
        </template>
      </el-table-column>
      <el-table-column label="重量" prop="weight" align="center"/>
      <el-table-column label="单价" prop="price" align="center"/>
      <el-table-column label="金额" align="left">
        <template #default="{row}">
          {{ NP.times(row.price, row.quantity) }}
        </template>
      </el-table-column>
    </el-table>

    <el-form
        size="default"
        label-width="100px"
        ref="formRef"
        :model="form"
        :rules="rules"
    >
      <el-form-item label="出库时间" prop="departure_time">
        <el-date-picker
            v-model.number="form.departure_time"
            type="date"
            placeholder="请选择出库时间"
            size="default"
            value-format="X"
        />
      </el-form-item>
      <el-form-item label="是否承运" prop="has_carrier">
        <el-radio-group
            v-model="form.has_carrier"
            @change="handleChange"
        >
          <el-radio :label="true">是</el-radio>
          <el-radio :label="false">否</el-radio>
        </el-radio-group>
      </el-form-item>
      <CarrierPageItem
          v-if="form.has_carrier"
          :form="form"
      />
      <el-form-item
          v-if="form.has_carrier"
          label="运费"
          prop="carrier_cost"
      >
        <el-input-number
            v-model.trim="form.carrier_cost"
            class="w300"
            :controls="false"
            :precision="3"
            :min="0"
        />
      </el-form-item>
      <el-form-item
          label="其他费用"
          prop="other_cost"
      >
        <el-input-number
            v-model.trim="form.other_cost"
            class="w300"
            :controls="false"
            :step="100"
            :precision="3"
            :min="0"
        />
      </el-form-item>
      <el-form-item>
        <el-button
            plain
            type="primary"
            size="default"
            @click="handleSubmit"
        >出库
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<style scoped lang="scss">

</style>