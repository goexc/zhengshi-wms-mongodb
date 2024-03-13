<!--确认出库单-->
<script setup lang="ts">
import {onMounted, ref} from "vue";
import {
  OutboundOrderMaterial,
  OutboundPickingMaterial,
  OutboundOrderConfirmRequest
} from "@/api/outbound/types.ts";
import {Inventory, InventoryListRequest, InventoryListResponse} from "@/api/inventory/types.ts";
import {reqInventoryList} from "@/api/inventory";
import {ElMessage, ElTable, FormInstance, FormRules} from "element-plus";
import {reqConfirmOutboundOrder, reqOutboundOrderMaterials} from "@/api/outbound";
import {reactive} from "vue";
import {TimeFormat} from "@/utils/time.ts";
import dayjs from "dayjs";

defineOptions({
  name: 'Confirm'
})

let props = defineProps(['code'])
const emit = defineEmits(['success', 'cancel'])
let loading = ref<boolean>(false)

let materials = ref<OutboundOrderMaterial[]>([])
//查询出库单物料列表
let getMaterials = async (order_code: string) => {
  let res = await reqOutboundOrderMaterials({order_code: order_code})
  if (res.code === 200) {
    materials.value = res.data
  } else {
    ElMessage.error(res.msg)
  }
}
onMounted(async () => {
  await getMaterials(props.code)
})

let initInventoryListForm = () => {
  return <InventoryListRequest>{
    material_id: ''
  }
}
//物料库存参数
let form = ref<InventoryListRequest>(initInventoryListForm())
//物料库存列表
let materialsInventorys = new Map<string, Inventory[]>()

//查询物料库存
let getInventorys = async () => {
  let res: InventoryListResponse = await reqInventoryList(form.value)
  materialId.value = form.value.material_id
  if (res.code === 200) {
    // 已经设置发货数量的库存记录，更新inventorys时必须修改发货数量
    let inventorys: Inventory[] = materialsInventorys.get(form.value.material_id) || []
    let inventorys_id = inventorys.map(one => one.id)
    res.data.forEach(one => {
      if (inventorys_id.includes(one.id)) {
        one.shipment_quantity = inventorys.find(item => item.id === one.id)?.shipment_quantity || 0
      }
    })

    materialsInventorys.set(form.value.material_id, res.data)
  } else {
    ElMessage.error(res.msg)
    materialsInventorys.set(form.value.material_id, [])
  }
}

let materialTableRef = ref<InstanceType<typeof ElTable>>()
let inventoryTableRef = ref<InstanceType<typeof ElTable>>()

//当前选中的物料id
let materialId = ref<string>('')

//勾选的物料列表
let selectedMaterials = ref<{ id: string, index: number }[]>([])

//切换物料
let handleMaterialChange = async (item: OutboundOrderMaterial) => {
  //1.请求物料列表
  form.value.material_id = item.material_id
  await getInventorys()
}

//勾选出库物料列表
let handleSelectedMaterials = (materials: OutboundOrderMaterial[]) => {
  //收集物料id、序号
  selectedMaterials.value = materials.map(material => ({
    id: material.material_id,
    index: material.index,
  }))
}

const rules = reactive<FormRules>({
  confirm_time: [
    {
      required: true,
      message: "必填",
      trigger: ["blur", "change"],
    },
    {
      message: '请选择日期',
      type: "number",
      min: 1,
      trigger: ['blur', 'change']
    },
    {
      message: '日期不能超过当前时间',
      type: "number",
      max: dayjs().unix(),
      trigger: ['blur', 'change']
    }
  ],
});

let reqRef = ref<FormInstance>()

//拣货单请求参数
let req = reactive<OutboundOrderConfirmRequest>({
  code: props.code,
  confirm_time: 0,
  materials: []
})

//提交出库单
let submit = async () => {
  //0.批次出库编号校验
  let valid = await reqRef.value?.validate((isValid) => {
    if (!isValid) {
    }
    return
  })
  if (!valid) {
    return
  }

  //1.勾选的物料不能为空
  if (selectedMaterials.value.length === 0) {
    ElMessage.error('请选择物料')
    return
  }

  //2.清除发货数量为0的库存单
  let inventorys = <Inventory[]>[] // 所有入库单
  materialsInventorys.forEach((value, key) => {
    console.log('物料所有库存:', [...value])
    //1.过滤未选择的物料
    // if (!selectedMaterials.value.find(one => one.id === key)) {
    if (!selectedMaterials.value.map(one => one.id).includes(key)) {
      console.log('选择的物料：', selectedMaterials.value.map(one => one.id))
      console.log("没有选择物料:", key)
      return
    }
    //2.过滤发货数量为0的库存单
    inventorys.push(...value.filter((item: Inventory) => item.shipment_quantity && item.shipment_quantity > 0))
  })
  if (inventorys.length === 0) {
    ElMessage.error('没有填写发货数量')
    return
  }

  //3.勾选的物料出货数量>0
  let exist = selectedMaterials.value.every(one => inventorys.map(item => item.material_id).includes(one.id))
  if (!exist) {
    ElMessage.error('部分勾选的物料没有填写发货数量')
    return
  }

  //4.整理数据
  //4.1 记录勾选的物料
  req.materials = []
  selectedMaterials.value.forEach(one => {
    req.materials.push({
      material_id: one.id,
      index: one.index,
      inventorys: [],
    })
  })

  //4.2 确保拣货都是所需物料
  let tag = true
  inventorys.forEach(one => {
    let material = req.materials.find((item: OutboundPickingMaterial) => item.material_id === one.material_id)
    if (!material) {
      tag = false
      ElMessage.error('没有找到对应的物料:' + one.name)
      return
    }

    material.inventorys.push({
      inventory_id: one.id,
      shipment_quantity: one.shipment_quantity
    })
  })

  if (!tag) {
    return
  }

  loading.value = true
  let res = await reqConfirmOutboundOrder(req)
  loading.value = false
  if (res.code === 200) {
    ElMessage.success(res.msg)
    emit('success')
  } else {
    ElMessage.error(res.msg)
  }
}

</script>

<template>
  <div>
    <el-form
        inline
        size="default"
        ref="reqRef"
        :model="req"
    >
      <el-form-item label="出库单号" prop="code">
        <el-text size="default">{{ req.code }}</el-text>
      </el-form-item>
    </el-form>
    <div style="display: flex">
      <el-table
          ref="materialTableRef"
          border
          size="default"
          style="flex: 1.1;"
          height="700px"
          :data="materials"
          highlight-current-row
          @current-change="handleMaterialChange"
          @selection-change="handleSelectedMaterials"
      >
        <template #empty>
          <el-empty/>
        </template>
        <el-table-column type="selection" width="45" align="center"/>
        <el-table-column label="序号" prop="index" width="60px" align="center"/>
        <el-table-column prop="name" label="物料">
          <template #default="{row}">
            <el-text tag="b" size="default">{{ row.name }}</el-text>
            <br/>
            <el-text size="small" tag="i">({{ row.model }})</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="price" label="单价" align="center" width="90px"/>
        <el-table-column prop="name" label="剩余数量" width="90px">
          <template #default="{row}">
            <el-text size="small" tag="b" type="danger">{{ row.quantity }} {{ row.unit }}</el-text>
          </template>
        </el-table-column>
      </el-table>
      <!--  间隔  -->
      <div style="flex: 0.1"></div>
      <el-table
          ref="inventoryTableRef"
          border
          stripe
          height="700px"
          size="default"
          style="flex: 3;"
          :data="materialsInventorys.get(materialId)"
      >
        <template #empty>
          <el-empty/>
        </template>
        <el-table-column type="index" label="#" width="45" align="center"/>
        <el-table-column prop="type" label="入库类型" align="center"/>
        <el-table-column prop="time" label="入库时间" align="center" width="164px">
          <template #default="{row}">
            {{ TimeFormat(row.entry_time) }}
          </template>
        </el-table-column>
        <!--    <el-table-column prop="receipt_code" label="入库单编号"/>-->
        <!--    <el-table-column prop="receive_code" label="批次编号"/>-->
        <el-table-column prop="warehouse_name" label="仓库">
          <template #default="{row}">
            <el-text v-if="row.warehouse_name.length>0" size="default">{{ row.warehouse_name }}</el-text>
            <el-text v-else size="default">-</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="warehouse_zone_name" label="库区">
          <template #default="{row}">
            <el-text v-if="row.warehouse_zone_name.length>0" size="default">{{ row.warehouse_zone_name }}</el-text>
            <el-text v-else size="default">-</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="warehouse_rack_name" label="货架">
          <template #default="{row}">
            <el-text v-if="row.warehouse_rack_name.length>0" size="default">{{ row.warehouse_rack_name }}</el-text>
            <el-text v-else size="default">-</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="warehouse_bin_name" label="货位">
          <template #default="{row}">
            <el-text v-if="row.warehouse_bin_name.length>0" size="default">{{ row.warehouse_bin_name }}</el-text>
            <el-text v-else size="default">-</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="quantity" label="库存数量">
          <template #default="{row}">
            <el-text type="primary" size="default">{{ row.quantity }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="frozen_quantity" label="冻结数量">
          <template #default="{row}">
            <el-text type="danger" size="default">{{ row.frozen_quantity }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="locked_quantity" label="锁定数量">
          <template #default="{row}">
            <el-text type="warning" size="default">{{ row.locked_quantity }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="available_quantity" label="可用数量">
          <template #default="{row}">
            <el-text type="success" size="default">{{
                row.available_quantity - row.frozen_quantity - row.locked_quantity
              }}
            </el-text>
          </template>
        </el-table-column>
        <el-table-column label="发货数量" width="180px" align="center">
          <template #default="{row}">
            <el-input-number
                v-model.trim="row.shipment_quantity"
                :controls="false"
                :precision="3"
                :min="0"
                :max="row.quantity-row.frozen_quantity-row.locked_quantity"
                :value-on-clear="0"
                :step="1"
                size="small"
            />
            <el-text type="primary" size="small" @click="row.shipment_quantity=row.available_quantity - row.frozen_quantity - row.locked_quantity">全部</el-text>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <div class="m-t-2">
      <el-form
      inline
      :model="req"
      ref="reqRef"
      :rules="rules"
      >
        <el-form-item label="确认出库单日期" prop="confirm_time">
          <el-date-picker
              v-model.number="req.confirm_time"
              type="date"
              placeholder="请选择确认出库单日期"
              size="default"
              value-format="X"
          />
        </el-form-item>
        <el-form-item >

          <el-button type="primary" size="default" plain :disabled="loading" @click="submit">确认拣货清单</el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<style scoped lang="scss">

</style>