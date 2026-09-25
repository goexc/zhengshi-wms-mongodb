import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import vm from 'node:vm';
import { stripTypeScriptTypes } from 'node:module';
import { stripImports } from './helpers/source.mjs';

const source = path => readFileSync(new URL('../' + path, import.meta.url), 'utf8');
function harness(path, props = {}) {
  const pending = [], selected = [];
  const request = (...args) => new Promise((resolve, reject) => pending.push({ args, resolve, reject }));
  const context = vm.createContext({
    ref: value => ({ value }), computed: fn => ({ get value() { return fn(); } }), watch: () => {},
    defineOptions: () => {}, defineProps: () => props, defineEmits: () => (name, data) => selected.push({ name, data }),
    useStore: () => ({ user: { hasCredential: () => true, canAccess: () => true } }),
    getMaterials: request, getOutboundPage: request, fetchPartners: request, fetchCustomerTransactions: request, fetchPlans: request,
    findCustomerChoices: request, getOutboundSummary: request,
    summarizeOutbound: rows => ({ lineCount: rows.length, amount: rows.reduce((sum, row) => sum + row.quantity * row.price, 0) }),
    outboundDateTimestamp: () => 100, outboundQuery: page => ({ page, size: 20 }),
    onMounted: () => {}, onUnmounted: () => {}, onLoad: () => {}, onShow: () => {}, onUnload: () => {},
    uni: { $on: () => {}, $off: () => {} }, router: {}, setTimeout, clearTimeout,
    errorMessage: error => error.message
  });
  const script = source(path).match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
  vm.runInContext(stripTypeScriptTypes(stripImports(script)), context);
  return { pending, selected, run: code => vm.runInContext(code, context) };
}
const targets = [
  ['pages/material/index.uvue', 'items', 'loadError'],
  ['pages/outbound/order.uvue', 'orders', 'error'],
  ['pages/business/components/PartnerList.uvue', 'items', 'error'],
  ['pages/business/customer-transactions.uvue', 'items', 'error'],
  ['pages/plan/index.uvue', 'items', 'error']
];
for (const [path, rows, error] of targets) {
  test(`${path}: latest page wins, failed page retries, shrinking totals clamp the page`, async () => {
    const h = harness(path, { kind: 'customer' });
    const first = h.run('load(1)');
    h.pending[0].resolve({ total: 60, list: [{ id: 'first' }] }); await first;
    const second = h.run('load(2)');
    assert.equal(h.run(`${rows}.value.length`), 0, 'old-page data must not remain under the new page number');
    const third = h.run('load(3)');
    h.pending[2].resolve({ total: 60, list: [{ id: 'third' }] }); await third;
    h.pending[1].resolve({ total: 60, list: [{ id: 'stale-second' }] }); await second;
    assert.equal(h.run(`${rows}.value[0].id`), 'third');
    assert.equal(h.run('currentPage.value'), 3);
    const fail = h.run('load(2)'); h.pending[3].reject(new Error('offline')); await fail;
    assert.equal(h.run(`${error}.value`), 'offline');
    assert.equal(h.run('currentPage.value'), 2);
    h.run('retry()');
    const retry = h.pending[4];
    const query = path.includes('PartnerList') ? retry.args[1] : retry.args[0];
    const requestedPage = path.includes('customer-transactions') ? retry.args[1] : typeof query === 'object' ? query.page : query;
    assert.equal(requestedPage, 2);
    retry.resolve({ total: 1, list: [] }); await new Promise(setImmediate);
    assert.equal(h.run('currentPage.value'), 1);
    assert.equal(h.run('loading.value'), true);
    h.pending[5].resolve({ total: 1, list: [{ id: 'remaining' }] }); await new Promise(setImmediate);
    assert.equal(h.run(`${rows}.value[0].id`), 'remaining');
    assert.equal(h.run('loading.value'), false);
    h.run('refresh()');
    h.pending[6].resolve({ total: 0, list: [] }); await new Promise(setImmediate);
    assert.equal(h.run('total.value'), 0);
    assert.equal(h.run('refreshing.value'), false);
  });
}
test('report pages only slice details and preserve the complete query totals', async () => {
  const h = harness('pages/outbound/report.uvue');
  h.run("customerID.value = 'customer';");
  const request = h.run('queryReport()');
  h.pending[0].resolve(Array.from({ length: 45 }, (_, i) => ({ id: i, quantity: 2, price: 3 }))); await request;
  assert.equal(h.run('visibleRows.value.length'), 20);
  h.run('changePage(3)');
  assert.equal(h.run('visibleRows.value.length'), 5);
  assert.equal(h.run('visibleRows.value[0].id'), 40);
  assert.equal(h.run('totals.value.amount'), 270);
  assert.equal(h.run('totals.value.lineCount'), 45);
  h.run("selectCustomer({id:'new',name:'New'})");
  assert.equal(h.run('currentPage.value'), 1);
  assert.equal(h.run('rows.value.length'), 0);
});
test('customer popup rejects late searches and closing never commits a selection', async () => {
  const h = harness('components/wms/CustomerPicker.uvue', { selectedId: 'second', disabled: false });
  h.run('open()');
  const newer = h.run("keyword.value = 'new'; searchNow()");
  h.pending[1].resolve([{ id: 'second', name: 'New', code: 'C2' }]); await newer;
  h.pending[0].resolve([{ id: 'first', name: 'Old', code: 'C1' }]); await new Promise(setImmediate);
  assert.equal(h.run('choices.value[0].id'), 'second');
  h.run('cancel()'); assert.equal(h.selected.length, 0);
  h.run('open()'); h.run('cancel()');
  h.pending[2].resolve([{ id: 'late' }]); await new Promise(setImmediate);
  assert.equal(h.run('choices.value.length'), 0);
  h.run('open()'); h.pending[3].resolve([{ id: 'chosen', name: 'Chosen', code: 'C' }]); await new Promise(setImmediate);
  h.run("choose(choices.value[0])");
  assert.equal(h.selected[0].data.id, 'chosen');
  assert.equal(h.run('visible.value'), false);
});
test('closing a list view cancels measurement and ignores an absent or detached node', () => {
  const content = source('uni_modules/cool-ui/components/cl-list-view/cl-list-view.uvue');
  const script = content.slice(content.indexOf('let measureTimer ='), content.indexOf('// 组件挂载后的初始化逻辑'));
  let measure, cleanup, complete, cancelled;
  const context = vm.createContext({
    setTimeout: callback => { measure = callback; return 1; }, clearTimeout: id => { cancelled = id; },
    onUnmounted: callback => { cleanup = callback; }, proxy: {}, scrollerHeight: { value: 0 },
    isEmpty: value => !value || value.length === 0,
    uni: { createSelectorQuery: () => ({ in() { return this; }, select() { return this; }, boundingClientRect() { return this; }, exec(callback) { complete = callback; } }) }
  });
  vm.runInContext(stripTypeScriptTypes(script), context);
  vm.runInContext('getScrollerHeight()', context); measure(); complete([null]);
  assert.equal(vm.runInContext('scrollerHeight.value', context), 0);
  complete([{ height: 300 }]);
  assert.equal(vm.runInContext('scrollerHeight.value', context), 300);
  vm.runInContext('getScrollerHeight()', context); cleanup(); complete([{ height: 500 }]);
  assert.equal(cancelled, 1);
  assert.equal(vm.runInContext('scrollerHeight.value', context), 300);
});
