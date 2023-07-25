<script setup lang="ts">

import {onMounted, ref} from "vue";
import {ElMessage, FormInstance} from "element-plus";
import {Supplier, SuppliersRequest, SupplierStatusRequest} from "@/api/supplier/types.ts";
import {reqChangeSupplierStatus, reqSuppliers} from "@/api/supplier";
import {Sizes, Types} from "@/utils/enum.ts";
import {TimeFormat} from "@/utils/time.ts";
import Status from "./components/Status.vue";
import Item from "./components/Item.vue";
import {SupplierLevels} from "@/enums/supplier.ts";


let initSuppliersForm = () => {
  return <SuppliersRequest>{
    page: 1,
    size: 10,
    name: '',
    code: '',
    manager: '',
    contact: '',
    email: '',
    level: '',
  }
}

let levelText = (level: number) => {
  switch (level) {
    case 1:
      return '一级'
    case 2:
      return '二级'
    case 3:
      return '三级'
    default:
      return '未知'
  }
}

let levelType = (level: number) => {
  switch (level) {
    case 1:
      return 'success'
    case 2:
      return 'warning'
    case 3:
      return 'danger'
    default:
      return 'info'
  }
}

let statusType = (status:string) => {
  switch (status) {
    case '待审核':
      return ''
    case '审核不通过':
      return 'danger'
    case '活动':
      return 'success'
    case '停用':
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
let suppliersForm = ref<SuppliersRequest>(initSuppliersForm())
let suppliersRef = ref<FormInstance>()
let suppliers = ref<Supplier[]>([])
let total = ref<number>(0)

let getSuppliers = async () => {
  loading.value = true
  let res = await reqSuppliers(suppliersForm.value)
  if (res.code === 200) {
    suppliers.value = res.data.list
    total.value = res.data.total
  } else {
    suppliers.value = []
    total.value = 0
  }
  loading.value = false

}

let reset = async () => {
  suppliersForm.value = initSuppliersForm()
  await getSuppliers()
}

let handleSizeChange = () => {
  getSuppliers()
}
let handleCurrentChange = () => {
  getSuppliers()
}


//删除供应商
const remove = async (id: string) => {
  let req = <SupplierStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeSupplierStatus(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getSuppliers()
  } else {
    ElMessage.error(res.msg)
  }
}

const title = ref<string>('')
const visible = ref<boolean>(false)
const action = ref<string>('')

let initSupplier = () => {
  return <Supplier>{
    id: '',
    type: '企业',
    name: '',
    level: 1,
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

let supplier = ref<Supplier>(initSupplier())

//添加供应商
let add = async () => {

  action.value = 'add'
  supplier.value = initSupplier()
  title.value = '添加供应商'
  visible.value = true
}

//修改供应商
const edit = (item: Supplier) => {
  action.value = 'edit'
  supplier.value = item
  title.value = `修改供应商[${item.name}]`
  visible.value = true
}

//修改状态
const changeStatus = (item: Supplier) => {
  action.value = 'status'
  supplier.value = item
  title.value = `修改供应商[${item.name}]状态`
  visible.value = true
}

//表单提交成功
const handleSuccess = () => {
  getSuppliers()
  visible.value = false
}

onMounted(() => {
  getSuppliers()
})
</script>

<template>
  <div>
    <el-card>
      <el-form
          inline
          :model="suppliersForm"
          ref="suppliersRef"
          label-width="80px"
          style="display: flex; flex-wrap: wrap;"
      >
        <el-form-item prop="name" label="名称">
          <el-input v-model="suppliersForm.name" placeholder="请填写供应商名称" clearable/>
        </el-form-item>
        <el-form-item prop="code" label="编号">
          <el-input v-model="suppliersForm.code" placeholder="请填写供应商编号" clearable/>
        </el-form-item>
        <el-form-item prop="level" label="等级">
          <el-select v-model="suppliersForm.level" placeholder="请选择供应商等级" clearable>
            <el-option v-for="(one, idx) in SupplierLevels" :key="idx" :label="one.label" :value="one.value"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item prop="manager" label="负责人">
          <el-input v-model="suppliersForm.manager" placeholder="请填写负责人" clearable/>
        </el-form-item>
        <el-form-item prop="contact" label="联系方式">
          <el-input v-model="suppliersForm.contact" placeholder="请填写联系方式" clearable/>
        </el-form-item>
        <el-form-item prop="email" label="Email">
          <el-input v-model="suppliersForm.email" placeholder="请填写Email" clearable/>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" plain icon="Search" @click="getSuppliers">查询</el-button>
          <el-button @click="reset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>
    <!--  供应商分页  -->
    <el-card
        class="data"
    >
      <el-button type="primary" plain icon="Plus" @click="add">添加供应商</el-button>
      <el-table
          class="table"
          border
          stripe
          :data="suppliers"
      >
        <el-table-column label="供应商名称" prop="name" fixed width="250px">
          <template #default="{row}">
            <el-text type="primary" size="default" tag="b" truncated>{{ row.name }}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="供应商图片" width="150px" align="center">
          <template #default="{row}">
            <el-image
                v-if="row.image"
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
        <el-table-column label="供应商状态" prop="status" width="120px" align="center">
          <template #default="{row}">
           <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="级别" prop="level" width="80px" align="center">
          <template #default="{row}">
            <el-tag :type="levelType(row.level)" size="small" truncated>{{ levelText(row.level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="法定代表人" prop="legal_representative" width="150px"/>
        <el-table-column label="统一社会信用代码" prop="unified_social_credit_identifier" width="150px"/>
        <el-table-column label="负责人" prop="" width="190px">
          <template #default="{row}">
            {{ row.manager }} {{ row.contact }}
          </template>
        </el-table-column>
        <el-table-column label="供应商类型" prop="type" width="120px"></el-table-column>
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
        <el-table-column label="操作" fixed="right" width="280px">
          <template #default="{row}">
            <perms-button
                perms="supplier:supplier:status"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @click="changeStatus(row)"
            />
            <perms-button
                perms="supplier:supplier:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除供应商[${row.name}]吗?`"
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
                    perms="supplier:supplier:delete"
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
          v-model:current-page="suppliersForm.page"
          v-model:page-size="suppliersForm.size"
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
          :supplier="supplier"
          @success="handleSuccess"
          @cancel="visible=false"
      />
      <Status
          v-if="visible&&action === 'status'"
          :supplier="supplier"
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
</style>