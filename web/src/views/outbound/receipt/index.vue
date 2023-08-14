<script setup lang="ts">

import {OutboundReceiptStatus, OutboundReceiptTypes} from "@/enums/outbound.ts";
import {Sizes, Types} from "@/utils/enum.ts";
import {OutboundReceipt, OutboundReceiptsRequest} from "@/api/outbound/types.ts";
import {onMounted, ref} from "vue";
import {reqOutboundReceipts, reqRemoveOutboundReceipt} from "@/api/outbound";
import {ElMessage} from "element-plus";
import Status from "./components/Status.vue";
import Item from "./components/Item.vue";
import Outbound from "./components/Outbound.vue";



const initOutboundReceiptsRequest = () => {
  return <OutboundReceiptsRequest>{
    page: 1,
    size: 10,
    status: '',
    type: '',
    code: '',
    supplier_id: '',
    customer_id: '',
  }
}

//初始化出库单
let initReceipt = () => {
  return <OutboundReceipt>{
    id: '',
    code: '',
    type: '',
    status: '',
    total_amount: 0,
    supplier_id: '',
    customer_id: '',
    receiving_date: 0,
    materials: [],
    annex: [],
    remark: '',
  }
}


const form = ref<OutboundReceiptsRequest>(initOutboundReceiptsRequest())
const total = ref<number>(0)
const loading = ref<boolean>(false)
let receipts = ref<OutboundReceipt[]>([])
let receipt = ref<OutboundReceipt>()
let getReceipts = async () => {
  let res = await reqOutboundReceipts(form.value)
  if (res.code === 200) {
    receipts.value = res.data.list
    total.value = res.data.total
  } else {
    receipts.value = []
    total.value = 0
    ElMessage.error(res.msg)
  }
}

let reset = async () => {
  form.value = initOutboundReceiptsRequest()
  await getReceipts()
}

//dialog
let visible = ref<boolean>(false)
let title = ref<string>('')
const action = ref<string>('')

//创建出库单
let add = async () => {
  title.value = '创建出库单'
  visible.value = true
  action.value = 'add'
  receipt.value = initReceipt()
}

//修改出库单
let edit = async (item: OutboundReceipt) => {
  title.value = `编辑出库单[${item.code}]`
  visible.value = true
  action.value = 'edit'
  receipt.value = item
}

//审核出库单
let check = async (item: OutboundReceipt) => {
  title.value = `审核出库单`
  visible.value = true
  action.value = 'check'
  receipt.value = item
}

//出库
let outbound = async (item: OutboundReceipt) => {
  console.log('发货出库界面：', item)
  title.value = `出库`
  visible.value = true
  action.value = 'outbound'
  receipt.value = item
}


//表单提交成功
const handleSuccess = async () => {
  await getReceipts()
  visible.value = false
}

//删除出库单
let remove = async (item: OutboundReceipt) => {
  let res = await reqRemoveOutboundReceipt(item.id)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getReceipts()
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(async () => {
  await getReceipts()
})
</script>


<template>
  <div>
    <el-card>
      <el-form
          inline
          size="default"
          style="display: flex; flex-wrap: wrap;"
      >
        <el-form-item
            label="出库状态"
        >
          <el-radio-group v-model="form.status">
            <el-radio-button plain lable="">全部</el-radio-button>
            <el-radio-button v-for="(item, idx) in OutboundReceiptStatus" :key="idx" plain :label="item.label"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item
            label="出库类型"
        >
          <el-radio-group v-model="form.type">
            <el-radio-button plain label="">全部</el-radio-button>
            <el-radio-button v-for="(item, idx) in OutboundReceiptTypes" :key="idx" plain :label="item"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item
            label="出库单号"
        >
          <el-input
              v-model="form.code"
              clearable
              placeholder="请填写出库单号"/>
        </el-form-item>
        <SupplierPageItem
            :form="form"
        />
        <el-form-item label=" ">
          <perms-button
              perms="outbound:receipt:list"
              :type="Types.primary"
              :size="Sizes.default"
              :plain="true"
              @click="getReceipts"
          />
          <perms-button
              perms="outbound:receipt:list"
              :type="Types.empty"
              :size="Sizes.default"
              :plain="true"
              icon="Refresh"
              text="重置"
              @click="reset"
          />
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 出库单列表 -->
    <perms-button
        class="m-t-2"
        perms="outbound:receipt:add"
        :type="Types.success"
        :size="Sizes.default"
        :plain="true"
        @click="add"
    />
    <el-table
        class="table"
        border
        stripe
        :data="receipts"
    >
      <template #empty>
        <el-empty/>
      </template>
      <el-table-column type="expand">
        <template #default="props">
          <div class="m-4">
            <el-text size="small" type="info">物料列表</el-text>
            <el-table :data="props.row.materials" size="small" border>
              <el-table-column label="序号" prop="index" width="80px"/>
              <el-table-column label="物料名称" prop="name"/>
              <el-table-column label="物料规格" prop="model"/>
              <el-table-column label="计划数量" prop="estimated_quantity" align="center"/>
              <el-table-column label="实际数量" prop="actual_quantity" align="center"/>
              <el-table-column label="金额" prop="price" align="center"/>
              <el-table-column label="仓库/库区/货架/货位" width="500px" align="center">
                <template #default="{row}">
                  <span v-if="row.warehouse_id.length>0">{{ row.warehouse_name }}</span>
                  <span v-if="row.warehouse_zone_id.length>0">/{{ row.warehouse_zone_name }}</span>
                  <span v-if="row.warehouse_rack_id.length>0">/{{ row.warehouse_rack_name }}</span>
                  <span v-if="row.warehouse_bin_id.length>0">/{{ row.warehouse_bin_name }}</span>
                </template>
              </el-table-column>
              <el-table-column label="出库状态" prop="status" align="center">
                <template #default="{row}">
                  {{ OutboundReceiptStatus.find(item => item.value === row.status)?.label }}
                </template>
              </el-table-column>

            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="code" label="出库单编号"/>
      <el-table-column prop="type" label="出库类型" width="110px" align="center"/>
      <el-table-column prop="status" label="出库状态" width="110px" align="center"/>
      <el-table-column prop="supplier_name" label="供应商">
        <template #default="{row}">
          <span v-if="row.supplier_name">{{ row.supplier_name }}</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="customer_name" label="客户">
        <template #default="{row}">
          <span v-if="row.customer_name">{{ row.customer_name }}</span>
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column prop="total_amount" label="总金额" width="120px"/>
      <el-table-column prop="remark" label="备注"/>
      <el-table-column label="操作" width="360px">
        <template #default="{row}">
          <perms-button
              v-if="['待审核', '审核不通过'].includes(row.status)"
              perms="outbound:receipt:edit"
              :type="Types.primary"
              :size="Sizes.small"
              :plain="true"
              @click="edit(row)"
          />
          <perms-button
              v-if="row.status === '待审核'"
              perms="outbound:receipt:check"
              :type="Types.warning"
              :size="Sizes.small"
              :plain="true"
              @click="check(row)"
          />
          <el-popconfirm
              v-if="['待审核', '审核不通过'].includes(row.status)"
              :title="`确定删除出库单[${row.code}]吗?`"
              icon="InfoFilled"
              icon-color="#F56C6C"
              cancel-button-text="取消"
              confirm-button-text="确认删除"
              cancel-button-type="info"
              confirm-button-type="danger"
              @confirm="remove(row)"
              width="300"
          >
            <template #reference>
              <perms-button
                  perms="outbound:receipt:delete"
                  :type="Types.danger"
                  :size="Sizes.small"
                  :plain="true"/>
            </template>
          </el-popconfirm>
          <perms-button
              v-if="!['待审核', '审核不通过', '出库完成'].includes(row.status)"
              perms="outbound:receipt:material"
              :type="Types.success"
              :size="Sizes.small"
              :plain="true"
              @click="outbound(row)"
          />
        </template>
      </el-table-column>
    </el-table>
    <!--   分页   -->
    <el-pagination
        v-model:current-page="form.page"
        v-model:page-size="form.size"
        @size-change="getReceipts"
        @current-change="getReceipts"
        :page-sizes="[10, 20, 30, 40]"
        background
        layout="total, sizes, prev, pager, next, ->,jumper"
        :pager-count="9"
        :disabled="loading"
        :hide-on-single-page="false"
        :total="total"
    ></el-pagination>
    <el-dialog
        v-model="visible"
        :title="title"
        draggable
        :fullscreen="['add', 'edit', 'outbound'].includes(action)"
        :close-on-click-modal="false"
        align-center
        width="600px"
    >
      <Item
          v-if="visible&&['add', 'edit'].includes(action)"
          :form="receipt"
          :action="action"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Status
          v-if="visible&&action === 'check'"
          :receipt="receipt"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Outbound
          v-if="visible&&action === 'outbound'"
          :form="receipt"
          @cancel="visible=false"
          @success="handleSuccess"
      />
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.data {
  margin: 20px 0;
}

.table {
  margin: 20px 0;
}

</style>