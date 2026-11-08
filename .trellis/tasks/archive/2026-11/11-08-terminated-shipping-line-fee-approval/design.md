# 技术方案

## 船公司名称

API已正确返回订单shippingLineName与保留单证的船公司名称。前端SeaCarrierField只读不调用request，故搜索种子无效。

直接将当前订单船公司ID/名称作为TemplateProps的显式当前展示选项传入字段，或复用已传入seaDocuments.masterBill中同ID名称；实现时选择最小且不改变活动运输语义的入口。不能为只读名称复活seaMasterBill活动关联、发起额外全量请求或使用UUID当名称。正常活动主单和来源加票继续按真实ID关联；名称缺失时明确中文提示。

主要文件：detail.tsx、templates/types.ts（仅确需新展示prop时）、SeaBasicInfoSection.tsx及定向回归。测试覆盖终态无活动摘要但有名称、重新进入、readonly不搜索、正常主单与共享锁定、来源创建。

## 费用补审验收

使用现有OrderFeeCorrectionUsecase及真实仓储授权，先创建100元ADD申请，检查PENDING不影响正式费用，再由具备普通有效grant的用户预览并批准，核对1000→1100、恰好新增一次、锁保持及版本/令牌/重复批准结果。

初始化管理员被真实资格函数排除，不能以bootstrap身份代替审核人；不改权限规则、不临时加权、不重置密码、不冒用他人会话。用户已选择专用隔离PostgreSQL库，按正式迁移建立测试Schema和普通用户/角色/grant，不接触开发业务数据。已撤回原申请保持撤回，不恢复原申请。

## 验证与交付

先定向Vitest/Biome及必要tsc，再按最终范围运行check:web与后端真实库受影响测试；后端实现不变时不为验收重构权限。本次仅隔离测试Schema内产生补审验收数据，开发环境只读核验。此前独立代理服务曾返回额度403，如仍不可用如实记录并由主会话复核，不声称独立review通过。
