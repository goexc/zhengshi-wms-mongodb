import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import test from 'node:test';
import vm from 'node:vm';
import { transformSync } from 'esbuild';

const source = path => readFileSync(new URL(path, import.meta.url), 'utf8');
function evaluate(text, dependencies = {}, globals = {}) {
  const { code } = transformSync(text.replaceAll('import.meta.env', '({VITE_TITLE:"正时 WMS"})'), { loader: 'ts', format: 'cjs', target: 'es2020' });
  const module = { exports: {} };
  vm.runInNewContext(`(function(module,exports,require){${code}\n})`, globals)(module, module.exports, name => {
    assert.ok(name in dependencies, `Unexpected dependency: ${name}`);
    return dependencies[name];
  });
  return module.exports;
}
const navigation = evaluate(source('../src/utils/menu.ts'));
const { getNavigationTitle, prepareNavigationMenus } = navigation;
const plain = value => JSON.parse(JSON.stringify(value));
const menu = (path, sort_id = 0, extra = {}) => ({
  id: path, parent_id: 'root', type: 1, sort_id, path, name: path,
  component: '', meta: { title: '', hidden: false }, ...extra,
});

test('legacy titles, English route names and component aliases resolve without renaming routes', () => {
  assert.equal(getNavigationTitle(menu('/finance/acceptance', 0, { name: 'Finance_acceptance', meta: { title: 'Acceptance' } })), '客户承兑');
  assert.equal(getNavigationTitle(menu('/finance/invoice', 0, { name: 'Finance_invoice' })), '销售开票记录');
  assert.equal(getNavigationTitle(menu('/custom/invoices', 0, { component: '/finance/invoice/index.vue' })), '销售开票记录');
  assert.equal(getNavigationTitle(menu('/custom/outbound', 0, { component: '/src/views/outbound/receipt/index2.vue' })), '出库单管理');
  assert.equal(getNavigationTitle(menu('/custom', 0, { name: '我的客户台账' })), '我的客户台账');
  assert.equal(getNavigationTitle(menu('/finance/invoice', 0, { meta: { title: '  开票归档  ' } })), '开票归档');
});

test('every existing Web page has a Chinese fallback when stored metadata is English', () => {
  const pages = readdirSync(new URL('../src/views/', import.meta.url), { recursive: true })
    .map(path => path.replaceAll('\\', '/')).filter(path => path.endsWith('.vue') && !path.includes('/components/'));
  assert.ok(pages.length >= 34);
  for (const [index, component] of pages.entries()) {
    const label = getNavigationTitle(menu(`/custom-route-${index}`, 0, { name: `Page${index}`, component: `/${component}`, meta: { title: 'English Title' } }));
    assert.match(label, /[\u3400-\u9fff]/, `Missing Chinese title for ${component}`);
  }
});

test('finance follows partners at the same level, preserving identities and unrelated relative order', () => {
  const input = [menu('/finance', 60, { name: 'Finance', children: [menu('/finance/invoice', 2), menu('/finance/acceptance', 1)] }), menu('/acl', 5), menu('/home', 0), menu('/warehouse', 4), menu('/business_partner', 3)];
  const before = structuredClone(input);
  const result = prepareNavigationMenus(input);
  assert.deepEqual(plain(result.map(item => item.path)), ['/home', '/business_partner', '/finance', '/warehouse', '/acl']);
  const finance = result[2];
  assert.equal(finance.parent_id, 'root');
  assert.equal(finance.name, 'Finance');
  assert.equal(finance.sort_id, 60);
  assert.deepEqual(plain(finance.children.map(item => item.path)), ['/finance/acceptance', '/finance/invoice']);
  assert.equal(finance.meta.title, '客户财务');
  assert.deepEqual(input, before);
});

test('nested roots and restricted menus do not invent, reparent or reveal entries', () => {
  const root = menu('', 0, { meta: { hidden: true }, children: [menu('/finance', 60), menu('/acl', 5), menu('/business_partner', 3)] });
  assert.deepEqual(plain(prepareNavigationMenus([root])[0].children.map(item => item.path)), ['/business_partner', '/finance', '/acl']);
  const restricted = [menu('/finance', 60, { children: [menu('/finance/invoice', 1)] }), menu('/acl', 5)];
  assert.deepEqual(plain(prepareNavigationMenus(restricted).map(item => item.path)), ['/acl', '/finance']);
  assert.equal(prepareNavigationMenus(restricted)[1].children.length, 1);
  const hiddenPartner = [menu('/business_partner', 3, { meta: { hidden: true } }), menu('/acl', 5), menu('/finance', 60)];
  assert.deepEqual(plain(prepareNavigationMenus(hiddenPartner).map(item => item.path)), ['/business_partner', '/acl', '/finance']);
  assert.deepEqual(plain(prepareNavigationMenus([menu('/supplier'), menu('/business_partner/supplier')]).map(item => item.path)), ['/business_partner/supplier']);
});

test('authorized menu loading uses the same normalized titles for dynamic routes without changing permissions', async () => {
  const response = { menus: [menu('/finance', 60), menu('/warehouse', 4), menu('/business_partner', 3)], buttons: [{ name: '查看', perms: 'finance:invoice:list' }] };
  const original = structuredClone(response);
  const { useAuthStore: definition } = evaluate(source('../src/store/modules/auth.ts'), {
    pinia: { defineStore: (_id, definition) => definition },
    '@/api/modules/login': { getAuthMenuListApi: async () => ({ data: response }) },
    '@/utils/menu.ts': navigation,
  });
  const state = definition.state();
  await definition.actions.getAuthMenuList.call(state);
  assert.deepEqual(plain(state.authMenuList.map(item => item.path)), ['/business_partner', '/finance', '/warehouse']);
  assert.strictEqual(state.authButtonList, response.buttons);
  assert.deepEqual(response, original);
  assert.deepEqual(plain(definition.getters.flatMenuListGet(state).map(item => item.meta.title)), ['合作伙伴', '客户财务', '仓储管理']);
});

test('actual router guard shows Chinese browser titles while keeping route names for permissions', async () => {
  let guard;
  const browserDocument = { title: '' };
  const routeNames = [];
  const router = { beforeEach: callback => { guard = callback; }, afterEach() {}, onError() {} };
  evaluate(source('../src/router/index.ts'), {
    'vue-router': { createRouter: () => router, createWebHistory: () => ({}) },
    '@/store/modules/user.ts': { useUserStore: () => ({ token: 'isolated-test' }) },
    '@/store/modules/auth.ts': { useAuthStore: () => ({ authMenuListGet: [{}], setRouteName: name => routeNames.push(name) }) },
    '@/config': { LOGIN_URL: '/login', ROUTER_WHITE_LIST: ['/500'] },
    '@/config/nprogress': { start() {}, done() {} },
    '@/router/modules/dynamicRouter.ts': { initDynamicRouter: async () => {} },
    '@/router/modules/staticRouter.ts': { errorRouter: [], staticRouter: [] },
    '@/utils/menu.ts': navigation,
  }, { document: browserDocument });
  for (const [path, name, expected] of [['/finance/acceptance', 'Finance_acceptance', '客户承兑'], ['/finance/invoice', 'Finance_invoice', '销售开票记录'], ['/warehouse/rack', 'WarehouseRack', '货架管理']]) {
    await guard({ path, name, meta: { title: 'English' } }, {});
    assert.equal(browserDocument.title, `${expected} - 正时 WMS`);
    assert.equal(routeNames.at(-1), name);
  }
});
