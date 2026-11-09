# 技术设计

## 1. 退役边界与分工
这是已存在纵向功能的完整删除，不是UI隐藏。以research/contracts.md、frontend.md和storage.md的精确清单实施；不按所有Tag/tags字样盲删。三个交付包：后端契约/运行代码，Ent与正式迁移，前端入口/当前公司绑定。主会话负责规范、生成协调、独立审查与提交。

## 2. 契约与运行代码
- 删除OrderTagService整个源和独占生成输出、OrderFee3个RPC、Settlement8个RPC、EnterpriseTagGroup4个RPC及独占消息。
- 在仍存在的Proto消息中移除tags/tag_ids/oneof tag，reserved原名称和编号。EnterpriseResourceType TAG=4保留reserved，其他5..10值不动；BusinessTagSummary等无消费者后删除。
- 删除biz.BusinessTagUsecase/Repo、data.businessTagRepo、专属service和注册；同步NewOrderUsecase、OrderFeeService、SettlementService所有正式和测试构造、provider/Wire，不留空实现或旧接口回退。
- 列表和详情不再attach tags或按tag IDs查过滤；拆票只删除标签复制/转移逻辑，原订单复制、费用守恒、锁与事务不改。
- Manifest只退役独立fee.tag，通用权限key保持，企业资源权限说明删除新标签功能承诺。正式migrate原有SyncPermissionManifest负责删除旧权限和role_permissions关联。

## 3. 持久化与正式迁移
Ent真相源删除EnterpriseTagGroup/EnterpriseTag/三种业务关联实体和各owner边，保留通用资源及非TAG明细/关系；枚举去TAG但不误以为现库有PostgreSQL独立enum类型。

新迁移在统一执行事务内：
1. 精确定位resource_type=TAG。预检TAG未错误关联地址/备注/图片/主体/文本/地址人员/地址类型；预检冗余TAG类型关联未指向非TAG资源。异常时报错，不能帮用户静默清理别的资源。
2. DROP三张业务标签关联表，默认RESTRICT；未知依赖使迁移失败，不用CASCADE掩盖。
3. 按TAG资源ID删除enterprise_resource_partners中的关系，不按模糊字符串删整表。
4. DROP enterprise_tags，再DROP enterprise_tag_groups，默认RESTRICT。
5. DELETE enterprise_resources中TAG行。保留核心单据、其他资源和全部audit_logs。
6. 验证末态与Ent生成结构一致；不改历史SQL、迁移校验和或历史审计内容。若需类型CHECK，按当前Ent与正式结构确切声明，不猜旧约束名。

此设计含标签数据永久删除，需最终批准；当前开发库标签数量0不等于其他库0。只用隔离测试库跑迁移回归，未授权前不应用真实库，不新建在线归档/兼容表。需要保留外部数据时可在迁移前离线备份，不增加运行期分支。

## 4. 前端与工作区
- 删除BusinessTagModal/List及相关页面状态、候选请求、筛选项、列、可选列偏好、批量动作和CSV标签字段。公共OrderListTemplate/SearchFilter/Toolbar/types同步收窄，只保留真实消费者。
- 企业资源移除标签页签、TagGroupModal、tag字段/props和类型分支。其他类型key/type保持；查询无效tab继续既有默认行为，不新增专用旧tags迁移路由。现有key按类型隔离的列表/编辑会话和form.submit/onFinish修复不变。
- 账单整个公司/标签topBar移除。useInitialState读取currentUser.currentOrganization.id作为列表唯一org来源，不从queryState恢复organizationId；组织未就绪时显式不发查询，不默认查询所有公司。当前系统workspace原本无财务经营入口，不因删选择框扩大权限。
- 查询恢复模型去tagIds/tagLabels/账单organizationId；不新增双读和旧缓存纠正逻辑。workspace按用户+当前组织已有隔离继续复用。
- BillCreationWorkbench内部公司选择保持原范围，不趁标签退役重构业务写入流。保留财务采纳快照、金额汇总、跨页勾选和筛选清空规则。

## 5. 生成物与规范
按源变更运行Ent、Proto/Wire、OpenAPI客户端、数字枚举及权限keys生成；生成器不清理的已无源文件专属产物精确删除，不手补内容，不整目录盲删。正式生成后再跑一轮验证幂等。
现行能力导航/领域图谱/类型契约/权限说明同步更新，不回写历史任务日志/迁移。历史审计静态描述可留，仅为历史可读，不提供运行期标签能力。

## 6. 风险与回退
主要风险为误删非TAG数据、构造注入残留、费用/拆票相邻路径变化、当前组织未加载时列表越界和生成物残留。用隔离库迁移前后快照、真实路由/反射、相邻业务及UI回归验证。数据库破坏性变更无法仅git revert恢复标签数据；仅标签专用信息有恢复需求时先离线备份。不存在重建数据库或批量改业务单据方案。
