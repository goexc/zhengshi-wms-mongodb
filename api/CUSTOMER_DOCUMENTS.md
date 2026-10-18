# 客户承兑与销售开票记录

本功能在现有客户、客户流水、图片素材与 COS 上传服务上增量实现。承兑与发票分别存入 `customer_acceptance`、`customer_invoice`，附件继续使用相对于 `images/` 的文件引用以及原有 `annex` 存储约定。

## 业务规则

- 承兑必填客户、票据号码、票据金额、收到日期、出票日期、到期日期；出票日期不晚于收到日期或到期日期。允许补录收到时已经逾期的票据。
- 登记承兑不改变客户应收。整票等额确认到账时，以兑付日期创建一笔 `payment` 并冲减应收；不填写收款账户，到账凭证可为空。
- 也可选择同客户、同金额、有效且未被其他承兑占用的既有人工回款，只建立关联，不再次扣减余额。
- 已兑付记录禁止普通编辑/作废。“撤销到账”必须填写原因：自动生成的回款标记为 `reversed`，恢复余额；原人工回款只解除关联，余额不变。两种操作均保留历史。撤销后可重新兑付。
- 发票必填客户、发票号码、开票日期、税率、不含税金额、税额、价税合计。税率 `13` 表示 `13%`；税额可按票面校正，后端精确校验未税金额加税额等于合计。登记、修改、作废发票均不改变应收。
- 本期不处理部分兑付、贴现、背书、手续费差额、红蓝发票、出库关联或税务开票。发票作废只表示撤销本系统登记。
- 票号/发票号在各自集合内唯一，包括已作废记录。金额最多两位小数，数据库同时保存整数分供汇总与精确比较。
- 创建及状态动作使用幂等键；编辑使用版本条件更新，并保存原版本和请求指纹作为操作回执。同一次编辑的精确重试不会再次更新版本或覆盖后续操作。重试必须保留原操作键与请求内容，遇到冲突应先刷新核对。

## 附件目录及兼容

沿用 `POST /images`，multipart 文件字段仍为 `files`，单次请求上传一个文件，组件负责多选逐个上传。可选 `scope`：

| scope | COS 对象路径 | 返回并保存的引用 |
| --- | --- | --- |
| 不传/空 | `images/<随机文件名>` | `<随机文件名>` |
| `acceptance` | `images/acceptance/YYYY/MM/<随机文件名>` | `acceptance/YYYY/MM/<随机文件名>` |
| `acceptance_redemption` | `images/acceptance/redemption/YYYY/MM/<随机文件名>` | `acceptance/redemption/YYYY/MM/<随机文件名>` |
| `invoice` | `images/invoice/YYYY/MM/<随机文件名>` | `invoice/YYYY/MM/<随机文件名>` |

客户端不能自定义目录。每个文件最大 20 MiB，每组附件最多 10 个。上传接口使用 90 秒超时，避免普通接口的 3 秒默认时限中断较大文件上传。新财务附件允许 JPG/JPEG/PNG/WEBP/GIF/BMP/PDF/OFD/DOC/DOCX/XLS/XLSX；旧无 scope 上传方式保留。现有基础文件地址含 `/images/`，不能再次把该前缀写入业务引用。

`GET /images` 支持同名 `scope` 查询。无 scope 保持旧素材列表，仅展示旧通用素材，不混入新的财务文件。图片使用原有预览，PDF/其他文件使用原文件地址打开或下载，不追加缩略图样式后缀。

新单据保存附件前检查素材存在，并与附件删除通过同一素材行写入相互排斥。历史附件引用保留，作废或移除显示引用不会使历史文件可被物理删除。物理删除先预留 `deleting` 状态，再删除 COS；COS 删除失败可重试。旧文件不需要移动，旧业务附件不需要批量迁移。

新关联的财务附件还须匹配用途：承兑原件、到账凭证、发票原件不可跨目录混用。旧通用附件及单据已保存的历史引用可以保留。素材名称按普通文字匹配，含方括号、括号等字符时不会被解释成正则表达式。

COS 请求不输出文件正文及鉴权头调试日志。文件上传成功但元数据写入响应异常时，按预分配 ID 核对保存结果；无法确认时保留 COS 对象并提示核对素材库，避免误删可能已保存或已关联的附件。未落库的孤立对象只记录定位信息，不自动删除。

## 接口

沿用 `{code,msg,data}` 返回信封；业务失败需检查 `code`，不能仅看 HTTP 状态。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET / POST / PUT | `/customer/acceptance` | 分页查询、新增、修改承兑 |
| GET | `/customer/acceptance/detail?id=...` | 承兑详情 |
| GET | `/customer/acceptance/payments` | 查询可绑定的等额人工回款 |
| POST | `/customer/acceptance/redeem` | 生成回款到账或绑定已有回款 |
| POST | `/customer/acceptance/reverse` | 受控撤销到账 |
| POST | `/customer/acceptance/void` | 作废未兑付登记 |
| GET / POST / PUT | `/customer/invoice` | 分页查询、新增、修改发票 |
| GET | `/customer/invoice/detail?id=...` | 发票详情 |
| POST | `/customer/invoice/void` | 作废发票登记 |

字段定义以 `apis/customer_documents.api` 为准。所有日期为 Unix 秒，业务日期按北京时间自然日输入、展示和比较。历史记录的原始秒值及请求指纹保持不变，同一天不同时间不会被误判为日期逆序。列表支持 `page/size/customer_id/keyword/status/date_start/date_end`，日期筛选分别对应收票日期、开票日期；承兑另支持到期与兑付日期、到期提示筛选，发票支持税率筛选。列表 `summary` 为完整筛选集汇总，作废金额独立列示，不加入有效金额。

到账请求包含 `id/version/redemption_date/idempotency_key`，可选 `redemption_annex/remark/transaction_id`；指定 `transaction_id` 表示关联旧回款。作废、撤销到账请求包含 `id/version/reason/idempotency_key`。

## 菜单与部署

API 启动会幂等创建新集合索引，并只补齐缺失的菜单和 API 目录，不覆盖已有菜单定制、不改已有角色授权。

- Web 路由为 `/finance/acceptance`、`/finance/invoice`，组件为 `/finance/acceptance/index`、`/finance/invoice/index`。
- 承兑按钮权限：`finance:acceptance:list/add/edit/redeem/reverse/void`；发票按钮权限：`finance:invoice:list/add/edit/void`。
- 系统管理员沿用既有全部权限规则。普通角色须在角色管理中分配相应菜单、按钮和对应 API；客户选择依赖 `GET /customer`（Web）或 `GET /customer/list`（Windows/App），附件依赖 `GET /images`、`POST /images`。关联回款还需 `GET /customer/acceptance/payments`。
- Web 与 Windows 提供登记、编辑、到账、撤销到账及作废操作；App 提供独立台账、客户入口、详情及附件查看，维护操作在 Web 或 Windows 完成。
- 新财务文件的素材查询/上传还会检查对应业务 API 权限，只有旧通用图片权限不能访问财务素材目录。
- MongoDB 必须支持事务（副本集或支持事务的集群），与当前应收入账要求一致。不要用 standalone 降级拆成多次非事务写入。
- 更新 API 与客户端后重新登录/刷新权限菜单。客户删除会检查承兑、发票及其历史引用。

## 验证

```powershell
go test ./...
go build ./...
# 仅使用无凭据的本机隔离副本集；测试自行创建并删除随机测试数据库。
$env:WMS_TEST_MONGO_URI = 'mongodb://127.0.0.1:27129/?replicaSet=codexFinanceAudit'
go test -p 1 ./internal/handler ./internal/handler/customer ./internal/logic/customer/documents ./internal/logic/customer/transaction ./internal/logic/images -count=1
# 仅为专用一次性 MongoDB 启用 enableTestCommands=1 后使用；故障注入用例须串行执行。
$env:WMS_TEST_MONGO_FAILPOINTS = '1'
go test -p 1 ./internal/logic/customer/documents ./internal/logic/images -count=1
```

COS 测试使用本机 HTTP 替身验证实际 SDK 写入路径、校验和及元数据；不会上传真实 COS。数据库集成测试不读取生产配置。部署后仍需用授权账户确认真实 COS/CDN 文件打开、角色授权以及客户端实际使用环境。

### 2026-10-18 检查与修复

| 检查项 | 验证结果与修复 |
| --- | --- |
| 应收变化 | 隔离副本集验证：应收 1000，登记承兑不变，兑付 100 后为 900；并发及重试仍只有一笔回款。撤销自动回款恢复 1000，再次兑付为 900；关联或解除既有人工回款不再次改变应收。发票登记、修改、作废均不入账。 |
| 重复与未知结果 | 修复编辑精确重试误报版本冲突、人工回款重复写后在已中止事务内查询的问题。增加请求指纹核对，事务外恢复回执；故障注入覆盖提交已完成但响应异常。Web/Windows 在异步校验前锁定提交，结果未知后继续使用原请求，后续冲突不能清除未知状态。 |
| 三类附件 | 实际 COS SDK 对本机 HTTP 替身验证三个目录及旧平铺目录。增加新绑定的用途校验；旧通用文件、已有当前及历史引用可保留。验证绑定与删除竞争、已引用文件禁止删除、COS 删除失败后重试和元数据确认异常恢复。 |
| 上传与文件查看 | 修复默认 3 秒上传超时，真实注册路由通过超过 3 秒、超过默认请求大小的上传测试。素材搜索使用文字匹配；客户端修复并发上传回调、素材加载异常、PDF 原文件链接和 Windows BMP/WebP 解码。 |
| 日期、金额与会话 | 北京时间输入、筛选、回填与自然日校验一致；修复半分税额舍入、金额上限不一致、迟到响应覆盖以及 App 切换用户后残留的筛选与详情上下文。 |

本轮 API 全量测试（含本机副本集及提交/元数据故障注入）、构建与 API 定义校验通过；日期修复后补跑受影响的 documents 全套测试通过。Web 73 项测试、类型检查、构建与隔离浏览器交互通过。Windows 全量测试、构建及相关包 race 检查通过；App 72 项测试、UTS/SFC/路由检查，以及 HBuilderX Web 与 Android 资源编译通过。浏览器检查证据见 `doc/verification/customer-documents-20261018-web/`。

兼容边界：旧人工流水 API 仍接受不带幂等键的请求，此类旧调用无法识别跨请求重试；需要可靠重试的调用方应保留同一个键及原始请求内容。客户端未知请求保留在当前页面或弹窗内，关闭或重启后应先读取最新记录核对。上述验证没有访问生产数据库、真实 COS/CDN，也不替代 Windows 实际界面与移动真机验收。
