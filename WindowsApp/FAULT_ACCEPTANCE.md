# Windows 客户端故障验收矩阵

此矩阵只验证客户端的恢复与反馈，不改变后端业务规则。线上自动化阶段只允许只读测试；写请求故障必须使用负责人指定的一次性测试数据并由人工逐项执行。

| 场景 | 注入方式 | 预期结果 | 自动化证据 |
|---|---|---|---|
| 查询超时 | 本地 `httptest` 延迟超过客户端超时 | 返回可识别的网络超时，不覆盖新查询结果 | `internal/api` 单元测试 |
| 主动取消导出 | 后台任务窗口或页面“取消导出” | 页面恢复可操作，任务清理后移除，不覆盖目标文件、不遗留 `.partial` | `internal/ui` 导出与任务注册表测试 |
| HTTP 401 | 本地服务返回业务码 401 | 只触发一次失效处理并清除本机缓存，返回登录页 | `internal/api` 单元测试 |
| 配置主文件损坏 | 将测试目录主文件替换为非法 JSON | 从最近有效备份恢复并修复主文件 | `internal/config` 单元测试 |
| DPAPI 会话主文件损坏 | 仅在测试用户配置目录副本中破坏主文件 | 从密文备份恢复；任何文件均无明文 Token | `internal/securestore` 单元测试与人工抽查 |
| 后台 goroutine panic | 使用受控测试入口触发 | 主窗口保持运行，显示故障编号，日志不包含密码、Token、请求体或异常文本 | `internal/ui`、`internal/diagnostics` 单元测试 |
| 写请求响应不确定 | 测试代理在请求发出后断开连接 | 不自动重提；操作复核保留“结果待确认”，重启后可只读回查 | 必须人工、一次性测试数据 |
| 图纸/附件下载失败 | 切断图片服务或返回无效图片 | 清空旧图，显示失败与重试入口，不继续显示上一张图片 | 单元测试加 Windows 人工检查 |

发布验证命令：

```powershell
.\scripts\verify-release.ps1
.\scripts\verify-release.ps1 -OnlineReadOnly
```

正式签名与安装包：

```powershell
.\scripts\verify-release.ps1 -CertificateThumbprint "<SHA1>" -RequireSignature -PackageInstaller
```

未提供有效证书、`signtool.exe`、Inno Setup 或 ARM64 Windows 11 真机时，对应门禁必须记录为“未执行/阻塞”，不得用未签名 EXE、交叉编译或当前 x64 设备代替。
