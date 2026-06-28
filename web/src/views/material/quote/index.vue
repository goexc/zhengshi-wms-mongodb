<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import dayjs from "dayjs";
import { ElMessage, ElMessageBox } from "element-plus";
import { Customer } from "@/api/customer/types.ts";
import { reqCustomerList } from "@/api/customer";
import {
  MaterialDeliveryRebuildTask,
  NewCustomerMaterialExportRequest,
  NewCustomerMaterialItem,
  NewCustomerMaterialRequest,
  QuoteStatus,
} from "@/api/material/types.ts";
import {
  reqExportNewCustomerMaterialQuotes,
  reqLatestMaterialDeliveryRebuildTask,
  reqNewCustomerMaterials,
  reqRebuildNewCustomerMaterials,
} from "@/api/material";
import { DateFormat } from "@/utils/time.ts";
import MaterialQuoteDialog from "./components/MaterialQuoteDialog.vue";
import RebuildTaskDialog from "./components/RebuildTaskDialog.vue";

const quoteStatusOptions = [
  { label: "未报价", value: "unquoted" },
  { label: "报价中", value: "quoting" },
  { label: "已报价", value: "quoted" },
  { label: "已定价", value: "priced" },
];

const quoteStatusText: Record<QuoteStatus, string> = {
  unquoted: "未报价",
  quoting: "报价中",
  quoted: "已报价",
  priced: "已定价",
};

const quoteStatusType: Record<
  QuoteStatus,
  "info" | "warning" | "success" | "primary"
> = {
  unquoted: "info",
  quoting: "warning",
  quoted: "primary",
  priced: "success",
};

const rebuildTaskStatusText: Record<string, string> = {
  queued: "等待执行",
  running: "执行中",
  success: "执行成功",
  failed: "执行失败",
};

const customers = ref<Customer[]>([]);
const deliveryList = ref<NewCustomerMaterialItem[]>([]);
const total = ref(0);
const loading = ref(false);
const exportLoading = ref(false);
const quoteVisible = ref(false);
const rebuildTaskVisible = ref(false);
const rebuildSubmitting = ref(false);
const selectedDelivery = ref<NewCustomerMaterialItem | null>(null);
const latestRebuildTask = ref<MaterialDeliveryRebuildTask | null>(null);
const dateRange = ref<string[]>([
  String(dayjs().startOf("month").unix()),
  String(dayjs().endOf("day").unix()),
]);
const initSearchForm = (): NewCustomerMaterialRequest => ({
  page: 1,
  size: 10,
  customer_id: "",
  start_time: Number(dateRange.value[0]),
  end_time: Number(dateRange.value[1]),
  quote_status: "",
  material_name: "",
  material_model: "",
});

const searchForm = ref<NewCustomerMaterialRequest>(initSearchForm());

let rebuildTaskTimer: ReturnType<typeof setInterval> | null = null;

const isRebuildTaskActive = computed(() => {
  return (
    latestRebuildTask.value?.status === "queued" ||
    latestRebuildTask.value?.status === "running"
  );
});

const rebuildTaskButtonType = computed<
  "info" | "warning" | "success" | "danger"
>(() => {
  const status = latestRebuildTask.value?.status || "";
  if (status === "failed") return "danger";
  if (status === "success") return "success";
  if (status === "queued" || status === "running") return "warning";
  return "info";
});

const rebuildTaskButtonText = computed(() => {
  const status = latestRebuildTask.value?.status || "";
  const label = rebuildTaskStatusText[status] || status || "未知";
  return `任务状态：${label}`;
});

const loadCustomers = async () => {
  const res = await reqCustomerList();
  if (res.code === 200) {
    customers.value = res.data.list;
  } else {
    ElMessage.error(res.msg);
  }
};

const syncSearchTime = () => {
  searchForm.value.start_time = Number(dateRange.value?.[0] || 0);
  searchForm.value.end_time = Number(dateRange.value?.[1] || 0);
};

const getDeliveries = async () => {
  syncSearchTime();
  if (!searchForm.value.customer_id) {
    ElMessage.warning("请选择客户");
    return;
  }
  if (!searchForm.value.start_time || !searchForm.value.end_time) {
    ElMessage.warning("请选择首次交付时间范围");
    return;
  }
  loading.value = true;
  const res = await reqNewCustomerMaterials(searchForm.value);
  if (res.code === 200) {
    deliveryList.value = res.data.list;
    total.value = res.data.total;
  } else {
    ElMessage.error(res.msg);
    deliveryList.value = [];
    total.value = 0;
  }
  loading.value = false;
};

const buildExportPayload = (): NewCustomerMaterialExportRequest | null => {
  syncSearchTime();
  if (!searchForm.value.customer_id) {
    ElMessage.warning("请选择客户");
    return null;
  }
  if (!searchForm.value.start_time || !searchForm.value.end_time) {
    ElMessage.warning("请选择首次交付时间范围");
    return null;
  }
  return {
    customer_id: searchForm.value.customer_id,
    start_time: searchForm.value.start_time,
    end_time: searchForm.value.end_time,
    quote_status: searchForm.value.quote_status,
    material_name: searchForm.value.material_name,
    material_model: searchForm.value.material_model,
  };
};

const sanitizeExportFileName = (name: string, fallback = "客户") => {
  const fileName = (name || "").replace(/[\\/:*?"<>|]/g, "_").trim();
  return fileName || fallback;
};

const buildExportFileName = (payload: NewCustomerMaterialExportRequest) => {
  const customerName =
    customers.value.find((customer) => customer.id === payload.customer_id)
      ?.name || "客户";
  const start = dayjs.unix(payload.start_time).format("YYYYMMDD");
  const end = dayjs.unix(payload.end_time).format("YYYYMMDD");
  return `${sanitizeExportFileName(
    customerName,
  )}-新增物料报价-${start}至${end}.csv`;
};

const exportNewDeliveryQuotes = async () => {
  const payload = buildExportPayload();
  if (!payload) return;

  exportLoading.value = true;
  try {
    const body = await reqExportNewCustomerMaterialQuotes(payload);
    const blob =
      body instanceof Blob
        ? body
        : new Blob([body as unknown as string | ArrayBuffer | Blob], {
            type: "text/csv;charset=utf-8",
          });
    if (blob.type.includes("application/json")) {
      const text = await blob.text();
      const result = JSON.parse(text);
      ElMessage.error(result.msg || "导出客户新增物料报价失败");
      return;
    }
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = buildExportFileName(payload);
    link.click();
    window.URL.revokeObjectURL(url);
  } finally {
    exportLoading.value = false;
  }
};

const resetSearch = () => {
  dateRange.value = [
    String(dayjs().startOf("month").unix()),
    String(dayjs().endOf("day").unix()),
  ];
  searchForm.value = initSearchForm();
  deliveryList.value = [];
  total.value = 0;
};

const rebuildDeliveries = async () => {
  const result = await ElMessageBox.confirm(
    "将根据历史出库单重建客户新增物料记录，报价状态会尽量保留，是否继续？",
    "重建记录",
    { type: "warning", confirmButtonText: "重建", cancelButtonText: "取消" },
  ).catch((reason) => reason);
  if (result !== "confirm") return;
  rebuildSubmitting.value = true;
  try {
    const res = await reqRebuildNewCustomerMaterials();
    if (res.code === 200) {
      ElMessage.success(res.msg);
      if (res.data?.id) {
        latestRebuildTask.value = res.data;
      }
      if (isRebuildTaskActive.value) {
        startRebuildTaskPolling();
      }
    } else {
      ElMessage.error(res.msg);
    }
  } finally {
    rebuildSubmitting.value = false;
  }
};

const refreshLatestRebuildTask = async (fromPolling = false) => {
  const wasActive = isRebuildTaskActive.value;
  const res = await reqLatestMaterialDeliveryRebuildTask();
  if (res.code === 200) {
    latestRebuildTask.value = res.data?.id ? res.data : null;
    if (isRebuildTaskActive.value) {
      startRebuildTaskPolling();
      return;
    }
    stopRebuildTaskPolling();
    if (
      fromPolling &&
      wasActive &&
      latestRebuildTask.value?.status === "success" &&
      searchForm.value.customer_id
    ) {
      await getDeliveries();
    }
  } else if (!fromPolling) {
    ElMessage.error(res.msg);
  }
};

const startRebuildTaskPolling = () => {
  if (rebuildTaskTimer) return;
  rebuildTaskTimer = setInterval(() => {
    refreshLatestRebuildTask(true);
  }, 3000);
};

const stopRebuildTaskPolling = () => {
  if (!rebuildTaskTimer) return;
  clearInterval(rebuildTaskTimer);
  rebuildTaskTimer = null;
};

const handleSizeChange = () => getDeliveries();
const handleCurrentChange = () => getDeliveries();

const openQuote = (row: NewCustomerMaterialItem) => {
  selectedDelivery.value = row;
  quoteVisible.value = true;
};

const priceText = (value: number) =>
  value > 0 ? Number(value || 0).toFixed(3) : "-";
const statusLabel = (status: QuoteStatus) =>
  quoteStatusText[status] || status || "-";
const statusTagType = (status: QuoteStatus) =>
  quoteStatusType[status] || "info";

onMounted(async () => {
  await loadCustomers();
  await refreshLatestRebuildTask(true);
});

onUnmounted(() => {
  stopRebuildTaskPolling();
});
</script>

<template>
  <div class="material-quote-page">
    <el-form
      :inline="true"
      :model="searchForm"
      class="filter-form"
      label-width="96px"
    >
      <el-form-item label="客户">
        <el-select
          v-model="searchForm.customer_id"
          filterable
          clearable
          placeholder="请选择客户"
          class="filter-control"
        >
          <el-option
            v-for="customer in customers"
            :key="customer.id"
            :label="customer.name"
            :value="customer.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="首次交付">
        <el-date-picker
          v-model="dateRange"
          type="datetimerange"
          value-format="X"
          start-placeholder="开始时间"
          end-placeholder="结束时间"
          class="date-range"
        />
      </el-form-item>
      <el-form-item label="报价状态">
        <el-select
          v-model="searchForm.quote_status"
          clearable
          placeholder="全部"
          class="filter-control"
        >
          <el-option
            v-for="item in quoteStatusOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="物料名称">
        <el-input
          v-model.trim="searchForm.material_name"
          clearable
          placeholder="物料名称"
          class="filter-control"
        />
      </el-form-item>
      <el-form-item label="物料型号">
        <el-input
          v-model.trim="searchForm.material_model"
          clearable
          placeholder="物料型号"
          class="filter-control"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" plain icon="Search" @click="getDeliveries"
          >查询</el-button
        >
        <el-button
          plain
          icon="Download"
          :loading="exportLoading"
          @click="exportNewDeliveryQuotes"
          >导出报价</el-button
        >
        <el-button plain icon="RefreshRight" @click="resetSearch"
          >重置</el-button
        >
        <el-button
          plain
          icon="Refresh"
          :loading="rebuildSubmitting"
          :disabled="isRebuildTaskActive"
          @click="rebuildDeliveries"
        >
          重建记录
        </el-button>
        <el-button
          v-if="latestRebuildTask?.id"
          plain
          icon="InfoFilled"
          :type="rebuildTaskButtonType"
          @click="rebuildTaskVisible = true"
        >
          {{ rebuildTaskButtonText }}
        </el-button>
      </el-form-item>
    </el-form>

    <el-card class="data">
      <el-pagination
        v-model:current-page="searchForm.page"
        v-model:page-size="searchForm.size"
        class="pagination"
        :page-sizes="[10, 20, 30, 50]"
        :total="total"
        :disabled="loading"
        background
        layout="total, sizes, prev, pager, next, ->, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
      <el-table
        v-loading="loading"
        border
        stripe
        :data="deliveryList"
        class="table"
      >
        <el-table-column label="首次交付" width="116">
          <template #default="{ row }">
            {{ DateFormat(row.first_delivery_time) }}
          </template>
        </el-table-column>
        <el-table-column
          label="客户"
          prop="customer_name"
          min-width="150"
          show-overflow-tooltip
        />
        <el-table-column
          label="物料名称"
          prop="material_name"
          min-width="170"
          show-overflow-tooltip
        />
        <el-table-column
          label="型号"
          prop="material_model"
          min-width="160"
          show-overflow-tooltip
        />
        <el-table-column
          label="规格"
          prop="material_specification"
          min-width="180"
          show-overflow-tooltip
        />
        <el-table-column label="单位" prop="material_unit" width="72" />
        <el-table-column
          label="首次出库单"
          prop="first_delivery_order_code"
          min-width="150"
          show-overflow-tooltip
        />
        <el-table-column label="首次数量" width="100" align="right">
          <template #default="{ row }">
            {{ row.first_delivery_quantity }}
          </template>
        </el-table-column>
        <el-table-column label="首次单价" width="110" align="right">
          <template #default="{ row }">
            {{ priceText(row.first_delivery_price) }}
          </template>
        </el-table-column>
        <el-table-column label="报价状态" width="96" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTagType(row.quote_status)">
              {{ statusLabel(row.quote_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最新报价" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.latest_quote_no">{{ row.latest_quote_no }}</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="最新单价" width="110" align="right">
          <template #default="{ row }">
            {{ priceText(row.latest_price) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right" align="center">
          <template #default="{ row }">
            <el-button
              type="primary"
              link
              icon="EditPen"
              @click="openQuote(row)"
            >
              {{ row.latest_quote_id ? "编辑报价" : "新增报价" }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <MaterialQuoteDialog
      v-model="quoteVisible"
      :delivery="selectedDelivery"
      @changed="getDeliveries"
    />
    <RebuildTaskDialog
      v-model="rebuildTaskVisible"
      :task="latestRebuildTask"
      @refresh="refreshLatestRebuildTask(false)"
    />
  </div>
</template>

<style scoped lang="scss">
.material-quote-page {
  .filter-form {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 0;
  }

  .filter-control {
    width: 180px;
  }

  .date-range {
    width: 360px;
  }

  .data {
    margin-top: 12px;
  }

  .pagination {
    margin-bottom: 12px;
  }

  .table {
    width: 100%;
  }
}
</style>
