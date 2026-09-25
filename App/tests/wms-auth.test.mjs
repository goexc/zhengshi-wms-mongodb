import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { stripTypeScriptTypes } from 'node:module';
import { test } from 'node:test';

const read = relative => readFileSync(new URL(`../${relative}`, import.meta.url), 'utf8');
const compile = source => stripTypeScriptTypes(source.replace(/^import .*;\r?\n/gm, '').replace(/^export /gm, ''), { mode: 'strip' });
const plain = value => JSON.parse(JSON.stringify(value ?? {}));
function harness() {
  const now = 1_800_000_000_000;
  const data = new Map(), expires = new Map(), writes = [], requests = [], events = [];
  let loginCount = 0, api;
  const storage = {
    get: key => data.get(key) ?? null,
    set: (key, value, seconds) => { data.set(key, value); expires.set(key, seconds > 0 ? now + seconds * 1000 : Infinity); writes.push({ key, value, seconds }); },
    remove: key => { data.delete(key); expires.delete(key); },
    isExpired: key => !expires.has(key) || expires.get(key) <= now,
  };
  const uni = { request: options => requests.push(options), $emit: name => events.push(name) };
  const ref = value => ({ value }), computed = fn => ({ get value() { return fn(); } });
  class FixedDate extends Date { static now() { return now; } }
  const User = new Function('ref', 'computed', 'parseToObject', 'storage', 'keys', 'router', 'request', 'errorMessage', 'uni', 'Date', `${compile(read('.cool/store/user.ts'))}\nreturn User;`)(ref, computed, plain, storage, Object.keys, { login: () => loginCount++ }, options => api.request(options), (error, fallback) => api.errorMessage(error, fallback), uni, FixedDate);
  const user = new User();
  api = new Function('config', 'user', 'uni', `${compile(read('.cool/service/index.ts'))}\nreturn { request, requestEnvelope, errorMessage };`)({ baseUrl: 'https://wms.example.test' }, user, uni);
  const login = (token = 'token-A') => user.setToken({ token, exp: now / 1000 + 3600 });
  const reply = (body, status = 200, index = 0) => requests[index].success({ data: body, statusCode: status });
  return { api, user, storage, writes, requests, events, login, reply, now, get loginCount() { return loginCount; } };
}

test('WMS request unwraps data and sends the raw token without a Bearer prefix', async () => {
  const h = harness(); h.login();
  const promise = h.api.request({ url: '/outbound/page', method: 'GET', data: { page: 1 } });
  assert.equal(h.requests[0].url, 'https://wms.example.test/outbound/page');
  assert.equal(h.requests[0].header.Authorization, 'token-A');
  assert.deepEqual(h.requests[0].data, { page: 1 });
  h.reply({ code: 200, msg: '成功', data: { total: 0, list: [] } });
  assert.deepEqual(await promise, { total: 0, list: [] });
});

test('HTTP 200 alone cannot make malformed envelopes successful', async () => {
  for (const body of [null, '', '<html>gateway</html>', [], {}, { data: [] }, { code: '200' }, { code: '200oops' }, { code: NaN }]) {
    const h = harness(); h.login();
    const promise = h.api.request({ url: '/material' });
    h.reply(body);
    await assert.rejects(promise, error => error.code === 502 && error.message.includes('格式异常'));
    assert.equal(h.user.token, 'token-A');
  }
});

test('business 401 logs out once and late parallel replies cannot restore data', async () => {
  const h = harness(); h.login();
  const first = h.api.request({ url: '/customer' }), second = h.api.request({ url: '/supplier' });
  h.reply({ code: 401, msg: '登录已过期' }, 200, 0);
  await assert.rejects(first, error => error.code === 401 && error.message === '登录已过期');
  h.reply({ code: 200, data: { secret: true } }, 200, 1);
  await assert.rejects(second, error => error.code === 401 && error.message.includes('会话已变更'));
  assert.equal(h.user.token, null); assert.equal(h.loginCount, 1);
});

test('HTTP 401 is honored without a JSON body; HTTP/business 403 preserve the session', async () => {
  const unauth = harness(); unauth.login();
  const missing = unauth.api.request({ url: '/material' }); unauth.reply('', 401);
  await assert.rejects(missing, error => error.code === 401); assert.equal(unauth.user.token, null);
  for (const [status, body] of [[403, { code: 200, msg: '禁止访问' }], [200, { code: 403 }], [403, '<html>Forbidden</html>']]) {
    const h = harness(); h.login(); const promise = h.api.request({ url: '/material' }); h.reply(body, status);
    await assert.rejects(promise, error => error.code === 403 && error.message !== '');
    assert.equal(h.user.token, 'token-A'); assert.equal(h.loginCount, 0);
  }
});

test('only account/menu maps WMS 204 to empty permissions and envelope calls remain enveloped', async () => {
  const h = harness(); h.login();
  const menu = h.api.request({ url: '/account/menu' }); h.reply({ code: 204, msg: '用户没有绑定任何角色' });
  assert.deepEqual(await menu, { menus: [], buttons: [] });
  const wrapped = h.api.requestEnvelope({ url: '/account/menu' }); h.reply({ code: 204, msg: '无菜单' }, 200, 1);
  assert.deepEqual(await wrapped, { code: 200, msg: '无菜单', data: { menus: [], buttons: [] } });
  const other = h.api.request({ url: '/customer' }); h.reply({ code: 204, msg: '未找到客户' }, 200, 2);
  await assert.rejects(other, error => error.code === 204 && error.message === '未找到客户');
});

test('message fallbacks skip empty values and prefer WMS msg for API failures', async () => {
  const h = harness(); h.login();
  assert.equal(h.api.errorMessage({ message: '', msg: 'WMS 原因' }, 'fallback'), 'WMS 原因');
  assert.equal(h.api.errorMessage(new Error(''), 'fallback'), 'fallback');
  assert.equal(h.api.errorMessage(500, 'fallback'), 'fallback');
  assert.equal(h.api.errorMessage(['wrong'], 'fallback'), 'fallback');
  const promise = h.api.request({ url: '/material' }); h.reply({ code: 400, msg: '业务错误', message: 'secondary' });
  await assert.rejects(promise, error => error.message === '业务错误');
  const fallback = h.api.request({ url: '/material' }); h.reply({ code: 400, msg: '', message: '兼容错误' }, 200, 1);
  await assert.rejects(fallback, error => error.message === '兼容错误');
});

test('login failures do not redirect or erase a newer session', async () => {
  const h = harness();
  const invalid = h.api.request({ url: '/auth/login', method: 'POST' });
  assert.equal(h.requests[0].header.Authorization, null);
  h.reply({ code: 401, msg: '密码错误' });
  await assert.rejects(invalid, error => error.message === '密码错误'); assert.equal(h.loginCount, 0);
  const late = h.api.request({ url: '/auth/login', method: 'POST' }); h.login('new-session');
  h.reply({ code: 401, msg: '旧请求失败' }, 200, 1);
  await assert.rejects(late, error => error.message.includes('会话已变更'));
  assert.equal(h.user.token, 'new-session'); assert.equal(h.loginCount, 0);
});

test('expiry is interpreted as Unix seconds and invalid token responses cannot create a session', () => {
  const h = harness(); h.login();
  assert.equal(h.writes.find(row => row.key === 'wms.auth.token').seconds, 3600);
  for (const token of [{}, { token: null, exp: h.now / 1000 + 3600 }, { token: 'x' }, { token: 'x', exp: NaN }, { token: 'x', exp: Infinity }, { token: 'x', exp: h.now / 1000 }, { token: '  ', exp: h.now / 1000 + 3600 }]) {
    assert.throws(() => h.user.setToken(token), /登录凭证无效/);
    assert.equal(h.user.token, 'token-A');
  }
});

test('old profile completions cannot overwrite identity or loading of a newer session', async () => {
  const h = harness(); h.login('A');
  const old = h.user.get();
  h.user.clear(); h.login('B');
  const current = h.user.get();
  h.reply({ code: 200, data: { name: 'Old user' } }, 200, 0);
  await old;
  assert.equal(h.user.info.value, null); assert.equal(h.user.loading.value, true); assert.equal(h.requests.length, 2);
  h.reply({ code: 200, data: { name: 'New user', mobile: '13800000000' } }, 200, 1);
  await Promise.resolve(); await Promise.resolve();
  assert.equal(h.requests[2].url, 'https://wms.example.test/account/menu');
  h.reply({ code: 200, data: { menus: [], buttons: [] } }, 200, 2);
  await current;
  assert.equal(h.user.info.value.name, 'New user'); assert.equal(h.user.loading.value, false);
  assert.equal(h.user.permissionsReady.value, true);
});

test('menu normalization handles nullable lists, nested grants, malformed structures and stale refreshes', async () => {
  const h = harness(); h.login();
  const first = h.user.loadPermissions();
  h.reply({ code: 200, data: { menus: [{ path: '/outbound', meta: { perms: '' }, children: [{ path: '/outbound/order', meta: { perms: 'outbound:order:list' } }] }], buttons: [{ perms: 'material:material:edit' }] } });
  await first;
  assert.equal(h.user.canAccess('outbound'), true); assert.equal(h.user.hasPermission('material:material:edit'), true);
  assert.ok(h.events.includes('wms.permissions.ready'));
  const stale = h.user.loadPermissions(), fresh = h.user.loadPermissions();
  h.reply({ code: 200, data: { menus: null, buttons: null } }, 200, 2); await fresh;
  h.reply({ code: 200, data: { menus: [{ path: '/outbound' }], buttons: [] } }, 200, 1); await stale;
  assert.equal(h.user.canAccess('outbound'), false); assert.deepEqual(h.user.menuPaths.value, []);
  for (const value of [{}, { menus: {}, buttons: [] }, { menus: [null], buttons: [] }, { menus: [], buttons: 'bad' }]) {
    const pending = h.user.loadPermissions(); h.reply({ code: 200, data: value }, 200, h.requests.length - 1); await pending;
    assert.equal(h.user.permissionsReady.value, false); assert.equal(h.user.canAccess('outbound'), false); assert.notEqual(h.user.permissionError.value, '');
  }
});

test('logout clears session, permission state and pending navigation without touching preferences', () => {
  const h = harness(); h.login(); h.user.set({ name: 'User' });
  h.storage.set('router-params', { outboundOrder: { code: 'private' } }, 0); h.storage.set('wms.largeText', true, 0);
  h.user.permissions.value = ['outbound:order:list']; h.user.permissionsReady.value = true;
  h.user.logout();
  assert.equal(h.storage.get('wms.auth.token'), null); assert.equal(h.storage.get('wms.auth.user'), null); assert.equal(h.storage.get('router-params'), null);
  assert.equal(h.storage.get('wms.largeText'), true); assert.equal(h.user.info.value, null); assert.equal(h.user.canAccess('outbound'), false);
});

test('malformed profile data preserves the last valid identity while permission refresh remains possible', async () => {
  const h = harness(); h.login(); h.user.set({ name: 'Cached user' });
  const pending = h.user.get(); h.reply({ code: 200, data: {} });
  await Promise.resolve(); await Promise.resolve();
  h.reply({ code: 200, data: { menus: [], buttons: [] } }, 200, 1); await pending;
  assert.equal(h.user.info.value.name, 'Cached user'); assert.equal(h.user.loading.value, false);
});

function profilePage(h) {
  const actions = [];
  const source = read('pages/index/my.uvue').match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
  const deps = {
    ref: value => ({ value }), computed: fn => ({ get value() { return fn(); } }), watch: () => {}, useStore: () => ({ user: h.user }), request: options => h.api.request(options),
    router: { login: () => actions.push('login') }, config: {}, largeText: { value: false }, setLargeText: () => {},
    onUnload: () => {}, uni: { $on: () => {}, $off: () => {}, removeStorageSync: key => h.storage.remove(key), showToast: value => actions.push(value.title) },
  };
  const page = new Function('deps', `const { ${Object.keys(deps).join(',')} } = deps;\n${compile(source)}\nreturn { leave, leaving };`)(deps);
  return { ...page, actions };
}

test('late logout completion never clears or redirects a newer session', async () => {
  const h = harness(); h.login('A'); const page = profilePage(h);
  const pending = page.leave(); h.user.clear(); h.login('B');
  h.reply({ code: 200, msg: '退出成功' }); await pending;
  assert.equal(h.user.token, 'B'); assert.deepEqual(page.actions, []); assert.equal(page.leaving.value, false);
});

test('offline logout still clears the original local session and explains the unconfirmed server logout', async () => {
  const h = harness(); h.login(); const page = profilePage(h);
  const pending = page.leave(); h.requests[0].fail({ errMsg: 'request:fail network' }); await pending;
  assert.equal(h.user.token, null); assert.equal(page.actions[0], 'login'); assert.match(page.actions[1], /服务端退出未确认/);
});
