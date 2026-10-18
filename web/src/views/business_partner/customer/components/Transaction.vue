<script setup lang="ts">
import {onBeforeUnmount, onMounted, reactive, ref, watch} from "vue";
import {
  CustomerTransaction,
  CustomerTransactionAddRequest,
  CustomerTransactionPageRequest
} from "@/api/customer/types.ts";
import {reqAddCustomerTransaction, reqCustomerTransactions} from "@/api/customer";
import {ElMessage, FormRules} from "element-plus";
import {DateFormat} from "@/utils/time.ts";
import {CustomerTransactionTypeLabels, CustomerTransactionTypes, CustomerTransactionSourceLabels, CustomerTransactionStatusLabels} from "@/enums/customer.ts";
import DocumentAttachments from '@/views/finance/components/DocumentAttachments.vue';
import dayjs from "dayjs";

defineOptions({
  name: 'Transaction'
})

const props = defineProps(['customer'])
const emit = defineEmits(['success'])

const form = ref<CustomerTransactionPageRequest>({
  page: 1,
  size: 20,
  customer_id: props.customer.id,
})
const loading = ref<boolean>(false)
const visible = ref<boolean>(false)
const submitting = ref(false)
const uploading = ref(false)
const recordError = ref('')
const pendingRecord = ref<CustomerTransactionAddRequest>()
let recordWasUncertain = false
let loadGeneration = 0
let disposed = false

//交易流水
const list = ref<CustomerTransaction[]>([])
//交易金额
const total = ref<number>(0)

//获取交易流水
const getTransactions = async () => {
  const generation = ++loadGeneration
  loading.value = true
  list.value = []
  try {
  const res = await reqCustomerTransactions({...form.value})
  if (generation !== loadGeneration) return
  if (res.code !== 200) throw new Error(res.msg)
    list.value = res.data.list
    total.value = res.data.total
  } catch (error) {
    if (generation !== loadGeneration) return
    list.value = []
    total.value = 0
    ElMessage.error(error instanceof Error ? error.message : '交易流水加载失败，请重试')
  } finally { if (generation === loadGeneration) loading.value = false }
}


let handleSizeChange = () => {
  getTransactions()
}
let handleCurrentChange = () => {
  getTransactions()
}

//交易类型样式
const transactionType = (type: string) => {
  switch (type) {
    case 'outbound_ar':
    case 'opening_ar':
    case 'manual_adjustment':
      return ''
    case 'payment':
      return 'success'
    case 'return_credit':
    case 'ar_adjustment':
      return 'danger'
    default:
      return ''
  }
}

const transactionLabel = (row: CustomerTransaction) => {
  return row.type || CustomerTransactionTypeLabels[row.transaction_type] || row.transaction_type
}
const transactionAnnex = (value: string) => (value || '').split(',').map(item => item.trim()).filter(Boolean);

const createIdempotencyKey = () => {
  const uuid = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(16).slice(2)}`
  return `manual:${props.customer.id}:${uuid}`
}

//添加回款记录
const handleRecord = () => {
  if (submitting.value || uploading.value) return
  if (!pendingRecord.value) {
    recordError.value = ''
    recordForm.value = { customer_id: props.customer.id, time: 0, transaction_type: 'payment', direction: 'receivable_decrease', idempotency_key: createIdempotencyKey(), amount: 0, remark: '', annex: [] }
  }
  visible.value = true
}

//添加交易记录表单
const recordForm = ref<CustomerTransactionAddRequest>({
  customer_id: props.customer.id,
  time: 0,
  transaction_type: 'payment',
  direction: 'receivable_decrease',
  idempotency_key: '',
  amount: 0,
  remark: '',
  annex: [],
})
const recordFormRef = ref()

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

//添加交易记录参数校验规则
const rules = reactive<FormRules>({
  time: [
    {required: true, message: '必填', trigger: ['blur', 'change']},
    {type: "number", min: 1, message: '请选择交易时间', trigger: ['blur', 'change']},
    {type: "number", max: dayjs().unix(), message: '交易时间不能超过当前时间', trigger: ['blur', 'change']},
  ],
  transaction_type: [
    {required: true, message: '必填', trigger: ['blur', 'change']},
    {type: 'enum', enum: CustomerTransactionTypes.map(one => one.value), message: '请选择指定的交易类型', trigger: ['blur', 'change']},
  ],
  amount: [
    {required: true, message: '必填', trigger: ['blur', 'change']},
    {type: "number", min: 0.001, message: '请填写交易金额', trigger: ['blur', 'change']},
  ],
  remark: [
    {required: true, message: '必填', trigger: ['blur', 'change']},
  ]
})

//更换附件
const handleSelect = (image: string) => {
  if (submitting.value || pendingRecord.value) return
  if(!recordForm.value.annex.includes(image)) {
    recordForm.value.annex.push(image)
  }
}
//删除附件
const handleRemove = (image: string) => {
  if (submitting.value || pendingRecord.value) return
  recordForm.value.annex = recordForm.value.annex.filter(item => item !== image)
}

//添加交易记录
const handleAdd = async () => {
  if (disposed || submitting.value || uploading.value) return
  submitting.value = true
  recordError.value = ''
  try {
  if (!pendingRecord.value) {
  if (!(await recordFormRef.value?.validate().catch(() => false)) || disposed) return
  if (!recordForm.value.idempotency_key) {
    recordForm.value.idempotency_key = createIdempotencyKey()
  }
  const option = CustomerTransactionTypes.find(one => one.value === recordForm.value.transaction_type)
  recordForm.value.direction = option?.direction || 'receivable_decrease'
  pendingRecord.value = {...recordForm.value, annex: [...recordForm.value.annex]}
  }
  const res = await reqAddCustomerTransaction(pendingRecord.value)
  if (disposed) return
  if (res.code !== 200) throw Object.assign(new Error(res.msg), {status: res.code})
    pendingRecord.value = undefined
    recordWasUncertain = false
    ElMessage.success(res.msg)
    visible.value = false
    await getTransactions()
    emit('success')
  } catch (error) {
    if (disposed) return
    const candidate = error as {status?: number; response?: {status?: number; data?: {msg?: string}}; message?: string}
    const status = candidate.response?.status ?? candidate.status
    recordWasUncertain ||= !status || status >= 500
    if (!recordWasUncertain) pendingRecord.value = undefined
    recordError.value = `${candidate.response?.data?.msg || candidate.message || '请求失败'}。${pendingRecord.value ? '尚未确认结果，已保留本次提交，请点击“重试确认”。' : '请按提示修正后提交。'}`
  } finally { if (!disposed) submitting.value = false }
}

onMounted(() => {
  getTransactions()
})
watch(() => props.customer.id, customerID => { form.value.customer_id = customerID; form.value.page = 1; void getTransactions() })
onBeforeUnmount(() => { disposed = true; ++loadGeneration })

</script>

<template>
  <el-form
  >
    <el-form-item>
      <el-button type="success" plain size="default" icon="CirclePlus" :loading="loading" @click="handleRecord">
        添加交易记录
      </el-button>
      <el-button type="primary" plain size="default" icon="Refresh" :loading="loading" @click="getTransactions">刷新
      </el-button>
    </el-form-item>
  </el-form>
  <!--   分页   -->
  <el-pagination
      class="m-y-2"
      v-model:current-page="form.page"
      v-model:page-size="form.size"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
      :page-sizes="[20, 30, 40]"
      background
      layout="total, sizes, prev, pager, next, ->,jumper"
      :pager-count="9"
      :disabled="loading"
      :hide-on-single-page="false"
      :total="total"
  ></el-pagination>
  <el-table
      :data="list"
      size="default"
      border
  >
    <template #empty>
      <el-empty/>
    </template>
    <el-table-column label="序号" type="index" prop="index" width="60px" align="center"/>
    <el-table-column label="交易类型" prop="type" align="center">
      <template #default="{row}">
        <el-tag :type="transactionType(row.transaction_type)">{{ transactionLabel(row) }}</el-tag>
      </template>

    </el-table-column>
    <el-table-column label="来源" min-width="120">
      <template #default="{row}">
        <div>{{ CustomerTransactionSourceLabels[row.source_type] || '其他来源' }}</div>
        <el-text v-if="row.source_code" type="info" size="small">{{ row.source_code }}</el-text>
      </template>
    </el-table-column>
    <el-table-column label="状态" width="90" align="center">
      <template #default="{row}">{{ CustomerTransactionStatusLabels[row.status] || '—' }}</template>
    </el-table-column>
    <el-table-column label="交易时间" prop="time" align="center">
      <template #default="{row}">
        {{ DateFormat(row.time) }}
      </template>
    </el-table-column>
    <el-table-column label="交易金额" prop="amount" align="right">
      <template #default="{row}">
        <el-text size="default" tag="b">￥{{ row.amount.toFixed(4) }}</el-text>
      </template>
    </el-table-column>
    <el-table-column label="备注" prop="remark"/>
    <el-table-column label="附件">
      <template #default="{row}">
        <DocumentAttachments v-if="row.source_type === 'customer_acceptance'" :model-value="transactionAnnex(row.annex)" />
        <template v-else>
        <el-image
            class="m-t-1 m-x-1"
            v-if="!!row.annex&&row.annex.length>0"
            v-for="($annex,$idx) in row.annex.split(',')"
            :key="$idx"
            :src="`${ oss_domain }${$annex}_296x148`"
            :infinite="false"
            :hide-on-click-modal="true"
            :preview-teleported="true"
            :preview-src-list="row.annex.split(',').map((one:string)=>oss_domain+one+'_1024x1024')"
            style="width: 296px;height: 148px;"
        ></el-image>
        </template>
      </template>
    </el-table-column>
  </el-table>
  <!--   分页   -->
  <el-pagination
      class="m-y-2"
      v-model:current-page="form.page"
      v-model:page-size="form.size"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
      :page-sizes="[20, 30, 40]"
      background
      layout="total, sizes, prev, pager, next, ->,jumper"
      :pager-count="9"
      :disabled="loading"
      :hide-on-single-page="false"
      :total="total"
  ></el-pagination>
  <el-dialog
      v-model.trim="visible"
      title="添加交易记录"
      draggable
      width="800"
      :close-on-click-modal="false"
      :close-on-press-escape="!submitting && !uploading && !pendingRecord"
      :show-close="!submitting && !uploading && !pendingRecord"
  >
    <el-form
        label-width="80"
        size="default"
        :model="recordForm"
        ref="recordFormRef"
        :rules="rules"
        :disabled="submitting || !!pendingRecord"
    >
      <el-form-item
          label="客户"
      >
        <el-text size="default" tag="b">{{ props.customer.name }}</el-text>
      </el-form-item>
      <el-form-item
          label="交易时间"
          prop="time"
      >
        <el-date-picker
            v-model.number="recordForm.time"
            type="date"
            placeholder="请选择交易时间"
            size="default"
            value-format="X"
        />
      </el-form-item>
      <el-form-item
          label="交易类型"
          prop="transaction_type"
      >
        <el-radio-group v-model.trim="recordForm.transaction_type" size="default">
          <el-radio-button
              v-for="($type,$idx) in CustomerTransactionTypes"
              :key="$idx"
              :label="$type.value"
          >{{ $type.label }}</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item
          label="交易金额"
          prop="amount"
      >
        <el-input size="default" v-model.number="recordForm.amount" placeholder="请填写交易金额" clearable/>
      </el-form-item>
      <el-form-item
          label="备注"
          prop="remark"
      >
        <el-input size="default" v-model.trim="recordForm.remark" placeholder="请填写备注" clearable/>
      </el-form-item>
      <el-form-item
          label="附件"
      >
      <ImageUpload
          @select="handleSelect"
          @remove="handleRemove"
          :multiple="true"
          :limit="10"
          :urls.sync="recordForm.annex"
          :disabled="submitting || !!pendingRecord"
          @uploading="uploading = $event"
      />
      </el-form-item>
      <el-form-item>
        <div
            v-for="($image, $index) in recordForm.annex"
        >
          <el-image
              :key="$index"
              :src="`${ oss_domain }${$image}_148x148`"
              :infinite="true"
              :preview-teleported="true"
              :preview-src-list="[`${ oss_domain }${$image}`]"
              style="width: 148px;height: 148px;"
          ></el-image>

          <div class="image-action space-evenly color-white">
            <el-button
                type="danger"
                icon="Delete"
                circle
                @click="handleRemove($image)"
                size="small"
            ></el-button>
          </div>
        </div>
      </el-form-item>
    </el-form>
    <el-alert v-if="recordError" :title="recordError" type="error" :closable="false" show-icon />
    <template #footer>
        <el-button plain @click="visible=false" :disabled="submitting || uploading || !!pendingRecord">取消</el-button>
        <el-button plain type="primary" @click="handleAdd" :loading="submitting" :disabled="uploading">{{ pendingRecord ? '重试确认' : '确定' }}</el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">

.image-action {
  //background-color: rgba(204, 204, 204, 0.5);
  position: relative;
  bottom: 160px;
  left: 100px;
  font-size: 24px;

  vertical-align: middle;
  margin: 10px;

}
</style>
