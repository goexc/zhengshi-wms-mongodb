import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";
import { transformSync } from "esbuild";
import * as vue from "vue";
import { compileScript, parse } from "vue/compiler-sfc";

const source = path => readFileSync(new URL(path, import.meta.url), "utf8");
function evaluate(text, dependencies = {}) {
  const { code } = transformSync(text, { loader: "ts", format: "cjs", target: "es2020" });
  const module = { exports: {} };
  vm.runInThisContext(`(function(module,exports,require){${code}\n})`)(module, module.exports, name => {
    assert.ok(name in dependencies, `Unexpected dependency ${name}`);
    return dependencies[name];
  });
  return module.exports;
}
const menuUtils = evaluate(source("../src/utils/menu.ts"));
const rules = evaluate(source("../src/views/acl/menu/rules.ts"), { vue });
const fixture = (overrides = {}) => ({ id: "finance-id", parent_id: "root", name: "Finance", path: "/finance", type: 1, sort_id: 60, component: "", remark: "", meta: { title: "客户财务", icon: "Wallet", hidden: false, fixed: true, is_full: false, transition: "", perms: "finance" }, ...overrides });
function mountManagement() {
  const writes = [];
  const api = { reqMenuList: async () => ({ code: 200, data: [] }), reqAddOrUpdateMenu: async form => { writes.push(JSON.parse(JSON.stringify(form))); return { code: 200, msg: "成功" }; }, reqRemoveMenu: async () => ({ code: 200 }) };
  const path = "../src/views/acl/menu/index.vue";
  const { descriptor } = parse(source(path), { filename: path });
  const script = compileScript(descriptor, { id: path });
  const component = evaluate(script.content, { vue, "@/api/acl/menu": api, "@/utils/time.ts": { TimeFormat: String }, "@/utils/enum.ts": { Sizes: {}, Types: {} }, "@element-plus/icons-vue": {}, "element-plus": { ElMessage: { success() {}, error() {} } }, "./rules": rules, "@/utils/menu": menuUtils }).default;
  component.render = () => null;
  const renderer = vue.createRenderer({ createElement: () => ({}), createText: () => ({}), createComment: () => ({}), insert() {}, remove() {}, setText() {}, setElementText() {}, patchProp() {}, parentNode: () => null, nextSibling: () => null });
  const app = renderer.createApp(component); const instance = app.mount({});
  const state = instance.$.setupState;
  state.formRef = { validate: async () => true, clearValidate() {} };
  return { state, writes, unmount: () => app.unmount() };
}

test("editing a display title sends it separately while cancellation leaves the original menu unchanged", async () => {
  const view = mountManagement();
  try {
    const row = fixture();
    view.state.editMenu(row); view.state.menuForm.meta.title = "客户票据管理";
    assert.equal(row.meta.title, "客户财务"); assert.equal(view.state.menuForm.name, "Finance");
    view.state.handleClose(); assert.equal(row.meta.title, "客户财务"); assert.equal(view.writes.length, 0);
    view.state.editMenu(row); view.state.menuForm.meta.title = "客户票据管理";
    await view.state.handleSubmit();
    assert.equal(view.writes[0].meta.title, "客户票据管理"); assert.equal(view.writes[0].name, "Finance"); assert.equal(view.writes[0].id, "finance-id");
  } finally { view.unmount(); }
});

test("legacy Chinese names and known financial routes prefill display titles without changing route names", () => {
  const view = mountManagement();
  try {
    const chinese = fixture({ name: "合作伙伴", path: "/business_partner", meta: { title: "", icon: "User" } });
    view.state.editMenu(chinese); assert.equal(view.state.menuForm.meta.title, "合作伙伴"); assert.equal(view.state.menuForm.name, "合作伙伴");
    const finance = fixture({ meta: { title: "", icon: "Wallet" } });
    view.state.editMenu(finance); assert.equal(view.state.menuForm.meta.title, "客户财务"); assert.equal(view.state.menuForm.name, "Finance");
  } finally { view.unmount(); }
});

test("new menu submits the display title and keeps the administrator's route name editable", async () => {
  const view = mountManagement();
  try {
    view.state.addMenu(fixture());
    Object.assign(view.state.menuForm, { name: "InvoiceReview", path: "/invoice-review" });
    view.state.menuForm.meta.title = "发票审核";
    await view.state.handleSubmit();
    assert.equal(view.writes[0].id, ""); assert.equal(view.writes[0].parent_id, "finance-id"); assert.equal(view.writes[0].name, "InvoiceReview"); assert.equal(view.writes[0].meta.title, "发票审核");
    view.state.editMenu(fixture()); view.state.menuForm.name = "FinanceAdmin"; view.state.menuForm.meta.title = "客户财务";
    await view.state.handleSubmit(); assert.equal(view.writes[1].name, "FinanceAdmin"); assert.equal(view.writes[1].meta.title, "客户财务");
  } finally { view.unmount(); }
});
