<script setup lang="ts">

import {OutboundOrderTypes} from "@/enums/outbound.ts";
import {Sizes, Types} from "@/utils/enum.ts";
import {OutboundOrder, OutboundOrderMaterial, OutboundOrdersRequest} from "@/api/outbound/types.ts";
import {onMounted, ref} from "vue";
import {
  reqOutboundOrderMaterials,
  reqOutboundOrders,
  reqRemoveOutboundOrder,
} from "@/api/outbound";
import {ElMessage} from "element-plus";
import Status from "./components/Status.vue";
import Order from "./components/Order.vue";
import Weigh from "./components/Weigh.vue";
import Receipt from "./components/Receipt.vue";
import SupplierPageItem from "@/components/Supplier/SupplierPageItem.vue";
import CustomerPageItem from "@/components/Customer/CustomerPageItem.vue";
import Confirm from "@/views/outbound/receipt/components/Confirm.vue";
import Departure from "@/views/outbound/receipt/components/Departure.vue";
import NP from "number-precision";
import {DateFormat} from "@/utils/time.ts";
import Pack from "@/views/outbound/receipt/components/Pack.vue";
import Pick from "@/views/outbound/receipt/components/Pick.vue";

//当前Tab页状态
let globalStatus = ref<string>('')

const initOutboundOrdersRequest = () => {
  return <OutboundOrdersRequest>{
    page: 1,
    size: 20,
    status: globalStatus.value,
    type: '',
    code: '',
    supplier_id: '',
    customer_id: '',
    is_pack: -1,
    is_weigh: -1,
  }
}

//初始化发货单
let initReceipt = () => {
  return <OutboundOrder>{
    id: '',
    code: '',
    type: '',
    status: '',
    is_weigh: 0,
    is_pack: 0,
    total_amount: 0,
    supplier_id: '',
    customer_id: '',
    customer_name: '',
    carrier_name: '',
    carrier_cost: 0,
    other_cost: 0,
    date: 0,
    departure_time: 0,
    receipt_time: 0,
    materials: [],
    annex: [],
    receipt: [],
    remark: '',
  }
}


const form = ref<OutboundOrdersRequest>(initOutboundOrdersRequest())
const total = ref<number>(0)
const loading = ref<boolean>(false)
let receipts = ref<OutboundOrder[]>([])
let receipt = ref<OutboundOrder>(initReceipt())
let expand = ref<boolean>(false) //展开表格
let getReceipts = async () => {
  loading.value = true
  expand.value = false
  let res = await reqOutboundOrders(form.value)
  loading.value = false
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
  form.value = initOutboundOrdersRequest()
  await getReceipts()
}

//dialog
let visible = ref<boolean>(false)
let title = ref<string>('')
const action = ref<string>('')
let dialogWidth = ref<string | number>('')

//创建发货单
let add = () => {
  title.value = '创建发货单'
  visible.value = true
  action.value = 'add'
  receipt.value = initReceipt()
}

/*

//审核发货单
let check = (item: OutboundOrder) => {
  title.value = `审核发货单`
  dialogWidth.value = '600px'
  visible.value = true
  action.value = 'check'
  receipt.value = item
}
*/


//确认发货单
let confirm = (item: OutboundOrder) => {
  title.value = `确认发货单`
  dialogWidth.value = '1880px'
  visible.value = true
  action.value = 'confirm'
  receipt.value = item
}

//确认拣货
let pick = async (item: OutboundOrder) => {
  title.value = `确认拣货`
  dialogWidth.value = '680px'
  visible.value = true
  action.value = 'pick'
  receipt.value = item
}

//打包
let pack = async (item: OutboundOrder) => {
  title.value = `确认打包`
  dialogWidth.value = '680px'
  visible.value = true
  action.value = 'pack'
  receipt.value = item
}

//称重，更新物料重量
let weigh = async (item: OutboundOrder) => {
  receipt.value = item
  action.value = 'weigh'
  title.value = `称重`
  dialogWidth.value = '1200px'
  visible.value = true
}

//出库
let handleDeparture = async (item: OutboundOrder) => {
  receipt.value = item
  action.value = 'departure'
  title.value = `出库`
  dialogWidth.value = '1200px'
  visible.value = true

/*
  let result = await ElMessageBox.confirm(
      '此操作不可逆, 确认出库?',
      '提示',
      {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning',
      }
  ).catch((reason) => {
    return reason
  })

  if (result !== 'confirm') {
    ElMessage.info('取消操作')
    return
  }


  let res = await reqDepartureOutboundOrder(item.code)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getReceipts()
  }else {
    ElMessage.error(res.msg)
  }
  */
}

//签收
let handleReceipt = async (item: OutboundOrder) => {
  receipt.value = item
  action.value = 'receipt'
  title.value = `签收`
  dialogWidth.value = '1200px'
  visible.value = true

/*  let result = await ElMessageBox.confirm(
      '确认签收?',
      '提示',
      {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning',
      }
  ).catch((reason) => {
    return reason
  })

  if (result !== 'confirm') {
    ElMessage.info('取消操作')
    return
  }

  let res = await reqReceiptOutboundOrder(item.code)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getReceipts()
  }else {
    ElMessage.error(res.msg)
  }*/
}


//表单提交成功
const handleSuccess = async () => {
  await getReceipts()
  visible.value = false
}

//删除发货单
let remove = async (item: OutboundOrder) => {
  let res = await reqRemoveOutboundOrder(item.id)
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

let tabs = [
  {label: '发货单', status: '', props: {is_pack: -1, is_weigh: -1,}, icon: 'List'},
  {label: '预发货', status: '预发货', props: {is_pack: -1, is_weigh: -1,}, icon: 'List'},
  {label: '待拣货', status: '待拣货', props: {is_pack: -1, is_weigh: -1,}, icon: 'ShoppingCart'},
  {label: '已拣货', status: '已拣货', props: {is_pack: -1, is_weigh: -1,}, icon: 'ShoppingCartFull'},
  {label: '待打包', status: '待打包', props: {is_pack: 0, is_weigh: -1,}, icon: 'Ticket'},
  {label: '已打包', status: '已打包', props: {is_pack: 1, is_weigh: -1,}, icon: 'GoodsFilled'},
  {label: '待称重', status: '待称重', props: {is_pack: -1, is_weigh: 0,}, icon: 'List'},
  {label: '已称重', status: '已称重', props: {is_pack: -1, is_weigh: 1,}, icon: 'List'},
  {label: '待出库', status: '待出库', props: {is_pack: -1, is_weigh: -1,}, icon: 'List'},
  {label: '已出库', status: '已出库', props: {is_pack: -1, is_weigh: -1,}, icon: 'Promotion'},
  {label: '已签收', status: '已签收', props: {is_pack: -1, is_weigh: -1,}, icon: 'SuccessFilled'},
]
//查询发货单
let handleOutboundList = async (status: string, props: { is_pack: number, is_weigh: number }) => {
  globalStatus.value = status
  form.value.status = status
  form.value.is_pack = props.is_pack
  form.value.is_weigh = props.is_weigh
  await getReceipts()
}

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

//展开表格
let handleExpandChange = async (row: OutboundOrder, rows: OutboundOrder[]) => {
  if (rows.length === 0) {
    return
  }

  await getMaterials(row.code)
  row.materials = materials.value
}

//出库状态样式
let orderStatus = (status:string) => {
  switch (status) {
    case '预发货':
      return 'danger'
    case '待拣货':
      return 'danger'
    case '已拣货':
      return ''
    case '已打包':
      return 'info'
    case '已称重':
      return 'info'
    case '已出库':
      return 'warning'
    case '已签收':
      return 'success'
    default:
      return ''
  }
}
</script>


<template>
  <div>

    <el-tabs
        :stretch="true"
    >
      <el-tab-pane
          v-for="(row, idx) in tabs" :key="idx"
      >
        <template #label>
          <div
              style="flex:1;flex-direction: column;height: 46px;text-align: center"
              @click="handleOutboundList(row.status, row.props)"
          >
            <el-icon size="22">
              <component :is="row.icon"></component>
            </el-icon>
            <div>{{ row.label }}</div>
          </div>
        </template>
      </el-tab-pane>
    </el-tabs>

    <el-card>
      <el-form
          inline
          size="default"
          style="display: flex; flex-wrap: wrap;"
      >
        <el-form-item
            label="出库类型"
        >
          <el-radio-group v-model.trim="form.type">
            <el-radio-button plain label="">全部</el-radio-button>
            <el-radio-button v-for="(item, idx) in OutboundOrderTypes" :key="idx" plain :label="item"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item
            label="发货单号"
        >
          <el-input
              v-model.trim="form.code"
              clearable
              placeholder="请填写发货单号"/>
        </el-form-item>
        <SupplierPageItem
            :form="form"
        />
        <CustomerPageItem
            :form="form"/>
        <el-form-item label=" ">
          <perms-button
              perms="outbound:order:list"
              :type="Types.primary"
              :size="Sizes.default"
              :plain="true"
              @click="getReceipts"
          />
          <perms-button
              perms="outbound:order:list"
              :type="Types.empty"
              :size="Sizes.default"
              :plain="true"
              icon="Refresh"
              text="重置"
              @click="reset"
          />
        </el-form-item>
      </el-form>
      <el-form
          v-if="globalStatus===''"
      >
        <el-form-item>
          <!-- 新增发货单 -->
          <perms-button
              perms="outbound:order:add"
              :type="Types.success"
              :size="Sizes.default"
              :plain="true"
              @click="add"
          />
        </el-form-item>
      </el-form>
      <!-- 发货单列表 -->
      <el-table
          v-if="!loading"
          class="table"
          border
          stripe
          size="large"
          :default-expand-all="expand"
          :data="receipts"
          row-key="id"
          @expand-change="handleExpandChange"
      >
        <template #empty>
          <el-empty/>
        </template>
        <el-table-column type="expand">
          <template #default="props">
            <div class="m-4">
              <el-text size="default" type="info">物料列表</el-text>
              <el-table
                  :data="props.row.materials"
                  size="default"
                  border>
                <el-table-column label="序号" prop="index" width="80px"/>
                <el-table-column label="物料名称" prop="name"/>
                <el-table-column label="物料规格" prop="model"/>
                <el-table-column label="数量" prop="quantity" align="center">
                  <template #default="{row}">
                    <el-text type="primary" size="small">{{ row.quantity }}</el-text>
                  </template>
                </el-table-column>
                <el-table-column label="单价" prop="price" align="center"/>
                <el-table-column label="金额" align="center">
                  <template #default="{row}">
                    {{ NP.times(row.price, row.quantity) }}
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="code" label="发货单号" width="200px">
          <template #default="{row}">
          <el-text tag="b" size="default">{{ row.code }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="receipt_time" label="签收日期" width="150px" align="center">
          <template #default="{row}">
            <el-text tag="b" v-if="row.receipt_time" size="default">{{ DateFormat(row.receipt_time) }}</el-text>
            <el-text v-else>-</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="出库类型" width="110px" align="center"/>
        <el-table-column prop="status" label="出库状态" width="110px" align="center">
          <template #default="{row}">
            <el-tag size="default" :type="orderStatus(row.status)">{{row.status}}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="supplier_name" label="供应商" width="300px">
          <template #default="{row}">
            <span v-if="row.supplier_name">{{ row.supplier_name }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="customer_name" label="客户" width="300px">
          <template #default="{row}">
            <span v-if="row.customer_name">{{ row.customer_name }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="total_amount" label="物料总金额" width="120px"/>
        <el-table-column prop="carrier_name" label="承运商" width="120px"/>
        <el-table-column prop="carrier_cost" label="运费" width="120px"/>
        <el-table-column prop="other_cost" label="其他费用" width="120px"/>
        <el-table-column prop="remark" label="备注"/>
        <el-table-column label="操作" width="400px">
          <template #default="{row}">
            <perms-button
                v-if="globalStatus==='预发货'"
                perms="outbound:order:confirm"
                :disabled="!['预发货'].includes(row.status)"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @click="confirm(row)"
            />
            <el-popconfirm
                v-if="globalStatus===''"
                :title="`确定删除发货单[${row.code}]吗?`"
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
                    :disabled="!['预发货'].includes(row.status)"
                    perms="outbound:order:delete"
                    :type="Types.danger"
                    :size="Sizes.small"
                    :plain="true"/>
              </template>
            </el-popconfirm>
            <perms-button
                v-if="globalStatus==='待拣货'"
                perms="outbound:order:pick"
                :disabled="!['待拣货'].includes(row.status)"
                :type="Types.success"
                :size="Sizes.small"
                :plain="true"
                @click="pick(row)"
            />
            <perms-button
                v-if="globalStatus==='待打包'"
                perms="outbound:order:pack"
                :disabled="!(row.status==='已拣货' || (row.status==='已称重' && row.is_pack===0))"
                :type="Types.success"
                :size="Sizes.small"
                :plain="true"
                @click="pack(row)"
            />
            <perms-button
                v-if="globalStatus==='待称重'"
                perms="outbound:order:weigh"
                :disabled="!(row.status==='已拣货' || (row.status==='已打包' && row.is_weigh===0))"
                :type="Types.success"
                :size="Sizes.small"
                :plain="true"
                @click="weigh(row)"
            />

            <perms-button
                v-if="globalStatus==='待出库'"
                perms="outbound:order:departure"
                :disabled="!['已拣货','已打包','已称重'].includes(row.status)"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="handleDeparture(row)"
            />

            <perms-button
                v-if="globalStatus==='已出库'"
                perms="outbound:order:receipt"
                :disabled="!['已出库'].includes(row.status)"
                :type="Types.danger"
                :size="Sizes.small"
                :plain="true"
                @click="handleReceipt(row)"
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
          :page-sizes="[20, 30, 50]"
          background
          layout="total, sizes, prev, pager, next, ->,jumper"
          :pager-count="9"
          :disabled="loading"
          :hide-on-single-page="false"
          :total="total"
      ></el-pagination>
    </el-card>
    <el-dialog
        v-model.trim="visible"
        :title="title"
        draggable
        :fullscreen="['add', 'edit'].includes(action)"
        :close-on-click-modal="false"
        align-center
        :width="dialogWidth"
    >
      <Order
          v-if="visible&&['add'].includes(action)"
          :form="receipt"
          :action="action"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Confirm
          v-if="visible&&['confirm'].includes(action)"
          :code="receipt!.code"
          :action="action"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Pick
          v-if="visible&&['pick'].includes(action)"
          :code="receipt.code"
          :action="action"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Pack
          v-if="visible&&['pack'].includes(action)"
          :code="receipt.code"
          :action="action"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Weigh
          v-if="visible&&action === 'weigh'"
          :title="title"
          :code="receipt?.code"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Departure
          v-if="visible&&action === 'departure'"
          :title="title"
          :code="receipt?.code"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Status
          v-if="visible&&action === 'check'"
          :receipt="receipt"
          @cancel="visible=false"
          @success="handleSuccess"
      />
      <Receipt
        v-if="visible&&action === 'receipt'"
        :code="receipt?.code"
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


.receipt {
  padding-bottom: 40px;
  margin-bottom: 40px;
  border: 2px dashed #1e80ff;
  background-color: rgba(215, 209, 35, 0.4);
}

.table {
  margin: 20px 30px;
}

table {
  width: 100%;
  border-collapse: collapse;
  border: 2px solid #1e80ff;
  font-size: 18px;
}

table th, table td {
  border: 1px solid #1e80ff;
  padding: 5px;
}


.el-tabs {
  --el-tabs-header-height: auto !important;
}
</style>