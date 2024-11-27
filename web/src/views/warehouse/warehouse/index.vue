<script setup lang="ts">

import {ref} from "vue";
import {reqChangeWarehouseStatus, reqWarehouses} from "@/api/warehouse";
import {Warehouse, WarehousesRequest, WarehouseStatusRequest} from "@/api/warehouse/types.ts";
import {onMounted} from "vue";
import {ElMessage} from "element-plus";
import {TimeFormat} from "@/utils/time.ts";
import {WarehouseStatus, WarehouseTypes} from "@/enums/warehouse.ts";
import Item from "./components/Item.vue";
import {Sizes, Types} from "@/utils/enum.ts";
import Status from "@/views/warehouse/warehouse/components/Status.vue";

const loading = ref<boolean>(false)
const visible = ref<boolean>(false)
const action = ref<string>('')
const title = ref<string>('')

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

const total = ref<number>(0)
const initWarehousesForm = () => {
  return {
    page: 1,
    size: 10,
    type: '',
    name: '',
    code: '',
    status: ''
  }
}

//仓库列表
let warehouses = ref<Warehouse[]>([])
let warehousesForm = ref<WarehousesRequest>(initWarehousesForm())

//获取仓库列表
const getWarehouses = async () => {
  loading.value = true
  let res = await reqWarehouses(warehousesForm.value)
  if (res.code === 200) {
    warehouses.value = res.data.list
    total.value = res.data.total
  } else {
    warehouses.value = []
    total.value = 0
    ElMessage.error(res.msg)
  }
  loading.value = false
}

//重置表单
const reset = () => {
  warehousesForm.value = initWarehousesForm()
  getWarehouses()
}


//仓库数据
const initWarehouse = () => {
  return <Warehouse>{
    id: '',
    type: '',
    name: '',
    code: '',
    address: '',
    capacity: 0,
    capacity_unit: '',
    status: '',
    manager: '',
    contact: '',
    image: '',
    remark: '',
  }
}

const warehouse = ref<Warehouse>(initWarehouse())

const handleSizeChange = () =>{
  getWarehouses()
}
const handleCurrentChange = () =>{
  getWarehouses()
}

//添加仓库
const add = () => {
  action.value = 'add'
  warehouse.value = initWarehouse()
  title.value = '添加仓库'
  visible.value = true
}

//修改仓库
const edit = (item: Warehouse) => {
  action.value = 'edit'
  warehouse.value = item
  title.value = '修改仓库'
  visible.value = true
}

//修改状态
const changeStatus = (item: Warehouse) => {
  action.value= 'status'
  warehouse.value = item
  title.value = `修改仓库[${item.name}]状态`
  visible.value = true
}

//删除仓库
const remove = async (id: string) => {
  console.log('删除仓库：', id)
  let req = <WarehouseStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeWarehouseStatus(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getWarehouses()
  } else {
    ElMessage.error(res.msg)
  }
}

//表单提交成功
const handleSuccess = () => {
  getWarehouses()
  visible.value = false
}

const statusType = (status:string) => {
  switch(status){
    case '激活':
      return 'success'
    case '盘点中':
      return 'warning'
    case '关闭':
      return 'info'
    case '禁用':
      return 'danger'
    default:
      return ''
  }
}

onMounted(async () => {
  await getWarehouses()
})
</script>

<template>
  <div>
    <el-card
    >
      <!--  三级组件  -->
      <el-form
          :inline="true"
          style="display: flex; flex-wrap: wrap;"
          :model="warehousesForm"
      >
        <el-form-item label="仓库名称" prop="name">
          <el-input v-model.trim="warehousesForm.name" clearable placeholder="请填写仓库名称"/>
        </el-form-item>
        <el-form-item label="仓库类型" prop="type">
          <el-select filterable v-model.trim="warehousesForm.type" clearable placeholder="请选择仓库类型">
            <el-option v-for="(item,idx) in WarehouseTypes" :key="idx" :label="`${idx+1}.${item}`"
                       :value="item"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="仓库编号" prop="code">
          <el-input v-model.trim="warehousesForm.code" clearable placeholder="请填写仓库编号"/>
        </el-form-item>
        <el-form-item label="仓库状态" prop="status">
          <el-select filterable v-model.trim="warehousesForm.status" clearable placeholder="请选择仓库状态">
            <el-option v-for="(item,idx) in WarehouseStatus" :key="idx" :label="`${idx+1}.${item}`"
                       :value="item"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" plain @click="getWarehouses" icon="Search">查询</el-button>
          <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card
        class="data"
    >
      <el-button type="primary" plain icon="CirclePlus" @click="add">添加仓库</el-button>
      <el-table
          class="table"
          border
          stripe
          :data="warehouses"
      >
        <el-table-column label="仓库名称" prop="name" fixed width="150px"></el-table-column>
        <el-table-column label="仓库图片" width="150px" align="center">
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
        <el-table-column label="仓库类型" prop="type" width="120px"></el-table-column>
        <el-table-column label="仓库编号" prop="code" min-width="100px"></el-table-column>
        <el-table-column label="仓库状态" prop="status" width="90px" align="center">
          <template #default="{row}">
            <el-tag size="default" :type="statusType(row.status)">{{row.status}}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="容积" width="100px">
          <template #default="{row}">
            {{ row.capacity }} {{ row.capacity_unit }}
          </template>
        </el-table-column>
        <el-table-column label="负责人" prop="" min-width="180px">
          <template #default="{row}">
            {{ row.manager }} {{ row.contact }}
          </template>
        </el-table-column>
        <el-table-column label="地址" prop="address" min-width="120px"></el-table-column>
        <el-table-column label="备注" prop="remark"></el-table-column>
        <el-table-column label="创建人" prop="created_at" min-width="180px">
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
                perms="warehouse:warehouse:status"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @click="changeStatus(row)"
            />
            <perms-button
                perms="warehouse:warehouse:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除仓库[${row.name}]吗?`"
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
                    perms="warehouse:warehouse:delete"
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
          v-model:current-page="warehousesForm.page"
          v-model:page-size="warehousesForm.size"
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
        v-model.trim="visible"
        :title="title"
        draggable
        width="800"
        :close-on-click-modal="false"
    >
      <Item
          v-if="visible&&['add', 'edit'].includes(action)"
          :warehouse="warehouse"
          @success="handleSuccess"
          @cancel="visible=false"
      />
      <Status
          v-if="visible&&action === 'status'"
          :warehouse="warehouse"
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

.image {
  width: 100px;
  height: 100px;
}
</style>