<script setup lang="ts">

import {onMounted, ref} from "vue";
import {WarehouseTree} from "@/api/warehouse/types.ts";
import {reqWarehouseTree} from "@/api/warehouse";
import CustomerListItem from "@/components/Customer/CustomerListItem.vue";
import {InboundReceiptMaterialStatus, InboundReceiptMaterialStatusText} from "@/enums/inbound.ts";
import {InboundReceiptMaterialRequest} from "@/api/inbound/types.ts";
import {reqUpdateInboundReceiptMaterial} from "@/api/inbound";
import {ElMessage} from "element-plus";

defineOptions({
  name: 'Inbound'
})
const emit = defineEmits(['success', 'cancel'])

let props = defineProps(['form'])

let receipt = ref(JSON.parse(JSON.stringify(props.form)))
let materials = ref(JSON.parse(JSON.stringify(props.form.materials)))
//仓库树
let warehouses = ref<WarehouseTree[]>()

//更新物料仓储位置
let changePositions = (value: string[], idx:number) => {
  console.log('级联选择器：', value)
  console.log('index:', idx)

  if(!!value &&value.length>0){
    materials.value[idx].warehouse_id = value[0]
  }else{
    materials.value[idx].warehouse_id = ''

  }

  if(!!value &&value.length>1){
    materials.value[idx].warehouse_zone_id = value[1]
  }else{
    materials.value[idx].warehouse_zone_id = ''

  }

  if(!!value &&value.length>2){
    materials.value[idx].warehouse_rack_id = value[2]
  }else{
    materials.value[idx].warehouse_rack_id = ''

  }

  if(!!value &&value.length>3){
    materials.value[idx].warehouse_bin_id = value[3]
  }else{
    materials.value[idx].warehouse_bin_id = ''
  }
}

//计算总金额
let computeTotalAmount = ()=>{
  receipt.value.total_amount =  materials.value.reduce((total, current)=>{
    return total+current.price;
  }, 0)
}

//关闭表单
const cancel = () => {
  emit('cancel')
}

//保存物料状态
let submit = async () => {
  let req:InboundReceiptMaterialRequest = {
    id: receipt.value.id,
    total_amount: receipt.value.total_amount,
    materials: []
  }

  materials.value.forEach(item=>{
    req.materials.push({
      id: item.id,
      status: InboundReceiptMaterialStatusText[item.status]??'',
      actual_quantity: item.actual_quantity,
    })
  })

  console.log('物料信息：', req)
  let res =await reqUpdateInboundReceiptMaterial(req)
   if(res.code === 200){
     ElMessage.success(res.msg)
     emit('success')
   }else{
     ElMessage.error(res.msg)
   }
}

onMounted(async ()=>{
  //1.获取仓库树
  let res = await reqWarehouseTree()
  console.log(res)
  if (res.code === 200) {
    warehouses.value = res.data
  } else {
    warehouses.value = []
  }


  //2.整理物料对应的仓储信息
  materials.value.forEach(material => {
    material.positions = []

    if(material.warehouse_id){
      material.positions.push(material.warehouse_id)
    }

    if(material.warehouse_zone_id){
      material.positions.push(material.warehouse_zone_id)
    }

    if(material.warehouse_rack_id){
      material.positions.push(material.warehouse_rack_id)
    }

    if(material.warehouse_bin_id){
      material.positions.push(material.warehouse_bin_id)
    }

  })
})
</script>

<template>
  <el-form
    label-width="100px"
    size="default"
    >
    <el-form-item label="入库单号">
      {{receipt.code}}
    </el-form-item>
    <el-form-item label="入库状态">
      {{receipt.status}}
    </el-form-item>
    <el-form-item label="入库类型">
      {{receipt.type}}
    </el-form-item>
    <el-form-item v-if="['采购入库', '外协入库'].includes(receipt.type)" label="供应商">
      {{receipt.supplier_name}}
    </el-form-item>
    <el-form-item v-if="['退货入库'].includes(receipt.type)" label="客户">
      {{receipt.customer_name}}
    </el-form-item>
    <el-form-item
        label="总金额"
        prop="total_amount"
    >
      <el-input-number
          v-model="receipt.total_amount"
          class="w300"
          :controls="false"
          :step="100"
          :precision="3"
          :min="0"
      />
    </el-form-item>
    <el-form-item label="备注">
      {{receipt.remark}}
    </el-form-item>
  </el-form>
  <el-divider/>
  <el-table
      class="table"
      border
      stripe
      size="default"
      :data="materials"
    >
    <template #empty>
      <el-empty/>
    </template>
    <el-table-column label="序号" prop="index" width="80px"/>
    <el-table-column label="物料名称" prop="name"/>
    <el-table-column label="物料规格" prop="model"/>
    <el-table-column label="计划数量" prop="estimated_quantity" align="center"/>
    <el-table-column label="实际数量" prop="actual_quantity" align="center">
      <template #default="{row}">
        <el-input-number
            v-model="row.actual_quantity"
            :controls="false"
            :precision="3"
            :min="0"
            :value-on-clear="1"
            :step="1"
            size="default"
        />
      </template>
    </el-table-column>
    <el-table-column label="金额" prop="price" align="center">
      <template #default="{row}">
        <el-input-number
            v-model="row.price"
            :controls="false"
            :precision="3"
            :step="100"
            :value-on-clear="1"
            size="default"
            @change="computeTotalAmount"
        />
      </template>
    </el-table-column>
    <el-table-column label="仓库/库区/货架/货位" width="500px" align="center">
      <template #default="{row, col, $index}">
        <el-cascader
            size="default"
            :key="$index"
            :options="warehouses"
            :props="{children:'children', label:'name', value: 'id', checkStrictly: true}"
            v-model="row.positions"
            clearable
            style="width: 450px"
            @change="changePositions($event, $index)"
            placeholder="请选择仓储位置"
        >

        </el-cascader>
      </template>
    </el-table-column>
    <el-table-column label="入库状态" prop="status" align="center">
      <template #default="{row, col, $index}">
        <el-select
            size="default"
            v-model="row.status"
          clearable
          placeholder="请选择入库状态"
        >
          <el-option v-for="(item, idx) in InboundReceiptMaterialStatus.filter(one=>one.value>=form.materials[$index].status)" :key="idx" :label="item.label" :value="item.value"/>
        </el-select>
      </template>
    </el-table-column>
  </el-table>
  <div style="flex:1;text-align: center">
    <el-button
      plain
      size="default"
      @click="cancel"
    >取消</el-button>
    <el-button
        type="primary"
        plain
        size="default"
        @click="submit"
    >保存</el-button>
  </div>
</template>

<style scoped lang="scss">
.w300{
  width: 300px;
}
.table{
  margin: 20px 0;
}
</style>