import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { stripTypeScriptTypes } from 'node:module';
import { test } from 'node:test';
import { stripImports } from './helpers/source.mjs';

const read = (relative) => readFileSync(new URL(`../${relative}`, import.meta.url), 'utf8');
const strip = (source) => stripTypeScriptTypes(stripImports(source).replace(/^export /gm, ''), { mode: 'strip' });
const parseToObject = (value) => JSON.parse(JSON.stringify(value ?? {}));
function service(request = async () => null) {
  return new Function('request', 'parseToObject', `${strip(read('services/outbound.ts'))}\nreturn { outboundQuery, getOutboundPage, getExactOutboundOrder, getOutboundMaterials, getOutboundSummary, normalizeOutboundMaterial, normalizeOutboundOrder, outboundLineAmount, summarizeOutbound, outboundDateTimestamp };`)(request, parseToObject);
}
function deferred() { let resolve, reject; const promise = new Promise((yes, no) => { resolve = yes; reject = no; }); return { promise, resolve, reject }; }

test('outbound page preserves unrestricted packing filters and nullable monetary data', async () => {
  let request;
  const api = service(async (value) => { request = value; return { total: 1, list: [{ id: 'a', code: 'CK-1', total_amount: null }] }; });
  const query = api.outboundQuery();
  query.customer_id = 'customer-1'; query.status = '待出库'; query.code = 'CK';
  const result = await api.getOutboundPage(query);
  assert.equal(request.url, '/outbound/page');
  assert.equal(request.data.is_pack, -1); assert.equal(request.data.is_weigh, -1);
  assert.equal(request.data.status, '待出库'); assert.equal(request.data.customer_id, 'customer-1');
  assert.equal(result.list[0].total_amount, null);
  assert.equal(api.normalizeOutboundOrder({ total_amount: 0 }).total_amount, 0);
  assert.deepEqual(await service(async () => ({ total: 0, list: null })).getOutboundPage(query), { total: 0, list: [] });
  await assert.rejects(service(async () => ({ total: 1, list: {} })).getOutboundPage(query), /格式异常/);
});

test('exact order lookup rejects a partial-code match and searches remaining pages', async () => {
  const pages = [];
  const api = service(async ({ data }) => {
    pages.push(data.page);
    return data.page === 1 ? { total: 51, list: [{ id: 'other', code: 'CK-10' }] } : { total: 51, list: [{ id: 'wanted', code: 'CK-1' }] };
  });
  assert.equal((await api.getExactOutboundOrder('CK-1')).id, 'wanted');
  assert.deepEqual(pages, [1, 2]);
  assert.equal(await service(async () => ({ total: 1, list: [{ code: 'CK-10' }] })).getExactOutboundOrder('CK-1'), null);
});

test('material arrays keep historical rows, material IDs and business ordering', async () => {
  const api = service(async ({ url, data }) => {
    assert.equal(url, '/outbound/materials'); assert.equal(data.order_code, 'CK-1');
    return [{ id: 'b', index: 2, name: '旧名称', material_id: 'm2', price: 0, quantity: 5 }, { id: 'a', index: 1, price: 1.25, quantity: 2 }];
  });
  const rows = await api.getOutboundMaterials('CK-1');
  assert.deepEqual(rows.map(row => row.id), ['a', 'b']);
  assert.equal(rows[1].name, '旧名称'); assert.equal(rows[1].material_id, 'm2');
  assert.equal(api.outboundLineAmount(rows[0]), 2.5); assert.equal(api.outboundLineAmount(rows[1]), 0);
  assert.equal(api.outboundLineAmount(api.normalizeOutboundMaterial({ quantity: 2 })), null);
  assert.deepEqual(await service(async () => null).getOutboundMaterials('CK-1'), []);
});

test('report sums material amounts and keeps unlike quantity units separate', async () => {
  let request;
  const api = service(async (value) => { request = value; return [
    { id: 'a', code: 'CK-1', receipt_date: 30, departure_date: 10, quantity: 2, price: 1.5, unit: '件' },
    { id: 'b', code: 'CK-1', receipt_date: 30, departure_date: 10, quantity: 0.5, price: 10, unit: '千克' },
    { id: 'c', code: 'CK-2', receipt_date: 50, departure_date: 20, quantity: 3, price: 0.2, unit: '件' }
  ]; });
  const rows = await api.getOutboundSummary('customer-1', 100, 200);
  assert.equal(request.url, '/outbound/summary');
  assert.deepEqual(request.data, { customer_id: 'customer-1', start_date: 100, end_date: 200 });
  assert.equal(rows[0].receipt_date, 50); assert.equal(rows[0].departure_date, 20);
  const summary = api.summarizeOutbound(rows);
  assert.equal(summary.orderCount, 2); assert.equal(summary.lineCount, 3); assert.equal(summary.amount, 8.6);
  assert.deepEqual(summary.units, [{ unit: '件', quantity: 5 }, { unit: '千克', quantity: 0.5 }]);
  const missing = api.summarizeOutbound([api.normalizeOutboundMaterial({ order_code: 'CK-3', quantity: null, price: 1 })]);
  assert.equal(missing.amount, null); assert.equal(missing.hasMissingQuantity, true); assert.deepEqual(missing.units, []);
});

test('date parsing rejects rollover and supplies seconds at the selected day boundary', () => {
  const api = service();
  assert.equal(api.outboundDateTimestamp('2026-02-30'), 0);
  assert.equal(api.outboundDateTimestamp(''), 0);
  const date = new Date(api.outboundDateTimestamp('2026-09-16') * 1000);
  assert.equal(date.getFullYear(), 2026); assert.equal(date.getMonth(), 8); assert.equal(date.getDate(), 16); assert.equal(date.getHours(), 0);
});

function orderPage(getOutboundPage) {
  const source = read('pages/outbound/order.uvue').match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
  const events = new Map();
  const dependencies = {
    ref: value => ({ value }), computed: fn => ({ get value() { return fn(); } }), watch: () => {},
    errorMessage: (error, fallback) => error.message || fallback, router: { push: () => {}, to: () => {} },
    useStore: () => ({ user: { hasCredential: () => true, canAccess: () => true } }), parseToObject,
    getOutboundPage, outboundQuery: service().outboundQuery,
    companyNameValue: value => value, formatDateValue: value => value, formatMoneyValue: value => value,
    uni: { $on: (name, callback) => events.set(name, callback), $off: name => events.delete(name) },
    onShow: () => {}, onUnload: () => {},
  };
  const factory = new Function('dependencies', `const { ${Object.keys(dependencies).join(', ')} } = dependencies;\n${strip(source)}\nreturn { code, orders, total, error, currentPage, loading, load, scheduleSearch, stopTimer, clearSession, events: null };`);
  return factory(dependencies);
}

test('a delayed previous search cannot replace the new filter result', async () => {
  const oldRequest = deferred(), newRequest = deferred();
  const requests = [];
  const page = orderPage(query => { requests.push(query); return query.code === 'NEW' ? newRequest.promise : oldRequest.promise; });
  const oldLoad = page.load();
  page.code.value = 'NEW'; page.scheduleSearch(); page.stopTimer();
  const newLoad = page.load();
  newRequest.resolve({ total: 1, list: [{ id: 'new', code: 'NEW' }] }); await newLoad;
  oldRequest.resolve({ total: 1, list: [{ id: 'old', code: 'OLD' }] }); await oldLoad;
  assert.deepEqual(page.orders.value.map(row => row.id), ['new']);
  assert.equal(requests[1].page, 1); assert.equal(page.loading.value, false);
});

test('failed page switching retries the same page and logout invalidates in-flight results', async () => {
  const calls = [];
  let failOnce = true;
  const late = deferred();
  const page = orderPage(async query => {
    calls.push(query.page);
    if (query.page === 1) return { total: 60, list: [{ id: 'a' }] };
    if (failOnce) { failOnce = false; throw new Error('network'); }
    if (query.page === 2) return { total: 60, list: [{ id: 'a' }, { id: 'b' }] };
    return late.promise;
  });
  await page.load(1); await page.load(2);
  assert.equal(page.error.value, 'network'); assert.deepEqual(page.orders.value, []);
  await page.load(page.currentPage.value);
  assert.deepEqual(calls, [1, 2, 2]); assert.deepEqual(page.orders.value.map(row => row.id), ['a', 'b']);
  const pending = page.load(3); page.clearSession(); late.resolve({ total: 60, list: [{ id: 'secret' }] }); await pending;
  assert.deepEqual(page.orders.value, []); assert.equal(page.total.value, 0);
});
