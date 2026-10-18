import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import { transformSync } from 'esbuild';
import * as vue from 'vue';
import { compileScript, parse } from 'vue/compiler-sfc';

function evaluate(source, dependencies = {}) {
  const { code } = transformSync(source.replaceAll('import.meta.env', '({VITE_OSS_DOMAIN:"https://files.invalid/images/",VITE_IMAGE_UPLOAD:"/images"})'), { loader: 'ts', format: 'cjs', target: 'es2020' });
  const module = { exports: {} };
  vm.runInThisContext(`(function(module,exports,require){${code}\n})`)(module, module.exports, name => {
    assert.ok(name in dependencies, `Unexpected dependency: ${name}`);
    return dependencies[name];
  });
  return module.exports;
}
const source = path => readFileSync(new URL(path, import.meta.url), 'utf8');
const utils = evaluate(source('../src/views/finance/components/document-utils.ts'));
function mount(path, dependencies, props) {
  const { descriptor } = parse(source(path), { filename: path });
  const script = compileScript(descriptor, { id: path });
  const resolved = { vue, ...dependencies };
  for (const match of script.content.matchAll(/from\s+["']([^"']+\.vue)["']/g)) resolved[match[1]] = { default: {} };
  const component = evaluate(script.content, resolved).default;
  component.render = () => null;
  const renderer = vue.createRenderer({ createElement: () => ({}), createText: () => ({}), createComment: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null });
  const app = renderer.createApp(component, props);
  const instance = app.mount({});
  return { state: instance.$.setupState, unmount: () => app.unmount() };
}
const settle = async () => { for (let i = 0; i < 5; i++) { await Promise.resolve(); await vue.nextTick(); } };
const deferred = () => { let resolve; let reject; const promise = new Promise((yes, no) => { resolve = yes; reject = no; }); return { promise, resolve, reject }; };
const ok = data => ({ code: 200, msg: '成功', data });
const emptyPage = () => ok({ total: 0, list: [], summary: { count: 0 } });
const acceptance = { id: 'acceptance-a', customer_id: 'customer-a', customer_name: '客户甲', bill_number: 'BILL-1', amount: 123, status: 'pending', version: 3, annex: [], received_date: 1792200000, issue_date: 1792100000, due_date: 1800000000 };
function ledger(kind = 'acceptance', overrides = {}, allowed = true) {
  const messages = [];
  const api = { getDocuments: async () => emptyPage(), getDocument: async () => ok(acceptance), saveDocument: async () => ok(acceptance), getAcceptancePayments: async () => ok({ total: 0, list: [] }), redeemAcceptance: async () => ok(acceptance), reverseAcceptance: async () => ok(acceptance), voidDocument: async () => ok(acceptance), ...overrides };
  const mounted = mount('../src/views/finance/components/CustomerDocumentLedger.vue', {
    'vue-router': { useRoute: () => vue.reactive({ query: {} }) },
    'element-plus': { ElMessage: { error: value => messages.push(value), success: value => messages.push(value), warning: value => messages.push(value) } },
    '@/store/modules/auth': { useAuthStore: () => ({ authButtonListGet: allowed ? ['list', 'add', 'edit', 'redeem', 'reverse', 'void'].map(action => ({ perms: `finance:${kind}:${action}` })) : [] }) },
    '@/api/customer': { reqCustomers: async () => ok({ total: 1, list: [{ id: 'customer-a', name: '客户甲' }] }) },
    '@/api/customer_documents': api,
    './document-utils': utils,
  }, { kind });
  return { ...mounted, messages };
}

test('invoice trial tax rounds half-up in cents and totals avoid binary floating point drift', () => {
  assert.deepEqual(utils.invoiceTotals(0.5, 1), { tax_amount: 0.01, total_amount: 0.51 });
  assert.deepEqual(utils.invoiceTotals(100.01, 13), { tax_amount: 13, total_amount: 113.01 });
  assert.deepEqual(utils.invoiceTotals(0.29, 13), { tax_amount: 0.04, total_amount: 0.33 });
  assert.deepEqual(utils.invoiceTotals(9.99, 0), { tax_amount: 0, total_amount: 9.99 });
});

test('query dates include the full selected final day, and old responses cannot replace current results', async () => {
  const calls = [];
  const slow = deferred();
  const view = ledger('acceptance', { getDocuments: async (_kind, query) => { calls.push(query); return query.keyword === 'old' ? slow.promise : ok({ total: 1, list: [{ ...acceptance, bill_number: query.keyword || 'initial' }], summary: { count: 1 } }); } });
  await settle();
  view.state.dates = ['1792200000', '1792286400'];
  view.state.query.keyword = 'old'; const older = view.state.load();
  view.state.query.keyword = 'new'; await view.state.load();
  slow.resolve(ok({ total: 1, list: [{ ...acceptance, bill_number: 'old' }], summary: { count: 1 } })); await older;
  assert.equal(view.state.rows[0].bill_number, 'new');
  assert.equal(calls.at(-1).date_end, utils.dayStart('1792286400') + 86399);
  view.unmount();
});

test('no list permission means no finance API load', async () => {
  let calls = 0;
  const view = ledger('invoice', { getDocuments: async () => { calls++; return emptyPage(); } }, false);
  await settle(); await view.state.load(); assert.equal(calls, 0); view.unmount();
});

test('due reminders cover exactly today and the following six calendar days', async () => {
  const view = ledger(); await settle();
  const today = utils.dayStart(Math.floor(Date.now() / 1000));
  assert.equal(view.state.dueNotice({ ...acceptance, due_date: today - 86400 }), '已逾期');
  assert.equal(view.state.dueNotice({ ...acceptance, due_date: today + 6 * 86400 }), '7 日内到期');
  assert.equal(view.state.dueNotice({ ...acceptance, due_date: today + 7 * 86400 }), '');
  assert.equal(view.state.dueNotice({ ...acceptance, due_date: today - 86400, status: 'redeemed' }), '');
  view.unmount();
});

test('invoice edit preserves recorded tax adjustment; save sends adjusted tax and the exact resulting total', async () => {
  const invoice = { ...acceptance, id: 'invoice-a', status: 'active', invoice_number: 'INV-1', invoice_date: 1792200000, amount_excluding_tax: 100, tax_rate: 13, tax_amount: 12.99, total_amount: 112.99 };
  let sent;
  const view = ledger('invoice', { getDocument: async () => ok(invoice), saveDocument: async (_kind, payload) => { sent = payload; return ok(invoice); } });
  await settle(); await view.state.edit(invoice);
  assert.equal(view.state.form.tax_amount, 12.99);
  assert.equal(view.state.totalAmount, 112.99);
  view.state.formRef = { validate: async () => true, clearValidate() {} };
  await view.state.save();
  assert.equal(sent.tax_amount, 12.99); assert.equal(sent.total_amount, 112.99); assert.equal(sent.version, 3);
  view.unmount();
});

test('failed creation retains form and idempotency key across retry', async () => {
  const sent = [];
  const view = ledger('acceptance', { saveDocument: async (_kind, payload) => { sent.push(payload); if (sent.length === 1) throw new Error('网络超时'); return ok(acceptance); } });
  await settle(); await view.state.edit();
  Object.assign(view.state.form, { customer_id: 'customer-a', bill_number: 'BILL-1', amount: 123, received_date: '1792200000', issue_date: '1792100000', due_date: '1800000000', annex: ['acceptance/proof.pdf'] });
  view.state.formRef = { validate: async () => true, clearValidate() {} };
  await view.state.save();
  assert.equal(view.state.editVisible, true); assert.equal(view.state.form.annex[0], 'acceptance/proof.pdf');
  view.state.form.remark = 'a later edit must not change the uncertain request';
  view.state.form.annex.push('acceptance/later.pdf');
  await view.state.save();
  assert.equal(sent.length, 2); assert.ok(sent[0].idempotency_key); assert.equal(sent[0].idempotency_key, sent[1].idempotency_key);
  assert.deepEqual(sent[0], sent[1]); assert.equal(sent[1].remark, ''); assert.deepEqual(sent[1].annex, ['acceptance/proof.pdf']);
  view.unmount();
});

test('acceptance dates reject impossible order but allow recording an overdue bill', async () => {
  const sent = [];
  const view = ledger('acceptance', { saveDocument: async (_kind, payload) => { sent.push(payload); return ok(acceptance); } });
  await settle(); await view.state.edit();
  view.state.formRef = { validate: async () => true, clearValidate() {} };
  Object.assign(view.state.form, { customer_id: 'customer-a', bill_number: 'OVERDUE-1', amount: 123, issue_date: '1792200000', received_date: '1792100000', due_date: '1792300000' });
  await view.state.save(); assert.equal(sent.length, 0); assert.match(view.state.formError, /收票日期不能早于出票日期/);
  view.state.form.received_date = '1792400000';
  await view.state.save(); assert.equal(sent.length, 1, 'receipt after maturity is a valid historical registration');
  view.unmount();
});

test('link existing payment sends that transaction once, and a pending submission blocks a double click', async () => {
  const pending = deferred(); const sent = [];
  const view = ledger('acceptance', { getAcceptancePayments: async () => ok({ total: 1, list: [{ id: 'payment-a', amount: 123 }] }), redeemAcceptance: async payload => { sent.push(payload); return pending.promise; } });
  await settle(); await view.state.openRedeem(acceptance);
  view.state.redeemMode = 'link'; await settle();
  await view.state.redeem(); assert.equal(sent.length, 0);
  view.state.paymentID = 'payment-a'; view.state.redemptionAnnex = ['acceptance_redemption/proof.pdf'];
  const submitting = view.state.redeem(); await view.state.redeem();
  assert.equal(sent.length, 1); assert.equal(sent[0].transaction_id, 'payment-a'); assert.equal(sent[0].version, 3);
  assert.deepEqual(sent[0].redemption_annex, ['acceptance_redemption/proof.pdf']);
  pending.resolve(ok({ ...acceptance, status: 'redeemed' })); await submitting; view.unmount();
});

test('scoped uploads reuse /images form fields, accept documents, enforce size/count, and retain legacy defaults', async () => {
  const imageQueries = [];
  const dependencies = {
    '@/api/image': { reqImages: async query => { imageQueries.push(query); return ok({ total: 0, list: [] }); } },
    '@/store/modules/user.ts': { useUserStore: () => ({ token: 'token' }) },
    'element-plus': { ElMessage: { error() {}, success() {}, warning() {} } },
  };
  const view = mount('../src/components/ImageUpload/index.vue', dependencies, { fileMode: true, scope: 'invoice', urls: ['invoice/one.pdf'], limit: 2 });
  assert.deepEqual(view.state.uploadData, { scope: 'invoice' });
  await view.state.loadImages(); assert.equal(imageQueries[0].scope, 'invoice');
  assert.equal(view.state.beforeUpload({ uid: 1, name: 'two.pdf', size: 1024 }), true);
  assert.equal(view.state.beforeUpload({ uid: 2, name: 'three.pdf', size: 1024 }), false);
  view.state.removeUpload({ uid: 1 });
  assert.equal(view.state.uploads.size, 0, 'canceling an upload must release the dialog/save lock');
  assert.equal(view.state.beforeUpload({ uid: 3, name: 'too-large.pdf', size: 20 * 1024 * 1024 + 1 }), false);
  assert.equal(view.state.beforeUpload({ uid: 4, name: 'run.exe', size: 1024 }), false);
  assert.equal(view.state.fileURL('invoice/proof.pdf'), 'https://files.invalid/images/invoice/proof.pdf');
  assert.equal(view.state.isImage('invoice/proof.pdf'), false);
  view.unmount();
  const legacy = mount('../src/components/ImageUpload/index.vue', dependencies, {});
  assert.deepEqual(legacy.state.uploadData, {});
  await legacy.state.loadImages(); assert.equal(imageQueries[1].scope, undefined);
  assert.equal(legacy.state.beforeUpload({ uid: 5, name: 'old.png', size: 1024 }), true);
  legacy.unmount();
});

test('reversal requires a reason and uses its dedicated action rather than void', async () => {
  let reversed = 0; let voided = 0; let sent;
  const view = ledger('acceptance', { reverseAcceptance: async payload => { reversed++; sent = payload; return ok(acceptance); }, voidDocument: async () => { voided++; return ok(acceptance); } });
  await settle(); view.state.openVoid({ ...acceptance, status: 'redeemed' }, 'reverse');
  await view.state.confirmVoid(); assert.equal(reversed, 0); assert.equal(voided, 0);
  view.state.voidReason = '到账日期填错，撤销后重新登记';
  await view.state.confirmVoid(); assert.equal(reversed, 1); assert.equal(voided, 0); assert.equal(sent.reason, '到账日期填错，撤销后重新登记'); assert.equal(sent.version, 3);
  view.unmount();
});

function fillAcceptance(state) {
  Object.assign(state.form, { customer_id: 'customer-a', bill_number: 'BILL-1', amount: 123, received_date: '2026-10-17', issue_date: '2026-10-16', due_date: '2026-11-01', annex: [] });
  state.formRef = { validate: async () => true, clearValidate() {} };
}

test('save locks before async validation and releases on validation failure', async () => {
  const validation = deferred(); const sent = [];
  const view = ledger('acceptance', { saveDocument: async (_kind, payload) => { sent.push(payload); return ok(acceptance); } });
  await settle(); await view.state.edit(); fillAcceptance(view.state);
  view.state.formRef.validate = () => validation.promise;
  const first = view.state.save(); await view.state.save();
  assert.equal(view.state.saving, true); assert.equal(sent.length, 0);
  validation.resolve(true); await first; assert.equal(sent.length, 1);
  await view.state.edit(); fillAcceptance(view.state); view.state.formRef.validate = async () => false;
  await view.state.save(); assert.equal(view.state.saving, false); assert.equal(sent.length, 1);
  view.unmount();
});

test('late edit responses cannot replace a newer record, a new form, or reopen after unmount', async () => {
  let slow = deferred();
  const view = ledger('acceptance', { getDocument: async (_kind, id) => id === 'slow' ? slow.promise : ok({ ...acceptance, id, bill_number: id }) });
  await settle();
  let older = view.state.edit({ ...acceptance, id: 'slow' });
  await view.state.edit({ ...acceptance, id: 'newer' });
  slow.resolve(ok({ ...acceptance, id: 'slow' })); await older; assert.equal(view.state.form.id, 'newer');
  slow = deferred(); older = view.state.edit({ ...acceptance, id: 'slow' });
  await view.state.edit(); slow.resolve(ok({ ...acceptance, id: 'slow' })); await older; assert.equal(view.state.form.id, '');
  slow = deferred(); older = view.state.edit({ ...acceptance, id: 'slow' }); view.unmount();
  slow.resolve(ok({ ...acceptance, id: 'slow' })); await older; assert.equal(view.state.form.id, '');
});

test('known validation failure releases frozen save content for a corrected request', async () => {
  const sent = [];
  const view = ledger('acceptance', { saveDocument: async (_kind, payload) => { sent.push(payload); return sent.length === 1 ? { code: 400, msg: '票号错误' } : ok(acceptance); } });
  await settle(); await view.state.edit(); fillAcceptance(view.state);
  await view.state.save(); assert.equal(view.state.pendingSave, undefined);
  view.state.form.bill_number = 'CORRECTED'; await view.state.save(); assert.equal(sent[1].bill_number, 'CORRECTED'); view.unmount();
});

test('redeem and reversal retry exact original content after uncertain outcomes', async () => {
  for (const action of ['redeem', 'reverse', 'void']) {
    const sent = [];
    const send = async payload => { sent.push(structuredClone(vue.toRaw(payload))); if (sent.length === 1) throw new Error('timeout'); return ok(acceptance); };
    const view = ledger('acceptance', { redeemAcceptance: send, reverseAcceptance: send, voidDocument: async (_kind, payload) => send(payload) });
    await settle();
    if (action === 'redeem') {
      await view.state.openRedeem(acceptance); view.state.redemptionDate = '2026-10-18';
      view.state.redemptionRemark = 'original'; await view.state.redeem();
      view.state.redemptionRemark = 'later'; view.state.redemptionAnnex = ['acceptance_redemption/later.pdf'];
      await view.state.redeem();
    } else {
      view.state.openVoid(acceptance, action); view.state.voidReason = 'original'; await view.state.confirmVoid();
      view.state.voidReason = 'later'; await view.state.confirmVoid();
    }
    assert.equal(sent.length, 2, action); assert.deepEqual(sent[0], sent[1], action); view.unmount();
  }
});

test('leaving existing-payment mode invalidates its pending error and selection', async () => {
  const slow = deferred(); const view = ledger('acceptance', { getAcceptancePayments: () => slow.promise });
  await settle(); await view.state.openRedeem(acceptance); view.state.redeemMode = 'link'; await settle();
  assert.equal(view.state.paymentLoading, true); view.state.redeemMode = 'create';
  slow.reject(new Error('stale failure')); await settle();
  assert.equal(view.state.paymentLoading, false); assert.equal(view.state.actionError, ''); assert.equal(view.state.paymentID, ''); view.unmount();
});

test('financial date inputs, payloads, display and day boundaries agree in different client time zones', async () => {
  const originalTZ = process.env.TZ;
  try {
    for (const zone of ['UTC', 'Asia/Shanghai', 'America/Los_Angeles']) {
      process.env.TZ = zone;
      const start = Date.parse('2026-10-18T00:00:00+08:00') / 1000;
      assert.equal(utils.dayStart('2026-10-18'), start, zone);
      assert.equal(utils.dayEnd('2026-10-18'), start + 86399, zone);
      assert.equal(utils.dateInput(start), '2026-10-18', zone);
      let sent; const view = ledger('acceptance', { saveDocument: async (_kind, payload) => { sent = payload; return ok(acceptance); } });
      await settle(); await view.state.edit(); fillAcceptance(view.state); view.state.form.received_date = '2026-10-18';
      await view.state.save(); assert.equal(sent.received_date, start, zone); assert.match(view.state.dateText(start), /2026\/10\/18/, zone); view.unmount();
    }
  } finally { if (originalTZ === undefined) delete process.env.TZ; else process.env.TZ = originalTZ; }
  assert.doesNotMatch(source('../src/views/finance/components/CustomerDocumentLedger.vue'), /value-format="X"/);
});

test('attachments preserve multiple completions in the same render cycle and enforce ten files', () => {
  const emitted = [];
  const view = mount('../src/views/finance/components/DocumentAttachments.vue', { 'element-plus': { ElMessage: { warning() {} } } }, { modelValue: [], editable: true, 'onUpdate:modelValue': value => emitted.push(value) });
  view.state.add('invoice/one.pdf'); view.state.add('invoice/two.pdf'); view.state.add('invoice/one.pdf');
  assert.deepEqual(emitted.at(-1), ['invoice/one.pdf', 'invoice/two.pdf']);
  for (let i = 0; i < 20; i++) view.state.add(`invoice/${i}.pdf`);
  assert.equal(emitted.at(-1).length, 10);
  view.state.remove('invoice/one.pdf'); view.state.add('invoice/replacement.pdf');
  assert.equal(emitted.at(-1).length, 10); assert.ok(emitted.at(-1).includes('invoice/replacement.pdf'));
  view.unmount();
});

test('upload gallery handles failures and stale responses; canceled callbacks cannot add files', async () => {
  const slow = deferred(); let fail = true; const selections = []; const user = vue.reactive({ token: 'first' });
  const view = mount('../src/components/ImageUpload/index.vue', {
    '@/api/image': { reqImages: async query => { if (query.name === 'old') return slow.promise; if (fail) throw new Error('network'); return ok({ total: 1, list: [{ url: 'invoice/new.pdf' }] }); } },
    '@/store/modules/user.ts': { useUserStore: () => user },
    'element-plus': { ElMessage: { error() {}, success() {}, warning() {} } },
  }, { fileMode: true, scope: 'invoice', onSelect: path => selections.push(path) });
  await view.state.loadImages(); assert.equal(view.state.loading, false); assert.equal(view.state.images.length, 0);
  fail = false; view.state.form.name = 'old'; const older = view.state.loadImages();
  view.state.form.name = 'new'; await view.state.loadImages(); slow.resolve(ok({ total: 0, list: [] })); await older;
  assert.equal(view.state.images[0].url, 'invoice/new.pdf');
  user.token = 'renewed'; assert.equal(view.state.uploadHeaders.Authorization, 'renewed');
  const file = { uid: 42, name: 'proof.pdf', size: 10 };
  view.state.beforeUpload(file); view.state.removeUpload(file); view.state.handleUpload(ok({ url: 'invoice/canceled.pdf' }), file);
  assert.equal(selections.length, 0); view.unmount();
});

test('timeout followed by 409 or 401 never unlocks or changes the original financial request', async () => {
  for (const action of ['save', 'redeem', 'reverse', 'void']) {
    const sent = [];
    const send = async payload => {
      sent.push(structuredClone(vue.toRaw(payload)));
      if (sent.length === 1) throw new Error('timeout');
      if (sent.length < 4) throw { response: { status: sent.length === 2 ? 409 : 401, data: { msg: 'cannot prove the first attempt failed' } } };
      return ok(acceptance);
    };
    const view = ledger('acceptance', { saveDocument: async (_kind, payload) => send(payload), redeemAcceptance: send, reverseAcceptance: send, voidDocument: async (_kind, payload) => send(payload) });
    await settle();
    if (action === 'save') { await view.state.edit(); fillAcceptance(view.state); }
    else if (action === 'redeem') { await view.state.openRedeem(acceptance); view.state.redemptionDate = '2026-10-18'; }
    else { view.state.openVoid(acceptance, action); view.state.voidReason = 'original'; }
    const submit = () => action === 'save' ? view.state.save() : action === 'redeem' ? view.state.redeem() : view.state.confirmVoid();
    const pendingKey = action === 'save' ? 'pendingSave' : action === 'redeem' ? 'pendingRedemption' : 'pendingVoid';
    for (let i = 0; i < 3; i++) { await submit(); assert.ok(view.state[pendingKey], `${action} attempt ${i + 1}`); }
    view.state.form.remark = 'later'; view.state.redemptionRemark = 'later'; view.state.voidReason = 'later';
    await submit(); assert.equal(view.state[pendingKey], undefined);
    for (const request of sent) assert.deepEqual(request, sent[0], action);
    view.unmount();
  }
});

function transaction(overrides = {}) {
  return mount('../src/views/business_partner/customer/components/Transaction.vue', {
    '@/api/customer': { reqCustomerTransactions: async () => ok({ total: 0, list: [] }), reqAddCustomerTransaction: async () => ok({}), ...overrides },
    'element-plus': { ElMessage: { error() {}, success() {} } },
    '@/utils/time.ts': { DateFormat: value => `${value}` },
    '@/enums/customer.ts': { CustomerTransactionTypes: [{ value: 'payment', direction: 'receivable_decrease' }], CustomerTransactionTypeLabels: {}, CustomerTransactionSourceLabels: {}, CustomerTransactionStatusLabels: {} },
    dayjs: () => ({ unix: () => Math.floor(Date.now() / 1000) }),
  }, { customer: { id: 'customer-a', name: '客户甲' } });
}

test('manual customer payment locks validation, preserves unknown requests, and starts a clean form after success', async () => {
  const validation = deferred(); const sent = [];
  const view = transaction({ reqAddCustomerTransaction: async payload => {
    sent.push(structuredClone(vue.toRaw(payload)));
    if (sent.length === 1) throw new Error('timeout');
    if (sent.length === 2) throw { response: { status: 409 } };
    return ok({});
  } });
  await settle(); view.state.handleRecord();
  Object.assign(view.state.recordForm, { time: 1792200000, amount: 123, remark: 'original', annex: ['proof.jpg'] });
  view.state.recordFormRef = { validate: () => validation.promise };
  const first = view.state.handleAdd(); await view.state.handleAdd(); assert.equal(sent.length, 0);
  validation.resolve(true); await first; assert.equal(sent.length, 1); assert.equal(view.state.submitting, false);
  await view.state.handleAdd(); assert.ok(view.state.pendingRecord);
  view.state.recordForm.amount = 456; view.state.recordForm.annex.push('later.jpg'); await view.state.handleAdd();
  assert.deepEqual(sent[0], sent[1]); assert.deepEqual(sent[0], sent[2]);
  view.state.handleRecord(); assert.equal(view.state.recordForm.amount, 0); assert.deepEqual(view.state.recordForm.annex, []);
  assert.notEqual(view.state.recordForm.idempotency_key, sent[0].idempotency_key); view.unmount();
});

test('customer transaction reads clear loading on failure and ignore older pages', async () => {
  const slow = deferred(); let fail = true;
  const view = transaction({ reqCustomerTransactions: async query => {
    if (fail) throw new Error('network');
    return query.page === 1 ? slow.promise : ok({ total: 1, list: [{ id: 'new-page' }] });
  } });
  await settle(); assert.equal(view.state.loading, false); assert.equal(view.state.total, 0);
  fail = false; const older = view.state.getTransactions(); view.state.form.page = 2; await view.state.getTransactions();
  slow.resolve(ok({ total: 0, list: [] })); await older; assert.equal(view.state.list[0].id, 'new-page'); view.unmount();
});
