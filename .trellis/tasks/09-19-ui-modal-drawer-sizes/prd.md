# 标准化 Modal 与 Drawer 弹窗尺寸

## Goal

消除全站 18 种 Modal 宽度与 10 种 Drawer 宽度随意硬编码的现状，建立标准的 T-shirt 尺寸常量并在全站推行。

## Requirements

1. **尺寸常量体系规范**：
   在 `web/src/components/ui`（例如 `web/src/components/ui/constants/dialog-sizes.ts`）导出标准常量：
   - **Modal Sizes**：
     - `MODAL_SIZE.SM`: `520`（单列简易输入、扫码、审批确认、密码重置）
     - `MODAL_SIZE.MD`: `680`（标准业务表单、双列表单、规则编辑）
     - `MODAL_SIZE.LG`: `960`（多列表单、大网格、跨实体配置）
   - **Drawer Sizes**：
     - `DRAWER_SIZE.MD`: `860`（标准下钻详情、历史记录抽屉）
     - `DRAWER_SIZE.LG`: `1080`（复杂表单、规则抽屉、多页签抽屉）
     - `DRAWER_SIZE.XL`: `1200`（双表复杂工作台，如分单建账、共享箱货物分配）
2. **替换范围**：
   - 替换 `pages/finance/`、`pages/orders/`、`pages/admin/`、`pages/partners/`、`pages/settings/` 中的随意数值。
   - 消除类似 `440, 540, 560, 580, 620, 750, 760, 780, 820, 850, 920, 980, 1040, 1050, 1060, 1120, 1240` 等杂散尺寸。

## Acceptance Criteria

- [ ] `@/components/ui` 提供标准 `MODAL_SIZE` 与 `DRAWER_SIZE` 常量导出及类型支持。
- [ ] 核心业务模块（orders, finance, admin, partners）的 Modal 与 Drawer 宽度全部收敛到上述标准常量档位。
- [ ] 弹窗内部表单与布局无溢出或截断，响应式体验良好。
- [ ] 单测与类型检查全部通过。
