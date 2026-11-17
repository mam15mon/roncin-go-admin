# 执行计划

当前阶段：用户已批准实施，任务已进入 in_progress。

## 实施顺序

- [x] 用户批准后激活任务；初始工作树干净，分支 codex/retire-shipment-mode，保留其他会话改动。
- [x] 按implement.jsonl向trellis-implement传递当前任务、需求和研究；先改Proto/Schema源，再贯通服务端及前端实现。
- [x] 删除模式枚举、字段和汇总API及专用对象；同步领域、service、data、拆票与幂等路径。
- [x] 生成Ent，提交正式删列迁移；不清理或重建开发库，不改历史迁移。
- [x] 删除四类订单模式UI/适配、汇总抽屉/回调；费用标题统一为“费用明细”。
- [x] make -C server api，然后pnpm run generate:web-client；重生成错误目录，审查生成差异。
- [x] 实现后修改/补充针对性测试、E2E夹具和验收脚本，更新现行规范；禁止先写测试驱动实现。
- [x] 定向验证通过后交trellis-check复核分层、遗漏调用方、生成物及保留能力；修正问题。
- [x] 本次范围标准前后端门禁及隔离库迁移验证通过，源文件及生成物已提交；额外PG整包的20项基线失败见research/baseline-failures.md，不计为通过。归档和会话记录在收尾阶段完成。

## 验证清单

1. 修改文件Biome检查；Vitest定向运行orders、sea-order-policy、海空form-adapter、标签/布局、OrderListTemplate、TagsView相关测试；pnpm --dir web tsc。
2. go -C server test -p 32 ./api/order/v1 ./internal/biz ./internal/service ./internal/data；真实data集成验证通过专用连接与正式迁移执行，不把默认Skip算通过。
3. 独立PostgreSQL校验正式迁移空库链、已有3类模式值升级、订单及关系保留；验证四类订单无模式创建/更新与幂等重放，回归拆票及共享主单。
4. 检查生成客户端无旧字段/旧RPC、HTTP访问规则无旧方法，现行代码无模式消费；历史迁移/任务、企业名称等按研究中的保留清单排除。
5. pnpm run check:fast、git diff --check；未执行或失败检查记录原因。不要求pnpm run build，除非实施出现生产入口／构建配置变更。

## 回归重点

不能因删除汇总RPC就移除LCL主单共享、退关成员和HBL状态的全部验证；调整验收脚本时使用现存主单和订单接口核对底层事实。删除字段会改变Ent查询投影，order_release_pod_test等SQL驱动夹具必须同步，防止“编译通过但查询列错位”。
