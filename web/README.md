# Web 开发与构建

本工程使用 Vue 3、Element Plus、TypeScript 和 Vite。以下命令均在 `web` 目录执行。

## 安装依赖

使用 Node.js **22.22.2 及以上的 22.x、24.15.0 及以上的 24.x，或 26 及以上版本**，满足新版 npm-check-updates 的要求。统一使用 `package.json` 声明的 **pnpm 12.14.0**，以 `pnpm-lock.yaml` 为唯一依赖锁文件。

```sh
npm install --global pnpm@12.14.0
pnpm --version
pnpm install --frozen-lockfile
```

常规安装不要使用 `--no-frozen-lockfile` 或 `--ignore-scripts`。修改依赖时使用 pnpm 并一并提交 `package.json` 和 `pnpm-lock.yaml`。

工程默认编译器使用微软官方 **TypeScript 7.0.2**，`pnpm exec tsc --version` 应输出 `Version 7.0.2`。Vue 与 ESLint 需要的经典编译器 API 隔离在 `tooling/vue-typescript` 工作区中，通过 **typescript-native-bridge 6.0.3-bridge.18.tsgo.7.0.2** 接入原生 TypeScript 7 检查器；其中 vue-tsc 为 **3.3.13**。该适配器由 Volar 作者维护，并非微软官方 TypeScript 包；版本中的 `6.0.3` 表示兼容 API 外观，实际类型检查引擎为 `tsgo 7.0.2`。详见[适配器说明](https://github.com/johnsoncodehk/typescript-native-bridge)。

`pnpm run typecheck` 先运行 `vue-typecheck`，检查全部 TS 源码、Vue 脚本和模板，再用官方 `tsc` 检查 Vite 配置。`TNB ACTIVE` 是适配器确认原生引擎已启用的信息。应用与 Vite 配置仍使用 `Bundler` 模块解析，保留严格类型检查和未使用变量检查；没有关闭模板检查或放宽 peer 依赖要求。

两个工作区统一使用根锁文件，安装时须保留 `tooling/vue-typescript` 和 `pnpm-workspace.yaml`。适配器提供 Windows x64/arm64、macOS 和 glibc Linux 的预编译组件；Linux 要求 glibc 2.35 以上，Alpine/musl 不受支持。当前 CLI 已在 Windows x64 验证。编辑器不会自动跟随 CLI 切换 SDK；如需 Vue 编辑器使用该适配器，应在支持选择工作区 TypeScript SDK 的编辑器中选择 `tooling/vue-typescript/node_modules/typescript/lib`（以 `web` 为工作区根目录），并重新加载 Vue 语言服务。编辑器配置与提示效果未作自动化验收。

Vite 固定为 **8.3.4**，配套 `@vitejs/plugin-vue` **6.0.10**、`@vitejs/plugin-vue-jsx` **5.1.6**。配置使用 `vite.config.mts`，通过 ESM 加载 Vite；Vite 8 默认使用 Sass modern API，全局变量通过 `@use "@/styles/variable.scss" as *` 注入，新增样式依赖也应使用 `@use`。

pnpm 12 的安装设置位于 `pnpm-workspace.yaml`，通过 `allowBuilds` 只允许 `esbuild`、`vue-demi`、`@parcel/watcher` 执行依赖安装脚本。文件中的发布年龄例外由 pnpm 在此次安装时生成，并限定到具体包版本。旧的 VueUse PURE 注释补丁已随依赖更新移除。

如果从旧安装目录切换过来且这些脚本曾被跳过，可运行：

```sh
pnpm rebuild esbuild vue-demi @parcel/watcher
```

## 常用命令

```sh
pnpm run dev         # 启动开发服务器
pnpm run typecheck   # 检查全部 Vue/TS 源码及 Vite 配置，不生成 JS
pnpm run build       # 类型检查通过后打包
pnpm run build:prod  # 类型检查通过后按 production 模式打包
pnpm run build:test  # 类型检查通过后按 test 模式打包
pnpm run preview    # 预览已构建的 dist
pnpm test           # 持久化、菜单及 TS7/Vue 类型检查正反例回归
pnpm run deps:check # 检查符合当前兼容策略的依赖更新
pnpm run deps:update # 更新 package.json，之后必须安装并验证
```

所有构建入口都会先执行 `typecheck`，类型检查失败时停止构建。构建产物位于 `dist`，Vite 配置的增量类型检查缓存位于 `node_modules/.cache`。

生产构建使用 Vite 8 的 `rolldownOptions.output.codeSplitting`，将 ExcelJS 单独放入 `vendor-excel`，随出库报表页面加载；其余第三方依赖和共享运行时放入 `vendor`。保留 1300 kB 资源体积警告阈值和原有构建目标（ES2020、Edge 88、Firefox 78、Chrome 87、Safari 14）。调整分包时需验证登录、动态路由、极速出库时间联动和报表的两种 Excel 导出，并检查入口没有预加载 `vendor-excel`。

模式对应的环境变量由 `.env` / `.env.<mode>` 加载。当前目录提供 development、production 配置；使用 test 模式时按实际环境提供相应配置。

构建通过表示静态类型检查与资源编译通过，不代替真实出入库业务验收。

## 依赖升级与兼容性

使用本地开发依赖 `npm-check-updates`，配置见 `.ncurc.cjs`：递归检查根工程和隔离工具包的最新版本，已移除 TypeScript 6.0 限制。官方 TypeScript 与适配器的 tsgo 基础版本必须一起更新，`pnpm test` 会校验版本一致性，并验证脚本、模板、组件 props、默认 v-model、空值和未使用变量的错误仍可检出。升级流程：

```sh
pnpm run deps:check
pnpm run deps:update
pnpm install --no-frozen-lockfile
pnpm peers check
pnpm test
pnpm run build
```

升级后应一并检查配置文件、锁文件和实际页面；不能仅以依赖安装成功作为兼容结论。

本次主要版本为 Vue 3.5.44、Vue Router 5.4.0、Pinia 4.1.0、Element Plus 2.14.7、ECharts 6.1.0、ESLint 10.13.0 和 Stylelint 17.16.0。

- 原 `pinia-plugin-persist` 仅兼容旧 Pinia，已迁移到 `pinia-plugin-persistedstate`。保留 `zs-user`、`geeker-global`、`zs-layout`、`p_setting` 的 localStorage key、JSON 格式和状态字段，已有会话及设置可以恢复；`paths` 对应新插件的 `pick`。
- Vue Router 5 导航守卫使用返回值代替已弃用的 `next()`，保留登录跳转、白名单和动态菜单逻辑。
- 移除未使用且仅支持 ECharts 5 的 `echarts-liquidfill`、过时的 ECharts 类型包及 `any` 模块声明，使用 ECharts 自带类型。
- 移除 Vite 未使用的 `sass-loader` 及未使用的 ESLint 旧插件，替换不兼容的新旧 lint 规则和配置。
- ESLint 使用 `eslint.config.mjs`，Stylelint 使用新版 SCSS/Vue 配置和 stylistic 插件。`pnpm run lint`、`pnpm run lint:eslint`、`pnpm run lint:style` 只检查；修复使用 `pnpm run fix` 或 `pnpm run lint:style:fix`。
- ESLint 的 TypeScript 插件和解析器由 `@zhengshi/vue-typescript/eslint` 导出，与 Vue 检查共用隔离的适配器，保留原有 Vue 模板 lint 规则。菜单回归测试使用 esbuild 转译测试源码，不再调用 TS7 已移除的 `transpileModule` API。

目前 lint 工具配置和解析正常，但源码尚未全量满足规则。旧 ESLint 默认只扫描 TS，已有大量 Prettier 格式错误；新配置同时检查 Vue 文件。Stylelint 旧配置原先存在缺失 preset，现在能执行并报告源码格式、顺序等问题。此次未对业务源码进行全量格式化，不能将类型检查和构建通过理解为 lint 全部通过。

验证环境为 Node.js 26.10.0 / pnpm 12.14.0。TypeScript 7 迁移后，严格 peer 模式的冻结锁文件安装、21 项自动测试及 production/test 模式的完整类型检查与构建均通过，构建日志无 Sass 弃用或资源体积警告。ESLint 对 181 个源码文件的诊断与迁移前逐项一致，没有解析错误；隔离适配器的类型感知规则正反例也通过。

此前依赖升级的隔离浏览器回归共 38 项通过（生产 19 项、极速出库 12 项、开发 7 项），覆盖登录与旧会话恢复、主要列表查询、库存筛选、极速出库选项值与时间联动、报表金额、两份 XLSX 重读及 ExcelJS 懒加载。TypeScript 7 迁移后重新生成的 151 个生产产物与该次验收产物逐文件 SHA-256 一致；未重复执行浏览器回归。这些浏览器测试的 API 全部使用模拟响应，未验收真实后端或生产数据。
