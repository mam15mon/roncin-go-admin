# 提成 CNY 前端展示与导出实施计划

## 实施步骤

- [ ] 确认阶段 1/2 的 OpenAPI 客户端和权限键已生成，不手工修改生成物。
- [ ] 在提成页面增加月份范围类型、转换纯函数和单元测试。
- [ ] 列表请求增加归属日期参数，新增归属日期和 CNY 提成列。
- [ ] 增加 `canExportFinanceCommissions`，按权限显示导出按钮。
- [ ] 增加页面就近 CSV 序列化/文件名函数和安全测试。
- [ ] 调用导出接口，处理 loading、空结果、成功和错误提示。
- [ ] 生成弹窗显示 CNY 折算依据，调整创建成功文案和刷新行为。
- [ ] 详情抽屉按本位币/CNY 分组展示三组金额及取消状态。
- [ ] 保持现有 ProTable 和公共 UI 模板风格，不做无关页面重构。

## 针对性验证

- [ ] 月份转换覆盖单月、跨月、跨年、闰年二月和单边值。
- [ ] 列表与导出请求复用同一规范化筛选。
- [ ] CSV 覆盖 BOM、逗号、双引号、CR/LF、中文及 `= + - @` 文本。
- [ ] 权限允许/拒绝两种按钮状态。
- [ ] 预览、详情、取消状态和空导出交互。

## 验证命令

```bash
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web lint
pnpm --dir web biome:lint
pnpm run check:web
pnpm run build
```

## 风险与回滚

- 使用 Ant Design 组件前查询本地组件 API，不猜测属性。
- 不修改 `web/src/services/roncin/`、`web/types/` 或权限生成物。
- 页面代码和测试保持一个提交，可独立回滚，不影响服务端快照数据。

