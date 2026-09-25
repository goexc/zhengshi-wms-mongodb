import test from "node:test";
import assert from "node:assert/strict";
import fs from "node:fs";
import vm from "node:vm";
import { stripTypeScriptTypes } from "node:module";
import { computed, ref, reactive, watch, nextTick } from "vue";
import { stripImports } from "./helpers/source.mjs";

const read = (path) => fs.readFileSync(new URL("../" + path, import.meta.url), "utf8");
const clPagePath = "uni_modules/cool-ui/components/cl-page/cl-page.uvue";
function platformSource(source, platform) {
	const enabled = [true];
	return source.split(/\r?\n/).filter((line) => {
		const directive = line.match(/^\s*\/\/ #(ifdef|ifndef|endif)\s*(\w+)?/);
		if (!directive) return enabled.at(-1);
		if (directive[1] === "endif") enabled.pop();
		else {
			const matches = directive[2] === platform;
			enabled.push(enabled.at(-1) && (directive[1] === "ifdef" ? matches : !matches));
		}
		return false;
	}).join("\n");
}
function component(path, platform, props, dependencies = {}) {
	let source = read(path).match(/<script setup lang="ts">([\s\S]*?)<\/script>/)[1];
	source = stripTypeScriptTypes(stripImports(platformSource(source, platform)));
	const events = [], hooks = {}, stops = [];
	const context = vm.createContext({
		ref, computed, nextTick, largeText: ref(false),
		watch: (...args) => { const stop = watch(...args); stops.push(stop); return stop; },
		defineOptions: () => {}, defineExpose: () => {},
		defineProps: (defaults) => {
			for (const [key, item] of Object.entries(defaults)) if (!(key in props)) props[key] = item.default;
			return props;
		},
		defineEmits: () => (name) => events.push(name),
		onMounted: () => {}, onUnmounted: () => {}, onResize: () => {},
		config: { backTop: false }, scroller: {
			on: () => {}, emit: () => {},
			onEvent: (name, fn) => { hooks[name] = fn; return () => { delete hooks[name]; }; }
		},
		...dependencies
	});
	vm.runInContext(source + "\nthis.subject = { refresh, reachBottom };", context);
	return { ...context.subject, events, hooks, stop: () => stops.forEach((stop) => stop()) };
}

test("native page forwards refresh and pagination once, without page-level event hooks", () => {
	const props = reactive({ refresherEnabled: true, refresherTriggered: false });
	const page = component(clPagePath, "APP", props);
	page.refresh();
	page.reachBottom();
	assert.deepEqual(page.events, ["refresherrefresh", "scrolltolower"]);
	assert.deepEqual(page.hooks, {});
	props.refresherTriggered = true;
	page.reachBottom();
	assert.equal(page.events.length, 2);
	props.refresherEnabled = false;
	page.refresh();
	assert.equal(page.events.length, 2);
	page.stop();
});

test("Web page bridges page-level scrolling and stops pull refresh on completion or denial", async () => {
	let stopped = 0;
	const props = reactive({ refresherEnabled: true, refresherTriggered: false });
	const page = component(clPagePath, "H5", props, { uni: { stopPullDownRefresh: () => stopped++ } });
	page.hooks.refresh();
	props.refresherTriggered = true;
	await nextTick();
	page.hooks.reachBottom();
	assert.deepEqual(page.events, ["refresherrefresh"]);
	props.refresherTriggered = false;
	await nextTick();
	assert.equal(stopped, 1);
	page.hooks.reachBottom();
	assert.deepEqual(page.events, ["refresherrefresh", "scrolltolower"]);
	props.refresherEnabled = false;
	page.hooks.refresh();
	assert.equal(stopped, 2);
	assert.equal(page.events.length, 2);
	page.stop();
});

test("shared page only forwards list events when the current session has access", () => {
	const allowed = ref(true);
	const user = { revision: ref(0), token: "test", canAccess: () => allowed.value };
	const props = reactive({ active: "", refresherEnabled: true });
	const page = component("components/wms/Page.uvue", "APP", props, { useStore: () => ({ user }), router: { path: () => "/pages/material/index" } });
	page.refresh(); page.reachBottom();
	assert.deepEqual(page.events, ["refresherrefresh", "scrolltolower"]);
	allowed.value = false; user.revision.value++;
	page.refresh(); page.reachBottom();
	assert.equal(page.events.length, 2);
	user.token = null; user.revision.value++;
	page.refresh(); page.reachBottom();
	assert.equal(page.events.length, 2);
	page.stop();
});

test("all routes allow Web document scrolling and keep the main scroll owner in cl-page", () => {
	const config = JSON.parse(read("pages.json").replace(/^\s*\/\/.*$/gm, ""));
	const frame = read("components/wms/Page.uvue").split("<script")[0];
	assert.match(frame, /<cl-page\b/);
	assert.match(frame, /<cl-sticky\b/);
	assert.match(frame, /<cl-topbar\b/);
	assert.doesNotMatch(frame, /<scroll-view\b|shell-body|height \+ 'px'/);
	for (const route of config.pages) {
		assert.notEqual(route.style.disableScroll, true, route.path);
		const template = read(route.path + ".uvue").split("<script")[0];
		assert.match(template, /<(WmsPage|PartnerList)\b/, route.path);
		assert.doesNotMatch(template, /<scroll-view\b/, route.path);
	}
	assert.doesNotMatch(read("pages/business/components/PartnerList.uvue").split("<script")[0], /<scroll-view\b/);
	for (const path of ["pages/material/index", "pages/outbound/order", "pages/business/customer", "pages/business/supplier", "pages/business/customer-transactions", "pages/plan/index"]) {
		assert.equal(config.pages.find((route) => route.path === path).style.enablePullDownRefresh, true, path);
	}
});

test("page events reach only the active route and cleanup retains other page listeners", () => {
	let current = "/pages/material/index";
	const context = vm.createContext({ router: { path: () => current } });
	const source = stripTypeScriptTypes(stripImports(read(".cool/scroller/index.ts")).replaceAll("export ", ""));
	vm.runInContext(source + "\nthis.subject = scroller;", context);
	const calls = [];
	const scroller = context.subject;
	const offMaterial = scroller.onEvent("reachBottom", () => calls.push("material"));
	current = "/pages/plan/index";
	const offPlan = scroller.onEvent("reachBottom", () => calls.push("plan"));
	scroller.emitEvent("reachBottom");
	assert.deepEqual(calls, ["plan"]);
	offMaterial();
	scroller.emitEvent("reachBottom");
	assert.deepEqual(calls, ["plan", "plan"]);
	current = "/pages/material/index";
	scroller.emitEvent("reachBottom");
	assert.equal(calls.length, 2);
	offPlan();
});
