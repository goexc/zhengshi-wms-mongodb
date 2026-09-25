# 正时 WMS 移动端

`App/` 是本仓库 WMS 移动客户端的开发与发布工程。它使用迁入的 uni-app x、Vue 3、Cool Unix / Cool UI、TypeScript、TailwindCSS 和 SCSS 工具链，业务接口与本仓库 `api/` 对齐。

迁移实施、源备份及验收状态见 [App 迁移实施与验收记录](../doc/App迁移实施与验收记录.md)。原方案保留为设计依据，当前完成情况以实施记录为准。

迁移前 App 与 Appx 源码已保存到 [本机源码归档](D:/Code/_backups/zhengshi-wms-mongodb/App-migration-20260916-source.zip)，哈希和条目清单见实施记录。

## 功能与入口

底部导航为“工作台 / 物料 / 出库 / 我的”。工作台按账号权限显示业务入口。

| 模块 | 当前实现 |
| --- | --- |
| 账号 | 手机号密码登录、`device_type: app`、个人信息、菜单权限、到期退出、本地会话清理 |
| 物料 | 型号搜索、分页、完整资料、客户价格、图片预览和保存；有权限时编辑基础资料 |
| 出库 | 单号及客户查询、状态与类型筛选、分页、单据详情、历史物料明细、当前物料联查 |
| 出库报表 | 按客户和签收日期查询，展示物料金额及按单位分组的数量汇总 |
| 客户 | 名称及资料筛选、分页、工商资料、真实应收与贷项余额、只读交易流水 |
| 供应商 | 名称及资料筛选、分页、工商资料、等级与联系方式 |
| 计划 | 全部 / 执行中 / 已完成筛选、分页、卡片内完整详情与物料联查 |
| 我的 | 当前用户资料、大字显示、版本、使用说明、退出登录 |

供应商财务流水没有对应的现有 API，因此没有流水或余额占位按钮。计划目前只读。拣货、打包、称重、签收执行、库存调整、扫码、离线写入和消息推送不属于本次已实现客户端能力。

## 工程结构

```text
.cool/               框架工具、统一请求、账号与路由
components/wms/      页面壳、客户选择等业务组件
config/              API 环境、文件服务、品牌配置
pages/               WMS 页面
services/            物料、出库、往来单位与计划服务
types/               WMS 业务类型
utils/               格式化、图片、显示偏好
styles/wms.scss      WMS 颜色、字体与布局规则
scripts/             UTS、SFC、路由和引用检查
tests/               会话、接口契约与页面行为回归
```

页面通过 `services/` 使用 `.cool/service`。请求层读取 WMS `{code,msg,data}`，成功返回 `data`；HTTP 200 不代表业务成功。`Authorization` 发送后端约定的原 token；没有虚构 refresh token 或自动刷新接口。

只读入口依据 `/account/menu` 返回的权限和菜单路径显示，物料编辑要求 `material:material:edit`。后端 path + method 授权始终是实际访问边界。

## 安装与本地检查

项目 Node.js 约束为 **>=22.19.0 <23**；行为测试使用 Node.js 自带的 TypeScript 去类型能力。包管理器锁定为 pnpm 10.15.0。项目依赖锁定源码 Sass 编译器 `sass@1.77.8`，通过锁文件安装，不依赖其他项目的 node_modules。

在本目录执行：

```powershell
pnpm install --frozen-lockfile
pnpm run validate:uts
pnpm run validate:sfc
pnpm run validate:integration
pnpm test
```

- `validate:uts`：业务源码及页面脚本的语法和已知 UTS 约束检查。
- `validate:sfc`：Vue SFC 脚本、模板解析与编译检查。
- `validate:integration`：路由注册、页面引用、局部导入和静态资源检查。
- `test`：执行 `tests/wms-*.test.mjs` 中的接口与页面逻辑测试。

这些检查不启动真实 API、不修改线上业务数据，也不能替代 HBuilderX 原生编译和真机验收。

## 运行与环境

本轮使用 **HBuilderX 5.24**，安装位置为 `D:\Program Files\HBuilderX5.24`。可通过 IDE 运行/发行，也可使用下面的 PowerShell 7.2+ 脚本重现 Web 或 Android 资源编译。脚本调用该安装目录中的 Node 和 uni 插件，不使用系统 Node 执行 HBuilderX 构建。

```powershell
pwsh -NoProfile -File ./scripts/build-hbuilderx.ps1 -HBuilderXPath 'D:\Program Files\HBuilderX5.24' -Platform web
pwsh -NoProfile -File ./scripts/build-hbuilderx.ps1 -HBuilderXPath 'D:\Program Files\HBuilderX5.24' -Platform app-android
```

`-HBuilderXPath` 接受实际安装根目录，`-Platform` 接受 `web` 或 `app-android`。产物位于 `unpackage/dist/build/<platform>`，逐次日志位于 `unpackage/verification/hbuilderx-<platform>-<时间>-<进程>.log`。脚本流式显示编译输出，结束后恢复环境变量和工作目录，不清理缓存或删除目录。除检查 CLI 退出码外，它还检查 TypeScript、UTS、Kotlin 等编译错误；遇到退出码为 0 但日志出现 `error TS2339` 的情况，脚本仍返回非零。

这里的 `build -p app-android` 是**资源编译**，可生成 Kotlin 源文件，不能据此认定 Kotlin 编译、DEX/APK 打包或真机运行已经成功。完整 Kotlin 检查使用 HBuilderX 开发编译链路：相同工具链环境下运行 `uni.js -p app-android`（不带 `build`），会进入持续监听。本轮检查使用独立 `unpackage/native-audit-dev/app-android` 输出目录；其最终结果见实施记录。不要同时对同一工程输出目录启动多个构建进程。

`vite.config.ts` 使用 HBuilderX 配套的 `@dcloudio/vite-plugin-uni`。`package.json` 仍没有普通 Vue 项目的 `dev` / `build` 命令；上面的脚本才是当前可复现的资源编译入口。

| 配置 | 位置与当前值 |
| --- | --- |
| 开发 API | `config/proxy.ts`；优先 `VITE_WMS_API_URL`，默认 `https://wmsx.api.goexc.cn:1443` |
| Web 开发代理 | `config/dev.ts` 使用 `/dev`；代理去掉该前缀后转发到开发 API |
| 发布 API | `config/prod.ts`：`https://wmsx.api.goexc.cn:1443` |
| 图片服务 | `config/index.ts` 的 `fileBaseUrl`：`https://wms.file.goexc.cn/images/` |
| 应用身份和平台配置 | `manifest.json` |
| 页面和 Tab | `pages.json` |

默认 API 地址由 `config/endpoints.ts` 统一定义，原生开发和发布均读取该值。Web 开发代理可额外用 `VITE_WMS_API_URL` 覆盖代理目标；此变量不覆盖原生或 Web 发布地址。进行物料保存等写入验收前，应配置对应平台的测试服务，并使用测试账号和测试物料。

客户端不存在 `/api` 固定前缀。图片地址独立于 API 地址；物料列表缩略图使用 `_148x148` 后缀，预览和保存使用原图。

## 数据与验收约定

- 物料 `quantity` 表示安全库存，不显示为实时库存。
- 单据明细展示开单时的名称、规格、价格；查看当前物料不会改写历史记录。
- 客户应收和贷项余额来自后端字段；未返回的金额显示“—”。客户端不通过当前页流水重新计算总余额。
- 出库报表按**签收日期**查询，数量按单位汇总。物料金额合计不包含运费等其他单据费用。
- 列表失败不推进页码；查询序号阻止慢响应覆盖新条件；退出会清理页面状态和账号资料。
- 当前 UI 使用浅色设计和大字显示。Web 检查通过并不等于 Android/iOS 已验收。

原生编译、线上接口联调、设备图片权限、签名安装、覆盖升级和业务验收结果持续登记在 [实施与验收记录](../doc/App迁移实施与验收记录.md)。在这些项目完成前，不把源码迁移完成表述为生产发布验收完成。
