# R3 分单当前快照链路

## 根因与变更边界

普通创建及普通 HBL 保存只写工作实体；不可变版本原先仅在审核或改单等动作产生。
模式切换执行同时要求实体版本和 current_version_id，因此从普通创建进入该操作会失败；
原预览却允许缺快照，形成可执行性漂移。

新增 HBL 版本来源 DRAFT_SAVE（界面显示「草稿保存」），复用 createHouseVersion：

- 普通订单创建、整单保存 HBL 和独立分单保存，在原事务内追加快照并推进当前指针。
- 拆票新分单、原分单改挂以及整票改配后，同步归档当前 HBL 身份与版本，避免指针留在旧主单。
- 模式切换预览要求存在同组织/订单/主单/分单的快照，且实体版本和内容哈希匹配；不再展示缺基线的可执行预览。
- 既有 expectedCurrentVersionId 校验、订单/Link/HBL 版本、业务锁与财务门禁均保留。
- 不扩展 MBL/TE 普通保存，不自动更新存量业务记录。旧的无快照 HBL 可由用户显式保存分单建立当前快照。

草稿快照不代表审核；审核可复用内容与实体版本相同的快照，不伪造 ORDER_LOCK 来源。

## 契约与生成

- proto 新增 SEA_DOCUMENT_VERSION_SOURCE_DRAFT_SAVE = 6；同步 biz 常量及 service 映射。
- HBL Version Ent Schema 扩展 source 枚举及显式 CHECK；正式迁移 20261106090000_sea_house_bill_draft_versions.sql 仅替换来源 CHECK。
- 迁移 SQL 从 Ent 生成元数据 SeaHouseBillVersionsTable.Annotation.Checks 中的 sea_house_bill_versions_source_check 生成，生成器为临时 Go 程序，未保留运行产物。
- 已执行 make -C server api、go -C server generate ./internal/data/ent、pnpm run generate:web-client；生成 CLI 对 go.sum 的额外工具依赖写入已移除，业务没有新依赖。

## 验证

新增 TestSeaHouseBillDraftVersionLifecyclePostgres 从正式 Usecase.Create 进入，验证创建后可预览、整单与独立保存快照、旧基线冲突、HOUSE→DIRECT、重放和反向转换。未人工补足版本。

新增 TestSeaHouseBillModePreviewRejectsInvalidBaselinePostgres、TestSeaHouseBillDraftVersionSourceMetadata、TestSeaHouseBillDraftVersionMigrationUpgradePostgres，验证缺失/过期快照拒绝、Ent CHECK、正式旧前态升级及重复迁移只有一条 revision。

补充独立保存审计失败时快照数量/当前指针回滚、拆票新分单快照、改配后主单和实体版本一致断言。

已通过：新增四项真实 PostgreSQL 测试；service/biz 海运定向测试；SeaDocumentHistoryActions、HOUSE→DIRECT、DIRECT→HOUSE 共三个前端文件八个测试；修改文件 Biome、git diff --check。

全部 TestSea 与订单创建/幂等/审核相邻集成回归运行 121 秒，仅两个旧单证测试因 actorID 为未入库随机 UUID 而失败；将这两处夹具改为真实 User 后，重跑这两项及分单印字、完整拆票改配、业务锁、真实创建至转换链路，共六个父测试组 PASS（15.7 秒）。没有放宽 created_by 外键。

二次执行全部上述生成命令，逐文件 SHA-256 比较生成目标，差异为空。最终修改文件 Biome 与 git diff --check 通过。主会话负责最终全量门禁、独立 review 和合并。

## 开发种子契约同步

sync-dev 在本次新建订单的演示操作完成后，调用导出的 SaveSeaHouseBillDraftVersion，为有效 HBL 归档最终工作状态。仅处理 newOrderIDs，既有用户记录不回填；历史事件仍引用旧的不可变版本。种子审核样例为 DIRECT，当前快照推进不会改写审核引用。TestDevelopmentSeedPostgres 真实 PostgreSQL 验证当前身份/版本/内容完整、重复执行版本数不变（3.796 秒通过）。
