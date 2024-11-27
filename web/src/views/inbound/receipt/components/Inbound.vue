<script setup lang="ts">

import {onMounted, reactive, ref} from "vue";
import {WarehouseTree} from "@/api/warehouse/types.ts";
import {reqWarehouseTree} from "@/api/warehouse";
import {InboundMaterial, InboundReceiptReceiveRequest} from "@/api/inbound/types.ts";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import NP from "number-precision";
import {InboundReceiptMaterialStatus} from "@/enums/inbound.ts";
import CarrierPageItem from "@/components/Carrier/CarrierPageItem.vue";
import {reqInboundReceiptReceive} from "@/api/inbound";
import dayjs from "dayjs";

defineOptions({
  name: 'Inbound'
})
const emit = defineEmits(['success', 'cancel'])

let props = defineProps(['form'])

onMounted(async () => {
  //1.获取仓库树
  let res = await reqWarehouseTree()
  if (res.code === 200) {
    warehouses.value = res.data
  } else {
    warehouses.value = []
  }

  //2.初始化采购物料列表
  //排除入库完成、作废的物料
  // let exclude = InboundReceiptMaterialStatus.filter((status) => ['作废', '入库完成'].includes(status.label)).map(status => status.value)
  materials.value = JSON.parse(JSON.stringify(props.form.materials))

  materials.value.forEach(material => {
    material.actual_quantity = 0
  })

  //3.创建批次入库表单
  receive.value.id = receipt.value.id
  receive.value.carrier_id = ''
  receive.value.carrier_cost = 0
  receive.value.other_cost = 0
  receive.value.materials = []
})

//入库单
let receipt = ref(JSON.parse(JSON.stringify(props.form)))
//采购物料列表
let materials = ref<InboundMaterial[]>([])
//批次入库
let receive = ref<InboundReceiptReceiveRequest>({
  id: receipt.value.id,
  code: 'BI-'+ dayjs().format('YYYY-MM-DD-HH-mm-ss-SSS'),
  // receiving_date: dayjs().startOf('day').unix(),
  receiving_date: '',
  carrier_id: '',
  carrier_cost: 0,
  other_cost: 0,
  materials: [],
  remark: ''
})

//仓库树
let warehouses = ref<WarehouseTree[]>()

//计算总金额
let total_amount = ref<number>(0)
let computeTotalAmount = () => {
  total_amount.value = materials.value.reduce((total, current) => {
    return total + NP.times(current.price, current.actual_quantity);
  }, 0)
  total_amount.value = NP.plus(total_amount.value, receive.value.carrier_cost, receive.value.other_cost)
}

//关闭表单
const cancel = () => {
  emit('cancel')
}

let formRef = ref<FormInstance>()
let rules = reactive<FormRules>({
  code: [
    {
      required: true,
      message: "请填写批次入库编号",
      type: "string",
      trigger: ["blur", "change"],
    },
  ],
  receiving_date: [
    {
      required: true,
      message: "请选择批次入库日期",
      type: "integer",
      trigger: ["blur", "change"],
    },
  ]
})

//保存物料状态
let submit = async () => {
  //1.表单校验
  let valid = await formRef.value!.validate()
  if (!valid) {
    return
  }

  //2.入库物料不能全部为0
  let actual_quantity = materials.value.reduce((total, current) => {
    return total + current.actual_quantity;
  }, 0)

  if (actual_quantity <= 0) {
    ElMessage.error('批次入库物料不能全部是 0.')
    return
  }

  //3.入库物料不能全部未发货
  if(materials.value.filter((item)=>item.status === '未发货').length === materials.value.length){
    ElMessage.error('出库状态不能全部是「未发货」.')
    return
  }

  //清空物料列表
  receive.value.materials = []

  //重新写入物料
  materials.value.forEach((item) => {
    receive.value.materials.push({
      id: item.id,
      index: item.index,
      price: item.price,
      actual_quantity: item.actual_quantity,
      position: item.position,
      status: item.status,
    })
  })

  let res = await reqInboundReceiptReceive(receive.value)
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
      label-width="100px"
      size="default"
      :rules="rules"
      ref="formRef"
      :model="receive"
  >
    <el-form-item label="入库单号">
      {{ receipt.code }}
    </el-form-item>
    <el-form-item label="入库状态">
      {{ receipt.status }}
    </el-form-item>
    <el-form-item label="入库类型">
      {{ receipt.type }}
    </el-form-item>
    <el-form-item v-if="['采购入库', '外协入库'].includes(receipt.type)" label="供应商">
      {{ receipt.supplier_name }}
    </el-form-item>
    <el-form-item v-if="['退货入库'].includes(receipt.type)" label="客户">
      {{ receipt.customer_name }}
    </el-form-item>
    <el-form-item label="批次入库编号" prop="code">
      <el-input
          v-model.trim="receive.code"
          class="w300"
      />
    </el-form-item>
    <el-form-item label="批次入库日期" prop="receiving_date">
      <el-date-picker
          v-model.number="receive.receiving_date"
          type="date"
          placeholder="请选择当前批次入库日期"
          size="default"
          value-format="X"
      />
    </el-form-item>
    <CarrierPageItem
        :form="receive"
    />
    <el-form-item
        label="运费"
        prop="carrier_cost"
    >
      <el-input-number
          v-model.trim="receive.carrier_cost"
          class="w300"
          :controls="false"
          :precision="3"
          :min="0"
          @change="computeTotalAmount"
      />
    </el-form-item>
    <el-form-item
        label="其他费用"
        prop="other_cost"
    >
      <el-input-number
          v-model.trim="receive.other_cost"
          class="w300"
          :controls="false"
          :step="100"
          :precision="3"
          :min="0"
          @change="computeTotalAmount"
      />
    </el-form-item>
    <el-form-item
        label="批次金额"
        prop="total_amount"
    >
      ￥{{ total_amount }} 元
    </el-form-item>
    <el-form-item label="备注">
      {{ receipt.remark }}
    </el-form-item>
    <el-form-item label="批次入库备注">
      <el-input
          v-model.trim="receive.remark"
          type="textarea"
          rows="3"
          maxlength="125"
          :show-word-limit="true"
          placeholder="请填写备注"/>
    </el-form-item>
    <el-form-item label="采购清单">
      <!--   此处为静态数据，因此使用父组件传递过来的静态数据   -->
      <el-table
          class="table"
          border
          stripe
          size="default"
          :data="form.materials"
      >
        <template #empty>
          <el-empty/>
        </template>
        <el-table-column label="序号" prop="index" width="80px"/>
        <el-table-column label="物料名称" prop="name"/>
        <el-table-column label="物料规格" prop="model"/>
        <el-table-column label="计划数量" prop="estimated_quantity" align="center">
          <template #default="{row}">
            <el-text type="danger">{{ row.estimated_quantity }}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="入库数量" prop="estimated_quantity" align="center">
          <template #default="{row}">
            <el-text type="primary">{{ row.actual_quantity }}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="单价" prop="price"/>
        <el-table-column label="预计金额">
          <template #default="{row}">
            {{ NP.times(row.price, row.estimated_quantity) }}
          </template>
        </el-table-column>
        <el-table-column label="支出金额">
          <template #default="{row}">
            {{ NP.times(row.price, row.actual_quantity) }}
          </template>
        </el-table-column>

      </el-table>
    </el-form-item>
  </el-form>
  <el-divider>
    <el-text type="primary" size="default">新增批次入库</el-text>
  </el-divider>
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
    <el-table-column label="本批次入库数量" prop="actual_quantity" align="center">
<!--      <template #default="{row, col, $index}">-->
      <template #default="{row, col, $index}">
        <el-text :hidden="true">{{col}}</el-text>
        <el-input-number
            v-model.trim="row.actual_quantity"
            :key="$index"
            :disabled="['作废'].includes(row.status)"
            :controls="false"
            :precision="3"
            :min="0"
            :value-on-clear="1"
            :step="1"
            size="default"
            @change="computeTotalAmount"
        />
        <el-text type="primary" @click="row.actual_quantity=row.estimated_quantity" size="small">全部</el-text>
      </template>
    </el-table-column>
    <el-table-column label="单价" prop="price" align="center" width="200"/>
    <el-table-column label="金额" width="120">
      <template #default="{row}">
        {{ NP.times(row.price, row.actual_quantity) }}
      </template>
    </el-table-column>
    <el-table-column label="仓库/库区/货架/货位" width="500px" align="center">
      <template #default="{row, col, $index}">
        <el-text :hidden="true">{{col}}</el-text>
        <el-cascader
            size="default"
            :key="$index"
            :options="warehouses"
            :props="{children:'children', label:'name', value: 'id', checkStrictly: true}"
            v-model.trim="row.position"
            clearable
            style="width: 450px"
            placeholder="请选择仓储位置"
        >

        </el-cascader>
      </template>
    </el-table-column>
    <el-table-column label="入库状态" prop="status" align="center">
      <template #default="{row, col, $index}">
        <el-text :hidden="true">{{col}}</el-text>
        <el-select filterable
            size="default"
            v-model.trim="row.status"
            :key="$index"
            clearable
            placeholder="请选择入库状态"
        >
<!--          <el-option
              v-for="(item, idx) in InboundReceiptMaterialStatus.filter((one, idx)=> idx>=InboundReceiptMaterialStatus.findIndex((current) => current === form.materials[$index].status))"
              :key="idx"
              :label="item"
              :value="item"
          />-->
          <el-option
              v-for="(item, idx) in InboundReceiptMaterialStatus"
              :key="idx"
              :label="item"
              :value="item"
          />
        </el-select>
      </template>
    </el-table-column>
  </el-table>
  <div style="flex:1;text-align: center">
    <el-button
        plain
        size="default"
        @click="cancel"
    >取消
    </el-button>
    <el-button
        type="primary"
        plain
        size="default"
        @click="submit"
    >保存
    </el-button>
  </div>
</template>

<style scoped lang="scss">
.w300 {
  width: 300px;
}

.table {
  margin: 20px 0;
}
</style>