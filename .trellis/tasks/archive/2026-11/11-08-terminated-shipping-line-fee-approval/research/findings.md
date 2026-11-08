# 只读定位

- server/internal/data/order_convert.go:72：详情shippingLineName来自预载ShippingLine.NameZh。
- web/src/pages/orders/detail.tsx:578、633：存在名称种子，但仅传给searchShippingLines。
- web/src/pages/orders/templates/components/sea/SeaBasicInfoSection.tsx:298-342：当前选项仅看活动MBL；readonly时request缺省，种子不被消费。
- 正式GET详情：SE2026110600004返回shippingLineName=中远海运集运，seaMasterBill=null；GET sea-documents保留主单返回完整中文英文名称。
- server/internal/data/order_fee_correction.go:175-179：CanApprove=真实grant且PENDING。
- server/internal/data/order_lock_qualification.go:93-119：合格审核用户必须非bootstrap，角色/成员/公司/用户启用且同grant具备品类lock及前置权限。
- 数据库两条100元验收申请均WITHDRAWN，申请人为系统初始化管理员，订单SE2026110600001；未改数据。
- 本期读写边界：只读研究尚未改业务代码、角色、账号或申请；既有布局任务保留。
