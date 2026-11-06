# 海运操作阻塞排查与修复

## 目标与状态
让满足业务条件的拆票、改配、HOUSE→DIRECT 完成正向闭环，失败提示可操作，保存和返回订单页面稳定。用户已确认推荐的拆票货量口径，并于 2026-11-06 批准开始实施、review 后合并主分支并 push。

## 问题与要求
- R1/P1：用户报告实际箱、委托件重尺存在但货物项为 0。拆票上下文只查 OrderCargoItem（server/internal/data/sea_order_change_split.go:82），非零检查只累计货物分配（同文件:659）。已有货物维护入口（web/src/pages/orders/list.tsx:286）。按用户确认，继续以实际货物明细为守恒依据，拆票页明确区分委托量、实际箱与实际货物，补充缺失提示和直接录入入口；录入成功后刷新上下文，重新确认分配，不覆盖已有未保存输入。
- R2/P1：版本仅从成功预览写入，失败清空预览但不清空旧版本，确认按钮未按预览状态限制（web/src/pages/orders/components/drawers/SeaOrderReassignmentModal.tsx:181、310、511）；服务端已映射版本（server/internal/service/sea_order_change.go:360）。须修复预览和提交状态、误导错误及迟到响应。原始现场尚未复现，不能断言接口漏字段。
- R3/P1：用户新建无财务事实的 SE2026110600003 预览成功而确认失败。普通创建 HBL 未建立当前快照（server/internal/data/order_write_sea_document.go:101）；预览允许无快照（server/internal/data/sea_document_change_preview.go:127），前端执行要求 currentVersionId（web/src/pages/orders/templates/components/sea/SeaDocumentSection.tsx:460），服务端也比较快照（server/internal/data/sea_document_change_mode_change.go:135）。须打通正常创建到模式切换的版本生命周期并保留历史。
- R4/P1：拆票返回、保存后偶发错误页，刷新恢复且数据已保存。通用提示可由路由错误边界或工作区切换失败显示（web/src/router/adaptRoutes.tsx:64；web/src/app/AppLayout.tsx:234）。须取得实际错误栈，定位并针对性修复，不能仅凭文案推断根因。

## 约束与范围外
- 已建账费用或活动账单行阻断拆票/改配是既有业务规则（server/internal/data/sea_order_change.go:127、145），不放开门禁以完成正向测试。
- 保留权限、组织范围、审核锁、守恒、版本、事务和幂等约束，不自动把委托量当实际货量。
- 不重置数据库、不自动改写用户原订单、不新增历史兼容逻辑。正向验证使用正式创建入口的新测试数据。
- 统一验收同一订单操作链路，不涉及页面视觉重构或新的财务回退能力。

## 验收标准
- AC1/R1：只有委托量或实际箱、没有实际货物明细时，明确提示并可直接进入有权限的货物录入；无权限时解释原因。录入后可重新加载分配基线，刷新覆盖已有输入前须确认。有效未建账订单完成拆票，原票和新票均有非零货物，件重尺及费用守恒、历史可追溯。
- AC2/R2：合法 NEW/CANDIDATE 可完成改配；预览失败、未完成、输入变化及乱序响应不能拿旧预览提交，提示具体原因；并发变更仍报冲突。
- AC3/R3：正式创建未审核 HOUSE 订单，无阻断事实时可转 DIRECT，原 HBL 作废且历史完整；反向转换、已有快照及版本冲突路径正常。
- AC4/R4：拆票返回、保存后重读、重复进入详情正常；失败不丢输入，不把导航异常误报为保存失败。未取得复现证据不能声称已修复。
- AC5：真实 PostgreSQL 正式迁移集成验证及前端测试覆盖真实创建到执行，不仅依赖人工补版本的夹具；执行风险匹配门禁。
