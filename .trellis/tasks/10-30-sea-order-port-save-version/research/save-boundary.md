# 保存边界排查

## 因果链

1. `buildSeaExportDetailInitialValues` 与 `useSeaMasterBillCandidate` 把主单版本放入 Form store，字段未注册。
2. `OrderFormTemplate.handleFinish` 原样传递 `onFinish(values)`；`detail.tsx.handleSaveEdit` 只给订单版本和单证内容补取完整 store，没有给主单身份输入补取。
3. `buildSeaExportUpdatePayload` 仅从 values 中读取主单版本，因此请求中的 `expectedCandidateVersion` 缺失。
4. 单票 DRAFT 主单的航程变化进入后端 `order_write_sea_master_bill.go:511`，版本缺失即返回截图错误。

## 实施注意

- 在海运详情本地边界选择最小修复，避免改动全站模板提交语义。
- 必须验证真实 Form 的未注册字段丢失，不能手工调用提交回调并传入完整假载荷来代替。
- 检查 Hook 会否在恢复草稿、刷新后把旧版本改成最新查询版本；如发现直接影响本缺陷的漂移，随本任务修复并记录。
- 现有 `detail-draft-lifecycle.test.tsx` 使用真实 antd Form 加模板替身，可参考测试基础，但文件已有多个生命周期用例，新增用例应避免重型单文件长尾。
- 实现先于新增测试，遵守禁止 TDD。
- 后端规则保持不变，真实版本冲突通过前端拒绝响应及输入保留验证。

## 预计验证

实现后运行新增港口保存回归、详情草稿生命周期、主单候选身份/继承、表单适配器及保存 Hook 的受影响定向测试；随后按风险运行 Biome、tsc 和一次 `check:web`。
