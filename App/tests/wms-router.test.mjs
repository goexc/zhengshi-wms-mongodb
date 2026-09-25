import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";
import { stripTypeScriptTypes } from "node:module";
import { parse as parseScript } from "@babel/parser";

function executable(path, expose, context = {}) {
	let source = fs.readFileSync(new URL("../" + path, import.meta.url), "utf8");
	const ast = parseScript(source, { sourceType: "module", plugins: ["typescript"] });
	for (const item of ast.program.body.filter((item) => item.type === "ImportDeclaration").reverse()) {
		source = source.slice(0, item.start) + source.slice(item.end);
	}
	const sandbox = vm.createContext(context);
	vm.runInContext(stripTypeScriptTypes(source.replaceAll("export ", "")) + `\nthis.subject = { ${expose} };`, sandbox);
	return sandbox.subject;
}

function harness() {
	const config = JSON.parse(fs.readFileSync(new URL("../pages.json", import.meta.url), "utf8").replace(/^\s*\/\/.*$/gm, ""));
	const values = new Map();
	const navigations = [];
	const uni = {
		// uni.getStorageSync returns an empty string for an absent key, not null.
		getStorageSync: (key) => values.has(key) ? values.get(key) : "",
		setStorageSync: (key, value) => values.set(key, value),
		removeStorageSync: (key) => values.delete(key),
		switchTab: (options) => navigations.push({ mode: "switchTab", ...options }),
		navigateTo: (options) => navigations.push({ mode: "navigateTo", ...options })
	};
	const { storage } = executable(".cool/utils/storage.ts", "storage", { uni });
	const utils = executable(".cool/utils/comm.ts", "isObject, isNull, isEmpty, debounce", {
		UTSJSONObject: { keys: Object.keys }, setTimeout, clearTimeout
	});
	const { router } = executable(".cool/router/index.ts", "router", {
		...utils, storage, uni, TABS: config.tabBar.list.map((item) => ({ pagePath: "/" + item.pagePath }))
	});
	return { router, storage, values, navigations };
}

function assertEmptyParams(params) {
	assert.equal(typeof params, "object");
	assert.notEqual(params, null);
	assert.equal(Array.isArray(params), false);
	assert.deepEqual(Object.keys(params), []);
}

test("first tab navigation without params returns an object for the absent storage key", () => {
	const { router, storage, navigations } = harness();
	router.push({ path: "/pages/index/home" });
	assert.equal(navigations[0].mode, "switchTab");
	assert.equal(storage.get("router-params"), "");
	assertEmptyParams(router.params());
});

test("material and outbound are ordinary pages while home and profile remain tabs", () => {
	const { router, navigations } = harness();
	for (const path of ["/pages/material/index", "/pages/outbound/order"]) {
		router.to(path);
		assert.equal(navigations.at(-1).mode, "navigateTo");
		assert.equal(navigations.at(-1).url, path);
	}
	router.to("/pages/index/my");
	assert.equal(navigations.at(-1).mode, "switchTab");
});

test("non-object stored params fall back to an empty object", () => {
	const { router, storage } = harness();
	for (const value of ["", "stale", '{"materialModel":"A"}', 0, 123, false, true, [], ["A"], null, undefined]) {
		storage.set("router-params", value, 0);
		assertEmptyParams(router.params());
	}
});

test("router push preserves material and nested outbound params", () => {
	const { router, values } = harness();
	const cases = [
		{ path: "/pages/material/index", params: { materialModel: "M & 中文" } },
		{ path: "/pages/outbound/detail", params: { outboundOrder: { code: "OUT-001", items: [{ quantity: 2 }], remark: null } } }
	];
	for (const options of cases) {
		router.push(options);
		assert.equal(router.params(), options.params);
		assert.equal(values.has("router-params"), true, "reading must not consume the params");
	}
});

test("re-entering a page after it consumes params returns an empty object", () => {
	const { router, storage } = harness();
	router.push({ path: "/pages/material/index", params: { materialModel: "M-01" } });
	assert.equal(router.params().materialModel, "M-01");
	storage.remove("router-params");
	assert.equal(storage.get("router-params"), "");
	assertEmptyParams(router.params());
	assertEmptyParams(router.params());
});

test("invalid route params do not clear unrelated storage or change empty-string storage semantics", () => {
	const { router, storage, values } = harness();
	storage.set("wms.auth.token", "test-token", 0);
	storage.set("wms.largeText", true, 0);
	storage.set("empty-setting", "", 0);
	storage.set("router-params", "invalid", 0);
	const before = [...values];
	assertEmptyParams(router.params());
	assert.deepEqual([...values], before);
	assert.equal(storage.get("empty-setting"), "");
});
