# 海管家脚本补充证据与规划影响

## 来源与证据边界

用户提供两轮脚本分析：上一轮为会话内状态码与列表入口条件，本轮附件为
`/home/vince/.codex/attachments/9412daed-87de-4d45-8633-d4623161acbb/已粘贴的文本.txt`。
本文仅归纳用户提供的分析，未独立读取目标脚本或调用目标系统接口。来源明确本轮没有提交真实开票、核销、撤销、删除，也没有捕获提交响应；请求结构来自代码构造，不是完整服务端 schema。

证据索引（来源提供）：
- 列表脚本：`ev_b9eb030019cec80317b5cd0e7e5174b156bc923d9eaa34df01827d2c12a6cc32`。
- 状态映射：`ev_1116bb2a1e1db3caa49b66e6058baee567add965a16075fb05509be6a6b33d77`。
- 账单详情与发票：`EditBill~6a2c624d.49dc0def.js`、`chunk-f0b56138.c1fcdeee.js`，`ev_7f8e8f93b649b1cccc8f852b82841ed5f98170ff87925d2e8ecd33eb34d6efbd`。
- 提成：`EditCommission~21833f8f.7d13ae8d.js`，`ev_2b3c623e46d596b7d6347ee2e9410d0a6c2cc5b84a0bd2e90975e4ec0b02d30f`。
- 报表：`Finance~31ecd969.4154b0eb.js`，`ev_b65be3ea7873ba47f397bad3e6d5c87c1deffb9f45fd594398ea24c3806ec6fa`。

## 状态与入口结论

目标账单 state 为 0 未开票、1 未核销、2 部分核销、3 已完成、4 部分开票、99 未知；billType 为 0 应收、1 应付、2 预收、3 预付。费用另有独立状态，包含已核销未开票和部分核销未开票。
这些是参考系统映射，不移植为本项目枚举，也不能按数字建立状态流转。

- 普通账单详情允许 state 1/4 打开发票修改，InvoiceSet 带入已有发票行且允许新增；解决了列表仅允许 state 0 录票与继续补票的入口疑问。
- 普通账单详情核销允许 state 1/2/4，预收付允许 1/2；解决了列表不允许 state 2 与继续部分核销的入口疑问。
- 增删费用普通账单条件为 state 0，预收付为 state 1，还受权限、corpStatus 和 hedgeType 限制。corpStatus 含义未知，不自行命名；不取代本项目已确认的按发票/核销/对冲事实锁定规则。
- 本项目详情和批量入口应分别覆盖可操作场景，不能简单以单一展示状态代替服务端能力校验。

## 发票数据与未决项

`saas-service/settlement/addBillNo` 载荷为 billId 或 billIds、invoiceTime、invoiceRecordModels、partialInvoiceReviewType。行包含 invoiceNo、invoiceAmount、invoiceTypeCode、serialNumber、可能的 source；已有行可能还有额外字段。
批量初始金额取未开票余额总和；新增行金额取剩余值与零的较大者。参考界面最多 100 行，表单内号码去重；不等于已确认本项目数量上限或跨账单唯一性。
跨账单逐项金额分配、混合税率分摊、实际提交总额限制仍未知。不能将一张发票全额重复计入每张账单。
行级作废 `saas-support/invoice/voidInvoice` 与批量清空票号不同；传播范围仍未知，不用清空字符串模拟撤销事实。
类型 7 显示「-」且写入同样号码，类型 8 显示「形式发票」；不足以证明无需发票语义。本项目是否需要无需发票由用户业务决定，不引入假号码。

## 费用与核销数据

- 添加候选使用 `saas-service/fee/list`，有 billStatus:[0]、结算单位名称与方向等字段，currency 可为空；目标服务端语义未验证。本项目仍按用户已确认的公司、结算对象 ID、币种、方向、未建账条件校验，不照搬名称匹配或放宽币种。
- addFee/removeFee 提交账单 ID、账单编号及费用 ID 数组。原始费用是否保留已由用户确认；目标服务端重新计额、移除最后一行和空账单行为仍未知。
- verification 提交 billNo、feeDetailIds、总额/币种/汇率/时间/账户/方式、手续费、损益及 verificationDetailModels（收付编号与本次金额）；单费用为 feePartVerification。多费用分配、换算舍入及损益算法仍未知。
- 使用预收付款不足时，参考前端先调用 prmanage/verificationInsert 创建差额流水，提示未保存核销流水仍存在。本项目必须明确两步操作或整体提交的语义，不在仅选择/预览时偷偷落业务流水；用户后续已确认：最终提交时原子创建差额流水并核销，全部成功或全部失败；取消弹窗不产生新流水。
- cancelVerification、cancelVerificationBatch、revokeVerification 为不同撤销入口；summaryVerify 提示影响全部关联账单。本项目反向操作应展示真实影响范围，不能只撤销状态而不恢复资金分配。

## 提成与报表

- 提成按 commissionType 选订单或账单；前端折本币利润后调用 getCommissionAmount，服务端返回参考金额，最终 commissionAmount 可编辑。计提依据、区间算法仍未知；本项目不据此直接允许覆盖已计提金额，需核对现有手工调整机制。
- 利润列表/合计/分组分别请求 profitList/profitStat/profitGroupList，金额由服务端返回；actualProfit 等名称不构成公式证据。
- financialReceivePayment 返回应收实收需收、应付实付需付及损益，精确口径未知。
- age/list 默认 INVOICED_TIME、近两年 beginTime/endTime，响应 feeGroups 键形成分组列；这不能证明历史截止余额或账龄区间算法。

## 后续研究重点

无需继续追已解答的入口存在性。若 REA 能只读取得资料，优先找响应类型、脱敏既有详情、前端校验及字段映射；缺少服务端算法时转为本项目明确产品规则和可对账样例，不以不可见算法无限阻塞规划。
待定重点：跨账单/费用的金额分配、无需发票标记与已有发票的边界、提成基础及区间、报表日期和历史账龄。本项目撤销解锁已由用户后续确认：全部有效发票、核销和对冲关联解除后恢复费用增删，保留历史；这不是对目标系统行为的证明。外部证据不等于用户对这些规则的确认。

本项目用户已明确批准财务按权限填写原因标记无需发票，结清即可完成；普通需票账单要求票齐且结清。此业务设计不代表参考系统类型 7「-」的真实含义已得到验证。
