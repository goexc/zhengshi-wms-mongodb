<script setup lang="ts">

import {onMounted, ref} from "vue";
import {ElMessage, FormInstance} from "element-plus";
import {Carrier, CarriersRequest, CarrierStatusRequest} from "@/api/carrier/types.ts";
import {reqChangeCarrierStatus, reqCarriers} from "@/api/carrier";
import {Sizes, Types} from "@/utils/enum.ts";
import {TimeFormat} from "@/utils/time.ts";
import Status from "./components/Status.vue";
import Item from "./components/Item.vue";


let initCarriersForm = () => {
  return <CarriersRequest>{
    page: 1,
    size: 10,
    name: '',
    code: '',
    manager: '',
    contact: '',
    email: '',
  }
}


let statusType = (status:string) => {
  switch (status) {
    case '待审核':
      return ''
    case '审核中':
      return 'info'
    case '审核不通过':
      return 'warning'
    case '审核通过':
      return 'success'
    case '活跃':
      return 'success'
    case '停用':
      return 'warning'
    case '暂停合作':
      return 'warning'
    case '终止合作':
      return 'danger'
    case '资质过期':
      return 'info'
    default:
      return ''
  }
}

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

let loading = ref<boolean>(false)
let carriersForm = ref<CarriersRequest>(initCarriersForm())
let carriersRef = ref<FormInstance>()
let carriers = ref<Carrier[]>([])
let total = ref<number>(0)

let getCarriers = async () => {
  loading.value = true
  let res = await reqCarriers(carriersForm.value)
  if (res.code === 200) {
    carriers.value = res.data.list
    total.value = res.data.total
  } else {
    carriers.value = []
    total.value = 0
  }
  loading.value = false

}

let reset = async () => {
  carriersForm.value = initCarriersForm()
  await getCarriers()
}

let handleSizeChange = () => {
  getCarriers()
}
let handleCurrentChange = () => {
  getCarriers()
}


//删除承运商
const remove = async (id: string) => {
  let req = <CarrierStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeCarrierStatus(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getCarriers()
  } else {
    ElMessage.error(res.msg)
  }
}

const title = ref<string>('')
const visible = ref<boolean>(false)
const action = ref<string>('')

let initCarrier = () => {
  return <Carrier>{
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

let carrier = ref<Carrier>(initCarrier())

//添加承运商
let add = async () => {

  action.value = 'add'
  carrier.value = initCarrier()
  title.value = '添加承运商'
  visible.value = true
}

//修改承运商
const edit = (item: Carrier) => {
  action.value = 'edit'
  carrier.value = item
  title.value = `修改承运商[${item.name}]`
  visible.value = true
}

//修改状态
const changeStatus = (item: Carrier) => {
  action.value = 'status'
  carrier.value = item
  title.value = `修改承运商[${item.name}]状态`
  visible.value = true
}

//表单提交成功
const handleSuccess = () => {
  getCarriers()
  visible.value = false
}

onMounted(() => {
  getCarriers()
})
</script>

<template>
  <div>
    <el-card
        v-auth="'business_partner:carrier:list'"
    >
      <el-form
          inline
          :model="carriersForm"
          ref="carriersRef"
          label-width="80px"
          style="display: flex; flex-wrap: wrap;"
      >
        <el-form-item prop="name" label="名称">
          <el-input v-model="carriersForm.name" placeholder="请填写承运商名称" clearable/>
        </el-form-item>
        <el-form-item prop="code" label="编号">
          <el-input v-model="carriersForm.code" placeholder="请填写承运商编号" clearable/>
        </el-form-item>
        <el-form-item prop="manager" label="负责人">
          <el-input v-model="carriersForm.manager" placeholder="请填写负责人" clearable/>
        </el-form-item>
        <el-form-item prop="contact" label="联系方式">
          <el-input v-model="carriersForm.contact" placeholder="请填写联系方式" clearable/>
        </el-form-item>
        <el-form-item prop="email" label="Email">
          <el-input v-model="carriersForm.email" placeholder="请填写Email" clearable/>
        </el-form-item>
        <el-form-item label=" ">
          <perms-button
              perms="business_partner:carrier:list"
              :type="Types.primary"
              :size="Sizes.large"
              :plain="true"
              @click="getCarriers"
          />
          <perms-button
              perms="business_partner:carrier:list"
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
    <!--  承运商分页  -->
    <el-card
        class="data"
    >
      <perms-button
          perms="business_partner:carrier:add"
          :type="Types.primary"
          :size="Sizes.large"
          :plain="true"
          icon=""
          text="添加承运商"
          @click="add"
      />
      <el-table
          class="table"
          border
          stripe
          :data="carriers"
      >
        <el-table-column label="承运商名称" prop="name" fixed width="250px">
          <template #default="{row}">
            <el-text type="primary" size="default" tag="b" truncated>{{ row.name }}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="承运商图片" width="150px" align="center">
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
        <el-table-column label="应收账款" class="money" width="250px" align="center">
          <template #default="{row}">
            <p>10000.000</p>
            <el-text class="money" type="primary" size="small">+应收</el-text>
            <el-text class="money" type="primary" size="small">-结款</el-text>
            <el-text class="money" type="primary" size="small">查看流水</el-text>
          </template>
        </el-table-column>
        <el-table-column label="承运商状态" prop="status" width="120px" align="center">
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
        <el-table-column label="承运商类型" prop="type" width="120px"></el-table-column>
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
                perms="business_partner:carrier:status"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @click="changeStatus(row)"
            />
            <perms-button
                perms="business_partner:carrier:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除承运商[${row.name}]吗?`"
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
                    perms="business_partner:carrier:delete"
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
          v-model:current-page="carriersForm.page"
          v-model:page-size="carriersForm.size"
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
          :carrier="carrier"
          @success="handleSuccess"
          @cancel="visible=false"
      />
      <Status
          v-if="visible&&action === 'status'"
          :carrier="carrier"
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
.money {
  cursor: pointer;
  padding: 5px 11px;
    margin-left: 12px;
}
</style>