<script setup lang="ts">

import {onMounted, ref} from "vue";
import {ElMessage, FormInstance} from "element-plus";
import {Customer, CustomersRequest, CustomerStatusRequest} from "@/api/customer/types.ts";
import {reqChangeCustomerStatus, reqCustomers} from "@/api/customer";
import {Sizes, Types} from "@/utils/enum.ts";
import {TimeFormat} from "@/utils/time.ts";
import Item from "./components/Item.vue";
import Status from "./components/Status.vue";


let initCustomersForm = () => {
  return <CustomersRequest>{
    page: 1,
    size: 10,
    name: '',
    code: '',
    manager: '',
    contact: '',
    email: '',
  }
}


let statusType = (status: string) => {
  switch (status) {
    case '潜在':
      return ''
    case '活动':
      return 'success'
    case '停用':
      return 'danger'
    case '冻结':
      return 'info'
    case '黑名单':
      return 'danger'
    case '合同到期':
      return 'warning'
    default:
      return ''
  }
}

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

let loading = ref<boolean>(false)
let customersForm = ref<CustomersRequest>(initCustomersForm())
let customersRef = ref<FormInstance>()
let customers = ref<Customer[]>([])
let total = ref<number>(0)

let getCustomers = async () => {
  loading.value = true
  let res = await reqCustomers(customersForm.value)
  if (res.code === 200) {
    customers.value = res.data.list
    total.value = res.data.total
  } else {
    customers.value = []
    total.value = 0
  }
  loading.value = false

}

let reset = async () => {
  customersForm.value = initCustomersForm()
  await getCustomers()
}

let handleSizeChange = () => {
  getCustomers()
}
let handleCurrentChange = () => {
  getCustomers()
}


//删除客户
const remove = async (id: string) => {
  let req = <CustomerStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeCustomerStatus(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getCustomers()
  } else {
    ElMessage.error(res.msg)
  }
}

const title = ref<string>('')
const visible = ref<boolean>(false)
const action = ref<string>('')

let initCustomer = () => {
  return <Customer>{
    id: '',
    type: '企业',
    name: '',
    code: '',
    image: '',
    legal_representative: '',
    unified_social_credit_identifier: '',
    address: '',
    manager: '',
    contact: '',
    status: '',
    remark: '',
  }
}

let customer = ref<Customer>(initCustomer())

//添加客户
let add = async () => {

  action.value = 'add'
  customer.value = initCustomer()
  title.value = '添加客户'
  visible.value = true
}

//修改客户
const edit = (item: Customer) => {
  action.value = 'edit'
  customer.value = item
  title.value = `修改客户[${item.name}]`
  visible.value = true
}

//修改状态
const changeStatus = (item: Customer) => {
  action.value = 'status'
  customer.value = item
  title.value = `修改客户[${item.name}]状态`
  visible.value = true
}

//表单提交成功
const handleSuccess = () => {
  getCustomers()
  visible.value = false
}

onMounted(() => {
  getCustomers()
})
</script>

<template>
  <div>
    <el-card
    v-auth="'business_partner:customer:list'"
    >
      <el-form
          inline
          :model="customersForm"
          ref="customersRef"
          label-width="80px"
          style="display: flex; flex-wrap: wrap;"
      >
        <el-form-item prop="name" label="名称">
          <el-input v-model="customersForm.name" placeholder="请填写客户名称" clearable/>
        </el-form-item>
        <el-form-item prop="code" label="编号">
          <el-input v-model="customersForm.code" placeholder="请填写客户编号" clearable/>
        </el-form-item>
        <el-form-item prop="manager" label="负责人">
          <el-input v-model="customersForm.manager" placeholder="请填写负责人" clearable/>
        </el-form-item>
        <el-form-item prop="contact" label="联系方式">
          <el-input v-model="customersForm.contact" placeholder="请填写联系方式" clearable/>
        </el-form-item>
        <el-form-item prop="email" label="Email">
          <el-input v-model="customersForm.email" placeholder="请填写Email" clearable/>
        </el-form-item>
        <el-form-item label=" ">
          <perms-button
              perms="business_partner:customer:list"
              :type="Types.primary"
              :size="Sizes.large"
              :plain="true"
              @click="getCustomers"
          />
          <perms-button
              perms="business_partner:customer:list"
              :type="Types.empty"
              :size="Sizes.large"
              :plain="true"
              icon="Refresh"
              text="重置"
              @click="reset"
          />
        </el-form-item>
      </el-form>
    </el-card>
    <!--  客户分页  -->
    <el-card
        class="data"
    >
      <el-button type="primary" plain icon="Plus" @click="add">添加客户</el-button>
      <el-table
          class="table"
          border
          stripe
          :data="customers"
      >
        <el-table-column label="客户名称" prop="name" fixed width="250px">
          <template #default="{row}">
            <el-text type="primary" size="default" tag="b" truncated>{{ row.name }}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="客户类型" prop="type" width="120px"></el-table-column>
        <el-table-column label="客户图片" width="150px" align="center">
          <template #default="{row}">
            <el-image
                v-if="row.image.endsWith('.svg')"
                class="image"
                fit="contain"
                :src="`${oss_domain}${row.image}`"
                :preview-src-list="[`${oss_domain}${row.image}`]"
                hide-on-click-modal
                preview-teleported
            />
            <el-image
                v-else-if="row.image"
                class="image"
                fit="contain"
                :src="`${oss_domain}${row.image}_148x148`"
                :preview-src-list="[`${oss_domain}${row.image}`]"
                hide-on-click-modal
                preview-teleported
            />
          </template>
        </el-table-column>
        <el-table-column label="编号" prop="code" width="150px"/>
        <el-table-column label="应收账款" width="250px" align="center">
          <template #default="{row}">
            <p>10000.000</p>
            <el-text class="money" type="primary" size="small">+应收</el-text>
            <el-text class="money" type="primary" size="small">-结款</el-text>
            <el-text class="money" type="primary" size="small">查看流水</el-text>
          </template>
        </el-table-column>
        <el-table-column label="客户状态" prop="status" width="120px" align="center">
          <template #default="{row}">
            <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="法定代表人" prop="legal_representative" width="150px"/>
        <el-table-column label="统一社会信用代码" prop="unified_social_credit_identifier" width="150px"/>
        <el-table-column label="负责人" prop="" width="190px">
          <template #default="{row}">
            {{ row.manager }} {{ row.contact }}
          </template>
        </el-table-column>
        <el-table-column label="Email" prop="email" min-width="150px"></el-table-column>
        <el-table-column label="地址" prop="address" min-width="120px"></el-table-column>
        <el-table-column label="备注" prop="remark"></el-table-column>
        <el-table-column label="创建人" prop="created_at" width="100px">
          <template #default="{row}">
            {{ row.create_by }}
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="created_at" width="180px">
          <template #default="{row}">
            {{ TimeFormat(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="修改时间" prop="updated_at" width="180px">
          <template #default="{row}">
            {{ TimeFormat(row.updated_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="300px">
          <template #default="{row}">
            <perms-button
                perms="business_partner:customer:status"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @click="changeStatus(row)"
            />
            <perms-button
                perms="business_partner:customer:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除客户[${row.name}]吗?`"
                icon="InfoFilled"
                icon-color="#F56C6C"
                cancel-button-text="取消"
                confirm-button-text="确认删除"
                cancel-button-type="info"
                confirm-button-type="danger"
                @confirm="remove(row.id)"
                width="300"
            >
              <template #reference>
                <perms-button
                    perms="business_partner:customer:delete"
                    :type="Types.danger"
                    :size="Sizes.small"
                    :plain="true"/>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <!--   分页   -->
      <el-pagination
          v-model:current-page="customersForm.page"
          v-model:page-size="customersForm.size"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
          :page-sizes="[10, 20, 30, 40]"
          background
          layout="total, sizes, prev, pager, next, ->,jumper"
          :pager-count="9"
          :disabled="loading"
          :hide-on-single-page="false"
          :total="total"
      ></el-pagination>
    </el-card>
    <el-dialog
        v-model="visible"
        :title="title"
        draggable
        width="800"
        :close-on-click-modal="false"
    >
      <Item
          v-if="visible&&['add', 'edit'].includes(action)"
          :customer="customer"
          @success="handleSuccess"
          @cancel="visible=false"
      />
      <Status
          v-if="visible&&action === 'status'"
          :customer="customer"
          @success="handleSuccess"
          @cancel="visible=false"
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

//应收账款
.cell .money {
  cursor: pointer;
  padding: 5px 11px;
    margin-left: 12px;
}
</style>