<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';
import type { FormInstance, FormRules } from 'element-plus';
import { useAuthStore } from '@/store/modules/auth';
import { reqCustomers } from '@/api/customer';
import { getDocuments, getDocument, saveDocument, getAcceptancePayments, redeemAcceptance, reverseAcceptance, voidDocument } from '@/api/customer_documents';
import type { CustomerDocumentKind, CustomerDocument, DocumentQuery, DocumentSummary, DocumentSaveRequest, PaymentOption, RedemptionRequest, DocumentVoidRequest } from '@/api/customer_documents/types';
import DocumentAttachments from './DocumentAttachments.vue';
import { dayStart, dayEnd, dateInput, invoiceTotals, operationKey } from './document-utils';

const props = defineProps<{ kind: CustomerDocumentKind }>();
const route = useRoute();
const auth = useAuthStore();
const isAcceptance = computed(() => props.kind === 'acceptance');
const title = computed(() => isAcceptance.value ? '客户承兑' : '销售开票记录');
const can = (action: string) => auth.authButtonListGet.some(item => item.perms === `finance:${props.kind}:${action}`);
const query = ref<DocumentQuery>({ page: 1, size: 20, customer_id: '', keyword: '', status: '' });
const dates = ref<string[]>([]);
const dueDates = ref<string[]>([]);
const redemptionDates = ref<string[]>([]);
const rows = ref<CustomerDocument[]>([]);
const summary = ref<DocumentSummary>({ count: 0 });
const total = ref(0);
const loading = ref(false);
const pageError = ref('');
let loadGeneration = 0;
const customerOptions = ref<{ id: string; name: string }[]>([]);
const customersLoading = ref(false);
let customerGeneration = 0;
const statusOptions = computed(() => isAcceptance.value ? [{ value: 'pending', label: '待到账' }, { value: 'redeemed', label: '已到账' }, { value: 'voided', label: '已作废' }] : [{ value: 'active', label: '有效' }, { value: 'voided', label: '已作废' }]);
const statusLabel = (status: string) => statusOptions.value.find(item => item.value === status)?.label || status;
const statusType = (status: string) => status === 'voided' ? 'info' : status === 'pending' ? 'warning' : 'success';
const money = (value?: number) => (value ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
const dateText = (value?: number, withTime = false) => value ? new Date(value * 1000).toLocaleString('zh-CN', withTime ? { hour12: false, timeZone: 'Asia/Shanghai' } : { year: 'numeric', month: '2-digit', day: '2-digit', timeZone: 'Asia/Shanghai' }) : '—';
const dueNotice = (row: CustomerDocument) => {
  if (row.status !== 'pending' || !row.due_date) return '';
  const now = Math.floor(Date.now() / 1000);
  if (dayEnd(row.due_date) < now) return '已逾期';
  return row.due_date < dayStart(now) + 7 * 86400 ? '7 日内到期' : '';
};
const historyLabel = (value: string) => ({ create: '登记', update: '修改', edit: '修改', redeem: '确认到账', reverse: '撤销到账', link_payment: '关联已有回款', void: '作废' }[value] || value);
const failure = (error: unknown) => {
  const candidate = error as { response?: { data?: { msg?: string } }; message?: string };
  return candidate.response?.data?.msg || candidate.message || '请求失败，请重试';
};
let disposed = false;
const responseError = (response: { code: number; msg: string }) => Object.assign(new Error(response.msg), { status: response.code });
const uncertain = (error: unknown) => {
  const candidate = error as { status?: number; response?: { status?: number } };
  const status = candidate.response?.status ?? candidate.status;
  return !status || status >= 500;
};
const retryMessage = '尚未确认操作结果，已保留本次提交。请点击“重试确认”，使用原内容核对结果。';
async function loadCustomers(name = '') {
  const generation = ++customerGeneration;
  customersLoading.value = true;
  try {
    const response = await reqCustomers({ page: 1, size: 50, name, code: '', manager: '', contact: '', email: '', level: 0 });
    if (generation !== customerGeneration) return;
    if (response.code !== 200) throw new Error(response.msg);
    const pinned = customerOptions.value.filter(item => item.id === query.value.customer_id || item.id === form.value.customer_id);
    customerOptions.value = [...new Map([...pinned, ...response.data.list].map(item => [item.id, { id: item.id, name: item.name }])).values()];
  } catch (error) { if (generation === customerGeneration) ElMessage.error(failure(error)); }
  finally { if (generation === customerGeneration) customersLoading.value = false; }
}
async function load() {
  if (!can('list')) return;
  const generation = ++loadGeneration;
  loading.value = true;
  pageError.value = '';
  try {
    const response = await getDocuments(props.kind, {
      ...query.value,
      date_start: dates.value?.length === 2 ? dayStart(dates.value[0]) : undefined,
      date_end: dates.value?.length === 2 ? dayEnd(dates.value[1]) : undefined,
      due_start: isAcceptance.value && dueDates.value?.length === 2 ? dayStart(dueDates.value[0]) : undefined,
      due_end: isAcceptance.value && dueDates.value?.length === 2 ? dayEnd(dueDates.value[1]) : undefined,
      redemption_start: isAcceptance.value && redemptionDates.value?.length === 2 ? dayStart(redemptionDates.value[0]) : undefined,
      redemption_end: isAcceptance.value && redemptionDates.value?.length === 2 ? dayEnd(redemptionDates.value[1]) : undefined,
    });
    if (generation !== loadGeneration) return;
    if (response.code !== 200) throw new Error(response.msg);
    rows.value = response.data.list || [];
    total.value = response.data.total;
    summary.value = response.data.summary;
  } catch (error) {
    if (generation !== loadGeneration) return;
    rows.value = []; total.value = 0; summary.value = { count: 0 };
    pageError.value = failure(error);
  } finally { if (generation === loadGeneration) loading.value = false; }
}
function search() { query.value.page = 1; void load(); }
function reset() { query.value = { page: 1, size: 20, customer_id: '', keyword: '', status: '' }; dates.value = []; dueDates.value = []; redemptionDates.value = []; void load(); }
function applyCustomerRoute() {
  query.value.customer_id = typeof route.query.customer_id === 'string' ? route.query.customer_id : '';
  if (query.value.customer_id && typeof route.query.customer_name === 'string') {
    customerOptions.value = [{ id: query.value.customer_id, name: route.query.customer_name }];
  }
  search();
}

const detail = ref<CustomerDocument>();
const detailVisible = ref(false);
const detailLoading = ref(false);
let detailGeneration = 0;
async function openDetail(row: CustomerDocument) {
  const generation = ++detailGeneration;
  detailVisible.value = true;
  detailLoading.value = true;
  detail.value = undefined;
  try {
    const response = await getDocument(props.kind, row.id);
    if (generation !== detailGeneration) return;
    if (response.code !== 200) throw new Error(response.msg);
    detail.value = response.data;
  } catch (error) { if (generation === detailGeneration) { ElMessage.error(failure(error)); detailVisible.value = false; } }
  finally { if (generation === detailGeneration) detailLoading.value = false; }
}

interface EditorForm {
  id: string; version: number; customer_id: string; bill_number: string; amount: number;
  received_date: string; issue_date: string; due_date: string; invoice_number: string; invoice_date: string;
  tax_rate: number; amount_excluding_tax: number; tax_amount: number; annex: string[]; remark: string;
}
const today = () => dateInput(Math.floor(Date.now() / 1000));
const emptyForm = (): EditorForm => ({ id: '', version: 0, customer_id: query.value.customer_id || '', bill_number: '', amount: 0, received_date: today(), issue_date: '', due_date: '', invoice_number: '', invoice_date: today(), tax_rate: 13, amount_excluding_tax: 0, tax_amount: 0, annex: [], remark: '' });
const form = ref<EditorForm>(emptyForm());
const formRef = ref<FormInstance>();
const editVisible = ref(false);
const saving = ref(false);
const uploading = ref(false);
const formError = ref('');
const pendingSave = ref<DocumentSaveRequest>();
let saveWasUncertain = false;
let editorGeneration = 0;
let saveKey = '';
const totalAmount = computed(() => (Math.round((form.value.amount_excluding_tax || 0) * 100) + Math.round((form.value.tax_amount || 0) * 100)) / 100);
function calculateTax() { form.value.tax_amount = invoiceTotals(form.value.amount_excluding_tax, form.value.tax_rate).tax_amount; }
watch(() => [form.value.amount_excluding_tax, form.value.tax_rate], () => { if (editVisible.value) calculateTax(); }, { flush: 'sync' });
const positive = (_rule: unknown, value: number, callback: (error?: Error) => void) => callback(Number.isFinite(value) && value > 0 ? undefined : new Error('请输入大于 0 的金额'));
const rules: FormRules = {
  customer_id: [{ required: true, message: '请选择客户', trigger: 'change' }],
  bill_number: [{ required: true, whitespace: true, message: '请填写票号', trigger: 'blur' }],
  invoice_number: [{ required: true, whitespace: true, message: '请填写发票号码', trigger: 'blur' }],
  amount: [{ required: true, validator: positive, trigger: 'blur' }],
  amount_excluding_tax: [{ required: true, validator: positive, trigger: 'blur' }],
  received_date: [{ required: true, message: '请选择收票日期', trigger: 'change' }],
  invoice_date: [{ required: true, message: '请选择开票日期', trigger: 'change' }],
  issue_date: [{ required: true, message: '请选择出票日期', trigger: 'change' }],
  due_date: [{ required: true, message: '请选择到期日期', trigger: 'change' }],
};
async function edit(row?: CustomerDocument) {
  if (disposed || saving.value || uploading.value || pendingSave.value) return;
  const generation = ++editorGeneration;
  formError.value = ''; uploading.value = false;
  if (row) {
    try {
      const response = await getDocument(props.kind, row.id);
      if (generation !== editorGeneration) return;
      if (response.code !== 200) throw new Error(response.msg);
      const current = response.data;
      if (current.status !== (isAcceptance.value ? 'pending' : 'active')) { ElMessage.warning('记录状态已变化，请刷新列表'); await load(); return; }
      customerOptions.value = [...new Map([...customerOptions.value, { id: current.customer_id, name: current.customer_name }].map(item => [item.id, item])).values()];
      editVisible.value = false;
      form.value = { ...emptyForm(), ...current, bill_number: current.bill_number || '', amount: current.amount || 0, received_date: current.received_date ? dateInput(current.received_date) : '', issue_date: current.issue_date ? dateInput(current.issue_date) : '', due_date: current.due_date ? dateInput(current.due_date) : '', invoice_number: current.invoice_number || '', invoice_date: current.invoice_date ? dateInput(current.invoice_date) : '', tax_rate: current.tax_rate ?? 13, amount_excluding_tax: current.amount_excluding_tax || 0, tax_amount: current.tax_amount || 0, annex: [...(current.annex || [])] };
    } catch (error) { if (generation === editorGeneration) ElMessage.error(failure(error)); return; }
  } else { editVisible.value = false; form.value = emptyForm(); }
  saveKey = operationKey(props.kind);
  editVisible.value = true;
  formRef.value?.clearValidate();
  void loadCustomers();
}
async function save() {
  if (disposed || saving.value || uploading.value) return;
  saving.value = true; formError.value = '';
  try {
  if (!pendingSave.value) {
  if (!(await formRef.value?.validate().catch(() => false)) || disposed) return;
  if (isAcceptance.value && form.value.issue_date && dayStart(form.value.due_date) < dayStart(form.value.issue_date)) { formError.value = '到期日期不能早于出票日期'; return; }
  if (isAcceptance.value && dayStart(form.value.received_date) < dayStart(form.value.issue_date)) { formError.value = '收票日期不能早于出票日期'; return; }
  if (!isAcceptance.value && (!Number.isFinite(form.value.tax_amount) || form.value.tax_amount < 0 || !Number.isFinite(form.value.tax_rate))) { formError.value = '请填写有效税额和税率'; return; }
  if (!isAcceptance.value && totalAmount.value > 9999999999.99) { formError.value = '价税合计不能超过 9,999,999,999.99 元'; return; }
  pendingSave.value = {
    customer_id: form.value.customer_id, annex: [...form.value.annex], remark: form.value.remark.trim(),
    ...(form.value.id ? { id: form.value.id, version: form.value.version } : { idempotency_key: saveKey }),
    ...(isAcceptance.value ? { bill_number: form.value.bill_number.trim(), amount: form.value.amount, received_date: dayStart(form.value.received_date), issue_date: dayStart(form.value.issue_date), due_date: dayStart(form.value.due_date) } : { invoice_number: form.value.invoice_number.trim(), invoice_date: dayStart(form.value.invoice_date), amount_excluding_tax: form.value.amount_excluding_tax, tax_rate: form.value.tax_rate, tax_amount: form.value.tax_amount, total_amount: totalAmount.value }),
  };
  }
    const response = await saveDocument(props.kind, pendingSave.value);
    if (disposed) return;
    if (response.code !== 200) throw responseError(response);
    pendingSave.value = undefined;
    saveWasUncertain = false;
    ElMessage.success('保存成功'); editVisible.value = false; await load();
  } catch (error) {
    if (disposed) return;
    saveWasUncertain ||= uncertain(error);
    if (!saveWasUncertain) pendingSave.value = undefined;
    formError.value = `${failure(error)}。${pendingSave.value ? retryMessage : '表单已保留，请按提示修正；记录已变化时请关闭后重新打开核对。'}`;
  }
  finally { if (!disposed) saving.value = false; }
}

const actionRecord = ref<CustomerDocument>();
const redeemVisible = ref(false);
const redeemMode = ref<'create' | 'link'>('create');
const redemptionDate = ref(today());
const redemptionAnnex = ref<string[]>([]);
const redemptionRemark = ref('');
const paymentID = ref('');
const payments = ref<PaymentOption[]>([]);
const paymentPage = ref(1);
const paymentTotal = ref(0);
const paymentLoading = ref(false);
let paymentGeneration = 0;
const acting = ref(false);
const actionError = ref('');
const pendingRedemption = ref<RedemptionRequest>();
const pendingVoid = ref<DocumentVoidRequest>();
let redemptionWasUncertain = false;
let voidWasUncertain = false;
let redeemKey = '';
async function openRedeem(row: CustomerDocument) {
  if (disposed || acting.value || uploading.value || pendingRedemption.value || pendingVoid.value) return;
  ++paymentGeneration; paymentLoading.value = false;
  actionRecord.value = row; redemptionDate.value = today(); redemptionAnnex.value = []; redemptionRemark.value = '';
  redeemMode.value = 'create'; paymentID.value = ''; payments.value = []; paymentPage.value = 1; paymentTotal.value = 0;
  actionError.value = ''; uploading.value = false; redeemKey = operationKey('acceptance-redeem'); redeemVisible.value = true;
}
async function loadPayments() {
  if (!actionRecord.value || !redeemVisible.value || redeemMode.value !== 'link' || acting.value || pendingRedemption.value) return;
  const generation = ++paymentGeneration;
  paymentLoading.value = true; paymentID.value = ''; payments.value = []; paymentTotal.value = 0; actionError.value = '';
  try {
    const response = await getAcceptancePayments({ customer_id: actionRecord.value.customer_id, amount: actionRecord.value.amount || 0, page: paymentPage.value, size: 10 });
    if (generation !== paymentGeneration) return;
    if (response.code !== 200) throw new Error(response.msg);
    payments.value = response.data.list || []; paymentTotal.value = response.data.total;
  } catch (error) { if (generation === paymentGeneration) { actionError.value = failure(error); payments.value = []; paymentTotal.value = 0; } }
  finally { if (generation === paymentGeneration) paymentLoading.value = false; }
}
watch(redeemMode, value => {
  ++paymentGeneration; paymentLoading.value = false; paymentID.value = ''; payments.value = []; paymentTotal.value = 0;
  if (value === 'link' && redeemVisible.value) { paymentPage.value = 1; void loadPayments(); }
}, { flush: 'sync' });
async function redeem() {
  if (disposed || !actionRecord.value || acting.value || uploading.value || paymentLoading.value) return;
  if (!pendingRedemption.value) {
  if (!redemptionDate.value) { actionError.value = '请选择到账日期'; return; }
  if (dayStart(redemptionDate.value) < dayStart(actionRecord.value.received_date || 0)) { actionError.value = '到账日期不能早于收票日期'; return; }
  if (redeemMode.value === 'link' && !paymentID.value) { actionError.value = '请选择要关联的已有回款'; return; }
  pendingRedemption.value = { id: actionRecord.value.id, version: actionRecord.value.version, redemption_date: dayStart(redemptionDate.value), redemption_annex: [...redemptionAnnex.value], remark: redemptionRemark.value.trim(), idempotency_key: redeemKey, ...(redeemMode.value === 'link' ? { transaction_id: paymentID.value } : {}) };
  }
  acting.value = true; actionError.value = '';
  try {
    const response = await redeemAcceptance(pendingRedemption.value);
    if (disposed) return;
    if (response.code !== 200) throw responseError(response);
    pendingRedemption.value = undefined;
    redemptionWasUncertain = false;
    ElMessage.success('到账已确认'); redeemVisible.value = false; await load();
  } catch (error) {
    if (disposed) return;
    redemptionWasUncertain ||= uncertain(error);
    if (!redemptionWasUncertain) pendingRedemption.value = undefined;
    actionError.value = `${failure(error)}。${pendingRedemption.value ? retryMessage : '请按提示修正；记录已变化时请关闭后重新打开核对。'}`;
  }
  finally { if (!disposed) acting.value = false; }
}
const voidVisible = ref(false);
const actionKind = ref<'void' | 'reverse'>('void');
const voidReason = ref('');
let voidKey = '';
function openVoid(row: CustomerDocument, action: 'void' | 'reverse' = 'void') {
  if (disposed || acting.value || pendingRedemption.value || pendingVoid.value) return;
  actionRecord.value = row; actionKind.value = action; voidReason.value = ''; actionError.value = ''; voidKey = operationKey(`${props.kind}-${action}`); voidVisible.value = true;
}
async function confirmVoid() {
  if (disposed || !actionRecord.value || acting.value) return;
  if (!pendingVoid.value && !voidReason.value.trim()) { actionError.value = actionKind.value === 'reverse' ? '请填写撤销原因' : '请填写作废原因'; return; }
  acting.value = true; actionError.value = '';
  try {
    pendingVoid.value ||= { id: actionRecord.value.id, version: actionRecord.value.version, reason: voidReason.value.trim(), idempotency_key: voidKey };
    const response = actionKind.value === 'reverse' ? await reverseAcceptance(pendingVoid.value) : await voidDocument(props.kind, pendingVoid.value);
    if (disposed) return;
    if (response.code !== 200) throw responseError(response);
    pendingVoid.value = undefined;
    voidWasUncertain = false;
    ElMessage.success(actionKind.value === 'reverse' ? '已撤销到账' : '已作废'); voidVisible.value = false; detailVisible.value = false; await load();
  } catch (error) {
    if (disposed) return;
    voidWasUncertain ||= uncertain(error);
    if (!voidWasUncertain) pendingVoid.value = undefined;
    actionError.value = `${failure(error)}。${pendingVoid.value ? retryMessage : '请关闭后重新打开核对记录。'}`;
  }
  finally { if (!disposed) acting.value = false; }
}

onMounted(() => { applyCustomerRoute(); if (can('list')) void loadCustomers(); });
watch(() => route.query.customer_id, () => applyCustomerRoute());
watch(detailVisible, value => { if (!value) { ++detailGeneration; detailLoading.value = false; } });
watch(redeemVisible, value => { if (!value) { ++paymentGeneration; paymentLoading.value = false; } });
onBeforeUnmount(() => { disposed = true; ++loadGeneration; ++customerGeneration; ++detailGeneration; ++paymentGeneration; ++editorGeneration; });
</script>

<template>
  <div class="customer-document-ledger">
    <el-result v-if="!can('list')" icon="warning" title="暂无查看权限" sub-title="请联系管理员开通对应台账权限" />
    <el-card v-else shadow="never">
      <template #header><div class="document-heading"><strong>{{ title }}</strong><el-button v-if="can('add')" type="primary" icon="Plus" @click="edit()">{{ isAcceptance ? '登记承兑' : '登记发票' }}</el-button></div></template>
      <el-alert :title="isAcceptance ? '登记承兑时不改变客户应收；确认到账时生成回款，或关联已有等额回款。' : '发票记录用于查询与留档，不改变客户应收余额。'" type="info" :closable="false" show-icon class="mb-3" />
      <el-form inline @submit.prevent="search">
        <el-form-item label="客户"><el-select v-model="query.customer_id" filterable remote clearable :remote-method="loadCustomers" :loading="customersLoading" placeholder="全部客户 / 输入名称" style="width: 220px"><el-option v-for="customer in customerOptions" :key="customer.id" :label="customer.name" :value="customer.id" /></el-select></el-form-item>
        <el-form-item :label="isAcceptance ? '票号' : '发票号码'"><el-input v-model="query.keyword" clearable placeholder="输入号码查询" /></el-form-item>
        <el-form-item label="状态"><el-select v-model="query.status" clearable placeholder="全部状态" style="width: 130px"><el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" /></el-select></el-form-item>
        <el-form-item :label="isAcceptance ? '收票日期' : '开票日期'"><el-date-picker v-model="dates" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" style="width: 260px" /></el-form-item>
        <el-form-item v-if="isAcceptance" label="到期日期"><el-date-picker v-model="dueDates" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" style="width: 260px" /></el-form-item>
        <el-form-item v-if="isAcceptance" label="到账日期"><el-date-picker v-model="redemptionDates" type="daterange" value-format="YYYY-MM-DD" start-placeholder="开始日期" end-placeholder="结束日期" style="width: 260px" /></el-form-item>
        <el-form-item v-if="isAcceptance" label="到期提醒"><el-select v-model="query.due_state" clearable placeholder="全部" style="width: 145px"><el-option label="已逾期未到账" value="overdue" /><el-option label="7 日内到期" value="soon" /></el-select></el-form-item>
        <el-form-item v-else label="税率"><el-select v-model="query.tax_rate" filterable allow-create clearable placeholder="全部 / 输入税率" style="width: 160px"><el-option v-for="rate in ['0', '1', '3', '6', '9', '13']" :key="rate" :label="`${rate}%`" :value="rate" /></el-select></el-form-item>
        <el-form-item><el-button native-type="submit" type="primary" icon="Search" :loading="loading">查询</el-button><el-button icon="RefreshRight" @click="reset">重置</el-button></el-form-item>
      </el-form>
      <div class="document-summary">
        <span>筛选结果：<b>{{ summary.count }}</b> 条</span>
        <template v-if="isAcceptance"><span>票面金额：<b>¥ {{ money(summary.amount) }}</b></span><span>待到账：<b>¥ {{ money(summary.pending_amount) }}</b></span><span>已到账：<b>¥ {{ money(summary.redeemed_amount) }}</b></span><span>逾期未到账：<b>¥ {{ money(summary.overdue_amount) }}</b></span></template>
        <template v-else><span>有效未税金额：<b>¥ {{ money(summary.amount_excluding_tax) }}</b></span><span>有效税额：<b>¥ {{ money(summary.tax_amount) }}</b></span><span>有效价税合计：<b>¥ {{ money(summary.total_amount) }}</b></span></template>
        <span>已作废金额：¥ {{ money(summary.voided_amount) }}</span>
      </div>
      <el-alert v-if="pageError" :title="pageError" type="error" :closable="false" class="mb-3" />
      <el-table v-loading="loading" :data="rows" border stripe row-key="id" empty-text="暂无符合条件的记录">
        <el-table-column prop="customer_name" label="客户" min-width="160" show-overflow-tooltip />
        <el-table-column :prop="isAcceptance ? 'bill_number' : 'invoice_number'" :label="isAcceptance ? '票号' : '发票号码'" min-width="180" show-overflow-tooltip />
        <el-table-column :label="isAcceptance ? '收票日期' : '开票日期'" width="115"><template #default="{ row }">{{ dateText(isAcceptance ? row.received_date : row.invoice_date) }}</template></el-table-column>
        <el-table-column v-if="isAcceptance" label="出票日期" width="115"><template #default="{ row }">{{ dateText(row.issue_date) }}</template></el-table-column>
        <el-table-column v-if="isAcceptance" label="到期日期" width="115"><template #default="{ row }">{{ dateText(row.due_date) }}</template></el-table-column>
        <el-table-column v-if="isAcceptance" label="到账日期" width="115"><template #default="{ row }">{{ dateText(row.redemption_date) }}</template></el-table-column>
        <el-table-column v-if="isAcceptance" label="到期提醒" width="115"><template #default="{ row }"><el-tag v-if="dueNotice(row)" :type="dueNotice(row) === '已逾期' ? 'danger' : 'warning'">{{ dueNotice(row) }}</el-tag><span v-else>—</span></template></el-table-column>
        <el-table-column v-if="!isAcceptance" label="税率" width="80" align="right"><template #default="{ row }">{{ row.tax_rate }}%</template></el-table-column>
        <el-table-column v-if="!isAcceptance" label="未税金额" width="140" align="right"><template #default="{ row }">¥ {{ money(row.amount_excluding_tax) }}</template></el-table-column>
        <el-table-column v-if="!isAcceptance" label="税额" width="130" align="right"><template #default="{ row }">¥ {{ money(row.tax_amount) }}</template></el-table-column>
        <el-table-column :label="isAcceptance ? '票面金额' : '价税合计'" width="150" align="right"><template #default="{ row }">¥ {{ money(isAcceptance ? row.amount : row.total_amount) }}</template></el-table-column>
        <el-table-column label="状态" width="100" align="center"><template #default="{ row }"><el-tag :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag></template></el-table-column>
        <el-table-column label="附件" width="80" align="center"><template #default="{ row }">{{ (row.annex || []).length }}</template></el-table-column>
        <el-table-column prop="remark" label="备注" min-width="150" show-overflow-tooltip />
        <el-table-column label="操作" :width="isAcceptance ? 270 : 190" fixed="right"><template #default="{ row }">
          <el-button link type="primary" @click="openDetail(row)">详情</el-button>
          <el-button v-if="can('edit') && row.status === (isAcceptance ? 'pending' : 'active')" link type="primary" @click="edit(row)">编辑</el-button>
          <el-button v-if="isAcceptance && can('redeem') && row.status === 'pending'" link type="success" @click="openRedeem(row)">确认到账</el-button>
          <el-button v-if="isAcceptance && can('reverse') && row.status === 'redeemed'" link type="danger" @click="openVoid(row, 'reverse')">撤销到账</el-button>
          <el-button v-if="can('void') && row.status === (isAcceptance ? 'pending' : 'active')" link type="danger" @click="openVoid(row)">作废</el-button>
        </template></el-table-column>
      </el-table>
      <el-pagination v-model:current-page="query.page" v-model:page-size="query.size" :total="total" :page-sizes="[10, 20, 50]" layout="total, sizes, prev, pager, next, jumper" background class="mt-3" @current-change="load" @size-change="search" />
    </el-card>

    <el-dialog v-model="editVisible" :title="`${form.id ? '编辑' : '登记'}${title}`" width="760px" destroy-on-close :close-on-click-modal="false" :close-on-press-escape="!saving && !uploading && !pendingSave" :show-close="!saving && !uploading && !pendingSave">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px" :disabled="saving || !!pendingSave">
        <el-form-item label="客户" prop="customer_id"><el-select v-model="form.customer_id" :disabled="!!form.id" filterable remote :remote-method="loadCustomers" :loading="customersLoading" placeholder="输入名称选择客户" style="width: 100%"><el-option v-for="customer in customerOptions" :key="customer.id" :label="customer.name" :value="customer.id" /></el-select></el-form-item>
        <template v-if="isAcceptance">
          <el-form-item label="票号" prop="bill_number"><el-input v-model="form.bill_number" maxlength="100" /></el-form-item>
          <el-form-item label="票面金额" prop="amount"><el-input-number v-model="form.amount" :min="0" :max="9999999999.99" :precision="2" :controls="false" /><span class="ml-2">元</span></el-form-item>
          <el-form-item label="收票日期" prop="received_date"><el-date-picker v-model="form.received_date" value-format="YYYY-MM-DD" type="date" /></el-form-item>
          <el-form-item label="出票日期" prop="issue_date"><el-date-picker v-model="form.issue_date" value-format="YYYY-MM-DD" type="date" /></el-form-item>
          <el-form-item label="到期日期" prop="due_date"><el-date-picker v-model="form.due_date" value-format="YYYY-MM-DD" type="date" /></el-form-item>
        </template>
        <template v-else>
          <el-form-item label="发票号码" prop="invoice_number"><el-input v-model="form.invoice_number" maxlength="100" /></el-form-item>
          <el-form-item label="开票日期" prop="invoice_date"><el-date-picker v-model="form.invoice_date" value-format="YYYY-MM-DD" type="date" /></el-form-item>
          <el-form-item label="未税金额" prop="amount_excluding_tax"><el-input-number v-model="form.amount_excluding_tax" :min="0" :max="9999999999.99" :precision="2" :controls="false" /><span class="ml-2">元</span></el-form-item>
          <el-form-item label="税率"><el-input-number v-model="form.tax_rate" :min="0" :max="100" :precision="2" :controls="false" /><span class="ml-2">%</span></el-form-item>
          <el-form-item label="税额" required><el-input-number v-model="form.tax_amount" :min="0" :max="9999999999.99" :precision="2" :controls="false" /><el-button link type="primary" class="ml-2" @click="calculateTax">按税率试算</el-button><el-text type="info" class="ml-2">可按票面税额调整</el-text></el-form-item>
          <el-form-item label="价税合计"><el-text>¥ {{ money(totalAmount) }}</el-text></el-form-item>
        </template>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" maxlength="1000" show-word-limit /></el-form-item>
        <el-form-item label="附件"><DocumentAttachments v-model="form.annex" :scope="kind" editable :disabled="saving || !!pendingSave" @uploading="uploading = $event" /></el-form-item>
      </el-form>
      <el-alert v-if="formError" :title="formError" type="error" :closable="false" show-icon />
      <template #footer><el-button :disabled="saving || uploading || !!pendingSave" @click="editVisible = false">取消</el-button><el-button type="primary" :loading="saving" :disabled="uploading" @click="save">{{ uploading ? '附件上传中' : pendingSave ? '重试确认' : '保存' }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="detailVisible" :title="`${title}详情`" width="850px" destroy-on-close>
      <div v-loading="detailLoading">
        <template v-if="detail">
          <el-descriptions border :column="2">
            <el-descriptions-item label="客户">{{ detail.customer_name }}</el-descriptions-item><el-descriptions-item label="状态"><el-tag :type="statusType(detail.status)">{{ statusLabel(detail.status) }}</el-tag></el-descriptions-item>
            <el-descriptions-item :label="isAcceptance ? '票号' : '发票号码'">{{ isAcceptance ? detail.bill_number : detail.invoice_number }}</el-descriptions-item><el-descriptions-item :label="isAcceptance ? '票面金额' : '价税合计'">¥ {{ money(isAcceptance ? detail.amount : detail.total_amount) }}</el-descriptions-item>
            <template v-if="isAcceptance"><el-descriptions-item label="收票日期">{{ dateText(detail.received_date) }}</el-descriptions-item><el-descriptions-item label="出票日期">{{ dateText(detail.issue_date) }}</el-descriptions-item><el-descriptions-item label="到期日期">{{ dateText(detail.due_date) }}</el-descriptions-item><el-descriptions-item label="到账日期">{{ dateText(detail.redemption_date) }}</el-descriptions-item><el-descriptions-item v-if="detail.transaction_id" label="关联回款" :span="2">{{ detail.transaction_id }}</el-descriptions-item></template>
            <template v-else><el-descriptions-item label="开票日期">{{ dateText(detail.invoice_date) }}</el-descriptions-item><el-descriptions-item label="税率">{{ detail.tax_rate }}%</el-descriptions-item><el-descriptions-item label="未税金额">¥ {{ money(detail.amount_excluding_tax) }}</el-descriptions-item><el-descriptions-item label="税额">¥ {{ money(detail.tax_amount) }}</el-descriptions-item></template>
            <el-descriptions-item label="备注" :span="2">{{ detail.remark || '—' }}</el-descriptions-item><el-descriptions-item label="登记人">{{ detail.creator_name || '—' }}</el-descriptions-item><el-descriptions-item label="登记时间">{{ dateText(detail.created_at, true) }}</el-descriptions-item>
          </el-descriptions>
          <el-divider content-position="left">{{ isAcceptance ? '承兑附件' : '发票附件' }}</el-divider><DocumentAttachments :model-value="detail.annex || []" />
          <template v-if="isAcceptance && detail.status === 'redeemed'"><el-divider content-position="left">到账凭证</el-divider><DocumentAttachments :model-value="detail.redemption_annex || []" /></template>
          <el-divider content-position="left">操作记录</el-divider>
          <el-table :data="detail.history || []" border empty-text="暂无操作记录"><el-table-column label="时间" width="175"><template #default="{ row }">{{ dateText(row.time, true) }}</template></el-table-column><el-table-column label="操作" width="125"><template #default="{ row }">{{ historyLabel(row.action) }}</template></el-table-column><el-table-column prop="operator_name" label="操作人" width="100" /><el-table-column prop="remark" label="说明" min-width="180" /></el-table>
        </template>
      </div>
    </el-dialog>

    <el-dialog v-model="redeemVisible" title="确认承兑到账" width="800px" destroy-on-close :close-on-click-modal="false" :close-on-press-escape="!acting && !uploading && !pendingRedemption" :show-close="!acting && !uploading && !pendingRedemption">
      <template v-if="actionRecord">
        <el-alert :title="`${actionRecord.customer_name} · 票号 ${actionRecord.bill_number} · ¥ ${money(actionRecord.amount)}`" type="info" :closable="false" class="mb-3" />
        <el-form label-width="110px" :disabled="acting || !!pendingRedemption">
          <el-form-item label="回款方式"><el-radio-group v-model="redeemMode"><el-radio value="create">生成新的回款</el-radio><el-radio value="link">关联已有回款</el-radio></el-radio-group></el-form-item>
          <el-alert :title="redeemMode === 'create' ? '确认后按票面金额生成一次回款，并减少客户应收。' : '仅可关联同客户、同金额且未关联承兑的有效回款；关联不会再次减少应收。'" type="warning" :closable="false" show-icon class="mb-3" />
          <template v-if="redeemMode === 'link'">
            <el-table v-loading="paymentLoading" :data="payments" border empty-text="没有可关联的等额有效回款"><el-table-column label="选择" width="70"><template #default="{ row }"><el-radio v-model="paymentID" :value="row.id"><span class="sr-only">选择 {{ row.code }}</span></el-radio></template></el-table-column><el-table-column prop="code" label="回款编号" min-width="150" /><el-table-column label="日期" width="115"><template #default="{ row }">{{ dateText(row.time) }}</template></el-table-column><el-table-column label="金额" width="130"><template #default="{ row }">¥ {{ money(row.amount) }}</template></el-table-column><el-table-column prop="remark" label="备注" min-width="150" /></el-table>
            <el-pagination v-model:current-page="paymentPage" :total="paymentTotal" :page-size="10" :disabled="acting || paymentLoading || !!pendingRedemption" layout="total, prev, pager, next" class="my-3" @current-change="loadPayments" />
          </template>
          <el-form-item label="到账日期" required><el-date-picker v-model="redemptionDate" value-format="YYYY-MM-DD" type="date" /></el-form-item>
          <el-form-item label="到账说明"><el-input v-model="redemptionRemark" type="textarea" :rows="2" maxlength="1000" /></el-form-item>
          <el-form-item label="到账凭证"><DocumentAttachments v-model="redemptionAnnex" scope="acceptance_redemption" editable :disabled="acting || !!pendingRedemption" @uploading="uploading = $event" /></el-form-item>
        </el-form>
        <el-alert v-if="actionError" :title="actionError" type="error" :closable="false" />
      </template>
      <template #footer><el-button :disabled="acting || uploading || !!pendingRedemption" @click="redeemVisible = false">取消</el-button><el-button type="primary" :loading="acting" :disabled="uploading || paymentLoading" @click="redeem">{{ pendingRedemption ? '重试确认' : '确认到账' }}</el-button></template>
    </el-dialog>

    <el-dialog v-model="voidVisible" :title="actionKind === 'reverse' ? '撤销承兑到账' : '作废记录'" width="520px" :close-on-click-modal="false" :close-on-press-escape="!acting && !pendingVoid" :show-close="!acting && !pendingVoid">
      <el-alert :title="actionKind === 'reverse' ? '自动生成的回款将撤销并恢复客户应收；关联的已有人工回款只解除关联，应收余额不变。操作历史保留。' : '作废后保留原记录、附件及操作记录。'" type="warning" :closable="false" class="mb-3" />
      <el-form label-width="80px"><el-form-item :label="actionKind === 'reverse' ? '撤销原因' : '作废原因'" required><el-input v-model="voidReason" :disabled="acting || !!pendingVoid" type="textarea" :rows="3" maxlength="1000" /></el-form-item></el-form>
      <el-alert v-if="actionError" :title="actionError" type="error" :closable="false" />
      <template #footer><el-button :disabled="acting || !!pendingVoid" @click="voidVisible = false">取消</el-button><el-button type="danger" :loading="acting" @click="confirmVoid">{{ pendingVoid ? '重试确认' : actionKind === 'reverse' ? '确认撤销到账' : '确认作废' }}</el-button></template>
    </el-dialog>
  </div>
</template>

<style scoped>
.document-heading { display: flex; align-items: center; justify-content: space-between; }
.document-summary { display: flex; flex-wrap: wrap; gap: 12px 24px; padding: 12px; margin-bottom: 16px; background: var(--el-fill-color-light); border-radius: 4px; font-size: 14px; }
.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
</style>
