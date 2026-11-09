# 实施与验证计划

## 启动条件
用户已在最终摘要后明确批准实施及标签数据清理范围，并要求完成后独立review、合并main、安全删除所有分支、push。2026-11-09 已执行task.py start；仅安全删除已合并分支，不强删仍含未合并工作的分支。

## 交付包与依赖
1. 存储包：移除Ent标签Schema/edges、生成Ent并增加正式退役迁移及迁移前态测试。需要与契约运行包协调，因为业务代码仍引用被移除实体；不得通过临时禁编译/规则豁免交付。
2. 后端包：删除18个RPC及专属类型、reserved编号，删除biz/data/service/注册/provider依赖，同步构造测试；权限manifest退役并生成。生成所有必要Go/Wire/OpenAPI/客户端后，前端依此对齐。
3. 前端包：全局标签入口/状态/候选/列/导出/公共props删除，企业资源去type4分支，账单列表去公司selector绑定当前workspace；不误删普通状态Tag。
4. 主会话：更新AGENTS与spec/能力导航、按交付包提交；独立trellis-check审核整个范围；完整门禁后归档和日志。合并/推送沿用本会话已有授权，先核对远端及其他并行工作，不强推。

原checkout已有其他聊天的收付页面改动和海运布局任务，禁止覆盖或带入本次提交；本任务使用隔离worktree /home/vince/.codex/worktrees/shared-ui-contracts/roncin-go-admin，分支codex/retire-business-tags。代理均显式workdir，清楚分配文件owner，不回退其他人的修改；生成流程只设一个owner，避免同时重写目录。

## 实施顺序与生成
- 先改源Proto/Ent/manifest，再对齐共享运行代码与测试构造；删除独立order_tag源时同步精确处理陈旧生成物，不手修生成内容。
- 按仓库入口执行 go -C server generate / make -C server api / pnpm run generate:web-client / pnpm run generate:permission-keys，按实际需要运行Wire。审查依赖/锁文件无无关升级。
- 前端移除BusinessTag组件后再检查所有正式调用及fixture；更新资源类型切换测试到保留的备注/收货人，继续覆盖迟到隔离/表单正常校验。
- 删标签专属用例不等于删相邻业务验证：费用删除/批量删除仍须断言账单行历史来源清空、金额快照与事务结果；拆票费用归属/守恒等断言保留。
- 对现存运行代码精确搜索BusinessTag/EnterpriseTag/TagGroup/tagIds/tag_ids/fee.tag残留，历史迁移/Proto reserved/历史审计文本不算功能残留；不以所有Tag字符串清零作为验收。

## 定向验证
- 正式迁移隔离库：从退役前前缀SQL建立有标签组/标签/订单费用账单关联/标签客商关联的合法夹具，混合其他企业资源与审计；升级后五专表与TAG行消失，其余业务主键/计数/关键值不变。异常TAG引用导致事务回滚，不CASCADE。冷启动完整SQL序列末态与Ent一致，不用Schema.Create，不在真实开发库做测试。
- 真实权限同步：fee.tag权限及授权消失，其他permission/role/user不变；18个RPC/HTTP注册不存在，核心业务RPC仍存在；Proto reserved反射或契约断言有效。
- 后端相关data/biz/service/server测试/vet，保留订单列表详情/feeCRUD/拆票/账单/企业资源CRUD导入的邻接路径。
- 前端定向测试验证各页面无业务标签入口、请求无标签字段/候选、CSV无标签列、普通状态/候选标签和TagsView仍在；资源地址/备注/图片/主体正常，旧type4不复用。
- 账单在当前公司org-A下列表只请求org-A，不恢复测试注入的org-B旧值；切换工作区后按新身份请求，未就绪不越界。上栏消失但主要搜索/财务汇总/分页/选择/操作不变。
- 修改文件Biome、tsc、git diff --check；重型suite按职责拆分，禁止先写测试的TDD，先实现再风险验证。
- 生成命令重跑幂等；最终 pnpm run check:fast（含架构、权限/枚举生成校验、前后端测试与安全门禁）。真实库用例必须PASS，未配置导致SKIP不能当通过。

## 交付前检查
明确记录正式迁移是否只生成提交、是否获准应用真实库，不能笼统说“数据已清理”；不向输出/Git写入秘密或备份；不改历史迁移；独立review无阻塞后提交中文Conventional Commit，保护并行改动。

## 实施起点
用户授权后恢复隔离worktree，重建codex/retire-business-tags分支并先合入main最新bfcbd9de（包含另一窗口的收付方式展示修复6a3a9c22），保护并行改动。数据库实际应用与否在最终验收单独记录。
