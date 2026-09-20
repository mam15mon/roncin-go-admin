# 技术设计

## 候选能力归属
| 当前内容 | 目标公开入口 |
| --- | --- |
| getMasterDataOptions/getCachedPorts/getCachedAirports/getOrderPersonnelOptions/clearOrderMasterDataCache | `features/orders/options/index.ts`；完整保留订单消费缓存的生命周期，主数据来源不代表缓存必须归全局主数据 |
| searchPartnerOptions 与 PartnerOption 类型 | 扩展 `features/partners/index.ts` |
| getCurrencies/getCurrencyOptions | `features/master-data/currencies/index.ts` |
| searchShippingLineOptions | `features/master-data/shipping-lines/index.ts` |
| disableCreditExceededOptions 与信用候选约束类型 | 扩展 `features/finance/credit-control/index.ts` |
| 基础 SelectOption（label/value/code/name/disabled） | `types/select-option.ts`，领域类型通过组合增加业务字段 |

所有页面与 features 调用方使用公开入口；内部采用相对路径。删除旧 utils/options 与 order-options-cache，测试随所属能力拆迁。组织切换/退出直接消费订单能力清理入口，无需新增缓存注册中心。

币种缓存保持全局主数据缓存，forceRefresh 替换当前 Promise；catch 仅在失败 Promise 仍为当前缓存时清空。订单四种缓存保留原 key/分页和身份保护，不为四段相似代码创造通用缓存库。请求返回值的迟到展示保护仍由既有页面身份门禁负责，不能把 cache key 隔离当成完整 UI 隔离。

## 重复发现工具
入口计划 `pnpm run report:duplicates`。根 `scripts/report-duplicates.mjs` 组织扫描和输出；前端适配器放 web/scripts，使用已有 @babel/parser；Go 辅助工具放 `scripts/duplicate-scan-go/`，仅使用 Go 标准库 go/parser、go/ast、go/token，不变更 server 模块依赖。检查器不执行网络模型调用。

覆盖：web/src 手写 JS/TS 函数、方法、箭头函数；server 内手写 Go 函数/方法（包括有函数体的 schema 方法）。源码按语言分别比较。排除测试、生成标记及已知生成后缀、services/roncin、vendor/node_modules、构建缓存。不能粗暴排除 ent/schema。脚本自身、配置/数据/协议和无函数体声明不纳入产品重复报告。

实现最薄函数指纹：
- 相同结构：去掉位置、注释和格式后规范化语法树；保留操作符、字面量、外部符号、属性名与类型信息。
- 局部改名结构：按词法绑定规范化参数与局部变量及其引用；保留外部调用、属性、标签及字面量。对不支持的绑定结构标明仅参与相同结构比较，不能盲目把所有 Identifier 改成同一占位。
- 用哈希分组而非两两比较，组内再次比较规范化内容；默认最少 60 个有意义语法节点，允许命令参数调整并在报告中记录。
- 同一组优先报告最大完整函数，避免嵌套回调使报告重复堆叠；排序按组覆盖节点总量、文件数和稳定路径，不能混入当前时间导致每次报告无意义差异。

报告支持终端摘要与 `--format json`，字段包括模式、阈值、覆盖/排除统计、语言、符号、路径与起止行、结构依据及解析失败。显式输出到用户指定文件，默认不污染工作区。无候选退出 0、有候选退出 0、解析/运行错误退出非 0。不能把“发现零组”表述为“无业务重复”。

自动运行的是工具针对性测试；整仓报告为按需审查命令，避免 Go 工具链成为普通前端门禁的新依赖。前端检测器单测接入 check:web，Go 辅助测试接入 check:server；现有架构规则仍保持阻断。

## 审查与文档
实仓运行后将本次快照与前十组审阅写入任务 research；说明同一业务能力可复用或分层映射/实体差异导致合理重复，不未经评估合并。更新能力导航，加入候选领域入口及扫描用法/限制，避免声称 utils/options 自动提供组织隔离。

## 风险与回退
barrel 迁移可能影响 mock、类型推断及代码加载；以调用链回归和 tsc 覆盖。规范化必须尊重绑定作用域，改名模式误报风险以明确证据与人工审阅控制。按候选迁移、扫描器、文档三组提交，必要时按提交回退，无数据迁移。
