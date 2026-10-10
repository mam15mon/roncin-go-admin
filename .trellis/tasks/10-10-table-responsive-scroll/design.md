# 技术设计

## 最小变更边界

行为缺口是宽表内容不可访问或滚动/固定规则分散。修复位置是Table布局props与真实滚动容器，不在数据层删列、不通过全局脚本遍历DOM硬补overflow。

## 共享能力

在components/ui内提供最薄表格展示适配能力，经公开入口导出，集中处理有效可见列的宽度、scroll.x默认值、既有scroll.y保留，以及窄容器固定列取舍。直接兼容Table/ProTable/EditableProTable，不托管request、actionRef、列设置或编辑状态。

保持已有业务列宽与财务拖拽宽度。已明确总宽度经审查保留；不正确的固定常量改为可见列驱动或max-content。列组/选择/展开列参与布局；宽屏少列弹性适配。实际容器空间作为固定区依据，抽屉窄宽不能只看window宽度。

列设置先输出有效列，再由共享布局适配，保留用户排序、手动隐藏和固定区语义。用户列表的业务隐藏组织继续由buildUserColumns控制。

## 滚动与样式

复用antd官方scroll/sticky，页面长表使用正确布局顶部偏移与吸底条；抽屉/弹窗绑定实际滚动容器。global.less只保留外层边界隔离与统一视觉，修正错层overflow导致的裁切，不用全局.ant-table无限叠加!important或自建DOM同步滚动器。

公共模板先接入：MasterDataTemplate、FinanceLedgerTemplate、OrderListTemplate、SettingTableTemplate、SubEntityDrawerTemplate。再按清单处理pages/features的普通、可编辑、弹层和嵌套表。前端已有模块边界保持。

## 风险与验证

主要风险：fixed+ellipsis改变tableLayout，财务拖拽与手动隐藏后的总宽度，编辑表光标/选择，sticky表头与页面壳层级，弹层滚动容器、百分比/未指定宽度列以及分组列。每组实施后以真实浏览器验证可见区域、scrollLeft首尾、固定列命中与滚动条可达性；jsdom只验证配置与行为，不能证明几何。

研究数据覆盖主页面17组合，尚未完成全站每个弹层实际验收。实施期间补齐典型弹层/嵌套/编辑样本及全量接入清单；没有证据不宣称全站均已验收。

变更纯前端，可按分组提交回退；无数据迁移和偏好清理。
