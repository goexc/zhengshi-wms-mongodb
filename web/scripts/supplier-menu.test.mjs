import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";
import { transformSync } from "esbuild";

function loadTypeScript(relativePath, dependencies = {}) {
  const source = readFileSync(new URL(relativePath, import.meta.url), "utf8");
  const { code } = transformSync(source, {
    loader: "ts", format: "cjs", target: "es2020",
  });
  const module = { exports: {} };
  const evaluate = vm.runInThisContext(`(function (module, exports, require) { ${code}\n})`);
  evaluate(module, module.exports, (name) => {
    assert.ok(name in dependencies, `Unexpected dependency: ${name}`);
    return dependencies[name];
  });
  return module.exports;
}

const menuUtils = loadTypeScript("../src/utils/menu.ts");
const { filterRetiredSupplierMenus, getFlatMenuList } = menuUtils;

function menu(path, overrides = {}) {
  return {
    id: path,
    parent_id: "",
    type: 1,
    sort_id: 0,
    path,
    name: path,
    component: "",
    meta: { hidden: false, perms: "" },
    ...overrides,
  };
}

test("removes old supplier routes and aliases pointing to retired components", () => {
  const retired = [
    menu("/supplier"),
    menu("/supplier/"),
    menu("/supplier/detail/123"),
    menu("/legacy", { component: "/supplier/index" }),
    menu("/legacy-component", { component: "supplier/index.vue" }),
    menu(" /supplier/index "),
  ];
  assert.deepEqual(filterRetiredSupplierMenus(retired), []);
});

test("keeps the partner supplier subtree and other supplier-related paths", () => {
  const retained = [
    menu("/business_partner", {
      children: [
        menu("/business_partner/supplier", {
          component: "/business_partner/supplier/index",
          children: [menu("/business_partner/supplier/detail")],
        }),
        menu("/business_partner/customer"),
        menu("supplier", { component: "/business_partner/supplier/index" }),
      ],
    }),
    menu("/supplier-report"),
    menu("/suppliers"),
    menu("/other/supplier"),
    menu("supplier/index"),
    menu("/materials", { component: "/material/supplier/index" }),
    menu("/renamed", { name: "供应商管理", component: "/business_partner/supplier/index" }),
  ];
  assert.deepEqual(filterRetiredSupplierMenus(retained), retained);
});

test("recursively removes retired branches without mutating the response", () => {
  const input = [
    menu("/workspace", {
      children: [
        menu("/supplier", { children: [menu("detail")] }),
        menu("/group", {
          children: [
            menu("/alias", { component: "/supplier/index" }),
            menu("/business_partner/supplier", { sort_id: 2 }),
            menu("/business_partner/customer", { sort_id: 1 }),
          ],
        }),
      ],
    }),
  ];
  const original = structuredClone(input);
  const result = filterRetiredSupplierMenus(input);
  const group = result[0].children[0];
  assert.equal(group.path, "/group");
  assert.deepEqual(group.children.map((item) => item.path), [
    "/business_partner/supplier",
    "/business_partner/customer",
  ]);
  group.children.sort((a, b) => a.sort_id - b.sort_id);
  assert.deepEqual(input, original);
  assert.deepEqual(getFlatMenuList(result).map((item) => item.path), [
    "/workspace",
    "/group",
    "/business_partner/customer",
    "/business_partner/supplier",
  ]);
});

test("handles empty menu lists and groups whose old child was removed", () => {
  assert.deepEqual(filterRetiredSupplierMenus([]), []);
  assert.deepEqual(filterRetiredSupplierMenus([
    menu("/group", { children: [menu("/supplier")] }),
  ]), [menu("/group", { children: [] })]);
});

test("auth loading excludes retired routes, preserves sorting and every button permission", async () => {
  const response = {
    menus: [
      menu("/supplier", { sort_id: 0 }),
      menu("/business_partner", {
        sort_id: 2,
        children: [
          menu("/business_partner/supplier", { sort_id: 2 }),
          menu("/alias", { component: "/supplier/index", sort_id: 0 }),
          menu("/business_partner/customer", { sort_id: 1 }),
        ],
      }),
      menu("/home", { sort_id: 1 }),
    ],
    buttons: [
      { name: "查询", perms: "business_partner:supplier:list" },
      { name: "编辑", perms: "business_partner:supplier:edit" },
      { name: "状态", perms: "business_partner:supplier:status" },
      { name: "删除", perms: "business_partner:supplier:delete" },
    ],
  };
  const original = structuredClone(response);
  const { useAuthStore: storeDefinition } = loadTypeScript("../src/store/modules/auth.ts", {
    pinia: { defineStore: (_id, definition) => definition },
    "@/api/modules/login": { getAuthMenuListApi: async () => ({ data: response }) },
    "@/utils/menu.ts": menuUtils,
  });
  const state = storeDefinition.state();
  await storeDefinition.actions.getAuthMenuList.call(state);
  assert.deepEqual(state.authMenuList.map((item) => item.path), ["/home", "/business_partner"]);
  assert.deepEqual(state.authMenuList[1].children.map((item) => item.path), [
    "/business_partner/customer",
    "/business_partner/supplier",
  ]);
  assert.strictEqual(state.authButtonList, response.buttons);
  assert.deepEqual(response, original);
  assert.deepEqual(storeDefinition.getters.flatMenuListGet(state).map((item) => item.path), [
    "/home",
    "/business_partner",
    "/business_partner/customer",
    "/business_partner/supplier",
  ]);
});
