import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";
import { stripTypeScriptTypes } from "node:module";
import { computed, ref, watch } from "vue";
import { stripImports } from "./helpers/source.mjs";

function source(file, page = false) {
 let value = fs.readFileSync(new URL("../" + file, import.meta.url), "utf8");
 if (page) value = value.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
 return stripTypeScriptTypes(stripImports(value).replace(/^export /gm, ""));
}
const run = (context, code) => vm.runInContext(code, context);
const plain = value => JSON.parse(JSON.stringify(value));
const settle = () => new Promise(resolve => setImmediate(resolve));
function domain(context) { run(context, source("types/customer-documents.ts")); }

test("document service preserves scoped filters, attachment arrays and invoice rounding", async () => {
 const calls = []; let response = { total: 0, list: null, summary: null };
 const c = vm.createContext({ parse: v => v, request: async options => { calls.push(plain(options)); return response; } }); domain(c); run(c, source("services/customer-documents.ts"));
 const result = await run(c, "fetchCustomerDocuments('acceptance','0123456789abcdef01234567',' 票号 ','pending',2)");
 assert.equal(result.total, 0); assert.deepEqual(plain(result.list), []); assert.equal(result.summary.overdue_amount, 0);
 assert.deepEqual(calls[0], { url: "/customer/acceptance", method: "GET", data: { customer_id: "0123456789abcdef01234567", keyword: "票号", status: "pending", page: 2, size: 20 } });
 response = { id: "0123456789abcdef01234567", tax_rate: 13, tax_amount: 12.99, total_amount: 112.99, annex: ["invoice/2026/10/1.pdf", "invoice/2026/10/2.png"] };
 const detail = await run(c, "fetchCustomerDocument('invoice','0123456789abcdef01234567')");
 assert.equal(detail.tax_amount, 12.99); assert.equal(detail.annex.length, 2); assert.equal(calls[1].url, "/customer/invoice/detail");
 await assert.rejects(run(c, "fetchCustomerDocument('acceptance','invalid')"), /标识无效/);
 await assert.rejects(run(c, "fetchCustomerDocuments('payment','','','',1)"), /台账类型无效/);
 assert.equal(calls.length, 2);
});

test("image attachments preview while PDF and office files open in the system browser", () => {
 const previews = [], opens = [];
 const c = vm.createContext({ materialImageUrl: p => "https://files.test/images/" + p, previewMaterialImage: p => previews.push(p), openWeb: p => { opens.push(p); return true; }, uni: { showToast: () => {} } });
 run(c, source("services/customer-documents.ts"));
 run(c, "openDocumentAttachment('acceptance/2026/10/image.png'); openDocumentAttachment('invoice/2026/10/invoice.pdf'); openDocumentAttachment('invoice/2026/10/a.ofd')");
 assert.deepEqual(previews, ["https://files.test/images/acceptance/2026/10/image.png"]); assert.equal(opens.length, 2); assert.match(opens[0], /invoice\.pdf$/);
 assert.equal(run(c, "documentStatus('redeemed',1)"), "已到账"); assert.equal(run(c, "documentStatus('voided',1)"), "已作废");
});

test("finance list applies searches once, ignores stale results and clears data on permission loss", async () => {
 const pending = [], stops = []; const allowed = ref(true);
 const c = vm.createContext({ ref, computed, watch: (v, cb) => { const stop = watch(v, cb, { flush: "sync" }); stops.push(stop); return stop; }, useStore: () => ({ user: { hasPermission: p => allowed.value && p === "finance:acceptance:list" } }), onLoad: () => {}, onUnload: () => {}, uni: { $on: () => {}, $off: () => {} }, router: {}, errorMessage: e => e.message, fetchCustomerDocuments: (...args) => new Promise((resolve, reject) => pending.push({ args, resolve, reject })) });
 domain(c); run(c, source("pages/business/customer-documents.uvue", true));
 try {
  const older = run(c, "load(1)"); run(c, "keyword.value='新票'; search()");
  pending[1].resolve({ total: 21, list: [{ id: "new" }], summary: {} }); await Promise.resolve();
  pending[0].resolve({ total: 1, list: [{ id: "old" }], summary: {} }); await older;
  assert.equal(run(c, "items.value[0].id"), "new");
  run(c, "keyword.value='尚未搜索'"); const page2 = run(c, "load(2)"); assert.equal(pending[2].args[2], "新票"); assert.equal(pending[2].args[4], 2);
  allowed.value = false; pending[2].resolve({ total: 21, list: [{ id: "secret" }], summary: { total_amount: 999 } }); await page2;
  assert.equal(run(c, "items.value.length"), 0); assert.equal(run(c, "total.value"), 0); assert.equal(run(c, "summary.value.total_amount"), 0);
 } finally { stops.forEach(stop => stop()); }
});

test("document detail cannot be restored by a late previous-session response", async () => {
 let resolve; const allowed = ref(true); const stops = [];
 const c = vm.createContext({ ref, computed, watch: (v, cb) => { const stop = watch(v, cb, { flush: "sync" }); stops.push(stop); return stop; }, useStore: () => ({ user: { hasPermission: () => allowed.value } }), onLoad: () => {}, onUnload: () => {}, uni: { $on: () => {}, $off: () => {} }, errorMessage: e => e.message, fetchCustomerDocument: () => new Promise(done => { resolve = done; }) });
 domain(c); run(c, source("pages/business/customer-document-detail.uvue", true));
 try { const request = run(c, "id.value='0123456789abcdef01234567'; load()"); allowed.value = false; resolve({ id: "secret", customer_name: "客户", amount: 100 }); await request; assert.equal(run(c, "ready.value"), false); assert.equal(run(c, "record.value.id"), ""); } finally { stops.forEach(stop => stop()); }
});

test("shared shell requires the exact finance grant independently of customer directory access", () => {
 const finance = ref(false), customer = ref(true);
 const c = vm.createContext({ ref, computed, watch: () => {}, nextTick: () => {}, largeText: ref(false), defineOptions: () => {}, defineEmits: () => () => {}, defineProps: defs => { const props = Object.fromEntries(Object.entries(defs).map(([k, v]) => [k, v.default])); props.requiredPermission = "finance:invoice:list"; return props; }, useStore: () => ({ user: { token: "valid", revision: ref(0), canAccess: () => customer.value, hasPermission: p => finance.value && p === "finance:invoice:list" } }), router: { path: () => "/pages/business/customer-document-detail" }, onMounted: () => {}, onUnmounted: () => {}, onResize: () => {} });
 run(c, source("components/wms/Page.uvue", true)); assert.equal(run(c, "accessible.value"), false); finance.value = true; customer.value = false; assert.equal(run(c, "accessible.value"), true);
});

test("session change removes customer filters and stops queued navigation and previous-session results", async () => {
 const pending = [], stops = [], routes = [], events = new Map(); const permission = ref(true); let mount, unmount;
 const c = vm.createContext({ ref, computed, watch: (v, cb) => { const stop = watch(v, cb, { flush: "sync" }); stops.push(stop); return stop; }, useStore: () => ({ user: { hasPermission: () => permission.value } }), onLoad: cb => { mount = cb; }, onUnload: cb => { unmount = cb; }, uni: { $on: (name, cb) => events.set(name, cb), $off: name => events.delete(name) }, router: { to: route => routes.push(route) }, documentKind: value => value, errorMessage: e => e.message, fetchCustomerDocuments: (...args) => new Promise(resolve => pending.push({ args, resolve })) });
 domain(c); run(c, source("pages/business/customer-documents.uvue", true));
 try {
  mount({ kind: "acceptance", customer_id: "0123456789abcdef01234567", name: "旧客户" });
  run(c, "keyword.value='旧票'; appliedKeyword.value='旧票'; status.value='pending'; currentPage.value=3");
  permission.value = false; events.get("wms.session.changed")();
  assert.equal(run(c, "customerId.value"), ""); assert.equal(run(c, "customerName.value"), "全部客户");
  for (const name of ["keyword", "appliedKeyword", "status"]) assert.equal(run(c, `${name}.value`), "");
  assert.equal(run(c, "currentPage.value"), 1);
  run(c, "openDetail('old')"); assert.equal(routes.length, 0);
  permission.value = true;
  assert.equal(pending.length, 2); assert.equal(pending[1].args[1], ""); assert.equal(pending[1].args[2], "");
  pending[0].resolve({ total: 1, list: [{ id: "old", customer_name: "旧客户" }], summary: {} }); await settle();
  assert.equal(run(c, "items.value.length"), 0);
  pending[1].resolve({ total: 1, list: [{ id: "new" }], summary: {} }); await settle();
  run(c, "openDetail('new')"); assert.equal(routes.length, 1);
  unmount(); run(c, "openDetail('new'); load(1)");
  assert.equal(routes.length, 1); assert.equal(pending.length, 2); assert.equal(events.size, 0);
 } finally { stops.forEach(stop => stop()); }
});

test("document attachment clicks require current permission, active detail and owned references", async () => {
 const opened = [], stops = [], events = new Map(); const permission = ref(true); let mount, unmount, calls = 0;
 const c = vm.createContext({ ref, computed, watch: (v, cb) => { const stop = watch(v, cb, { flush: "sync" }); stops.push(stop); return stop; }, useStore: () => ({ user: { hasPermission: () => permission.value } }), onLoad: cb => { mount = cb; }, onUnload: cb => { unmount = cb; }, uni: { $on: (name, cb) => events.set(name, cb), $off: name => events.delete(name) }, documentKind: value => value, errorMessage: e => e.message, openDocumentAttachment: ref => opened.push(ref), fetchCustomerDocument: async () => { calls++; return { id: "0123456789abcdef01234567", annex: ["old.png"], redemption_annex: ["acceptance/redemption/voucher.pdf"] }; } });
 domain(c); run(c, source("pages/business/customer-document-detail.uvue", true));
 try {
  mount({ kind: "acceptance", id: "0123456789abcdef01234567" }); await settle();
  run(c, "openAttachment('old.png'); openAttachment('acceptance/redemption/voucher.pdf'); openAttachment('unrelated.pdf')");
  assert.deepEqual(opened, ["old.png", "acceptance/redemption/voucher.pdf"]);
  permission.value = false; run(c, "openAttachment('old.png')");
  events.get("wms.session.changed")(); assert.equal(run(c, "id.value"), "");
  permission.value = true; await settle(); assert.equal(calls, 1);
  assert.equal(run(c, "ready.value"), false); assert.equal(opened.length, 2);
  unmount(); run(c, "openAttachment('old.png'); load()"); assert.equal(calls, 1); assert.equal(events.size, 0);
 } finally { stops.forEach(stop => stop()); }
});

test("financial attachment resolution retains legacy paths and signed URLs; dates use Beijing calendar", () => {
 const previews = [], opens = [];
 class OtherTimezoneDate extends Date { getFullYear() { throw new Error("host calendar used"); } getMonth() { throw new Error("host calendar used"); } getDate() { throw new Error("host calendar used"); } }
 const c = vm.createContext({ Date: OtherTimezoneDate, fileBaseUrl: "https://files.test/images/", openWeb: url => { opens.push(url); return true; }, uni: { previewImage: value => previews.push(value.urls[0]), showToast: () => {} } });
 run(c, source("utils/material-image.ts")); run(c, source("services/customer-documents.ts"));
 run(c, "openDocumentAttachment('legacy.jpg'); openDocumentAttachment('acceptance/2026/10/a.webp'); openDocumentAttachment('acceptance/redemption/2026/10/b.pdf'); openDocumentAttachment('invoice/2026/10/c.ofd'); openDocumentAttachment(' https://legacy.test/a.PNG?signature=old ')");
 assert.deepEqual(previews, ["https://files.test/images/legacy.jpg", "https://files.test/images/acceptance/2026/10/a.webp", "https://legacy.test/a.PNG?signature=old"]);
 assert.deepEqual(opens, ["https://files.test/images/acceptance/redemption/2026/10/b.pdf", "https://files.test/images/invoice/2026/10/c.ofd"]);
 assert.equal(run(c, "documentAttachmentLabel('https://legacy.test/a.pdf?signature=old')"), "打开 PDF");
 const stamp = Date.UTC(2026, 9, 17, 16) / 1000;
 assert.equal(run(c, `documentDate(${stamp})`), "2026-10-18"); assert.equal(run(c, `documentDate(${stamp - 1})`), "2026-10-17"); assert.equal(run(c, "documentDate(0)"), "—");
});
