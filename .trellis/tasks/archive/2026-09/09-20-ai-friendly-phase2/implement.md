# 执行计划

## 前置
- [x] 用户选定全部方向（门禁/去重/导航/地图/文档纠错）并批准直接实施。
- [x] 分层现状勘察完成：全绿，biz 的 ErrorReason 例外已定位 5 文件。
- [x] prd.md / design.md 落盘；主会话直接实现，不走子代理。

## 实施分组（每组一次提交）
1. [ ] 分组A 门禁：scripts/layer-check-go（规则表+ErrorReason 标识符例外+测试）；
   接线 check:layers:go、check:server、CI；验证 `go run .` 零违规、`go test` 全绿。
   提交 feat。
2. [ ] 分组B 去重：5 组逐一提取（先查 business-tag 既有组件）；每组改后跑受影响定向
   测试；全部完成后 `pnpm --dir web tsc`、biome（改动文件）、check:architecture、
   重跑 report:duplicates 确认 5 组消失。提交 refactor。
3. [ ] 分组C 导航与文档：后端 capability-navigation、architecture-map 补账、spec 链接/
   路径校验与符号残留扫描、重复报告文档更新。提交 docs。

## 质量门禁
- 最终 `pnpm run check:fast`（含 check:web + check:server，自动带上新门禁）。
- Go 侧仅新增 scripts 工具与 biz 包内提取，无业务行为变更：`go -C server test ./...`
  财务定向用例必跑，其余按风险。
- 每组 `git diff --check`；完成按 finish-work 归档，不自行推送。
