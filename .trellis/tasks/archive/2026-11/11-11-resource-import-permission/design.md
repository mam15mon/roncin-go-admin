# 资源覆盖导入权限边界设计

## 行为与分层

导入 RPC 保留 create 基础权限，覆盖参数为 true 时在传输边界复用现有权限能力检查额外要求 update，覆盖实际写入路径没有仅依赖前端的放行。预览的 overwriteAllowed 与前端确认按钮结合真实 update 权限；纯新建导入仍开放。复用原事务与匹配规则，不改 CSV/Excel 格式，不扩大主权限依赖。

## 影响文件

server/internal/service/enterprise_resource.go 及权限测试；web/src/pages/enterprise-resources/index.tsx 与导入用例；必要时修正该接口权限投影但不手改生成物。

## 依赖与并发协作

独立交付；web/src/pages/enterprise-resources/index.tsx 同 R3 相交，主协调者串行整合该文件或由同一负责人修改。

## 契约、数据与回退

若修改 Proto，正式生成服务端绑定/OpenAPI/客户端，源与生成物同组审阅。现阶段不预设 Schema 变更；如确有需要，先同步 Ent 与正式迁移方案，禁止自动建表或删除数据。回退通过本子任务提交反向提交，不重置主分支；已写业务数据不自动回写。
