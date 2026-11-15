# 四类业务、航空直单与运单内容的实施影响

这是当前源码下的设计建议，未实施、未运行测试。命名、唯一性范围和直单共享约束将列入最终设计复核。

## 方向身份与唯一性

- 推荐两套海/空运输单证实体承载四种业务，避免复制四套表。主单具有不可变的业务类型身份，成员必须同公司、同方向。
- 现海运主单唯一键是公司+船司+规范号（`server/internal/data/ent/schema/sea_master_bill.go:51`）；空运是公司+规范主单号（`air_master_waybill.go:36`），无方向字段。
- 若同号进出口各自合法，唯一键必须纳入方向/业务类型；候选、按ID关联、恢复、预览和提交全链同步校验，不能只在前端筛选。
- 不从活动成员推断方向：零活动成员仍需保留主单身份。新增身份的迁移需检查实际存量链接；不能把异常混合数据静默归类。
- `data/air_export.go:88` 成员权限及 `:234` 完整成员读取需按真实业务类型检查，不可过滤掉越界成员后继续写入。

## 航空直单/主分单

- 海运 Link 已有 DIRECT/HOUSE（`ent/schema/sea_master_bill_order_link.go:26`）；航空 Link 当前没有模式（`air_master_waybill_order_link.go:19`）。
- 现航空建票无条件建 HAWB（`data/air_order_sync.go:70`、`:73`），读取/更新/流程/锁/恢复都依赖分单（`data/air_export.go:162`、`data/air_order_sync.go:106`、`:188`、`:233`、`data/air_export.go:465`）。
- 新航空模式必须真实表达 HAWB 存在性，DIRECT 不插入 HAWB、不生成空号，HOUSE 需要真实 HAWB；请求矛盾时显式拒绝而非静默丢弃。
- 当前 HAWB 对 order_id 永久唯一（`air_house_waybill.go:29`），若支持模式转换/重建，需设计生命周期与部分唯一索引或保留同实体事件，不能删除旧分单抹去历史。
- 建议首期 DIRECT 主单只关联一张操作票且不混入 HOUSE；HOUSE 可多票。这项约束尚未由“支持直单”一句话自动批准，需最终复核。

## 重量与财务

- 重量确认绑定 ORDER/MASTER、target、来源和版本，没有 HAWB 外键（`air_weight_confirmation.go:20`）；可以保留算法与当前费用来源。
- 主单汇总按 ACTIVE Link 取订单货物（`data/air_weight.go:61`、`:98`），直单无需伪造 HAWB 才能参与。
- AI 权限需改掉 AE 固定判定（`air_weight.go:33`、`:77`、`:98`、`:204`）。ORDER 与 MASTER 仍是独立确认，即使直单只有一个成员数值相同，也不互相自动确认。
- 不自动把整主单确认重量填入每票费用；`order_fee_quantity_source.go:72`、`:84` 的显式 ORDER 来源及指纹继续执行。
- 主单/成员/货量变化使旧确认过期，既有费用历史不自动改写。

## 结构化航空单证与审核快照

- MAWB 当前只有运输和货量等字段（`air_master_waybill.go:17`）；HAWB 和版本当前是 content string（`air_house_waybill.go:18`、`air_house_waybill_version.go:17`）。
- 建议独立航空纯Go内容模型、当前 JSONB 与不可变版本；MAWB/HAWB 分别持有内容，不共用可变实例，不引入 SeaBillContent 依赖。
- 金额、费率为 decimal 字符串；单证计费量与运费金额不能因海管家同名 billingWeight 字段而合并。
- 正式迁移一次性保留既有自由文本为明确备注字段，不做旧新双读/自动回退，也不自动删除开发数据。
- 审核锁表 CHECK 目前只允许 SE 填海运快照FK（`ent/schema/order_lock_record.go:25`）；SI 必须正式修改 CHECK 和仓储快照创建（`data/order_lock_apply.go:50`、`:97`）。
- 航空快照表达 `{structure,master,optional house,link}`，明确无HAWB；模式、主/分内容和版本参与冻结与历史读取。
- 单证制作不以财务结清为前提；输出已保存版本，主单多人共享修改与页面外部变动保护沿用现有要求。

## 进口服务与进度

用户已明确进口全链按本票承接服务决定适用环节。建议固定的到货/换单/清关/提货/派送进度记录与明确适用性，不恢复租户通用流程设计器。

- 环节可并行；不强行把报关、换单、提货压成一条统一线性 enum。
- 未适用与已完成不同；不因跳过某环节自动写成完成。
- 经营事实、人工审核锁、终止和结案独立；服务范围变更受版本/锁约束并留历史。
- 取消已完成环节、修改既有事实、服务范围变更影响完结条件，均需明确原因及校验；具体接口/事件命名在最终设计中复核。
- 旧 `status_template_id/service_type_ids` 已 reserved（`order.proto:293`），不能重新占用；`order_lifecycle_event.go:23` 的 dimension CHECK 要与新增事件维度同步，或使用独立进度事件实体。

## API 与 Schema 预计影响

- API源：order.proto、air_export.proto（推广覆盖AE/AI时同步服务名/路径和权限）、air_weight.proto、sea_document.proto、order_lock.proto、order_release_pod.proto，以及进口进度/航空打印新契约。根据实际字段影响同步 order_fee.proto。
- Schema：海空主单方向与索引，航空 Link 模式，结构化主分运单及版本历史，模式变化历史（若批准），SI锁快照约束，进口承接服务及进度、航空打印版本。
- 所有变更经 Ent/正式迁移与 Proto 生成；源文件、迁移、OpenAPI和客户端同组提交。无手工改生成物、无清库。

## 建议交付拆分与依赖

1. 方向身份/权限/候选隔离与正式迁移基础；SE/AE回归且同号跨方向与跨方向ID拒绝可验证。
2. 航空直单/主分单、结构化内容和版本；依赖1；覆盖HAWB可选、恢复/审核锁/重量与费用来源。
3. SI/AI进口全流程；依赖1及航空核心2；覆盖固定服务适用性、节点事实、主分单与费用闭环。
4. MAWB/HAWB制作预览/PDF；依赖2；共享字段模型不共用可变数据，验证版式、金额、长文本、分页及并发保护。
5. 四类横切与整体验收；依赖1–4；工作台/通知/台账/日期汇率/编号/导出和四类财务链全部接通。

以上不是把最终目标收缩为基础框架；全部实施交付及其他五类调研覆盖确认完成，才可能通过父任务验收。
