<script setup lang="ts">

import {onMounted, ref} from "vue";
import useClipboard from 'vue-clipboard3'
import {Inventory, InventorysRequest} from "@/api/inventory/types.ts";
import {InboundReceiptTypes} from "@/enums/inbound.ts";
import RackPageItem from "@/components/WarehouseRack/RackPageItem.vue";
import WarehousePageItem from "@/components/Warehouse/WarehousePageItem.vue";
import ZonePageItem from "@/components/WarehouseZone/ZonePageItem.vue";
import BinPageItem from "@/components/WarehouseBin/BinPageItem.vue";
import {reqInventory} from "@/api/inventory";
import {ElMessage} from "element-plus";
import {TimeFormat} from "@/utils/time.ts";
import dayjs from "dayjs";

const initForm = () => {
  return <InventorysRequest>{
    page: 1,
    size: 20,
    type: '',
    material_name: '',
    material_model: '',
    warehouse_id: '',
    warehouse_zone_id: '',
    warehouse_rack_id: '',
    warehouse_bin_id: '',
  }
}
const form = ref<InventorysRequest>(initForm());

let total = ref<number>(0)//入库记录数量
let quantity = ref<number>(0)//物料库存总数量
let list = ref<Inventory[]>([])
let loading = ref<boolean>(false)


//查询库存
let getInventorys = async () => {
  loading.value = true

  let res = await reqInventory(form.value);
  loading.value = false
  if (res.code === 200) {
    total.value = res.data.total
    quantity.value = res.data.quantity
    list.value = res.data.list
  } else {
    list.value = []
    total.value = 0
    quantity.value = 0
    ElMessage.error(res.msg)
  }
}

//重置表单
let reset = async () => {
  form.value = await initForm()
  await getInventorys()
}

onMounted(async () => {
  await getInventorys()
})

const {toClipboard} = useClipboard()

//复制到剪贴板
let copyText = (text: string) => {
  toClipboard(text)
  ElMessage({
    message: '复制成功',
    grouping: true,
    type: 'success'
  })
}

//入库类型
let inboundType = (text: string) => {
  switch (text) {
    case '采购入库':
      return 'success'
    case '外协入库':
      return ''
    case '退货入库':
      return 'danger'
    default:
      return 'warning'
  }
}

</script>

<template>
  <div>
    <el-card>
      <el-form
          :model="form"
          inline
          label-width="80px"
          size="default"
          style="display: flex; flex-wrap: wrap;"
      >
        <el-form-item
            label="入库类型"
        >
          <el-radio-group v-model.trim="form.type">
            <el-radio-button plain label="">全部</el-radio-button>
            <el-radio-button v-for="(item, idx) in InboundReceiptTypes" :key="idx" plain :label="item"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="物料名称">
          <el-input v-model.trim="form.material_name" clearable placeholder="请填写物料名称"/>
        </el-form-item>
        <el-form-item label="物料型号">
          <el-input v-model.trim="form.material_model" clearable placeholder="请填写物料型号"/>
        </el-form-item>
        <WarehousePageItem
            :form="form"
        />
        <ZonePageItem
            :form="form"
        />
        <RackPageItem
            :form="form"
        />
        <BinPageItem
            :form="form"
        />
        <el-form-item label="">
          <el-button
              type="primary"
              plain
              icon="Search"
              size="default"
              @click="getInventorys()"
          >查询
          </el-button>
          <el-button plain @click="reset" icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <div class="m-t-2 m-b-1">
      <el-text type="success" size="default">库存总数量：{{ quantity }}</el-text>
    </div>
    <el-table
        border
        stripe
        :data="list"
    >
      <template #empty>
        <el-empty/>
      </template>
      <el-table-column prop="name" label="物料名称" width="250px">
        <template #default="{row}">
          <span @click="copyText(row.name)">{{ row.name }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="model" label="型号" width="250px">
        <template #default="{row}">
          <span @click="copyText(row.model)">{{ row.model }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="type" label="入库类型" width="110px" align="center">
        <template #default="{row}">
          <el-tag size="default" :type="inboundType(row.type)">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="unit" label="计量单位" width="70px" align="center"/>
      <el-table-column prop="price" label="物料单价" width="100px" align="right">
        <template #default="{row}">
          <el-text underline type="primary" size="default">{{ row.price.toFixed(3) }}</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="quantity" label="入库数量" width="90px" align="right"/>
      <el-table-column prop="available_quantity" label="可用库存数量" width="80px" align="right"/>
      <el-table-column prop="locked_quantity" label="锁定库存数量" width="80px" align="right"/>
      <el-table-column prop="frozen_quantity" label="冻结库存数量" width="80px" align="right"/>
      <el-table-column prop="receipt_code" label="入库单编号" width="280px" align="center">
        <template #default="{row}">
          <el-link underline type="primary" @click="copyText(row.receipt_code)">{{ row.receipt_code }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="receive_code" label="批次入库编号" width="280px" align="center">
        <template #default="{row}">
          <el-link underline type="success" @click="copyText(row.receive_code)">{{ row.receive_code }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="warehouse_name" label="仓库名称" width="140px">
        <template #default="{row}">
          <el-text v-if="row.warehouse_name.length>0" size="default">{{ row.warehouse_name }}</el-text>
          <el-text v-else size="default">/</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="warehouse_zone_name" label="库区名称" width="140px">
        <template #default="{row}">
          <el-text v-if="row.warehouse_zone_name.length>0" size="default">{{ row.warehouse_zone_name }}</el-text>
          <el-text v-else size="default">/</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="warehouse_rack_name" label="货架名称" width="140px">
        <template #default="{row}">
          <el-text v-if="row.warehouse_rack_name.length>0" size="default">{{ row.warehouse_rack_name }}</el-text>
          <el-text v-else size="default">/</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="warehouse_bin_name" label="货位名称" width="140px">
        <template #default="{row}">
          <el-text v-if="row.warehouse_bin_name.length>0" size="default">{{ row.warehouse_bin_name }}</el-text>
          <el-text v-else size="default">/</el-text>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="入库时间" width="180px">
        <template #default="{row}">
          {{ TimeFormat(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="库存天数" width="70px" align="center">
        <template #default="{row}">
          {{ ((dayjs().unix() - row.created_at) / (3600 * 24)).toFixed(1) }}
        </template>
      </el-table-column>

      <el-table-column label="操作" width="100px" align="center" fixed="right">
<!--        <template #default="{row}">-->
        <template>
          <el-button size="small" type="warning" plain icon="WarnTriangleFilled">盘点</el-button>
        </template>
      </el-table-column>
    </el-table>
    <!--   分页   -->
    <el-pagination
        class="m-t-2"
        v-model:current-page="form.page"
        v-model:page-size="form.size"
        @size-change="getInventorys()"
        @current-change="getInventorys()"
        :page-sizes="[20, 30, 40, 50, 100]"
        background
        layout="total, sizes, prev, pager, next, ->,jumper"
        :pager-count="9"
        :disabled="loading"
        :hide-on-single-page="false"
        :total="total"
    ></el-pagination>
  </div>
</template>

<style scoped lang="scss">


</style>