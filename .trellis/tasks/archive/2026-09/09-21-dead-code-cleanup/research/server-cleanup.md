# 后端删除复核

## 方法与边界

对候选在全仓（含测试、生成绑定、Wire、命令和脚本）搜索符号，并检查定义、真实用例入口、接口实现与测试调用。删除范围仅手写 Go，不改 Proto、Ent、生成代码、模块依赖或数据库。其他代理负责依赖与前端。

## 确认删除

| 符号 | 证据与处理 |
| --- | --- |
| `financeBillableFeeIDs` | biz 包私有辅助函数，只有定义，无函数值传递或调用。删除函数。 |
| `businessTypeFromAPI` / `businessTypeToAPI` | service 私有转换函数，只有定义；实际处理器使用其他在用映射。删除两函数，不改契约枚举。 |
| `documentStringPointer` | data 私有函数，只有定义。相邻版本与事件映射保留。 |
| `exchangeRateBaselineScope` / `exchangeRateLocalFirstOrder` | data 私有旧点查包装，无查询调用；实际汇率仓储使用自身的上下文与范围谓词。删除包装。 |
| `baselineOrLocalWhere` / `baselineLocalFirstOrder` | 仅上述两个死包装引用，删除后没有消费。连带删除，文件注释只描述仍使用的列表 shadowing 查询。港口、机场、费用谓词保留。 |
| `commissionListPredicates` | 无消费的单组织薄包装，实际列表与导出使用 `commissionListPredicatesScoped`。仅删除薄包装。 |
| `GetSummaryByOrderID` | biz 接口、data 薄包装和测试桩组成死链；真实 `order_usecase.go` 使用 `GetSummariesByOrderIDs`。删除三个位置和仅该桩方法读取、从未赋值的 `summary` 字段，保留批量入口。 |
| `CommissionUsecase.ListRules` / `CommissionRepo.ListRules` / data 实现 | 生产 `service/settlement_commission.go` 使用 `ListRulesScoped`；唯一旧调用是关闭事务上下文测试。删除旧薄包装与接口声明，将测试调用改为 `ListRulesScoped`，传入同一组织的单元素集合，原拒绝关闭事务断言完整保留。 |
| `ErrOrderUnlockDingTalkDispatchUnknown` | 全仓只有变量声明，不是当前错误分支或契约生成来源。删除变量，不更改钉钉流程。 |
| `invoiceCommandResult` | 测试文件内未实例化、未引用的残留类型。仅删除类型，真实发票事务测试全部保留。 |

## 保留

一次性回填命令、仅测试使用的生产符号、`WithInjectedAuditFailure` 测试钩子、重复实现均不动。未删除任何在用 RPC、仓储查询、测试用例或断言。

## 验证

- 已执行 gofmt（仅修改文件）。
- 删除符号在 server/ 内复扫无残留；`git diff --check` 通过。
- `go -C server test -p 32 ./internal/biz ./internal/data ./internal/service` 通过（biz 1.914s、data 13.762s、service 1.035s）。
- 本次不声明真实 PostgreSQL 集成验证通过；默认测试命令可能跳过未配置专用数据库的用例。
