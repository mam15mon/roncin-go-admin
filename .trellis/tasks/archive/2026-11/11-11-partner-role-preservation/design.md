# 客商业务身份资料保护设计

## 行为与分层

在 partner.proto 增加专用 UpdatePartnerRoles RPC，沿用现有客商编辑权限和公司范围。service 只做 DTO 转换/鉴权，用例只校验角色集合；data 在现有 WithTx 与目标行 ForUpdate 下仅替换角色和写审计，不调用全量 Partner.Update。复用角色级黑名单及已有角色行，不覆盖结算规则、资料与分工。不新增实体/版本字段或重写全量档案契约。

## 影响文件

server/api/partner/v1/partner.proto；server/internal/service/partner_profile.go；server/internal/biz/partner*.go；server/internal/data/partner*.go；RoleSwitchModal 及相关测试；正式生成的绑定/OpenAPI/前端客户端。

## 依赖与并发协作

独立交付；R3 与本任务有客商入口交集，先完成本任务再审 R3 客商分支；共享生成物统一串行生成。

## 契约、数据与回退

若修改 Proto，正式生成服务端绑定/OpenAPI/客户端，源与生成物同组审阅。现阶段不预设 Schema 变更；如确有需要，先同步 Ent 与正式迁移方案，禁止自动建表或删除数据。回退通过本子任务提交反向提交，不重置主分支；已写业务数据不自动回写。
