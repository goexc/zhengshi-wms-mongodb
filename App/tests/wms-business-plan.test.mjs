import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";
import { stripTypeScriptTypes } from "node:module";
import { computed, ref, watch } from "vue";
import { stripImports } from "./helpers/source.mjs";

const root = new URL("../", import.meta.url);
function source(path, page = false) {
	let text = fs.readFileSync(new URL(path, root), "utf8");
	if (page) text = text.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
	return stripTypeScriptTypes(stripImports(text).replace(/^export /gm, ""));
}
function evaluate(context, expression) { return vm.runInContext(expression, context); }
function plain(value) { return JSON.parse(JSON.stringify(value)); }

test("partner and ledger services preserve the real WMS data and required query", async () => {
	const calls = [];
	let response = { total: 0, list: null };
	const context = vm.createContext({ parse: value => value, parseToObject: value => value, request: async options => { calls.push(plain(options)); return response; } });
	vm.runInContext(source("services/business.ts"), context);
	const empty = await evaluate(context, "fetchPartners('supplier', {page:1,size:20,name:'钢材',code:'',manager:'',contact:'',email:'',level:3})");
	assert.deepEqual(plain(empty), { total: 0, list: [] });
	assert.equal(calls[0].url, "/supplier");
	assert.equal(calls[0].data.level, 3);
	response = { total: 1, list: [{ id: "c1", receivable_balance: 28.125, credit_balance: 0 }] };
	const customer = await evaluate(context, "fetchPartners('customer', {page:1,size:20,name:'',code:'',manager:'',contact:'',email:''})");
	assert.equal(customer.list[0].receivable_balance, 28.125);
	await evaluate(context, "fetchCustomerTransactions('0123456789abcdef01234567', 2)");
	assert.deepEqual(calls[2], { url: "/customer/transaction", method: "GET", data: { customer_id: "0123456789abcdef01234567", page: 2, size: 20 } });
	await assert.rejects(evaluate(context, "fetchCustomerTransactions('', 1)"), /客户信息不完整/);
	assert.equal(calls.length, 3);
});

test("ledger directions and statuses do not guess adjustment signs or count voided records", () => {
	const context = vm.createContext({});
	vm.runInContext(source("services/business.ts"), context);
	assert.equal(evaluate(context, "transactionDirection({direction:'receivable_decrease', transaction_type:'ar_adjustment'})"), "应收减少");
	assert.equal(evaluate(context, "transactionDirection({direction:'', transaction_type:'ar_adjustment', type:'应收调整'})"), "方向未记录");
	assert.equal(evaluate(context, "transactionDirection({direction:'', transaction_type:'', type:'回款'})"), "应收减少");
	assert.equal(evaluate(context, "transactionDirection({direction:'future_direction', transaction_type:'payment'})"), "方向未识别");
	assert.match(evaluate(context, "transactionStatus('voided')"), /不计入余额/);
	assert.match(evaluate(context, "transactionStatus('reversed')"), /不计入余额/);
});

function pageHarness(path) {
	const pending = [];
	const cleanup = [];
	const allowed = ref(true);
	const request = (...args) => new Promise((resolve, reject) => pending.push({ args: plain(args), resolve, reject }));
	const context = vm.createContext({
		ref, computed, watch: (value, fn) => { const stop = watch(value, fn, { flush: "sync" }); cleanup.push(stop); return stop; },
		defineProps: () => ({ kind: "customer" }), defineOptions: () => {}, useStore: () => ({ user: { canAccess: () => allowed.value } }),
		fetchPartners: request, fetchCustomerTransactions: request, fetchPlans: request,
		onMounted: () => {}, onUnmounted: fn => cleanup.push(fn), onLoad: () => {}, onUnload: fn => cleanup.push(fn),
		setTimeout, clearTimeout, uni: { $on: () => {}, $off: () => {} }, router: {},
		errorMessage: (error, fallback) => error?.message || fallback
	});
	vm.runInContext(source(path, true), context);
	return { context, pending, allowed, cleanup: () => cleanup.forEach(fn => fn()) };
}

test("partner search rejects stale responses and page requests use applied filters", async () => {
	const h = pageHarness("pages/business/components/PartnerList.uvue");
	try {
		const older = evaluate(h.context, "load(1, true)");
		evaluate(h.context, "code.value = 'NEW'");
		const newer = evaluate(h.context, "load(1, true)");
		h.pending[1].resolve({ total: 21, list: Array.from({ length: 20 }, (_, i) => ({ id: `n${i}` })) });
		await newer;
		h.pending[0].resolve({ total: 1, list: [{ id: "old" }] });
		await older;
		assert.equal(evaluate(h.context, "items.value[0].id"), "n0");
		evaluate(h.context, "code.value = 'UNAPPLIED'");
		const more = evaluate(h.context, "load(2)");
		assert.equal(h.pending[2].args[1].code, "NEW");
		assert.equal(h.pending[2].args[1].page, 2);
		h.pending[2].reject(new Error("offline"));
		await more;
		assert.equal(evaluate(h.context, "currentPage.value"), 2);
		const retry = evaluate(h.context, "load(2)");
		assert.equal(h.pending[3].args[1].page, 2);
		h.pending[3].resolve({ total: 21, list: [{ id: "n20" }] });
		await retry;
		assert.equal(evaluate(h.context, "currentPage.value"), 2);
		assert.equal(evaluate(h.context, "items.value.length"), 1);
	} finally { h.cleanup(); }
});

test("clearing a partner session removes cached financial records and invalidates requests", async () => {
	const h = pageHarness("pages/business/components/PartnerList.uvue");
	try {
		const loading = evaluate(h.context, "load(1, true)");
		h.allowed.value = false;
		evaluate(h.context, "clearSession()");
		h.pending[0].resolve({ total: 1, list: [{ id: "secret", receivable_balance: 500 }] });
		await loading;
		assert.equal(evaluate(h.context, "items.value.length"), 0);
		assert.equal(evaluate(h.context, "total.value"), 0);
	} finally { h.cleanup(); }
});

test("ledger preserves distinct records with equal amount and time; plan uses Unix seconds", async () => {
	const ledger = pageHarness("pages/business/customer-transactions.uvue");
	const plan = pageHarness("pages/plan/index.uvue");
	try {
		const loading = evaluate(ledger.context, "load(1, true)");
		ledger.pending[0].resolve({ total: 2, list: [{ time: 100, amount: 10 }, { time: 100, amount: 10 }] });
		await loading;
		assert.equal(evaluate(ledger.context, "items.value.length"), 2);
		assert.equal(evaluate(plan.context, "overdue({status:'执行中',deadline:Math.floor(Date.now()/1000)+3600})"), false);
		assert.equal(evaluate(plan.context, "overdue({status:'执行中',deadline:Math.floor(Date.now()/1000)-3600})"), true);
		assert.equal(evaluate(plan.context, "overdue({status:'已完成',deadline:1})"), false);
	} finally { ledger.cleanup(); plan.cleanup(); }
});
