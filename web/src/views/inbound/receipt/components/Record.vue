<script setup lang="ts">

//入库记录
import {onMounted, ref} from "vue";
import NP from "number-precision";
import {reqInboundReceiptRecords} from "@/api/inbound";
import {ElMessage} from "element-plus";
import {InboundReceivedRecord} from "@/api/inbound/types.ts";
import dayjs from "dayjs";

defineOptions({
  name: 'Record'
})

let props = defineProps(['form'])
let receipt = ref(JSON.parse(JSON.stringify(props.form)))
let records = ref<InboundReceivedRecord[]>([])

let getRecords = async () => {
  let req = {inbound_receipt_id: receipt.value.id}
  let res = await reqInboundReceiptRecords(req)
  if (res.code === 200) {
    records.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
  console.log('批次入库记录：', res)
}

onMounted(async () => {
  await getRecords()
})

</script>

<template>

  <el-form
      label-width="100px"
      size="default"
  >
    <el-form-item label="入库单号:">
      <b>{{ receipt.code }}</b>
    </el-form-item>
    <el-form-item label="入库状态:">
      <b>{{ receipt.status }}</b>
    </el-form-item>
    <el-form-item label="入库类型:">
      <b>{{ receipt.type }}</b>
    </el-form-item>
    <el-form-item v-if="['采购入库', '外协入库'].includes(receipt.type)" label="供应商:">
      <b>{{ receipt.supplier_name }}</b>
    </el-form-item>
    <el-form-item v-if="['退货入库'].includes(receipt.type)" label="客户:">
      <b>{{ receipt.customer_name }}</b>
    </el-form-item>
    <el-form-item label="备注:">
      <i>{{ receipt.remark }}</i>
    </el-form-item>
    <el-form-item label="采购清单:">
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
        <el-table-column label="实际支出金额">
          <template #default="{row}">
            {{ NP.times(row.price, row.actual_quantity) }}
          </template>
        </el-table-column>

      </el-table>
    </el-form-item>
  </el-form>
  <div
      v-for="(record, idx) in records"
      class="record"
  >
    <div class="title">
      <h3>批次入库记录 ({{idx+1}})</h3>
    </div>
    <!-- 批次入库记录 -->
    <el-form
    >
      <el-form-item label="批次入库编号:">
        <b>{{ record.code }}</b>
      </el-form-item>
      <el-form-item label="批次入库日期:">
        <b>{{ dayjs.unix(record.receiving_date).format('YYYY-MM-DD') }}</b>
      </el-form-item>
      <el-form-item label="承运商:">
        <b>￥{{ record.carrier_name }}</b>
      </el-form-item>
      <el-form-item label="运费:">
        <b>￥{{record.carrier_cost}}</b>
      </el-form-item>
      <el-form-item label="其他费用:">
        <b>￥{{record.other_cost}}</b>
      </el-form-item>
      <el-form-item label="批次入库金额:">
        <b>￥{{record.total_amount}}</b>
      </el-form-item>
      <el-form-item label="备注:">
        <i>{{ record.remark }}</i>
      </el-form-item>
      <el-form-item label="清单:">
        <el-table
            class="table"
            border
            stripe
            size="default"
            :data="record.materials"
        >
          <template #empty>
            <el-empty/>
          </template>
          <el-table-column label="序号" prop="index" width="80px"/>
          <el-table-column label="物料名称" prop="name"/>
          <el-table-column label="物料规格" prop="model"/>
          <el-table-column label="本批次入库数量" prop="actual_quantity" align="center"/>
          <el-table-column label="单价" prop="price" align="center" width="200"/>
          <el-table-column label="金额" width="120">
            <template #default="{row}">
              {{ NP.times(row.price, row.actual_quantity).toFixed(3) }}
            </template>
          </el-table-column>
          <el-table-column label="仓库/库区/货架/货位" width="500px" align="center">
            <template #default="{row}">
              <span v-if="row.warehouse_name.length>0">{{ row.warehouse_name }}</span>
              <span v-if="row.warehouse_zone_name.length>0">/{{ row.warehouse_zone_name }}</span>
              <span v-if="row.warehouse_rack_name.length>0">/{{ row.warehouse_rack_name }}</span>
              <span v-if="row.warehouse_bin_name.length>0">/{{ row.warehouse_bin_name }}</span>
            </template>
          </el-table-column>
          <el-table-column label="出库状态" prop="status" align="center"/>
        </el-table>
      </el-form-item>
    </el-form>
  </div>

</template>

<style scoped lang="scss">
.title{
  text-align: center;
}
.record{
  border: 2px dashed #1e80ff;
  padding: 10px;
  margin-bottom: 10px;
}
</style>