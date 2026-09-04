package access

import (
	"fmt"
	"strings"
)

type Permission struct {
	Key         string
	Name        string
	Group       string
	Description string
	// Requires 声明勾选该权限时必须同时具备的基础权限（如“编辑客户”依赖“查看客户”），
	// 由角色保存与前端权限树联动共同消费。
	Requires []string
}

const (
	PlatformAccess = "system.platform.access"

	OrganizationRead            = "system.organization.read"
	OrganizationCreate          = "system.organization.create"
	OrganizationUpdate          = "system.organization.update"
	UserRead                    = "system.user.read"
	UserCreate                  = "system.user.create"
	UserUpdate                  = "system.user.update"
	UserTerminate               = "system.user.delete"
	UserAuthorizeWeCom          = "system.user.authorize_wecom"
	UserResetPassword           = "system.user.reset_password"
	RoleRead                    = "system.role.read"
	RoleCreate                  = "system.role.create"
	RoleUpdate                  = "system.role.update"
	PermissionRead              = "system.permission.read"
	AuditRead                   = "system.audit.read"
	FinanceExchangeRateRead     = "system.finance.exchange_rate.read"
	FinanceExchangeRateCreate   = "system.finance.exchange_rate.create"
	FinanceExchangeRateUpdate   = "system.finance.exchange_rate.update"
	FinanceExchangeRateDisable  = "system.finance.exchange_rate.disable"
	FinanceExchangeRateOverride = "system.finance.exchange_rate.override"
	FinanceFeeSettingRead       = "system.finance.fee_setting.read"
	FinanceFeeSettingCreate     = "system.finance.fee_setting.create"
	FinanceFeeSettingUpdate     = "system.finance.fee_setting.update"
	FinanceFeeRead              = "system.finance.fee.read"
	FinanceFeeTag               = "system.finance.fee.tag"
	FinanceBillRead             = "system.finance.bill.read"
	FinanceBillCreate           = "system.finance.bill.create"
	FinanceBillUpdate           = "system.finance.bill.update"
	FinanceBillConfirm          = "system.finance.bill.confirm"
	FinanceInvoiceRead          = "system.finance.invoice.read"
	FinanceInvoiceCreate        = "system.finance.invoice.create"
	FinanceInvoiceUpdate        = "system.finance.invoice.update"
	FinanceCashflowRead         = "system.finance.cashflow.read"
	FinanceCashflowCreate       = "system.finance.cashflow.create"
	FinanceCashflowUpdate       = "system.finance.cashflow.update"
	FinanceVerificationRead     = "system.finance.verification.read"
	FinanceVerificationCreate   = "system.finance.verification.create"
	FinanceVerificationReverse  = "system.finance.verification.reverse"
	FinanceCommissionRead       = "system.finance.commission.read"
	FinanceCommissionManage     = "system.finance.commission.manage"
	FinanceCommissionExport     = "system.finance.commission.export"

	PartnerRead                 = "business.partner.read"
	PartnerCreate               = "business.partner.create"
	PartnerUpdate               = "business.partner.update"
	PartnerBlacklist            = "business.partner.blacklist"
	PartnerImport               = "business.partner.import"
	PartnerExport               = "business.partner.export"
	PartnerAccountRead          = "business.partner.account.read"
	PartnerAccountCreate        = "business.partner.account.create"
	PartnerAccountUpdate        = "business.partner.account.update"
	PartnerContractRead         = "business.partner.contract.read"
	PartnerContractCreate       = "business.partner.contract.create"
	PartnerContractUpdate       = "business.partner.contract.update"
	PartnerSettlementRuleRead   = "business.partner.settlement_rule.read"
	PartnerSettlementRuleCreate = "business.partner.settlement_rule.create"
	PartnerSettlementRuleUpdate = "business.partner.settlement_rule.update"
	PartnerAttachmentRead       = "business.partner.attachment.read"
	PartnerAttachmentRegister   = "business.partner.attachment.register"
	PartnerShippingPresetRead   = "business.partner.shipping_preset.read"
	PartnerShippingPresetCreate = "business.partner.shipping_preset.create"
	PartnerShippingPresetUpdate = "business.partner.shipping_preset.update"
	PartnerAuditRead            = "business.partner.audit.read"
	PartnerAssignmentOptionRead = "business.partner.assignment_option.read"
	EnterpriseResourceRead      = "business.enterprise_resource.read"
	EnterpriseResourceCreate    = "business.enterprise_resource.create"
	EnterpriseResourceUpdate    = "business.enterprise_resource.update"
	EnterpriseResourceDelete    = "business.enterprise_resource.delete"

	MasterDataCurrencyRead             = "system.master_data.currency.read"
	MasterDataAdministrativeRegionRead = "system.master_data.administrative_region.read"
	MasterDataOptionRead               = "system.master_data.option.read"
	MasterDataItemRead                 = "system.master_data.item.read"
	MasterDataItemCreate               = "system.master_data.item.create"
	MasterDataItemUpdate               = "system.master_data.item.update"
	MasterDataItemImport               = "system.master_data.item.import"
	MasterDataPortRead                 = "system.master_data.port.read"
	MasterDataPortCreate               = "system.master_data.port.create"
	MasterDataPortUpdate               = "system.master_data.port.update"
	MasterDataAirportRead              = "system.master_data.airport.read"
	MasterDataAirportCreate            = "system.master_data.airport.create"
	MasterDataAirportUpdate            = "system.master_data.airport.update"
	MasterDataAirlineRead              = "system.master_data.airline.read"
	MasterDataAirlineCreate            = "system.master_data.airline.create"
	MasterDataAirlineUpdate            = "system.master_data.airline.update"
	MasterDataShippingLineRead         = "system.master_data.shipping_line.read"
	MasterDataShippingLineCreate       = "system.master_data.shipping_line.create"
	MasterDataShippingLineUpdate       = "system.master_data.shipping_line.update"
	MasterDataNumberRuleRead           = "system.master_data.number_rule.read"
	MasterDataNumberRuleCreate         = "system.master_data.number_rule.create"
	MasterDataNumberRuleUpdate         = "system.master_data.number_rule.update"
	TaskRead                           = "system.task.read"
	TaskRequeue                        = "system.task.requeue"
)

const UserAuthorizeDingTalk = "system.user.authorize_dingtalk"

type OrderBusinessType string

const (
	OrderBusinessSE OrderBusinessType = "SE"
	OrderBusinessSI OrderBusinessType = "SI"
	OrderBusinessAE OrderBusinessType = "AE"
	OrderBusinessAI OrderBusinessType = "AI"
)

type OrderOperation string

const (
	OrderRead                 OrderOperation = "read"
	OrderCreate               OrderOperation = "create"
	OrderUpdate               OrderOperation = "update"
	OrderTransition           OrderOperation = "transition"
	OrderMilestoneRead        OrderOperation = "milestone.read"
	OrderMilestoneSet         OrderOperation = "milestone.set"
	OrderAttachmentRead       OrderOperation = "attachment.read"
	OrderAttachmentRegister   OrderOperation = "attachment.register"
	OrderPersonnelRead        OrderOperation = "personnel.read"
	OrderPersonnelAssign      OrderOperation = "personnel.assign"
	OrderPersonnelRemove      OrderOperation = "personnel.remove"
	OrderContainerRead        OrderOperation = "container.read"
	OrderContainerCreate      OrderOperation = "container.create"
	OrderContainerUpdate      OrderOperation = "container.update"
	OrderContainerDelete      OrderOperation = "container.delete"
	OrderCargoItemRead        OrderOperation = "cargo_item.read"
	OrderCargoItemCreate      OrderOperation = "cargo_item.create"
	OrderCargoItemUpdate      OrderOperation = "cargo_item.update"
	OrderCargoItemDelete      OrderOperation = "cargo_item.delete"
	OrderAbnormalCaseRead     OrderOperation = "abnormal_case.read"
	OrderAbnormalCaseCreate   OrderOperation = "abnormal_case.create"
	OrderAbnormalCaseResolve  OrderOperation = "abnormal_case.resolve"
	OrderAbnormalCaseDelete   OrderOperation = "abnormal_case.delete"
	OrderReleasePodRead       OrderOperation = "release_pod.read"
	OrderReleasePodCreate     OrderOperation = "release_pod.create"
	OrderReleasePodUpdate     OrderOperation = "release_pod.update"
	OrderReleasePodTransition OrderOperation = "release_pod.transition"
	OrderReleasePodDelete     OrderOperation = "release_pod.delete"
	OrderFeeRead              OrderOperation = "fee.read"
	OrderFeeCreate            OrderOperation = "fee.create"
	OrderFeeUpdate            OrderOperation = "fee.update"
	OrderFeeDelete            OrderOperation = "fee.delete"
	OrderSplit                OrderOperation = "split"
	OrderReassign             OrderOperation = "reassign"
	OrderLock                 OrderOperation = "lock"
	OrderAmend                OrderOperation = "amend"
	OrderVoid                 OrderOperation = "void"
	OrderSwitch               OrderOperation = "switch"
)

var manifest = append([]Permission{
	{Key: PlatformAccess, Name: "访问管理后台", Group: "系统管理 · 平台", Description: "登录并访问管理后台"},
	{Key: OrganizationRead, Name: "查看组织", Group: "系统管理 · 组织", Description: "查看公司、部门和组的组织架构"},
	{Key: OrganizationCreate, Name: "新建组织", Group: "系统管理 · 组织", Description: "新建公司、部门或组", Requires: []string{OrganizationRead}},
	{Key: OrganizationUpdate, Name: "编辑组织", Group: "系统管理 · 组织", Description: "修改组织名称和启停状态", Requires: []string{OrganizationRead}},
	{Key: UserRead, Name: "查看用户", Group: "系统管理 · 用户", Description: "查看用户及其组织成员关系"},
	{Key: UserCreate, Name: "新建用户", Group: "系统管理 · 用户", Description: "新建用户并配置初始成员关系", Requires: []string{UserRead}},
	{Key: UserUpdate, Name: "编辑用户", Group: "系统管理 · 用户", Description: "修改用户资料、状态和成员关系", Requires: []string{UserRead}},
	{Key: UserTerminate, Name: "办理离职", Group: "系统管理 · 用户", Description: "停用员工账号和全部组织权限，保留身份绑定与历史业务记录", Requires: []string{UserRead}},
	{Key: UserAuthorizeWeCom, Name: "授权企业微信成员", Group: "系统管理 · 用户", Description: "读取企业微信成员并创建或绑定系统用户", Requires: []string{UserRead}},
	{Key: UserAuthorizeDingTalk, Name: "授权钉钉成员", Group: "系统管理 · 用户", Description: "读取钉钉成员并创建或绑定系统用户", Requires: []string{UserRead}},
	{Key: UserResetPassword, Name: "重置用户密码", Group: "系统管理 · 用户", Description: "为系统用户重置登录密码", Requires: []string{UserRead}},
	{Key: RoleRead, Name: "查看角色", Group: "系统管理 · 角色与权限", Description: "查看角色、权限和数据范围", Requires: []string{PermissionRead, OrganizationRead}},
	{Key: RoleCreate, Name: "新建角色", Group: "系统管理 · 角色与权限", Description: "新建角色并配置权限和数据范围", Requires: []string{RoleRead}},
	{Key: RoleUpdate, Name: "编辑角色", Group: "系统管理 · 角色与权限", Description: "修改角色、权限和数据范围", Requires: []string{RoleRead}},
	{Key: PermissionRead, Name: "查看权限字典", Group: "系统管理 · 角色与权限", Description: "查看系统功能权限字典"},
	{Key: AuditRead, Name: "查看审计日志", Group: "系统管理 · 审计", Description: "查看安全与业务操作审计"},
	{Key: TaskRead, Name: "查看后台任务", Group: "系统管理 · 后台任务", Description: "查看后台任务执行状态"},
	{Key: TaskRequeue, Name: "重新入队后台任务", Group: "系统管理 · 后台任务", Description: "重新入队失败或死信后台任务", Requires: []string{TaskRead}},
	{Key: EnterpriseResourceRead, Name: "查看资源备忘", Group: "业务资料 · 企业资源 · 资源备忘", Description: "查看地址、备注、图片、标签和单证主体资源"},
	{Key: EnterpriseResourceCreate, Name: "新建资源备忘", Group: "业务资料 · 企业资源 · 资源备忘", Description: "新建资源及标签组", Requires: []string{EnterpriseResourceRead}},
	{Key: EnterpriseResourceUpdate, Name: "编辑资源备忘", Group: "业务资料 · 企业资源 · 资源备忘", Description: "编辑资源、标签组和企业关联", Requires: []string{EnterpriseResourceRead}},
	{Key: EnterpriseResourceDelete, Name: "删除资源备忘", Group: "业务资料 · 企业资源 · 资源备忘", Description: "删除资源及空标签组", Requires: []string{EnterpriseResourceRead}},
	{Key: FinanceExchangeRateRead, Name: "查看汇率", Group: "费用管理 · 汇率", Description: "查看组织汇率主数据和时间标准"},
	{Key: FinanceExchangeRateCreate, Name: "新建汇率", Group: "费用管理 · 汇率", Description: "新建组织汇率", Requires: []string{FinanceExchangeRateRead}},
	{Key: FinanceExchangeRateUpdate, Name: "编辑汇率", Group: "费用管理 · 汇率", Description: "修改组织汇率和时间标准", Requires: []string{FinanceExchangeRateRead}},
	{Key: FinanceExchangeRateDisable, Name: "停用汇率", Group: "费用管理 · 汇率", Description: "停用组织汇率", Requires: []string{FinanceExchangeRateRead}},
	{Key: FinanceExchangeRateOverride, Name: "覆盖财务汇率", Group: "费用管理 · 汇率", Description: "在订单费用或资金流水中手工覆盖系统汇率", Requires: []string{FinanceExchangeRateRead}},
	{Key: FinanceFeeSettingRead, Name: "查看费用设置", Group: "费用管理 · 费用设置", Description: "查看费用设置及关联基础资料"},
	{Key: FinanceFeeSettingCreate, Name: "新建费用设置", Group: "费用管理 · 费用设置", Description: "新建费用设置及关联基础资料", Requires: []string{FinanceFeeSettingRead}},
	{Key: FinanceFeeSettingUpdate, Name: "编辑费用设置", Group: "费用管理 · 费用设置", Description: "编辑和停用费用设置及关联基础资料", Requires: []string{FinanceFeeSettingRead}},
	{Key: FinanceFeeRead, Name: "查看费用总台账", Group: "费用管理 · 费用总台账", Description: "查看当前组织全部业务线的应收应付费用"},
	{Key: FinanceFeeTag, Name: "维护费用标签", Group: "费用管理 · 费用总台账", Description: "在费用总台账批量添加或移除业务标签", Requires: []string{FinanceFeeRead}},
	{Key: FinanceBillRead, Name: "查看账单", Group: "费用管理 · 账单", Description: "查看应收应付账单及明细"},
	{Key: FinanceBillCreate, Name: "创建账单", Group: "费用管理 · 账单", Description: "按结算单位聚合已确认费用创建账单", Requires: []string{FinanceBillRead}},
	{Key: FinanceBillUpdate, Name: "编辑账单", Group: "费用管理 · 账单", Description: "编辑、撤回或作废未结清账单", Requires: []string{FinanceBillRead}},
	{Key: FinanceBillConfirm, Name: "确认账单", Group: "费用管理 · 账单", Description: "确认账单并锁定账单费用", Requires: []string{FinanceBillRead}},
	{Key: FinanceInvoiceRead, Name: "查看开票记录", Group: "费用管理 · 开票", Description: "查看销项和进项发票台账"},
	{Key: FinanceInvoiceCreate, Name: "登记发票", Group: "费用管理 · 开票", Description: "登记发票并向账单分配开票金额", Requires: []string{FinanceInvoiceRead}},
	{Key: FinanceInvoiceUpdate, Name: "处理发票", Group: "费用管理 · 开票", Description: "开具、作废或红冲发票", Requires: []string{FinanceInvoiceRead}},
	{Key: FinanceCashflowRead, Name: "查看收付", Group: "费用管理 · 收付", Description: "查看银行流水和收付款单"},
	{Key: FinanceCashflowCreate, Name: "登记收付", Group: "费用管理 · 收付", Description: "登记银行流水和收付款单", Requires: []string{FinanceCashflowRead}},
	{Key: FinanceCashflowUpdate, Name: "处理收付", Group: "费用管理 · 收付", Description: "认领、确认或冲销收付款单", Requires: []string{FinanceCashflowRead}},
	{Key: FinanceVerificationRead, Name: "查看核销", Group: "费用管理 · 核销", Description: "查看账单与收付款核销记录"},
	{Key: FinanceVerificationCreate, Name: "执行核销", Group: "费用管理 · 核销", Description: "将收付款金额分配到应收应付账单", Requires: []string{FinanceVerificationRead}},
	{Key: FinanceVerificationReverse, Name: "反核销", Group: "费用管理 · 核销", Description: "按原因撤销有效核销分配", Requires: []string{FinanceVerificationRead}},
	{Key: FinanceCommissionRead, Name: "查看提成", Group: "费用管理 · 提成", Description: "查看单票毛利和人员提成结果"},
	{Key: FinanceCommissionManage, Name: "管理提成", Group: "费用管理 · 提成", Description: "维护提成规则并计算、确认提成", Requires: []string{FinanceCommissionRead}},
	{Key: FinanceCommissionExport, Name: "导出提成", Group: "费用管理 · 提成", Description: "按当前筛选条件导出提成双口径数据", Requires: []string{FinanceCommissionRead}},
	{Key: PartnerRead, Name: "查看往来单位", Group: "业务资料 · 往来单位 · 单位档案", Description: "查看客户、供应商和国外代理档案"},
	{Key: PartnerCreate, Name: "新建往来单位", Group: "业务资料 · 往来单位 · 单位档案", Description: "新建客户、供应商或国外代理档案", Requires: []string{PartnerRead}},
	{Key: PartnerUpdate, Name: "编辑往来单位", Group: "业务资料 · 往来单位 · 单位档案", Description: "修改客户、供应商或国外代理档案", Requires: []string{PartnerRead}},
	{Key: PartnerBlacklist, Name: "管理供应商黑名单", Group: "业务资料 · 往来单位 · 单位档案", Description: "调整供应商黑名单状态", Requires: []string{PartnerRead}},
	{Key: PartnerImport, Name: "导入往来单位", Group: "业务资料 · 往来单位 · 单位档案", Description: "批量导入往来单位档案", Requires: []string{PartnerRead}},
	{Key: PartnerExport, Name: "导出往来单位", Group: "业务资料 · 往来单位 · 单位档案", Description: "批量导出往来单位档案", Requires: []string{PartnerRead}},
	{Key: PartnerAccountRead, Name: "查看收付款账户", Group: "业务资料 · 往来单位 · 收付款账户", Description: "查看往来单位收付款账户"},
	{Key: PartnerAccountCreate, Name: "新建收付款账户", Group: "业务资料 · 往来单位 · 收付款账户", Description: "新建往来单位收付款账户", Requires: []string{PartnerAccountRead}},
	{Key: PartnerAccountUpdate, Name: "编辑收付款账户", Group: "业务资料 · 往来单位 · 收付款账户", Description: "修改往来单位收付款账户", Requires: []string{PartnerAccountRead}},
	{Key: PartnerContractRead, Name: "查看合同", Group: "业务资料 · 往来单位 · 合同", Description: "查看往来单位合同"},
	{Key: PartnerContractCreate, Name: "新建合同", Group: "业务资料 · 往来单位 · 合同", Description: "新建往来单位合同", Requires: []string{PartnerContractRead}},
	{Key: PartnerContractUpdate, Name: "编辑合同", Group: "业务资料 · 往来单位 · 合同", Description: "修改往来单位合同", Requires: []string{PartnerContractRead}},
	{Key: PartnerSettlementRuleRead, Name: "查看结算规则", Group: "业务资料 · 往来单位 · 结算规则", Description: "查看往来单位结算规则"},
	{Key: PartnerSettlementRuleCreate, Name: "新建结算规则", Group: "业务资料 · 往来单位 · 结算规则", Description: "新建往来单位结算规则", Requires: []string{PartnerSettlementRuleRead}},
	{Key: PartnerSettlementRuleUpdate, Name: "编辑结算规则", Group: "业务资料 · 往来单位 · 结算规则", Description: "修改往来单位结算规则", Requires: []string{PartnerSettlementRuleRead}},
	{Key: PartnerAttachmentRead, Name: "查看单位附件", Group: "业务资料 · 往来单位 · 附件", Description: "查看往来单位附件"},
	{Key: PartnerAttachmentRegister, Name: "登记单位附件", Group: "业务资料 · 往来单位 · 附件", Description: "登记往来单位附件元数据", Requires: []string{PartnerAttachmentRead}},
	{Key: PartnerShippingPresetRead, Name: "查看单证预设", Group: "业务资料 · 往来单位 · 单证预设", Description: "查看往来单位常用单证预设"},
	{Key: PartnerShippingPresetCreate, Name: "新建单证预设", Group: "业务资料 · 往来单位 · 单证预设", Description: "新建往来单位常用单证预设", Requires: []string{PartnerShippingPresetRead}},
	{Key: PartnerShippingPresetUpdate, Name: "编辑单证预设", Group: "业务资料 · 往来单位 · 单证预设", Description: "修改往来单位常用单证预设", Requires: []string{PartnerShippingPresetRead}},
	{Key: PartnerAuditRead, Name: "查看单位操作记录", Group: "业务资料 · 往来单位 · 审计", Description: "查看往来单位操作记录"},
	{Key: PartnerAssignmentOptionRead, Name: "查看责任人选项", Group: "业务资料 · 往来单位 · 责任人", Description: "查看往来单位责任人候选项"},
	{Key: MasterDataCurrencyRead, Name: "查看币种", Group: "主数据 · 公共字典", Description: "查看币种字典"},
	{Key: MasterDataAdministrativeRegionRead, Name: "查看行政区划", Group: "主数据 · 公共字典", Description: "查看行政区划字典"},
	{Key: MasterDataOptionRead, Name: "查看订单选项", Group: "主数据 · 公共字典", Description: "查看订单表单的聚合选项"},
	{Key: MasterDataItemRead, Name: "查看目录项", Group: "主数据 · 基础目录", Description: "查看主数据基础目录项"},
	{Key: MasterDataItemCreate, Name: "新建目录项", Group: "主数据 · 基础目录", Description: "新建主数据基础目录项", Requires: []string{MasterDataItemRead}},
	{Key: MasterDataItemUpdate, Name: "编辑目录项", Group: "主数据 · 基础目录", Description: "修改主数据基础目录项", Requires: []string{MasterDataItemRead}},
	{Key: MasterDataItemImport, Name: "导入目录项", Group: "主数据 · 基础目录", Description: "批量导入主数据基础目录项", Requires: []string{MasterDataItemRead}},
	{Key: MasterDataPortRead, Name: "查看港口", Group: "主数据 · 港口", Description: "查看港口资料"},
	{Key: MasterDataPortCreate, Name: "新建港口", Group: "主数据 · 港口", Description: "新建港口资料", Requires: []string{MasterDataPortRead}},
	{Key: MasterDataPortUpdate, Name: "编辑港口", Group: "主数据 · 港口", Description: "修改港口资料", Requires: []string{MasterDataPortRead}},
	{Key: MasterDataAirportRead, Name: "查看机场", Group: "主数据 · 机场", Description: "查看机场资料"},
	{Key: MasterDataAirportCreate, Name: "新建机场", Group: "主数据 · 机场", Description: "新建机场资料", Requires: []string{MasterDataAirportRead}},
	{Key: MasterDataAirportUpdate, Name: "编辑机场", Group: "主数据 · 机场", Description: "修改机场资料", Requires: []string{MasterDataAirportRead}},
	{Key: MasterDataAirlineRead, Name: "查看航空公司", Group: "主数据 · 航空公司", Description: "查看航空公司资料"},
	{Key: MasterDataAirlineCreate, Name: "新建航空公司", Group: "主数据 · 航空公司", Description: "新建航空公司资料", Requires: []string{MasterDataAirlineRead}},
	{Key: MasterDataAirlineUpdate, Name: "编辑航空公司", Group: "主数据 · 航空公司", Description: "修改航空公司资料", Requires: []string{MasterDataAirlineRead}},
	{Key: MasterDataShippingLineRead, Name: "查看船公司", Group: "主数据 · 船公司", Description: "查看船公司资料"},
	{Key: MasterDataShippingLineCreate, Name: "新建船公司", Group: "主数据 · 船公司", Description: "新建船公司资料", Requires: []string{MasterDataShippingLineRead}},
	{Key: MasterDataShippingLineUpdate, Name: "编辑船公司", Group: "主数据 · 船公司", Description: "修改船公司资料", Requires: []string{MasterDataShippingLineRead}},
	{Key: MasterDataNumberRuleRead, Name: "查看编号规则", Group: "主数据 · 编号规则", Description: "查看业务编号规则"},
	{Key: MasterDataNumberRuleCreate, Name: "新建编号规则", Group: "主数据 · 编号规则", Description: "新建业务编号规则", Requires: []string{MasterDataNumberRuleRead}},
	{Key: MasterDataNumberRuleUpdate, Name: "编辑编号规则", Group: "主数据 · 编号规则", Description: "修改业务编号规则", Requires: []string{MasterDataNumberRuleRead}},
}, orderManifest()...)

type orderPermissionDefinition struct {
	operation     OrderOperation
	name          string
	resource      string
	description   string
	businessTypes []OrderBusinessType
}

var orderPermissionDefinitions = []orderPermissionDefinition{
	{operation: OrderRead, name: "查看订单", resource: "订单", description: "查看当前数据范围内的订单"},
	{operation: OrderCreate, name: "新建订单", resource: "订单", description: "新建订单并检查业务编号"},
	{operation: OrderUpdate, name: "编辑订单", resource: "订单", description: "修改订单基础与业务资料"},
	{operation: OrderTransition, name: "流转订单状态", resource: "订单", description: "执行订单状态流转"},
	{operation: OrderMilestoneRead, name: "查看里程碑", resource: "里程碑", description: "查看订单里程碑"}, {operation: OrderMilestoneSet, name: "设置里程碑", resource: "里程碑", description: "完成、跳过或重置订单里程碑"},
	{operation: OrderAttachmentRead, name: "查看附件", resource: "附件", description: "查看订单附件"}, {operation: OrderAttachmentRegister, name: "管理附件", resource: "附件", description: "登记订单附件元数据与解除引用"},
	{operation: OrderPersonnelRead, name: "查看协作人员", resource: "协作人员", description: "查看订单协作人员"}, {operation: OrderPersonnelAssign, name: "指派协作人员", resource: "协作人员", description: "指派订单协作人员"}, {operation: OrderPersonnelRemove, name: "移除协作人员", resource: "协作人员", description: "移除订单协作人员"},
	{operation: OrderContainerRead, name: "查看集装箱", resource: "集装箱", description: "查看订单集装箱"}, {operation: OrderContainerCreate, name: "新增集装箱", resource: "集装箱", description: "新增订单集装箱"}, {operation: OrderContainerUpdate, name: "编辑集装箱", resource: "集装箱", description: "修改订单集装箱"}, {operation: OrderContainerDelete, name: "删除集装箱", resource: "集装箱", description: "删除订单集装箱"},
	{operation: OrderCargoItemRead, name: "查看货物明细", resource: "货物", description: "查看订单货物明细"}, {operation: OrderCargoItemCreate, name: "新增货物明细", resource: "货物", description: "新增订单货物明细"}, {operation: OrderCargoItemUpdate, name: "编辑货物明细", resource: "货物", description: "修改订单货物明细"}, {operation: OrderCargoItemDelete, name: "删除货物明细", resource: "货物", description: "删除订单货物明细"},
	{operation: OrderAbnormalCaseRead, name: "查看异常事件", resource: "异常", description: "查看订单异常事件"}, {operation: OrderAbnormalCaseCreate, name: "登记异常事件", resource: "异常", description: "登记订单异常事件"}, {operation: OrderAbnormalCaseResolve, name: "处理异常事件", resource: "异常", description: "解决或重新打开订单异常事件"}, {operation: OrderAbnormalCaseDelete, name: "删除异常事件", resource: "异常", description: "删除订单异常事件"},
	{operation: OrderReleasePodRead, name: "查看放货凭证", resource: "放货", description: "查看订单放货凭证"}, {operation: OrderReleasePodCreate, name: "新增放货凭证", resource: "放货", description: "新增订单放货凭证"}, {operation: OrderReleasePodUpdate, name: "编辑放货凭证", resource: "放货", description: "修改订单放货凭证"}, {operation: OrderReleasePodTransition, name: "流转放货状态", resource: "放货", description: "执行放货状态流转"}, {operation: OrderReleasePodDelete, name: "删除放货凭证", resource: "放货", description: "删除订单放货凭证"},
	{operation: OrderFeeRead, name: "查看费用", resource: "费用", description: "查看订单应收应付费用"}, {operation: OrderFeeCreate, name: "录入费用", resource: "费用", description: "录入订单应收应付费用"}, {operation: OrderFeeUpdate, name: "编辑费用", resource: "费用", description: "修改订单应收应付费用"}, {operation: OrderFeeDelete, name: "删除费用", resource: "费用", description: "删除订单应收应付费用"},
	{operation: OrderSplit, name: "拆票", resource: "订单", description: "执行海运出口拆票", businessTypes: []OrderBusinessType{OrderBusinessSE}},
	{operation: OrderReassign, name: "整体改配", resource: "订单", description: "执行海运出口整体改配", businessTypes: []OrderBusinessType{OrderBusinessSE}},
	{operation: OrderLock, name: "锁定/直接解锁订单", resource: "订单", description: "执行海运出口订单锁定与直接解锁", businessTypes: []OrderBusinessType{OrderBusinessSE}},
	{operation: OrderAmend, name: "改单", resource: "订单", description: "执行海运出口单证改单", businessTypes: []OrderBusinessType{OrderBusinessSE}},
	{operation: OrderVoid, name: "作废提单", resource: "订单", description: "执行海运出口单证作废", businessTypes: []OrderBusinessType{OrderBusinessSE}},
	{operation: OrderSwitch, name: "Switch B/L", resource: "订单", description: "执行海运出口换单（Switch B/L）", businessTypes: []OrderBusinessType{OrderBusinessSE}},
}

func OrderPermission(businessType OrderBusinessType, operation OrderOperation) string {
	if !businessType.Valid() || !operation.Valid() {
		return ""
	}
	for _, def := range orderPermissionDefinitions {
		if def.operation == operation && len(def.businessTypes) > 0 {
			supported := false
			for _, bt := range def.businessTypes {
				if bt == businessType {
					supported = true
					break
				}
			}
			if !supported {
				return ""
			}
		}
	}
	return fmt.Sprintf("business.order.%s.%s", businessType.code(), operation)
}

func (v OrderBusinessType) Valid() bool {
	return v == OrderBusinessSE || v == OrderBusinessSI || v == OrderBusinessAE || v == OrderBusinessAI
}

func (v OrderBusinessType) code() string {
	return map[OrderBusinessType]string{OrderBusinessSE: "se", OrderBusinessSI: "si", OrderBusinessAE: "ae", OrderBusinessAI: "ai"}[v]
}

func (v OrderBusinessType) name() string {
	return map[OrderBusinessType]string{OrderBusinessSE: "海运出口（SE）", OrderBusinessSI: "海运进口（SI）", OrderBusinessAE: "空运出口（AE）", OrderBusinessAI: "空运进口（AI）"}[v]
}

func (v OrderOperation) Valid() bool {
	for _, definition := range orderPermissionDefinitions {
		if definition.operation == v {
			return true
		}
	}
	return false
}

func orderManifest() []Permission {
	types := []OrderBusinessType{OrderBusinessSE, OrderBusinessSI, OrderBusinessAE, OrderBusinessAI}
	items := make([]Permission, 0, len(types)*len(orderPermissionDefinitions))
	for _, businessType := range types {
		for _, definition := range orderPermissionDefinitions {
			if len(definition.businessTypes) > 0 {
				supported := false
				for _, bt := range definition.businessTypes {
					if bt == businessType {
						supported = true
						break
					}
				}
				if !supported {
					continue
				}
			}
			items = append(items, Permission{Key: OrderPermission(businessType, definition.operation), Name: fmt.Sprintf("%s %s", businessType.name(), definition.name), Group: fmt.Sprintf("订单管理 · %s · %s", businessType.name(), definition.resource), Description: definition.description, Requires: orderPermissionRequires(businessType, definition.operation)})
		}
	}
	return items
}

// orderPermissionRequires 推导订单权限依赖：操作权限依赖同资源读权限，子资源
// 权限（里程碑、集装箱、费用等）还依赖该业务线的订单读权限。
func orderPermissionRequires(businessType OrderBusinessType, operation OrderOperation) []string {
	orderRead := OrderPermission(businessType, OrderRead)
	if operation == OrderSplit || operation == OrderReassign || operation == OrderLock || operation == OrderAmend || operation == OrderVoid || operation == OrderSwitch {
		return []string{orderRead, OrderPermission(businessType, OrderUpdate)}
	}
	resource := ""
	action := string(operation)
	if prefix, suffix, found := strings.Cut(string(operation), "."); found {
		resource, action = prefix, suffix
	}
	switch {
	case resource == "" && action == "read":
		return nil
	case resource == "", action == "read":
		return []string{orderRead}
	default:
		return []string{OrderPermission(businessType, OrderOperation(resource+".read")), orderRead}
	}
}

func Manifest() []Permission { return append([]Permission(nil), manifest...) }

// ResolveDependencies 返回 granted 连同其全部传递依赖的闭包：新增依赖按深度
// 优先追加在原键之后并去重，不在 Manifest 中的键原样保留，维持调用方语义。
func ResolveDependencies(granted []string) []string {
	requiresOf := make(map[string][]string, len(manifest))
	for _, definition := range manifest {
		requiresOf[definition.Key] = definition.Requires
	}
	seen := make(map[string]struct{}, len(granted))
	result := make([]string, 0, len(granted))
	var visit func(key string)
	visit = func(key string) {
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, key)
		for _, required := range requiresOf[key] {
			visit(required)
		}
	}
	for _, key := range granted {
		visit(key)
	}
	return result
}
