<script setup lang="ts">
import {CustomerOutbondReceiptTypes, OutboundReceiptTypes, SupplierOutbondReceiptTypes} from "@/enums/outbound.ts";
import {onMounted, reactive, ref} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {WarehouseTree} from "@/api/warehouse/types.ts";
import {reqWarehouseTree} from "@/api/warehouse";
import {OutboundMaterial, OutboundReceipt} from "@/api/outbound/types.ts";
import MaterialCategoryListItem from "@/components/MaterialCategory/MaterialCategoryListItem.vue";
import {Material, MaterialPrice, MaterialsRequest} from "@/api/material/types.ts";
import {reqMaterialPrices, reqMaterials, reqRemoveMaterialPrice} from "@/api/material";
import {reqAddOrUpdateOutboundReceipt} from "@/api/outbound";
import CustomerListItem from "@/components/Customer/CustomerListItem.vue";
import * as dayjs from "dayjs";
import NP from "number-precision";

defineOptions({
  name: 'Item'
})
const emit = defineEmits(['success', 'cancel'])

let props = defineProps(['form', 'action'])
//接收属性中的数据
let receipt = ref<OutboundReceipt>({
  id: '',
  code: '',
  type: '',
  status: '',
  total_amount: 0,
  supplier_id: '',
  customer_id: '',
  receiving_date: 0,
  materials: [],
  annex: [],
  remark: '',
})
let receiptRef = ref<FormInstance>()


//出库单物料列表
let outboundMaterials = ref<OutboundMaterial[]>([])

let warehouses = ref<WarehouseTree[]>()

//更新物料仓储位置
let changePositions = (value: string[], idx: number) => {
  console.log('index:', idx)

  if (!!value && value.length > 0) {
    outboundMaterials.value[idx].warehouse_id = value[0]
  } else {
    outboundMaterials.value[idx].warehouse_id = ''
  }

  if (!!value && value.length > 1) {
    outboundMaterials.value[idx].warehouse_zone_id = value[1]
  } else {
    outboundMaterials.value[idx].warehouse_zone_id = ''
  }

  if (!!value && value.length > 2) {
    outboundMaterials.value[idx].warehouse_rack_id = value[2]
  } else {
    outboundMaterials.value[idx].warehouse_rack_id = ''
  }

  if (!!value && value.length > 3) {
    outboundMaterials.value[idx].warehouse_bin_id = value[3]
  } else {
    outboundMaterials.value[idx].warehouse_bin_id = ''
  }
}

//删除物料
let removeMaterial = (idx: number) => {
  console.log('删除物料：', idx)
  outboundMaterials.value.splice(idx, 1)
  outboundMaterials.value.forEach((item, idx) => {
    item.index = idx + 1
  })
}


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

//图片域名
const oss_domain = ref<string>(import.meta.env.VITE_OSS_DOMAIN as string)

let visible = ref<boolean>(false)
let materialsForm = ref<MaterialsRequest>(initMaterialsForm())
const materials = ref<Material[]>([])
const total = ref<number>(0)
const loading = ref<boolean>(false)

//获取物料分页
let getMaterials = async () => {
  //查询物料分页
  loading.value = true
  let res = await reqMaterials(materialsForm.value)
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

//物料是否可选
let selectable = (row, rowIndex) => {
  return !outboundMaterials.value.some(item => item.id === row.id);
}

//添加物料
let addMaterial = async () => {
  await getMaterials()

  visible.value = true
}

//选择物料
let selectedMaterials = ref<Material[]>([])
let handleSelectionChange = (val: Material[]) => {
  console.log('选择物料：', val)
  selectedMaterials.value = val
}

//确认物料
let confirmMaterials = async () => {
  let index: number = 0
  if (!!outboundMaterials.value && outboundMaterials.value.length > 0) {
    index = outboundMaterials.value[outboundMaterials.value.length - 1].index
  }
  selectedMaterials.value.forEach((item) => {
    outboundMaterials.value.push({
      index: ++index,
      id: item.id,
      name: item.name,
      model: item.model,
      price: 0,
      estimated_quantity: 0,
      warehousing: [],
      warehouse_id: '',
      warehouse_zone_id: '',
      warehouse_rack_id: '',
      warehouse_bin_id: '',
    })
  })

  visible.value = false
}

//查询物料价格列表
let prices = ref<MaterialPrice[]>([])
let getPrices = async (id: string) => {
  prices.value = []
  let res = await reqMaterialPrices(id)
  console.log('物料单价列表：', res.data)
  if (res.code === 200) {
    prices.value = res.data?.sort((a, b) => {
      // 根据需要的排序逻辑进行比较
      return a.since - b.since
    })
  } else {
    ElMessage.error(res.msg)
  }
}

//计算总金额
let computeTotalAmount = () => {
  receipt.value.total_amount = outboundMaterials.value.reduce((total, current) => {
    return total + NP.times(current.price , current.estimated_quantity);
  }, 0)
}

//删除物料价格
let removeMaterialPrice= async (id:string, price:number) =>{
  let res = await reqRemoveMaterialPrice(id, price)
  if (res.code === 200){
    ElMessage.success(res.msg)
    await getPrices(id)
  }else{
    ElMessage.error(res.msg)
  }
}

//关闭表单
const cancel = () => {
  emit('cancel')
}

let validTotalAmount = (rule, value, callback) => {
  if (value >= 0) {
    callback(); //通过验证
  } else {
    callback(new Error('总金额必须≥0'))
  }
}

let rules = reactive<FormRules>({
  code: [
    {required: true, message: '出库单号不能为空'}
  ],
  type: [
    {required: true, message: '请选择出库类型'}
  ],
  supplier_id: [
    {required: true, message: '请选择供应商'}
  ],
  customer_id: [
    {required: true, message: '请选择客户'}
  ],
  total_amount: [
    {required: true, message: '请填写总金额'},
    {type: "number", validator: validTotalAmount}
  ]
})

//提交表单
const submit = async () => {
  let valid = await receiptRef.value?.validate()
  if (!valid) {
    return
  }

  // props.form.materials = outboundMaterials.value
  receipt.value!.materials = outboundMaterials.value
  //提交数据
  let res = await reqAddOrUpdateOutboundReceipt(receipt.value!)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(async () => {
  //1.初始化出库单编号
  receipt.value = JSON.parse(JSON.stringify(props.form))
  if (receipt.value.code.length === 0) {
    receipt.value.code = 'O-' + dayjs().format('YYYY-MM-DD-HH-mm-ss-SSS')
  }

  //1.接收出库单物料列表
  outboundMaterials.value = JSON.parse(JSON.stringify(props.form.materials.sort((a, b) => {
    // 根据需要的排序逻辑进行比较
    return a.index - b.index
  })))

  //1.获取仓库树
  let res = await reqWarehouseTree()
  console.log(res)
  if (res.code === 200) {
    warehouses.value = res.data
  } else {
    warehouses.value = []
  }

  //2.整理物料对应的仓储信息
  outboundMaterials.value.forEach(material => {
    material.warehousing = []

    if (material.warehouse_id) {
      material.warehousing.push(material.warehouse_id)
    }

    if (material.warehouse_zone_id) {
      material.warehousing.push(material.warehouse_zone_id)
    }

    if (material.warehouse_rack_id) {
      material.warehousing.push(material.warehouse_rack_id)
    }

    if (material.warehouse_bin_id) {
      material.warehousing.push(material.warehouse_bin_id)
    }

  })
})
</script>

<template>
  <el-form
      :model="receipt"
      :rules="rules"
      ref="receiptRef"
      size="default"
      label-width="100px"
  >
    <el-form-item
        disabled
        label="出库单号"
        prop="code"
    >
      <el-input
          v-model="receipt.code"
          class="w300"
          :disabled="action==='edit'"
      />
    </el-form-item>
    <el-form-item label="出库类型" prop="type">
      <el-radio-group v-model="receipt.type">
        <el-radio-button v-for="(item, idx) in OutboundReceiptTypes" :key="idx" plain :label="item"/>
      </el-radio-group>
    </el-form-item>
    <SupplierListItem
        v-if="SupplierOutbondReceiptTypes.includes(receipt.type)"
        :form="receipt"
    />
    <CustomerListItem
        v-if="CustomerOutbondReceiptTypes.includes(receipt.type)"
        :form="receipt"
    />
    <el-form-item
        label="总金额"
        prop="total_amount"
    >
      <el-input-number
          v-model="receipt.total_amount"
          class="w300"
          :controls="false"
          :step="100"
          :precision="3"
          :min="0"
      />
    </el-form-item>
    <el-form-item label="备注" prop="remark">
      <el-input
          v-model="receipt.remark"
          type="textarea"
          rows="3"
          maxlength="125"
          :show-word-limit="true"
          placeholder="请填写备注"/>
    </el-form-item>
  </el-form>
  <el-divider/>
  <el-button
      icon="Plus"
      size="default"
      type="primary"
      plain
      class="add"
      @click="addMaterial"
  >
    添加物料
  </el-button>
  <el-table
      border
      stripe
      :data="outboundMaterials"
  >
    <template #empty>
      <el-empty/>
    </template>
    <el-table-column label="序号" prop="index" width="80px"/>
    <el-table-column label="物料名称" prop="name"/>
    <el-table-column label="物料规格" prop="model"/>
    <el-table-column label="计划数量" prop="estimated_quantity" align="center">
      <template #default="{row}">
        <el-input-number
            v-model="row.estimated_quantity"
            :controls="false"
            :precision="3"
            :min="0.001"
            :step="1"
            @change="computeTotalAmount"
            size="default"
        />
      </template>
    </el-table-column>

    <el-table-column label="单价" prop="price">
      <template #default="{row}">
        <el-popover
            placement="right"
            :title="`[${row.name}] 历史价格：`"
            :width="300"
            trigger="hover"
            @beforeEnter="getPrices(row.id)"
        >
          <template #reference>
            <el-input-number
                v-model="row.price"
                :controls="false"
                :precision="3"
                :step="100"
                :min="0.001"
                size="default"
                @change="computeTotalAmount"/>
          </template>
          <el-tag
              v-if="prices?.length>0"
              v-for="(one, idx) in prices"
              :key="idx"
              class="m-x-1"
              size="default"
              closable
              @click="row.price=one.price"
              @close="removeMaterialPrice(row.id, one.price)"
          >
            {{ one.price }}
          </el-tag>
          <el-text
            v-else
            size="small"
            >暂无</el-text>
        </el-popover>
      </template>
    </el-table-column>
    <el-table-column label="金额">
      <template #default="{row}">
        {{ NP.times(row.price , row.estimated_quantity) }}
      </template>
    </el-table-column>
    <!--    <el-table-column label="实到数量" prop="actual_quantity"/>-->
    <el-table-column label="仓库/库区/货架/货位" width="500px">
      <template #default="{row, col, $index}">
        <el-cascader
            size="default"
            :key="$index"
            :options="warehouses"
            :props="{children:'children', label:'name', value: 'id', checkStrictly: true}"
            v-model="row.warehousing"
            clearable
            style="width: 450px"
            @change="changePositions($event, $index)"
            placeholder="请选择仓储位置"
        >

        </el-cascader>
      </template>
    </el-table-column>
    <!--    <el-table-column label="出库状态">
          <template #default="{row}">
            <el-radio-group v-model="row.status">
              <el-radio-button plain lable="">全部</el-radio-button>
              <el-radio-button v-for="(item, idx) in OutboundReceiptStatus" :key="idx" plain :label="item"/>
            </el-radio-group>
          </template>
        </el-table-column>-->
    <el-table-column label="操作">
      <template #default="{row, column, $index}">
        <el-text
            type="danger"
            size="small"
            @click="removeMaterial($index)"
        >删除
        </el-text>
      </template>
    </el-table-column>
  </el-table>
  <el-row>
    <el-col :span="24" style="text-align: center">
      <el-button
          icon="Plus"
          size="default"
          type="primary"
          plain
          class="add"
          @click="addMaterial"
      >
        添加物料
      </el-button>
    </el-col>
  </el-row>
  <el-row>
    <el-col :span="24" style="text-align: center">
      <el-button
          icon="RefreshLeft"
          size="default"
          plain
          class="add"
          @click="cancel"
      >
        取消
      </el-button>
      <el-button
          icon="Select"
          size="default"
          type="primary"
          plain
          class="add"
          @click="submit"
      >
        保存
      </el-button>
    </el-col>
  </el-row>

  <el-dialog
      v-model="visible"
      title="添加物料"
      width="1400"
      draggable
      :close-on-click-modal="false"
  >
    <el-form
        inline
    >
      <MaterialCategoryListItem
          :form="materialsForm"
      />
      <el-form-item label=" ">
        <el-button type="primary" plain @click="getMaterials" icon="Search">查询</el-button>
      </el-form-item>
    </el-form>
    <el-table
        class="table"
        border
        stripe
        height="640"
        :data="materials"
        @selection-change="handleSelectionChange"
    >
      <el-table-column type="selection" fixed :selectable="selectable"/>
      <el-table-column label="物料名称" prop="name" fixed min-width="180px">
        <template #default="{row}">
          <el-text type="primary" size="default" tag="b" truncated>{{ row.name }}</el-text>
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
      <el-table-column label="备注" prop="remark" min-width="100px"></el-table-column>
    </el-table>

    <!--   分页   -->
    <el-pagination
        v-model:current-page="materialsForm.page"
        v-model:page-size="materialsForm.size"
        @size-change="getMaterials"
        @current-change="getMaterials"
        :page-sizes="[10, 20, 30, 40]"
        background
        layout="total, sizes, prev, pager, next, ->,jumper"
        :pager-count="9"
        :disabled="loading"
        :hide-on-single-page="false"
        :total="total"
    ></el-pagination>
    <template #footer>
      <el-button
          size="default"
          plain
          @click="visible=false"
      >取消
      </el-button>
      <el-button
          size="default"
          type="primary"
          plain
          @click="confirmMaterials"
      >确认
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.w300 {
  width: 300px;
}

.add {
  margin: 10px;
}

.table {
  margin-bottom: 20px;
}
</style>