# 船代字典来源与核验

## 已观察事实
- 来源页面：`https://eyun.hgj.com/order/edit_sea_export/`。
- 请求：`GET https://anoingress.hgj.com/saas-support/data/item/1020`，HTTP 200，`success=true`，`code=0`。
- REA 运行证据：`ev_6d6607fe63ef63df3755673e588b3b9be7e9b5d9324c9784fbd90d7167fac80e`。
- 接口一次返回 29 项；与页面船代输入框展开的 29 个候选名称和顺序完全一致，无重复 ID。
- 原始业务数据仅保留响应 JSON，见同目录 `eyun-ship-agents.json`；不保留认证请求、Cookie、令牌、账号、密码或其他业务资料。
- 整次网页捕获因其他资源状态标记不完整；本字典请求、200 响应和响应正文均已完整保留，因此 29 项结论限于本字典响应。

## 字段与使用限制
- 第三方字段：`id`、`key`、`value`、`enValue`、`aliasNames`、`order`、`type`、`code`。
- 中文候选名取 `value`；部分 `enValue` 与中文完全相同，不应编造英文译名。可以保留原始源值并在转换时把同名英文视为缺省。
- `id` 只用于来源追溯及种子稳定键，不能直接用作本系统 UUID。
- 所有 `aliasNames` 和 `code` 均为 null，不存在可沿用的官方业务代码。
- 仅证明该页面引用的字典内容，不证明是全量行业船代名录，不证明底层数据库表结构。
- 本期使用静态快照初始化，运行时不依赖海管家账号、网站或联网同步。

## 校验
源响应数据 SHA-256（本任务 UTF-8 格式化 JSON）：`93dca1753a3db49847b6396853b12beac20d5f6772e6b16cd7ebbd86d193069c`。
后续种子转换须逐条核对中文名称和源 ID，正好 29 条；按源 ID 固定编码后重复运行保持本系统记录 ID 和管理员修改不变。
