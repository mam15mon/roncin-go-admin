# 实施计划

## 启动条件

用户审阅本次规划并明确同意实施后，再 task.py start。开始前检查 git status、当前任务和分支，保留其他任务改动，不沿用上一个已归档任务的推送授权。

## 顺序

1. 复现共享箱双查询失败并记录 HTTP 请求/鉴权根因；补最小必要修复。
2. 实现共享箱/候选独立请求状态、失败提示与重试，保持手动分配草稿。
3. 在详情箱量分节接入普通实际箱维护及成功摘要；新建页提供保存后维护提示，区分共享箱。
4. 实现后补针对性回归，覆盖拒绝、undefined、空数组、重试、版本、未保存订单字段和相邻列表入口。
5. 独立 review 与浏览器验收后执行风险匹配门禁，更新规范与验收记录，再提交；本次任务创建不授权直接合并推送。

## 验证

- 前端定向：SeaSharedContainerDrawer、ContainerCargoDelete、detail-shared-container-workbench、SeaExportDetailFeatures 及新增详情实际箱测试；使用 pnpm --dir web exec vitest run，路径相对 web。
- 后端若有改动：go -C server test -p 32 ./internal/server ./internal/service ./internal/biz（按实际影响缩小），真实鉴权场景需经过 HTTP 路由绑定。
- 修改文件 Biome、pnpm --dir web tsc、git diff --check；最终前端 pnpm run check:web，后端改动按风险追加相应门禁。
- 浏览器控制403/网络/5xx/成功空数据，验证双分辨率与详情不丢输入；权限行为必须有真实后端测试证据，不仅依赖 page.route。
- 不为纯入口/UI 改动固定运行生产构建；不提交截图、Cookie或敏感请求。
