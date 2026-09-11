# 重构页头为标准双层结构

## Goal

将 `PageHeaderShell` 重构为企业级标准“上下双层”页头结构（对齐 Ant Design Pro / 飞书标准）：
- 上层：纯净的 12px 标准面包屑路径（含层级回溯与当前页节点，Hover 变蓝微交互）。
- 下层：醒目独立的实体大标题、状态 Tag、副标题与右侧操作按钮栏。

## Requirements

1. **结构分层**：
   - 垂直拆分为两行：顶部面包屑行（`roncin-page-header-breadcrumbs`）与下方主操作行（`roncin-page-header-main-row`）。
   - 面包屑顶部完整呈现路径：前面的父级节点 + 柔和斜杠 `/` + 末尾当前页文本节点（不可点击）。
2. **主操作行美化**：
   - 左侧：轻量返回按钮（带 Tooltip 提示 `backText` 与 `aria-label`）+ 醒目加粗大标题（16~18px）+ 业务 Tag + 副标题。
   - 右侧：主操作按钮组（`extra`）。
3. **高密度视觉与样式收拢**：
   - 有面包屑时内边距精调为 `8px 16px 10px 16px`，整体高度约 64~66px，纯白底细边框微阴影。
   - 无面包屑时高度自适应在 48px 紧凑高度。
4. **向后兼容性**：
   - 保持全部 props 兼容（`title`, `subTitle`, `breadcrumbs`, `onBack`, `backText`, `tags`, `extra`, `sticky` 等）。
   - 现有全部单元测试通过。

## Acceptance Criteria

- [x] 面包屑位于页头上方独立成行，末尾包含当前页文本，前面节点支持 hover 变蓝高亮。
- [x] 实体大标题与返回按钮、副标题在下方独立成行，视觉层次分明，不再生硬拼接在同一行。
- [x] 单测 `PageHeaderShell.test.tsx` 和 `OrderPageHeader.test.tsx` 100% 通过。

