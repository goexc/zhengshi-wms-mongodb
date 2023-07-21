<script setup lang="ts">

import {RackStatus, RackTypes} from "@/enums/rack.ts";
import {Rack, RacksRequest, RackStatusRequest} from "@/api/warehouse_rack/types.ts";
import {onMounted, ref} from "vue";
import {reqChangeRackStatus, reqRacks} from "@/api/warehouse_rack";
import {ElMessage} from "element-plus";
import {Sizes, Types} from "@/utils/enum.ts";
import {TimeFormat} from "@/utils/time.ts";
import Item from "./components/Item.vue";
import Status from "./components/Status.vue";

//货架列表
const initRacksForm = () => {
  return <RacksRequest>{
    page: 1,
    size: 10,
    warehouse_id: '',
    warehouse_zone_id: '',
    type: '',
    name: '',
    code: '',
    status: '',
  }
}


//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN)

const form = ref<RacksRequest>(initRacksForm())
const racks = ref<Rack[]>([])
const total = ref<number>(0)
const loading = ref<boolean>(false)
const getRacks = async () => {
  loading.value = true
  let res = await reqRacks(form.value)
  if (res.code === 200) {
    racks.value = res.data.list
    total.value = res.data.total
  } else {
    ElMessage.error(res.msg)
    racks.value = []
    total.value = 0
  }
  loading.value = false
}
const reset = async () => {
  form.value = initRacksForm()
  await getRacks()
}

const handleSizeChange = () =>{
  getRacks()
}
const handleCurrentChange = () =>{
  getRacks()
}

const title = ref<string>('')
const visible = ref<boolean>(false)
const action = ref<string>('')
//货架数据
const initRack = () => {
  return <Rack>{
    id: '',
    warehouse_id: '',
    warehouse_name: '',
    name: '',
    type: '',
    code: '',
    capacity: 0,
    capacity_unit: '',
    status: '',
    manager: '',
    contact: '',
    remark: '',
  }
}

const rack = ref<Rack>(initRack())

//添加货架
const add = () => {
  action.value = 'add'
  rack.value = initRack()
  title.value = '添加货架'
  visible.value = true
}

//修改货架
const edit = (item: Rack) => {
  action.value = 'edit'
  rack.value = item
  title.value = `修改货架[${item.name}]`
  visible.value = true
}

//修改状态
const changeStatus = (item: Rack) => {
  action.value = 'status'
  rack.value = item
  title.value = `修改货架[${item.name}]状态`
  visible.value = true
}

//删除货架
const remove = async (id: string) => {
  let req = <RackStatusRequest>{id: id, status: '删除'}
  let res = await reqChangeRackStatus(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getRacks()
  } else {
    ElMessage.error(res.msg)
  }
}

//表单提交成功
const handleSuccess = () => {
  getRacks()
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
  await getRacks()
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
          :model="form"
      >
        <WarehousePageItem
            :form="form"
        />
        <ZonePageItem
            :form="form"
        />
        <el-form-item label="货架名称" prop="name">
          <el-input v-model="form.name" clearable placeholder="请填写货架名称"/>
        </el-form-item>
        <el-form-item label="货架编号" prop="code">
          <el-input v-model="form.code" clearable placeholder="请填写货架编号"/>
        </el-form-item>
        <el-form-item label="货架类型" prop="type">
          <el-select v-model="form.type" clearable placeholder="请选择货架类型">
            <el-option v-for="(item,idx) in RackTypes" :key="idx" :label="`${idx+1}.${item}`"
                       :value="item"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="货架状态" prop="status">
          <el-select v-model="form.status" clearable placeholder="请选择货架状态">
            <el-option v-for="(item,idx) in RackStatus" :key="idx" :label="`${idx+1}.${item}`"
                       :value="item"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" plain @click="getRacks" icon="Search">查询</el-button>
          <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 货架列表 -->
    <el-card
        class="data"
    >
      <el-button type="primary" plain icon="Plus" @click="add">添加货架</el-button>
      <el-table
          class="table"
          border
          stripe
          :data="racks"
      >
        <el-table-column label="货架名称" prop="name" fixed width="150px">
          <template #default="{row}">
            <el-text type="primary" size="default" tag="b" truncated>{{row.name}}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="货架图片" width="150px" align="center">
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
        <el-table-column label="所属仓库" prop="warehouse_name" width="150px">
          <template #default="{row}">
            <el-text type="info" size="default" tag="ins" truncated>{{row.warehouse_name}}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="所属库区" prop="warehouse_zone_name" width="150px">
          <template #default="{row}">
            <el-text type="info" size="default" tag="ins" truncated>{{row.warehouse_zone_name}}</el-text>
          </template>
        </el-table-column>
        <!--        <el-table-column label="货架类型" prop="type" width="120px"></el-table-column>-->
        <el-table-column label="货架编号" prop="code" min-width="100px"></el-table-column>
        <el-table-column label="货架类型" prop="type" width="100px"></el-table-column>
        <el-table-column label="货架状态" prop="status" width="90px">
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
        <!--        <el-table-column label="地址" prop="address" min-width="120px"></el-table-column>-->
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
                perms="warehouse:rack:status"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @click="changeStatus(row)"
            />
            <perms-button
                v-if="row.status === '激活'"
                perms="warehouse:rack:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除货架[${row.name}]吗?`"
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
                    perms="warehouse:rack:delete"
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
          v-model:current-page="form.page"
          v-model:page-size="form.size"
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
          :rack="rack"
          @success="handleSuccess"
          @cancel="visible=false"
      />
      <Status
          v-if="visible&&action === 'status'"
          :rack="rack"
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

// vue3 + element-plus 使用Image 图片组件，点击图片预览功能。
// 发现在table中，会出现预览的图片被遮罩。
// 审查元素后，感觉应该是定位fixed导致的，
// 因为如果父元素的 transform, perspective 或 filter 属性不为 none 时，fixed 元素就会相对于父元素来进行定位 的。
// 解决办法
// vue3的新特性中有一项 Teleport 标签，它可以将我们的模板移动到 DOM中的其他位置。
// element-plus基于vue3的plus版本中，新增了一项新属性：preview-teleported
// preview-teleported:
// image-viewer 是否插入至 body 元素上。 嵌套的父元素属性会发生修改时应该将此属性设置为 true
</style>