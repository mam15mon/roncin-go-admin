# 移除未完成计息入口：技术设计

## 边界与实现

移除未交付UI、成功toast、本地interestRule state及没有其它消费者的InterestRuleModal；同步清理SettlementSection中专属编辑入口、启用摘要和参数传递。实际已有结算规则仍正常保存回填。
实现前使用rg核对组件/类型消费者，确认删除仅影响未完成前端能力；不修改后端、依赖、生成客户端、Schema或开发数据。不得用重命名提示或静默处理保留假生效状态。

## 文件所有权与前置条件

本任务在10-12-ui-save-protection完成并提交后启动，与其共享partner-detail.tsx，因此不能并发修改该文件。若实现时发现该组件存在其他真实业务消费者，先核实其契约，保留有实际用途的能力；不把未知计息业务纳入本次。

## 影响文件

- `web/src/pages/partners/partner-detail.tsx`
- `web/src/pages/partners/components/SettlementSection.tsx`
- `web/src/pages/partners/components/InterestRuleModal.tsx`
