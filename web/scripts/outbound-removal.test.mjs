import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import {transformSync} from 'esbuild';
import * as vue from 'vue';
import {compileScript, parse} from 'vue/compiler-sfc';
import axios from 'axios';

function loadTypeScript(path, dependencies = {}) {
  const source = readFileSync(new URL(path, import.meta.url), 'utf8');
  return evaluateTypeScript(source, dependencies);
}

function evaluateTypeScript(source, dependencies = {}) {
  const {code} = transformSync(source, {loader: 'ts', format: 'cjs', target: 'es2020'});
  const module = {exports: {}};
  const evaluate = vm.runInThisContext(`(function(module, exports, require) { ${code}\n})`);
  evaluate(module, module.exports, (name) => {
    assert.ok(name in dependencies, `Unexpected dependency: ${name}`);
    return dependencies[name];
  });
  return module.exports;
}

// Run the real SFC setup and Vue lifecycle without a browser or replacing its business logic.
function mountSfc(path, dependencies, props = {}) {
  const source = readFileSync(new URL(path, import.meta.url), 'utf8');
  const {descriptor} = parse(source, {filename: path});
  const script = compileScript(descriptor, {id: path});
  const resolved = {vue, ...dependencies};
  for (const match of script.content.matchAll(/from\s+["']([^"']+\.vue)["']/g)) {
    resolved[match[1]] = {default: {}};
  }
  const component = evaluateTypeScript(
    script.content.replaceAll('import.meta.env', '({VITE_OSS_DOMAIN: ""})'), resolved,
  ).default;
  component.render = () => null;
  const renderer = vue.createRenderer({
    createElement: () => ({}), createText: () => ({}), createComment: () => ({}),
    insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {},
    parentNode: () => null, nextSibling: () => null,
  });
  const app = renderer.createApp(component, props);
  const instance = app.mount({});
  return {state: instance.$.setupState, unmount: () => app.unmount()};
}

const settle = async () => { await Promise.resolve(); await vue.nextTick(); };

function createRealRequest(messages) {
  const source = readFileSync(new URL('../src/utils/request.ts', import.meta.url), 'utf8');
  return evaluateTypeScript(source.replaceAll('import.meta.env', '({VITE_SERVER: ""})'), {
    axios: {__esModule: true, default: axios},
    'element-plus': {ElMessage: ({message}) => messages.push(message)},
    '@/store/modules/user.ts': {useUserStore: () => ({token: ''})},
  }).default;
}

const {useOutboundRemoval} = loadTypeScript(
  '../src/views/outbound/receipt/components/useOutboundRemoval.ts', {vue},
);

function deferred() {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return {promise, resolve, reject};
}

function result(id = 'order-a', overrides = {}) {
  return {code: 200, msg: '成功', data: {
    id, code: `OUT-${id}`, customer_name: '测试客户', status: '已签收', allowed: true,
    block_reason: '', preview_token: `token-${id}`, transaction_count: 2,
    receivable_reduction: 125, receivable_before: 100, credit_before: 0,
    receivable_after: 0, credit_after: 25,
    inventorys: [
      {inventory_id: 'batch-a', material_id: 'material-a', name: '物料甲', model: 'A', unit: '件', warehouse_name: '一库', release_locked_quantity: 0, restore_available_quantity: 2},
      {inventory_id: 'batch-b', material_id: 'material-b', name: '物料乙', model: 'B', unit: 'kg', warehouse_name: '二库', release_locked_quantity: 0, restore_available_quantity: 3.5},
    ], ...overrides,
  }};
}

test('preview is required, denied orders cannot submit, and mixed units remain separate', async () => {
  let calls = 0;
  let response = result('order-a', {allowed: false, preview_token: '', block_reason: '库存已变动'});
  const state = useOutboundRemoval({preview: async () => response, remove: async () => { calls++; }}, () => {});
  await state.submit();
  await state.select('order-a');
  assert.equal(state.canSubmit.value, false);
  assert.equal(state.preview.value.block_reason, '库存已变动');
  await state.submit();
  assert.equal(calls, 0);
  response = result();
  await state.refresh();
  assert.equal(state.canSubmit.value, true);
  assert.deepEqual(state.preview.value.inventorys.map(row => [row.unit, row.restore_available_quantity]), [['件', 2], ['kg', 3.5]]);
  assert.equal(state.preview.value.credit_after, 25);
});

test('changing orders ignores late preview responses and keeps current loading state', async () => {
  const first = deferred(), second = deferred();
  const state = useOutboundRemoval({preview: id => id === 'order-a' ? first.promise : second.promise, remove: async () => {}}, () => {});
  const a = state.select('order-a');
  const b = state.select('order-b');
  first.resolve(result());
  await a;
  assert.equal(state.loading.value, true);
  assert.equal(state.preview.value, null);
  second.resolve(result('order-b'));
  await b;
  assert.equal(state.preview.value.id, 'order-b');
  assert.equal(state.loading.value, false);
});

test('closing a dialog discards pending preview errors', async () => {
  const pending = deferred();
  const state = useOutboundRemoval({preview: () => pending.promise, remove: async () => {}}, () => {});
  const task = state.select('order-a');
  state.clear();
  pending.reject(new Error('offline'));
  await task;
  assert.equal(state.preview.value, null);
  assert.equal(state.error.value, '');
  assert.equal(state.loading.value, false);
});

test('duplicate refresh and submit clicks send one request and consume the preview token', async () => {
  const previewRequest = deferred(), removeRequest = deferred();
  let previews = 0;
  const deletes = [], successes = [];
  const state = useOutboundRemoval({
    preview: () => { previews++; return previewRequest.promise; },
    remove: (...args) => { deletes.push(args); return removeRequest.promise; },
  }, message => successes.push(message));
  const loading = state.select('order-a');
  await state.refresh();
  assert.equal(previews, 1);
  previewRequest.resolve(result());
  await loading;
  const deleting = state.submit();
  await state.submit();
  await state.refresh();
  assert.deepEqual(deletes, [['order-a', 'token-order-a']]);
  assert.equal(previews, 1);
  assert.equal(state.canSubmit.value, false);
  removeRequest.resolve({code: 200, msg: '已删除或不存在'});
  await deleting;
  await state.submit();
  assert.equal(deletes.length, 1);
  assert.deepEqual(successes, ['已删除或不存在']);
  assert.equal(state.preview.value, null);
});

for (const failure of [
  {name: 'business conflict', response: {code: 409, msg: '变更'}},
  {name: 'HTTP conflict', error: {response: {status: 409}}},
  {name: 'business failure', response: {code: 400, msg: '库存不足'}},
  {name: 'unknown network outcome', error: new Error('timeout')},
]) {
  test(`${failure.name} clears the token and requires a fresh preview`, async () => {
    let previewCalls = 0, deleteCalls = 0, successes = 0;
    const state = useOutboundRemoval({
      preview: async () => { previewCalls++; return result('order-a', {preview_token: `token-${previewCalls}`}); },
      remove: async () => { deleteCalls++; if (failure.error) throw failure.error; return failure.response; },
    }, () => successes++);
    await state.select('order-a');
    await state.submit();
    assert.equal(state.preview.value, null);
    assert.equal(state.canSubmit.value, false);
    assert.match(state.error.value, /重新预览/);
    await state.submit();
    assert.equal(deleteCalls, 1);
    assert.equal(previewCalls, 1);
    assert.equal(successes, 0);
    await state.refresh();
    assert.equal(state.preview.value.preview_token, 'token-2');
    assert.equal(state.error.value, '');
    assert.equal(state.canSubmit.value, true);
  });
}

test('completion from an old order does not close a newly opened dialog', async () => {
  const pending = deferred();
  let successes = 0;
  const state = useOutboundRemoval({preview: async id => result(id), remove: () => pending.promise}, () => successes++);
  await state.select('order-a');
  const task = state.submit();
  await state.select('order-b');
  pending.resolve({code: 200, msg: '成功'});
  await task;
  assert.equal(successes, 0);
  assert.equal(state.preview.value.id, 'order-b');
  assert.equal(state.canSubmit.value, true);
});

test('incomplete or mismatched preview responses cannot authorize deletion', async () => {
  for (const data of [null, result('other').data, result('order-a', {preview_token: ''}).data, result('order-a', {inventorys: null}).data]) {
    const state = useOutboundRemoval({preview: async () => ({code: 200, data}), remove: async () => assert.fail('must not delete')}, () => {});
    await state.select('order-a');
    await state.submit();
    assert.equal(state.canSubmit.value, false);
    assert.match(state.error.value, /不完整/);
  }
});

test('API methods preserve DELETE compatibility and send the server preview token', async () => {
  const requests = [];
  const api = loadTypeScript('../src/api/outbound/index.ts', {
    '@/utils/request.ts': {__esModule: true, default: {
      get: (...args) => { requests.push(['GET', ...args]); return Promise.resolve(result()); },
      delete: (...args) => { requests.push(['DELETE', ...args]); return Promise.resolve({code: 200}); },
    }},
  });
  await api.reqOutboundRemovePreview('order-a');
  await api.reqRemoveOutboundOrder('order-a', 'signed-preview');
  assert.deepEqual(requests, [
    ['GET', '/outbound/remove-preview', {params: {id: 'order-a'}}],
    ['DELETE', '/outbound', {params: {id: 'order-a', preview_token: 'signed-preview'}}],
  ]);
});

test('a paid signed order can delete, and success teardown cannot submit its token again', async () => {
  const pending = deferred();
  const events = [], deletes = [];
  let dialog;
  dialog = mountSfc('../src/views/outbound/receipt/components/Remove.vue', {
    'element-plus': {ElMessage: {success: message => events.push(['message', message])}},
    '@/api/outbound': {
      reqOutboundRemovePreview: async () => result('order-a', {
        receivable_before: 0, credit_before: 10, receivable_after: 0, credit_after: 135,
      }),
      reqRemoveOutboundOrder: (...args) => { deletes.push(args); return pending.promise; },
    },
    './useOutboundRemoval': {useOutboundRemoval},
  }, {
    id: 'order-a', code: 'OUT-order-a',
    onBusy: value => events.push(['busy', value]),
    onSuccess: () => { events.push(['success']); dialog.unmount(); },
  });
  await settle();
  assert.equal(dialog.state.canSubmit, true);
  assert.equal(dialog.state.preview.status, '已签收');
  assert.equal(dialog.state.preview.receivable_before, 0);
  assert.equal(dialog.state.preview.credit_after, 135);
  const removing = dialog.state.submit();
  await dialog.state.submit();
  assert.equal(dialog.state.submitting, true);
  pending.resolve({code: 200, msg: '成功'});
  await removing;
  await dialog.state.submit();
  assert.equal(deletes.length, 1);
  assert.deepEqual(events, [['busy', true], ['message', '成功'], ['success'], ['busy', false]]);
  assert.equal(dialog.state.canSubmit, false);
  assert.equal(dialog.state.error, '');
});

test('a preview after a committed deletion with a lost response reconciles the missing order without another DELETE', async () => {
  let previewCalls = 0;
  const deletes = [], successes = [], missing = [];
  const state = useOutboundRemoval({
    preview: async () => ++previewCalls === 1 ? result('order-a', {preview_token: 'token-1'})
      : {code: 404, msg: '出库单已删除或不存在'},
    remove: async (...args) => {
      deletes.push(args);
      throw new Error('connection lost after commit');
    },
  }, message => successes.push(message), () => missing.push('order-a'));
  await state.select('order-a');
  await state.submit();
  await state.submit();
  assert.equal(deletes.length, 1);
  assert.equal(state.canSubmit.value, false);
  await state.refresh();
  await state.submit();
  await state.refresh();
  assert.deepEqual(deletes, [['order-a', 'token-1']]);
  assert.deepEqual(successes, []);
  assert.deepEqual(missing, ['order-a']);
  assert.equal(previewCalls, 2);
  assert.equal(state.loading.value, false);
  assert.equal(state.error.value, '');
});

test('the real removal component reports an initially absent order separately from successful deletion', async () => {
  const events = [];
  const dialog = mountSfc('../src/views/outbound/receipt/components/Remove.vue', {
    'element-plus': {ElMessage: {
      success: () => assert.fail('absence cannot claim this deletion succeeded'),
      info: message => events.push(message),
    }},
    '@/api/outbound': {
      reqOutboundRemovePreview: async () => ({code: 404, msg: '出库单已删除或不存在'}),
      reqRemoveOutboundOrder: () => assert.fail('absence must never issue DELETE'),
    },
    './useOutboundRemoval': {useOutboundRemoval},
  }, {id: 'order-a', code: 'OUT-order-a', onMissing: () => events.push('missing')});
  await settle();
  assert.deepEqual(events, ['该出库单已不存在，列表已更新', 'missing']);
  assert.equal(dialog.state.canSubmit, false);
  dialog.unmount();
});

test('only a current business 404 may reconcile an absent order', async () => {
  const old = deferred();
  const missing = [];
  let response = result('order-b');
  const state = useOutboundRemoval({
    preview: id => id === 'order-a' ? old.promise : Promise.resolve(response),
    remove: () => assert.fail('preview does not delete'),
  }, () => assert.fail('preview is not deletion success'), () => missing.push(true));
  const first = state.select('order-a');
  await state.select('order-b');
  old.resolve({code: 404, msg: '出库单已删除或不存在'});
  await first;
  assert.equal(state.preview.value.id, 'order-b');
  assert.deepEqual(missing, []);
  for (const code of [400, 500]) {
    response = {code, msg: '预览错误'};
    await state.refresh();
    assert.deepEqual(missing, []);
    assert.equal(state.error.value, '预览错误');
    assert.equal(state.canSubmit.value, false);
  }
});

test('an HTTP 404 route failure cannot prove the order is absent', async () => {
  const cause = new Error('route not found');
  cause.response = {status: 404};
  const state = useOutboundRemoval({
    preview: async () => { throw cause; },
    remove: () => assert.fail('preview does not delete'),
  }, () => assert.fail('preview is not deletion success'), () => assert.fail('HTTP error is not business absence'));
  await state.select('order-a');
  assert.match(state.error.value, /预览失败/);
  assert.equal(state.canSubmit.value, false);
});

for (const failure of [
  {name: 'network timeout without a response', status: undefined, expectedToast: '无网络', expectedInline: /结果未确认/},
  {name: 'HTTP 409 with server explanation', status: 409, msg: '客户回款已变化，请重新预览', expectedToast: '客户回款已变化，请重新预览', expectedInline: /预览已失效/},
  {name: 'HTTP 409 without an explanation', status: 409, expectedToast: '数据已变化，请刷新后重试', expectedInline: /预览已失效/},
  {name: 'HTTP 503 after commit', status: 503, expectedToast: '服务暂时不可用，请核对操作结果', expectedInline: /结果未确认/},
]) {
  test(`real response interceptor preserves ${failure.name} and removal requires a new preview`, async () => {
    const messages = [], requests = [];
    const original = new Error('request failed');
    if (failure.status) original.response = {status: failure.status, data: {msg: failure.msg}};
    const request = createRealRequest(messages);
    request.defaults.adapter = async config => {
      requests.push(config.method);
      if (config.method === 'get') return {config, status: 200, statusText: 'OK', headers: {}, data: result()};
      throw original;
    };
    const api = loadTypeScript('../src/api/outbound/index.ts', {'@/utils/request.ts': {__esModule: true, default: request}});
    let rejected;
    const state = useOutboundRemoval({
      preview: api.reqOutboundRemovePreview,
      remove: async (...args) => {
        try { return await api.reqRemoveOutboundOrder(...args); }
        catch (cause) { rejected = cause; throw cause; }
      },
    }, () => assert.fail('unknown or conflicting deletion cannot report success'));
    await state.select('order-a');
    await state.submit();
    await state.submit();
    assert.equal(rejected, original);
    assert.deepEqual(messages, [failure.expectedToast]);
    assert.match(state.error.value, failure.expectedInline);
    assert.equal(state.preview.value, null);
    assert.equal(state.canSubmit.value, false);
    assert.deepEqual(requests, ['get', 'delete']);
  });
}

test('real response interceptor leaves HTTP 200 business conflicts to the removal dialog', async () => {
  const messages = [];
  const request = createRealRequest(messages);
  const businessConflict = {code: 409, msg: '库存已变化'};
  request.defaults.adapter = async config => ({config, status: 200, statusText: 'OK', headers: {}, data: businessConflict});
  const response = await request.delete('/outbound');
  assert.equal(response, businessConflict);
  assert.deepEqual(messages, []);
});

for (const [page, apiName] of [['index.vue', 'reqOutboundOrders'], ['index2.vue', 'reqOutboundOrders2']]) {
  function mountPage(request) {
    const errors = [];
    const pageInstance = mountSfc(`../src/views/outbound/receipt/${page}`, {
      '@/enums/outbound.ts': {OutboundOrderTypes: []},
      '@/utils/enum.ts': {Sizes: {}, Types: {}},
      '@/api/outbound': {[apiName]: request},
      'element-plus': {ElMessage: {error: message => errors.push(message)}},
      'number-precision': {default: {}},
      '@/utils/time.ts': {DateFormat: () => ''},
    });
    return {...pageInstance, errors};
  }
  const orderA = {id: 'order-a', code: 'OUT-A', status: '已签收'};
  const orderB = {id: 'order-b', code: 'OUT-B', status: '已签收'};
  const list = rows => ({code: 200, msg: '成功', data: {list: rows, total: rows.length}});

  test(`${page}: a list requested before deletion cannot restore the deleted row`, async () => {
    const old = deferred(), current = deferred();
    const requests = [];
    const pageInstance = mountPage(params => {
      requests.push(params);
      return requests.length === 1 ? Promise.resolve(list([orderA, orderB]))
        : requests.length === 2 ? old.promise : current.promise;
    });
    try {
      await settle();
      const {state} = pageInstance;
      const oldTask = state.getReceipts();
      state.form.code = 'changed after request';
      assert.equal(requests[1].code, '');
      state.remove(orderA);
      assert.equal(state.visible, true);
      state.handleRemoveSuccess();
      assert.equal(state.visible, false);
      assert.deepEqual(state.receipts.map(row => row.id), ['order-b']);
      assert.equal(state.total, 1);
      assert.equal(state.loading, true);
      current.resolve(list([orderB]));
      await settle();
      old.resolve(list([orderA, orderB]));
      await oldTask;
      assert.deepEqual(state.receipts.map(row => row.id), ['order-b']);
      assert.equal(state.total, 1);
      assert.equal(state.loading, false);
      assert.deepEqual(pageInstance.errors, []);
    } finally {
      pageInstance.unmount();
    }
  });

  test(`${page}: failed post-delete refresh ends loading and keeps the confirmed deletion`, async () => {
    const old = deferred(), current = deferred();
    let calls = 0;
    const pageInstance = mountPage(() => ++calls === 1 ? Promise.resolve(list([orderA, orderB]))
      : calls === 2 ? old.promise : current.promise);
    try {
      await settle();
      const {state} = pageInstance;
      const oldTask = state.getReceipts();
      state.remove(orderA);
      state.handleRemoveSuccess();
      old.reject(new Error('stale request failed'));
      await oldTask;
      assert.equal(state.loading, true);
      assert.deepEqual(pageInstance.errors, []);
      current.reject(new Error('offline'));
      await settle();
      assert.equal(state.loading, false);
      assert.deepEqual(state.receipts.map(row => row.id), ['order-b']);
      assert.equal(state.total, 1);
      assert.deepEqual(pageInstance.errors, ['出库单列表刷新失败，请重新查询']);
    } finally {
      pageInstance.unmount();
    }
  });

  test(`${page}: a business failure after deletion preserves the remaining list and total`, async () => {
    const refresh = deferred();
    let calls = 0;
    const pageInstance = mountPage(() => ++calls === 1
      ? Promise.resolve(list([orderA, orderB])) : refresh.promise);
    try {
      await settle();
      const {state} = pageInstance;
      state.remove(orderA);
      state.handleRemoveSuccess();
      assert.equal(state.loading, true);
      assert.equal(state.visible, false);
      refresh.resolve({code: 500, msg: '服务器内部错误'});
      await settle();
      assert.equal(state.loading, false);
      assert.deepEqual(state.receipts.map(row => row.id), ['order-b']);
      assert.equal(state.total, 1);
      assert.deepEqual(pageInstance.errors, ['服务器内部错误']);
    } finally {
      pageInstance.unmount();
    }
  });

  test(`${page}: an initial business failure ends loading and permits a successful retry`, async () => {
    let calls = 0;
    const pageInstance = mountPage(async () => ++calls === 1
      ? {code: 500, msg: '服务器内部错误'} : list([orderA]));
    try {
      await settle();
      const {state} = pageInstance;
      assert.equal(state.loading, false);
      assert.deepEqual(state.receipts, []);
      assert.equal(state.total, 0);
      assert.deepEqual(pageInstance.errors, ['服务器内部错误']);
      await state.getReceipts();
      assert.equal(state.loading, false);
      assert.deepEqual(state.receipts.map(row => row.id), ['order-a']);
      assert.equal(state.total, 1);
      assert.deepEqual(pageInstance.errors, ['服务器内部错误']);
    } finally {
      pageInstance.unmount();
    }
  });

  test(`${page}: an unmounted page ignores pending list results`, async () => {
    const pending = deferred();
    const pageInstance = mountPage(() => pending.promise);
    pageInstance.unmount();
    pending.resolve(list([orderA]));
    await settle();
    assert.deepEqual(pageInstance.state.receipts, []);
    assert.deepEqual(pageInstance.errors, []);
  });
}
