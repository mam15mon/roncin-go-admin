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
)

var manifest = []Permission{
	{Key: PlatformAccess, Name: "访问管理后台", Group: "系统管理", Description: "允许登录并访问管理后台"},
	{Key: OrganizationManage, Name: "管理组织", Group: "系统管理", Description: "创建和维护公司、分支与部门"},
	{Key: UserManage, Name: "管理用户", Group: "系统管理", Description: "创建用户并维护组织成员关系"},
	{Key: RoleManage, Name: "管理角色", Group: "系统管理", Description: "维护角色、权限和数据范围"},
	{Key: AuditRead, Name: "查看审计日志", Group: "系统管理", Description: "查看安全与业务操作审计"},
}

func Manifest() []Permission { return append([]Permission(nil), manifest...) }
