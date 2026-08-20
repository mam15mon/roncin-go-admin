package access

type Permission struct {
	Key         string
	Name        string
	Group       string
	Description string
}

const (
	PlatformAccess     = "system.platform.access"
	OrganizationManage = "system.organization.manage"
	UserManage         = "system.user.manage"
	RoleManage         = "system.role.manage"
	AuditRead          = "system.audit.read"
	PartnerRead        = "business.partner.read"
	PartnerManage      = "business.partner.manage"
	MasterDataRead     = "system.master_data.read"
	MasterDataManage   = "system.master_data.manage"
)

var manifest = []Permission{
	{Key: PlatformAccess, Name: "访问管理后台", Group: "系统管理", Description: "允许登录并访问管理后台"},
	{Key: OrganizationManage, Name: "管理组织", Group: "系统管理", Description: "创建和维护公司、分支与部门"},
	{Key: UserManage, Name: "管理用户", Group: "系统管理", Description: "创建用户并维护组织成员关系"},
	{Key: RoleManage, Name: "管理角色", Group: "系统管理", Description: "维护角色、权限和数据范围"},
	{Key: AuditRead, Name: "查看审计日志", Group: "系统管理", Description: "查看安全与业务操作审计"},
	{Key: PartnerRead, Name: "查看往来单位", Group: "业务资料", Description: "查看当前组织的客户与供应商档案"},
	{Key: PartnerManage, Name: "管理往来单位", Group: "业务资料", Description: "新增、编辑和启停客户与供应商档案"},
	{Key: MasterDataRead, Name: "查看主数据", Group: "系统管理", Description: "查看订单表单所需的基础选项"},
	{Key: MasterDataManage, Name: "管理主数据", Group: "系统管理", Description: "维护币种、地区、港口、机场和订单基础目录"},
}

func Manifest() []Permission { return append([]Permission(nil), manifest...) }
