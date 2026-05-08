<!-- 极速出库 -->
<script setup lang="ts">
import { reactive, ref } from "vue";
import dayjs from "dayjs";
import NP from "number-precision";
import { ElMessage, FormInstance, FormRules } from "element-plus";
import { CustomerOutbondReceiptTypes } from "@/enums/outbound.ts";
import { reqFastDepartureOutboundOrder } from "@/api/outbound";
import { FastOutboundRequest } from "@/api/outbound/types.ts";
import { Material, MaterialPrice, MaterialsRequest } from "@/api/material/types.ts";
import { reqMaterialPrices, reqMaterials, reqRemoveMaterialPrice } from "@/api/material";
import { OutboundOrderMaterial } from "@/api/outbound/types.ts";
import MaterialCategoryListItem from "@/components/MaterialCategory/MaterialCategoryListItem.vue";
import CustomerListItem from "@/components/Customer/CustomerListItem.vue";
import { DateFormat } from "@/utils/time.ts";
import { nameEncrypt } from "@/utils/name_encrypt.ts";

defineOptions({
  name: "FastOutbound"
});

const emit = defineEmits(["success"]);

const nowSeconds = () => Math.floor(Date.now() / 1000);

const initFastForm = (): FastOutboundRequest => {
  const now = nowSeconds();
  return {
    code: `O-${dayjs().format("YYYYMM")}-`,
    type: "销售出库",
    customer_id: "",
    departure_time: now,
    picking_time: now,
    packing_time: undefined,
    weighing_time: undefined,
    receipt_time: now,
    materials: []
  };
};

type MaterialQueryForm = MaterialsRequest & { category_id: string };

const initMaterialsForm = (): MaterialQueryForm => ({
  page: 1,
  size: 10,
  name: "",
  category_id: "",
  image: "",
  model: "",
  material: "",
  specification: "",
  surface_treatment: "",
  strength_grade: ""
});

const dialogVisible = ref(false);
const materialDialogVisible = ref(false);
const submitting = ref(false);
const formRef = ref<FormInstance>();
const fastForm = reactive<FastOutboundRequest>(initFastForm());
const outboundMaterials = ref<OutboundOrderMaterial[]>([]);
const materialsForm = ref<MaterialQueryForm>(initMaterialsForm());
const materials = ref<Material[]>([]);
const materialsTotal = ref(0);
const materialsLoading = ref(false);
const prices = ref<MaterialPrice[]>([]);
const ossDomain = ref<string>(import.meta.env.VITE_OSS_DOMAIN as string);

const rules = reactive<FormRules>({
  code: [{ required: true, message: "出库单号不能为空", trigger: "blur" }],
  type: [{ required: true, message: "请选择出库类型", trigger: "change" }],
  customer_id: [{ required: true, message: "请选择客户", trigger: "change" }],
  picking_time: [{ required: true, message: "请选择拣货日期", trigger: "change" }],
  departure_time: [{ required: true, message: "请选择出库日期", trigger: "change" }]
});

const openDialog = () => {
  resetFastForm();
  dialogVisible.value = true;
};

const resetFastForm = () => {
  Object.assign(fastForm, initFastForm());
  outboundMaterials.value = [];
  prices.value = [];
};

const resetMaterials = async () => {
  materialsForm.value = initMaterialsForm();
  await getMaterials();
};

const getMaterials = async () => {
  materialsLoading.value = true;
  const res = await reqMaterials(materialsForm.value);
  materialsLoading.value = false;
  if (res.code === 200) {
    materials.value = res.data.list;
    materialsTotal.value = res.data.total;
  } else {
    materials.value = [];
    materialsTotal.value = 0;
    ElMessage.error(res.msg);
  }
};

const openMaterialDialog = async () => {
  await getMaterials();
  materialDialogVisible.value = true;
};

const resortMaterials = () => {
  outboundMaterials.value.forEach((item, idx) => {
    item.index = idx + 1;
  });
};

const pushMaterial = (row: Material) => {
  if (outboundMaterials.value.find(one => one.material_id === row.id)) return;

  outboundMaterials.value.push({
    index: outboundMaterials.value.length + 1,
    id: "",
    material_id: row.id,
    name: row.name,
    model: row.model,
    specification: row.specification,
    price: 0,
    quantity: 1,
    unit: row.unit,
    weight: 0,
    returned_quantity: 0
  });
};

const popMaterial = (row: Material) => {
  outboundMaterials.value = outboundMaterials.value.filter(item => item.material_id !== row.id);
  resortMaterials();
};

const removeMaterial = (idx: number) => {
  outboundMaterials.value.splice(idx, 1);
  resortMaterials();
};

const getPrices = async (materialId: string) => {
  if (!fastForm.customer_id) {
    ElMessage.warning("请选择客户");
    return;
  }

  prices.value = [];
  const res = await reqMaterialPrices(materialId, fastForm.customer_id);
  if (res.code === 200) {
    prices.value = (res.data || []).sort((a: MaterialPrice, b: MaterialPrice) => a.since - b.since);
  } else {
    ElMessage.error(res.msg);
  }
};

const removeMaterialPrice = async (id: string, customerId: string, price: number) => {
  const res = await reqRemoveMaterialPrice(id, customerId, price);
  if (res.code === 200) {
    ElMessage.success(res.msg);
    await getPrices(id);
  } else {
    ElMessage.error(res.msg);
  }
};

const validateTimeChain = () => {
  const now = nowSeconds();
  const departureTime = Number(fastForm.departure_time);
  const pickingTime = Number(fastForm.picking_time || departureTime);
  const receiptTime = Number(fastForm.receipt_time || now);
  const chain = [
    { label: "拣货时间", value: pickingTime },
    ...(fastForm.packing_time ? [{ label: "打包时间", value: Number(fastForm.packing_time) }] : []),
    ...(fastForm.weighing_time ? [{ label: "称重时间", value: Number(fastForm.weighing_time) }] : []),
    { label: "出库时间", value: departureTime },
    { label: "签收时间", value: receiptTime }
  ];

  for (const one of chain) {
    if (one.value > now) {
      ElMessage.warning(`${one.label}不能超过当前时间`);
      return false;
    }
  }

  for (let i = 1; i < chain.length; i++) {
    if (chain[i].value < chain[i - 1].value) {
      ElMessage.warning(`${chain[i].label}不能早于${chain[i - 1].label}`);
      return false;
    }
  }
  return true;
};

const validateMaterials = () => {
  if (outboundMaterials.value.length === 0) {
    ElMessage.warning("请添加至少一项物料明细");
    return false;
  }

  for (let i = 0; i < outboundMaterials.value.length; i++) {
    const item = outboundMaterials.value[i];
    if (!item.material_id) {
      ElMessage.warning(`第 ${i + 1} 行物料为空`);
      return false;
    }
    if (Number(item.quantity) <= 0) {
      ElMessage.warning(`第 ${i + 1} 行物料数量必须大于0`);
      return false;
    }
    if (Number(item.price) < 0) {
      ElMessage.warning(`第 ${i + 1} 行物料价格不能小于0`);
      return false;
    }
  }
  return true;
};

const totalAmount = () => {
  return outboundMaterials.value.reduce((total, current) => total + NP.times(current.price, current.quantity), 0);
};

const submit = async () => {
  if (!formRef.value || submitting.value) return;

  const valid = await formRef.value.validate();
  if (!valid || !validateMaterials() || !validateTimeChain()) return;
  if (!CustomerOutbondReceiptTypes.includes(fastForm.type)) {
    ElMessage.warning("极速出库只允许客户应收类出库");
    return;
  }

  const departureTime = Number(fastForm.departure_time);
  const payload: FastOutboundRequest = {
    ...fastForm,
    code: fastForm.code.trim(),
    departure_time: departureTime,
    picking_time: Number(fastForm.picking_time || departureTime),
    packing_time: fastForm.packing_time ? Number(fastForm.packing_time) : undefined,
    weighing_time: fastForm.weighing_time ? Number(fastForm.weighing_time) : undefined,
    receipt_time: fastForm.receipt_time ? Number(fastForm.receipt_time) : undefined,
    materials: outboundMaterials.value.map(item => ({
      material_id: item.material_id,
      price: Number(item.price || 0),
      quantity: Number(item.quantity)
    }))
  };

  try {
    submitting.value = true;
    const res = await reqFastDepartureOutboundOrder(payload);
    if (res.code === 200) {
      ElMessage.success(res.msg || "极速出库成功");
      dialogVisible.value = false;
      emit("success");
    } else {
      ElMessage.error(res.msg || "极速出库失败");
    }
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <el-button icon="Promotion" size="default" type="warning" plain @click="openDialog">极速出库</el-button>

  <el-dialog
      v-model="dialogVisible"
      title="极速出库"
      fullscreen
      draggable
      destroy-on-close
      :close-on-click-modal="false"
  >
    <el-form
        ref="formRef"
        class="fast-form"
        :model="fastForm"
        :rules="rules"
        size="default"
        label-width="100px"
    >
      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="出库单号" prop="code">
            <el-input v-model.trim="fastForm.code" class="w300"/>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="出库类型" prop="type">
            <el-radio-group v-model.trim="fastForm.type">
              <el-radio-button
                  v-for="(item, idx) in CustomerOutbondReceiptTypes"
                  :key="idx"
                  plain
                  :label="item"
              />
            </el-radio-group>
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <CustomerListItem :form="fastForm"/>
        </el-col>
      </el-row>

      <el-row :gutter="20">
        <el-col :span="8">
          <el-form-item label="拣货时间" prop="picking_time">
            <el-date-picker
                v-model.number="fastForm.picking_time"
                type="date"
                value-format="X"
                placeholder="请选择拣货日期"
                style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="打包时间" prop="packing_time">
            <el-date-picker
                v-model.number="fastForm.packing_time"
                type="date"
                value-format="X"
                placeholder="可选"
                style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="称重时间" prop="weighing_time">
            <el-date-picker
                v-model.number="fastForm.weighing_time"
                type="date"
                value-format="X"
                placeholder="可选"
                style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="出库时间" prop="departure_time">
            <el-date-picker
                v-model.number="fastForm.departure_time"
                type="date"
                value-format="X"
                placeholder="请选择出库日期"
                style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="签收时间" prop="receipt_time">
            <el-date-picker
                v-model.number="fastForm.receipt_time"
                type="date"
                value-format="X"
                placeholder="默认当前日期"
                style="width: 100%"
            />
          </el-form-item>
        </el-col>
        <el-col :span="8">
          <el-form-item label="总金额">
            <el-input-number
                :model-value="totalAmount()"
                class="w300"
                :controls="false"
                :precision="3"
                disabled
            />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>

    <el-divider/>

    <el-button
        icon="CirclePlus"
        size="default"
        type="primary"
        plain
        class="add"
        @click="openMaterialDialog"
    >
      添加物料
    </el-button>

    <el-table border stripe :data="outboundMaterials">
      <template #empty>
        <el-empty/>
      </template>
      <el-table-column label="序号" prop="index" width="80px"/>
      <el-table-column label="物料名称" prop="name" min-width="180px"/>
      <el-table-column label="型号" prop="model" min-width="160px"/>
      <el-table-column label="规格" prop="specification" min-width="160px"/>
      <el-table-column label="数量" prop="quantity" align="center" width="220px">
        <template #default="{ row }">
          <el-input-number
              v-model.trim="row.quantity"
              :controls="false"
              :precision="3"
              :min="1"
              :step="1"
              size="default"
          />
          {{ row.unit }}
        </template>
      </el-table-column>
      <el-table-column label="出库价" prop="price" align="center" width="260px">
        <template #default="{ row }">
          <el-popover
              placement="right"
              :title="`[${row.name}] 历史价格：`"
              :width="300"
              trigger="hover"
              @beforeEnter="getPrices(row.material_id)"
          >
            <template #reference>
              <el-input-number
                  v-model.trim="row.price"
                  :controls="false"
                  :precision="3"
                  :step="100"
                  :min="0"
                  size="default"
              />
            </template>
            <el-tag
                v-if="prices?.length > 0"
                v-for="(one, idx) in prices"
                :key="idx"
                class="m-x-1 price-tag"
                size="default"
                closable
                @click="row.price = one.price"
                @close="removeMaterialPrice(row.material_id, fastForm.customer_id, one.price)"
            >
              {{ one.price }} ({{ nameEncrypt(one.customer_name) }}) [{{ DateFormat(one.since) }}]
            </el-tag>
            <el-text v-else size="small">暂无</el-text>
          </el-popover>
        </template>
      </el-table-column>
      <el-table-column label="金额" width="140px">
        <template #default="{ row }">
          {{ NP.times(row.price, row.quantity) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100px" align="center">
        <template #default="{ $index }">
          <el-text type="danger" size="small" @click="removeMaterial($index)">删除</el-text>
        </template>
      </el-table-column>
    </el-table>

    <el-row>
      <el-col :span="24" class="footer-actions">
        <el-button icon="RefreshLeft" plain @click="dialogVisible = false">取消</el-button>
        <el-button icon="Select" type="primary" plain :loading="submitting" @click="submit">极速出库</el-button>
      </el-col>
    </el-row>

    <el-dialog
        v-model.trim="materialDialogVisible"
        title="添加物料"
        width="1800px"
        draggable
        :close-on-click-modal="false"
        top="0vh"
    >
      <el-form class="material-search-form" inline label-width="80px">
        <MaterialCategoryListItem :form="materialsForm"/>
        <el-form-item label="名称" prop="name">
          <el-input v-model.trim="materialsForm.name" clearable placeholder="请填写名称"/>
        </el-form-item>
        <el-form-item label="型号" prop="model">
          <el-input v-model.trim="materialsForm.model" clearable placeholder="请填写型号"/>
        </el-form-item>
        <el-form-item label="规格" prop="specification">
          <el-input v-model.trim="materialsForm.specification" clearable placeholder="请填写规格"/>
        </el-form-item>
        <el-form-item label="材质" prop="material">
          <el-input v-model.trim="materialsForm.material" clearable placeholder="请填写材质"/>
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
          <el-button plain @click="resetMaterials" icon="RefreshRight">重置</el-button>
        </el-form-item>
      </el-form>

      <el-table
          class="table"
          border
          stripe
          size="default"
          height="640"
          :data="materials"
          v-loading="materialsLoading"
      >
        <el-table-column label="操作" fixed align="center" width="90px">
          <template #default="{ row }">
            <el-link
                v-if="!outboundMaterials.map(item => item.material_id).includes(row.id)"
                type="primary"
                icon="Check"
                @click="pushMaterial(row)"
            >
              选择
            </el-link>
            <el-link v-else type="danger" icon="Delete" @click="popMaterial(row)">剔除</el-link>
          </template>
        </el-table-column>
        <el-table-column label="物料名称" prop="name" fixed min-width="180px">
          <template #default="{ row }">
            <el-text type="primary" size="default" tag="b" truncated>{{ row.name }}</el-text>
          </template>
        </el-table-column>
        <el-table-column label="物料图片" width="150px" align="center">
          <template #default="{ row }">
            <el-image
                v-if="row.image.endsWith('.svg')"
                class="image"
                fit="contain"
                :src="`${ossDomain}${row.image}`"
                :preview-src-list="[`${ossDomain}${row.image}`]"
                hide-on-click-modal
                preview-teleported
            />
            <el-image
                v-else-if="row.image"
                class="image"
                fit="contain"
                :src="`${ossDomain}${row.image}_148x148`"
                :preview-src-list="[`${ossDomain}${row.image}`]"
                hide-on-click-modal
                preview-teleported
            />
          </template>
        </el-table-column>
        <el-table-column label="型号" prop="model" min-width="180px"/>
        <el-table-column label="分类" prop="category_name" min-width="60px" align="center"/>
        <el-table-column label="材质" prop="material" min-width="100px"/>
        <el-table-column label="规格" prop="specification" min-width="160px"/>
        <el-table-column label="表面处理" prop="surface_treatment" min-width="100px"/>
        <el-table-column label="强度等级" prop="strength_grade" min-width="100px"/>
        <el-table-column label="安全库存" prop="quantity" min-width="60px" align="center"/>
        <el-table-column label="计量单位" prop="unit" min-width="60px" align="center"/>
        <el-table-column label="备注" prop="remark" min-width="100px"/>
      </el-table>

      <el-pagination
          v-model:current-page="materialsForm.page"
          v-model:page-size="materialsForm.size"
          @size-change="getMaterials"
          @current-change="getMaterials"
          :page-sizes="[10, 20, 30, 40]"
          background
          layout="total, sizes, prev, pager, next, ->,jumper"
          :pager-count="9"
          :disabled="materialsLoading"
          :hide-on-single-page="false"
          :total="materialsTotal"
      />
    </el-dialog>
  </el-dialog>
</template>

<style scoped lang="scss">
.w300 {
  width: 300px;
}

.add {
  margin: 10px;
}

.fast-form,
.material-search-form {
  :deep(.el-form-item) {
    margin-bottom: 18px;
  }
}

.table {
  margin-bottom: 20px;
}

.image {
  width: 96px;
  height: 96px;
}

.price-tag {
  cursor: pointer;
}

.footer-actions {
  margin-top: 20px;
  text-align: center;
}
</style>
