<script setup lang="ts">
import {onMounted, reactive, ref} from "vue";
import {
  CustomerTransaction,
  CustomerTransactionAddRequest,
  CustomerTransactionPageRequest
} from "@/api/customer/types.ts";
import {reqAddCustomerTransaction, reqCustomerTransactions} from "@/api/customer";
import {ElMessage, FormRules} from "element-plus";
import {DateFormat} from "@/utils/time.ts";
import {CustomerTransactionTypes} from "@/enums/customer.ts";
import dayjs from "dayjs";

defineOptions({
  name: 'Transaction'
})

const props = defineProps(['customer'])
// const emit = defineEmits(['success', 'cancel'])

const form = ref<CustomerTransactionPageRequest>({
  page: 1,
  size: 20,
  customer_id: props.customer.id,
})
const loading = ref<boolean>(false)
const visible = ref<boolean>(false)

//交易流水
const list = ref<CustomerTransaction[]>([])
//交易金额
const total = ref<number>(0)

//获取交易流水
const getTransactions = async () => {
  loading.value = true
  list.value = []

  let res = await reqCustomerTransactions(form.value)
  if (res.code === 200) {
    list.value = res.data.list
    total.value = res.data.total
  } else {
    list.value = []
    total.value = 0
    ElMessage.error(res.msg)
  }
  loading.value = false
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
    case '应收账款':
      return ''
    case '回款':
      return 'success'
    case '退货':
      return 'danger'
    default:
      return ''
  }
}

//添加回款记录
const handleRecord = () => {
  visible.value = true
}

//添加交易记录表单
const recordForm = ref<CustomerTransactionAddRequest>({
  customer_id: props.customer.id,
  time: 0,
  type: '回款',
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
  type: [
    {required: true, message: '必填', trigger: ['blur', 'change']},
    {type: 'enum', enum: CustomerTransactionTypes, message: '请选择指定的交易类型', trigger: ['blur', 'change']},
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
  if(!recordForm.value.annex.includes(image)) {
    recordForm.value.annex.push(image)
  }
}
//删除附件
const handleRemove = (image: string) => {
  console.log('handleRemove:', image)
  recordForm.value.annex = recordForm.value.annex.filter(item => item !== image)
}

//添加交易记录
const handleAdd = async () => {

  //表单校验
  let valid = await recordFormRef.value?.validate((isValid:boolean) => {
    if (!isValid) {
    }
    return
  })

  if (!valid) {
    return
  }

  loading.value = true
  let res = await reqAddCustomerTransaction(recordForm.value)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    visible.value = false
    await getTransactions()
  } else {
    ElMessage.error(res.msg)
  }
  loading.value = false
}

onMounted(() => {
  getTransactions()
})

</script>

<template>
  <el-form
  >
    <el-form-item>
      <el-button type="success" plain size="default" icon="CirclePlus" :loading="loading" @click="handleRecord">
        添加回款记录
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
        <el-tag :type="transactionType(row.type)">{{ row.type }}</el-tag>
      </template>

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
      title="添加回款记录"
      draggable
      width="800"
      :close-on-click-modal="false"
  >
    <el-form
        label-width="80"
        size="default"
        :model="recordForm"
        ref="recordFormRef"
        :rules="rules"
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
          prop="type"
      >
        <el-radio-group v-model.trim="recordForm.type" size="default">
          <el-radio-button
              v-for="($type,$idx) in CustomerTransactionTypes"
              :key="$idx"
              :label="$type"
          />
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
        <el-input size="default" v-model.number="recordForm.remark" placeholder="请填写备注" clearable/>
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
      <el-form-item
          label=""
      >
        <el-button plain @click="visible=false">取消</el-button>
        <el-button plain type="primary" @click="handleAdd" :loading="loading">确定</el-button>
      </el-form-item>
    </el-form>
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