<script setup lang="ts">
import {InboundReceiptTypes} from "@/enums/inbound.ts";
import {onMounted, reactive, ref} from "vue";
import {ElMessage, FormInstance, FormRules} from "element-plus";
import {InboundMaterial, InboundReceipt} from "@/api/inbound/types.ts";
import MaterialCategoryListItem from "@/components/MaterialCategory/MaterialCategoryListItem.vue";
import {Material, MaterialPrice, MaterialsRequest} from "@/api/material/types.ts";
import {reqMaterialPrices, reqMaterials, reqRemoveMaterialPrice} from "@/api/material";
import {reqAddOrUpdateInboundReceipt} from "@/api/inbound";
import CustomerListItem from "@/components/Customer/CustomerListItem.vue";
import dayjs from "dayjs";
import NP from 'number-precision'
import SupplierListItem from "@/components/Supplier/SupplierListItem.vue";

defineOptions({
  name: 'Order'
})
const emit = defineEmits(['success', 'cancel'])

let props = defineProps(['form', 'action'])
//接收属性中的数据
let receipt = ref<InboundReceipt>({
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


//入库单物料列表
let inboundMaterials = ref<InboundMaterial[]>([])

//删除物料
let removeMaterial = (idx: number) => {
  console.log('删除物料：', idx)
  inboundMaterials.value.splice(idx, 1)
  inboundMaterials.value.forEach((item, idx) => {
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

//重置物料查询表单
let reset = async () => {
  materialsForm.value = initMaterialsForm()
  await getMaterials()
}

//物料是否可选
let selectable = (row:any, rowIndex:any) => {
  console.log('rowIndex:',rowIndex)
  return !inboundMaterials.value.some(item => item.id === row.id);
}

//添加物料
let addMaterial = async () => {
  await getMaterials()

  visible.value = true
}

//选择物料
let selectedMaterials = ref<Material[]>([])
let handleSelectionChange = (val: Material[]) => {
  selectedMaterials.value = val
}

//确认物料
let confirmMaterials = () => {
  let index: number = 0
  if (!!inboundMaterials.value && inboundMaterials.value.length > 0) {
    index = inboundMaterials.value[inboundMaterials.value.length - 1].index
  }
  selectedMaterials.value.forEach((item) => {
    //检测是否已存在该物料
    if(inboundMaterials.value.find(one=>one.id===item.id)){
      console.log('物料已存在：', item.name, item.model)
      return
    }

    inboundMaterials.value.push({
      index: ++index,
      id: item.id,
      name: item.name,
      model: item.model,
      price: 0,
      estimated_quantity: 0,
      actual_quantity: 0,
      unit: item.unit,
      status: '',
      position: [],
      // warehouse_id: '',
      // warehouse_zone_id: '',
      // warehouse_rack_id: '',
      // warehouse_bin_id: '',
    })
  })

  // visible.value = false
}

//查询物料价格列表
let prices = ref<MaterialPrice[]>([])
let getPrices = async (id: string) => {
  prices.value = []
  let res = await reqMaterialPrices(id, '')
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
  receipt.value.total_amount = inboundMaterials.value.reduce((total, current) => {
    return total + NP.times(current.price, current.estimated_quantity);
  }, 0)
}

//删除物料价格
let removeMaterialPrice = async (id: string, customer_id:string, price: number) => {
  let res = await reqRemoveMaterialPrice(id, customer_id, price)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    await getPrices(id)
  } else {
    ElMessage.error(res.msg)
  }
}


//关闭表单
const cancel = () => {
  emit('cancel')
}

let validTotalAmount = (rule:any, value:any, callback:any) => {
  console.log('rule:', rule)
  if (value >= 0) {
    callback(); //通过验证
  } else {
    callback(new Error('总金额必须≥0'))
  }
}

let rules = reactive<FormRules>({
  code: [
    {required: true, message: '入库单号不能为空'}
  ],
  type: [
    {required: true, message: '请选择入库类型'}
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

  // props.form.materials = inboundMaterials.value
  receipt.value!.materials = inboundMaterials.value
  //提交数据
  let res = await reqAddOrUpdateInboundReceipt(receipt.value!)
  if (res.code === 200) {
    ElMessage.success(res.msg)
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

onMounted(async () => {
  //1.初始化入库单编号
  receipt.value = JSON.parse(JSON.stringify(props.form))
  if (receipt.value.code.length === 0) {
    receipt.value.code = 'I-' + dayjs().format('YYYY-MM-DD-HH-mm-ss-SSS')
  }

  //1.接收入库单物料列表
  inboundMaterials.value = JSON.parse(JSON.stringify(props.form.materials.sort((a:InboundMaterial, b:InboundMaterial) => {
    // 根据需要的排序逻辑进行比较
    return a.index - b.index
  })))
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
        label="入库单号"
        prop="code"
    >
      <el-input
          v-model.trim="receipt.code"
          class="w300"
          :disabled="action==='edit'"
      />
    </el-form-item>
    <el-form-item label="入库类型" prop="type">
      <el-radio-group v-model.trim="receipt.type">
        <el-radio-button v-for="(item, idx) in InboundReceiptTypes" :key="idx" plain :label="item"/>
      </el-radio-group>
    </el-form-item>
    <SupplierListItem
        v-if="['采购入库','外协入库'].includes(receipt.type)"
        :form="receipt"
    />
    <CustomerListItem
        v-if="receipt.type ==='退货入库'"
        :form="receipt"
    />
    <el-form-item
        label="总金额"
        prop="total_amount"
    >
      <el-input-number
          v-model.trim="receipt.total_amount"
          class="w300"
          :controls="false"
          :step="100"
          :precision="3"
          :min="0"
      />
    </el-form-item>
    <el-form-item label="备注" prop="remark">
      <el-input
          v-model.trim="receipt.remark"
          type="textarea"
          rows="3"
          maxlength="125"
          :show-word-limit="true"
          placeholder="请填写备注"/>
    </el-form-item>
  </el-form>
  <el-divider/>
  <el-button
      icon="CirclePlus"
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
      :data="inboundMaterials"
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
            v-model.trim="row.estimated_quantity"
            :controls="false"
            :precision="3"
            :min="1"
            :step="1"
            @change="computeTotalAmount"
            size="default"
        />
        {{ row.unit }}
      </template>
    </el-table-column>

    <el-table-column label="入库价" prop="price">
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
                v-model.trim="row.price"
                :controls="false"
                :precision="3"
                :step="100"
                :min="0"
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
              @click="()=>{row.price=one.price;computeTotalAmount()}"
              @close="removeMaterialPrice(row.material_id, receipt.customer_id, one.price)"
          >
            {{ one.price }}
          </el-tag>
          <el-text
              v-else
              size="small"
          >暂无
          </el-text>
        </el-popover>
      </template>
    </el-table-column>
    <el-table-column label="金额">
      <template #default="{row}">
        {{ NP.times(row.price, row.estimated_quantity) }}
      </template>
    </el-table-column>
    <el-table-column label="操作">
      <template #default="{row, column, $index}">
        <el-text :hidden="true">{{row}}{{column}}</el-text>
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
          icon="CirclePlus"
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
      v-model.trim="visible"
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
      <el-form-item label="名称" prop="name">
        <el-input v-model.trim="materialsForm.name" clearable placeholder="请填写名称"/>
      </el-form-item>
      <el-form-item label="型号" prop="model">
        <el-input v-model.trim="materialsForm.model" clearable placeholder="请填写型号"/>
      </el-form-item>
      <el-form-item label="材质" prop="material">
        <el-input v-model.trim="materialsForm.material" clearable placeholder="请填写材质"/>
      </el-form-item>
      <el-form-item label="规格" prop="specification">
        <el-input v-model.trim="materialsForm.specification" clearable placeholder="请填写规格"/>
      </el-form-item>
      <br/>
      <el-form-item label="表面处理" prop="surface_treatment">
        <el-input v-model.trim="materialsForm.surface_treatment" clearable placeholder="请选择表面处理"/>
      </el-form-item>
      <el-form-item label="强度等级" prop="strength_grade">
        <el-input v-model.trim="materialsForm.strength_grade" clearable placeholder="请选择强度等级"/>
      </el-form-item>
      <el-form-item label=" ">
        <el-button type="primary" plain @click="getMaterials" icon="Search">查询</el-button>
        <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
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