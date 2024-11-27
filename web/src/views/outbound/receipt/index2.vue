<script setup lang="ts">

import {OutboundOrderTypes} from "@/enums/outbound.ts";
import {Sizes, Types} from "@/utils/enum.ts";
import {OutboundOrder, OutboundOrdersRequest} from "@/api/outbound/types.ts";
import {onMounted, ref} from "vue";
import {
  reqOutboundOrders2,
  reqRemoveOutboundOrder,
} from "@/api/outbound";
import {ElMessage} from "element-plus";
import Status from "./components/Status.vue";
import Order from "./components/Order.vue";
import Weigh from "./components/Weigh.vue";
import Pick from "./components/Pick.vue";
import Pack from "./components/Pack.vue";
import Receipt from "./components/Receipt.vue";
import SupplierPageItem from "@/components/Supplier/SupplierPageItem.vue";
import CustomerPageItem from "@/components/Customer/CustomerPageItem.vue";
import Confirm from "@/views/outbound/receipt/components/Confirm.vue";
import Departure from "@/views/outbound/receipt/components/Departure.vue";
import NP from "number-precision";
import {DateFormat} from "@/utils/time.ts";
import Revise from "@/views/outbound/receipt/components/Revise.vue";


//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)
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
    start_time: 0,
    end_time: 0,
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
  let res = await reqOutboundOrders2(form.value)
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

//修正发货单的物料数量/物料单价
const handleRevise =  async (item: OutboundOrder) => {
  receipt.value = item
  action.value = 'revise'
  title.value = `修正物料数量/物料单价`
  dialogWidth.value = '1880px'
  visible.value = true
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

// let materials = ref<OutboundOrderMaterial[]>([])

/*

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
*/

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

        <el-form-item label="起始日期">
          <el-date-picker
              type="date"
              placeholder="请选择签收起始日期"
              size="default"
              value-format="X"
              v-model.number="form.start_time"
          />
        </el-form-item>
        <el-form-item label="截止日期">
          <el-date-picker
              type="date"
              placeholder="请选择签收截止日期"
              size="default"
              value-format="X"
              v-model.number="form.end_time"
          />
        </el-form-item>
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

      <!--   分页   -->
      <el-pagination
          class="m-b-1"
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
      <!-- 发货单列表 -->
      <div v-for="(item,index) in receipts" :key="index" class="receipt">
        <div style="text-align: right;padding: 10px 10px 0 0">
          <!-- 修正发货单的物料数量/物料单价 -->
          <perms-button
              v-if="globalStatus===''"
              perms="outbound:order:revise"
              :type="Types.success"
              :size="Sizes.small"
              :plain="true"
              @click="handleRevise(item)"
          />
          <perms-button
              v-if="globalStatus==='预发货'"
              perms="outbound:order:confirm"
              :disabled="!['预发货'].includes(item.status)"
              :type="Types.primary"
              :size="Sizes.small"
              :plain="true"
              @click="confirm(item)"
          />
          <el-popconfirm
              :title="`确定删除发货单[${item.code}]吗?`"
              icon="InfoFilled"
              icon-color="#F56C6C"
              cancel-button-text="取消"
              confirm-button-text="确认删除"
              cancel-button-type="info"
              confirm-button-type="danger"
              @confirm="remove(item)"
              width="300"
          >
            <template #reference>
              <perms-button
                  perms="outbound:order:delete"
                  :type="Types.danger"
                  :size="Sizes.small"
                  :plain="true"/>
            </template>
          </el-popconfirm>
          <perms-button
              v-if="globalStatus==='待拣货'"
              perms="outbound:order:pick"
              :disabled="!['待拣货'].includes(item.status)"
              :type="Types.success"
              :size="Sizes.small"
              :plain="true"
              @click="pick(item)"
          />
          <perms-button
              v-if="globalStatus==='待打包'"
              perms="outbound:order:pack"
              :disabled="!(item.status==='已拣货' || (item.status==='已称重' && item.is_pack===0))"
              :type="Types.success"
              :size="Sizes.small"
              :plain="true"
              @click="pack(item)"
          />
          <perms-button
              v-if="globalStatus==='待称重'"
              perms="outbound:order:weigh"
              :disabled="!(item.status==='已拣货' || (item.status==='已打包' && item.is_weigh===0))"
              :type="Types.success"
              :size="Sizes.small"
              :plain="true"
              @click="weigh(item)"
          />

          <perms-button
              v-if="globalStatus==='待出库'"
              perms="outbound:order:departure"
              :disabled="!['已拣货','已打包','已称重'].includes(item.status)"
              :type="Types.warning"
              :size="Sizes.small"
              :plain="true"
              @click="handleDeparture(item)"
          />

          <perms-button
              v-if="globalStatus==='已出库'"
              perms="outbound:order:receipt"
              :disabled="!['已出库'].includes(item.status)"
              :type="Types.danger"
              :size="Sizes.small"
              :plain="true"
              @click="handleReceipt(item)"
          />
        </div>
        <div style="text-align: center"><h3>出库单</h3></div>
        <div>
          <el-row :gutter="20">
            <el-col :span="8" :offset="1">客户：<b>{{ item.customer_name }}</b><el-tag v-if="item.materials.find((one) => one.price===0)" type="danger">包含未定价产品</el-tag></el-col>
            <el-col :span="8">出库类型：<b>{{item.type}}</b></el-col>
            <el-col :span="7">编号：<b>{{ item.code }}</b></el-col>
<!--            <el-col :span="8">含税：<b>{{ !item.has_tax ? '否' : `是(${item.tax}%)` }}</b></el-col>-->
          </el-row>
        </div>
        <div class="m-y-1">
          <el-row :gutter="20">
            <el-col :span="8" :offset="1">状态：<b>{{item.status}}</b></el-col>
            <el-col :span="8">出库日期：<b>{{ DateFormat(item.departure_time) }}</b></el-col>
            <el-col :span="7">签收日期：<b>{{ DateFormat(item.receipt_time) }}</b></el-col>
<!--            <el-col :span="8" :offset="1">单据类型：<b>估价入库</b><el-tag v-if="item.materials.find((one) => one.price===0)">包含未定价产品</el-tag></el-col>-->
          </el-row>
        </div>
        <div class="m-y-1">
          <el-row :gutter="20">
            <el-col :span="8" :offset="1">承运商：<b v-if="item.carrier_name">{{item.carrier_name}}</b><b v-else>-</b></el-col>
            <el-col :span="8">运费：<b>{{ item.carrier_cost }}</b></el-col>
            <el-col :span="7">其他费用：<b>{{ item.other_cost }}</b></el-col>
          </el-row>
        </div>
        <div class="m-y-1">
          <el-row :gutter="20">
            <el-col :span="8" :offset="1">备注：<b v-if="item.remark">{{item.remark}}</b><b v-else>-</b></el-col>
          </el-row>
        </div>
        <div style="align-content: center;padding: 0 20px">
          <table>
            <thead>
            <tr>
              <th>序号</th>
              <th>产品</th>
              <th>型号</th>
              <th>规格</th>
              <th>数量</th>
              <th>单价</th>
              <th>金额</th>
            </tr>
            </thead>
            <tbody>
            <tr v-for="(row,idx) in item.materials" :key="idx">
              <td>{{ row.index }}</td>
              <td>{{ row.name }}</td>
              <td>{{ row.model }}</td>
              <td>{{ row.specification }}</td>
              <td>{{ row.quantity }}</td>
              <td>{{ row.price }}</td>
              <td>{{ NP.times(row.quantity, row.price) }}</td>
            </tr>
            <tr>
              <td>合计</td>
              <td>-</td>
              <td>-</td>
              <td>-</td>
              <td>{{ item.materials.map(material => material.quantity).reduce((total, value) => total + value, 0) }}</td>
              <td>-</td>
              <td>{{ item.total_amount }}</td>
            </tr>
            </tbody>
          </table>
          <el-form
          >
            <el-form-item :label="`单据(${!!item.annex?item.annex.length:0})`">
              <el-image
                  class="m-t-1 m-x-1"
                  v-if="!!item.annex"
                  v-for="($annex,$idx) in item.annex"
                  :key="$idx"
                  :src="`${ oss_domain }${$annex}_296x148`"
                  :infinite="false"
                  :hide-on-click-modal="true"
                  :preview-teleported="true"
                  :preview-src-list="item.annex.map(item=>oss_domain+item+'_1024x1024')"
                  style="width: 296px;height: 148px;"
              ></el-image>
            </el-form-item>
          </el-form>

<!--
          <div
              v-for="($image, $index) in item.receipt"
          >
            <el-image
                v-if="$image&&$image.endsWith('.svg')"
                :key="$index"
                :src="`${ oss_domain }${$image}`"
                :infinite="true"
                :preview-teleported="true"
                :preview-src-list="[`${ oss_domain }${$image}`]"
                style="width: 148px;height: 148px;"
            ></el-image>
            <el-image
                v-if="$image&&!$image.endsWith('.svg')"
                :key="$index"
                :src="`${ oss_domain }${$image}_148x148`"
                :infinite="true"
                :preview-teleported="true"
                :preview-src-list="[`${ oss_domain }${$image}`]"
                style="width: 148px;height: 148px;"
            ></el-image>
          </div>
          -->
        </div>
      </div>
      
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
          :code="receipt.code"
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

      <Revise
        v-if="visible&&action === 'revise'"
        :code="receipt?.code"
        :customer_id="receipt?.customer_id"
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