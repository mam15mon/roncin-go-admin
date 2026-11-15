# 箱货、海运进口单证与输出模板的只读导航边界

## 证据范围

本记录仅分析实际 DOM 引用或已加载脚本捕获中存在的交付 CDN JavaScript；未运行下载脚本、未调用业务接口、未读取登录资料、未操作浏览器或修改产品代码。来源资产前缀为 `https://filecdn.hgj.com/packages/saas-web/1794456941597/js/`。

基础来源证据为主会话成功 DOM 记录及扩域脚本捕获 `ev_b61f804124e6fdf136f4d117531759d06d27de3f7695e309f79c52a1b502cee7`，此前静态 REA 分析 `ev_6c1f88ae4535a4a4d13908c374bdbd2aa36dd87fd756327d7317ad57f9c7ac2d` 有部分语义覆盖限制。新增 BoxInfo/LadingManage/SeaPackList/CDN 资产是人工逐段静态复核，不能冒称已纳入那次 REA 图分析。

所有“无自动保存”仅指下述已复核客户端调用链没有业务保存调用。接口内部副作用、其他未知子组件或服务端日志无法由静态前端证明。以下路由必须采用已经观察到的内部 `orderNo`、实际业务类型和原页面参数；不能用用户可见业务号冒充内部 ID，也不能只看标题就认定真实详情已载入。

## 快速结论

| 入口 | 已复核客户端行为 | 可做的只读调研 | 不包含的动作 |
| --- | --- | --- | --- |
| 四类详情→箱货 | 资格检查后路由；BoxInfo 初始化读取订单、箱货、字段设计，本地装载汇总 | 打开并记录字段/网络读取 | 保存、接收/取消关联、批次改名、关联箱、引入港区数据 |
| SI详情→单证制作 | 资格检查→确认客户未变化→DocumentMakeSea；初始化读单证列表 | 打开列表、查看种类与已有记录 | 删除、批量新增确认、保存 |
| 海运列表单个新增MBL/HBL | clickAdd只路由，billId=0；目的页createBill只建本地草稿并读取订单/候选 | 打开空白制作表单取字段 | 保存、改变号码/关联设置、复制覆盖、写入件重尺设置 |
| 海运新增装箱单 | id=0，只本地默认与读取列表/箱候选 | 打开空白表单 | 保存、删除、批量新增确认 |
| 输出模板弹窗 | 显示弹窗、读取许可模板/结算账户/可选抬头 | 打开弹窗看模板种类和格式 | 预览、下载、批量导出、邮件发送、生成任务 |

## 四类箱货导航

### 详情入口

- 四类 `goBox` 均先 `isPassAudit({permissionCode:'order_box_cargo',recordIds:[base.orderNo],verifyModule:2,bussTypeCode})`，类型码 SE=1、SI=2、AE=3、AI=4。
- AE/AI 还要求 `base.delegateCustomer === base.oldCusId`；未保存客户变化会提示并返回。正常只读应从已完整加载、未修改的详情进入。
- SE/SI 的 `goBox(e,t,r)` 可选 e 分支仅改本地 `base.orderLocking` 并写 sessionStorage；没有调用订单保存或服务端锁方法。正常按钮事件不会因为本地变量名带 lock 就证明后端锁已改变。
- 路由固定 `/order/box_info/:type/:orderNo`，type 为 `sea_export / sea_import / air_export / air_import`，orderNo 是内部 ID。
- 资格检查实际映射到 `saas-support/applications/flows/audit`（POST）；不能按 HTTP 方法直接当业务写，也不能仅凭命名保证后台绝无副作用。

### BoxInfo 和批次组件初始化

1. BoxInfo `created → init` 读取路由 type/orderNo，调用 `orderDetail({orderNo,isCropOrder})`；采用返回的真实订单，再进入 `getBoxInfo`。
2. `getBoxInfo` 调用 `orderGetBoxInfo`，然后 `designCargo({bussType,tmpId,orderNo})`；本地设置 orderCustoms.orderNo、base 和批次内容。
3. 子组件 BatchInfo `setBatchInfo` 只装载箱/货、批次、件重尺、委托汇总，计算剩余量并选择页签/批次；空运默认 `tabBoxInfo(1)` 货物页签，海运默认 `tabBoxInfo(0)` 箱页签。`tabBatch` 只更新活动批次、重置本地地址列表并回传本地基线。
4. 该初始化链没有 `orderEditBoxInfo`、`editBatchName`、`relatedBatch` 保存调用。部分 tabBoxInfo 分支会读取跨境/自拼货物，但不是保存。
5. 若 sessionStorage 有 `add-boxlist` 或 `cover-boxlist`，加载时会合并/替换**本地未保存**箱数组并清掉缓存。它不提交业务数据，但会使离开时出现未保存变化确认；不能假称所有初始页面永远无本地变动。
6. `beforeRouteLeave` 调用 Batch.matchData('noVertify') 汇总到本地，再 compareData；没有保存。它可触发离开确认，不能用自动确认后保存的方式处理。

真正保存为 BoxInfo.saveInfo：锁检查→Batch.matchData()→`orderEditBoxInfo(base)`→重读→saveTempBox。子组件“关联箱”“接收/取消货物关联”“批次改名”有独立写调用；不应在只读研究中点击。港区数据获取也会引入外部查询与本地覆盖，不属于正常页面打开。

## 海运进口单证制作与空白MBL/HBL

### 列表入口和初始化

- SI `goDocumentMake` 先检查 order_billlading 资格与客户未修改，然后路由 `DocumentMakeSea`。
- params：`type:'sea_import'`、`boxType:base.boxType`、`supplierId:base.delegateCustomer`、内部 orderNo、展示 outerOrderNo、corpStatus、lock。
- 实际路由 `/order/document_make_sea/:type/:boxType/:supplierId/:orderNo/:outerOrderNo/:corpStatus/:lock?`；注意 corpStatus 在 lock 前，不能沿用编辑器参数顺序手拼。
- DocumentMakeSea.created 只读取参数；mounted 设置页签、恢复本地过滤并 `getDocList`，再订阅 OCR 事件重读。`getDocList` 使用 `getSeaDocMakeList({orderNo,type:0,isCropOrder})`，把返回数据设置到本地分页列表。没有初始化保存调用。

### 单个新增与目的页

- `clickAdd(1)` 新增 MBL，`clickAdd(2)` 新增 HBL，先检查 lock；仅跳转 LadingManage，传原 type/boxType/supplierId/orderNo/outerOrderNo/lock/corpStatus，新增 `billType:1/2,billId:0`。
- 实际路由 `/order/lading/:type/:boxType/:supplierId/:orderNo/:outerOrderNo/:lock/:billType/:billId/:corpStatus/:orderType?`。此处 lock 在 corpStatus 前，与列表不同。
- LadingManage.created：读取路由→读取基础订单/常用资料→初始化名称（查询已有名称后本地取名）→读取主/分号候选→getDivisionBill。
- billId=0 时 getDivisionBill 调用 `createBill()`；**这个名称不代表接口创建**，实现只是 `takeInOrder()` 读取订单→复制本地 copyBase→推入本地 billList→selectBill。没有 orderEditDivisionBill/save。
- selectBill 在只有一项候选号时可本地预选该号，并调用 getBoxList/表单 fillData；getBoxList 是读取关联箱的 POST 查询，不是单证保存。Form.mounted 的 takeLadingData 只读取件重尺设置，`setTakeLadingDataSetting` 才会写设置，勿改此选项。
- 真正保存是 `editDivisionBill`，需锁/号码/名称检查后调用 `orderEditDivisionBill({data:base,noLoad})`；不能因为 createBill 是本地操作就认为保存按钮也无写。
- 目的编辑器“导出”要求 `authExport && base.id`，空白草稿不能为看模板而先保存；应查看真实已有单证，或回订单详情的“导出单证”弹窗。

### 装箱单与批量新增

- `clickAdd(3)` 只路由 SeaPackList，id=0。路由 `/order/sea_packlist/:type/:boxType/:supplierId/:orderNo/:outerOrderNo/:corpStatus/:lock/:id?`。
- SeaPackList.created 读取参数与基础订单；mounted 读箱候选/单证列表，只有 id 非零时读既有详情；watch 仅确保本地明细至少一行。初始化未见 savePackListBill。
- 批量新增走 AddPackListDialog.openDialog（读取完整箱候选），点击确认后会通过父组件 handleFinishBatch 创建，不能将“打开批量弹窗无写”扩展为“批量新增无写”。本轮建议不操作批量。

## 输出模板查看与生成边界

### 订单详情与箱货页

- SI 详情 openExport 只做 order_export_billlading 资格检查后 `showExport=true`；没有 saveInfo/orderSave。箱货页输出按钮直接 showExport；其 props type=箱货信息、pageFlag=BOX_GOODS_INFO、实际公司/订单号等。
- BaseExport.created 展示弹窗，若具有 company_title 权限读取可选导出抬头；moreType 为真时读取模板类型，模板/账户读取没有自动生成文件。moreType 为假则按 type 设固定 locationFlag 和默认文件名。
- 公司选择是海管家原始输出 UI，不代表本系统应恢复独立财务公司选择。

### 海运制作列表与编辑器

- 列表 openExport('提单(MBL)'/'提单(HBL)') 必须先勾选已有同类型记录；无选择或混选只提示，不显示 BaseExport。通过后只存 ids、exportName 和 showExport。
- 制作列表的 pageFlag 为 ORDER_MANAGE，再按 SI_/SE_和 _MBL/_HBL 拼成模板权限类别（如 SI_ORDER_MANAGE_MBL）。箱货页为 SI_/SE_/AE_/AI_ + BOX_GOODS_INFO。
- 编辑器本身传布尔属性 `moreType:""`，Vue 将空字符串布尔 prop 转为 true，所以仍走 getTypeList，且需已有 base.id。不能把空字符串误判为固定导出模式；需以运行时模板读取请求确认。
- BaseExport `getTypeList` 调用 tempPermissionList、getExportAccounts；模板字段可含 templateShowName、fileTypes、locationFlag、exportNamePrefix、isShowInvoice 等。许可模板列表因业务、主分单、账号不同，静态枚举不能当全部可用模板清单。
- 真正预览/导出由 previewFn/judgeExport/doExport 发起 exportUnifiedEntrance、批量导出或其他服务端生成动作；可能创建导出任务并下载文件。发送邮件另走 isSendEmail 路径，不属于只读查看弹窗。本轮不执行这些动作。

## 已定位接口（仅静态路径候选）

| 目的 | 客户端调用/路径 | 边界 |
| --- | --- | --- |
| 订单读取 | orderDetail → `saas-service/order/detail` | orderNo 是内部 ID |
| 箱货读取 | orderGetBoxInfo → `saas-service/order/box/cargoInfo` | orderNo/isCropOrder |
| 箱货设计 | designCargo → `saas-service/defined/field/boxCargo` | bussType/tmpId/orderNo |
| 箱货保存 | orderEditBoxInfo → `saas-service/order/box/edit` | 保存按钮执行，本轮不调用 |
| 海运单证列表 | getSeaDocMakeList → `saas-service/order/billLading/billMultiList` | type=0 查询 |
| 海运单证读取 | orderGetBillLadingDetail → `saas-service/order/billLading/detail/{id}` | 既有 billId |
| 主分候选号 | getHblOrMbl → `saas-service/order/billLading/getBillOfLadingNos` | type=1/2、showPrimary |
| 关联箱查询 | getBoxOddInBillLadingList → `saas-service/order/billLading/getBoxs` | POST 读查询，不能按方法名当保存 |
| 单证保存 | orderEditDivisionBill → `saas-service/order/billLading/save` | 不执行 |
| 许可输出模板 | tempPermissionList → `saas-portal/export/tempPermissionList` | 只读模板候选 |
| 导出账户 | getExportAccounts → `saas-tenant/company/getExportAccounts` | 只读；账户值不复制进研究文档 |
| 导出抬头 | fetchExportCompanyOptions → `saas-tenant/company/letterhead/options` | 具备 company_title 时读取 |
| 真正导出 | exportUnifiedEntrance / exportUnifiedEntranceBatch → `saas-portal/export/exportUnifiedEntrance` / `…/batchExportUnifiedEntrance` | 生成文件/任务，不纳入只读弹窗结论 |

## 来源指纹与范围

原始资料在本机 `/home/vince/.local/share/rea/hgj/static-analysis/`；完整段落文本、范围摘要保存在 `cargo-navigation-ranges.json`。偏移均为零基 UTF-8 字节半开区间，不是 UTF-16 字符位置。

| 交付文件 | SHA-256 |
| --- | --- |
| [BoxDetail~BoxInfo~EditBooking~513dcb80.8831aed0.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/BoxDetail~BoxInfo~EditBooking~513dcb80.8831aed0.js) | `ac8081c70ace52455f5cf10b0738d137e6abf092195f5c67b43533f08350f33a` |
| [BoxInfo~31ecd969.f00870b7.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/BoxInfo~31ecd969.f00870b7.js) | `547e6f95bf7a57ffa42413d14d05a28a8eff63d68b84f4f27058bd0dedc2bf64` |
| [DocumentMakeSea~21833f8f.0f4b8a9a.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/DocumentMakeSea~21833f8f.0f4b8a9a.js) | `2b4305e681af36f138d0e4bbc01c6847538227df562aa980e97871a52c002879` |
| [EditAirExport~21833f8f.ceb04dc7.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditAirExport~21833f8f.ceb04dc7.js) | `d02ea9de5ee7444fd8d83e930fdeca30b616549f8159bb32a2c6df7b36b6f831` |
| [EditAirImport~21833f8f.bf3376f8.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditAirImport~21833f8f.bf3376f8.js) | `2eff52e3f9c497ff0ccb129b8c4d3130a8d77934189d838cd0b3b502e8f97cb7` |
| [EditSeaExport~31ecd969.af321e09.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditSeaExport~31ecd969.af321e09.js) | `41ea5cef166f6cd5b1b58fb52158d6b295ddcca3e4eecd2da36a2a43dbc6d502` |
| [EditSeaImport~31ecd969.d4bb6e32.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/EditSeaImport~31ecd969.d4bb6e32.js) | `3bfae46c1104e3f7488d92ddc00052982436ac06233c3155f4cfd916fb9a8d13` |
| [LadingDetail~LadingManage~b88f4497.150374cc.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/LadingDetail~LadingManage~b88f4497.150374cc.js) | `768b039710039fa7114726943ffeeb023ff51ee58ad9f9a0c5c24a794dae944d` |
| [LadingManage~3fdd10e2.07a37045.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/LadingManage~3fdd10e2.07a37045.js) | `d291eed9f5dd2bdd8a5b1e29694ed6e034e0fa301729d9878a1b5270eb29b373` |
| [SeaPackList~21833f8f.a80cd43b.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/SeaPackList~21833f8f.a80cd43b.js) | `fdd157a1803be07bc3adfc329f3cebd519d43768b80ac4aae3eb8ce1a781a2b8` |
| [app~c714bc7b.aad8b8e0.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/app~c714bc7b.aad8b8e0.js) | `27934724a2acc3d827ba5d19e74ffe236d679325707071ca0e0a3c7843ce9e64` |
| [chunk-1a4bd546.81c9e723.js](https://filecdn.hgj.com/packages/saas-web/1794456941597/js/chunk-1a4bd546.81c9e723.js) | `f34879b13883201b0d18cf9307d60e657083d24a64a8b190bb53f7ecb6aabd99` |

| 范围编号 | 文件 | UTF-8 字节范围 |
| --- | --- | --- |
| SE-box-nav | `EditSeaExport~31ecd969.af321e09.js` | `[71258,71749)` |
| SI-box-nav | `EditSeaImport~31ecd969.d4bb6e32.js` | `[52346,52837)` |
| AE-box-nav | `EditAirExport~21833f8f.ceb04dc7.js` | `[42461,43000)` |
| AI-box-nav | `EditAirImport~21833f8f.bf3376f8.js` | `[50458,50997)` |
| SI-document-nav | `EditSeaImport~31ecd969.d4bb6e32.js` | `[44944,45660)` |
| SI-export-dialog | `EditSeaImport~31ecd969.d4bb6e32.js` | `[44570,44944)` |
| BOX-init | `BoxInfo~31ecd969.f00870b7.js` | `[8821,9489)` |
| BOX-read | `BoxInfo~31ecd969.f00870b7.js` | `[9667,10539)` |
| BOX-save | `BoxInfo~31ecd969.f00870b7.js` | `[10928,11577)` |
| BOX-leave | `BoxInfo~31ecd969.f00870b7.js` | `[8176,8687)` |
| BOX-design | `BoxInfo~31ecd969.f00870b7.js` | `[10619,10928)` |
| BATCH-init | `BoxDetail~BoxInfo~EditBooking~513dcb80.8831aed0.js` | `[42736,44267)` |
| BATCH-select | `BoxDetail~BoxInfo~EditBooking~513dcb80.8831aed0.js` | `[49544,50154)` |
| BATCH-tabs | `BoxDetail~BoxInfo~EditBooking~513dcb80.8831aed0.js` | `[37549,40649)` |
| BATCH-match | `BoxDetail~BoxInfo~EditBooking~513dcb80.8831aed0.js` | `[54640,55269)` |
| DOC-init | `DocumentMakeSea~21833f8f.0f4b8a9a.js` | `[15016,15911)` |
| DOC-read | `DocumentMakeSea~21833f8f.0f4b8a9a.js` | `[18234,18679)` |
| DOC-add | `DocumentMakeSea~21833f8f.0f4b8a9a.js` | `[20183,20953)` |
| DOC-export | `DocumentMakeSea~21833f8f.0f4b8a9a.js` | `[16185,16707)` |
| DOC-export-props | `DocumentMakeSea~21833f8f.0f4b8a9a.js` | `[9724,10176)` |
| LADING-init | `LadingManage~3fdd10e2.07a37045.js` | `[5476,6543)` |
| LADING-read | `LadingManage~3fdd10e2.07a37045.js` | `[11604,12978)` |
| LADING-local-create | `LadingManage~3fdd10e2.07a37045.js` | `[14491,15579)` |
| LADING-select | `LadingManage~3fdd10e2.07a37045.js` | `[15579,17562)` |
| LADING-save | `LadingManage~3fdd10e2.07a37045.js` | `[19120,20266)` |
| LADING-export | `LadingManage~3fdd10e2.07a37045.js` | `[1633,1869)` |
| FORM-read-volume | `LadingDetail~LadingManage~b88f4497.150374cc.js` | `[37723,38296)` |
| FORM-write-volume-setting | `LadingDetail~LadingManage~b88f4497.150374cc.js` | `[38296,38996)` |
| PACK-init | `SeaPackList~21833f8f.a80cd43b.js` | `[22532,23264)` |
| EXPORT-init | `chunk-1a4bd546.81c9e723.js` | `[10559,11139)` |
| EXPORT-type-read | `chunk-1a4bd546.81c9e723.js` | `[12928,15038)` |
| EXPORT-account-read | `chunk-1a4bd546.81c9e723.js` | `[18638,19020)` |
| EXPORT-actions | `chunk-1a4bd546.81c9e723.js` | `[12790,12928)` |
| APP-box-route | `app~c714bc7b.aad8b8e0.js` | `[188733,188833)` |
| APP-lading-route | `app~c714bc7b.aad8b8e0.js` | `[183850,184010)` |
| APP-seadoc-route | `app~c714bc7b.aad8b8e0.js` | `[185561,185711)` |
| APP-pack-route | `app~c714bc7b.aad8b8e0.js` | `[185357,185517)` |
| APP-box-read | `app~c714bc7b.aad8b8e0.js` | `[68405,68715)` |
| APP-box-design | `app~c714bc7b.aad8b8e0.js` | `[71563,71668)` |
| APP-seadoc-read | `app~c714bc7b.aad8b8e0.js` | `[71756,71976)` |
| APP-lading-read | `app~c714bc7b.aad8b8e0.js` | `[67798,68128)` |
| APP-lading-write | `app~c714bc7b.aad8b8e0.js` | `[68117,68397)` |
| APP-templates | `app~c714bc7b.aad8b8e0.js` | `[55859,56259)` |
| APP-accounts | `app~c714bc7b.aad8b8e0.js` | `[63464,63594)` |
| APP-export-write | `app~c714bc7b.aad8b8e0.js` | `[56615,57025)` |

## 四类订单详情输出入口与模板类别补充

- SE/SI/AE/AI 的详情 `openExport` 都只做 `order_export_billlading` 资格检查并设置 `showExport=true`，没有 `saveInfo/orderSave`；bussTypeCode 分别 1/2/3/4。
- 可见头部按钮为 BaseBtn 渲染的 `article.base-btn`，文本“导出单证”。可用 `article.base-btn:has(.content-wrap:text-is("导出单证"))` 定位，但必须先确认当前实际 DOM 只有一个匹配，且该详情已真正加载了内部 orderNo 对应的订单。
- 四类传入 BaseExport 的 `moreType:""` 是布尔 prop 的空属性，Vue 解析为 true，因而初始化走 `getTypeList`；不能以 JavaScript 空字符串假值推断它不会读取模板。
- 详情 type 直接是“海运出口/海运进口/空运出口/空运进口”，不含“提单管理”且不在箱货/报关/费用录入类别，所以 getTypeList 不再叠加业务前缀和主分后缀；直接使用 SE_DETAIL、SI_DETAIL、AE_DETAIL、AI_DETAIL。
- 相比之下，海运运单编辑器与制作列表使用 type=提单管理、pageFlag=ORDER_MANAGE，按品类与主分单生成 SE_ORDER_MANAGE_MBL/HBL 或 SI_ORDER_MANAGE_MBL/HBL；空运运单编辑器 pageFlag=AIR_BILLLADING_DETAIL 有专门清空业务前缀/后缀的分支，不能机械套海运规则。
- 订单详情文件名日期候选：SE/AE 用 estimatedTimeDeparture，SI/AI 用 estimatedTimeArrival；不代表此处自动刷新业务汇率或改写订单日期。

| 页面来源 | 模板请求 pageFlag | 文件名业务日期 |
| --- | --- | --- |
| SE订单详情 | SE_DETAIL | ETD |
| SI订单详情 | SI_DETAIL | ETA |
| AE订单详情 | AE_DETAIL | ETD |
| AI订单详情 | AI_DETAIL | ETA |
| SI海运MBL编辑器/同类批量 | SI_ORDER_MANAGE_MBL | ETA |
| SI海运HBL编辑器/同类批量 | SI_ORDER_MANAGE_HBL | ETA |
| 四类箱货页 | SE_/SI_/AE_/AI_ + BOX_GOODS_INFO | 出口ETD、进口ETA |

这些是前端构造的模板分类；账号实际返回哪些模板、支持哪些文件格式由本次运行时许可列表决定，不能仅凭静态flag宣称全部可用。

| 补充范围 | 文件 | UTF-8 字节范围 |
| --- | --- | --- |
| SE-export-handler | `EditSeaExport~31ecd969.af321e09.js` | `[72510,72884)` |
| SE-export-props | `EditSeaExport~31ecd969.af321e09.js` | `[48176,48596)` |
| SI-export-handler | `EditSeaImport~31ecd969.d4bb6e32.js` | `[44570,44944)` |
| SI-export-props | `EditSeaImport~31ecd969.d4bb6e32.js` | `[33866,34288)` |
| AE-export-handler | `EditAirExport~21833f8f.ceb04dc7.js` | `[43761,44135)` |
| AE-export-props | `EditAirExport~21833f8f.ceb04dc7.js` | `[27757,28177)` |
| AI-export-handler | `EditAirImport~21833f8f.bf3376f8.js` | `[51758,52132)` |
| AI-export-props | `EditAirImport~21833f8f.bf3376f8.js` | `[35846,36268)` |
