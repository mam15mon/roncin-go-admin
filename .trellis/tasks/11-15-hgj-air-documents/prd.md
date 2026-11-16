# 航空直单与结构化主分运单

## 目标与范围

父任务：`.trellis/tasks/11-15-hgj-business-alignment`。本子任务只承担以下可独立验收的交付，完整目标由父任务统合。

- 航空Link增加DIRECT/HOUSE；DIRECT一MAWB一票且不混HOUSE，真实无HAWB；HOUSE一票一真实HAWB。
- 独立航空结构化主分内容/不可变版本/哈希/审核快照；旧content完整迁入备注，一次性同步源合同与所有调用方。
- 保留MAWB号规则、HAWB批次永久占号及重量确认来源；运输/单证/本票资料分别写入。
- 补AE公共字段UI/双向映射，制作页后续使用同一内容模型；普通保存不可切模式或换主单。

## 验收标准

- [x] AE的DIRECT/HOUSE全部创建/编辑/锁/终止恢复/重量费用可用；AI基础契约可复用且不借AE权限。
- [x] 直单第二票/混挂、HOUSE缺分单、跨类型ID、版本/锁冲突均拒绝且无半票。
- [x] 旧文本、主分内容和版本历史完整；真实wire零值/optional/清空往返不丢数据。

验证依据：`research/backend-validation.md`、`air-server-review.md`、`air-web-review.md`
及前端分工报告。真实 PostgreSQL、正式迁移、前端/类型检查与最终check:fast通过；
当前组未声称真实浏览器或PDF完成，后续输出及父完整业务财务链仍继续验收。

## 依赖与边界

前置：.trellis/tasks/archive/2026-11/11-15-hgj-order-direction。前置交付未完成不能把树形位置当作已满足依赖。

本期不增加真正多航段、复杂空运拆拼改配、批量导入或另五类实现；不提交海管家单据、不清库、不手改生成物。父最终方案已批准，按前置交付验收顺序start。

实施授权同步：父最终方案已获用户明确批准，按前置依赖执行；具体完成状态以实际验证结果为准。
