<script setup lang="ts">

import {InboundReceiptStatus, InboundReceiptTypes} from "@/enums/inbound.ts";
import SupplierPageItem from "@/components/Supplier/SupplierPageItem.vue";
import {onMounted, ref} from "vue";
import {InboundReceipt, InboundReceiptsRequest} from "@/api/inbound/types.ts";
import {reqInboundReceipts, reqRemoveInboundReceipt} from "@/api/inbound";
import {ElMessage} from "element-plus";
import {Sizes, Types} from "@/utils/enum.ts";
import Item from "@/views/inbound/receipt/components/Item.vue";
import Status from "./components/Status.vue";
import Inbound from "./components/Inbound.vue";

const initInboundReceiptsRequest = () => {
  return <InboundReceiptsRequest>{
    page: 1,
    size: 10,
    status: '',
    type: '',
    code: '',
    supplier_id: '',
    customer_id: '',
  }
}

//初始化入库单
let initReceipt = () => {
  return <InboundReceipt>{
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


const form = ref<InboundReceiptsRequest>(initInboundReceiptsRequest())
const total = ref<number>(0)
const loading = ref<boolean>(false)
let receipts = ref<InboundReceipt[]>([])
let receipt = ref<InboundReceipt>()
let getReceipts = async () => {
  let res = await reqInboundReceipts(form.value)
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
  form.value = initInboundReceiptsRequest()
  await getReceipts()
}

//dialog
let visible = ref<boolean>(false)
let title = ref<string>('')
const action = ref<string>('')

//创建入库单
let add = async () => {
  title.value = '创建入库单'
  visible.value = true
  action.value = 'add'
  receipt.value = initReceipt()
}

//修改入库单
let edit = async (item: InboundReceipt) => {
  title.value = `编辑入库单[${item.code}]`
  visible.value = true
  action.value = 'edit'
  receipt.value = item
}

//审核入库单
let check = async (item: InboundReceipt) => {
  title.value = `审核入库单`
  visible.value = true
  action.value = 'check'
  receipt.value = item
}

//入库
let inbound = async (item: InboundReceipt) => {
  console.log('发货入库界面：', item)
  title.value = `入库`
  visible.value = true
  action.value = 'inbound'
  receipt.value = item
}


//表单提交成功
const handleSuccess = async () => {
  await getReceipts()
  visible.value = false
}

//删除入库单
let remove = async (item: InboundReceipt) => {
  let res = await reqRemoveInboundReceipt(item.id)
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
            label="入库状态"
        >
          <el-radio-group v-model="form.status">
            <el-radio-button plain lable="">全部</el-radio-button>
            <el-radio-button v-for="(item, idx) in InboundReceiptStatus" :key="idx" plain :label="item.label"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item
            label="入库类型"
        >
          <el-radio-group v-model="form.type">
            <el-radio-button plain label="">全部</el-radio-button>
            <el-radio-button v-for="(item, idx) in InboundReceiptTypes" :key="idx" plain :label="item"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item
            label="入库单号"
        >
          <el-input
              v-model="form.code"
              clearable
              placeholder="请填写入库单号"/>
        </el-form-item>
        <SupplierPageItem
            :form="form"
        />
        <el-form-item label=" ">
          <perms-button
              perms="inbound:receipt:list"
              :type="Types.primary"
              :size="Sizes.default"
              :plain="true"
              @click="getReceipts"
          />
          <perms-button
              perms="inbound:receipt:list"
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

    <!-- 入库单列表 -->
    <perms-button
        class="m-t-2"
        perms="inbound:receipt:add"
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
            <el-table
                :data="props.row.materials"
                size="small"
                :border="true"
            >
              <el-table-column label="序号" prop="index" width="80px"/>
              <el-table-column label="物料名称" prop="name"/>
              <el-table-column label="物料规格" prop="model"/>
              <el-table-column label="计划数量" prop="estimated_quantity" align="center"/>
              <el-table-column label="实际数量" prop="actual_quantity" align="center"/>
              <el-table-column label="金额" prop="price" align="center"/>
              <el-table-column label="仓库/库区/货架/货位" width="500px" align="center">
                <template #default="{row}">
                  <span v-if="row.warehouse_id">{{ row.warehouse_name }}</span>
                  <span v-if="row.warehouse_zone_id">/{{ row.warehouse_zone_name }}</span>
                  <span v-if="row.warehouse_rack_id">/{{ row.warehouse_rack_name }}</span>
                  <span v-if="row.warehouse_bin_id">/{{ row.warehouse_bin_name }}</span>
                </template>
              </el-table-column>
              <el-table-column label="入库状态" prop="status" align="center">
                <template #default="{row}">
                  {{ InboundReceiptStatus.find(item => item.value === row.status)?.label }}
                </template>
              </el-table-column>

            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="code" label="入库单编号"/>
      <el-table-column prop="type" label="入库类型" width="110px" align="center"/>
      <el-table-column prop="status" label="入库状态" width="110px" align="center"/>
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
              perms="inbound:receipt:edit"
              :type="Types.primary"
              :size="Sizes.small"
              :plain="true"
              @click="edit(row)"
          />
          <perms-button
              v-if="row.status === '待审核'"
              perms="inbound:receipt:check"
              :type="Types.warning"
              :size="Sizes.small"
              :plain="true"
              @click="check(row)"
          />
          <el-popconfirm
              v-if="['待审核', '审核不通过'].includes(row.status)"
              :title="`确定删除入库单[${row.code}]吗?`"
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
                  perms="inbound:receipt:delete"
                  :type="Types.danger"
                  :size="Sizes.small"
                  :plain="true"/>
            </template>
          </el-popconfirm>
          <perms-button
              v-if="!['待审核', '审核不通过', '入库完成'].includes(row.status)"
              perms="inbound:receipt:material"
              :type="Types.success"
              :size="Sizes.small"
              :plain="true"
              @click="inbound(row)"
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
        :fullscreen="['add', 'edit', 'inbound'].includes(action)"
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
      <Inbound
          v-if="visible&&action === 'inbound'"
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