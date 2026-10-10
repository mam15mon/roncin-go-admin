# 现有表格合理性审查

## 方法与覆盖

产品TSX AST扫描排除测试：原生Table48、ProTable32、EditableProTable2，共82调用点；公共模板消费者24个（财务6、订单1、主数据7、设置5、子实体5），与模板内部表格分别列出。

原生调用65处无显式scroll，1处只配scroll.y，1处直接传可选scroll，15处显式横向配置。无scroll不直接等于缺陷：需看实际列宽与容器。共享模板接入可覆盖消费者，不逐页重复改模板。

代表真实页面17组视口实测，均验证window.innerWidth与目标一致。962/1440覆盖用户、角色、主数据、费用设置、客商、订单、账单；390另覆盖用户、主数据、账单。数据仅记录表头、容器尺寸与滚动能力，无人员行或凭据。原始数据见两个browser-baseline文件。

## 已确认缺陷

用户列表（users.tsx:177）没有scroll，列定义合计约1748px；962视口实际内容区815px，1440视口1133px。所有10个表头DOM都在，包括选择和操作；联系方式、同步时间、角色等超出视区。ant-table-content overflow-x:visible且不能scrollLeft，外层ant-table-container overflow:hidden裁切超出部分。是不可访问的横向溢出，未发现用户列responsive自动删列。390视口内容区299px，右固定操作230px占大部分空间。

global.less:1479/1485的宽表隔离规则需要区分外层contain与内层scroll；global.less:1392仅在<=768将overflow加到.ant-table，无法弥补缺少Table.scroll.x，也不是全站一致的滚动入口。

## 合理做法与仍需统一处

| 类别 | 现状判断 | 统一方向 |
| --- | --- | --- |
| MasterDataTemplate | x=max-content；390/962横向滚动，1440内容适配无需滚动，方向合理 | 保留内容决定宽度、适配宽容器 |
| FinanceLedgerTemplate | x=scrollX默认max-content；账单390/962/1440可滚动，列宽拖拽和列设置有效 | 保留财务专属列宽，统一固定列和滚动可达性 |
| OrderListTemplate | x=2600；实测内容约4622px，能滚动 | 实际宽度与固定常量不同，滚动宽度应跟可见列和列宽一致 |
| 客商列表 | x=1480、左右固定列；962/1440可滚动，方向合理 | 保留关键列，检查窄容器左右固定区占用 |
| 设置/抽屉/小表 | 部分scroll靠调用方传入，子实体模板无默认x；角色5列在962/1440能完整装下 | 小表自然适配，窄容器溢出时自动滚动，避免机械加巨大宽度 |

现有滚动表均未配置sticky。长列表的原生横向条在表格底部，不能声称已验证滚动条随屏可达。抽屉/弹窗/嵌套表仍需按清单实施时专项浏览器验收。

## 官方依据与本地API

本地安装antd6.6.0、ProComponents3.1.14-2。antd CLI Table API确认scroll、sticky、tableLayout、column.width/fixed/responsive受支持；CLI提示升级但本任务不升级依赖。

官方Table文档：https://ant.design/components/table-cn/ 。固定列要与scroll.x配合；需要明确固定列宽；sticky支持滚动条和表头，需配合真实滚动容器/偏移，不通过全局DOM脚本造第二套同步滚动。公开API实施前再次核对本地类型。

## 判断

保留现有合理的宽表滚动和专属编辑/列宽能力，将布局规则抽为最薄共享UI能力。全站统一信息可达性与滚动行为；宽度、金额/日期/长文本显示仍尊重业务密度。列设置手动隐藏、权限隐藏和待分配隐藏组织属于业务规则，不应为修复溢出自动恢复。
