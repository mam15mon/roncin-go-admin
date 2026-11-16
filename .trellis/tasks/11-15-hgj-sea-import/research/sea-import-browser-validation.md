# 海运进口真实浏览器验收

## 最终结果

2026-11-16：真实 PostgreSQL 隔离库、正式 SQL 迁移、真实后端和浏览器整例通过。实际下载的外部 HBL 归档 PDF 共4页，提字/逐页图片核对通过；本轮数据库、角色、进程组及验收端口全部清理并核验。没有使用业务响应 mock，没有写海管家或发送员工通知。

执行命令：

```bash
RONCIN_ACCEPTANCE_WEB_PORT=8002 node /tmp/roncin-si-browser-acceptance/run.mjs
```

退出码0；内部执行 `pnpm --dir web exec playwright test tests/e2e/sea-import.e2e.ts`，结果 `1 passed (1.4m)`。最终证据目录为仓库外私有目录 `/tmp/roncin-si-browser-vR74zZ`（0700，子文件0600），包含浏览器截图、PDF、提字、4页 PNG及 `pdf-validation.json`。

## 已执行覆盖

| 场景 | 实际结果 |
| --- | --- |
| 组织与身份 | 随机本地管理员真实登录、选择新建公司工作区；读取并验证创建人管理员成员资格，没有借系统工作区经营 |
| SI DIRECT | 真实API创建IMPORT直单、真实海运主单及航程，单证聚合不存在HBL；详情展开操作备注、保存200、刷新重读原值 |
| SI HOUSE | 真实API创建进口主分单，HBL绑定真实外部客商ID/名称，`printTemplate=EXTERNAL_ARCHIVE` |
| 共同航程权限 | 真实SI详情“维护共同航程”可用；联调发现的SE-only reassign误用已由前后端检查代理修复，SI按真实update权限，未借SE权限 |
| 五服务任意子集 | 仅承接换单与派送，两项可并行完成；到货、清关、提货不承接并保持不适用；API重读精确2项承接、2项完成 |
| 服务与整票输入 | 主表存在未保存操作备注时保存服务，备注原文保留；随后主表保存200、刷新重读成功，服务写入后的版本基线可继续提交 |
| 撤回与原因 | 通过日期控件清除派送完成时间，不填原因被明确拦截；填写原因保存成功，承接仍为true、完成时间optional省略 |
| 不可变服务历史 | API核对撤回事件before仍有完成时间、after已省略，真实页面展开历史可见原因 |
| 外部变动保护 | 第二真实API会话修改服务；浏览器回焦检查显示持续变动提示，服务编辑输入保留且保存按钮禁用 |
| 外部HBL维护 | UI录入20行发货人地址及末行、85行货描及末行、60行箱号及末箱；保存200、刷新原文重读、API核对所有原文 |
| 预览/下载/打印 | 真实字体200并生成PDF iframe；预览src、下载href、打开打印href为同一个Blob；实际下载文件名含`-archive-`，不提供本公司四类签发输出 |
| 生产同域 | 8010真实深链SPA 200且text/html、解析主JS 200、同域PDF字体200且正确二进制MIME、未知assets文件404、health/ready 200；直接进入8010 HBL原文重读并实际生成预览 |
| 出口相邻路径 | SE/AE新建页面真实可进入，委托单位combobox及创建订单按钮可见 |
| 页面异常 | 最终整例收集的pageerror为空 |

本组对应父矩阵V1、V8、V9、V14、V15、V16、V18的浏览器风险覆盖；后端越权/事务/锁/并发、财务和完整迁移矩阵仍由各专项证据承担。

## PDF 文件与实图

实际下载：证据目录内 Playwright 场景子目录的 `si-external-archive.pdf`。

- SHA-256：`d63925f775ea1cf45fa8995427d8433788c3940616cef40960e5b7e9e78991c2`。
- PyMuPDF提字结果：4页、6687字符，85行`ACCEPTANCE 1..85`精确计数与集合核对通过；货描末行、发货人末行及60箱末箱均存在。
- 真实外部签发主体名称和PDF作者一致；归档提示存在，未使用经营公司名称冒充抬头；逐字框未越出页面。
- 已实际逐页查看 `si-archive-page-1.png` 至 `si-archive-page-4.png`：中文/英文清晰，行距正常，没有截断、重叠或超边；每页包含HBL号、保存版本及当前/总页数。第3页可见货描末行、第4页可见末箱。
- `si-archive-preview.png` 为开发页面实际预览截图，`si-production-archive-preview.png` 为8010生产同域实际预览截图。
- PDF检查复用已有 `/tmp/roncin-hbl-pdf-tools` PyMuPDF，没有安装依赖。

## 隔离与清理核验

`/tmp/roncin-si-browser-acceptance/run.mjs` 裁用已有财务编排的随机角色/数据库、资源追踪、owner校验、进程组及信号清理能力，仅保留本组最小阶段。使用本机已授权 `sudo -n -u postgres psql` 创建与删除本次随机SI库/角色；不改开发角色权限。实际服务采用 `configs/config.acceptance.yaml`，通知、对象存储关闭；正式迁移不使用Ent运行时建表。

本轮随机资源为 `roncin_acc_si_browser_1794794358509_e3d161da`。harness完成进程组回收、库owner校验后DROP与角色DROP；独立只读核验：

- `pg_database` 指定库计数0、`pg_roles` 指定角色计数0。
- `ss`核对8010、9010、8002均无监听；没有停止8001的既有进程。
- 活跃进程、待清数据库、待清角色三个追踪集合均归零。

## 定向检查与中间问题

- `pnpm --dir web exec biome check tests/e2e/sea-import.e2e.ts`：通过。
- `pnpm --dir web tsc`：最终版本通过。
- `git diff --check`（本组两文件）：通过。
- `node --check /tmp/roncin-si-browser-acceptance/run.mjs`：通过。

联调中先修正了测试准备问题：旧财务示例的成员接口已变更新公司自动挂管理员成员；海运备注为空时初始折叠且textarea使用placeholder；DateTimePicker需点击正式清除按钮；带图标按钮/link的accessible name含图标；SE委托单位与签发主体radio重名需按combobox定位。另有一轮遇客户端生成HMR，代码稳定后重跑。所有中间轮均独立清理，均未当作整例通过。真实共同航程权限断点已反馈并由前后端检查代理修复，最终enabled断言已通过。

## 验收边界

本组通过API辅助建票，未将其宣称为SI新建表单逐字段浏览器提交。只选择五服务的代表子集，不穷举32种组合。实际验证了打印入口与预览/下载共享Blob，未连接物理打印机。SE/AE只验证相邻新建路由，不声明出口全流程浏览器回归。本组不代替费用财务全链路、审核锁/终止/结案、共享箱守恒、权限矩阵和正式迁移保留数据专项；不代表父任务四类整体验收完成。
