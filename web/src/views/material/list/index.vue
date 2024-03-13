<script setup lang="ts">

import {onMounted, ref} from "vue";
import {Material, MaterialIdRequest, MaterialsRequest} from "@/api/material/types.ts";
import {ElMessage, ElMessageBox} from "element-plus";
import {reqMaterials, reqRemoveMaterial, reqRemoveMaterialPrice} from "@/api/material";
import {Sizes, Types} from "@/utils/enum.ts";
import {DateFormat, TimeFormat} from "@/utils/time.ts";
import Item from "./components/Item.vue";

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN as string)

//物料列表
const initMaterialsForm = () => {
  return <MaterialsRequest>{
    page: 1,
    size: 10,
    name: '',//物料名称
    category_id: '', //物料分类id
    image: '', //物料图片
    model: '',//型号：用于唯一标识和区分不同种类的钢材，例如：RGV4102030035。
    material: '',//材质：碳钢、不锈钢、合金钢等。
    specification: '',//规格：包括长度、宽度、厚度等尺寸信息。
    surface_treatment: '',//表面处理。钢材经过的表面处理方式，如热镀锌、喷涂等。
    strength_grade: '',//强度等级：钢材的强度等级，常见的钢材强度等级：Q235、Q345
  }
}

const form = ref<MaterialsRequest>(initMaterialsForm())
const materials = ref<Material[]>([])
const total = ref<number>(0)
const loading = ref<boolean>(false)
const getMaterials = async () => {
  loading.value = true
  let res = await reqMaterials(form.value)
  if (res.code === 200) {
    materials.value = res.data.list
    total.value = res.data.total
  } else {
    ElMessage.error(res.msg)
    materials.value = []
    total.value = 0
  }
  loading.value = false
}
const reset = async () => {
  form.value = initMaterialsForm()
  await getMaterials()
}

const handleSizeChange = () => {
  getMaterials()
}
const handleCurrentChange = () => {
  getMaterials()
}

const title = ref<string>('')
const visible = ref<boolean>(false)
const action = ref<string>('')
//物料数据
const initMaterial = () => {
  return <Material>{
    id: '',
    name: '',
    category_id: '', //物料分类id
    category_name: '', //物料分类名称
    image: '',
    model: '',//型号：用于唯一标识和区分不同种类的钢材。
    material: '',//材质：碳钢、不锈钢、合金钢等。
    specification: '',//规格：包括长度、宽度、厚度等尺寸信息。
    surface_treatment: '',//表面处理。钢材经过的表面处理方式，如热镀锌、喷涂等。
    strength_grade: '',//强度等级：钢材的强度等级，常见的钢材强度等级：Q235、Q345
    quantity: 0,//安全库存
    unit: '',//计量单位，如个、箱、千克等
    remark: '',//
    prices: [],//
    creator: '',
    creator_name: '',
    created_at: 0,
    updated_at: 0,
  }
}

const material = ref<Material>(initMaterial())

//添加物料
const add = () => {
  action.value = 'add'
  material.value = initMaterial()
  title.value = '添加物料'
  visible.value = true
}

//修改物料
const edit = (item: Material) => {
  action.value = 'edit'
  material.value = item
  title.value = `修改物料[${item.name}]`
  visible.value = true
}

//删除物料
const remove = async (id: string) => {
  let req = <MaterialIdRequest>{id: id}
  let res = await reqRemoveMaterial(req)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getMaterials()
  } else {
    ElMessage.error(res.msg)
  }
}

//删除物料价格
let removeMaterialPrice= async (item:Material, customer_id:string, price:number) =>{
  let result = await ElMessageBox.confirm(
      `确认删除 [${item.model}] 单价：${price}？`,
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

  let res = await reqRemoveMaterialPrice(item.id,customer_id, price)
  if (res.code === 200){
    ElMessage.success(res.msg)
    item.prices = item.prices.filter(item=>item.price!==price)
  }else{
    ElMessage.error(res.msg)
  }
}

//表单提交成功
const handleSuccess = () => {
  getMaterials()
  visible.value = false
}

onMounted(async () => {
  await getMaterials()
})
</script>

<template>
  <div>

<!--    <el-card-->
<!--    >-->
      <!--  三级组件  -->
      <el-form
          :inline="true"
          style="display: flex; flex-wrap: wrap;"
          :model="form"
          label-width="80px"
          size="large"
      >
        <MaterialCategoryListItem
          :form="form"
          />
        <el-form-item label="名称" prop="name">
          <el-input v-model.trim="form.name" clearable placeholder="请填写名称"/>
        </el-form-item>
        <el-form-item label="型号" prop="model">
          <el-input v-model.trim="form.model" clearable placeholder="请填写型号"/>
        </el-form-item>
        <el-form-item label="材质" prop="material">
          <el-input v-model.trim="form.material" clearable placeholder="请填写材质"/>
        </el-form-item>
        <el-form-item label="规格" prop="specification">
          <el-input v-model.trim="form.specification" clearable placeholder="请填写规格"/>
        </el-form-item>
        <el-form-item label="表面处理" prop="surface_treatment">
          <el-input v-model.trim="form.surface_treatment" clearable placeholder="请选择表面处理"/>
        </el-form-item>
        <el-form-item label="强度等级" prop="strength_grade">
          <el-input v-model.trim="form.strength_grade" clearable placeholder="请选择强度等级"/>
        </el-form-item>
<!--        <el-form-item label="物料状态" prop="status">
          <el-select v-model.trim="form.status" clearable placeholder="请选择物料状态">
            <el-option v-for="(item,idx) in ['启用', '停用']" :key="idx" :label="`${idx+1}.${item}`"
                       :value="item"></el-option>
          </el-select>
        </el-form-item>-->
        <el-form-item label=" ">
          <el-button type="primary" plain @click="getMaterials" icon="Search">查询</el-button>
          <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
<!--    </el-card>-->
    <!-- 物料列表 -->
    <el-card
        class="data"
    >
      <el-button type="primary" plain icon="CirclePlus" @click="add">添加物料</el-button>
      <!--   分页   -->
      <el-pagination
          class="m-t-2"
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
      <el-table
          class="table"
          border
          stripe
          :data="materials"
      >
        <el-table-column label="物料名称" prop="name" fixed min-width="180px">
          <template #default="{row}">
            <el-text type="primary" size="default" tag="b" truncated>{{row.name}}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="物料图片" width="150px" align="center">
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
        <el-table-column label="型号" prop="model" min-width="180px"></el-table-column>
        <el-table-column label="分类" prop="category_name" min-width="100px"></el-table-column>
        <el-table-column label="材质" prop="material" min-width="100px"></el-table-column>
        <el-table-column label="规格" prop="specification" min-width="100px"></el-table-column>
        <el-table-column label="表面处理" prop="surface_treatment" min-width="100px"></el-table-column>
        <el-table-column label="强度等级" prop="strength_grade" min-width="100px"></el-table-column>
        <el-table-column label="安全库存" prop="quantity" min-width="100px"></el-table-column>
        <el-table-column label="计量单位" prop="unit" min-width="100px"></el-table-column>
        <el-table-column label="单价" prop="price" width="320px">
          <template #default="{row}">
            <el-popover placement="left" width="220" v-for="(one, idx) in row.prices" :key="idx">
              <template #reference>
                <el-tag type="danger" closable
                        @close="removeMaterialPrice(row, one.customer_id, one.price)"
                >￥{{ one.price }} ({{one.customer_name}}) {{DateFormat(one.since)}}</el-tag>
              </template>
              定价时间：{{DateFormat(one.since)}}
            </el-popover>
          </template>
        </el-table-column>
        <el-table-column label="备注" prop="remark" min-width="100px"></el-table-column>
        <el-table-column label="创建人" prop="creator_name" width="100px">
          <template #default="{row}">
            {{ row.creator_name }}
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
        <el-table-column label="操作" fixed="right" width="200px">
          <template #default="{row}">
            <perms-button
                perms="material:material:edit"
                :type="Types.warning"
                :size="Sizes.small"
                :plain="true"
                @click="edit(row)"
            />
            <el-popconfirm
                :title="`确定删除物料[${row.name}]吗?`"
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
                    perms="material:material:delete"
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
        v-model.trim="visible"
        :title="title"
        draggable
        width="800"
        :close-on-click-modal="false"
    >
      <Item
          v-if="visible"
          :material="material"
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