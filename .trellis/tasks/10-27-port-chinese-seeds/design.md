# 港口全量中文名称设计

## 行为缺口与责任边界

现有命令能解析官方海港并同步英文事实，但没有中文名称输入。行为应在 cmd/sync-unlocode 的数据装配处和 data.ApplyPorts 的持久化处补齐，而不是前端临时翻译或更改业务查询接口。

## 最小数据流

1. 当前官方 ZIP 按既有规则解析海港集合；集合和标准字段仍为唯一身份依据。
2. 在 server/seeds/ 维护与该版本配套的全量中文 CSV（建议 ports.zh-CN.csv），每行包含 un_locode、name_en、name_zh、name_kind、source_ref。country 由标准代码及 ZIP 校验，不另建国家别名主数据。
3. CSV 用标准代码精确关联，额外核对 name_en，严格校验完整集合、唯一性、中文长度/汉字和来源标记；全部成功后构造包含 NameZH 的 PortSyncRecord。
4. ApplyPorts 在既有 WithTx 事务中新增/更新 name_zh、name_en 等本来源字段；中文非空由已验证装配保证，不添加保留旧值或在线回退分支。已存在的其他来源代码冲突继续报错。
5. Ent Port 的现有 searchKeywordsHook 同时收到新中英文，更新 search_keywords；无需 Schema 或接口生成。未被当前发布包包含的本来源港口仍按现有规则停用。

## 名称编制与清洗

- 先使用有依据的已有中文名，复核已采集参考样本的国家和港区语义；没有证据的名称不标注为已有通用名。
- 长尾项按地名语义和当地拼写翻译或音译，编制完成后作为静态数据提交。允许本次数据准备使用模型编制译名，但不新增付费外部 API 依赖或从配置搜取凭据。
- 中国/日本等原有汉字地名优先参考原名；常用国际港和高风险样本单独复核。不按英文字符串跨国共享中文结论，不按 IATA 或旧行业码自动改写 UNLOCODE。
- CSV 的 name_kind 区分 existing / translated，source_ref 对已有名写来源依据，对译名写明按标准英文名称翻译/音译的编制说明。存在不确定性的长尾译法如实标为 translated，不假称官方。
- 底表中的区分词如 Pt、Terminal、Island 结合含义处理，确保城市/港口/港区仍为独立代码。
- 采集样本仅归档必要的公共字典字段及清洗说明，不提交账户资料、私人订单、浏览器状态、原始账户网络日志。

## 命令与完整性

- 日常 pnpm run sync:unlocode、sync:all 自动读取随仓库归档的中文 CSV；沿用现有 ZIP 路径解析模式，若添加中文源参数仅用于显式指定配套文件。
- 默认预览不写库。数据装配校验应在数据库读取/写入之前完成，预览汇总包括 17,524 总数、100% 中文覆盖、existing/translated 分类和指纹。
- source_hash 由基础 ZIP 与中文 CSV 的确定性组合共同计算；中文名修订必须反映到指纹。另打印各源文件摘要便于排查。
- ZIP 升级造成代码新增或英文身份改变时，必须同步修订中文 CSV，否则明确失败。禁止静默复用过期名称、自动补译或把英文写入中文列。

## 影响文件

- server/seeds/ports.zh-CN.csv：全量中文对照与来源标记。
- server/seeds/README.md 及必要的数据编制/清洗说明：来源、覆盖范围、译名限制和同步用法。
- server/cmd/sync-unlocode/main.go 及就近中文 CSV 解析文件：读取、关联、完整性校验、指纹与预览统计。
- server/cmd/sync-unlocode/*_test.go：解析和完整集合测试，真实配套文件验收。
- server/internal/data/industry_reference_sync.go：PortSyncRecord.NameZH 与新增/更新写入。
- server/internal/data/港口同步专项集成测试：统一事务、补齐、幂等、检索和来源冲突验证。
- .trellis/spec/server/backend/相关种子规范：记录全量中文覆盖与版本配套契约。

无需改动 Proto、Ent Schema、迁移或前端生成物。若实施中证明这些边界不足，先回到规划说明具体原因，不自行扩大。

## 风险与恢复

数据量大，非空不能证明语义正确：用常用港复核、按国家批次检查、专门检测跨国误配与城市港区混淆降低风险，译名始终保留编制标记。恢复通过回退本组代码和中文 CSV 的 Git 提交后重新预览；本任务不在业务库执行同步，所以没有自动回滚业务数据的动作。
