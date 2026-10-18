import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { after, test } from "node:test";
import { createApp, nextTick } from "vue";
import { createPinia, defineStore, disposePinia } from "pinia";
import piniaPluginPersistedstate from "pinia-plugin-persistedstate";
import { createServer } from "vite";

// Isolated storage: these tests never access a browser or an existing user session.
const entries = new Map();
const storage = {
  getItem: (key) => entries.get(key) ?? null,
  setItem: (key, value) => entries.set(key, String(value)),
};
const originalStorage = Object.getOwnPropertyDescriptor(
  globalThis,
  "localStorage",
);
Object.defineProperty(globalThis, "localStorage", {
  value: storage,
  configurable: true,
});

const server = await createServer({
  configFile: false,
  root: fileURLToPath(new URL("..", import.meta.url)),
  resolve: {
    alias: { "@": fileURLToPath(new URL("../src", import.meta.url)) },
  },
  server: { middlewareMode: true },
  optimizeDeps: { noDiscovery: true },
});
after(async () => {
  await server.close();
  if (originalStorage)
    Object.defineProperty(globalThis, "localStorage", originalStorage);
  else delete globalThis.localStorage;
});

const { default: appPinia } = await server.ssrLoadModule("/src/store/index.ts");
const { default: persistConfig } = await server.ssrLoadModule(
  "/src/config/piniaPersist.ts",
);
const { useUserStore } = await server.ssrLoadModule(
  "/src/store/modules/user.ts",
);
const { useGlobalStore } = await server.ssrLoadModule(
  "/src/store/modules/global.ts",
);
const { default: useLayoutStore } = await server.ssrLoadModule(
  "/src/store/modules/layout.ts",
);
const { default: useSettingStore } = await server.ssrLoadModule(
  "/src/store/modules/setting.ts",
);

function installPinia(pinia = createPinia().use(piniaPluginPersistedstate)) {
  createApp({}).use(pinia);
  return pinia;
}

const cases = [
  [
    "zs-user",
    useUserStore,
    {
      token: "legacy-token",
      account: { id: "fixture-account", name: "旧会话" },
    },
    { token: "updated-token" },
  ],
  [
    "geeker-global",
    useGlobalStore,
    { layout: "classic", primary: "#123456", isDark: true, tabs: false },
    { layout: "vertical", tabs: true },
  ],
  [
    "zs-layout",
    useLayoutStore,
    { collapse: true, refresh: false },
    { collapse: false, refresh: true },
  ],
  [
    "p_setting",
    useSettingStore,
    { theme: "dark", color: "#abcdef" },
    { theme: "", color: "#009688" },
  ],
];

for (const [key, useStore, legacy, update] of cases) {
  test(`${key}: restores legacy JSON, persists updates and survives a fresh store`, async () => {
    entries.clear();
    entries.set(key, JSON.stringify(legacy));
    entries.set("unrelated-session", "keep-me");
    // Exercise the actual application's plugin registration in the first case.
    const pinia = installPinia(key === "zs-user" ? appPinia : undefined);
    const store = useStore(pinia);
    for (const [name, value] of Object.entries(legacy))
      assert.deepEqual(store[name], value);
    store.$patch(update);
    await nextTick();
    const saved = JSON.parse(entries.get(key));
    assert.deepEqual(saved, JSON.parse(JSON.stringify(store.$state)));
    assert.deepEqual(
      Object.keys(saved).sort(),
      Object.keys(store.$state).sort(),
    );
    for (const [name, value] of Object.entries(update))
      assert.deepEqual(saved[name], value);
    assert.deepEqual(
      [...entries.keys()].sort(),
      [key, "unrelated-session"].sort(),
    );
    assert.equal(entries.get("unrelated-session"), "keep-me");
    disposePinia(pinia);

    const reloadedPinia = installPinia();
    assert.deepEqual(
      JSON.parse(JSON.stringify(useStore(reloadedPinia).$state)),
      saved,
    );
    disposePinia(reloadedPinia);
  });
}

for (const paths of [["token"], []]) {
  test(`selected paths ${JSON.stringify(paths)} retain their persistence contract`, async () => {
    entries.clear();
    const key = "fixture-selected-state";
    entries.set(
      key,
      JSON.stringify(paths.length ? { token: "legacy-token" } : {}),
    );
    const useStore = defineStore(key, {
      state: () => ({ token: "", transient: "default" }),
      persist: persistConfig(key, paths),
    });
    const pinia = installPinia();
    const store = useStore(pinia);
    assert.equal(store.token, paths.length ? "legacy-token" : "");
    assert.equal(store.transient, "default");
    store.$patch({ token: "updated-token", transient: "not-saved" });
    await nextTick();
    assert.deepEqual(
      JSON.parse(entries.get(key)),
      paths.length ? { token: "updated-token" } : {},
    );
    disposePinia(pinia);

    const reloadedPinia = installPinia();
    const reloaded = useStore(reloadedPinia);
    assert.equal(reloaded.token, paths.length ? "updated-token" : "");
    assert.equal(reloaded.transient, "default");
    disposePinia(reloadedPinia);
  });
}
