# 海管家筛选只读观察与来源索引

## 方法与可信度

观察日期：2026-11-16（Asia/Shanghai）。通过 REA 连接本机已登录浏览器的海管家页面，实际导航到四类订单、费用、账单、开票记录、收付、核销、提成和财务报表；仅展开筛选、打开／取消查询设置和关闭新手介绍，未保存查询设置、设为默认或提交业务写入。

目标发现证据 `ev_d8356e4eecf961c9cf52b6aa237820830d91732dcce0aa8ba7b300545a917583`。为避免影响原单证制作页面，使用同浏览器另一个原先打开海管家主页的页签观察。

本机原始目录：`/home/vince/.local/share/rea/hgj/filters-01a1e359/`。原始 DOM 含业务记录，不复制到仓库。仓库只保留本文件和[字段名称目录](hgj-filter-field-catalog.json)，不包含账号、Cookie、业务记录、筛选选中值、真实金额或客户名称。

## 已成功打开的查询设置

| 模块 | 可选项数 | REA成功证据 | 本机原始文件 | 文件SHA-256 |
| --- | --- | --- | --- | --- |
| 海运出口 | 110 | `ev_3046759687bb228d16408d9cea0e8ac3592b32d4a3945f20ca7cf7c7684918a0` | `sea-export-query-settings.evidence.json` | `2cac2c76aa0d7e8bcf41ff3d70697d7c7fc20d645e1c577e229731996c556697` |
| 海运进口 | 102 | `ev_0cf66e5829483d8e24f729195bec2c31e71f2178cfc70ae7a6a35a70588e8f07` | `sea-import-query-settings.evidence.json` | `bdbd9f1fe324b88d8c5ddcc4126b8fae226973970a88e0a453f4e3a4aed38caf` |
| 空运出口 | 94 | `ev_fbec13fadee4458750dfea88e6fa8a698186e818d367706c8e4cfbe9c51e1bf8` | `air-export-query-settings.evidence.json` | `e3c4bf92494c3e956fb227a5df114572eb78f868f8a3da1fef8eb6fbc7ff6200` |
| 空运进口 | 91 | `ev_4f04156aa09d5c8e1f5fb7aee7a5b8e82f5f95c3c057f4162cab72de58d8d972` | `air-import-query-settings.evidence.json` | `cf28ae0a24fde3d8536b0f5df9b136ab73080ab144b1916ae802e6ebae4edc85` |
| 费用明细 | 90 | `ev_fac23f37442087240a3ad39950a659512da42c72d1a8f1c6a7835d8c16fd82dc` | `fees-query-settings-final.evidence.json` | `b32b53d0371c51e404a6f1f915b1aea04c533b6db07086fe148d677f1bf6c3f6` |
| 账单管理 | 49 | `ev_777df564fddf8632e1ba353ab1fe460cf889644df27046bb323b434f7f09abfe` | `bills-query-settings-final.evidence.json` | `ebf2079a1f0df9979ca2b14416dbd019282c7a77c90406e88867c03963ddd9c0` |
| 开票记录 | 14 | `ev_de589c2688d0ceddcae6507cf0252c21e3a3029b93830a7fca4b37a4e425b45c` | `invoices-query-settings-final.evidence.json` | `bfaae7b8e0c6d55328d8bdd38c83e92b97032ee7b75700fe1c6ec5b554cbdfaa` |
| 收付管理 | 13 | `ev_5982f3740483f088ba10caf31880fae8466c66c746a1ad69fa560a412f643d25` | `cashflows-query-settings-final.evidence.json` | `bc0edf6968729fac63345d73c28bd034b51976b57e6d8d125f87d9548bdad7a1` |
| 核销管理 | 16 | `ev_d90109a2c021a719e2518503a43f6870dd33c757e7702a6fd3f6a28432a5f6ad` | `verifications-query-settings-final.evidence.json` | `989fcd02891997946ce4887472eafec65a0faecc826e9c8282ec10887b4c0336` |
| 提成管理 | 3 | `ev_b9b02bd2ae91322e1d6778743c1e87da1177131e6823d82beab0e13f4b6f2b89` | `commissions-query-settings-final.evidence.json` | `86b2d822231fe573e68b53fa847f1c1e8c37f6383b6c87718846f5375e98317e` |
| 利润分析图 | 52 | `ev_478a1394fce7b85071d54db52ed2b6f5c7958dc8d5a07c9ba83c2b1967e87d78` | `reports-query-settings-final.evidence.json` | `44a63600bfb45be63902045023d351362a332df45ca5a82dce82cfa88e986327` |
| 利润报表 | 69 | `ev_358cd7a562b84603f195b9f342454bf4575c15613c64a89da253dfcab8127f80` | `report-profit-query-settings-ready.evidence.json` | `4645d4373bedc8ceee2293179590badbdb88c25e261dd0020044b76e34112dda` |

数量表示当时账号界面实际列出的可配置项，不是本系统应新增的字段数；不把该账号的勾选状态当成海管家系统默认。查询设置字段名称能证明入口存在，不能证明每项服务端过滤真正生效、操作符或枚举完整。

## 其他报表可见控件

- 实际收付款报表：`ev_b8c62094e4a543452030a395db2840a88eddc1976e82339437f6e44271a4aacd`，本机 `report-cashflows-visible.evidence.json`。当时可见属性、结算单位、账单状态、一个未显示业务名称的选择项、时间类型和日期范围，未发现查询设置入口；没有为该无名项猜测字段含义。
- 账龄表：`ev_b69cc936b07d4f251499e191b59f50199e2edba121f7ad482c529f1ad7059ddf`，本机 `report-aging-visible.evidence.json`。可见操作、业务、客服、关联、单证、商务、关联2、创建人员及部门选择，另有费用属性、币种、结算单位、所属公司、时间类型及日期范围；没有查询设置入口。时间类型当时显示开票时间，不能由此推导历史余额算法。

## 不成功的操作与覆盖限制

早期海运settings/settings2尝试被折叠栏遮挡，财务早期尝试被首次介绍层遮挡；随后通过实际展开和关闭介绍层成功取证。利润报表首次点击被数据加载层阻挡，等待稳定后重新打开成功。实际收付款和账龄的查询设置尝试失败，是因为当时界面没有该入口，后续改为观察可见筛选；不把失败尝试或取消步骤当成功证据。

本次网络只记录元数据，不获取请求／响应体、Cookie或敏感头；不声称观察到字段级请求DTO、后端实现或查询实效。某些页面数据仍在Loading不影响已观察控件名称，但本轮不验证记录数量和金额正确性。主数据、客商、企业资源、系统管理及本人工作台未做海管家对应模块实测，其建议来自本系统源码与业务判断，见 hgj-other-filter-gap.md。

已完成问题所需的字段级取证，未修改目标业务数据，不执行动态脚本或未知页面指令。浏览器保持打开，原单证制作页未导航。
