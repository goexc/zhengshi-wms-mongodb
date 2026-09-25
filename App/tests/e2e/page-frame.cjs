// Run after the HBuilderX Web build. All API/image requests use local fixtures.
// Set PLAYWRIGHT_MODULE to an installed Playwright package when it is not on NODE_PATH.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const appRoot = path.resolve(__dirname, '../..');
const webRoot = path.join(appRoot, 'unpackage/dist/build/web');
const output = path.join(appRoot, 'unpackage/verification/page-frame-web');
fs.mkdirSync(output, { recursive: true });
const id = '0123456789abcdef01234567';
const specialModel = '型号 X & A+B/50%#?= 长编号 ABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789';
const materialPrices = (i) => i === 2 ? null : i === 3 ? [] : i === 1 ? [
  { customer_id: 'customer-a', customer_name: '华东精密制造有限公司', price: 12.5, since: 1789574400, source_valid: true },
  { customer_id: 'customer-a', customer_name: '华东精密制造有限公司', price: 10.125, since: 1786896000, source_valid: true },
  { customer_id: 'customer-b', customer_name: '华南精密制造与仓储物流设备有限公司（长客户名称）ABCDEFGHIJKLMNOPQRSTUVWXYZ', price: 123456.789, since: 1789574400, source_valid: true },
  { customer_id: '', customer_name: '', price: 0, since: 0, source_valid: true }
] : [{ customer_id: `customer-${i}`, customer_name: `物料 ${i} 专属客户`, price: i + 0.125, since: 1789574400, source_valid: true }];
const material = (i) => ({ id: `m${i}`, model: `M-${String(i).padStart(3, '0')}`, name: '六角螺栓', image: 'drawing.png', specification: 'M12×80', material: '35CrMo', unit: '件', category_id: 'cat', remark: '本地滚动验收数据', quantity: 10, prices: materialPrices(i) });
const partner = (i) => ({ id: i === 1 ? id : i.toString(16).padStart(24, '0'), name: `测试往来单位 ${i}`, code: `P-${i}`, type: '企业', manager: '测试负责人', contact: '13800000000', address: '本地测试地址', email: 'test@example.invalid', remark: '', image: '', status: '正常', receivable_balance: 123, credit_balance: 0, level: 1 });
const outbound = (i) => ({ id: `o${i}`, code: `OUT-${i}`, customer_id: id, customer_name: '测试客户', status: '已签收', type: '销售出库', receipt_time: 1800000000, departure_time: 1799900000, total_amount: 123 });
const line = (i) => ({ id: `line${i}`, index: i, order_code: 'OUT-1', code: 'OUT-1', material_id: 'm1', model: `M-${i}`, name: '六角螺栓', specification: 'M12×80', quantity: 10, price: 1.2, unit: '件', receipt_date: 1800000000, departure_date: 1799900000 });
const profile = { name: '本地测试账号', department_name: '仓储验收', avatar: '', mobile: '13800000000', email: '', department_id: '' };
const routes = [
  ['home', 'pages/index/home'], ['material', 'pages/material/index'], ['outbound', 'pages/outbound/order'],
  ['my', 'pages/index/my'], ['material-edit', 'pages/material/edit?id=m1'],
  ['outbound-detail', 'pages/outbound/detail?code=OUT-1'], ['report', 'pages/outbound/report'],
  ['customer', 'pages/business/customer'], ['supplier', 'pages/business/supplier'],
  ['ledger', `pages/business/customer-transactions?customer_id=${id}&name=测试客户`], ['plan', 'pages/plan/index'],
  ['about', 'pages/set/about'], ['privacy', 'pages/privacy/index'], ['login', 'pages/user/login']
];
const requestLog = [], failures = [], results = [], blockedRemote = [], interactions = {};
let materialSearchFailures = 0;
function fixture(url) {
  const page = Number(url.searchParams.get('page') || 1);
  const size = Number(url.searchParams.get('size') || 20);
  const paged = (make) => ({ total: 25, list: Array.from({ length: Math.max(0, Math.min(size, 25 - (page - 1) * size)) }, (_, n) => make((page - 1) * size + n + 1)) });
  switch (url.pathname) {
    case '/auth/login': return { ...profile, token: 'local-fixture-token', exp: Math.floor(Date.now() / 1000) + 3600 };
    case '/account/profile': return profile;
    case '/account/menu': return { menus: ['material', 'outbound', 'customer', 'supplier', 'plan'].map((name) => ({ path: `/${name}` })), buttons: [{ perms: 'material:material:edit' }] };
    case '/material': {
      if (size !== 10) return paged(material);
      const query = (url.searchParams.get('model') || '').trim();
      const models = query === 'FAIL' ? ['FAIL-01'] : query === 'LATE' ? ['LATE-01'] : query === '型号' ? [specialModel] : Array.from({ length: 10 }, (_, i) => `RGV41020300${35 + i}`).filter(model => model.includes(query));
      return { total: models.length, list: models.map((model, i) => ({ ...material(i + 1), model })) };
    }
    case '/material/info': return material(1);
    case '/material/category': return [{ id: 'cat', name: '紧固件', status: '正常' }];
    case '/outbound/page': return url.searchParams.get('code') ? { total: 1, list: [{ ...outbound(1), code: url.searchParams.get('code') }] } : url.searchParams.get('model') ? paged(i => ({ ...outbound(i), code: `MATCH-${i}` })) : paged(outbound);
    case '/outbound/materials': return Array.from({ length: 25 }, (_, i) => {
      const item = line(i + 1);
      if (url.searchParams.get('order_code') !== 'MATCH-1') return item;
      return { ...item, order_code: 'MATCH-1', model: i === 1 || i === 3 ? 'RGV4102030035' : i === 2 ? 'RGV41020300350' : i === 4 ? 'rgv4102030035' : i === 5 ? specialModel : item.model };
    });
    case '/outbound/summary': return Array.from({ length: 45 }, (_, i) => line(i + 1));
    case '/customer': case '/supplier': return paged(partner);
    case '/customer/list': return { list: Array.from({ length: 80 }, (_, i) => i === 0 ? { ...partner(1), name: '测试客户华东精密制造与仓储物流设备有限公司（长名称显示验收）', code: '' } : partner(i + 1)).filter(row => !url.searchParams.get('name') || row.name.includes(url.searchParams.get('name'))) };
    case '/customer/transaction': return paged((i) => ({ type: '应收账款', direction: 'receivable_increase', transaction_type: 'outbound_ar', status: 'confirmed', source_type: 'outbound_order', source_code: `OUT-${i}`, time: 1800000000 + i, amount: 12.5, remark: '', annex: '' }));
    case '/plan': return paged((i) => ({ id: `plan${i}`, type: '生产计划', status: '执行中', customer_name: '测试客户', supplier_name: '', material_id: 'm1', material_model: `M-${i}`, material_name: '六角螺栓', material_image: '', material_unit: '件', material_quantity: 20, deadline: 1800000000 }));
    default: throw new Error(`Unexpected API: ${url.pathname}`);
  }
}
const server = http.createServer((req, res) => {
  const requested = decodeURIComponent(new URL(req.url, 'http://localhost').pathname);
  const filename = path.resolve(webRoot, '.' + (requested === '/' ? '/index.html' : requested));
  if (!filename.startsWith(webRoot + path.sep)) { res.writeHead(403); res.end(); return; }
  const mime = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.png': 'image/png', '.svg': 'image/svg+xml', '.ttf': 'font/ttf' };
  if (!fs.existsSync(filename)) { res.writeHead(404); res.end(); return; }
  res.setHeader('Content-Type', mime[path.extname(filename)] || 'application/octet-stream');
  fs.createReadStream(filename).pipe(res);
});
async function main() {
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const base = `http://127.0.0.1:${server.address().port}`;
  const browser = await chromium.launch({ channel: 'msedge', headless: true });
  const context = await browser.newContext({ viewport: { width: 390, height: 844 }, hasTouch: true });
  const page = await context.newPage();
  page.on('pageerror', (error) => failures.push(error.message));
  await context.route('**/*', async (route) => {
    const url = new URL(route.request().url());
    if (url.origin === base) return route.continue();
    if (url.hostname === 'wms.file.goexc.cn') return route.fulfill({ contentType: 'image/png', body: fs.readFileSync(path.join(appRoot, 'static/logo.png')) });
    if (url.hostname !== 'wmsx.api.goexc.cn') { blockedRemote.push(url.href); return route.fulfill({ status: 204, body: '' }); }
    try {
      requestLog.push(url.pathname + url.search);
      if (url.pathname === '/material' && url.searchParams.get('size') === '10' && url.searchParams.get('model') === 'LATE') await new Promise(resolve => setTimeout(resolve, 700));
      if (url.pathname === '/material' && url.searchParams.get('model') === 'FAIL' && materialSearchFailures++ === 0) return route.fulfill({ contentType: 'application/json', headers: { 'access-control-allow-origin': '*' }, body: JSON.stringify({ code: 500, msg: '物料搜索测试失败，请重试' }) });
      await route.fulfill({ contentType: 'application/json', headers: { 'access-control-allow-origin': '*' }, body: JSON.stringify({ code: 200, data: fixture(url), msg: '' }) });
    } catch (error) { failures.push(error.message); await route.abort(); }
  });
  try {
    async function verifyOutboundSearchRow() {
      const layout = await page.locator('.order-search-row').evaluate(el => {
        const bounds = node => { const r = node.getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height }; };
        return { input: bounds(el.querySelector('.cl-input')), actions: [...el.querySelectorAll('.cl-button')].map(bounds), width: innerWidth };
      });
      assert.equal(layout.actions.length, 2);
      assert.ok(layout.input.width >= 130, 'compact search input remains usable on a narrow phone');
      for (const action of layout.actions) {
        assert.ok(action.width >= 48 && action.height >= 48, 'search actions must retain comfortable touch targets');
        assert.ok(Math.abs(action.y - layout.input.y) <= 1, 'input and buttons must share one row');
      }
      assert.ok(layout.actions[0].x >= layout.input.x + layout.input.width + 7);
      assert.ok(layout.actions[1].x >= layout.actions[0].x + layout.actions[0].width + 7);
      assert.ok(layout.actions[1].x + layout.actions[1].width <= layout.width - 15);
      assert.equal(await page.getByText('签收报表', { exact: true }).count(), 0);
      assert.equal(await page.locator('.wms-content > .wms-card').first().locator('.wms-label').count(), 0);
      return layout;
    }
    await page.goto(`${base}/#/pages/user/login`);
    await page.locator('.cl-input input').nth(0).fill('13800000000');
    await page.locator('.cl-input input').nth(1).fill('local-test-only');
    await page.locator('.cl-button').filter({ hasText: /^登录$/ }).click();
    await page.waitForURL((url) => ['#/', '#/pages/index/home'].includes(url.hash));
    await page.getByText('常用查询', { exact: true }).waitFor();
    const homeEntries = [
      ['物料查询', 'pages/material/index'], ['出库单', 'pages/outbound/order'],
      ['出库报表', 'pages/outbound/report'], ['客户', 'pages/business/customer'],
      ['供应商', 'pages/business/supplier'], ['计划任务', 'pages/plan/index']
    ];
    interactions.homeGrid = [];
    for (const width of [320, 360, 390]) {
      await page.setViewportSize({ width, height: 844 });
      await page.waitForFunction(() => Math.abs(document.querySelector('.cl-sticky').getBoundingClientRect().width - innerWidth) < 1);
      assert.deepEqual(await page.locator('.entry-card .entry-title').allTextContents(), homeEntries.map(([name]) => name));
      const cards = await page.locator('.entry-card').evaluateAll(nodes => nodes.map(node => {
        const bounds = node.getBoundingClientRect();
        return { x: bounds.x, y: bounds.y, width: bounds.width, height: bounds.height, right: bounds.right, bottom: bounds.bottom };
      }));
      for (let i = 0; i < cards.length; i += 2) {
        assert.ok(Math.abs(cards[i].y - cards[i + 1].y) < 1 && Math.abs(cards[i].width - cards[i + 1].width) < 1, 'home entries must form two equal columns');
        assert.ok(cards[i].height >= 140 && cards[i].width >= 130, 'home blocks must be comfortable touch targets');
        assert.ok(cards[i + 1].x - cards[i].right >= 11 && cards[i + 1].right <= width - 15, 'home column spacing and outer margins must remain visible');
        if (i > 0) assert.ok(cards[i].y >= cards[i - 2].bottom + 11, 'home rows must not overlap');
      }
      assert.deepEqual(await page.locator('.shell-tabs .shell-tab-active, .shell-tabs .shell-tab-label').allTextContents(), ['工作台', '我的']);
      await page.screenshot({ path: path.join(output, `home-grid-${width}.png`) });
      interactions.homeGrid.push({ width, columns: 2, entries: cards.length });
    }
    await page.locator('.shell-tab').filter({ hasText: '我的' }).click();
    await page.waitForURL('**/#/pages/index/my');
    await page.locator('.shell-tab').filter({ hasText: '工作台' }).click();
    await page.waitForURL(url => ['#/', '#/pages/index/home'].includes(url.hash));
    for (const [name, route] of homeEntries) {
      await page.locator('.entry-card').filter({ hasText: name }).click();
      await page.waitForURL(`**/#/${route}`);
      assert.equal(await page.locator('.shell-tabs:visible').count(), 0, `${name}: business page must not retain a tab footer`);
      await page.locator('.cl-topbar__prepend .cl-topbar__icon:visible').click();
      await page.waitForURL(url => ['#/', '#/pages/index/home'].includes(url.hash));
    }
    interactions.homeEntryNavigationAndReturn = true;
    interactions.twoBottomTabs = true;
    for (const [name, route] of routes) {
      await page.goto(`${base}/#/${route}`);
      await page.reload({ waitUntil: 'networkidle' });
      assert.equal(await page.locator('.shell-subtitle').count(), 0, `${name}: page subtitle must be removed`);
      assert.equal(await page.locator('.cl-topbar__inner').evaluate(el => el.getBoundingClientRect().height), 56, `${name}: single-line title bar must be 56px`);
      const before = await page.evaluate(() => ({ y: scrollY, max: document.documentElement.scrollHeight - innerHeight, width: document.documentElement.scrollWidth, viewport: innerWidth }));
      assert.ok(before.width <= before.viewport + 1, `${name}: horizontal overflow`);
      if (['material', 'outbound', 'customer', 'supplier', 'ledger', 'plan', 'outbound-detail'].includes(name)) assert.ok(before.max > 1000, `${name}: long fixture did not finish rendering`);
      await page.mouse.move(250, 420);
      await page.mouse.wheel(0, 900);
      if (before.max > 10) await page.waitForFunction(() => scrollY > 0);
      const down = await page.evaluate(() => ({ y: scrollY, top: document.querySelector('.cl-topbar').getBoundingClientRect().top, footerBottom: document.querySelector('.shell-tabs')?.getBoundingClientRect().bottom }));
      assert.ok(Math.abs(down.top) <= 1, `${name}: topbar did not stay sticky: ${down.top}`);
      if (down.footerBottom != null) assert.ok(Math.abs(down.footerBottom - 844) <= 1, `${name}: tab footer moved`);
      await page.mouse.wheel(0, -12000);
      await page.waitForFunction(() => scrollY === 0);
      await page.screenshot({ path: path.join(output, `${name}.png`) });
      results.push({ name, before, down, returnedToTop: true });
      console.log(`Verified scroll and sticky frame: ${name}`);
      if (['material', 'outbound', 'customer', 'supplier', 'ledger', 'plan'].includes(name)) {
        const start = requestLog.length;
        await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
        await page.waitForTimeout(400);
        assert.equal(requestLog.length, start, `${name}: scrolling still loads another page`);
        const pages = page.locator('.cl-pagination__item');
        await pages.filter({ hasText: /^2$/ }).click();
        await page.waitForFunction(() => document.querySelector('.wms-pagination')?.innerText.includes('第 2 页'));
        await page.waitForLoadState('networkidle');
        assert.ok(requestLog.slice(start).some(url => /page=2(?:&|$)/.test(url)), `${name}: page 2 was not requested`);
        const rowSelector = name === 'material' ? '.material-card' : name === 'outbound' ? '.order-code' : name === 'plan' ? '.plan-status' : name === 'ledger' ? '.wms-content .wms-card' : '.partner-status';
        await page.waitForFunction(selector => document.querySelectorAll(selector).length === 5, rowSelector);
        assert.equal(await page.locator(rowSelector).count(), 5, `${name}: page 2 must replace page 1`);
        if (name === 'material') {
          const firstCard = page.locator('.material-card').first();
          assert.match(await firstCard.innerText(), /物料 21 专属客户/);
          assert.match(await firstCard.innerText(), /¥ 21\.125/);
          assert.equal(await page.getByText('华东精密制造有限公司', { exact: true }).count(), 0);
          interactions.materialPricesFollowPagination = true;
        }
        await page.screenshot({ path: path.join(output, `${name}-page-2.png`) });
        await pages.filter({ hasText: /^1$/ }).click();
        await page.waitForFunction(selector => document.querySelectorAll(selector).length === 20, rowSelector);
        assert.equal(await page.locator(rowSelector).count(), 20, `${name}: return to page 1`);
        interactions[`${name}Pagination`] = true;
        if (name === 'outbound') {
          interactions.outboundSearchRow = await verifyOutboundSearchRow();
          const orderInput = page.locator('.order-search-row .cl-input input');
          await orderInput.fill('OUT-SEARCH');
          await page.getByText('OUT-SEARCH', { exact: true }).waitFor();
          const manualSearch = page.waitForRequest(request => {
            const url = new URL(request.url());
            return url.pathname === '/outbound/page' && url.searchParams.get('code') === 'OUT-SEARCH' && url.searchParams.get('page') === '1';
          });
          await page.locator('.order-search-row .cl-button').filter({ hasText: /^搜索$/ }).click();
          await manualSearch;
          await page.getByText('OUT-SEARCH', { exact: true }).waitFor();
          await page.locator('.order-search-row .cl-button').filter({ hasText: /^重置$/ }).click();
          await page.waitForFunction(() => document.querySelectorAll('.order-code').length === 20);
          assert.equal(await orderInput.inputValue(), '');
          interactions.outboundSearchAndResetButtons = true;
        }
      }
    }
    const materialPriceRequestStart = requestLog.length;
    await page.goto(`${base}/#/pages/material/index`);
    await page.reload({ waitUntil: 'networkidle' });
    await page.mouse.wheel(0, -24000);
    await page.waitForFunction(() => scrollY === 0);
    const materialUrl = page.url();
    const firstMaterial = page.locator('.material-card').first();
    assert.equal(await firstMaterial.locator('.material-prices .cl-list-item').count(), 4);
    assert.deepEqual(await firstMaterial.locator('.material-price-amount').allTextContents(), ['¥ 12.500', '¥ 10.125', '¥ 123456.789', '¥ 0.000']);
    assert.equal(await firstMaterial.getByText('华东精密', { exact: true }).count(), 2, 'shorten customer names without merging distinct price records');
    assert.equal(await firstMaterial.getByText('华东精密制造有限公司', { exact: true }).count(), 0);
    assert.match(await firstMaterial.innerText(), /2026-09-17 生效/);
    assert.match(await firstMaterial.innerText(), /2026-08-17 生效/);
    assert.match(await firstMaterial.innerText(), /未指定客户/);
    for (const index of [1, 2]) {
      const card = page.locator('.material-card').nth(index);
      assert.equal(await card.locator('.material-prices').count(), 0, 'null and empty prices must hide the entire price card');
      assert.doesNotMatch(await card.innerText(), /暂无客户价格|客户价格/);
    }
    const priceCardStyle = await firstMaterial.locator('.material-prices').evaluate(el => {
      const list = el.querySelector('.cl-list');
      const items = el.querySelector('.cl-list__items');
      return { cardBorder: getComputedStyle(el).borderTopWidth, cardRadius: getComputedStyle(el).borderTopLeftRadius, listBorder: getComputedStyle(list).borderTopWidth, itemsRadius: getComputedStyle(items).borderTopLeftRadius, separators: el.querySelectorAll('.material-price-divider').length };
    });
    assert.deepEqual(priceCardStyle, { cardBorder: '1px', cardRadius: '8px', listBorder: '0px', itemsRadius: '0px', separators: 3 });
    interactions.materialPriceCardVisibilityAndBorder = priceCardStyle;
    const priceStyle = await firstMaterial.locator('.material-price-amount .cl-text').first().evaluate(el => ({ size: getComputedStyle(el).fontSize, weight: getComputedStyle(el).fontWeight, color: getComputedStyle(el).color }));
    assert.deepEqual(priceStyle, { size: '22px', weight: '700', color: 'rgb(180, 35, 24)' });
    await firstMaterial.locator('.material-price-amount').first().click();
    assert.equal(page.url(), materialUrl, 'reading a price must not navigate to detail');
    assert.ok(!requestLog.slice(materialPriceRequestStart).some(url => url.startsWith('/material/price') || url.startsWith('/material/info')), 'list prices must reuse the material page response');
    interactions.materialInlinePrices = { count: 4, ...priceStyle, noExtraRequests: true };
    for (const width of [320, 360]) {
      await page.setViewportSize({ width, height: 844 });
      await page.waitForFunction(() => Math.abs(document.querySelector('.cl-sticky').getBoundingClientRect().width - innerWidth) < 1);
      await page.evaluate(() => scrollTo(0, 0));
      const textBounds = await firstMaterial.locator('.material-price-content').evaluateAll(rows => rows.map(row => ({ width: row.clientWidth, scrollWidth: row.scrollWidth })));
      assert.ok(textBounds.every(row => row.scrollWidth <= row.width + 1), 'long names and prices must wrap inside the list');
      const priceRows = await firstMaterial.locator('.material-price-main').evaluateAll(rows => rows.map(row => {
        const name = row.querySelector('.material-price-customer').getBoundingClientRect();
        const amount = row.querySelector('.material-price-amount').getBoundingClientRect();
        const date = row.querySelector('.material-price-date').getBoundingClientRect();
        const bounds = row.getBoundingClientRect();
        return { nameRight: name.right, amountLeft: amount.left, amountRight: amount.right, right: bounds.right, rowTop: bounds.top, amountTop: amount.top, amountBottom: amount.bottom, dateTop: date.top, dateRight: date.right };
      }));
      assert.ok(priceRows.every(row => row.amountLeft >= row.nameRight + 11 && Math.abs(row.amountRight - row.right) < 1 && Math.abs(row.amountTop - row.rowTop) < 1), 'prices must stay on the right of customer names without overlap');
      assert.ok(priceRows.every(row => Math.abs(row.dateRight - row.amountRight) < 1 && row.dateTop >= row.amountBottom + 3), 'effective dates must sit below prices on the same right edge');
      interactions.materialPriceNamesAndRightAlignment = true;
      assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth + 1));
      const searchBounds = await page.locator('.material-search .cl-button').evaluate(el => ({ right: el.getBoundingClientRect().right, width: innerWidth }));
      assert.ok(searchBounds.right <= searchBounds.width - 15, 'material search button must remain fully visible on small screens');
      await page.screenshot({ path: path.join(output, `material-prices-${width}.png`) });
    }
    await page.setViewportSize({ width: 390, height: 844 });
    await page.waitForFunction(() => Math.abs(document.querySelector('.cl-sticky').getBoundingClientRect().width - innerWidth) < 1);
    await page.evaluate(() => scrollTo(0, 0));
    await page.screenshot({ path: path.join(output, 'material-prices.png') });
    await page.locator('.material-card').nth(1).screenshot({ path: path.join(output, 'material-without-prices.png') });
    await page.locator('.material-thumbnail .cl-image__inner').first().click();
    const preview = page.locator('uni-page[style*="z-index: 999"]');
    await preview.waitFor({ state: 'visible' });
    assert.equal(page.url(), materialUrl, 'image preview navigated to material detail');
    await page.waitForFunction(() => {
      const dialog = document.querySelector('uni-page[style*="z-index: 999"]');
      return dialog && [...dialog.querySelectorAll('*')].some((node) => (node.getAttribute('src') || '').endsWith('/drawing.png') || /\/drawing\.png["')]/.test(getComputedStyle(node).backgroundImage));
    });
    await page.screenshot({ path: path.join(output, 'material-preview.png') });
    interactions.originalImagePreview = true;
    interactions.previewStopsNavigation = true;
    await page.touchscreen.tap(195, 300);
    await preview.waitFor({ state: 'hidden' });
    const cdp = await context.newCDPSession(page);
    async function swipe(from, to) {
      await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: 220, y: from }] });
      for (let i = 1; i <= 8; i++) {
        await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: 220, y: from + (to - from) * i / 8 }] });
        await page.waitForTimeout(25);
      }
      await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
    }
    await swipe(650, 300);
    await page.waitForFunction(() => scrollY > 100);
    interactions.touchScroll = true;
    // Stop touch inertia before setting up the separate pull-to-refresh gesture.
    await page.waitForTimeout(800);
    await page.evaluate(() => window.scrollTo({ top: 0, behavior: 'instant' }));
    await page.waitForFunction(() => scrollY === 0);
    const refreshStart = requestLog.length;
    await swipe(300, 720);
    await page.waitForFunction(() => document.body.innerText.includes('本页 20 条 / 共 25 条'));
    assert.ok(requestLog.slice(refreshStart).some((url) => url.startsWith('/material?') && /page=1(?:&|$)/.test(url)), 'pull refresh did not request page 1');
    interactions.pullRefresh = true;
    await page.locator('.material-model').first().click();
    assert.equal(page.url(), materialUrl, 'material cards no longer open a detail page');
    assert.equal(await page.getByText('查看详情 ›', { exact: true }).count(), 0);
    await firstMaterial.locator('.cl-button').filter({ hasText: /^保存图纸$/ }).click();
    await preview.waitFor({ state: 'visible' });
    await page.touchscreen.tap(195, 300);
    await preview.waitFor({ state: 'hidden' });
    await firstMaterial.locator('.cl-button').filter({ hasText: /^编辑物料$/ }).click();
    await page.waitForURL('**/#/pages/material/edit?id=m1');
    await page.getByText('保存修改', { exact: true }).waitFor();
    await page.locator('.cl-topbar__prepend .cl-topbar__icon:visible').click();
    await page.waitForURL(materialUrl);
    interactions.materialListActions = { noDetailNavigation: true, saveImage: true, editAndReturn: true };
    for (const route of ['pages/outbound/detail?code=OUT-1', 'pages/plan/index']) {
      await page.goto(`${base}/#/${route}`);
      await page.reload({ waitUntil: 'networkidle' });
      const start = requestLog.length;
      await page.locator('.cl-button').filter({ hasText: /^查询物料$/ }).first().click();
      await page.waitForURL('**/#/pages/material/index?model=M-1');
      await page.locator('.material-card').first().waitFor();
      assert.equal(await page.locator('.material-search input').inputValue(), 'M-1');
      assert.ok(requestLog.slice(start).some(url => url.startsWith('/material?') && new URL(url, base).searchParams.get('model') === 'M-1'));
      await page.reload({ waitUntil: 'networkidle' });
      assert.equal(await page.locator('.material-search input').inputValue(), 'M-1');
    }
    interactions.linkedMaterialSearch = true;
    // The same component is shared by the order filters and report customer selection.
    await page.goto(`${base}/#/pages/outbound/order`);
    await page.reload({ waitUntil: 'networkidle' });
    await page.locator('.filter-customer .cl-button').click();
    const popup = page.locator('.cl-popup.is-open');
    await popup.waitFor();
    await popup.getByText('测试往来单位 80', { exact: true }).waitFor({ state: 'attached' });
    await page.waitForFunction(() => document.querySelector('.cl-popup.is-open').getBoundingClientRect().top < 200);
    const scroller = popup.locator('.picker-scroller .uni-scroll-view > .uni-scroll-view');
    assert.equal(await popup.locator('.cl-list-item').count(), 80);
    const listStyles = await popup.locator('.cl-list').evaluate(el => ({
      borderWidth: getComputedStyle(el).borderTopWidth,
      borderStyle: getComputedStyle(el).borderTopStyle,
      radius: getComputedStyle(el.querySelector('.cl-list__items')).borderTopLeftRadius,
      padding: getComputedStyle(el.querySelector('.cl-list-item__inner')).paddingLeft,
      rowHeight: el.querySelector('.cl-list-item').getBoundingClientRect().height,
      nameSize: getComputedStyle(el.querySelector('.picker-choice > .cl-text')).fontSize,
      codeSize: getComputedStyle(el.querySelector('.picker-code .cl-text')).fontSize,
      nameHeight: el.querySelector('.picker-choice > .cl-text').getBoundingClientRect().height
    }));
    assert.equal(listStyles.borderStyle, 'solid');
    assert.ok(parseFloat(listStyles.borderWidth) >= 1 && parseFloat(listStyles.radius) >= 12);
    assert.equal(listStyles.padding, '12px');
    assert.equal(listStyles.nameSize, '16px');
    assert.equal(listStyles.codeSize, '13px');
    assert.ok(listStyles.rowHeight >= 76 && listStyles.nameHeight > 30, 'long names should wrap inside a comfortably sized row');
    await popup.getByText('未填写客户编号', { exact: true }).waitFor();
    await page.screenshot({ path: path.join(output, 'customer-list-styled.png') });
    interactions.customerListStyles = listStyles;
    const dims = await scroller.evaluate(el => ({ height: el.clientHeight, total: el.scrollHeight }));
    assert.ok(dims.height > 100 && dims.height < 650 && dims.total > dims.height * 2, `popup lacks bounded scrolling: ${JSON.stringify(dims)}`);
    await swipe(700, 430);
    await page.waitForFunction(() => document.querySelector('.cl-popup.is-open .picker-scroller .uni-scroll-view > .uni-scroll-view').scrollTop > 100);
    await page.waitForTimeout(800);
    const down = await scroller.evaluate(el => el.scrollTop);
    await swipe(450, 700);
    await page.waitForFunction(old => document.querySelector('.cl-popup.is-open .picker-scroller .uni-scroll-view > .uni-scroll-view').scrollTop < old, down);
    await page.waitForTimeout(800);
    await popup.waitFor({ state: 'visible' });
    await scroller.evaluate(el => { el.scrollTop = el.scrollHeight; });
    await page.screenshot({ path: path.join(output, 'customer-popup-bottom.png') });
    await popup.getByText('测试往来单位 80', { exact: true }).click();
    await popup.waitFor({ state: 'hidden' });
    await page.locator('.filter-customer').getByText('测试往来单位 80', { exact: true }).waitFor();
    await page.locator('.filter-customer .cl-button').click();
    await popup.waitFor();
    await popup.getByText('已选择', { exact: true }).waitFor({ state: 'attached' });
    await page.waitForFunction(() => document.querySelector('.cl-popup.is-open').getBoundingClientRect().top < 200);
    await scroller.evaluate(el => { el.scrollTop = el.scrollHeight; });
    assert.equal(await popup.locator('.picker-selected').count(), 1);
    assert.equal(await popup.locator('.picker-selected .cl-text').evaluate(el => getComputedStyle(el).color), 'rgb(22, 101, 52)');
    await page.waitForFunction(() => document.querySelector('.picker-selected').getBoundingClientRect().bottom <= innerHeight);
    await page.screenshot({ path: path.join(output, 'customer-list-selected.png') });
    await popup.locator('.cl-button').filter({ hasText: /^取消$/ }).click();
    await popup.waitFor({ state: 'hidden' });
    assert.match(await page.locator('.filter-customer').innerText(), /测试往来单位 80/);
    interactions.customerPopupTouchScroll = true;
    interactions.customerPopupSelectAndCancel = true;
    assert.doesNotMatch(await page.locator('body').innerText(), /状态范围|出库类型|更多筛选/);
    const modelTrigger = page.locator('.material-model-select > .cl-button');
    const materialPopup = page.locator('.material-model-popup.is-open');
    const modelInput = materialPopup.locator('.cl-input input');
    async function openMaterialPopup() {
      await modelTrigger.click();
      await materialPopup.waitFor();
      await page.waitForFunction(() => document.querySelector('.material-model-popup.is-open').getBoundingClientRect().top < innerHeight * 0.21);
    }
    const materialStart = requestLog.length;
    await openMaterialPopup();
    await materialPopup.getByText('请输入物料编号进行搜索', { exact: true }).waitFor();
    await modelInput.fill('RGV');
    await materialPopup.getByText('RGV4102030035', { exact: true }).waitFor();
    assert.ok(requestLog.slice(materialStart).some(url => url.startsWith('/material?') && new URL(url, base).searchParams.get('model') === 'RGV' && new URL(url, base).searchParams.get('size') === '10'));
    assert.ok(!requestLog.slice(materialStart).some(url => url.startsWith('/outbound/page?')), 'typing must not submit a partial model as an order filter');
    assert.equal(await materialPopup.locator('.cl-list-item').count(), 10);
    const materialScroller = materialPopup.locator('.material-model-options .uni-scroll-view > .uni-scroll-view');
    const candidateBox = await materialScroller.boundingBox();
    assert.ok(candidateBox && candidateBox.height > 250 && candidateBox.height < 550);
    await swipe(candidateBox.y + candidateBox.height - 15, candidateBox.y + 15);
    await page.waitForFunction(() => document.querySelector('.material-model-options .uni-scroll-view > .uni-scroll-view').scrollTop > 50);
    await page.waitForTimeout(800);
    const materialDown = await materialScroller.evaluate(el => el.scrollTop);
    await swipe(candidateBox.y + 15, candidateBox.y + candidateBox.height - 15);
    await page.waitForFunction(old => document.querySelector('.material-model-options .uni-scroll-view > .uni-scroll-view').scrollTop < old, materialDown);
    await page.waitForTimeout(800);
    await materialScroller.evaluate(el => { el.scrollTop = 0; });
    interactions.outboundMaterialCandidateScroll = true;
    await page.screenshot({ path: path.join(output, 'outbound-material-candidates.png') });
    await materialPopup.getByText('RGV4102030035', { exact: true }).click();
    await page.getByText('MATCH-1', { exact: true }).waitFor();
    await materialPopup.waitFor({ state: 'hidden' });
    await page.locator('.material-model-popup').waitFor({ state: 'detached' });
    assert.ok(await page.evaluate(() => document.activeElement?.tagName !== 'INPUT'), 'closing the popup must release its input without requiring a native keyboard API');
    interactions.outboundMaterialPopupReleasesInput = true;
    assert.equal(await modelTrigger.innerText(), 'RGV4102030035');
    await page.screenshot({ path: path.join(output, 'outbound-material-filtered.png') });
    await page.locator('.cl-pagination__item').filter({ hasText: /^2$/ }).click();
    await page.getByText('MATCH-21', { exact: true }).waitFor();
    const filteredPage = new URL(requestLog.filter(url => url.startsWith('/outbound/page?')).at(-1), base);
    assert.equal(filteredPage.searchParams.get('model'), 'RGV4102030035');
    assert.equal(filteredPage.searchParams.get('customer_id'), partner(80).id);
    assert.equal(filteredPage.searchParams.get('page'), '2');
    const cancelStart = requestLog.length;
    await openMaterialPopup();
    assert.equal(await modelInput.inputValue(), 'RGV4102030035');
    await materialPopup.locator('.cl-list-item .cl-icon').waitFor();
    await modelInput.fill('NONE');
    await materialPopup.getByText('没有匹配的物料，请调整编号', { exact: true }).waitFor();
    await modelInput.fill('');
    await materialPopup.getByText('请输入物料编号进行搜索', { exact: true }).waitFor();
    await materialPopup.locator('.cl-button').filter({ hasText: /^取消$/ }).click();
    await materialPopup.waitFor({ state: 'hidden' });
    assert.equal(await modelTrigger.innerText(), 'RGV4102030035');
    await page.getByText('MATCH-21', { exact: true }).waitFor();
    assert.ok(!requestLog.slice(cancelStart).some(url => url.startsWith('/outbound/page?')), 'editing, clearing search and cancelling must preserve both the filter and current page');
    interactions.outboundMaterialCancelPreservesFilterAndPage = true;
    await openMaterialPopup();
    assert.equal(await modelInput.inputValue(), 'RGV4102030035');
    await materialPopup.locator('.cl-button').filter({ hasText: /^全部物料$/ }).click();
    await materialPopup.waitFor({ state: 'hidden' });
    await page.getByText('OUT-1', { exact: true }).waitFor();
    assert.equal(new URL(requestLog.filter(url => url.startsWith('/outbound/page?')).at(-1), base).searchParams.get('model'), '');
    assert.equal(new URL(requestLog.filter(url => url.startsWith('/outbound/page?')).at(-1), base).searchParams.get('page'), '1');
    assert.equal(await modelTrigger.innerText(), '选择物料编号');
    await openMaterialPopup();
    await modelInput.fill('FAIL');
    await materialPopup.getByText('物料搜索测试失败，请重试', { exact: true }).waitFor();
    await materialPopup.locator('.cl-button').filter({ hasText: /^重试$/ }).click();
    await materialPopup.getByText('FAIL-01', { exact: true }).waitFor();
    const lateRequest = page.waitForRequest(request => new URL(request.url()).searchParams.get('model') === 'LATE');
    await modelInput.fill('LATE');
    await lateRequest;
    await materialPopup.locator('.cl-popup__header .cl-icon').click();
    await materialPopup.waitFor({ state: 'hidden' });
    await openMaterialPopup();
    await page.waitForTimeout(900);
    assert.equal(await modelInput.inputValue(), '');
    assert.equal(await materialPopup.locator('.cl-list-item').count(), 0, 'a response from the closed popup cannot populate a reopened search');
    await materialPopup.getByText('请输入物料编号进行搜索', { exact: true }).waitFor();
    await page.locator('.cl-popup-mask.is-open').click({ position: { x: 20, y: 20 } });
    await materialPopup.waitFor({ state: 'hidden' });
    await page.locator('.cl-button').filter({ hasText: /^重置$/ }).click();
    assert.equal(await modelTrigger.innerText(), '选择物料编号');
    await page.locator('.filter-customer').getByText('全部客户', { exact: true }).waitFor();
    interactions.outboundMaterialRemoteSelect = true;
    interactions.outboundMaterialPagination = true;
    interactions.outboundMaterialEmptyErrorRetryReset = true;
    interactions.outboundMaterialPopupCloseDiscardsLateResults = true;
    await page.goto(`${base}/#/pages/outbound/report`);
    await page.reload({ waitUntil: 'networkidle' });
    await page.locator('.cl-button').filter({ hasText: /^请选择客户$/ }).click();
    await popup.waitFor();
    await popup.locator('.cl-input input').fill('不存在的客户');
    await popup.getByText('没有匹配的客户', { exact: true }).waitFor();
    await popup.locator('.cl-input input').fill('单位 8');
    await popup.getByText('测试往来单位 8', { exact: true }).waitFor();
    await popup.getByText('测试往来单位 8', { exact: true }).click();
    await popup.waitFor({ state: 'hidden' });
    await page.locator('.cl-button').filter({ hasText: /^查询报表$/ }).click();
    await page.locator('.wms-pagination').waitFor();
    const reportSummary = await page.locator('.report-summary').innerText();
    assert.match(reportSummary, /单据总数\s*1 张/);
    assert.match(reportSummary, /物料金额合计（元）\s*540\.000/);
    assert.match(reportSummary, /物料数量合计\s*450件/);
    await page.locator('.cl-pagination__item').filter({ hasText: /^3$/ }).click();
    await page.getByText('明细 41', { exact: true }).waitFor();
    assert.equal(await page.locator('.wms-content .wms-record').count(), 5, 'report: final page has five detail records');
    assert.equal(await page.locator('.report-summary').count(), 1, 'report: complete query totals remain separate from detail cards');
    assert.equal(await page.locator('.report-summary').innerText(), reportSummary, 'pagination must preserve the complete query totals');
    await page.screenshot({ path: path.join(output, 'report-page-3.png') });
    interactions.reportPagination = true;
    interactions.customerPopupSearch = true;
    await cdp.detach();
    await page.setViewportSize({ width: 360, height: 640 });
    await page.goto(`${base}/#/pages/outbound/order`);
    await page.reload({ waitUntil: 'networkidle' });
    await page.locator('.filter-customer .cl-button').click();
    await popup.waitFor();
    await popup.getByText('测试往来单位 80', { exact: true }).waitFor({ state: 'attached' });
    await page.waitForFunction(() => document.querySelector('.cl-popup.is-open').getBoundingClientRect().top < 150);
    assert.ok(await scroller.evaluate(el => el.clientHeight > 150 && el.scrollHeight > el.clientHeight));
    assert.ok(await popup.locator('.cl-list').evaluate(el => el.scrollWidth <= el.clientWidth));
    await page.screenshot({ path: path.join(output, 'customer-list-long-name-small.png') });
    await scroller.evaluate(el => { el.scrollTop = el.scrollHeight; });
    await page.screenshot({ path: path.join(output, 'customer-popup-small.png') });
    await popup.getByText('测试往来单位 80', { exact: true }).click();
    await popup.waitFor({ state: 'hidden' });
    interactions.customerPopupSmallScreen = true;
    await openMaterialPopup();
    await modelInput.fill('RGV');
    await materialPopup.getByText('RGV4102030035', { exact: true }).waitFor();
    const materialSmall = await materialScroller.evaluate(el => ({ height: el.clientHeight, total: el.scrollHeight, width: el.scrollWidth, viewport: el.clientWidth }));
    assert.ok(materialSmall.height > 150 && materialSmall.total > materialSmall.height && materialSmall.width <= materialSmall.viewport);
    await page.screenshot({ path: path.join(output, 'outbound-material-popup-small.png') });
    await materialPopup.locator('.cl-button').filter({ hasText: /^取消$/ }).click();
    await materialPopup.waitFor({ state: 'hidden' });
    interactions.outboundMaterialPopupSmallScreen = materialSmall;
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${base}/#/pages/outbound/order`);
    await page.reload({ waitUntil: 'networkidle' });
    await openMaterialPopup();
    await modelInput.fill('RGV');
    await materialPopup.getByText('RGV4102030035', { exact: true }).click();
    await materialPopup.waitFor({ state: 'hidden' });
    await page.getByText('MATCH-1', { exact: true }).click();
    await page.waitForURL(url => url.hash.includes('/pages/outbound/detail?'));
    assert.equal(new URLSearchParams(page.url().split('?')[1]).get('model'), 'RGV4102030035');
    await page.getByText('已突出显示 2 项匹配明细，其他明细仍完整保留。', { exact: true }).waitFor();
    assert.equal(await page.locator('.detail-material').count(), 25);
    assert.equal(await page.locator('.detail-material-match').count(), 2);
    assert.deepEqual(await page.locator('.detail-model-match').allTextContents(), ['RGV4102030035', 'RGV4102030035']);
    const highlightStyle = await page.locator('.detail-material-match').first().evaluate(el => ({
      background: getComputedStyle(el).backgroundColor,
      border: getComputedStyle(el).borderLeftWidth,
      modelColor: getComputedStyle(el.querySelector('.detail-model-match')).color,
      label: el.querySelector('.detail-match-badge').textContent
    }));
    assert.equal(highlightStyle.background, 'rgb(255, 245, 243)');
    assert.equal(highlightStyle.border, '4px');
    assert.equal(highlightStyle.modelColor, 'rgb(180, 35, 24)');
    assert.match(highlightStyle.label, /筛选物料/);
    const similarCard = page.locator('.detail-material').filter({ has: page.getByText('RGV41020300350', { exact: true }) });
    assert.ok(!(await similarCard.getAttribute('class')).includes('detail-material-match'));
    await page.locator('.detail-material-match').first().evaluate(el => window.scrollTo(0, scrollY + el.getBoundingClientRect().top - 100));
    await page.screenshot({ path: path.join(output, 'outbound-detail-highlight.png') });
    await page.reload({ waitUntil: 'networkidle' });
    assert.equal(await page.locator('.detail-material-match').count(), 2, 'reload must preserve the highlight from query parameters');
    interactions.outboundDetailHighlight = highlightStyle;
    interactions.outboundDetailHighlightReload = true;
    for (const model of ['', 'MISSING']) {
      await page.goto(`${base}/#/pages/outbound/detail?code=OUT-1${model ? '&model=' + model : ''}`);
      await page.reload({ waitUntil: 'networkidle' });
      assert.equal(await page.locator('.detail-material-match').count(), 0);
      assert.equal(await page.locator('.detail-material').count(), 25);
      if (model) await page.getByText('当前明细中未找到该编号，已保留完整单据内容。', { exact: true }).waitFor();
      else assert.equal(await page.locator('.detail-match-summary').count(), 0);
    }
    interactions.outboundDetailUnfilteredAndUnmatched = true;
    await page.setViewportSize({ width: 360, height: 640 });
    await page.goto(`${base}/#/pages/outbound/order`);
    await page.reload({ waitUntil: 'networkidle' });
    await openMaterialPopup();
    await modelInput.fill('型号');
    await materialPopup.getByText(specialModel, { exact: true }).click();
    await materialPopup.waitFor({ state: 'hidden' });
    await page.getByText('MATCH-1', { exact: true }).click();
    await page.waitForURL(url => url.hash.includes('/pages/outbound/detail?'));
    // uni-app Web preserves URL-encoded values in its router query, then decodes
    // page options before onLoad. Verify the page receives the original value.
    assert.equal(decodeURIComponent(new URLSearchParams(page.url().split('?')[1]).get('model')), specialModel);
    await page.getByText('已突出显示 1 项匹配明细，其他明细仍完整保留。', { exact: true }).waitFor();
    assert.equal(await page.locator('.detail-model-match').innerText(), specialModel);
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    for (const selector of ['.detail-filter-model', '.detail-model-match']) {
      assert.ok(await page.locator(selector).evaluate(el => {
        const bounds = el.getBoundingClientRect();
        const textRange = document.createRange();
        textRange.selectNodeContents(el);
        return [...textRange.getClientRects()].every(rect => rect.left >= bounds.left - 1 && rect.right <= bounds.right + 1);
      }), `${selector}: every character of the long model must fit within the text bounds`);
    }
    await page.locator('.detail-material-match').evaluate(el => window.scrollTo(0, scrollY + el.getBoundingClientRect().top - 90));
    await page.screenshot({ path: path.join(output, 'outbound-detail-highlight-small.png') });
    await page.reload({ waitUntil: 'networkidle' });
    assert.equal(await page.locator('.detail-model-match').innerText(), specialModel, 'a refreshed special-character route must keep the same complete model');
    interactions.outboundDetailSpecialModelSmallScreen = true;
    interactions.compactSearchSmallScreens = [];
    for (const width of [320, 360]) {
      await page.setViewportSize({ width, height: 640 });
      await page.goto(`${base}/#/pages/outbound/order`);
      await page.reload({ waitUntil: 'networkidle' });
      interactions.compactSearchSmallScreens.push(await verifyOutboundSearchRow());
      await page.screenshot({ path: path.join(output, `outbound-compact-${width}.png`) });
    }
    await page.goto(`${base}/#/pages/user/login`);
    await page.reload({ waitUntil: 'networkidle' });
    await page.mouse.wheel(0, 800);
    await page.waitForFunction(() => scrollY > 0);
    await page.screenshot({ path: path.join(output, 'login-small-scrolled.png') });
    interactions.smallScreenScroll = true;
    assert.deepEqual(failures, []);
    console.log(JSON.stringify({ routes: results.length, ...interactions, output }));
  } catch (error) {
    await page.screenshot({ path: path.join(output, 'failure.png') });
    console.error(JSON.stringify({ url: page.url(), text: (await page.locator('body').innerText()).slice(0, 300), failures, layout: await page.evaluate(() => [...document.querySelectorAll('html,body,uni-app,uni-page,uni-page-wrapper,uni-page-body,.wms-page')].map((el) => ({ tag: el.tagName, cls: el.className, scrollHeight: el.scrollHeight, height: el.clientHeight, scrollTop: el.scrollTop, style: el.getAttribute('style'), css: { overflow: getComputedStyle(el).overflow, position: getComputedStyle(el).position, height: getComputedStyle(el).height } }))) }));
    throw error;
  } finally {
    fs.writeFileSync(path.join(output, 'result.json'), JSON.stringify({ results, interactions, requestLog, failures, blockedRemote }, null, 2));
    await browser.close();
    server.close();
  }
}
main().catch((error) => { console.error(error); server.close(); process.exitCode = 1; });
