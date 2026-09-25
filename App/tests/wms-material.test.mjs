import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";
import { stripTypeScriptTypes } from "node:module";
import { parse as parseScript } from "@babel/parser";

function executable(path, expose, context = {}) {
	let source = fs.readFileSync(new URL("../" + path, import.meta.url), "utf8");
	if (path.endsWith(".uvue")) source = source.match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
	const ast = parseScript(source, { sourceType: "module", plugins: ["typescript"] });
	for (const item of ast.program.body.filter((item) => item.type === "ImportDeclaration").reverse()) {
		source = source.slice(0, item.start) + source.slice(item.end);
	}
	source = source.replaceAll("export ", "");
	const sandbox = vm.createContext(context);
	vm.runInContext(stripTypeScriptTypes(source) + `\nthis.subject = { ${expose} };`, sandbox);
	return sandbox.subject;
}

function deferred() {
	let resolve, reject;
	const promise = new Promise((ok, fail) => { resolve = ok; reject = fail; });
	return { promise, resolve, reject };
}

function pageContext(overrides = {}) {
	const events = [];
	return {
		ref: (value) => ({ value }), computed: (get) => ({ get value() { return get(); } }), watch: () => {},
		setTimeout, clearTimeout,
		useStore: () => ({ user: { hasCredential: () => true, canAccess: () => true, hasPermission: () => true } }),
		onLoad: () => {}, onShow: () => {}, onUnload: () => {}, onBackPress: () => {},
		errorMessage: (error, fallback) => error?.message || fallback,
		uni: { $on: () => {}, $off: () => {}, $emit: (name) => events.push(name), showToast: () => {}, showModal: () => {} },
		router: { back: () => {}, push: () => {}, params: () => ({}) },
		storage: { remove: () => {} }, events, ...overrides
	};
}

test("material price customer names retain Appx shortening rules and readable fallbacks", () => {
	const { materialPriceCustomerName: name } = executable("utils/wms-format.ts", "materialPriceCustomerName");
	assert.equal(name("山东省诸城市华东机械科技有限公司"), "华东");
	assert.equal(name("潍坊市诸城华南股份农业汽车部件装备制造工贸有限责任公司"), "华南");
	assert.equal(name("华东精密制造有限公司"), "华东精密");
	assert.equal(name("ABC Trading"), "ABC Trading");
	assert.equal(name("科技有限公司"), "科技有限公司");
	assert.equal(name("  "), "未指定客户");
	assert.equal(name(null), "未指定客户");
});

test("material services normalize null collections and use the WMS contracts", async () => {
	const calls = [];
	let reply = { total: 0, list: null };
	const service = executable("services/material.ts", "getMaterials, getMaterial, getMaterialCategories, updateMaterial", {
		parse: (value) => value, parseToObject: (value) => value,
		request: async (options) => { calls.push(options); return reply; }
	});
	assert.deepEqual(JSON.parse(JSON.stringify(await service.getMaterials({ page: 1, size: 20, model: "A" }))), { total: 0, list: [] });
	assert.equal(calls[0].url, "/material");
	assert.equal(calls[0].data.model, "A");
	reply = null;
	assert.equal((await service.getMaterialCategories()).length, 0);
	reply = { id: "material-a", prices: null };
	assert.equal((await service.getMaterial("material-a")).prices.length, 0);
	assert.equal(calls.at(-1).url, "/material/info");
	assert.equal(calls.at(-1).data.id, "material-a");
});

test("new material searches supersede in-flight results and page changes replace rows", async () => {
	const calls = [];
	const page = executable("pages/material/index.uvue", "load, search, items, total, keyword, loading, clearSession, currentPage", pageContext({
		getMaterials: (query) => { const call = { query, ...deferred() }; calls.push(call); return call.promise; }
	}));
	page.keyword.value = "old";
	page.search();
	page.keyword.value = "new";
	page.search();
	assert.equal(calls.length, 2);
	calls[1].resolve({ total: 40, list: Array.from({ length: 20 }, (_, i) => ({ id: `new-${i}` })) });
	await new Promise((resolve) => setImmediate(resolve));
	calls[0].resolve({ total: 1, list: [{ id: "stale" }] });
	await new Promise((resolve) => setImmediate(resolve));
	assert.equal(page.items.value[0].id, "new-0");
	assert.equal(page.total.value, 40);
	assert.equal(page.loading.value, false);
	const pending = page.load(2);
	assert.equal(calls.length, 3);
	assert.equal(calls[2].query.page, 2);
	assert.equal(calls[2].query.model, "new");
	calls[2].resolve({ total: 40, list: Array.from({ length: 20 }, (_, i) => ({ id: `next-${i}` })) });
	await pending;
	assert.equal(page.items.value.length, 20);
	assert.equal(page.currentPage.value, 2);
	assert.equal(page.items.value[0].id, "next-0");
});

test("failed page loading retries the same page and logout discards a late response", async () => {
	const calls = [];
	const page = executable("pages/material/index.uvue", "load, items, loadError, clearSession, retry", pageContext({
		getMaterials: (query) => { const call = { query, ...deferred() }; calls.push(call); return call.promise; }
	}));
	const first = page.load(1, true);
	calls[0].reject(new Error("network"));
	await first;
	assert.equal(page.loadError.value, "network");
	page.retry();
	assert.equal(calls[1].query.page, 1);
	page.clearSession();
	calls[1].resolve({ total: 1, list: [{ id: "another-user" }] });
	await new Promise((resolve) => setImmediate(resolve));
	assert.equal(page.items.value.length, 0);
});

test("typing invalidates an old response before debounce and blocks old-query pagination", async () => {
	let onInput;
	let fireTimer;
	const calls = [];
	const page = executable("pages/material/index.uvue", "load, items, keyword, changePage", pageContext({
		watch: (_ref, callback) => { onInput = callback; },
		setTimeout: (callback) => { fireTimer = callback; return 1; }, clearTimeout: () => {},
		getMaterials: (query) => { const call = { query, ...deferred() }; calls.push(call); return call.promise; }
	}));
	const old = page.load(1, true);
	page.keyword.value = "new model";
	onInput();
	// The old-query paginator is hidden while search debounce is pending.
	assert.equal(calls.length, 1);
	calls[0].resolve({ total: 1, list: [{ id: "stale" }] });
	await old;
	assert.equal(page.items.value.length, 0);
	fireTimer();
	assert.equal(calls.length, 2);
	assert.equal(calls[1].query.model, "new model");
	calls[1].resolve({ total: 1, list: [{ id: "latest" }] });
	await new Promise((resolve) => setImmediate(resolve));
	assert.equal(page.items.value[0].id, "latest");

});

test("returning to the applied search restarts a query interrupted by typing", async () => {
	let onInput;
	let timer = null;
	const calls = [];
	const page = executable("pages/material/index.uvue", "load, items, keyword, loading", pageContext({
		watch: (_ref, callback) => { onInput = callback; },
		setTimeout: (callback) => { timer = callback; return 1; }, clearTimeout: () => { timer = null; },
		getMaterials: (query) => { const call = { query, ...deferred() }; calls.push(call); return call.promise; }
	}));
	const initial = page.load(1, true);
	page.keyword.value = "temporary";
	onInput();
	page.keyword.value = "";
	onInput();
	assert.equal(timer, null);
	assert.equal(calls.length, 2);
	assert.equal(calls[1].query.model, "");
	assert.equal(page.loading.value, true);
	calls[0].resolve({ total: 1, list: [{ id: "stale" }] });
	await initial;
	assert.equal(page.items.value.length, 0);
	calls[1].resolve({ total: 1, list: [{ id: "current" }] });
	await new Promise((resolve) => setImmediate(resolve));
	assert.equal(page.items.value[0].id, "current");
	assert.equal(page.loading.value, false);
});

test("material list editing preserves the selected identity and checks current permissions", () => {
	const calls = [];
	let allowed = true;
	const page = executable("pages/material/index.uvue", "edit, canEdit", pageContext({
		useStore: () => ({ user: { hasPermission: () => allowed } }),
		router: { push: options => calls.push(JSON.parse(JSON.stringify(options))) }
	}));
	page.edit({ id: "selected-material" });
	assert.deepEqual(calls, [{ path: "/pages/material/edit", query: { id: "selected-material" } }]);
	page.edit({ id: "" });
	allowed = false;
	assert.equal(page.canEdit.value, false);
	page.edit({ id: "selected-material" });
	assert.equal(calls.length, 1);
});

test("outbound and plan material links retain the complete model and respect access", () => {
	for (const [path, field] of [["pages/outbound/detail.uvue", "model"], ["pages/plan/index.uvue", "material_model"]]) {
		const calls = [];
		let allowed = true;
		const page = executable(path, "openMaterial", pageContext({
			useStore: () => ({ user: { canAccess: () => allowed } }),
			router: { push: options => calls.push(JSON.parse(JSON.stringify(options))) }
		}));
		const model = "型号 X & A+B/50%#?=";
		page.openMaterial({ material_id: "m1", [field]: model });
		assert.deepEqual(calls, [{ path: "/pages/material/index", query: { model } }]);
		page.openMaterial({ material_id: "m1", [field]: "" });
		allowed = false;
		page.openMaterial({ material_id: "m1", [field]: model });
		assert.equal(calls.length, 1);
	}
});

test("list drawing saves suppress duplicates, recover from failure and ignore old-session results", async () => {
	const calls = [], messages = [];
	const context = pageContext({
		materialImageUrlValue: path => "https://example.invalid/" + path,
		saveMaterialImage: url => { const call = { url, ...deferred() }; calls.push(call); return call.promise; }
	});
	context.uni.showToast = options => messages.push(options.title);
	const page = executable("pages/material/index.uvue", "saveImage, savingImageId, clearSession", context);
	const item = { id: "m1", image: "drawing.png" };
	const first = page.saveImage(item);
	await page.saveImage(item);
	assert.equal(calls.length, 1);
	assert.equal(calls[0].url, "https://example.invalid/drawing.png");
	calls[0].reject(new Error("保存失败"));
	await first;
	assert.equal(page.savingImageId.value, "");
	assert.deepEqual(messages, ["保存失败"]);
	const retry = page.saveImage(item);
	calls[1].resolve("保存成功");
	await retry;
	assert.deepEqual(messages, ["保存失败", "保存成功"]);
	const stale = page.saveImage(item);
	page.clearSession();
	calls[2].resolve("旧结果");
	await stale;
	assert.equal(messages.length, 2);
	assert.equal(page.savingImageId.value, "");
});

test("material edit retains failed input, preserves the image and identity, and refreshes only after success", async () => {
	const calls = [];
	let fail = true;
	const context = pageContext({ updateMaterial: async (payload) => { calls.push(payload); if (fail) throw new Error("validation failed"); } });
	const page = executable("pages/material/edit.uvue", "form, categoryOptions, quantityInput, submit, saveError, saving, validate, quantityError", context);
	page.form.value = { id: "original-id", category_id: "enabled-category", image: "original-image.jpg", name: "Edited name", model: " A-01 ", material: "Steel", specification: "20x30", surface_treatment: "", strength_grade: "", quantity: 0, unit: "件", remark: "Keep this text" };
	page.categoryOptions.value = [{ label: "Enabled", value: "enabled-category" }];
	page.quantityInput.value = "2.5";
	await page.submit();
	assert.equal(page.form.value.remark, "Keep this text");
	assert.equal(page.saveError.value, "validation failed");
	assert.equal(page.saving.value, false);
	assert.equal(context.events.length, 0);
	assert.equal(calls[0].id, "original-id");
	assert.equal(calls[0].image, "original-image.jpg");
	assert.equal(calls[0].quantity, 2.5);
	assert.equal(calls[0].model, "A-01");
	assert.equal(Object.hasOwn(calls[0], "price"), false);
	fail = false;
	await page.submit();
	assert.deepEqual(context.events, ["wms.material.changed"]);
	page.quantityInput.value = "-1";
	assert.equal(page.validate(), false);
	page.quantityInput.value = "";
	assert.equal(page.validate(), false);
	page.quantityInput.value = "NaN";
	assert.equal(page.validate(), false);
});
