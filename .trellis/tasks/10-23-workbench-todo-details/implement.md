# 实施与验证计划

## 启动条件

- [ ] 用户明确批准最新规划摘要后才执行 task.py start；当前不得修改产品代码。
- [ ] 刷新 git 状态、当前任务和分支，保留其他任务改动；在专属分支工作。
- [ ] 加载前后端规范、roncin-web-stack 和 Ant Design 指南。

## 实施步骤

1. [ ] 修改工作台 Proto：两类分页接口；补充原异常响应的可读名称。
2. [ ] 增加 biz 投影、仓储接口、分页校验；同步受影响桩实现。
3. [ ] 完成 service 会话/权限映射，data 共享查询谓词、分页与批量名称装配；摘要同步使用细分权限范围。
4. [ ] `make -C server api`、`pnpm run generate:web-client`，检查生成差异。
5. [ ] 完成工作台入口、两类分页抽屉、错误/空态和刷新。
6. [ ] 完成订单异常入口意图的一次性消费、独立权限及名称显示；保留现有锁门禁。
7. [ ] 功能实现后补充定向验证；独立 review 后修正问题并完成全栈门禁。
8. [ ] 每组可验证改动按 Conventional Commits 提交，更新必要契约说明和验收记录。

## 验证矩阵

| 维度 | 场景 | 验收 |
| --- | --- | --- |
| biz/service | 分页默认值、0/200/201 边界、固定会话身份、业务权限映射 | AC5、AC6 |
| PostgreSQL | 本人/同事、当前/其他公司、SE/AE、无订单/费用/异常读取权限、管理员个人协作 | AC5 |
| PostgreSQL | 无账单、有效草稿、非活动行、已取消账单；ACTIVE/RESOLVED/移除异常 | AC4 |
| PostgreSQL | 同票多项、同时间跨页、名称投影与停用类型、摘要列表同口径 | AC1、AC6、AC7 |
| Web | 类型切换、分页、空态、失败重试、显示格式、SE/AE 链接 | AC1、AC2、AC6、AC7 |
| 路由与权限 | 异常自动展开、参数消费、不重复弹出、已有页签；只读/解决/删除/锁定 | AC2、AC3 |
| 工作区 | 换组织/用户、旧请求迟到、返回工作台和重开后最新计数 | AC4、AC5、AC6 |

开发期先执行直接受影响测试文件及修改文件 Biome，再按类型影响执行 tsc。新测试文件随实现落地：

```bash
go -C server test -p 32 ./internal/biz ./internal/service -run 'Workbench|AbnormalCase'
go -C server test -p 32 ./internal/data -run 'Workbench.*Postgres|OrderAbnormalCase.*Postgres' -count=1
pnpm --dir web exec vitest run src/pages/workbench/WorkbenchSideCards.test.tsx src/pages/workbench/index.test.tsx
pnpm --dir web test:changed
pnpm --dir web tsc
git diff --check
```

集成验证使用 `RONCIN_INTEGRATION_DATABASE_SOURCE`，通过正式迁移建立独立测试 Schema；不输出连接串、不清空开发库，缺配置造成 skip 必须如实记录。重型组件测试超过 20 秒按功能拆分。

最终验收因涉及权限投影、接口及生成物运行 `pnpm run check:fast`，记录耗时/通过与失败数；本次不固定执行生产构建。

## 收尾

- [ ] 变更仅属本任务，无秘密/认证材料/临时产物，生成物无手改。
- [ ] 记录 review、实际测试和未运行验证原因；归档前确认验收完成。
- [ ] 根据用户批准的交付方式处理分支，不携带其他任务改动或删除其分支。
