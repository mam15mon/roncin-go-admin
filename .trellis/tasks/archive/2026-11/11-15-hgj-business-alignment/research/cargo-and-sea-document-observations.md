# 调研：四类箱货、海运进口单证制作与输出模板观察

- 问题：四类箱货如何表达委托、实际、批次、箱与货；SI 的 MBL/HBL 制作字段有何差异；输出模板弹窗提供哪些类型和格式？
- 范围：离线解析主会话既有 12 份 REA 成功场景；不操作浏览器、外站、凭据，不修改产品代码，不保存单据或生成文件。
- 日期：2026-11-16（北京时间）。

## 证据索引

原始文件只保留在 `/home/vince/.local/share/rea/hgj/order-research/`，其中包含真实客户与操作票信息；仓库文档只保留字段、规则、记录指纹和局限，不保留业务 ID、主分单号、客户/人员或联系方式。

| 简写 | 原始文件 | REA 证据 | SHA-256 | 丢弃事件数 |
| --- | --- | --- | --- | --- |
| SE-CARGO | `sea-export-cargo-readonly.evidence.json` | `ev_f0c6a0a0f647642a245eef472aba40cfdce2e2604a9c6da1d9d83fbea25908e5` | `e67d66d3cf91793501e77debeb667823d1b999db05adec97b327b2856bf4865b` | 3020 |
| SI-CARGO | `sea-import-cargo-readonly.evidence.json` | `ev_d47cc36fccda125efd81260b3e89096133a87382f074b0c2dd4d7563cbefc9ac` | `0c887ab920adb95812c7154c8aa314ac89178a925abd5ec164c2955d8d822867` | 2700 |
| AE-CARGO | `air-export-cargo-readonly.evidence.json` | `ev_44fd24a8f4c7d30abc3767adee7020953a00fed7c8422ff859cdb4287777ba51` | `e3909071eaa94fb051959dd10ffeaeb44c4ba2f21cf10a216a5205971a36274a` | 2829 |
| AI-CARGO | `air-import-cargo-readonly.evidence.json` | `ev_6915ad13df2853d1cb7e8cf13e050a3a040a1f57ec30cf658930f54dacb39070` | `b0ea1edc639c73d5d6f110901701f44856f054eb2b8152d1c6e50401b66b661c` | 2656 |
| SI-DOC | `sea-import-document-list.evidence.json` | `ev_48fcf0ab89b4a0d6e64ef6d69a88363fe6749b8d2355cb4bcbb7f5497937cba8` | `c099dbf1c268f58cdeda29ffd1f15a8f1f863985a4adb1af90b15cb3d7673713` | 2193 |
| SI-MBL | `si-mbl-empty-visible.evidence.json` | `ev_4f9e3330631bfccc4b1449ee8fd079c9593cf4f9d7d6f9f5da96bb1a9027d54b` | `51628d82aff2a6c16c12c1e8b3806016b74471dd3a386811a5f1a3a77f74f0eb` | 0 |
| SI-HBL | `si-hbl-empty-visible.evidence.json` | `ev_08abbc646e071a8473f6d4c073a3f96713abe2652193c01e3ae1125fc2aefadc` | `e769a97e1c431a90b4488630ef047e28a775935f877f7f09907c61f07b478afb` | 0 |
| SE-TPL | `sea-export-export-template-list.evidence.json` | `ev_76682ebaa4a6e2590d9b24674cab512134139b984f2c09fba25ffc0160b33e34` | `de7b203169d8483efa41559f38cf605d8a57ad9686f28ad5d4939608d9c7cdbd` | 3095 |
| SI-TPL | `sea-import-export-template-list.evidence.json` | `ev_fccb44404494e055ee6723da258af4fc20aa787edba75e7af45cd65ea28b20ed` | `5e172f23b64991238b451d51ed6e30a8d1ccb3200d80408cff510d9d5913f87a` | 3202 |
| AE-TPL | `air-export-export-template-list.evidence.json` | `ev_332e046a59d9429ade1e4989447e9f6fe425f7bd53fb5c73c074e28302c2b5e3` | `ef5f5797b94321f8c4aad3b3bf92517db93da099634307ad470d767c223fb5bb` | 2868 |
| AI-TPL | `air-import-export-template-list.evidence.json` | `ev_6dbd7de06570c1e1c21e8d53ffa2621f98ab1df88e5144f37ce7d134a52547b1` | `27da18c8b80728c54d365d0342306788897b5de20a3b1de398f7707cd25ba2cf` | 2685 |
| AI-TPL-ISO | `ai-export-template-isolated.evidence.json` | `ev_18d61bbd94c9e783d7957ae7ce4a1a9b3cb98b87c3325bd21280482b553c419a` | `0ae4055bb1b80537f921cd2e845b1fcf372bdfe539639da0d65637906e16fbe3` | 0 |

原 11 份场景的动作均 completed，最终 DOM 已捕获；四类箱货和模板分别只执行打开入口与等待。SI-MBL/SI-HBL 只展开新增选项并打开未保存草稿；没有单证保存、复制、删除、设置写入或导出动作。

12 份证据总体均为 `completeness.status=incomplete`、缺少 events，不因步骤完成而称整条取证完整。大量丢弃来自 `max-total-event-allocation-bytes`；正文仅用保留的 DOM 证明页面字段，用实际保留的网络记录证明具体请求，不从未记录推断未发生。

## 四类箱货界面分区

| 分区 | 四类共同内容 | 海运/空运差异与限制 |
| --- | --- | --- |
| 运输/仓储 | 顶部“剩余”件数/包装单位、KGS、CBM；带入；批次与新增；车辆、车队/仓库 | 海运另有箱型箱量、关联箱号/查看箱号；空运样本未显示相同箱列表操作面 |
| 批次基础 | 批次名称、编辑、件数/单位、毛重/体积、进仓编号 | 四类均出现“批次1”；这可由前端初始化产生，不能当作已有持久化批次的证据 |
| 批次地址与备注 | 拆/装箱地址及时间、提货地址及时间、送货地址及时间、运输委托备注、备注图片选择 | 地址和时间存在不等于已派车、仓库确认或发出运输委托 |
| 箱/货工作区 | 总件数/包装单位、总毛重 KGS、总体积 CBM，另有进仓信息 | 两海运默认箱信息/箱列表；两空运默认货信息；SE/AE/AI DOM 还有自拼货信息或货信息带入箱列表入口，未执行 |
| 海运箱表 | 箱号、封号、关联提单号、箱型、分件数/单位、分毛重 KGS、分体积 CBM、箱皮重 KGS、VGM 总重 KGS、VGM 联系人、称重方式/时间、备注 | 观察样本没有实际箱行；列名存在不证明行校验、VGM 公式或关联后的同步语义 |
| 汇总 | 总计和勾选总计 | 未选择、未新增，不能验证跨页合计、不同包装单位混合或源数据一致性 |

### 委托、实际与来源的证据分级

- **动态 DOM**：订单详情旁的箱货页展示“剩余”、顶部箱型需求摘要、下方箱货总件重尺，以及独立进仓字段。样本中可出现委托数量/需求摘要非空、实际箱表/汇总仍为空或零；文档不复制具体值。
- **不能据此断言**：空箱表不等于此订单无委托货物；“剩余”零不等于已运输完；默认“批次1”不等于服务器已有批次；无表格行不等于全公司从未存货。
- **静态交叉参考**：`cargo-document-navigation.md` 的 BoxInfo/BatchInfo 初始化链读取订单、箱货及字段设计，装载批次、委托量并计算剩余，海运初始化箱页签、空运初始化货页签；这是客户端代码观察，不是本次响应值/公式对照。
- **实际与委托的正式关联未知**：未点击带入、获取箱货数据、货信息带入箱列表、接收关联、改批次或保存；不能证明同步是单次/持续、来源优先级或修改是否覆盖人工值。

### 箱货控件与单位

| 字段 | HTML 观察 | 边界 |
| --- | --- | --- |
| 批次名称 | text，maxlength=25，当前 disabled，旁有编辑 | 未点击编辑，不推定后端名称不可改 |
| 件数/单位 | number + Select 内部 readonly text | readonly 不等于整个 Select 不可选；数值精度/下限未验证 |
| 毛重/体积 | number，标签 KGS / CBM | 无 maxlength 不代表无上限或允许负数 |
| 进仓编号 | text，maxlength=100 | 只是 HTML 长度限制 |
| 三种详细地址 | text，maxlength=200，配有对应日期/时间输入 | 不复制真实地址；未验证时区/时间先后 |
| 运输委托备注 | textarea，maxlength=1000 | 图片选择入口未执行，不验证存储/附件权限 |
| 箱货总毛重/总体积 | number，disabled | UI 禁用意味着当前界面不直接编辑，后端来源和精度仍待确认 |
| 进仓件数/单位/毛重/体积 | DOM 内有独立控件 | 对 SE 样本检查到祖先 `display:none`，不能当当前箱页签可编辑项；其余同类切换未实测 |

离线用 HTMLParser 按最近字段标签与控件祖先关联；没有计算 CSS、测量可见边界框或逐控件测试。侧栏/聊天/OCR 的输入不当作箱货业务字段。

## 海运进口单证制作列表

SI-DOC 实际出现类型“提单(MBL)”和“提单(HBL)”，新增、类型过滤、批量导出；列为序号、单证类型、名称、主单/分单号、创建人/时间和操作。该样本列表暂无行。

- 列表为空只说明当前样本/当前账号返回的可见列表，不能推出全公司无单证或服务器只允许一份。
- 本次 SI 列表没有取得装箱单入口的可见证据；静态文档中的通用 `clickAdd(3)` 不能自动当作 SI 运行期可用功能。
- 批量导出按钮存在，不代表无勾选也可生成；未勾选/点击，未验证批量类型约束。

## SI 未保存 MBL/HBL 制作表单

两份制作页都是未保存草稿；可以带入原单的关联候选与印字文本，因此“新增空表单”不等于所有字段空白。所有现有值均不进入本报告。

| 维度 | MBL | HBL |
| --- | --- | --- |
| 标题/复制 | MBL制作、复制MBL | HBL制作、复制HBL |
| 主身份标签 | 主单号 | 分单号 |
| 关联 | 主单号关联、分单号关联 | 分单号关联、主单号关联 |
| 详情收发通带入入口 | 本份 DOM 未见相同文案 | 出现“带入详情页收发通”；未执行 |
| 公共字段 | 起运/目的港、船公司、签发日期、收发通知人、国外代理、唛头、英文品名、付款/运输/提单条款、件重尺、备注 | 同左 |
| 共同动作文案 | 返回、保存、复制、删除、提单管理 | 同左；出现不等于 id=0 时均可用 |

### 制作字段与限制

| 字段 | DOM 控件/限制 |
| --- | --- |
| 提单名称 | text，maxlength=50 |
| 关联号码选择 | Select 内部 readonly text；不能推断不可切换 |
| 起运港/目的港 | text，各 maxlength=100 |
| 船公司 | 输入/选择控件，可挂载完整隐藏选项；不复制候选公司的值 |
| 签发日期 | 日期组件内部 text；保存格式/时区未验证 |
| 发货人/收货人/通知人/第二通知人 | 四个 textarea 搜索提示，maxlength=1000；通知人分区同时出现标签/页签，需实际交互核查展示切换 |
| 国外代理 | textarea，maxlength=1000；不同于业务代理 ID |
| 唛头 | textarea，maxlength=10000 |
| 英文品名 | textarea，maxlength=5000 |
| 免责条款 | checkbox + 当前 disabled text（maxlength=100）；未切换 |
| 付款方式 | Select 候选有英文预付、到付、目的地付款等及中文别名；隐藏候选不表示当前选中 |
| 提单形式 | Select，当前样本存在空候选提示；未知是否受配置/账号限制 |
| 运输条款 | text，maxlength=50 |
| 件数/单位、毛重、体积 | number / Select，单位 KGS、CBM；未验证实际来源、合计、精度、必填 |
| 自动带入提单数据 | 勾选入口；未执行，不推断它是否保存设置或持续同步 |
| 提单备注 | textarea，maxlength=1000 |

### 单证与箱货关联的运行/静态边界

- SI-MBL/SI-HBL 保留的网络中实际有 GET `/saas-service/order/billLading/billMultiList`、GET `/saas-service/order/billLading/getBillOfLadingNos`、POST `/saas-service/order/billLading/getBoxs`。POST 查询不能直接当成单证保存。
- 静态参考 `cargo-document-navigation.md`：billId=0 的 createBill 是本地草稿初始化，takeInOrder/号码候选/关联箱读取可能带入内容；真正保存另走 `/saas-service/order/billLading/save`。此处只是静态代码对应，不宣称所有服务端行为无副作用。
- 本次未保存、改关联、复制、删除或改设置；不能证明唯一性、签发资格、共享/私有关系、版本冲突、箱货引用和审计规则。

## 四类“导出单证”弹窗

四份模板弹窗均成功打开。弹窗包含文件类型、文件名、导出格式、编辑锁定，以及取消、导出、预览、发送邮件等文案。文件名 input maxlength=40；格式为 Select 内部 readonly text；编辑锁定为 checkbox。没有点击编辑模板/预览/导出/发送邮件。

| 来源 | DOM 文件类型候选 | 格式候选 |
| --- | --- | --- |
| SE-TPL | 订舱委托书、配舱通知、提单确认、HBL-套打、HBL-电放、HBL-COPY、HBL | EXCEL(XLSX)、EXCEL(XLS)、PDF |
| SI-TPL | 与本轮 SE 同名类型候选 | 同左 |
| AE-TPL | 订舱委托书、配舱通知、提单确认、HBL、HBL 套打 | 同左 |
| AI-TPL | 与本轮 AE 同名类型候选 | 同左 |

- 空运订单导出弹窗叫 HBL 不等于航空运单制作页叫 HAWB 的模型被取消，也不证明该模板采用标准 AWB 版式；这是两个入口的 UI 命名事实。
- SE-TPL 某些候选旁出现“编辑”，SI/AE/AI 对应位置未显示同样文案。尚未验证是角色、模板所有权、业务范围还是其他条件造成，不能据此推导权限算法。
- 模板名只保留这些通用业务种类，不复制公司/人员自定义命名、账户或抬头值。
- 候选格式不表示每个模板三种格式都可用，也不证明 PDF 页大小、字体、可编辑保护、分页、套打坐标或导出命名已验收。
- 没有读取实际生成文件，更没有发送邮件；不能把“预览”按钮存在当预览已验证。

### 响应证据为何不足

四份模板场景虽选取 request_body/response_body，但从整页导航开始捕获大量静态资产，保留预算耗尽。按事务 ID 关联保留请求/内容后，**没有找到** `tempPermissionList`、`getExportAccounts` 或 `letterhead/options` 的请求/响应记录。

因此**上述四份全导航记录**只能确认已渲染 DOM 的类型和格式，不能用它们还原模板 DTO 或完整许可清单。随后主会话从同一已加载 AI 页面仅关闭/重新打开模板弹窗，形成 AI-TPL-ISO，成功缩窄内容采集范围，见下节。

静态 `cargo-document-navigation.md` 记录了 tempPermissionList、getExportAccounts、letterhead/options 的候选读取，以及 templateShowName/fileTypes/locationFlag/exportNamePrefix/isShowInvoice 等客户端字段；这在上述四份全导航动态响应中未复核；AI-TPL-ISO 已对部分读取和字段提供直接响应证据。

### AI 缩窄弹窗场景的响应结构

AI-TPL-ISO 的动作序列为同页导航、关闭弹窗、重新打开、等待，全部 completed；本次保留 15 个事件、dropped=0。总体仍报告缺少 events，不能扩展成整站事件完整性保证；但下述两个响应 body 确实 captured，可离线解码为 JSON，结构和数量可验证。

| 读取 | 当前样本数量 | 包装结构 | data 项结构（实际 JSON 类型） |
| --- | --- | --- | --- |
| GET `/saas-portal/export/tempPermissionList` | 5 项 | `success,code,msg,data,additional`，data 为数组 | `id` 整数；`locationFlag,templateShowName,templatePosition,exportNamePrefix,fileType,exportFileType` 字符串；`itemOrder,isShowInvoice` 整数；`fileTypes,exportFileTypes` **对象** |
| GET `/saas-tenant/company/getExportAccounts` | 27 项 | 同上包装，data 为数组 | `accountId` 整数；`accountInfoName` 字符串；`defaultExport` 布尔 |

- 数量只代表该 AI 样本、该账号、该请求返回集合，不是全站/全公司模板或账户总数；不可推广到 SE/SI/AE。
- `fileTypes` 与 `exportFileTypes` 本样本是对象，不能按字段复数名假设数组；其键值到格式能力的映射仍需专门设计，不直接沿用第三方 DTO。
- `isShowInvoice` 是整数，与 `defaultExport` 布尔不同，不能全局按字段语义补布尔默认。
- 账户 ID、名称、模板自定义名称/前缀等实际值均不写入本报告；`defaultExport` 字段存在也不代表已验证在多账户下实际选取或输出行为。
- 同场景还捕获 POST `/saas-support/applications/flows/audit` 资格请求/响应，不把此 POST 自动当作业务保存，也不承诺其服务端绝无副作用。
- 未调用文件生成/下载/预览或邮件发送；响应结构不是 PDF/Excel 格式验收。

## 当前系统映射建议

- 继续保留委托、实际货物/箱货、单证申报量三个含义，不把海管家箱货页的“剩余”文本当作本系统唯一总量。
- 现有海运共享箱/货量分配有自己的权威关系和版本，不能因外站有批次表单就改成无版本自由数组；SI/AI 仍需独立进口关系与状态设计。
- 运单印字与关联号码/主单/箱货快照分离；SI 的单证制作字段相似不构成出口实体可直接通用于进口的证明。
- 模板输出要明确“订单详情导出”“已制作 MBL/HBL/MAWB/HAWB 输出”“箱货文档”三类来源，避免选错模板或从未保存输入生成正式单证。
- 邮件发送、港区数据、车辆定位和持续同步都不是本轮只读研究自动批准的实现。

## 限制与尚未找到的证据

- 箱货样本主要为初始化批次/空箱表，未取得完整已存箱/货明细行；不能验证所有货物字段、尺寸辅助、危险品、批次拆分或不同包装单位合计。
- SI 制作列表为空，MBL/HBL 仅新建草稿；没有已签发、作废、更正、并发或权限拒绝样本。
- DOM 未经计算样式/边界框逐控件验证；隐藏 Select 选项不等于当前值，内联隐藏的进仓字段不等于当前可编辑字段。
- 四类全导航模板响应未保留；AI 缩窄场景仅验证两个响应的当前样本结构，不是全站 DTO/权限完整性证明。
- 没有保存任何箱货/单证、写入关联/设置、生成 PDF/Excel、执行邮件发送；没有运行本系统测试。
- 关联报告：`cargo-document-navigation.md`（静态导航/副作用边界）、`form-field-matrix.md`（四类基本表单）、`air-waybill-observations.md`（航空运单制作）、`frontend-bundle-rules.md`（交付脚本规则）。

相关规范：`.trellis/spec/server/backend/sea-export-document-contract.md`、`.trellis/spec/server/backend/air-export-contract.md`、`.trellis/spec/server/backend/sea-shared-write-consistency.md`、`.trellis/spec/server/backend/sea-hbl-production.md`。
