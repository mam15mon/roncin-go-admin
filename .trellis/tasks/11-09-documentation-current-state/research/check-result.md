# 文档现状同步独立复核

- 日期：2026-11-09。
- 角色：独立 trellis-check；先复核已完成的前端、领域与 ADR，收到全范围完成通知后复核通用及后端组。
- 依据：任务 PRD、设计、执行计划、各域审计和更新记录，当前 Git 差异及相关 Proto、Schema、实现、生成脚本、CLI 帮助。
- 仅修改明确的局部文档事实；没有修改业务源码、配置、数据库、契约、生成物或第三方许可，没有 Git 暂存、提交或分支操作。

## 已修复

| 文件 | 问题 | 修正与源码依据 |
| --- | --- | --- |
| `README.md`、`docs/migration-matrix.md`、`.trellis/spec/server/backend/index.md` | 新摘要用“共享 MAWB/HAWB”容易将每票 HAWB 误认为共享单证；海运摘要同样含混 | 明确共享 MAWB/MBL 与每票 HAWB/HBL。空运 `air_house_waybill.go:29` 对 order_id 唯一；海运当前单值 HBL 契约及 Proto 与此一致 |
| `README.md` | 将 GIN 模糊检索表述为全文检索，混淆项目实际索引机制 | 改为 pg_trgm 与 GIN 模糊检索，依据 `20260827133000_selector_search_keywords.sql` 及后续 gin_trgm_ops 索引 |
| `docs/local-development.md` | 环境示例仍将系统工作台命名为“总部”，与当前身份说明及模板不一致 | 改为“系统管理”，与 `.env.example` 的 BOOTSTRAP_ORGANIZATION_NAME 一致 |
| `.trellis/spec/server/backend/quality-guidelines.md` | 新生成物清单列枚举而漏同命令输出的错误原因常量 | 补 `web/src/errorReasons.generated.ts`，依据 `scripts/generate-proto-constants.mjs:106-108` |

## 重点复核结论

- 前端 README/CLAUDE 已采用 Vite/React Router、当前权限能力及 pnpm 入口，没有恢复上游 Umi 架构；两个项目技能正确保护 Roncin 生成目录和现行工作区。技能的外部 issue 发送仍明确以用户授权为前提。
- 主详情未传 draftPathname，模板缺完整身份不读写持久草稿；新建/HBL 保留持久化。新文案与 dirty、关闭/路由离开、保存失败保留输入及旧版本基线的实际职责一致，没有通过文档削弱并发保护。
- ETA 规则对照 SeaScheduleDateFields：依赖 ETD 校验并保留输入，不存在原文宣称的静默清空。列设置例外引用既有明确审核全列规范，没有普遍关闭列设置要求。
- 契约生成顺序与 Makefile/Buf/前端生成脚本一致；实际枚举及错误常量输出齐全。Go generate 示例补 ./... 与当前 server 根目录无 Go 文件的事实一致。
- 后端审核锁、建账准入与源费待审粒度、NETTING 生成草稿对冲、当前有效替代费及历史冲回保护，均按相关 biz/data 路径复核；没有恢复独立费用预审或提成财务锁。
- Switch 旧 RPC/表与 REPLACED/SWITCH 枚举已按正式迁移和 Proto reserved 区分；现行模式切换签名、DIRECT/HOUSE 两态、外部确认字段与源码一致。历史 Manifest 仍存在 switch 名称的事实没有被抹去，也没有因此恢复代码。
- 全局船公司/机场/港口与公司经营资料分开；系统工作台治理资格不能代替公司经营准入。唯一初始管理员通过真实成员补齐进入公司，与普通系统管理员区分。
- 文档导航及历史计划明确时间语义；客户页面静态行政区划缺口仍如实记录，没有把所有旧计划项勾为已完成。历史验收和第三方部署状态没有被补造成本轮通过。

## 未修复项及原因

没有发现本次文档交付范围内仍需阻断提交的问题。

- `web/src/pages/orders/order-kinds/registry.ts` 仍有 AE 未注册的旧源码注释，但实际注册了 AE；属于已记录的范围外源码注释，未借纯文档任务修改业务代码。
- 客户页面仍使用静态行政区划来源属于实现现状/后续产品范围；本次已在历史计划中明确，不修改实现来迁就文档。
- 归档任务、会话日志、第三方许可与框架模板保留原始语义，不能把未重跑的历史验收或外部系统状态宣称为当前验证结果。

## 验证

| 检查 | 结果 |
| --- | --- |
| `git diff --check` | 通过 |
| MarkdownIt 解析的现行说明、规范及项目技能本地链接/锚点 | 124 份文档、397 个目标，失效 0；含引用式链接，远程 URL 不访问 |
| 显式 pnpm run 名称静态核对 | 188 处引用、33 种脚本，根/web package.json 中均存在；不执行写库命令 |
| 当前本地 antd changelog/migrate/doc --help | 只读执行通过；参数语法与技能示例一致；设置 ANTD_NO_AUTO_REPORT=1，没有提交或升级 |
| `node scripts/generate-error-catalog.mjs --check` | 通过：403 个唯一错误码；未手工改生成目录 |
| 变更范围 | Markdown 与当前 task.json 状态元数据；无非文档产品改动 |
| 用户原未跟踪任务 | 4 个原文件 SHA-256 与实施前一致，完整保留 |
| Lint / TypeCheck / 业务测试 / 构建 | 不适用，按已批准纯文档方案未运行；不记为“通过” |
| 数据库、OSS、钉钉及正式部署验收 | 未执行，无相应通过声明 |

链接扫描脚本位于 /tmp，仅用作本轮临时只读核查，不增加项目脚本或依赖。验证对应本轮最终复核时点；主会话负责提交前最终差异检查、按组提交与归档。
