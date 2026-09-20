# 技术设计

## 分组A：后端分层门禁（scripts/layer-check-go/）

独立 Go 模块（自带 go.mod，仅标准库 go/parser、go/ast、go/token），模式对齐
scripts/duplicate-scan-go；入口 `go -C scripts/layer-check-go run . <serverRoot>`，
默认仓库内 server/。逐文件解析 import 与限定标识符，输出违规（文件:行 + 规则名 +
原因）到 stderr，违规或解析错误非零退出，零违规退出 0。

### 层规则（对齐 .trellis/spec/server/backend/directory-structure.md，现状全绿）

| 包前缀 (internal/) | 允许 internal 依赖 | 允许 server/api 依赖 | 关键禁令 |
| --- | --- | --- | --- |
| service | biz, access, platform, security | 是（DTO 转换必需） | data（含 data/ent）、server |
| biz | access, security | 仅 ErrorReason 标识符（5 文件既有错误码契约） | service、data、server、platform |
| data | biz, data/ent（自有）, access, conf, platform | 否 | service、server |
| server | service, biz, access, platform, conf, webassets | 是（注册需要） | data（含 data/ent） |
| platform | conf | 否 | data、biz、service、server（_test.go 豁免迁移集成测试） |
| access/security/conf | （无） | 否 | 任何 internal 上层包 |

- cmd/ 为组装根，豁免；全部规则只检查非 `_test.go` 文件（测试可建夹具跨层）。
- biz 的 ErrorReason 例外在标识符级实现：对导入路径以 `server/api/` 开头的包别名，
  记录其 SelectorExpr 使用；`ErrorReason_` 前缀常量与 `ErrorReason` 类型本身放行，
  其余选择符报「biz 泄漏 Protobuf 类型」。允许名单集中在规则表常量中，不散落。
- 检查器自身测试（go test）：每条禁令一个正反例夹具（TempDir 构造 mini 模块树）、
  ErrorReason 精确性（用 api 包其他类型仍报错）、_test.go 豁免、cmd 豁免、解析错误
  非零退出、多模块前缀匹配（internal/data/ent 也命中 data 规则）。

### 接线
package.json 新增 `check:layers:go = go -C scripts/layer-check-go test ./... && go -C scripts/layer-check-go run .`；
`check:server` 链尾追加该命令；CI 服务端 job 追加。不改 check:fast 结构
（其内部调 check:server 自动获得）。

## 分组B：消除 5 组真实重复

| 组 | 提取方案 | 归属 |
| --- | --- | --- |
| 汇率模板下载 handleDownloadTemplate ×2 | 页面模块内共享函数（loading/message 由调用方传入） | pages/finance/exchange-rates/components/ 模块内部 |
| biz 财务组织 ID 校验 ×4 | 包内共享 `validateFinanceOrganizationIDs(ids []uuid.UUID, label string) ([]uuid.UUID, error)` 风格函数，四处调用替换 | server/internal/biz（新文件或就近 finance 公共文件） |
| 业务标签列渲染 ×3 | 先查现有 business-tag 组件（PartnerSelectOptionTags 同层先例），无则提取小组件 | 跨 finance/bills、finance/fees、orders → features 按业务标签领域归属（实现时定，倾向 features/finance/business-tags 或已有目录） |
| 订单费用候选合并 ×2 | 同文件提取局部纯合并函数，保持订单身份闭包参数 | pages/orders/use-order-fee-options.ts 文件内 |
| 汇兑损益展示 ×2 | 提取财务金额展示小组件（正负金额+币种） | features/finance（nettings 与 verifications 共用） |

约束：调用方列宽、搜索配置、页面操作不动；订单身份隔离语义不变；biz 提取只合并
相同纯校验，不动权限/数据范围判定；3 组合理重复（DTO 映射 ×2、枚举映射）保留并在
提交说明中注明。完成后重跑 report:duplicates 验证 5 组消失、其余不变。

## 分组C：导航与文档纠错

- 新增 `.trellis/spec/server/backend/capability-navigation.md`：场景→入口表（订单/费用/
  账单/核销/对冲/收付/提成/汇率/主数据/权限/审计），入口为 biz 文件或 service 文件真实
  路径；接入 backend index.md。
- architecture-map.md 补本次 features 新入口（orders/options、master-data/currencies、
  shipping-lines、partners/credit-control 扩展、types/select-option）。
- 文档纠错扫描：脚本化校验 .trellis/spec/**/*.md 的相对链接目标存在；对已知重命名符号
  （utils/options、order-options-cache 等）全库 grep；人工抽查导航表入口路径存在性。
  发现即修，单独 docs 提交。

## 风险与回退
门禁规则编码过严会阻断正常开发——规则表以「现状全绿」为准，新禁令先验证零违规再上；
按分组 A/B/C 三次提交，可独立回退；无数据迁移、无契约变更。
