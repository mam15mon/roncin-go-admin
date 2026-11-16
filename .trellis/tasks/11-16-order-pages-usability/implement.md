# 执行计划

用户已在整体规划后明确批准“写prd之后，开始实施”；规划已收敛，无未决用户产品问题。本任务包含主会话直接整合工作，作为单一集成交付任务，代理按文件划分职责。

1. [ ] 写PRD/design/执行计划及上下文，验证并启动任务。
2. [ ] A：空运模板/单证字段布局与就近插槽消费；相关模板与字段测试。
3. [ ] B：航空详情扩展、嵌入重量、原地运输编辑与独立动作保护；相关测试。
4. [ ] C：四类列表列/映射/默认配置及必要航空重量只读投影、生成物；相关测试。
5. [ ] 主会话：组合类型、页头单证、进口服务前移、新建/详情连续性、错误定位和整合。
6. [ ] 定向测试后按组提交，代理只提交自己负责的文件（默认由主会话统一提交，避免混入他人改动）。
7. [ ] trellis-check复核并修正；浏览器检查四类新建/详情与主要新交互及窄屏。
8. [ ] 最终若含后端合同则check:fast，否则check:web；更新规范、验收记录、提交、归档及会话记录。

## 所有权

- A：templates/air-template.tsx、templates/components/air/*（新增运输编辑窗口除外），相关模板测试。不得改B组件、detail.tsx或共享类型。
- B：order-kinds/air-export/AirExportDetailFeatures.tsx、AirWeightPanel.tsx及新建航空就近编辑/货物工作区模块及测试。不得改air-master.tsx现有编辑链（如需提取共用先协调），不得改A模板。
- C：list.tsx/list-query.ts及列表专属文件、components/ui/order-list-template/，必要只读Proto/biz/data/service及生成物。不得改订单详情模板及公共页头。
- 主会话：detail.tsx、order-kinds/types.ts、templates/types.ts、components/detail/OrderDetailHeader.tsx、进口服务位置、新建/错误定位整合、规范任务文档。

## 验证

实现后补验证（禁止TDD）。先pnpm --dir web exec vitest run <受影响文件>及修改文件Biome、git diff --check；接口/类型变更执行pnpm --dir web tsc，服务端投影执行相关Go测试。最长单文件>20秒需拆分。最终门禁只运行一次，失败修复按影响复验。

风险文件：detail.tsx（保存基线）、航空预览/提交（身份与版本）、AirWaybillFields（多个入口共用）、列表投影（零值及批量查询）。不降低规则、不跳过失败测试、不把截图当写入正确性的证明。
