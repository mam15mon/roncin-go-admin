# 总部依赖初查

| 位置 | 现有用途 | 规划影响 |
| --- | --- | --- |
| server/internal/biz/organization.go:26 | 总部身份与公共资料权限双校验 | 需明确系统管理授权来源，不可直接删除身份校验造成提权 |
| web/src/access.ts:157 | 当前总部视角投影 | 管理入口和按钮需与后端同步 |
| server/internal/biz/auth.go:64 | 工作台候选区分总部与公司 | 保持成员资格及角色绑定规则 |
| server/internal/biz/auth.go:570 | 钉钉注册总部成员收口 | 用户注册不应产生隐式公司业务访问权 |
| server/internal/biz/auth.go:991 | 审批通知兜底到总部 | 保留有负责人处理未指定公司申请的明确路径 |
| server/internal/data/finance_custom_setting.go:23 | 财务设置解析总部归属 | 公司独立配置需正式迁移及缺省初始化决策 |
| server/internal/data/order_fee.go | 已计费编辑策略消费 | 策略修改需同步真实费用写入校验 |
| server/internal/data/ent/schema/port.go:13、airport.go:13 | 基线与本地并存 | 本地行去重和所有业务引用需制定迁移映射 |

本文件为只读初查结果，不表示已完成全仓审计；后续方案需继续覆盖服务端契约、前端菜单、种子与正式迁移。
