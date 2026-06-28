<script setup lang="ts">
import { computed, ref, watch } from "vue";
import dayjs from "dayjs";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  MaterialQuote,
  MaterialQuoteCostItem,
  MaterialQuoteSaveRequest,
  NewCustomerMaterialItem,
} from "@/api/material/types.ts";
import {
  reqExportMaterialQuote,
  reqMaterialQuoteInfo,
  reqPriceMaterialQuote,
  reqSaveMaterialQuote,
  reqSubmitMaterialQuote,
  reqVoidMaterialQuote,
} from "@/api/material";

const quoteStatusMap: Record<string, string> = {
  draft: "草稿",
  submitted: "已提交",
  quoted: "已报价",
  priced: "已定价",
  void: "已作废",
};

const costSections = [
  {
    code: "material",
    name: "材料成本",
    defaults: ["原材料", "辅料", "材料损耗"],
  },
  {
    code: "process",
    name: "加工工序成本",
    defaults: ["下料", "折弯", "焊接", "机加工", "表面处理"],
  },
  {
    code: "labor_equipment",
    name: "人工/设备成本",
    defaults: ["人工工时", "设备工时", "调机费"],
  },
  { code: "quality", name: "质量成本", defaults: ["检验", "返工预估"] },
  {
    code: "packing_logistics",
    name: "包装/物流成本",
    defaults: ["清点", "标签", "打包", "纸箱", "运费", "装卸费"],
  },
  { code: "management", name: "管理成本", defaults: ["管理费", "财务成本"] },
  {
    code: "tooling",
    name: "模具/治具摊销",
    defaults: ["模具摊销", "治具摊销"],
  },
  { code: "loss", name: "损耗成本", defaults: ["生产损耗", "异常损耗"] },
  { code: "other", name: "其他成本", defaults: ["其他"] },
];

const MONEY_DECIMALS = 3;
const RATE_DECIMALS = 3;
const MONEY_STEP = 0.001;
const RATE_STEP = 0.001;

const props = defineProps<{
  modelValue: boolean;
  delivery: NewCustomerMaterialItem | null;
}>();

const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
  (event: "changed"): void;
}>();

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit("update:modelValue", value),
});
const selectedDelivery = computed(() => props.delivery);

const quoteLoading = ref(false);
const currentQuote = ref<MaterialQuote | null>(null);
const quoteValidRange = ref<string[]>([]);

const initQuoteForm = (): MaterialQuoteSaveRequest => ({
  id: "",
  delivery_id: "",
  quote_mode: "detailed",
  currency: "CNY",
  cost_items: createDefaultCostItems(),
  simple_price: 0,
  profit_amount: 0,
  tax_rate: 0,
  final_price: 0,
  valid_from: 0,
  valid_to: 0,
  remark: "",
});

const quoteForm = ref<MaterialQuoteSaveRequest>(initQuoteForm());

const loadQuote = async (row: NewCustomerMaterialItem) => {
  currentQuote.value = null;
  quoteValidRange.value = [];
  quoteForm.value = initQuoteForm();
  quoteForm.value.delivery_id = row.id;

  if (!row.latest_quote_id) return;

  const res = await reqMaterialQuoteInfo({ id: row.latest_quote_id });
  if (res.code === 200) {
    currentQuote.value = res.data;
    quoteForm.value = toQuoteForm(res.data);
    if (res.data.valid_from > 0 && res.data.valid_to > 0) {
      quoteValidRange.value = [
        String(res.data.valid_from),
        String(res.data.valid_to),
      ];
    }
  } else {
    ElMessage.warning(res.msg);
  }
};

watch(
  () => [props.modelValue, props.delivery?.id, props.delivery?.latest_quote_id],
  ([isVisible]) => {
    if (isVisible && props.delivery) {
      loadQuote(props.delivery);
    }
  },
  { immediate: true },
);

const toQuoteForm = (quote: MaterialQuote): MaterialQuoteSaveRequest => ({
  id: quote.id,
  delivery_id: quote.delivery_id,
  quote_mode: quote.quote_mode,
  currency: quote.currency || "CNY",
  cost_items: quote.cost_items?.length
    ? normalizeCostItemIndexes(quote.cost_items)
    : createDefaultCostItems(),
  simple_price: roundMoney(quote.simple_price),
  profit_amount:
    quote.profit_amount ||
    profitAmountFromRate(quote.total_cost, quote.profit_rate),
  tax_rate: roundRateFraction(quote.tax_rate),
  final_price: roundMoney(quote.final_price),
  valid_from: quote.valid_from,
  valid_to: quote.valid_to,
  remark: quote.remark,
});

const saveQuote = async (silent = false): Promise<MaterialQuote | null> => {
  const payload = buildQuotePayload();
  if (!payload) return null;
  quoteLoading.value = true;
  const res = await reqSaveMaterialQuote(payload);
  quoteLoading.value = false;
  if (res.code === 200) {
    quoteForm.value = toQuoteForm(res.data);
    currentQuote.value = res.data;
    if (!silent) ElMessage.success(res.msg);
    emit("changed");
    return res.data;
  }
  ElMessage.error(res.msg);
  return null;
};

const submitQuote = async () => {
  const quote = await saveQuote(true);
  if (!quote) return;
  const res = await reqSubmitMaterialQuote({ id: quote.id });
  if (res.code === 200) {
    currentQuote.value = res.data;
    quoteForm.value = toQuoteForm(res.data);
    ElMessage.success(res.msg);
    emit("changed");
  } else {
    ElMessage.error(res.msg);
  }
};

const exportQuote = async () => {
  let quote = currentQuote.value;
  if (!quote?.id) {
    quote = await saveQuote(true);
  }
  if (!quote?.id) return;
  const body = await reqExportMaterialQuote({ id: quote.id });
  const blob =
    body instanceof Blob
      ? body
      : new Blob([body as unknown as string | ArrayBuffer | Blob], {
          type: "text/csv;charset=utf-8",
        });
  if (blob.type.includes("application/json")) {
    const text = await blob.text();
    const result = JSON.parse(text);
    ElMessage.error(result.msg || "导出报价失败");
    return;
  }
  const url = window.URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${quote.quote_no || "material-quote"}.csv`;
  link.click();
  window.URL.revokeObjectURL(url);
};

const priceQuote = async () => {
  const quote = currentQuote.value;
  if (!quote?.id) {
    ElMessage.warning("请先保存并提交报价");
    return;
  }
  if (quote.status !== "quoted") {
    ElMessage.warning("只有已报价状态可以转最终定价");
    return;
  }
  const result = await ElMessageBox.prompt(
    "请输入客户最终确认单价",
    "转最终定价",
    {
      inputValue: String(
        quote.final_price ||
          quoteForm.value.final_price ||
          computedFinalPrice.value,
      ),
      inputPattern: /^(0|[1-9]\d*)(\.\d+)?$/,
      inputErrorMessage: "请输入有效金额",
      confirmButtonText: "确认定价",
      cancelButtonText: "取消",
    },
  ).catch((reason) => reason);
  if (!result?.value) return;
  const finalPrice = Number(result.value);
  if (finalPrice <= 0) {
    ElMessage.error("最终定价必须大于0");
    return;
  }
  const res = await reqPriceMaterialQuote({
    id: quote.id,
    final_price: finalPrice,
    effective_at: dayjs().unix(),
    remark: quoteForm.value.remark,
  });
  if (res.code === 200) {
    currentQuote.value = res.data;
    quoteForm.value = toQuoteForm(res.data);
    ElMessage.success(res.msg);
    emit("changed");
  } else {
    ElMessage.error(res.msg);
  }
};

const voidQuote = async () => {
  const quote = currentQuote.value;
  if (!quote?.id) return;
  const result = await ElMessageBox.confirm(
    `确认作废报价单 ${quote.quote_no}？`,
    "作废报价",
    {
      type: "warning",
      confirmButtonText: "作废",
      cancelButtonText: "取消",
    },
  ).catch((reason) => reason);
  if (result !== "confirm") return;
  const res = await reqVoidMaterialQuote({ id: quote.id });
  if (res.code === 200) {
    ElMessage.success(res.msg);
    visible.value = false;
    emit("changed");
  } else {
    ElMessage.error(res.msg);
  }
};

const buildQuotePayload = (): MaterialQuoteSaveRequest | null => {
  if (!selectedDelivery.value) {
    ElMessage.warning("请选择需要报价的新增物料");
    return null;
  }
  if (quoteValidRange.value?.length === 2) {
    quoteForm.value.valid_from = Number(quoteValidRange.value[0]);
    quoteForm.value.valid_to = Number(quoteValidRange.value[1]);
  } else {
    quoteForm.value.valid_from = 0;
    quoteForm.value.valid_to = 0;
  }
  const payload: MaterialQuoteSaveRequest = {
    ...quoteForm.value,
    cost_items: normalizedCostItems(),
  };
  if (payload.quote_mode === "simple") {
    const price = simpleQuotePrice.value;
    if (price <= 0) {
      ElMessage.warning("简单报价必须填写非税报价");
      return null;
    }
    payload.simple_price = price;
    payload.profit_amount = 0;
    payload.tax_rate = roundRateFraction(payload.tax_rate);
    payload.final_price = roundMoney(payload.final_price);
    payload.cost_items = [];
  }
  if (payload.quote_mode === "detailed") {
    payload.simple_price = 0;
    payload.profit_amount = roundMoney(payload.profit_amount);
    payload.tax_rate = roundRateFraction(payload.tax_rate);
    payload.final_price = roundMoney(payload.final_price);
    const hasEnabledCost = payload.cost_items.some(
      (item) => item.enabled && costAmount(item) > 0,
    );
    if (!hasEnabledCost && payload.final_price <= 0) {
      ElMessage.warning("详细报价需要启用成本项，或填写最终报价单价");
      return null;
    }
  }
  return payload;
};

function createDefaultCostItems(): MaterialQuoteCostItem[] {
  const items: MaterialQuoteCostItem[] = [];
  costSections.forEach((section) => {
    section.defaults.forEach((name, idx) => {
      items.push({
        index: idx + 1,
        category_code: section.code,
        category_name: section.name,
        name,
        enabled: section.code === "material" && idx === 0,
        custom: false,
        amount: 0,
        remark: "",
      });
    });
  });
  return items;
}

const sectionCostItems = (code: string) =>
  quoteForm.value.cost_items
    .filter((item) => item.category_code === code)
    .sort((a, b) => a.index - b.index);

const renumberCostSection = (code: string) => {
  sectionCostItems(code).forEach((item, index) => {
    item.index = index + 1;
  });
};

const normalizeCostItemIndexes = (items: MaterialQuoteCostItem[]) => {
  const clonedItems = items.map((item) => ({
    ...item,
    amount: roundMoney(Number(item.amount || 0)),
  }));
  costSections.forEach((section) => {
    clonedItems
      .filter((item) => item.category_code === section.code)
      .sort((a, b) => a.index - b.index)
      .forEach((item, index) => {
        item.index = index + 1;
      });
  });
  return clonedItems;
};

const costGroups = computed(() =>
  costSections.map((section) => ({
    ...section,
    items: sectionCostItems(section.code),
  })),
);

const normalizedCostItems = () =>
  costSections.flatMap((section) =>
    sectionCostItems(section.code).map((item, index) => ({
      ...item,
      index: index + 1,
      amount: costAmount(item),
    })),
  );

const costAmount = (item: MaterialQuoteCostItem) =>
  roundMoney(Number(item.amount || 0));

const sectionTotal = (code: string) =>
  roundMoney(
    quoteForm.value.cost_items
      .filter((item) => item.category_code === code && item.enabled)
      .reduce((sum, item) => sum + costAmount(item), 0),
  );

const totalCost = computed(() =>
  roundMoney(
    quoteForm.value.cost_items
      .filter((item) => item.enabled)
      .reduce((sum, item) => sum + costAmount(item), 0),
  ),
);

const profitAmount = computed(() =>
  roundMoney(Number(quoteForm.value.profit_amount || 0)),
);
const quoteBaseWithoutTax = computed(() =>
  roundMoney(totalCost.value + profitAmount.value),
);
const profitRate = computed(() =>
  quoteBaseWithoutTax.value > 0
    ? roundRateFraction(profitAmount.value / quoteBaseWithoutTax.value)
    : 0,
);
const simpleQuotePrice = computed({
  get: () =>
    roundMoney(
      Number(
        quoteForm.value.simple_price ||
          (quoteForm.value.quote_mode === "simple"
            ? quoteForm.value.final_price
            : 0) ||
          0,
      ),
    ),
  set: (value: number | undefined) => {
    const price = roundMoney(Number(value || 0));
    quoteForm.value.simple_price = price;
  },
});
const quoteBaseAmount = computed(() =>
  quoteForm.value.quote_mode === "simple"
    ? simpleQuotePrice.value
    : quoteBaseWithoutTax.value,
);
const taxAmount = computed(() =>
  roundMoney(quoteBaseAmount.value * Number(quoteForm.value.tax_rate || 0)),
);
const computedFinalPrice = computed(() =>
  roundMoney(quoteBaseAmount.value + taxAmount.value),
);
const taxRatePercent = computed({
  get: () =>
    roundDecimal(Number(quoteForm.value.tax_rate || 0) * 100, RATE_DECIMALS),
  set: (value: number | undefined) => {
    quoteForm.value.tax_rate = roundRateFraction(Number(value || 0) / 100);
  },
});

const addCostItem = (section: { code: string; name: string }) => {
  quoteForm.value.cost_items.push({
    index: sectionCostItems(section.code).length + 1,
    category_code: section.code,
    category_name: section.name,
    name: "",
    enabled: true,
    custom: true,
    amount: 0,
    remark: "",
  });
  renumberCostSection(section.code);
};

const removeCostItem = (item: MaterialQuoteCostItem) => {
  const categoryCode = item.category_code;
  quoteForm.value.cost_items = quoteForm.value.cost_items.filter(
    (one) => one !== item,
  );
  renumberCostSection(categoryCode);
};

const roundDecimal = (value: number, decimals = MONEY_DECIMALS) =>
  Number(Number(value || 0).toFixed(decimals));
const roundMoney = (value: number) => roundDecimal(value, MONEY_DECIMALS);
const roundRateFraction = (value: number) =>
  roundDecimal(value, RATE_DECIMALS + 2);
function profitAmountFromRate(totalCost?: number, profitRate?: number) {
  const cost = Number(totalCost || 0);
  const rate = Number(profitRate || 0);
  if (cost <= 0 || rate <= 0 || rate >= 1) return 0;
  return roundMoney((cost * rate) / (1 - rate));
}
const priceText = (value: number) =>
  value > 0 ? roundMoney(value).toFixed(MONEY_DECIMALS) : "-";
const percentText = (value: number) =>
  roundDecimal(value * 100, RATE_DECIMALS).toFixed(RATE_DECIMALS);
const quoteStatusLabel = (status: string) =>
  quoteStatusMap[status] || status || "-";
</script>

<template>
  <el-dialog
    v-model="visible"
    width="92%"
    top="3vh"
    :close-on-click-modal="false"
    destroy-on-close
    class="quote-dialog"
  >
    <template #header>
      <div class="dialog-title">
        <span>客户新增物料报价</span>
        <el-tag
          v-if="currentQuote"
          :type="currentQuote.status === 'priced' ? 'success' : 'info'"
        >
          {{ quoteStatusLabel(currentQuote.status) }}
        </el-tag>
      </div>
    </template>

    <div v-if="selectedDelivery" class="quote-layout">
      <section class="quote-meta">
        <div>
          <span class="meta-label">客户</span>
          <strong>{{ selectedDelivery.customer_name }}</strong>
        </div>
        <div>
          <span class="meta-label">物料</span>
          <strong>{{ selectedDelivery.material_name }}</strong>
        </div>
        <div>
          <span class="meta-label">型号</span>
          <strong>{{ selectedDelivery.material_model }}</strong>
        </div>
        <div>
          <span class="meta-label">首次出库</span>
          <strong>{{ selectedDelivery.first_delivery_order_code }}</strong>
        </div>
      </section>

      <el-form
        :model="quoteForm"
        label-width="96px"
        size="default"
        class="quote-form"
      >
        <div class="quote-toolbar">
          <el-form-item label="报价方式">
            <el-radio-group v-model="quoteForm.quote_mode">
              <el-radio-button label="detailed">详细报价</el-radio-button>
              <el-radio-button label="simple">简单报价</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="币种">
            <el-input
              v-model.trim="quoteForm.currency"
              class="currency-input"
            />
          </el-form-item>
          <el-form-item label="有效期">
            <el-date-picker
              v-model="quoteValidRange"
              type="datetimerange"
              value-format="X"
              start-placeholder="开始"
              end-placeholder="结束"
              class="valid-range-picker"
            />
          </el-form-item>
        </div>

        <template v-if="quoteForm.quote_mode === 'simple'">
          <div class="simple-quote">
            <el-form-item label="非税报价" class="simple-price-item">
              <el-input-number
                v-model="simpleQuotePrice"
                :min="0"
                :precision="MONEY_DECIMALS"
                :step="MONEY_STEP"
                :controls="false"
                class="decimal-input"
              />
            </el-form-item>
          </div>
        </template>

        <template v-else>
          <div class="cost-sections">
            <section
              v-for="section in costGroups"
              :key="section.code"
              class="cost-section"
            >
              <div class="section-head">
                <h3>{{ section.name }}</h3>
                <div>
                  <span>小计 {{ priceText(sectionTotal(section.code)) }}</span>
                  <el-button
                    link
                    type="primary"
                    icon="Plus"
                    @click="addCostItem(section)"
                    >增加</el-button
                  >
                </div>
              </div>
              <el-table
                :data="section.items"
                border
                class="cost-table"
                empty-text="暂无成本项"
              >
                <el-table-column label="启用" width="58" align="center">
                  <template #default="{ row }">
                    <el-switch v-model="row.enabled" />
                  </template>
                </el-table-column>
                <el-table-column label="序号" width="58" align="center">
                  <template #default="{ $index }">
                    <span class="cost-index">{{ $index + 1 }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="成本项" min-width="120">
                  <template #default="{ row }">
                    <el-input
                      v-model.trim="row.name"
                      placeholder="成本项名称"
                    />
                  </template>
                </el-table-column>
                <el-table-column label="金额" width="120">
                  <template #default="{ row }">
                    <el-input-number
                      v-model="row.amount"
                      :min="0"
                      :precision="MONEY_DECIMALS"
                      :step="MONEY_STEP"
                      :controls="false"
                      class="decimal-input"
                    />
                  </template>
                </el-table-column>
                <el-table-column label="备注" min-width="120">
                  <template #default="{ row }">
                    <el-input v-model.trim="row.remark" placeholder="备注" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="64" align="center">
                  <template #default="{ row }">
                    <el-button
                      link
                      type="danger"
                      icon="Delete"
                      @click="removeCostItem(row)"
                      >删除</el-button
                    >
                  </template>
                </el-table-column>
              </el-table>
            </section>
          </div>
        </template>

        <el-form-item label="备注" class="remark-item">
          <el-input
            v-model.trim="quoteForm.remark"
            type="textarea"
            :rows="2"
            placeholder="报价备注"
          />
        </el-form-item>

        <section
          class="quote-summary"
          :class="{ 'is-simple': quoteForm.quote_mode === 'simple' }"
        >
          <template v-if="quoteForm.quote_mode === 'simple'">
            <div class="summary-panel summary-total">
              <span class="summary-title">非税报价</span>
              <strong class="summary-value">{{
                priceText(simpleQuotePrice)
              }}</strong>
            </div>
            <div class="summary-panel">
              <span class="summary-title">税率/税额</span>
              <div class="summary-row has-suffix">
                <span>税率</span>
                <el-input-number
                  v-model="taxRatePercent"
                  :min="0"
                  :precision="RATE_DECIMALS"
                  :step="RATE_STEP"
                  :controls="false"
                  class="decimal-input"
                />
                <span class="summary-suffix">%</span>
              </div>
              <div class="summary-row">
                <span>税额</span>
                <strong>{{ priceText(taxAmount) }}</strong>
              </div>
            </div>
            <div class="summary-panel summary-price summary-price-wide">
              <span class="summary-title">定价结果</span>
              <div class="summary-row">
                <span>测算单价</span>
                <strong>{{ priceText(computedFinalPrice) }}</strong>
              </div>
              <div class="summary-row final-price-row">
                <span>最终定价</span>
                <el-input-number
                  v-model="quoteForm.final_price"
                  :min="0"
                  :precision="MONEY_DECIMALS"
                  :step="MONEY_STEP"
                  :controls="false"
                  class="decimal-input"
                />
              </div>
            </div>
          </template>
          <template v-else>
            <div class="summary-panel summary-total">
              <span class="summary-title">成本合计</span>
              <strong class="summary-value">{{ priceText(totalCost) }}</strong>
            </div>
            <div class="summary-panel">
              <span class="summary-title">利润/利润率</span>
              <div class="summary-row">
                <span>利润</span>
                <el-input-number
                  v-model="quoteForm.profit_amount"
                  :min="0"
                  :precision="MONEY_DECIMALS"
                  :step="MONEY_STEP"
                  :controls="false"
                  class="decimal-input"
                />
              </div>
              <div class="summary-row">
                <span>利润率</span>
                <strong>{{ percentText(profitRate) }}%</strong>
              </div>
            </div>
            <div class="summary-panel">
              <span class="summary-title">税率/税额</span>
              <div class="summary-row has-suffix">
                <span>税率</span>
                <el-input-number
                  v-model="taxRatePercent"
                  :min="0"
                  :precision="RATE_DECIMALS"
                  :step="RATE_STEP"
                  :controls="false"
                  class="decimal-input"
                />
                <span class="summary-suffix">%</span>
              </div>
              <div class="summary-row">
                <span>税额</span>
                <strong>{{ priceText(taxAmount) }}</strong>
              </div>
            </div>
            <div class="summary-panel summary-price">
              <span class="summary-title">定价结果</span>
              <div class="summary-row">
                <span>测算单价</span>
                <strong>{{ priceText(computedFinalPrice) }}</strong>
              </div>
              <div class="summary-row final-price-row">
                <span>最终定价</span>
                <el-input-number
                  v-model="quoteForm.final_price"
                  :min="0"
                  :precision="MONEY_DECIMALS"
                  :step="MONEY_STEP"
                  :controls="false"
                  class="decimal-input"
                />
              </div>
            </div>
          </template>
        </section>
      </el-form>
    </div>

    <template #footer>
      <el-button @click="visible = false">关闭</el-button>
      <el-button
        v-if="currentQuote?.id && currentQuote.status !== 'priced'"
        plain
        type="danger"
        icon="Delete"
        @click="voidQuote"
        >作废</el-button
      >
      <el-button
        v-if="currentQuote?.id"
        plain
        icon="Download"
        @click="exportQuote"
        >导出</el-button
      >
      <el-button
        v-if="currentQuote?.status === 'quoted'"
        plain
        type="success"
        icon="Money"
        @click="priceQuote"
      >
        转最终定价
      </el-button>
      <el-button
        :loading="quoteLoading"
        plain
        type="primary"
        icon="DocumentChecked"
        @click="saveQuote(false)"
      >
        保存草稿
      </el-button>
      <el-button
        :loading="quoteLoading"
        type="primary"
        icon="Promotion"
        @click="submitQuote"
      >
        提交报价
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped lang="scss">
.dialog-title {
  display: flex;
  gap: 12px;
  align-items: center;
  font-weight: 600;
}

.quote-dialog {
  :deep(.el-dialog) {
    max-width: 1680px;
  }

  :deep(.el-dialog__body) {
    padding-top: 8px;
  }
}

.quote-layout {
  max-height: 76vh;
  padding-right: 4px;
  overflow: auto;
}

.quote-meta {
  display: grid;
  grid-template-columns: repeat(4, minmax(160px, 1fr));
  gap: 10px 12px;
  padding: 8px 0 10px;
  margin-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-light);

  div {
    min-width: 0;
  }

  strong {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.meta-label {
  display: block;
  margin-bottom: 4px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.quote-form {
  :deep(.el-form-item) {
    margin-bottom: 10px;
  }

  :deep(.el-input),
  :deep(.el-input-number),
  :deep(.el-select),
  :deep(.el-date-editor) {
    max-width: 100%;
  }
}

.quote-toolbar {
  display: grid;
  grid-template-columns: minmax(250px, 1fr) 180px minmax(360px, 1.4fr);
  gap: 8px 12px;
  align-items: start;
}

.simple-quote {
  display: grid;
  grid-template-columns: minmax(260px, 360px);
  gap: 8px 12px;
  max-width: 760px;
}

.simple-price-item {
  max-width: 460px;
}

.currency-input {
  width: 96px;
}

.valid-range-picker {
  width: 100%;
}

.cost-sections {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  align-items: start;
}

.cost-section {
  min-width: 0;
  padding: 10px;
  background: var(--el-fill-color-blank);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
}

.section-head {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;

  h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
  }

  div {
    display: flex;
    gap: 8px;
    align-items: center;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }
}

.cost-table {
  width: 100%;

  :deep(.cell) {
    padding: 0 6px;
  }

  :deep(.el-table__cell) {
    padding: 6px 0;
  }

  :deep(.el-input__inner[type="number"]) {
    text-align: right;
  }
}

.cost-index {
  display: inline-block;
  min-width: 24px;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-regular);
}

.quote-summary {
  position: sticky;
  bottom: 0;
  z-index: 2;
  display: grid;
  grid-template-columns:
    minmax(150px, 0.9fr) repeat(2, minmax(220px, 1.2fr))
    minmax(280px, 1.6fr);
  gap: 10px;
  padding: 10px;
  margin-top: 12px;
  background: linear-gradient(
    180deg,
    rgb(255 255 255 / 96%),
    var(--el-bg-color)
  );
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  box-shadow: 0 -4px 14px rgb(0 0 0 / 5%);

  &.is-simple {
    grid-template-columns:
      minmax(170px, 0.8fr) minmax(220px, 1.2fr)
      minmax(280px, 1.6fr);
  }
}

.summary-panel {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: 8px 10px;
  background: var(--el-fill-color-lighter);
  border-left: 3px solid var(--el-border-color);
  border-radius: 4px;
}

.summary-total {
  background: var(--el-color-primary-light-9);
  border-left-color: var(--el-color-primary);
}

.summary-price {
  background: var(--el-color-success-light-9);
  border-left-color: var(--el-color-success);
}

.summary-price-wide {
  min-width: 0;
}

.summary-title {
  font-size: 12px;
  line-height: 1;
  color: var(--el-text-color-secondary);
}

.summary-value {
  font-size: 22px;
  font-variant-numeric: tabular-nums;
  line-height: 32px;
  color: var(--el-color-primary);
}

.summary-row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  min-height: 32px;

  span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  strong {
    overflow: hidden;
    font-size: 16px;
    font-variant-numeric: tabular-nums;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &.has-suffix {
    grid-template-columns: 72px minmax(0, 1fr) 16px;
    gap: 6px;
  }

  :deep(.el-input),
  :deep(.el-input-number) {
    width: 100%;
  }

  :deep(.el-input__inner) {
    text-align: right;
  }
}

.summary-suffix {
  text-align: right;
}

.final-price-row {
  :deep(.el-input__wrapper) {
    box-shadow: 0 0 0 1px var(--el-color-success) inset;
  }
}

.remark-item {
  margin-top: 12px;
}

@media (width <= 1200px) {
  .quote-toolbar {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .cost-sections {
    grid-template-columns: 1fr;
  }

  .quote-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .quote-meta,
  .quote-toolbar,
  .simple-quote,
  .quote-summary {
    grid-template-columns: 1fr;
  }

  .summary-panel {
    min-width: 0;
  }
}

.summary-panel,
.cost-section {
  box-sizing: border-box;
}

.section-head span,
.summary-row span,
.summary-title {
  word-break: keep-all;
}

.quote-summary,
.cost-sections {
  min-width: 0;
}

.summary-price .summary-row strong {
  color: var(--el-color-success);
}

.summary-total .summary-value,
.summary-price .summary-row strong {
  font-weight: 700;
}

.summary-panel:not(.summary-total, .summary-price) strong {
  color: var(--el-text-color-primary);
}

.cost-section :deep(.el-table__header-wrapper th) {
  background: var(--el-fill-color-light);
}

.cost-section :deep(.el-button.is-link) {
  padding: 0;
}

.cost-section :deep(.el-table__empty-block) {
  min-height: 42px;
}

.quote-toolbar :deep(.el-form-item__content),
.simple-quote :deep(.el-form-item__content) {
  min-width: 0;
}

.quote-toolbar :deep(.el-date-editor.el-input__wrapper) {
  width: 100%;
}

.quote-summary :deep(.el-input__wrapper) {
  background: var(--el-bg-color);
}

.decimal-input {
  width: 100%;

  :deep(.el-input__inner) {
    text-align: right;
  }
}

.quote-summary :deep(input[type="number"]::-webkit-outer-spin-button),
.quote-summary :deep(input[type="number"]::-webkit-inner-spin-button),
.cost-table :deep(input[type="number"]::-webkit-outer-spin-button),
.cost-table :deep(input[type="number"]::-webkit-inner-spin-button) {
  margin: 0;
}

.quote-summary :deep(input[type="number"]),
.cost-table :deep(input[type="number"]) {
  font-variant-numeric: tabular-nums;
}

.quote-summary :deep(.el-input__suffix) {
  color: var(--el-text-color-secondary);
}

.cost-section :deep(.el-switch) {
  vertical-align: middle;
}

.cost-section :deep(.el-table__body-wrapper) {
  overflow-x: auto;
}

.quote-layout :deep(.el-textarea__inner) {
  resize: vertical;
}

.quote-layout :deep(.el-input__wrapper),
.quote-layout :deep(.el-textarea__inner) {
  box-sizing: border-box;
}

.quote-layout :deep(.el-form-item__label) {
  line-height: 32px;
}

.quote-meta strong,
.summary-row strong,
.summary-value {
  min-width: 0;
}
</style>
