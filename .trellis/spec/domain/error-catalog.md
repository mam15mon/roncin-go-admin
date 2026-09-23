# biz 层业务错误码目录

> 自动生成：依据 `server/internal/biz/**/*.go` 扫描 `errors.*` 错误定义生成，请勿手工修改。
> 再生成命令：`node scripts/generate-error-catalog.mjs`；一致性校验：`node scripts/generate-error-catalog.mjs --check`。

## 口径

- 只收录 biz 层以 `errors.NotFound / BadRequest / Conflict / Forbidden / Unauthorized / InternalServer / ServiceUnavailable`（及预留的 `PreconditionFailed`）定义的领域错误；传输层参数校验、数据层驱动错误不在此表。
- 「语义」列为错误构造函数对应的 HTTP 状态类别；同一错误码以不同构造函数定义时列出全部类别。
- 「中文消息」列为定义处的静态消息；同一错误码多处定义时逐行列出全部消息。含 `%s` 的为动态模板（`fmt.Sprintf` 或字符串拼接），完全动态（变量传参）时记为「（动态消息）」。
- 「关联 proto ErrorReason」列仅在错误码经 `reasonFromProto(...)` 关联 `server/api/**/error_reason.proto` 枚举时填写。
- 分组按定义文件所属业务域（同一错误码出现在多个业务域时在各域分别列出），组内按错误码排序；「定义位置」为 `server/internal/biz/` 下的文件名。
- 当前共 371 个唯一错误码，373 条目录记录。

## 汇总

| 业务域 | 错误码数 |
| --- | --- |
| 订单 | 102 |
| 单证 | 29 |
| 财务 | 85 |
| 提成 | 37 |
| 往来单位 | 34 |
| 权限 | 60 |
| 平台 | 26 |

## 订单

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| FEE_SUPPLEMENT_APPROVER_UNAVAILABLE | 409 Conflict | 当前没有具备订单直接解锁资格的审批人，请先配置审批资格 | order_fee_supplement.go | — |
| FEE_SUPPLEMENT_CANCEL_BLOCKED | 409 Conflict | 当前补录费用不满足专用作废条件 | order_fee_supplement.go | — |
| FEE_SUPPLEMENT_IDEMPOTENCY_CONFLICT | 409 Conflict | 同一幂等键的补录申请内容已变化，请刷新后重新发起 | order_fee_supplement.go | — |
| FEE_SUPPLEMENT_NOT_APPLICABLE | 409 Conflict | 订单当前没有业务锁或财务锁，请使用普通费用新增入口 | order_fee_supplement.go | — |
| FEE_SUPPLEMENT_REQUEST_INVALID | 400 Bad Request | 补录费用申请参数不合法 | order_fee_supplement.go | — |
| FEE_SUPPLEMENT_REQUEST_NOT_FOUND | 404 Not Found | 补录费用申请不存在 | order_fee_supplement.go | — |
| FEE_SUPPLEMENT_TRANSITION | 409 Conflict | 当前补录申请状态不允许该操作 | order_fee_supplement.go | — |
| LOCK_BASIS_CHANGED | 409 Conflict | （动态消息） | order_fee_supplement.go | — |
| NUMBER_RULE_EXISTS | 409 Conflict | 单据类型的编号规则已存在 | orderconfig.go | — |
| NUMBER_RULE_NOT_FOUND | 404 Not Found | 编号规则不存在 | orderconfig.go | — |
| NUMBER_SEQUENCE_EXHAUSTED | 409 Conflict | 当前编号周期的序列已耗尽 | orderconfig.go | — |
| ORDER_ABNORMAL_CASE_EXISTS | 409 Conflict | 该异常已在订单上标记且未解决 | order_abnormal_case.go | — |
| ORDER_ABNORMAL_CASE_INVALID_ARGUMENT | 400 Bad Request | 订单异常标记字段不合法 | order_abnormal_case.go | — |
| ORDER_ABNORMAL_CASE_KIND_INVALID | 400 Bad Request | 异常类型必须是当前组织启用的异常类型主数据 | order_abnormal_case.go | — |
| ORDER_ABNORMAL_CASE_NOT_FOUND | 404 Not Found | 订单异常标记不存在 | order_abnormal_case.go | — |
| ORDER_ABNORMAL_CASE_STATUS_CONFLICT | 409 Conflict | 异常状态已被并发修改 | order_abnormal_case.go | — |
| ORDER_ALREADY_LOCKED | 409 Conflict | 订单已被锁定 | order_lock.go | — |
| ORDER_ATTACHMENT_EXISTS | 409 Conflict | 订单附件幂等键已存在 | order_attachment.go | — |
| ORDER_ATTACHMENT_INVALID_ARGUMENT | 400 Bad Request | 订单附件字段不合法 | order_attachment.go | — |
| ORDER_ATTACHMENT_NOT_FOUND | 404 Not Found | 订单附件不存在 | order_attachment.go | — |
| ORDER_BUSINESS_LOCKED | 409 Conflict | 订单已锁定，如需修改请先申请解锁 | order_lock.go | — |
| ORDER_BUSINESS_UNSUPPORTED | 400 Bad Request | 当前仅支持海运出口订单 | order.go | — |
| ORDER_CARGO_ITEM_INVALID_ARGUMENT | 400 Bad Request | 订单货物明细参数不合法 | order_cargo_item.go | — |
| ORDER_CARGO_ITEM_NOT_FOUND | 404 Not Found | 订单货物明细不存在 | order_cargo_item.go | — |
| ORDER_CLOSED | 409 Conflict | 订单已结案，不允许修改业务数据 | order.go | — |
| ORDER_CLOSURE_BLOCKED | 409 Conflict | 订单尚未满足结案条件 | order.go | — |
| ORDER_CLOSURE_INVALID | 400 Bad Request | 订单结案状态流转不合法 | order.go | — |
| ORDER_COMMISSION_PERSONNEL_MISSING | 400 Bad Request | 订单缺少%s、%s人员，请在内部信息区补全后再开单 | order.go | — |
| ORDER_CONSOLIDATION_SHIPMENT_TYPE_INVALID | 400 Bad Request | 仅拼箱订单可查看自拼汇总 | order.go | — |
| ORDER_CONTAINER_EXISTS | 409 Conflict | 该箱号已存在于当前订单 | order_container.go | — |
| ORDER_CONTAINER_INVALID_ARGUMENT | 400 Bad Request | 仅整箱(FCL)业务允许维护集装箱<br>订单集装箱参数不合法 | order_container.go | — |
| ORDER_CONTAINER_NOT_FOUND | 404 Not Found | 订单集装箱不存在 | order_container.go | — |
| ORDER_CONTAINER_SPEC_INVALID | 400 Bad Request | 集装箱规格不存在或已被禁用 | order_container.go | — |
| ORDER_CUSTOMER_INVALID | 400 Bad Request | 订单客户必须是启用的客户角色 | order.go | — |
| ORDER_FEE_BILLING_UNIT_INVALID | 400 Bad Request | 计费单位不存在、已停用或不属于当前组织 | order_fee.go | — |
| ORDER_FEE_BILL_OCCUPIED | 409 Conflict | 费用已进入未取消的账单，请先取消对应账单后再删除 | order_fee.go | — |
| ORDER_FEE_CURRENCY_INVALID | 400 Bad Request | 币种必须是启用的 ISO 币种 | order_fee.go | — |
| ORDER_FEE_EXCHANGE_RATE_OVERRIDE_FORBIDDEN | 403 Forbidden | 无权手工覆盖费用汇率 | order_fee.go | — |
| ORDER_FEE_FINANCE_LOCKED | 409 Conflict | 订单已因确认或发放提成进入财务锁定，请通过提成调整记录处理后续差异 | order_fee.go | — |
| ORDER_FEE_IDEMPOTENCY_CONFLICT | 409 Conflict | 费用请求幂等键已被使用 | order_fee.go | — |
| ORDER_FEE_INVALID_ARGUMENT | 400 Bad Request | 订单费用字段不合法 | order_fee.go | — |
| ORDER_FEE_INVALID_TRANSITION | 409 Conflict | 当前费用状态不允许执行该操作 | order_fee.go | — |
| ORDER_FEE_NOT_FOUND | 404 Not Found | 订单费用不存在 | order_fee.go | — |
| ORDER_FEE_PARTY_INVALID | 400 Bad Request | 结算单位必须是当前组织启用的往来单位 | order_fee.go | — |
| ORDER_FEE_SETTING_INVALID | 400 Bad Request | 费用设置不存在、已停用或不适用于当前订单 | order_fee.go | — |
| ORDER_FEE_VERSION_CONFLICT | 409 Conflict | 订单费用已被其他操作人修改，请刷新后重试 | order_fee.go | — |
| ORDER_IDEMPOTENCY_CONFLICT | 409 Conflict | 订单请求幂等键已被其他请求使用 | order.go | — |
| ORDER_INVALID_ARGUMENT | 400 Bad Request | 订单字段不合法 | order.go | — |
| ORDER_LOCK_ROLE_REQUIRED | 403 Forbidden | 当前用户未分配对应业务类型的订单锁定角色或数据范围不满足要求 | order_lock.go | — |
| ORDER_NOT_FOUND | 404 Not Found | 订单不存在<br>解锁请求记录不存在 | order.go、order_lock.go | — |
| ORDER_NOT_LOCKED | 409 Conflict | 订单当前未锁定 | order_lock.go | — |
| ORDER_NUMBER_EXISTS | 409 Conflict | 订单编号已存在 | order.go | — |
| ORDER_PARTNER_ROLE_BLACKLISTED | 400 Bad Request | 所选%s已列入黑名单，不能新增业务关联 | order.go | — |
| ORDER_PERSONNEL_EXISTS | 409 Conflict | 订单协作人员已存在 | order_personnel.go | — |
| ORDER_PERSONNEL_INVALID_ARGUMENT | 400 Bad Request | 订单协作人员字段不合法 | order_personnel.go | — |
| ORDER_PERSONNEL_NOT_FOUND | 404 Not Found | 订单协作人员不存在 | order_personnel.go | — |
| ORDER_PERSONNEL_USER_INVALID | 400 Bad Request | 协作人员必须是订单组织范围内启用的组织成员 | order_personnel.go | — |
| ORDER_RELEASE_POD_DOCUMENT_INVALID | 400 Bad Request | 关联提单必须是同一订单下的提单 | order_release_pod.go | — |
| ORDER_RELEASE_POD_INVALID_ARGUMENT | 400 Bad Request | 放货凭证字段不合法 | order_release_pod.go | — |
| ORDER_RELEASE_POD_INVALID_STATUS | 400 Bad Request | 放货凭证当前状态不允许该操作 | order_release_pod.go | — |
| ORDER_RELEASE_POD_NOT_FOUND | 404 Not Found | 放货凭证不存在 | order_release_pod.go | — |
| ORDER_RELEASE_POD_STATUS_CONFLICT | 409 Conflict | 放货凭证状态已被并发修改 | order_release_pod.go | — |
| ORDER_SHIPPING_DOCUMENT_EXISTS | 409 Conflict | 订单提单已存在 | order_shipping_document.go | — |
| ORDER_SHIPPING_DOCUMENT_INVALID_ARGUMENT | 400 Bad Request | 订单提单字段不合法 | order_shipping_document.go | — |
| ORDER_SHIPPING_DOCUMENT_INVALID_STATUS | 400 Bad Request | 提单当前状态不允许该操作 | order_shipping_document.go | — |
| ORDER_SHIPPING_DOCUMENT_NOT_FOUND | 404 Not Found | 订单提单不存在 | order_shipping_document.go | — |
| ORDER_SHIPPING_DOCUMENT_STATUS_CONFLICT | 409 Conflict | 提单状态已被并发修改 | order_shipping_document.go | — |
| ORDER_STATUS_CONFLICT | 409 Conflict | 订单状态已被其他操作修改<br>货物明细已被更新，请刷新后重试<br>集装箱已被更新，请刷新后重试 | order.go、order_cargo_item.go、order_container.go | — |
| ORDER_STATUS_INVALID | 400 Bad Request | 订单状态不合法 | order.go | — |
| ORDER_TERMINATED | 409 Conflict | 订单已终止，不允许修改业务数据 | order.go | — |
| ORDER_TERMINATION_INVALID | 400 Bad Request | 订单终止状态流转不合法 | order.go | — |
| ORDER_TERMINATION_IN_PROGRESS | 409 Conflict | 订单已进入终止流程，不允许修改业务数据 | order.go | — |
| ORDER_UNLOCK_APPROVAL_STALE | 409 Conflict | 审批请求已过期或已被直接解锁取代 | order_lock.go | — |
| ORDER_UNLOCK_APPROVER_INVALID | 403 Forbidden | 非有效审批人 | order_lock.go | — |
| ORDER_UNLOCK_APPROVER_NOT_CONFIGURED | 400 Bad Request | 未配置具备对应业务类型订单锁定权限的业务角色成员 | order_lock.go | — |
| ORDER_UNLOCK_DINGTALK_DISPATCH_FAILED | 500 Internal Server Error | 钉钉审批发起明确失败 | order_lock.go | — |
| ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED | 400 Bad Request | 申请人或审批候选人未绑定钉钉账号 | order_lock.go | — |
| ORDER_UNLOCK_REQUEST_ACTIVE | 409 Conflict | 当前订单已有生效中或审批中的解锁请求 | order_lock.go | — |
| SEA_CARGO_ALLOCATION_NOT_FOUND | 404 Not Found | 海运货物分配记录不存在 | sea_order_change.go | — |
| SEA_DOCUMENT_VERSION_CONFLICT | 409 Conflict | 单证版本冲突，请刷新后重试 | order_lock.go | — |
| SEA_MASTER_BILL_INVALID_ARGUMENT | 400 Bad Request | 海运出口订单必须选择船公司<br>海运出口订单必须提供主单信息 | order_usecase.go | — |
| SEA_MASTER_BILL_MEMBER_ORDER_LOCKED | 409 Conflict | 共享 MBL 关联的部分订单已被锁定，需全部解锁后才能修改共享 MBL | order_lock.go | — |
| SEA_ORDER_REASSIGNMENT_BLOCKED | 400 Bad Request | 当前操作票状态或财务事实不允许改配 | sea_order_change.go | — |
| SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT | 409 Conflict | 相同幂等键请求指纹冲突 | sea_order_change.go | — |
| SEA_ORDER_REASSIGNMENT_INVALID_ARGUMENT | 400 Bad Request | 改配参数不合法 | sea_order_change.go | — |
| SEA_ORDER_REASSIGNMENT_TARGET_CONFLICT | 409 Conflict | 目标 MBL 冲突或不可用 | sea_order_change.go | — |
| SEA_ORDER_REASSIGNMENT_VERSION_CONFLICT | 409 Conflict | 数据已被更新，请刷新后重试 | sea_order_change.go | — |
| SEA_ORDER_SPLIT_BLOCKED | 400 Bad Request | 当前操作票状态或财务事实不允许拆票 | sea_order_change.go | — |
| SEA_ORDER_SPLIT_CONSERVATION_FAILED | 400 Bad Request | 拆票数量守恒校验失败 | sea_order_change.go | — |
| SEA_ORDER_SPLIT_ENTITY_CROSSES_RESULTS | 400 Bad Request | 分单或集装箱不可跨结果票分配 | sea_order_change.go | — |
| SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT | 409 Conflict | 相同幂等键请求指纹冲突 | sea_order_change.go | — |
| SEA_ORDER_SPLIT_INVALID_ARGUMENT | 400 Bad Request | 拆票参数不合法 | sea_order_change.go | — |
| SEA_ORDER_SPLIT_VERSION_CONFLICT | 409 Conflict | 数据已被更新，请刷新后重试 | sea_order_change.go | — |
| SEA_SHARED_CONTAINER_CONFLICT | 409 Conflict | 共享箱已被更新，请刷新后重试 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_EXCEEDED | 400 Bad Request | 共享箱或来源货物的分配数量已超出 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_EXISTS | 409 Conflict | 该运输执行下已存在相同箱号 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_INCOMPLETE | 400 Bad Request | 共享箱或来源货物尚未严格分配完整 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_INVALID_ARGUMENT | 400 Bad Request | 共享箱参数不合法 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_INVALID_REFERENCE | 400 Bad Request | 共享箱引用的订单、分单或货物不合法 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_NOT_FOUND | 404 Not Found | 共享箱不存在 | sea_cargo_allocation.go | — |
| SEA_SHARED_CONTAINER_STATUS_CONFLICT | 409 Conflict | 共享箱当前状态不允许该操作 | sea_cargo_allocation.go | — |
| SEA_TRANSPORT_EXECUTION_NOT_FOUND | 404 Not Found | 海运运输执行记录不存在 | sea_order_change.go | — |

## 单证

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| ORDER_CUSTOMER_CHANGE_WITH_HOUSE_BILL_BLOCKED | 409 Conflict | 存在以委托单位签发的分单，请先调整分单签发主体后再修改客户 | sea_document.go | — |
| ORDER_SHIPPING_DOCUMENT_INVALID_ARGUMENT | 400 Bad Request | 海运出口订单禁止使用旧提单接口，请使用海运单证接口 | sea_document.go | — |
| SEA_BILL_CONTENT_INVALID_ARGUMENT | 400 Bad Request | %s长度不能超过限制<br>件数不能为负数<br>毛重必须为有效的非负数值<br>体积必须为有效的非负数值 | sea_document.go | — |
| SEA_DOCUMENT_AMENDMENT_EMPTY | 400 Bad Request | 改单内容与当前不可变版本没有差异 | sea_document_change.go | — |
| SEA_DOCUMENT_BATCH_MEMBER_EXIT_BLOCKED | 409 Conflict | 共享主单批次存在其他成员订单，不能转为直单 | sea_master_bill.go | — |
| SEA_DOCUMENT_CHANGE_BLOCKED | 409 Conflict | 单证存在不可自动调整的下游事实，当前操作已阻断 | sea_document_change.go | — |
| SEA_DOCUMENT_INVALID_ARGUMENT | 400 Bad Request | 海运单证参数不合法<br>订单整单更新禁止直接提交分单变更，请使用专用单证命令<br>修改单证结构必须提供 expected_link_version<br>修改主单内容必须提供 expected_mbl_version | sea_document.go | — |
| SEA_DOCUMENT_MODE_CHANGE_CONFLICT | 409 Conflict | 单证模式或版本已变化，请刷新后重试 | sea_document_change.go | — |
| SEA_DOCUMENT_NO_ACTIVE_LINK | 400 Bad Request | 海运订单未关联活动主单 | sea_document.go | — |
| SEA_DOCUMENT_STRUCTURE_CONFLICT | 409 Conflict | 单证结构状态或版本已被修改，请刷新后重试 | sea_document.go | — |
| SEA_DOCUMENT_STRUCTURE_INVALID | 400 Bad Request | 当前单证结构不允许该操作<br>海运出口订单单证模式必填<br>DIRECT 直单下不得包含分单<br>HOUSE 单证结构必须提供唯一分单 | sea_document.go | — |
| SEA_DOCUMENT_VERSION_NOT_FOUND | 404 Not Found | 单证不可变版本不存在 | sea_document_change.go | — |
| SEA_DOCUMENT_VOIDED | 409 Conflict | 单证已作废，不能再次修改 | sea_document_change.go | — |
| SEA_HOUSE_BILL_BATCH_NO_DUPLICATE | 409 Conflict | 分单号在该主单批次内已存在<br>分单号 %s 在该主单批次内已存在 | sea_master_bill.go | — |
| SEA_HOUSE_BILL_CONFLICT | 409 Conflict | 海运分单已被更新，请刷新后重试 | sea_document.go | — |
| SEA_HOUSE_BILL_EXISTS | 409 Conflict | 海运分单号已存在 | sea_document.go | — |
| SEA_HOUSE_BILL_INVALID_ARGUMENT | 400 Bad Request | 海运分单参数不合法<br>未找到所属公司或总部签发主体<br>海运分单号不能为空<br>海运分单号长度不能超过 128 个字符<br>分单号不能为空<br>签发主体来源无效<br>本公司或委托单位签发时不得指定其他合作伙伴<br>其他主体签发必须选择合作伙伴<br>分单备注不能超过 500 字符 | sea_document.go | — |
| SEA_HOUSE_BILL_NOT_FOUND | 404 Not Found | 海运分单不存在 | sea_document.go | — |
| SEA_HOUSE_BILL_STATUS_CONFLICT | 409 Conflict | 海运分单状态或版本已被修改，请刷新后重试 | sea_document.go | — |
| SEA_MASTER_BILL_BATCH_DIRECT_BLOCKED | 409 Conflict | 该主单已被直单订单占用，如需拼单请先将其转为分单 | sea_master_bill.go | — |
| SEA_MASTER_BILL_CONFIRMATION_REQUIRED | 409 Conflict | 发现已有海运主单，请确认关联 | sea_master_bill.go | — |
| SEA_MASTER_BILL_CONFLICT | 409 Conflict | 海运主单已被更新，请刷新后重试 | sea_document.go | — |
| SEA_MASTER_BILL_CORRECTION_BLOCKED | 409 Conflict | 共享主单禁止直接修改主单号或签发主体 | sea_master_bill.go | — |
| SEA_MASTER_BILL_EXISTS | 409 Conflict | 海运主单已存在 | sea_master_bill.go | — |
| SEA_MASTER_BILL_INVALID_ARGUMENT | 400 Bad Request | 海运主单参数不合法<br>海运出口订单必须填写主单号<br>主单号仅允许包含英文字母和数字 | sea_master_bill.go | — |
| SEA_MASTER_BILL_NOT_FOUND | 404 Not Found | 海运主单不存在 | sea_master_bill.go | — |
| SEA_MASTER_BILL_STATUS_CONFLICT | 409 Conflict | 海运主单状态或版本已被修改，请刷新后重试 | sea_master_bill.go | — |
| SEA_MASTER_BILL_VOYAGE_CONFLICT | 409 Conflict | 海运主单航程信息与本票不一致 | sea_master_bill.go | — |
| SEA_ORDER_BATCH_REQUIRES_HOUSE | 400 Bad Request | 该主单已存在共享批次，本单必须签发分单 | sea_master_bill.go | — |

## 财务

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| BILLED_FEE_BILL_LOCKED | 409 Conflict | 仅草稿账单中的费用允许修改 | finance_custom_setting.go | — |
| BILLED_FEE_CURRENCY_CONFLICT | 409 Conflict | 多费用账单不允许单独修改其中一条费用的币种 | finance_custom_setting.go | — |
| BILLED_FEE_EDIT_DISABLED | 409 Conflict | 账单创建后不允许修改费用 | finance_custom_setting.go | — |
| BILLED_FEE_FIELD_FORBIDDEN | 403 Forbidden | 当前字段不允许在账单创建后修改 | finance_custom_setting.go | — |
| BILLING_UNIT_CODE_EXISTS | 409 Conflict | 计费单位代码已存在 | fee_catalog.go | — |
| BILLING_UNIT_NOT_FOUND | 404 Not Found | 计费单位不存在 | fee_catalog.go | — |
| EXCHANGE_RATE_CURRENCY_INVALID | 400 Bad Request | 汇率币种必须是启用的 ISO 币种 | exchange_rate.go | — |
| EXCHANGE_RATE_IMPORT_EMPTY | 400 Bad Request | 汇率导入文件没有数据 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_EXPIRED | 409 Conflict | 汇率导入预检已过期，请重新上传 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_FILE_INVALID | 400 Bad Request | 汇率导入文件不合法 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_IDEMPOTENCY_CONFLICT | 409 Conflict | 汇率导入幂等键已被其他请求使用 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_INVALID | 409 Conflict | 汇率导入预检存在错误，不能确认 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_NOT_FOUND | 404 Not Found | 汇率导入预检不存在 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_STALE | 409 Conflict | 汇率数据已变化，请重新上传预检 | exchange_rate_import.go | — |
| EXCHANGE_RATE_IMPORT_TOO_MANY_ROWS | 400 Bad Request | 汇率导入最多支持 500 条数据 | exchange_rate_import.go | — |
| EXCHANGE_RATE_INVALID_ARGUMENT | 400 Bad Request | 汇率设置字段不合法 | exchange_rate.go | — |
| EXCHANGE_RATE_NOT_FOUND | 404 Not Found | 汇率设置不存在 | exchange_rate.go | — |
| EXCHANGE_RATE_ORGANIZATION_INVALID | 400 Bad Request | 当前组织未配置有效本币 | exchange_rate.go | — |
| EXCHANGE_RATE_OVERLAP | 409 Conflict | 汇率生效周与现有设置冲突，请刷新后重试 | exchange_rate.go | — |
| EXCHANGE_RATE_PERMISSION_DENIED | 403 Forbidden | 无权维护当前组织汇率 | exchange_rate.go | — |
| EXCHANGE_RATE_QUOTE_UNAVAILABLE | 400 Bad Request | 外汇牌价抓取失败，请稍后重试或手工录入汇率 | exchange_rate.go | — |
| EXCHANGE_RATE_SYNC_ROWS_EMPTY | 400 Bad Request | 汇率同步数据为空 | exchange_rate.go | — |
| EXCHANGE_RATE_SYNC_ROWS_INVALID | 400 Bad Request | 汇率同步数据不合法 | exchange_rate.go | — |
| EXCHANGE_RATE_SYNC_TARGET_INVALID | 400 Bad Request | 汇率同步目标周不合法 | exchange_rate.go | — |
| FEE_CATALOG_INVALID_ARGUMENT | 400 Bad Request | 费用设置字段不合法 | fee_catalog.go | — |
| FEE_CATALOG_REFERENCE_INVALID | 400 Bad Request | 费用设置引用的基础资料不存在、已停用或不属于当前组织 | fee_catalog.go | — |
| FEE_EXCHANGE_RATE_CONFLICT | 409 Conflict | 汇率日期命中多条生效汇率 | exchange_rate.go | — |
| FEE_EXCHANGE_RATE_MISSING | 400 Bad Request | 汇率日期未命中生效汇率 | exchange_rate.go | ERROR_REASON_FEE_EXCHANGE_RATE_MISSING |
| FEE_LEDGER_PREFERENCE_CONFLICT | 409 Conflict | 费用明细表头设置已被更新，请刷新后重试 | fee_ledger_preference.go | — |
| FEE_LEDGER_PREFERENCE_INVALID_ARGUMENT | 400 Bad Request | 费用明细表头设置不合法 | fee_ledger_preference.go | — |
| FEE_SETTING_CODE_EXISTS | 409 Conflict | 费用代码已存在 | fee_catalog.go | — |
| FEE_SETTING_NOT_FOUND | 404 Not Found | 费用设置不存在 | fee_catalog.go | — |
| FINANCE_BILL_BATCH_CONFLICT | 409 Conflict | 批量建单幂等键已被其他请求使用 | finance_bill.go | — |
| FINANCE_BILL_BATCH_MISMATCH | 400 Bad Request | 批量账单分组资料与服务端预览不一致 | finance_bill.go | — |
| FINANCE_BILL_FEE_INVALID | 409 Conflict | 所选费用必须为已确认状态且尚未进入其他账单 | finance_bill.go | ERROR_REASON_FINANCE_BILL_FEE_INVALID |
| FINANCE_BILL_FEE_MISMATCH | 400 Bad Request | 同一账单的费用必须具有相同收付方向、结算单位、币种和本币 | finance_bill.go | — |
| FINANCE_BILL_GROUPING_MODE_UNSUPPORTED | 400 Bad Request | 当前阶段暂不支持对冲建账模式 | finance_bill.go | — |
| FINANCE_BILL_IDEMPOTENCY_CONFLICT | 409 Conflict | 账单请求幂等键已被其他请求使用 | finance_bill.go | — |
| FINANCE_BILL_INVALID_ARGUMENT | 400 Bad Request | 账单字段不合法 | finance_bill.go | — |
| FINANCE_BILL_INVALID_TRANSITION | 409 Conflict | 当前账单状态不允许执行该操作 | finance_bill.go | — |
| FINANCE_BILL_MIXED_DIRECTION | 400 Bad Request | 普通账单需将应收、应付分别建账，请先完成一个方向，再创建另一个方向 | finance_bill.go | — |
| FINANCE_BILL_NOT_FOUND | 404 Not Found | 账单不存在 | finance_bill.go | — |
| FINANCE_BILL_PREVIEW_STALE | 409 Conflict | 费用或拆单结果已变化，请重新预览 | finance_bill.go | ERROR_REASON_FINANCE_BILL_PREVIEW_STALE |
| FINANCE_BILL_SETTLEMENT_ACCOUNT_INVALID | 400 Bad Request | 结算账户与账单结算单位、方向、币种或启用状态不匹配 | finance_bill.go | ERROR_REASON_FINANCE_BILL_SETTLEMENT_ACCOUNT_INVALID |
| FINANCE_BILL_VERSION_CONFLICT | 409 Conflict | 账单已被其他操作人修改，请刷新后重试 | finance_bill.go | — |
| FINANCE_CASHFLOW_CASUAL_SUPPLIER_ACCOUNT_REQUIRED | 400 Bad Request | 向散客供应商出款时，对方收款账户为必填项 | finance_cashflow.go | — |
| FINANCE_CASHFLOW_IDEMPOTENCY_CONFLICT | 409 Conflict | 流水请求幂等键已被其他请求使用 | finance_cashflow.go | — |
| FINANCE_CASHFLOW_INVALID_ARGUMENT | 400 Bad Request | 收付流水字段不合法 | finance_cashflow.go | — |
| FINANCE_CASHFLOW_INVALID_TRANSITION | 409 Conflict | 当前流水状态不允许执行该操作 | finance_cashflow.go | — |
| FINANCE_CASHFLOW_NOT_FOUND | 404 Not Found | 收付流水不存在 | finance_cashflow.go | — |
| FINANCE_CASHFLOW_RATE_OVERRIDE_FORBIDDEN | 403 Forbidden | 无权手工覆盖资金流水汇率 | finance_cashflow.go | — |
| FINANCE_CASHFLOW_VERSION_CONFLICT | 409 Conflict | 收付流水已被其他操作人修改 | finance_cashflow.go | — |
| FINANCE_CUSTOM_SETTING_CONFLICT | 409 Conflict | 财务自定义设置已被更新，请刷新后重试 | finance_custom_setting.go | — |
| FINANCE_CUSTOM_SETTING_INVALID_ARGUMENT | 400 Bad Request | 财务自定义设置字段不合法 | finance_custom_setting.go | — |
| FINANCE_INVOICE_BILL_INVALID | 409 Conflict | 所选账单必须已确认且未进入其他有效开票记录 | finance_invoice.go | — |
| FINANCE_INVOICE_BILL_MISMATCH | 400 Bad Request | 同一开票记录的账单必须具有相同收付方向、结算单位和币种 | finance_invoice.go | — |
| FINANCE_INVOICE_IDEMPOTENCY_CONFLICT | 409 Conflict | 开票请求幂等键已被其他请求使用 | finance_invoice.go | — |
| FINANCE_INVOICE_INVALID_ARGUMENT | 400 Bad Request | 开票记录字段不合法 | finance_invoice.go | — |
| FINANCE_INVOICE_INVALID_TRANSITION | 409 Conflict | 当前开票记录状态不允许执行该操作 | finance_invoice.go | — |
| FINANCE_INVOICE_NOT_FOUND | 404 Not Found | 开票记录不存在 | finance_invoice.go | — |
| FINANCE_INVOICE_PROFILE_REQUIRED | 409 Conflict | 请选择该结算单位下启用且完整的开票抬头 | finance_invoice.go | ERROR_REASON_FINANCE_INVOICE_PROFILE_REQUIRED |
| FINANCE_INVOICE_RED_NO_EXISTS | 409 Conflict | 红字发票号码已被其他开票记录使用 | finance_invoice.go | — |
| FINANCE_INVOICE_TAX_NO_EXISTS | 409 Conflict | 税控发票号码已被其他开票记录使用 | finance_invoice.go | — |
| FINANCE_INVOICE_VERSION_CONFLICT | 409 Conflict | 开票记录已被其他操作人修改，请刷新后重试 | finance_invoice.go | — |
| FINANCE_LEDGER_INVALID_ARGUMENT | 400 Bad Request | 费用台账查询条件不合法 | settlement.go | — |
| FINANCE_NETTING_BALANCE | 409 Conflict | 账单可用余额不足以完成本次对冲 | finance_netting.go | — |
| FINANCE_NETTING_BILL_NOT_CONFIRMED | 409 Conflict | 来源账单必须均为已确认状态方可确认对冲单 | finance_netting.go | — |
| FINANCE_NETTING_BILL_VERSION_CONFLICT | 409 Conflict | 来源账单已被其他操作人修改，请刷新后重试 | finance_netting.go | — |
| FINANCE_NETTING_DIRECTION | 400 Bad Request | 对冲需至少包含一张应收账单和一张应付账单 | finance_netting.go | — |
| FINANCE_NETTING_IDEMPOTENCY | 409 Conflict | 对冲请求幂等键已被其他请求使用 | finance_netting.go | — |
| FINANCE_NETTING_INVALID | 400 Bad Request | 对冲参数不合法 | finance_netting.go | — |
| FINANCE_NETTING_MISMATCH | 400 Bad Request | 对冲账单必须属于同一组织、同一结算单位并具有相同账单币种，或不在当前权限范围内 | finance_netting.go | — |
| FINANCE_NETTING_NOT_FOUND | 404 Not Found | 对冲单不存在 | finance_netting.go | — |
| FINANCE_NETTING_SINGLE_DIRECTION | 400 Bad Request | 对冲建账需在同一结算单位、同一账单币种下同时包含应收和应付费用，单方向费用请使用普通账单 | finance_netting.go | — |
| FINANCE_NETTING_TOO_MANY_BILLS | 400 Bad Request | 同一结算单位与账单币种下的候选账单数量超过上限，请缩小范围后重试 | finance_netting.go | — |
| FINANCE_NETTING_TRANSITION | 409 Conflict | 当前对冲状态不允许执行该操作 | finance_netting.go | — |
| FINANCE_NETTING_VERSION_CONFLICT | 409 Conflict | 对冲单已被其他操作人修改，请刷新后重试 | finance_netting.go | — |
| FINANCE_VERIFICATION_BALANCE | 409 Conflict | 核销金额超过资金或账单未核销余额 | finance_verification.go | — |
| FINANCE_VERIFICATION_IDEMPOTENCY | 409 Conflict | 幂等键已用于不同的核销请求 | finance_verification.go | — |
| FINANCE_VERIFICATION_INVALID | 400 Bad Request | 核销参数不合法 | finance_verification.go | — |
| FINANCE_VERIFICATION_MISMATCH | 400 Bad Request | 核销双方方向、结算单位或币种不一致 | finance_verification.go | — |
| FINANCE_VERIFICATION_NOT_FOUND | 404 Not Found | 核销记录不存在 | finance_verification.go | — |
| FINANCE_VERIFICATION_TRANSITION | 409 Conflict | 当前核销状态不允许该操作 | finance_verification.go | — |
| TAXABLE_SERVICE_NAME_EXISTS | 409 Conflict | 货物或应税劳务名称已存在 | fee_catalog.go | — |
| TAXABLE_SERVICE_NOT_FOUND | 404 Not Found | 货物或应税劳务名称不存在 | fee_catalog.go | — |

## 提成

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| COMMISSION_BASE_CURRENCY_MISMATCH | 409 Conflict | 补录费用本位币与历史提成快照本位币不一致，不支持跨本位币换算 | finance_commission.go | — |
| COMMISSION_CALCULATION_VERSION_UNSUPPORTED | 409 Conflict | 提成计算版本不受支持，无法判定应付成本影响 | finance_commission.go | — |
| COMMISSION_SNAPSHOT_UNAVAILABLE | 409 Conflict | 历史提成复算快照不可用，无法处理锁后费用补录 | finance_commission.go | — |
| FINANCE_COMMISSION_ADJUSTMENT_CANCEL_NOT_ALLOWED | 409 Conflict | 系统冲减建议确认或扣回后不允许取消 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_ADJUSTMENT_CANCEL_NOT_ALLOWED |
| FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS | 409 Conflict | 冲减后的有效提成金额不能小于零 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_ADJUSTMENT_EXCEEDS |
| FINANCE_COMMISSION_ADJUSTMENT_INVALID | 400 Bad Request | 提成调整参数不合法 | finance_commission.go | — |
| FINANCE_COMMISSION_ADJUSTMENT_NOT_FOUND | 404 Not Found | 提成调整记录不存在 | finance_commission.go | — |
| FINANCE_COMMISSION_ADJUSTMENT_TRANSITION | 409 Conflict | 当前提成调整状态不允许该操作 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_ADJUSTMENT_TRANSITION |
| FINANCE_COMMISSION_APPLICATION_CONFLICT | 409 Conflict | 该月申请已存在，请在申请详情中查看进度或重新提交 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_EMPLOYEE_INVALID | 409 Conflict | 当前用户不是当前组织的有效成员 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_EMPTY | 409 Conflict | 截至上一自然月末暂无可申请的合格提成 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_INVALID | 400 Bad Request | 提成申请参数不合法 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_NOT_FOUND | 404 Not Found | 提成申请不存在 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_NO_VALID_LINES | 409 Conflict | 申请内已无有效明细，无法重新提交 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_SOURCE_CONFLICT | 409 Conflict | 部分提成事实已变化或已被占用，请刷新后重试 | finance_commission_application.go | — |
| FINANCE_COMMISSION_APPLICATION_STATUS_CONFLICT | 409 Conflict | 该申请已被处理，请刷新后重试 | finance_commission_application.go | — |
| FINANCE_COMMISSION_DUPLICATE | 409 Conflict | 该核销、员工与人员角色已存在未取消提成记录 | finance_commission.go | — |
| FINANCE_COMMISSION_EMPLOYEE_ROLE | 409 Conflict | 所选员工未在客户档案中担任规则指定角色 | finance_commission.go | — |
| FINANCE_COMMISSION_EXPORT_LIMIT | 400 Bad Request | 提成导出行数超过单次上限，请缩小筛选范围后重试 | finance_commission.go | — |
| FINANCE_COMMISSION_INVALID | 400 Bad Request | 提成参数不合法 | finance_commission.go | — |
| FINANCE_COMMISSION_NOT_FOUND | 404 Not Found | 提成记录不存在 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_ASSIGNMENT_INVALID | 400 Bad Request | 提成方案员工分配参数不合法 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_CONFLICT | 409 Conflict | 提成规则名称已存在或版本已变化 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_RULE_CONFLICT |
| FINANCE_COMMISSION_RULE_EMPLOYEE_INVALID | 409 Conflict | 所选员工不是当前组织的有效成员 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_ENABLE_NO_EMPLOYEE | 409 Conflict | 启用提成方案前必须至少分配一名员工 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_INTERVAL_OVERLAP | 409 Conflict | 同一员工同一身份在重叠期间只能属于一个启用方案 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_INVALID | 400 Bad Request | 提成规则字段不合法 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_LEGACY_READ_ONLY | 409 Conflict | 迁移前的历史旧规则为只读，请复制为新方案后再使用 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_LOCKED_PARAMS | 409 Conflict | 方案已生效，人员身份、计提口径、比例和起始日不可原地修改，也不能停用，请复制为新方案 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_MEMBER_ASSIGNMENT | 409 Conflict | 员工仍有当前或未来的提成方案分配，请先终止分配后再停用成员<br>员工仍有当前或未来的提成方案分配（%s），请先以当天或未来的日期终止分配后再停用成员 | finance_commission.go、finance_commission_rules.go | — |
| FINANCE_COMMISSION_RULE_NOT_FOUND | 404 Not Found | 提成规则不存在 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_NOT_RESOLVED | 409 Conflict | 来源归属日期未唯一命中该员工该身份的有效提成方案与员工分配 | finance_commission.go | — |
| FINANCE_COMMISSION_RULE_RETROACTIVE | 409 Conflict | 已生效方案的名单与区间变更只能选择当天或未来的生效日期 | finance_commission.go | — |
| FINANCE_COMMISSION_SOURCE | 409 Conflict | 仅有效应收核销可计提，且必须存在可计算的已实现收入 | finance_commission.go | — |
| FINANCE_COMMISSION_SOURCE_CHANGED | 409 Conflict | 提成来源数据已变化，请取消当前草稿并重新生成 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_SOURCE_CHANGED |
| FINANCE_COMMISSION_TRANSITION | 409 Conflict | 当前提成状态不允许该操作 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_TRANSITION |
| FINANCE_COMMISSION_UNCONFIRMED_FEES | 409 Conflict | 关联订单仍有未建账费用，请先建账或删除后再确认提成 | finance_commission.go | ERROR_REASON_FINANCE_COMMISSION_UNCONFIRMED_FEES |

## 往来单位

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| PARTNER_ACCOUNT_DEFAULT_CONFLICT | 409 Conflict | 同一往来单位、币种和用途只能有一个默认结算账户 | partner_account.go | — |
| PARTNER_ACCOUNT_INVALID_ARGUMENT | 400 Bad Request | 结算账户字段不合法 | partner_account.go | — |
| PARTNER_ACCOUNT_NOT_FOUND | 404 Not Found | 结算账户不存在 | partner_account.go | — |
| PARTNER_ALIAS_EXISTS | 409 Conflict | 往来单位别名重复 | partner.go | — |
| PARTNER_ATTACHMENT_EXISTS | 409 Conflict | 附件幂等键已存在 | partner_attachment.go | — |
| PARTNER_ATTACHMENT_INVALID_ARGUMENT | 400 Bad Request | 附件字段不合法 | partner_attachment.go | — |
| PARTNER_ATTACHMENT_NOT_FOUND | 404 Not Found | 附件不存在 | partner_attachment.go | — |
| PARTNER_BLACKLISTED_ROLE | 400 Bad Request | 清除黑名单前不能移除已拉黑角色 | partner.go | — |
| PARTNER_BLACKLIST_REASON_REQUIRED | 400 Bad Request | 黑名单变更原因不能为空 | partner.go | — |
| PARTNER_BLACKLIST_ROLE_REQUIRED | 400 Bad Request | 往来单位没有指定的黑名单角色 | partner.go | — |
| PARTNER_CODE_EXISTS | 409 Conflict | 往来单位编码已存在 | partner.go | — |
| PARTNER_CONTRACT_INVALID_ARGUMENT | 400 Bad Request | 合同字段不合法 | partner_contract.go | — |
| PARTNER_CONTRACT_NOT_FOUND | 404 Not Found | 合同不存在 | partner_contract.go | — |
| PARTNER_CONTRACT_NO_EXISTS | 409 Conflict | 合同编号已存在 | partner_contract.go | — |
| PARTNER_CONTRACT_STATUS_CONFLICT | 409 Conflict | 合同状态不允许该变更 | partner_contract.go | — |
| PARTNER_CREDIT_LIMIT_EXCEEDED | 400 Bad Request | 该客户应收未核销总额已超出信用额度，系统已限制选择 | partner_credit.go | ERROR_REASON_PARTNER_CREDIT_LIMIT_EXCEEDED |
| PARTNER_INVALID_ARGUMENT | 400 Bad Request | 往来单位字段不合法<br>往来单位导入参数不合法 | partner.go | — |
| PARTNER_INVALID_ROLE | 400 Bad Request | 往来单位角色不合法 | partner.go | — |
| PARTNER_INVOICE_PROFILE_DEFAULT_REQUIRED | 409 Conflict | 请先将其他启用抬头设为默认抬头 | partner_invoice_profile.go | — |
| PARTNER_INVOICE_PROFILE_INVALID_ARGUMENT | 400 Bad Request | 开票抬头字段不合法 | partner_invoice_profile.go | — |
| PARTNER_INVOICE_PROFILE_NOT_FOUND | 404 Not Found | 开票抬头不存在 | partner_invoice_profile.go | — |
| PARTNER_INVOICE_PROFILE_TITLE_EXISTS | 409 Conflict | 该客户已存在同名开票抬头 | partner_invoice_profile.go | — |
| PARTNER_INVOICE_PROFILE_VERSION_CONFLICT | 409 Conflict | 开票抬头已被其他操作人修改，请刷新后重试 | partner_invoice_profile.go | — |
| PARTNER_NAME_EXISTS | 409 Conflict | 往来单位名称已存在 | partner.go | — |
| PARTNER_NOT_FOUND | 404 Not Found | 往来单位不存在 | partner.go | — |
| PARTNER_PRIMARY_CONTACT_CONFLICT | 400 Bad Request | 只能设置一个主联系人 | partner.go | — |
| PARTNER_ROLE_REQUIRED | 400 Bad Request | 往来单位至少需要一个有效角色 | partner.go | — |
| PARTNER_SETTLEMENT_RULE_EXISTS | 409 Conflict | 结算规则已存在 | partner_settlement_rule.go | — |
| PARTNER_SETTLEMENT_RULE_INVALID_ARGUMENT | 400 Bad Request | 结算规则字段不合法 | partner_settlement_rule.go | — |
| PARTNER_SETTLEMENT_RULE_NOT_FOUND | 404 Not Found | 结算规则不存在 | partner_settlement_rule.go | — |
| PARTNER_SHIPPING_PRESET_INVALID_ARGUMENT | 400 Bad Request | 常用单证预设字段不合法 | partner_shipping_preset.go | — |
| PARTNER_SHIPPING_PRESET_NOT_FOUND | 404 Not Found | 常用单证预设不存在 | partner_shipping_preset.go | — |
| PARTNER_SUPPLIER_ROLE_REQUIRED | 400 Bad Request | 往来单位没有供应商角色 | partner.go | — |
| PARTNER_USCC_EXISTS | 409 Conflict | 统一社会信用代码已存在 | partner.go | — |

## 权限

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| ADMIN_INVALID_ARGUMENT | 400 Bad Request | 管理参数不合法 | admin.go | — |
| ADMIN_ORGANIZATION_BASE_CURRENCY_IMMUTABLE | 400 Bad Request | 组织本币一旦设定不可变更 | admin_organization.go | — |
| ADMIN_ORGANIZATION_CODE_EXISTS | 409 Conflict | 组织编码已存在 | admin_organization.go | — |
| ADMIN_ORGANIZATION_CURRENCY_INVALID | 400 Bad Request | 组织本币必须是启用的 ISO 币种 | admin_organization.go | — |
| ADMIN_ORGANIZATION_HIERARCHY_INVALID | 400 Bad Request | 组织层级不合法 | admin_organization.go | — |
| ADMIN_ORGANIZATION_NOT_FOUND | 404 Not Found | 组织不存在 | admin_organization.go | — |
| ADMIN_ORGANIZATION_PARENT_REQUIRED | 400 Bad Request | 新建组织必须指定上级组织 | admin_organization.go | — |
| ADMIN_PERMISSION_INVALID | 400 Bad Request | 权限不存在或不属于当前请求 | admin_role.go | — |
| ADMIN_PRIVILEGE_ESCALATION_DENIED | 403 Forbidden | 不能分配超出自身权限范围的角色 | admin_role.go | — |
| ADMIN_ROLE_ANCHOR_INVALID | 400 Bad Request | 角色只能在公司/系统管理维护 | admin_role.go | — |
| ADMIN_ROLE_ASSIGNED | 409 Conflict | 该角色已分配成员，请先移除后重试 | admin_role.go | — |
| ADMIN_ROLE_CODE_EXISTS | 409 Conflict | 角色编码已存在 | admin_role.go | — |
| ADMIN_ROLE_IN_USE | 409 Conflict | 该角色仍被其他数据引用，无法删除 | admin_role.go | — |
| ADMIN_ROLE_NOT_FOUND | 404 Not Found | 角色不存在 | admin_role.go | — |
| ADMIN_ROLE_PROTECTED | 403 Forbidden | 系统管理员角色不允许删除 | admin_role.go | — |
| ADMIN_ROLE_SCOPE_DISABLED | 400 Bad Request | 仅本人数据范围已停用，请明确选择其他数据范围 | admin_role.go | — |
| ADMIN_USERNAME_EXISTS | 409 Conflict | 用户名已存在 | admin_user.go | — |
| ADMIN_USER_AUTHORIZATION_REQUIRED | 400 Bad Request | 外部身份账号必须通过身份授权流程启用 | admin_user.go | — |
| ADMIN_USER_LAST_MEMBERSHIP | 400 Bad Request | 在职用户必须保留至少一个有效组织；请先加入新组织或办理离职 | admin_user.go | — |
| ADMIN_USER_MEMBERSHIP_EXISTS | 409 Conflict | 用户已属于该组织 | admin_user_membership.go | — |
| ADMIN_USER_MEMBERSHIP_NOT_FOUND | 404 Not Found | 用户组织成员关系不存在 | admin_user_membership.go | — |
| ADMIN_USER_NOT_FOUND | 404 Not Found | 用户不存在或不在当前工作台范围 | admin_user.go | — |
| ADMIN_USER_SELF_DELETE | 400 Bad Request | 不能移除当前登录账号或为其办理离职 | admin_user.go | — |
| ADMIN_USER_TERMINATION_REQUIRED | 400 Bad Request | 停用员工请使用办理离职 | admin_user.go | — |
| AUTH_DINGTALK_ALREADY_REGISTERED | 409 Conflict | 该钉钉账号已完成注册，请直接登录 | auth.go | — |
| AUTH_DINGTALK_AUTHORIZATION_PENDING | 403 Forbidden | 账号已登记，请联系管理员分配角色并启用账号 | auth.go | — |
| AUTH_DINGTALK_CODE_INVALID | 401 Unauthorized | 钉钉登录凭证已失效，请重新扫码 | auth.go | — |
| AUTH_DINGTALK_DISABLED | 503 Service Unavailable | 钉钉认证未启用 | auth.go | — |
| AUTH_DINGTALK_LOGIN_FAILED | 401 Unauthorized | 钉钉登录失败 | auth.go | — |
| AUTH_DINGTALK_NOT_REGISTERED | 401 Unauthorized | 当前人员尚未登记 | auth.go | — |
| AUTH_DINGTALK_ORGANIZATION_MISMATCH | 403 Forbidden | 当前钉钉账号不属于本企业，无法继续注册 | auth.go | — |
| AUTH_DINGTALK_PERMISSION_DENIED | 401 Unauthorized | 钉钉应用无权读取成员信息，请检查应用权限 | auth.go | — |
| AUTH_DINGTALK_REGISTRATION_EXPIRED | 401 Unauthorized | 钉钉身份确认已过期，请重新扫码 | auth.go | — |
| AUTH_DINGTALK_STATE_INVALID | 401 Unauthorized | 钉钉验证状态已失效，请重新扫码 | auth.go | — |
| AUTH_INVALID_CREDENTIALS | 401 Unauthorized | 用户名或密码错误 | auth.go | — |
| AUTH_ORGANIZATION_FORBIDDEN | 403 Forbidden | 无权访问该组织<br>无该组织的成员资格或已停用 | auth.go | — |
| AUTH_ORGANIZATION_INVALID | 400 Bad Request | 所选组织无效或不可用 | auth.go | — |
| AUTH_PERMISSION_DENIED | 403 Forbidden | 无权执行此操作 | auth.go | — |
| AUTH_SESSION_EXPIRED | 401 Unauthorized | 登录已过期 | auth.go | — |
| AUTH_SESSION_REQUIRED | 401 Unauthorized | 请先登录 | auth.go | — |
| AUTH_WECOM_AUTHORIZATION_PENDING | 403 Forbidden | 账号已登记，请联系管理员分配角色并启用账号 | auth.go | — |
| AUTH_WECOM_CODE_INVALID | 401 Unauthorized | 企业微信登录凭证已失效，请重新扫码 | auth.go | — |
| AUTH_WECOM_DISABLED | 503 Service Unavailable | 企业微信登录未启用 | auth.go | — |
| AUTH_WECOM_LOGIN_FAILED | 401 Unauthorized | 企业微信登录失败 | auth.go | — |
| AUTH_WECOM_PERMISSION_DENIED | 401 Unauthorized | 企业微信应用无权读取成员信息，请检查应用可见范围和通讯录权限 | auth.go | — |
| AUTH_WECOM_STATE_INVALID | 401 Unauthorized | 企业微信登录状态已失效，请重新扫码 | auth.go | — |
| AUTH_WECOM_TRUSTED_IP_REQUIRED | 401 Unauthorized | 企业微信拒绝当前服务器 IP，请在应用管理中配置企业可信 IP | auth.go | — |
| DINGTALK_INVITATION_EXISTS | 409 Conflict | 该手机号在目标组织已存在有效定向邀请 | dingtalk_invitation.go | — |
| DINGTALK_INVITATION_EXPIRED | 400 Bad Request | 邀请已过期 | dingtalk_invitation.go | — |
| DINGTALK_INVITATION_NOT_CONSUMABLE | 409 Conflict | 邀请不可消费 | dingtalk_invitation.go | — |
| DINGTALK_INVITATION_NOT_FOUND | 404 Not Found | 邀请不存在或已失效 | dingtalk_invitation.go | — |
| DINGTALK_INVITATION_NOT_REVOCABLE | 409 Conflict | 邀请已消费或已撤销，不能再撤销 | dingtalk_invitation.go | — |
| DINGTALK_INVITATION_REVOKED | 400 Bad Request | 邀请已撤销 | dingtalk_invitation.go | — |
| DINGTALK_REGISTRATION_ALREADY_PROCESSED | 409 Conflict | 该注册申请已处理 | dingtalk_invitation.go | — |
| DINGTALK_REGISTRATION_COMPANY_REQUIRED | 400 Bad Request | 请先将注册申请转交到具体公司，再分配角色并同意 | dingtalk_registration.go | — |
| DINGTALK_REGISTRATION_NOT_FOUND | 404 Not Found | 注册申请不存在或已处理 | dingtalk_invitation.go | — |
| DINGTALK_REGISTRATION_ORGANIZATION_INVALID | 400 Bad Request | 所选组织无效或不可选 | dingtalk_invitation.go | — |
| DINGTALK_REGISTRATION_REASON_MISSING | 400 Bad Request | 原因说明不能为空 | dingtalk_invitation.go | — |
| DINGTALK_REGISTRATION_TRANSFER_SAME | 400 Bad Request | 不能转派至当前已申请组织 | dingtalk_invitation.go | — |
| OPERATING_COMPANY_REQUIRED | 403 Forbidden | 系统管理工作台不承载经营数据，请切换到具体公司后再维护经营数据 | organization.go | — |

## 平台

| 错误码 | 语义（HTTP 类别） | 中文消息 | 定义位置 | 关联 proto ErrorReason |
| --- | --- | --- | --- | --- |
| BACKGROUND_TASK_EXISTS | 409 Conflict | 后台任务已存在 | background_task.go | — |
| BACKGROUND_TASK_INVALID_ARGUMENT | 400 Bad Request | 后台任务参数不合法 | background_task.go | — |
| BACKGROUND_TASK_INVALID_STATUS | 400 Bad Request | 后台任务状态不合法 | background_task.go | — |
| BACKGROUND_TASK_LEASE_MISMATCH | 409 Conflict | 后台任务租约不匹配或已失效 | background_task.go | — |
| BACKGROUND_TASK_NOT_FOUND | 404 Not Found | 后台任务不存在 | background_task.go | — |
| BACKGROUND_TASK_NOT_REQUEUEABLE | 409 Conflict | 后台任务当前状态不可回放 | background_task.go | — |
| BACKGROUND_TASK_NO_TASK | 404 Not Found | 没有可执行的后台任务 | background_task.go | — |
| BUSINESS_TAG_INVALID_ARGUMENT | 400 Bad Request | 业务标签参数不合法 | business_tag.go | — |
| CURRENCY_BASE_CANNOT_BE_DISABLED | 400 Bad Request | 本位币不可禁用 | reference_data.go | — |
| CURRENCY_NOT_FOUND | 404 Not Found | 货币币种不存在 | reference_data.go | — |
| ENTERPRISE_IMAGE_STORAGE_UNAVAILABLE | 503 Service Unavailable | 图片对象存储未配置 | enterprise_resource.go | — |
| ENTERPRISE_RESOURCE_IMPORT_AMBIGUOUS | 409 Conflict | 导入行匹配到多个现有资源，请先修正企业名称或业务代码 | enterprise_resource.go | — |
| ENTERPRISE_RESOURCE_INVALID_ARGUMENT | 400 Bad Request | 企业资源字段不合法 | enterprise_resource.go | — |
| ENTERPRISE_RESOURCE_NOT_FOUND | 404 Not Found | 企业资源不存在 | enterprise_resource.go | — |
| ENTERPRISE_TAG_GROUP_NOT_EMPTY | 409 Conflict | 标签组下仍有标签 | enterprise_resource.go | — |
| ENTERPRISE_TAG_IN_USE | 409 Conflict | 标签正在被使用，请先移除关联 | enterprise_resource.go | — |
| INDUSTRY_REFERENCE_CODE_EXISTS | 409 Conflict | 行业标准码已存在 | industry_reference.go | — |
| INDUSTRY_REFERENCE_NOT_FOUND | 404 Not Found | 行业主数据不存在 | industry_reference.go | — |
| MASTER_DATA_CODE_EXISTS | 409 Conflict | 主数据编码已存在 | masterdata.go | — |
| MASTER_DATA_INVALID_ARGUMENT | 400 Bad Request | 主数据字段不合法 | masterdata.go | — |
| MASTER_DATA_INVALID_KIND | 400 Bad Request | 主数据类型不合法 | masterdata.go | — |
| MASTER_DATA_NOT_FOUND | 404 Not Found | 主数据不存在 | masterdata.go | — |
| MASTER_DATA_SYSTEM_REQUIRED | 403 Forbidden | 公共主数据只能在系统管理工作台维护 | masterdata.go | — |
| NOTIFICATION_NOT_FOUND | 404 Not Found | 通知明细不存在 | notification.go | — |
| REFERENCE_DATA_INVALID_ARGUMENT | 400 Bad Request | 基础字典查询参数不合法 | reference_data.go | — |
| WORKBENCH_INVALID_ARGUMENT | 400 Bad Request | 工作台查询参数不合法 | workbench.go | — |

