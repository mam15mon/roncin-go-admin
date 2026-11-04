# 实施计划

## 开始条件

- [x] 用户要求增加公司英文名并支持各公司自签 HBL。
- [x] 用户批准创建任务及规划。
- [x] 仓库现状研究、PRD、设计和子代理规范清单已准备。
- [ ] 最终规划摘要后收到用户明确实现确认，再执行 task.py start。

## 1 后端和生成物

派发 trellis-implement 负责 server/** 及生成的 web 客户端；派发首行带 Active task，使用原生上下文注入，必要时子侧拉取。所有代理明确其他人也在修改仓库，不回退他人工作。首次实现从完整 PRD/设计和相关规范加载开始。

上下文清单验证通过；component-guidelines.md 与 state-management.md 超过单文件注入上限，派发时必须要求子代理从工作区完整读取这两份规范，不修改注入上限或裁剪源文件。

- [ ] 公司 name_en Schema、DO、DTO 与新建/更新/列表映射、校验、审计及检索键。
- [ ] 同步 UpdateOrganization 的全部调用方和测试；不增加组织版本字段或新事务。
- [ ] 当前 HBL 投影真实签发公司英文名，自签公司通用模板标识替换 BJ 限制。
- [ ] Ent/API/OpenAPI 正规生成、正式迁移生成与北京数据补齐，默认北京种子仅影响新建。
- [ ] make -C server api、go -C server generate、pnpm run generate:web-client；检查生成差异。
- [ ] 实现后补充 biz/service 定向以及真实 PostgreSQL 迁移/往返/权限和审计原子性验证。

## 2 公司管理与 HBL 前端

契约稳定后派发前端 trellis-implement 负责手写 web/**，不碰生成物；以现有组织管理和 HBL 组件为边界。

- [ ] 公司创建/编辑 nameEn、清空回显、详情展示及本地搜索；仅 company 可维护。
- [ ] HBL 通用自签适用性、缺名输出提示及可编辑保存分离。
- [ ] HblPdfModel 贯穿真实抬头，移除全部 PDF 北京常量，加入公司名字符支持检查。
- [ ] 公司名/签发 ID/模板纳入变动指纹，输出前复验，保持输入与预览资源清理。
- [ ] 长公司名页眉与签发区正常分页，保留四版本和常规单页。
- [ ] 实现后补针对性测试：公司创建/编辑/清空、非公司、缺名、不同公司、主数据变化、锁定输出、长名称与长正文。

## 3 验证和交付

- [ ] 最小验证：go test 对应 biz/service/data；PostgreSQL 独立 Schema，无静默 Skip。
- [ ] 前端定向 vitest run、修改文件 Biome、tsc、git diff --check；单 Suite 重型渲染不超过 10 个且超过 20 秒拆分。
- [ ] 实际 PDF 文本/坐标/图片检查，北京、人工测试公司、200 字符名称、缺名阻止及四版本。
- [ ] 最后一次 trellis-check 覆盖任务全部受影响包及完整跨层链路，局部问题自修，设计问题回交主会话。
- [ ] 最终 pnpm run check:fast、pnpm run build；浏览器公司维护与 HBL 输出闭环、Go 同域静态资源。
- [ ] 更新公司英文名与 HBL 规范，准确说明新模板契约替换旧 BJ 范围。
- [ ] Conventional Commit 包含源与生成物、正式迁移及验证记录，保护原有未跟踪排版任务。
- [ ] 完成验收后按当前授权记录会话及归档；远端推送不由本规划自动执行。
