# 信用额度统一本币口径

## 目标与需求

对应父任务 R5：恢复原已确定产品契约：信用额度按当前公司本币维护，与账单既有本币未结余额比较；结算币种可为外币，但不能决定额度币种。禁止把现有外币额度直接重标成本币或查询时隐式换汇。

## 问题证据

server/internal/data/finance_bill_credit.go:104；web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx:395；.trellis/tasks/archive/2026-09/09-11-partner-terms-credit/prd.md 第3/5节明确本币额度且不做实时汇率换算。

## 验收

- [ ] CNY 与非 CNY 公司本币。
- [ ] 外币结算+本币额度。
- [ ] 伪造外币额度写入拒绝。
- [ ] 既有异常额度显式暴露且不自动改值。
- [ ] 嵌套和独立写入口一致。
- [ ] 余额低于/等于/高于额度、零/未设额度、散客、提醒/干预。
- [ ] 查询组织隔离。。

## 交付边界

依赖 R1 后再修改共享 partner 写入口；与 R3 客商页面交叉由同一负责人或主协调串行整合。
不重置数据库、不自动纠正已有业务金额；保留无关工作。按父任务授权完成独立 review、合并主分支和 push。
