<script setup lang="ts">
/*修改出库单物料单价*/

import {onMounted, ref} from "vue";
import {OutboundOrderMaterial, OutboundOrderMaterialRevise} from "@/api/outbound/types.ts";
import {reqOutboundOrderMaterials, reqReviseOutboundOrder} from "@/api/outbound";
import {ElMessage} from "element-plus";
import {MaterialPrice} from "@/api/material/types.ts";
import {reqMaterialPrices, reqRemoveMaterialPrice} from "@/api/material";
import NP from "number-precision";
import {DateFormat} from "@/utils/time.ts";
import {nameEncrypt} from "@/utils/name_encrypt.ts";

defineOptions({
  name: 'Revise'
})

let props = defineProps(['code', 'customer_id'])
const emit = defineEmits(['success', 'cancel'])
let loading = ref<boolean>(false)

let materials = ref<OutboundOrderMaterial[]>([])
//查询出库单物料列表
let getOrderMaterials = async (order_code: string) => {
  let res = await reqOutboundOrderMaterials({order_code: order_code})
  if (res.code === 200) {
    materials.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
}

//查询物料价格列表
let prices = ref<MaterialPrice[]>([])
let getPrices = async (material_id: string) => {
  if(props.customer_id===''){
    ElMessage.warning('请选择客户')
    return
  }

  console.log('物料id:', material_id, ',', '客户id:', props.customer_id)

  prices.value = []
  let res = await reqMaterialPrices(material_id,props.customer_id)
  if (res.code === 200) {
    prices.value = res.data?.sort((a:MaterialPrice, b:MaterialPrice) => {
      // 根据需要的排序逻辑进行比较
      return a.since - b.since
    })
  } else {
    ElMessage.error(res.msg)
  }
}

//计算总金额
const total_amount  = ref<number>(0)
let computeTotalAmount = () => {
  total_amount.value = materials.value.reduce((total, current) => {
    return total + NP.times(current.price , current.quantity);
  }, 0)
}


//删除物料价格
let removeMaterialPrice = async (id: string, customer_id:string, price: number) => {
  let res = await reqRemoveMaterialPrice(id, customer_id, price)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getPrices(id)
  } else {
    ElMessage.error(res.msg)
  }
}

//提交物料单价
const submit = async ()=>{
  let req = ref<OutboundOrderMaterialRevise>({
    code: props.code,
    customer_id: props.customer_id,
    materials_price: []
  })

  materials.value.forEach((item:OutboundOrderMaterial)=>{
    req.value.materials_price.push({
      material_id: item.material_id,
      price: item.price,
    })
  })

  loading.value = true
  let res = await reqReviseOutboundOrder(req.value)
  loading.value = false
  if (res.code === 200) {
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(async () => {
  await getOrderMaterials(props.code)
})

</script>

<template>
<div>
  <el-table
      border
      stripe
      :data="materials"
  >
    <template #empty>
      <el-empty/>
    </template>
    <el-table-column label="序号" prop="index" width="80px"/>
    <el-table-column label="物料名称" prop="name"/>
    <el-table-column label="物料规格" prop="model"/>
    <el-table-column label="计划数量" prop="quantity" align="center">
      <template #default="{row}">
        {{row.quantity}}{{ row.unit }}
      </template>
    </el-table-column>

    <el-table-column label="入库价" prop="price">
      <template #default="{row}">
        <el-popover
            placement="right"
            :title="`[${row.name}] 历史价格：`"
            :width="300"
            trigger="hover"
            @beforeEnter="getPrices(row.material_id)"
        >
          <template #reference>
            <el-input-number
                v-model.trim="row.price"
                :controls="false"
                :precision="3"
                :step="1"
                :min="0"
                size="default"
                autocomplete="off"
                @change="computeTotalAmount"/>
          </template>
          <el-tag
              v-if="prices?.length>0"
              v-for="(one, idx) in prices"
              :key="idx"
              class="m-x-1"
              size="default"
              closable
              @click="()=>{row.price=one.price;computeTotalAmount()}"
              @close="removeMaterialPrice(row.material_id, props.customer_id, one.price)"
          >
            {{ one.price }}({{nameEncrypt(one.customer_name)}})[{{DateFormat(one.since)}}]
          </el-tag>
          <el-text
              v-else
              size="small"
          >暂无
          </el-text>
        </el-popover>
      </template>
    </el-table-column>
    <el-table-column label="金额">
      <template #default="{row}">
        {{ NP.times(row.price, row.quantity) }}
      </template>
    </el-table-column>
  </el-table>
  <el-row class="m-t-2">
    <el-col :offset="20" :span="4">
      <el-button size="default" @click="emit('cancel')">取消</el-button>
      <el-button size="default" plain type="primary" :disabled="loading" @click="submit">确定</el-button>
    </el-col>
  </el-row>
</div>
</template>

<style scoped lang="scss">

</style>