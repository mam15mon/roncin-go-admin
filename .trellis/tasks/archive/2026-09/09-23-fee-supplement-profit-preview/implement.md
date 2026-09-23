# 执行计划：审批毛利预览

1. 等待重提子任务完成并提交；以其 `FeeSupplementSection` 版本为基础，保留既有撤回及新增重提交互。定义审批预览 proto 请求/响应与服务端 DTO 映射，按生成顺序刷新 Go/OpenAPI/前端客户端；检查生成差异没有无关漂移。
2. 在 biz/data 实现只读预览：复用实时 grant 与锁依据校验、审批费用事实解析；按有效费用和 decimal 计算本币毛利、零应收空毛利率；不触碰审批写事务。
3. 将 `FeeSupplementSection` 的列表决策入口改为“审核”，在同一视图显示申请事实与预览并承载确认通过/驳回；处理加载、失败、申请失效、订单切换和提交后刷新。
4. 功能实现后补服务端用例/数据层与前端定向测试，覆盖权限、版本冲突、锁依据变化、费用状态筛选、多币种折本币、零应收、预览不写入、驳回及审批相邻路径。
5. 运行 `go -C server test ./internal/biz ./internal/data ./internal/service -run 'FeeSupplement|OrderFeeSupplement' -count=1`、前端对应 Vitest、修改文件 Biome、`pnpm --dir web tsc`、`git diff --check`；跨层契约风险须运行 `pnpm run check:fast`。

风险点：预览不得从申请提交时的旧金额直接估算审批时成本；不得绕过实时资格与锁依据；不得将前端预览值用于正式审批写入。若预览与审批结果出现时间差，保留“预计”标识并以后端审批结果和刷新后的费用列表为准。
