# 利息规则界面与业务范围对齐：设计草案

当前设计为待决策草案，不能执行。
推荐方案：只移除未交付UI、成功toast、本地state及无其它消费者的InterestRuleModal，保留真实已有结算规则。实现前rg核对组件消费者；不修改后端、依赖、生成客户端或开发数据。
替代方案：完整计息属于新金融业务契约，涉及.proto/Ent迁移/计算、确认与核销相互关系，当前A03证据不足以推导规则；需另行领域规划，遵守事务/版本字段不明确先确认要求。
该任务与保存保护同时编辑partner-detail.tsx，须串行且在产品选择和最终规划批准后启动。

## 影响文件

- `web/src/pages/partners/partner-detail.tsx`
- `web/src/pages/partners/components/SettlementSection.tsx`
- `web/src/pages/partners/components/InterestRuleModal.tsx`
