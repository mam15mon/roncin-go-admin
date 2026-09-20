# AI 友好架构二期：后端门禁与去重导航

## 目标
把 AI 友好从「文档约束」升级为「自动门禁 + 低重复 + 可导航」：后端分层越界自动失败；
消除重复报告已确认的真实重复；后端补齐场景→入口导航并纠正文档漂移。

## 已确认背景
前端已有 check:architecture 模块门禁与能力导航；后端 service/biz/data 分层只有文档。
昨日 report:duplicates 确认 8 组候选，其中 5 组判定「可后续提取」、3 组判定合理重复保留
（跨 v1 协议 DTO 映射 ×2、跨层枚举映射）。Go 分层现状：service 无 data/ent、data 无
service、server 无 data、biz 仅 5 个文件为错误码关联导入 proto ErrorReason 枚举
（错误码目录契约，见 domain/error-catalog.md）。用户已选定全部四个方向，批准直接实施。

## 需求
- R1（后端分层门禁）：新增 Go import 边界检查器（仅标准库 AST），按 directory-structure.md
  编码分层规则：service 禁 data/ent/server；biz 禁 service/data/server/platform，api 包仅允许
  ErrorReason 标识符；data 禁 service/server；server 禁 data/ent；access/security/conf 等
  叶子包禁上层；非 _test.go 才检查，cmd 组装根豁免。违规输出文件、行号与规则原因并
  非零退出；接入 check:server 与 CI；现有仓库零违规。
- R2（消除 5 组真实重复）：汇率模板下载（页面模块内提取）、biz 财务组织 ID 校验 ×4
  （包内共享函数）、业务标签列渲染 ×3（先查已有组件再提取）、订单费用候选合并（局部纯
  函数）、汇兑损益展示 ×2（财务小组件）。保持调用方列宽/搜索/操作与订单身份隔离语义；
  不建通用缓存或 DTO 框架。3 组合理重复明确保留不动。
- R3（导航与文档）：新增后端能力导航（场景→biz/服务入口），补 architecture-map 本次
  features 新入口；系统校验 spec 全部相对链接与提到的路径真实存在、无旧符号残留，发现
  即修。重复报告文档同步更新（提取后重跑扫描验证候选消失）。

## 验收标准
- AC1（R1）：每条规则有独立违规样例失败的正反例测试；ErrorReason 例外精确（非
  ErrorReason 的 api 标识符仍报错）；现有仓库扫描零违规；check:server 与 CI 生效。
- AC2（R2）：5 组重复全部消除且功能回归通过（受影响定向测试 + tsc + biome + 架构门禁）；
  重跑 report:duplicates 对应组消失、无新增解析错误；3 组保留项不受影响。
- AC3（R3）：后端能力导航覆盖主要业务域且入口文件真实存在；spec 链接/路径校验通过；
  文档无 utils/options、order-options-cache 等已删符号残留。
- AC4：分组提交（feat/docs/refactor 分开），每组 Conventional Commit 中文说明；最终
  check:fast 通过；不改业务行为、API 契约、数据库与权限。

## 范围外
不自动合并其余重复候选；不引入 golangci-lint 等新依赖；不改前端模块边界规则本身；
不做后端代码大规模重构；不改业务权限、API、数据库。

## 状态
用户已选定全部方向并批准实施（2026-09-20 会话）；主会话直接实现，不走子代理。
