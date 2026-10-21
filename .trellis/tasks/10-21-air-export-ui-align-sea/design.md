# 技术设计：空运出口表单 UI 对齐海运出口信息架构

## 1. 现状与目标信息架构映射

| 海运出口（基准） | 空运现状 | 空运目标 |
| --- | --- | --- |
| 卡1 业务与客户：编号/编号时间/客户参考号/内部参考号、委托单位、服务类型 | 同左 + 订舱号混在首行 | 同海运（订舱号移出） |
| 卡2 订舱与运输：小节「订舱与主单」「航线与船期」「箱量与截关」+ 折叠备注（订舱/配舱/操作） | 卡2「航空运单与共同运输」平铺 20+ 字段 | 卡2 订舱与运输：小节「主单与订舱」「航线与班期」「时刻与申报」+ 主单备注（折叠）+ 操作备注（折叠 sectionHeading） |
| 卡3 货物与商业（贸易/合同/货值/保险/托运件重尺） | 卡3 本票货物（品类/描述/件重尺/操作备注平铺） | 卡3 本票货物（品类/描述/件重尺；操作备注移卡2 尾部） |
| 卡4 单证（提单模式/主单/分单页签） | HAWB 混在卡2 | 卡4 分单信息：HAWB 号/分单状态/单证资料/（详情已签发）修改原因 |
| 卡5 人员 | 卡4 人员 | 卡5 人员（共用 `buildSeaPersonnelSection`，不动） |

## 2. 目标卡片内部布局

卡2「订舱与运输」小节划分（对齐海运 `SeaFieldGroup` 分组语义）：

- **主单与订舱**：主单关联方式（新建/关联已有，仅新建）、共享主单选择与
  关联提示（仅新建 EXISTING）、共享主单摘要 Alert（仅详情）、订舱号
  `bookingNo`、（NEW 模式）MAWB + 航空公司 + 起运机场 + 目的机场。
- **航线与班期**：（NEW 模式）航班号 + 计划起飞 + 计划到达 + 实际起飞 +
  实际到达。
- **时刻与申报**：（NEW 模式）截单时间 + 报关截止 + 主单单证状态 + 主单
  申报件数/毛重/体积。
- 主单备注 `['airMaster','notes']` → `SeaCollapsibleNotesField`。
- 操作备注 `operationNotes` → `SeaCollapsibleNotesField sectionHeading`。

卡4「分单信息」：`airHouseNo` / `airHouseStatus` /
`airHouseContent`（FormRow）+ 详情已签发时 `airHouseChangeReason`，
逻辑与现状一致仅迁移位置。

## 3. 组件改动

### 3.1 提取共享小节标题组件

`sea-template.tsx` 私有的 `SeaFieldGroup`（品牌蓝竖标 + 标题）提取为
`web/src/pages/orders/templates/components/FieldGroup.tsx`（导出
`TemplateFieldGroup`）。`sea-template.tsx` 改为引用该组件，删除本地实现，
渲染结果与现状逐像素一致（纯搬移）。air-template 直接复用。

不放入 `@/components/ui`：该组件只在订单模板家族内使用，属于页面域而非
通用 UI（符合 AGENTS.md 的 features/ui 边界）。

### 3.2 拆分 AirMasterFields 为可分组子件

现状 `AirMasterFields.tsx` 内部固定 5 组 FormRow 平铺，被两处消费：

- `air-template.tsx`（订单表单，prefix=`['airMaster']`）
- `air-master.tsx` 工作台编辑弹窗（prefix=`['input']`，传 `master`）

拆分为同文件导出的三个子件（props 形状沿用现有 `props` + `prefix` +
`master`）：

- `AirMasterRouteFields`：MAWB、航空公司、起运/目的机场、航班号。
- `AirMasterScheduleFields`：计划起飞/到达、实际起飞/到达。
- `AirMasterDeclarationFields`：截单、报关截止、主单状态、申报件数/毛重/
  体积。

`AirMasterFields` 保持原签名与原平铺组合（工作台零变化）；air-template 改
为用 `TemplateFieldGroup` 包裹子件自行分组。隐藏字段（`airMaster` /
`airExport` / `airMasterId` / `airMasterExpectedVersion`）、`AirMasterIdentityFields`
的 NEW/EXISTING 分支、`AirMasterCandidateHint`（同号主单提示）与
masterId 预选逻辑原样保留，归入「主单与订舱」小节。

### 3.3 重组 air-template.tsx

`getAirTemplateSections` 返回五节（key：`businessCustomer` /
`bookingTransport` / `cargoCommercial` / `airDocuments` / 人员节沿用
`buildSeaPersonnelSection` 的 key）。要点：

- 卡1 首行去掉 `bookingNo`，其余不动。
- 备注从 `ProFormTextArea` 换 `SeaCollapsibleNotesField`；错误自动展开
  依赖 `onSeaFormErrorReveal` 订阅，`new.tsx` / `detail.tsx` 已对全部订单
  kind 统一传入 `revealSeaFormErrors`，空运无需新增接线。
- `templates/index.ts` 如需对外暴露新组件则同步导出。

## 4. 行为与风险控制

- **字段值契约**：所有 Form.Item `name` 不变，仅调整挂载位置；create/update
  payload 由 `form-adapter.ts` 生成，该文件不动。
- **锚点导航**：`OrderFormTemplate` 按 sections 数组驱动，自动跟随新分节。
- **草稿恢复**：`formDraftKey` 由 tabKey+pathname+scope 决定，与分节无关；
  旧草稿字段在新布局下依旧回填（name 未变）。
- **海运零回归**：sea-template 仅替换 `SeaFieldGroup` 为共享组件，跑海运
  全部既有模板测试锁定。
- **风险点**：`AirMasterFields` 拆件时保持每个字段 props 逐字不变
  （rules/request/options/disabled 表达式），避免校验与联动回归；工作台
  弹窗渲染路径用 `air-master.permissions.test.tsx` 锁定。

## 5. 回滚

纯前端、单仓库、无数据迁移；任一步异常直接 `git revert` 对应提交即可，
无运行时状态残留（表单草稿与分节无关）。
