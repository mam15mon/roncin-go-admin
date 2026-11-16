# AI 前置交付后的定位提示

这是SI终验期间的只读源码定位，不代表AI已实施或验收。航空结构和固定进口服务已在前置组
实现；AI应贯穿真实类型及权限、日期和界面，不复制航空或进口服务模型。

## 后端需复核的当前出口限制

- `biz/order_usecase.go` 的创建类型、已有主单延迟校验和 `AirExportInput` 校验；
  `service/order_write.go` 的选已有MAWB时update权限。
- `service/air_export.go` 的 `airPrincipal` 固定AE、候选只收AE，ID读取/共享预览/写入/
  恢复/分单历史须基于真实目标类型授权。查看生成权限路由的源配置，不能只改service。
- `data/air_order_sync.go` 的新主单 `SetBusinessType("AE")`、加载/恢复/投影的AE过滤；
  `data/air_export.go` 的候选、成员、共享锁、终止候选及直接ID固定AE。
- `data/air_weight.go` 与 `biz/air_weight.go` 的MASTER/ORDER对象识别、成员查询、锁内
  确认及真实品类授权。AE-only权限不能读取或修改AI；不能在全量集合筛选后静默丢AI成员。
- `data/order_write_lifecycle.go` 的航空终止恢复条件；`data/fee_ledger_master_no.go` 的
  航空号码投影；航空审核指纹与货物/费用来源同样需要验证。
- 权威计划时间/地点、ETA经营日与费用系统汇率同步沿用现模型；补跨中国午夜的真实PG
  场景，并核对锁/已建账导致的回滚。不能机械替换所有AE常量，财务/Partner枚举另有语义。
- `workbench.go`、通知和钉钉路径还存在出口限制，属父横切组；本组记录精确缺口，
  不因AI可建票就声称最终四类工作台已通过。

## 前端需复核的当前出口限制

- `order-kinds/types.ts` / `registry.ts` / `air-export/definition.ts` 和集中 routes；
  AIR公共表单与adapter需接受真实类型，AI定义只组合共有入口，避免逐页复制公共映射。
- `air-master.tsx` 的AE权限、queryKey、路由、成员链接/建票按钮/继续建票；
  `AirMasterFields.tsx` 的候选类型、主单链接及详情key；`air-master-change-notice.ts`。
- `use-order-detail-data.ts`、`detail.tsx`、`AirExportDetailFeatures.tsx` 与
  `ImportHandlingPanel` 集成；`list-documents-action.ts` 和服务保存后确切基线保护。
- 费用页/`OrderFeeHeader`/`OrderFeeTableTabs`、货物drawer、状态卡的isAir固定AE条件；
  须确认AI的重量来源和危险品显示，不能借AE权限。共享海运和航空控件分支按运输类型处理。
- 每个成功读取边界严格复验Order/master的ID、当前公司和真实业务类型，queryKey携带类型。
  服务表单仍固定打开快照，完成保存只采纳确切Order+1且航空Master/Link/House身份未变。

## 最小验证闭环

先实现，后补针对性测试。AI DIRECT/HOUSE新建与私有修改、候选和共享主单完整成员、
重量ORDER/MASTER及PLANNED/ACTUAL、货量变动过期但费用不追改、真实审核/终止恢复、
五服务子集/锁/版本/完结、方向/公司/真实类型权限、ETA费用日及跨午夜是本组底线。
沿现有随机隔离Schema执行正式SQL，不应用现开发库存在孤立主单的迁移，不清库。
精确结果落在本组research；普通Go中的集成SKIP不计实际PG通过。AWB制作/PDF和完整CSV
仍由后续依赖组交付，不虚称本组已通过。
