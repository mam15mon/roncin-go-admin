# 技术设计

## 1. 单一公司来源
页面/领域工作台从useInitialState读取currentUser.currentOrganization，复用access.canOperateBusiness及canOperateOrganization判断经营身份和目标，不硬编码kind、不借用含系统身份的admin工作台判断。认证ID是上下文，退出Form/search/queryState/本地可编辑state；不镜像到useState，不增加隐藏Form字段维持旧依赖。

沿用账单列表模式，List请求缺完整公司身份返回失败结果且不调用API，候选用enabled/事件入口guard，提交显式拒绝。请求组装先放业务筛选，最后显式赋当前公司。需要公司名或本币时使用认证投影，不再为此加载公司候选。多处简单读取不新建通用平台或无价值hook。

## 2. 列表、快照与导出
六列表和方案列表移除Select、候选请求、公司state和仅承载其的外壳。费用/收付恢复类型及提成过滤/normalize删除organizationId；保留其他业务筛选恢复。月度申请/调整子视图已有DTO支持显式当前公司，按同一来源补齐。

ProTable官方request/actionRef/onLoad及既有matchedSummary不变；当前页导出保持采纳快照，提成筛选导出显式带当前公司。选择的公司变化隔离继续依赖AppLayout/OrganizationWorkspace的用户+公司key，禁止另造序号令牌/独立数据状态。

## 3. 弹窗与来源校验
- 建账：删除公司候选/selector/变更回调及runPreview的公司覆盖参数；认证ID贯穿费用候选、预览、创建。来源组织props若保留，仅用于等值断言；同步费用台账和订单费用入口，异常来源明确拒绝。NORMAL/NETTING、预览会话/快照、账户/分组配置与幂等键不变。
- 收付：删除Form.organizationId及两组公司候选；结算单位request与创建均使用当前公司，保留方向/付款方式/汇率权限/账户/失败输入保护。
- 开票：账单候选使用当前公司，已选账单与当前公司等值后才加载抬头或提交；保留关闭重开和抬头请求隔离。创建DTO没有organizationId，不新增字段。
- 核销：从Scope移除公司，币种/方向/单位/日期/备注与分配保留；只移除bootstrap中的公司候选，保留币种。资金与账单候选固定当前公司，提交前核对归属；创建保持现有allocations契约。
- 提成生成：删Form/state的公司双源与候选请求，ProFormDependency不再依赖已删除公司字段；核销/对冲/计提候选及预览/创建采用认证ID，保留来源变化清预览与关闭清理。
- 提成方案：列表/员工搜索/新建均用认证ID；编辑、复制、名单record须等值当前公司，不把来源公司当可选目标。收窄多余组织props时一次同步所有调用者。

## 4. 公司候选契约退役
先改settlement.proto：删除ListFinanceOrganizationOptions及FinanceOrganizationOption、ListFinanceOrganizationOptionsRequest/Response。共享FinanceOrganizationPurpose只保留UNSPECIFIED=0、CASHFLOW_CREATE=4、VERIFICATION_CREATE=9；对1,2,3,5,6,7,8,10,11,12及原全名reserved。名称不重排，不做兼容端点。

删除service/biz/repo/data公司候选专用方法、类型及stub；用途权限helper收窄到4/9，不改变结算单位候选RPC与权限差异。按仓库生成流程生成Proto/路由授权/OpenAPI/客户端/枚举，唯一owner操作生成目录，重跑校验幂等。其余业务organization_id与Manifest保持；无Ent/迁移/数据操作。

## 5. 取舍与回退
不为删除选择器改造财务经济规则、权限模型或切换框架；现有服务端最终范围校验保留。代码回退只需正常Git回退，无数据迁移。若实施发现超出研究的新业务差异，先记录证据并回到规划，不静默加兼容。

## 6. 规范同步
扩展AGENTS财务工作区约束，更新finance-ledger-snapshots/相关能力导航或新增最薄专项规范；保留已归档标签退役任务的历史边界，当前规范可引用本次统一公司契约。全部文档、注释、提交中文。
