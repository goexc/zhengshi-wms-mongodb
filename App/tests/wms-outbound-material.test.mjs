import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
import { stripTypeScriptTypes } from 'node:module';
import { stripImports } from './helpers/source.mjs';

function executable(path, dependencies) {
  let source = fs.readFileSync(new URL('../' + path, import.meta.url), 'utf8');
  if (path.endsWith('.uvue')) source = source.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
  const context = vm.createContext(dependencies);
  vm.runInContext(stripTypeScriptTypes(stripImports(source).replace(/^export /gm, '')), context);
  return expression => vm.runInContext(expression, context);
}
const plain = value => JSON.parse(JSON.stringify(value));
const tick = () => new Promise(setImmediate);
function picker(selected = '') {
  const pending = [], events = [], timers = new Map();
  const props = { modelValue: selected, disabled: false };
  let disposed;
  let nextTimer = 0;
  const run = executable('components/wms/MaterialModelSelect.uvue', {
    ref: value => ({ value }), computed: fn => ({ get value() { return fn(); } }), watch: () => {}, defineOptions: () => {}, defineProps: () => props,
    defineEmits: () => (name, value) => { events.push({ name, value }); if (name === 'update:modelValue') props.modelValue = value; },
    findMaterialModels: query => new Promise((resolve, reject) => pending.push({ query, resolve, reject })),
    setTimeout: callback => { const id = ++nextTimer; timers.set(id, callback); return id; },
    clearTimeout: id => timers.delete(id), onUnmounted: callback => { disposed = callback; },
    // Match the current native base: hideKeyboard is not available.
    uni: { $on: () => {}, $off: () => {} }, errorMessage: error => error.message
  });
  return { run, pending, events, props, dispose: () => disposed(), flush: () => { const callbacks = [...timers.values()]; timers.clear(); callbacks.forEach(callback => callback()); } };
}

test('material model discovery uses the Web contract, trims and deduplicates up to ten choices', async () => {
  const requests = [];
  const run = executable('services/material.ts', {
    parse: value => value, parseToObject: value => value,
    request: async request => { requests.push(plain(request)); return { total: 30, list: [{ model: ' A-01 ' }, { model: 'A-01' }, { model: null }, ...Array.from({ length: 12 }, (_, i) => ({ model: `B-${i}` }))] }; }
  });
  const result = await run("findMaterialModels('  A  ')");
  assert.equal(result.length, 10);
  assert.deepEqual(plain(result.slice(0, 2)), ['A-01', 'B-0']);
  assert.deepEqual(requests[0], { url: '/material', method: 'GET', data: { page: 1, size: 10, model: 'A' } });
  assert.deepEqual(plain(await run("findMaterialModels('   ')")), []);
  assert.equal(requests.length, 1);
});

test('typing searches candidates after debounce, and only a selected complete model is emitted', async () => {
  const h = picker();
  h.run('open()');
  assert.equal(h.pending.length, 0, 'opening an empty selector does not search or filter orders');
  h.run("editKeyword('R')"); h.run("editKeyword('RGV')");
  assert.equal(h.pending.length, 0); assert.equal(h.events.length, 0);
  h.flush(); assert.equal(h.pending.length, 1); assert.equal(h.pending[0].query, 'RGV');
  h.pending[0].resolve(['RGV-01', 'RGV-02']); await tick();
  h.run("choose('RGV')"); assert.equal(h.events.length, 0, 'raw input cannot be committed');
  h.run("choose('RGV-02')");
  assert.deepEqual(h.events, [{ name: 'update:modelValue', value: 'RGV-02' }, { name: 'change', value: 'RGV-02' }]);
  assert.equal(h.run('keyword.value'), 'RGV-02'); assert.equal(h.run('visible.value'), false);
  assert.equal(h.run('options.value.length'), 0);
});

test('popup search preserves the committed filter, rejects stale replies, and cancel discards the draft', async () => {
  const h = picker('OLD-01');
  h.run('open()');
  assert.equal(h.pending[0].query, 'OLD-01');
  h.run("editKeyword('NEW')"); h.flush();
  h.pending[1].resolve(['NEW-01']); await tick();
  h.pending[0].resolve(['OLD-01']); await tick();
  assert.equal(h.run('keyword.value'), 'NEW');
  assert.deepEqual(plain(h.run('options.value')), ['NEW-01']);
  assert.equal(h.props.modelValue, 'OLD-01'); assert.equal(h.events.length, 0);
  h.run("editKeyword('')"); h.flush();
  assert.equal(h.props.modelValue, 'OLD-01'); assert.equal(h.run('visible.value'), true);
  assert.equal(h.run('loading.value'), false); assert.equal(h.pending.length, 2);
  h.run('cancel()');
  assert.equal(h.events.length, 0); assert.equal(h.run('visible.value'), false);
  h.run('open()'); assert.equal(h.run('keyword.value'), 'OLD-01');
  h.run("editKeyword('NEW')"); h.flush();
  h.pending[3].resolve(['NEW-01']); await tick();
  h.run("choose('NEW-01')");
  assert.equal(h.props.modelValue, 'NEW-01');
  assert.deepEqual(h.events, [{ name: 'update:modelValue', value: 'NEW-01' }, { name: 'change', value: 'NEW-01' }]);
});

test('all materials explicitly clears the committed filter and invalidates pending candidates', async () => {
  const h = picker('OLD-01');
  h.run('open()'); h.run('chooseAll()');
  assert.equal(h.props.modelValue, ''); assert.equal(h.run('visible.value'), false);
  assert.deepEqual(h.events, [{ name: 'update:modelValue', value: '' }, { name: 'change', value: '' }]);
  h.pending[0].resolve(['OLD-01']); await tick();
  assert.equal(h.run('options.value.length'), 0);
  h.run('open()'); assert.equal(h.run('keyword.value'), '');
  h.run('chooseAll()'); assert.equal(h.events.length, 2, 'same filter does not trigger a redundant order query');
  h.run("editKeyword('closed')"); h.flush(); assert.equal(h.pending.length, 1);
});

test('candidate failures are retryable and closing, resetting or unmounting discards late replies', async () => {
  const h = picker();
  h.run('open()');
  h.run("editKeyword('M')"); h.flush(); h.pending[0].reject(new Error('offline')); await tick();
  assert.equal(h.run('error.value'), 'offline'); assert.equal(h.run('loading.value'), false);
  const retry = h.run('searchNow()'); assert.equal(h.pending[1].query, 'M');
  h.run('cancel()'); h.pending[1].resolve(['M-1']); await retry;
  assert.equal(h.run('options.value.length'), 0);
  h.run("open(); editKeyword('session')"); h.flush();
  h.run('clearSession()'); h.pending[2].resolve(['secret']); await tick();
  assert.equal(h.run('keyword.value'), ''); assert.equal(h.run('options.value.length'), 0);
  h.run("open(); editKeyword('late')"); h.flush(); h.dispose(); h.pending[3].resolve(['late-model']); await tick();
  assert.equal(h.run('options.value.length'), 0);
});

test('order requests combine the selected model with customer and code through paging, refresh and reset', async () => {
  const requests = [], navigations = [];
  const service = executable('services/outbound.ts', {});
  const run = executable('pages/outbound/order.uvue', {
    ref: value => ({ value }), watch: () => {}, useStore: () => ({ user: { hasCredential: () => true, canAccess: () => true } }),
    outboundQuery: page => service(`outboundQuery(${page})`),
    getOutboundPage: async query => { requests.push(plain(query)); return { total: 40, list: [{ id: 'matching-order' }] }; },
    onShow: () => {}, onUnload: () => {}, setTimeout, clearTimeout,
    uni: { $on: () => {}, $off: () => {} }, router: { push: options => navigations.push(plain(options)) },
    parseToObject: plain, errorMessage: error => error.message
  });
  run("code.value = 'OUT'; customerID.value = 'customer'; materialModel.value = 'RGV-01'; searchNow()"); await tick();
  run("openOrder({ code: 'OUT-123' })");
  assert.deepEqual(navigations[0], { path: '/pages/outbound/detail', query: { code: 'OUT-123', model: 'RGV-01' }, params: { outboundOrder: { code: 'OUT-123' } } });
  await run('load(2)'); run('refresh()'); await tick();
  assert.deepEqual(requests.map(query => query.page), [1, 2, 2]);
  for (const query of requests) {
    assert.equal(query.model, 'RGV-01'); assert.equal(query.customer_id, 'customer'); assert.equal(query.code, 'OUT');
    assert.equal(query.status, ''); assert.equal(query.type, ''); assert.equal(query.is_pack, -1); assert.equal(query.is_weigh, -1);
  }
  run('reset()'); await tick();
  assert.equal(requests.at(-1).model, ''); assert.equal(requests.at(-1).customer_id, ''); assert.equal(requests.at(-1).code, '');
  assert.equal(run('currentPage.value'), 1); assert.equal(run('materialPickerKey.value'), 1);
  run('reset()'); await tick(); assert.equal(run('materialPickerKey.value'), 2, 'reset must also recreate an unselected draft input');
  run("openOrder({ code: 'OUT-456' })");
  assert.deepEqual(navigations.at(-1).query, { code: 'OUT-456' }, 'unfiltered navigation cannot carry a previous highlight');
});

function detailPage(rows) {
  const requests = [];
  let mount;
  const run = executable('pages/outbound/detail.uvue', {
    ref: value => ({ value }), computed: fn => ({ get value() { return fn(); } }),
    useStore: () => ({ user: { hasCredential: () => true, canAccess: () => true } }),
    getOutboundMaterials: async code => { requests.push(code); return rows; },
    getExactOutboundOrder: async code => ({ code }), normalizeOutboundOrder: value => value,
    router: { params: () => ({}) }, storage: { remove: () => {} },
    onLoad: callback => { mount = callback; }, onUnload: () => {},
    uni: { $on: () => {}, $off: () => {} }, errorMessage: error => error.message
  });
  return { run, requests, mount: query => mount(query) };
}

test('detail highlights all exact historical model matches without filtering or reordering rows', async () => {
  const rows = ['OTHER', 'RGV-01', 'RGV-010', 'rgv-01', 'RGV-01', '', null].map((model, i) => ({ id: `line${i}`, index: i, model }));
  const h = detailPage(rows);
  h.mount({ code: 'OUT-123', model: 'RGV-01' }); await tick();
  assert.deepEqual(h.requests, ['OUT-123'], 'highlight is local presentation, detail API still loads the entire order');
  assert.equal(h.run('highlightModel.value'), 'RGV-01');
  assert.equal(h.run('matchedCount.value'), 2);
  assert.deepEqual(plain(h.run('materials.value.filter(matchesMaterial).map(item => item.id)')), ['line1', 'line4']);
  assert.deepEqual(plain(h.run('materials.value')), rows);
  await h.run('load()');
  assert.equal(h.run('matchedCount.value'), 2, 'reloading the order preserves the routed selection');
  h.run('clearSession()');
  assert.equal(h.run('matchedCount.value'), 0);
});

test('detail without a valid model has no highlight and unmatched routes retain complete rows', async () => {
  for (const model of [undefined, '', '  ', null, 123, 'MISSING']) {
    const h = detailPage([{ id: 'line1', model: 'RGV-01' }]);
    h.mount({ code: 'OUT-123', model }); await tick();
    assert.equal(h.run('matchedCount.value'), 0);
    assert.equal(h.run('materials.value.length'), 1);
    assert.equal(h.run('highlightModel.value'), model === 'MISSING' ? 'MISSING' : '');
  }
});
