<script setup lang="ts">

import {Sizes, Types} from "@/utils/enum.ts";
import {nextTick, onMounted,  ref} from "vue";
import { reqMaterialCategoryList, reqRemoveMaterialCategory} from "@/api/material";
import {MaterialCategory, MaterialCategoryIdRequest, MaterialCategorysResponse} from "@/api/material/types.ts";
import {ElMessage} from "element-plus";
import {TimeFormat} from "@/utils/time.ts";
import Item from "./components/Item.vue";

//表格展开属性
const expand = ref<boolean>(true)
const tableVisible = ref<boolean>(true)
//物料分类展开
const switchExpand = (val: boolean) => {
  tableVisible.value = false
  expand.value = val

  nextTick(() => {
    tableVisible.value = true
  })
}

const categorys = ref<MaterialCategory[]>([])
const getMaterialCategorys = async () => {
  let res: MaterialCategorysResponse = await reqMaterialCategoryList()
  if (res.code === 200) {
    console.log('分类列表：', res.data)
    categorys.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
}

let StatusType = (status:string) => {
  switch (status) {
    case '启用':
      return 'success'
    case '停用':
      return 'danger'
    default:
      return 'info'
  }
}

onMounted(() => {
  getMaterialCategorys()
})

//dialog可见
const visible = ref<boolean>(false)
const title = ref<string>('')

//表单提交成功
const handleSuccess = () => {
  getMaterialCategorys()
  visible.value = false
}

const initMaterialCategory = () => {
  return <MaterialCategory>({
    id: '',
    parent_id: '',
    sort_id: 1,
    name: '',
    image: '',
    status: '',//状态：启用、停用
    remark: '',
  })
}


const category = ref<MaterialCategory>(initMaterialCategory())

//添加物料分类
const add = (parent: MaterialCategory) => {
  category.value = initMaterialCategory()
  category.value.parent_id = parent.id
  title.value = '添加物料分类'
  visible.value = true
}

//修改物料分类
const edit = (item: MaterialCategory) => {
  category.value = item
  title.value = `修改物料分类[${item.name}]`
  visible.value = true
}


//删除物料分类
const remove = async (id: string) => {
  let req = <MaterialCategoryIdRequest>({id: id})
  let res = await reqRemoveMaterialCategory(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getMaterialCategorys()
  } else {
    ElMessage.error(res.msg)
  }
}


</script>

<template>
  <div>
    <div id="auth" v-auth="'material:category:list'">
      <el-button plain @click="switchExpand(true)" size="default" icon="ArrowDown">展开全部</el-button>
      <el-button plain @click="switchExpand(false)" size="default" icon="ArrowUp">折叠全部</el-button>
      <perms-button
          perms="material:category:add"
          :type="Types.primary"
          :size="Sizes.default"
          :plain="true"
          @action="add(initMaterialCategory())"
      />
      <!--   展示物料分类列表   -->
      <el-table
          v-if="tableVisible"
          class="table"
          stripe
          border
          :show-overflow-tooltip="true"
          tooltip-effect="dark"
          :data="categorys"
          row-key="id"
          :default-expand-all="expand"
          :tree-props="{ children: 'children', hasChildren: '!!children' }"
      >

        <template #empty>
          <el-empty/>
        </template>
        <el-table-column prop="name" label="分类名称" min-width="220px" fixed>
          <template #default="{row}">
            <el-link>{{ row.name }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="排序" prop="sort_id" width="80px" align="center"></el-table-column>
        <el-table-column label="状态" prop="status" width="220px">
          <template #default="{row}">
            <el-tag :type="StatusType(row.status)" size="small">{{row.status}}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建人" prop="creator_name" width="220px"></el-table-column>
        <el-table-column label="备注" prop="remark" min-width="120px"></el-table-column>
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
        <el-table-column label="操作" min-width="300px" fixed="right">
          <template #default="{row}">
            <perms-button
                perms="material:category:add"
                :type="Types.primary"
                :size="Sizes.small"
                :plain="true"
                @action="add(row)"
            />
            <perms-button
                perms="material:category:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @action="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除物料分类[${row.name}]吗?`"
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
                    perms="material:category:delete"
                    :type="Types.danger"
                    :size="Sizes.small"
                    :plain="true"
                />
                <!--              <el-button type="danger" plain size="small">删除</el-button>-->
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      <el-dialog
          v-model.trim="visible"
          :title="title"
          draggable
          width="800"
          :close-on-click-modal="false"
      >
        <Item
            v-if="visible"
            :category="category"
            @success="handleSuccess"
            @cancel="visible=false"
        />
      </el-dialog>
    </div>

  </div>
</template>

<style scoped lang="scss">
.form-card {
  height: 80px;
}

.form {
  display: flex;
  justify-content: flex-start;
  align-items: center;
}

.table {
  margin: 10px 0;
}
</style>